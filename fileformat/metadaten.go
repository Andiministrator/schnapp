// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/Andiministrator/schnapp/document"
)

// Kopfdaten sind die Angaben einer Arbeitsdatei, die ohne Pixel auskommen (LeseMetadaten).
type Kopfdaten struct {
	// Meta aus dem snAP-Block; bei einem fremden PNG leer.
	Meta Metadaten
	// Breite und Hoehe des sichtbaren Bilds aus dem IHDR-Block.
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
// kommen zurück: ErrKeinPNG, ErrBeschaedigt, ErrZuGross, ErrFormatNeuer. Was sich nur durch Dekodieren der Pixel
// feststellen ließe (kaputte Bilddaten), bleibt unentdeckt; der Aufrufer sieht das erst beim Öffnen.
func LeseMetadaten(r io.Reader) (*Kopfdaten, error) {
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
			behalte = make([]byte, laenge)
			if _, err := io.ReadFull(r, behalte); err != nil {
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

	b, h, err := ihdrGroesse(ihdr)
	if err != nil {
		return nil, err
	}
	if !hatSVG && !hatOrig {
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
	// Das Original muss zur Größe im SVG passen: IHDR steht gleich am Anfang des snAO-Blocks (Signatur + 8 Byte
	// Blockkopf + Breite und Höhe).
	ob, oh, err := ihdrGroesse(origKopf[min(len(pngSignatur), len(origKopf)):])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", blockOriginal, err)
	}
	if ob != in.breite || oh != in.hoehe {
		return nil, fmt.Errorf("%w: Original %dx%d passt nicht zu %dx%d", ErrBeschaedigt, ob, oh, in.breite, in.hoehe)
	}
	return &Kopfdaten{Meta: in.meta, Breite: b, Hoehe: h}, nil
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
