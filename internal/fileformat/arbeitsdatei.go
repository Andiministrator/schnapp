// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// SchreibeArbeitsdatei schreibt die Arbeitsdatei: sichtbar ist das fertige Bild (Ansicht mit Beschriftungen, vom
// Aufrufer gerendert), dazu die Blöcke snAP (SVG) und snAO (Original). Schwärzungen stecken schon im Original des
// Dokuments (Kernprinzip 2). Die Datei trägt keine weiteren Metadaten (keine tEXt-Blöcke, keine Zeitstempel außer
// Metadaten.Erstellt).
func SchreibeArbeitsdatei(w io.Writer, d *document.Dokument, sichtbar image.Image, m Metadaten) error {
	var sicht, orig bytes.Buffer
	// Über saveprogress.Schreiber meldet das Kodieren Fortschritt je geschriebenem Stück (F-298).
	if err := png.Encode(saveprogress.Schreiber(&sicht), sichtbar); err != nil {
		return err
	}
	saveprogress.Marke("bild-kodiert")
	if err := png.Encode(saveprogress.Schreiber(&orig), d.OriginalBild()); err != nil {
		return err
	}
	saveprogress.Marke("original-kodiert")
	var svg bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&svg, zlib.BestCompression)
	if _, err := zw.Write(svgArbeitsdatei(d, m)); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	daten, err := fuegeEin(sicht.Bytes(), block{blockSVG, svg.Bytes()}, block{blockOriginal, orig.Bytes()})
	if err != nil {
		return err
	}
	// In Stücken schreiben, je Stück ein Puls: Auf einem langsamen Medium dauert eine große Datei sonst ohne Fortschritt
	// länger als die Frist des Editors (Verify F-298 B-6). fsync und Umbenennen danach melden nichts; ein dort
	// hängendes Medium gilt zu Recht als hängend.
	for len(daten) > 0 {
		n := min(len(daten), schreibStueck)
		if _, err := saveprogress.Schreiber(w).Write(daten[:n]); err != nil {
			return err
		}
		daten = daten[n:]
	}
	saveprogress.Marke("datei-geschrieben")
	return nil
}

// schreibStueck: Größe eines Schreibaufrufs der Arbeitsdatei (Fortschritt je Stück, F-298).
const schreibStueck = 1 << 20

// Gelesen ist das Ergebnis von LeseArbeitsdatei.
type Gelesen struct {
	// Dokument mit Original, Objekten, Zuschnitt und Streifen; der Verlauf ist leer. Bei einem fremden PNG ohne
	// Schnapp-Blöcke ist das sichtbare Bild das Original.
	Dokument *document.Dokument
	Meta     Metadaten
	// Fremd meldet ein PNG ohne Schnapp-Blöcke.
	Fremd bool
}

// LeseArbeitsdatei liest eine Arbeitsdatei oder ein gewöhnliches PNG. Fehlt einer der beiden Schnapp-Blöcke, ist
// die Datei beschädigt (ErrBeschaedigt) — der Aufrufer kann sie mit LeseBild als einfaches Bild öffnen.
func LeseArbeitsdatei(r io.Reader) (*Gelesen, error) {
	// Fortschritt je gelesenem Stück: Beim Speichern liest die Bibliothek die alte Datei, auf einem langsamen Medium
	// dauert das (F-298).
	bloecke, err := leseBloecke(saveprogress.Leser(r))
	if err != nil {
		return nil, err
	}
	var svgDaten, origDaten []byte
	var hatSVG, hatOrig, hatNachIEND bool
	for _, b := range bloecke {
		switch b.typ {
		case nachIEND:
			hatNachIEND = true
		case blockSVG:
			if hatSVG {
				return nil, fmt.Errorf("%w: Block %s doppelt", ErrBeschaedigt, blockSVG)
			}
			svgDaten, hatSVG = b.daten, true
		case blockOriginal:
			if hatOrig {
				return nil, fmt.Errorf("%w: Block %s doppelt", ErrBeschaedigt, blockOriginal)
			}
			origDaten, hatOrig = b.daten, true
		}
	}
	if !hatSVG && !hatOrig {
		fremd := ohneBloecke(bloecke, nachIEND)
		bild, err := decodiereStrom(func() io.Reader { return neuerBlockStrom(fremd) })
		if err != nil {
			return nil, err
		}
		d, err := document.Neu(bild)
		if err != nil {
			return nil, err
		}
		return &Gelesen{Dokument: d, Fremd: true}, nil
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
	original, err := decodiere(origDaten)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", blockOriginal, err)
	}
	if b := original.Bounds(); b.Dx() != in.breite || b.Dy() != in.hoehe {
		return nil, fmt.Errorf("%w: Original %v passt nicht zu %dx%d", ErrBeschaedigt, b, in.breite, in.hoehe)
	}
	d, err := document.Neu(original)
	if err != nil {
		return nil, err
	}
	for _, o := range in.objekte {
		if _, err := d.Hinzufuegen(o); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
		}
	}
	if in.zuschnitt != d.Zuschnitt() {
		if err := d.SetzeZuschnitt(in.zuschnitt); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
		}
	}
	if len(in.streifen) > 0 {
		if err := d.SetzeStreifen(in.streifen); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
		}
	}
	if err := d.VergissVerlauf(); err != nil {
		return nil, err
	}
	return &Gelesen{Dokument: d, Meta: in.meta}, nil
}

// LeseBild liest nur das sichtbare Bild eines PNG (auch einer beschädigten Arbeitsdatei).
func LeseBild(r io.Reader) (image.Image, error) {
	bloecke, err := leseBloecke(r)
	if err != nil {
		return nil, err
	}
	sichtbar := ohneBloecke(bloecke, blockSVG, blockOriginal, nachIEND)
	bloecke = nil // die Blöcke gehören ab hier nur noch dem Strom (F-64)
	return decodiereStrom(func() io.Reader { return neuerBlockStrom(sichtbar) })
}

// decodiere liest ein PNG und prüft vorher die Größe, damit eine fremde Datei keinen riesigen Speicher anfordert.
func decodiere(daten []byte) (image.Image, error) {
	return decodiereStrom(func() io.Reader { return bytes.NewReader(daten) })
}

// decodiereStrom wie decodiere, mit zwei Lesedurchgängen (Größe, dann Pixel) über oeffne, ohne die Daten vorher zu
// einem Puffer zusammenzusetzen (F-64).
func decodiereStrom(oeffne func() io.Reader) (image.Image, error) {
	cfg, err := png.DecodeConfig(oeffne())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > document.MaxSeite || cfg.Height > document.MaxSeite ||
		cfg.Width*cfg.Height > document.MaxPixel {
		return nil, fmt.Errorf("%w: Bild %dx%d", ErrZuGross, cfg.Width, cfg.Height)
	}
	bild, err := png.Decode(saveprogress.Leser(oeffne())) // Fortschritt beim Dekodieren (F-298)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
	}
	return bild, nil
}

// IstBeschaedigt meldet, ob err auf eine beschädigte oder zu große Datei zurückgeht. Eine Arbeitsdatei aus einer
// neueren Fassung (ErrFormatNeuer) ist nicht beschädigt; beide Fälle lassen sich mit LeseBild als Bild öffnen.
func IstBeschaedigt(err error) bool {
	return errors.Is(err, ErrBeschaedigt) || errors.Is(err, ErrZuGross) || errors.Is(err, ErrKeinPNG)
}
