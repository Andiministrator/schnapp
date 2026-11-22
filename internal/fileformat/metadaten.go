// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"io"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Kopfdaten sind die Angaben einer Arbeitsdatei, die ohne Pixel auskommen (LeseMetadaten).
type Kopfdaten struct {
	// Meta aus dem snAP-Block; bei einem fremden PNG leer.
	Meta Metadaten
	// Breite und Hoehe der Ansicht, so wie das volle Lesen sie meldet (Dokument.Groesse: Originalgröße aus dem
	// snAO-Kopf, Zuschnitt und Streifen aus dem snAP). Bei einem fremden PNG ist es die Größe aus dem IHDR.
	Breite, Hoehe int
	// Fremd meldet ein PNG ohne Schnapp-Blöcke (wie Gelesen.Fremd).
	Fremd bool
}

// LeseMetadaten liest nur die Angaben einer Arbeitsdatei: Metadaten aus dem snAP-Block und die Bildgröße aus IHDR.
// Weder die Pixel des sichtbaren Bilds noch der snAO-Block werden dekodiert oder in den Speicher geladen — der
// Datenstrom läuft einmal durch, übergroße Blöcke werden nur gezählt und auf ihre Prüfsumme geprüft. Das macht
// den Neuaufbau eines Index schnell (Bibliothek).
//
// Die Grenzen gelten wie bei LeseArbeitsdatei (maxDatei, maxBlock, maxSVG, maxElemente); dieselben Fehler
// kommen zurück: ErrKeinPNG, ErrBeschaedigt, ErrZuGross, ErrFormatNeuer. Objekte, Zuschnitt und Streifen des snAP
// prüft document.PruefeInhalt auf demselben Weg wie das Laden (gegen die Originalgröße aus dem snAO-Kopf), und die
// gemeldete Größe ist die der Ansicht wie beim vollen Lesen (F-272). Restabweichung: Was sich nur durch Dekodieren der
// Pixel feststellen ließe (kaputte Bilddaten des sichtbaren Bilds oder des Originals, ungültige Farbtiefe im IHDR),
// bleibt unentdeckt; der Aufrufer sieht das erst beim Öffnen.
//
// Angekündigte Blocklängen reservieren nichts im Voraus (F-273, F-59, ADR-0008): Der IHDR hat genau 13 Bytes, der
// snAP wächst mit den tatsächlich gelesenen Bytes.
func LeseMetadaten(r io.Reader) (_ *Kopfdaten, fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	r = io.LimitReader(r, maxDatei+1)
	sig := make([]byte, len(pngSignatur))
	if _, err := io.ReadFull(r, sig); err != nil || !bytes.Equal(sig, pngSignatur) {
		return nil, ErrKeinPNG
	}
	var (
		kopf                         = make([]byte, 8)
		gesamt                       = len(sig)
		ihdr                         []byte
		svgDaten                     []byte
		origKopf                     []byte // die ersten Bytes von snAO: PNG-Signatur und IHDR
		hatSVG, hatOrig, hatNachIEND bool
		ersterBlock                  = true
		puffer                       = make([]byte, 32<<10)
	)
	for {
		if _, err := io.ReadFull(r, kopf); err != nil {
			return nil, fmt.Errorf("%w: Blockkopf: %v", ErrBeschaedigt, err)
		}
		laenge := binary.BigEndian.Uint32(kopf[:4])
		typ := string(kopf[4:8])
		if laenge > maxBlock {
			return nil, fmt.Errorf("%w: Block %q mit %d Bytes", ErrZuGross, typ, laenge)
		}
		if gesamt += 12 + int(laenge); gesamt > maxDatei {
			return nil, ErrZuGross
		}
		if ersterBlock && typ != "IHDR" {
			return nil, fmt.Errorf("%w: IHDR fehlt", ErrBeschaedigt)
		}
		ersterBlock = false

		// Nur diese Blöcke werden gehalten; alles andere läuft durch den Puffer, ohne zu wachsen.
		var behalte []byte
		var halte bool
		switch typ {
		case "IHDR":
			if laenge != 13 {
				return nil, fmt.Errorf("%w: IHDR mit %d Bytes", ErrBeschaedigt, laenge)
			}
			halte = true
		case blockSVG:
			if hatSVG {
				return nil, fmt.Errorf("%w: Block %s doppelt", ErrBeschaedigt, blockSVG)
			}
			hatSVG, halte = true, true
		case blockOriginal:
			if hatOrig {
				return nil, fmt.Errorf("%w: Block %s doppelt", ErrBeschaedigt, blockOriginal)
			}
			hatOrig = true
		}
		crc := crc32.NewIEEE()
		crc.Write(kopf[4:8])
		rest := int64(laenge)
		if halte {
			var err error
			if behalte, err = leseDaten(r, laenge); err != nil {
				return nil, fmt.Errorf("%w: Block %q: %v", ErrBeschaedigt, typ, err)
			}
			crc.Write(behalte)
			rest = 0
		}
		for rest > 0 {
			n := int64(len(puffer))
			if n > rest {
				n = rest
			}
			if _, err := io.ReadFull(r, puffer[:n]); err != nil {
				return nil, fmt.Errorf("%w: Block %q: %v", ErrBeschaedigt, typ, err)
			}
			if typ == blockOriginal && len(origKopf) < 8+25 {
				origKopf = append(origKopf, puffer[:min(int(n), 8+25-len(origKopf))]...)
			}
			crc.Write(puffer[:n])
			rest -= n
		}
		pruef := make([]byte, 4)
		if _, err := io.ReadFull(r, pruef); err != nil {
			return nil, fmt.Errorf("%w: Prüfsumme %q: %v", ErrBeschaedigt, typ, err)
		}
		if crc.Sum32() != binary.BigEndian.Uint32(pruef) {
			return nil, fmt.Errorf("%w: Prüfsumme von %q falsch", ErrBeschaedigt, typ)
		}
		switch typ {
		case "IHDR":
			ihdr = behalte
		case blockSVG:
			svgDaten = behalte
		}
		if typ == "IEND" {
			if n, _ := r.Read(make([]byte, 1)); n > 0 {
				hatNachIEND = true
			}
			break
		}
	}

	if !hatSVG && !hatOrig {
		b, h, err := ihdrGroesse(ihdr)
		if err != nil {
			return nil, err
		}
		return &Kopfdaten{Breite: b, Hoehe: h, Fremd: true}, nil
	}
	if hatNachIEND {
		return nil, fmt.Errorf("%w: Daten nach IEND", ErrBeschaedigt)
	}
	if !hatSVG || !hatOrig {
		return nil, fmt.Errorf("%w: nur einer der Blöcke %s/%s vorhanden", ErrBeschaedigt, blockSVG, blockOriginal)
	}
	zr, err := zlib.NewReader(bytes.NewReader(svgDaten))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrBeschaedigt, blockSVG, err)
	}
	defer zr.Close()
	in, err := leseSVG(zr)
	if err != nil {
		return nil, err
	}
	// Das Original muss zur Größe im SVG passen: Der snAO-Block beginnt mit einem PNG (Signatur, IHDR).
	ob, oh, err := originalGroesse(origKopf)
	if err != nil {
		return nil, imBlock(blockOriginal, err)
	}
	if ob != in.breite || oh != in.hoehe {
		return nil, fmt.Errorf("%w: Original %dx%d passt nicht zu %dx%d", ErrBeschaedigt, ob, oh, in.breite, in.hoehe)
	}
	// Objekte, Zuschnitt und Streifen wie beim Laden prüfen; die Größe ist die der Ansicht (F-272).
	gr, err := document.PruefeInhalt(image.Pt(ob, oh), in.zuschnitt, in.streifen, in.objekte)
	if err != nil {
		if errors.Is(err, document.ErrBildGroesse) {
			return nil, fmt.Errorf("%w: %v", ErrZuGross, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
	}
	return &Kopfdaten{Meta: in.meta, Breite: gr.X, Hoehe: gr.Y}, nil
}

// originalGroesse liest Breite und Höhe aus den ersten 33 Bytes des snAO-Blocks: PNG-Signatur, dann ein IHDR-Block
// mit 13 Bytes Inhalt und richtiger Prüfsumme. Alles andere ist kein PNG-Original (F-272).
func originalGroesse(kopf []byte) (b, h int, err error) {
	const laenge = 8 + 4 + 4 + 13 + 4
	if len(kopf) < laenge || !bytes.Equal(kopf[:8], pngSignatur) ||
		binary.BigEndian.Uint32(kopf[8:12]) != 13 || string(kopf[12:16]) != "IHDR" {
		return 0, 0, fmt.Errorf("%w: kein PNG mit IHDR", ErrBeschaedigt)
	}
	if crc32.ChecksumIEEE(kopf[12:29]) != binary.BigEndian.Uint32(kopf[29:33]) {
		return 0, 0, fmt.Errorf("%w: IHDR-Prüfsumme falsch", ErrBeschaedigt)
	}
	return ihdrGroesse(kopf[16:29])
}

// leseDaten liest genau laenge Bytes, ohne die angekündigte Länge vorab zu reservieren: Eine kleine Länge
// bekommt ihren Puffer sofort, eine große wächst mit den tatsächlich gelieferten Bytes. Eine kurze Datei mit
// angekündigten 500 MB kostet so nur ihre eigene Größe (F-273, F-59, ADR-0008).
func leseDaten(r io.Reader, laenge uint32) ([]byte, error) {
	const sofort = 1 << 20
	const stueck = 1 << 20
	if laenge <= sofort {
		daten := make([]byte, laenge)
		_, err := io.ReadFull(r, daten)
		return daten, err
	}
	// Wachstum in festen Stücken mit genauem Endpuffer (Z-5): Ein bytes.Buffer verdoppelt und forderte bei großen echten
	// Blöcken das 2,6- bis 4-Fache an und behielt bis zur doppelten Kapazität. Hier wächst nur die Liste der Stücke mit
	// den gelieferten Bytes; am Ende steht ein Puffer genau der Länge, die Stücke werden frei. Spitzenbedarf: das
	// Doppelte der Länge, behalten wird genau sie.
	var stuecke [][]byte
	for rest := int64(laenge); rest > 0; {
		p := make([]byte, min(rest, int64(stueck)))
		if _, err := io.ReadFull(r, p); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		stuecke = append(stuecke, p)
		rest -= int64(len(p))
	}
	daten := make([]byte, 0, laenge)
	for i, p := range stuecke {
		daten = append(daten, p...)
		stuecke[i] = nil
	}
	return daten, nil
}

// ihdrGroesse liest Breite und Höhe aus einem Blockstück, das mit dem 8-Byte-Blockkopf des IHDR beginnt (Länge,
// Typ "IHDR") oder, bei dem rohen Inhalt des IHDR-Blocks, direkt mit den Daten. Beides unterscheidet die Länge:
// der Inhalt allein hat 13 Bytes.
func ihdrGroesse(d []byte) (b, h int, err error) {
	if len(d) >= 8+13 && string(d[4:8]) == "IHDR" {
		d = d[8:]
	}
	if len(d) < 13 {
		return 0, 0, fmt.Errorf("%w: IHDR zu kurz", ErrBeschaedigt)
	}
	b, h = int(binary.BigEndian.Uint32(d[0:4])), int(binary.BigEndian.Uint32(d[4:8]))
	if b < 1 || h < 1 || b > document.MaxSeite || h > document.MaxSeite || b*h > document.MaxPixel {
		return 0, 0, fmt.Errorf("%w: Bild %dx%d", ErrZuGross, b, h)
	}
	return b, h, nil
}
