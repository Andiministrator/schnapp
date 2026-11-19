// SPDX-License-Identifier: GPL-3.0-or-later

// Package fileformat liest und schreibt Schnapps Arbeitsdatei — ein PNG, das überall als fertiges Bild erscheint und in
// zwei privaten Blöcken alle Ebenen mitführt (ADR-0008 mit Nachtrag 1) — und die Exporte PNG, JPG und SVG.
//
// Blöcke: "snAP" trägt ein zlib-komprimiertes SVG mit Objekten, Zuschnitt, Stauch-Streifen und Metadaten; "snAO"
// trägt das unbeschnittene Original als PNG-Datenstrom. Beide sind nach PNG-Spezifikation zusätzlich (Betrachter
// ignorieren sie), privat und nicht kopiersicher (wer die Pixel ändert, muss sie verwerfen).
//
// Das sichtbare Bild (Ansicht mit gerenderten Beschriftungen) liefert der Aufrufer; dieses Paket rendert nicht.
package fileformat

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

// Blocknamen (ADR-0008 Nachtrag 1).
const (
	blockSVG      = "snAP"
	blockOriginal = "snAO"
	nachIEND      = "" // Pseudo-Block: hinter IEND folgen noch Daten
)

var pngSignatur = []byte("\x89PNG\r\n\x1a\n")

// Grenzen beim Lesen fremder Dateien.
const (
	maxBlock    = 512 << 20 // größter einzelner Block, den wir lesen
	maxDatei    = 1 << 30   // größte Datei insgesamt
	maxSVG      = 64 << 20  // entpacktes SVG
	maxElemente = 200_000   // SVG-Elemente
)

// Fehler beim Lesen.
var (
	ErrKeinPNG     = errors.New("datei: kein PNG")
	ErrBeschaedigt = errors.New("datei: Datei beschädigt")
	ErrZuGross     = errors.New("datei: Datei oder Block zu groß")
	// ErrFormatNeuer: die Arbeitsdatei stammt aus einer neueren Fassung von Schnapp.
	ErrFormatNeuer = errors.New("datei: Formatversion neuer als diese Fassung")
)

type block struct {
	typ   string
	daten []byte
}

// leseBloecke zerlegt einen PNG-Datenstrom in Blöcke und prüft Längen und Prüfsummen.
func leseBloecke(r io.Reader) ([]block, error) {
	r = io.LimitReader(r, maxDatei+1)
	sig := make([]byte, len(pngSignatur))
	if _, err := io.ReadFull(r, sig); err != nil || !bytes.Equal(sig, pngSignatur) {
		return nil, ErrKeinPNG
	}
	var bloecke []block
	gesamt := len(sig)
	kopf := make([]byte, 8)
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
		daten := make([]byte, laenge)
		if _, err := io.ReadFull(r, daten); err != nil {
			return nil, fmt.Errorf("%w: Block %q: %v", ErrBeschaedigt, typ, err)
		}
		pruef := make([]byte, 4)
		if _, err := io.ReadFull(r, pruef); err != nil {
			return nil, fmt.Errorf("%w: Prüfsumme %q: %v", ErrBeschaedigt, typ, err)
		}
		crc := crc32.NewIEEE()
		crc.Write(kopf[4:8])
		crc.Write(daten)
		if crc.Sum32() != binary.BigEndian.Uint32(pruef) {
			return nil, fmt.Errorf("%w: Prüfsumme von %q falsch", ErrBeschaedigt, typ)
		}
		bloecke = append(bloecke, block{typ, daten})
		if typ == "IEND" {
			// Daten nach IEND als eigener Pseudo-Block: nur die Arbeitsdatei behandelt sie als Fehler; fremde PNG
			// und LeseBild bleiben lesbar wie in jedem Bildbetrachter (F-65).
			if n, _ := r.Read(make([]byte, 1)); n > 0 {
				bloecke = append(bloecke, block{typ: nachIEND})
			}
			return bloecke, nil
		}
	}
}

// schreibeBloecke setzt Blöcke zu einem PNG-Datenstrom zusammen.
func schreibeBloecke(w io.Writer, bloecke []block) error {
	if _, err := w.Write(pngSignatur); err != nil {
		return err
	}
	for _, b := range bloecke {
		kopf := make([]byte, 8)
		binary.BigEndian.PutUint32(kopf[:4], uint32(len(b.daten)))
		copy(kopf[4:], b.typ)
		crc := crc32.NewIEEE()
		crc.Write(kopf[4:8])
		crc.Write(b.daten)
		pruef := binary.BigEndian.AppendUint32(nil, crc.Sum32())
		for _, teil := range [][]byte{kopf, b.daten, pruef} {
			if _, err := w.Write(teil); err != nil {
				return err
			}
		}
	}
	return nil
}

// fuegeEin setzt zusätzliche Blöcke vor IEND in einen PNG-Datenstrom aus image/png.
func fuegeEin(pngDaten []byte, zusatz ...block) ([]byte, error) {
	bloecke, err := leseBloecke(bytes.NewReader(pngDaten))
	if err != nil {
		return nil, err
	}
	n := len(bloecke)
	if n == 0 || bloecke[n-1].typ != "IEND" {
		return nil, fmt.Errorf("%w: IEND fehlt", ErrBeschaedigt)
	}
	alle := append(append(append([]block{}, bloecke[:n-1]...), zusatz...), bloecke[n-1])
	var buf bytes.Buffer
	if err := schreibeBloecke(&buf, alle); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ohneBloecke liefert den Datenstrom ohne die genannten Blocktypen.
func ohneBloecke(bloecke []block, weg ...string) []block {
	var res []block
	for _, b := range bloecke {
		raus := false
		for _, t := range weg {
			raus = raus || b.typ == t
		}
		if !raus {
			res = append(res, b)
		}
	}
	return res
}
