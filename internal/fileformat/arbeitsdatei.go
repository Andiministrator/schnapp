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
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// blockFehler nennt den Block, in dem ein Fehler auftrat, hinter dem Paketpräfix: „fileformat: snAO: …“ statt
// „snAO: fileformat: …“ (ADR-0019 Nachtrag 2, F-362). Unwrap lässt errors.Is und errors.As durchgreifen.
type blockFehler struct {
	block string
	err   error
}

func (e *blockFehler) Error() string {
	s := e.err.Error()
	if rest, ok := strings.CutPrefix(s, "fileformat: "); ok {
		return "fileformat: " + e.block + ": " + rest
	}
	return e.block + ": " + s
}

func (e *blockFehler) Unwrap() error { return e.err }

// imBlock hüllt err mit dem Blocknamen.
func imBlock(block string, err error) error { return &blockFehler{block: block, err: err} }

// SchreibeArbeitsdatei schreibt die Arbeitsdatei: sichtbar ist das fertige Bild (Ansicht mit Beschriftungen, vom
// Aufrufer gerendert), dazu die Blöcke snAP (SVG) und snAO (Original). Schwärzungen stecken schon im Original des
// Dokuments (Kernprinzip 2). Die Datei trägt keine weiteren Metadaten (keine tEXt-Blöcke, keine Zeitstempel außer
// Metadaten.Erstellt).
// schreibGrenzeSVG ist die Grenze des Schreibers, gleich maxSVG des Lesers; nur Tests setzen sie kleiner.
var schreibGrenzeSVG = maxSVG

func SchreibeArbeitsdatei(w io.Writer, d *document.Dokument, sichtbar image.Image, m Metadaten) (fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
	if n := document.FreihandPunkteGesamt(d.Objekte()); n > document.MaxFreihandGesamt {
		// Der Leser lehnte die Datei ab (F-1049); lieber jetzt scheitern, solange die alte Datei noch steht.
		return fmt.Errorf("die Freihand-Striche haben %d Punkte, mehr als die %d, die eine Arbeitsdatei tragen darf",
			n, document.MaxFreihandGesamt)
	}
	var sicht, orig bytes.Buffer
	// Beide PNG-Kodierungen laufen nebenläufig (F-310): Sie lesen nur ihr Bild und schreiben je in einen eigenen
	// Puffer; zwei gleichzeitige Kodierungen kosten kaum mehr als eine. Über saveprogress.Schreiber meldet jede
	// Fortschritt je geschriebenem Stück (F-298; Puls und Marke sind nebenläufig sicher). Zurück kommt erst, wenn
	// beide fertig sind, auch im Fehlerfall (kein auslaufender Schreiber); ein Fehler des Bilds geht vor.
	original := d.OriginalBild()
	var origFehler error
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		if origFehler = png.Encode(saveprogress.Schreiber(&orig), original); origFehler == nil {
			saveprogress.Marke("original-kodiert")
		}
	}()
	bildFehler := png.Encode(saveprogress.Schreiber(&sicht), sichtbar)
	if bildFehler == nil {
		saveprogress.Marke("bild-kodiert")
	}
	<-fertig
	if bildFehler != nil {
		return bildFehler
	}
	if origFehler != nil {
		return origFehler
	}
	angaben := svgArbeitsdatei(d, m)
	if len(angaben) > schreibGrenzeSVG {
		// Der Leser lehnte die Datei ab (maxSVG); lieber jetzt scheitern, solange die alte Datei noch steht (F-841).
		return fmt.Errorf("die Beschriftungen ergeben %d MiB SVG, mehr als die %d MiB, die eine Arbeitsdatei tragen darf",
			len(angaben)>>20, schreibGrenzeSVG>>20)
	}
	var svg bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&svg, zlib.BestCompression)
	if _, err := zw.Write(angaben); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	daten, err := fuegeEin(sicht.Bytes(), block{blockSVG, svg.Bytes()}, block{blockOriginal, orig.Bytes()})
	if err != nil {
		return err
	}
	// In Stücken schreiben. Ein Write auf eine Datei landet im Seitencache und kehrt sofort zurück; die Zeit eines
	// langsamen Mediums vergeht erst im Sync (F-313). Kann der Schreiber synchronisieren (*os.File), folgt deshalb auf
	// jedes Stück ein Sync und danach der Puls: Es pulst, was tatsächlich abgeflossen ist (dazu, unschädlich, der Puls
	// von saveprogress.Schreiber vor jedem Write). Ein Sync, der gar nicht
	// zurückkehrt (totes Netzlaufwerk), meldet nichts und gilt zu Recht als hängend; einen Puls aus einem Zeitgeber
	// gibt es nicht. Ohne Sync (Puffer, Attrappe) pulst jedes Stück schon beim Schreiben.
	sy, kannSync := w.(interface{ Sync() error })
	for len(daten) > 0 {
		n := min(len(daten), schreibStueck)
		if _, err := saveprogress.Schreiber(w).Write(daten[:n]); err != nil {
			return err
		}
		daten = daten[n:]
		if kannSync {
			if err := sy.Sync(); err != nil {
				return err
			}
			saveprogress.Puls()
		}
	}
	saveprogress.Marke("datei-geschrieben")
	return nil
}

// schreibStueck: Größe eines Schreibaufrufs der Arbeitsdatei (Fortschritt je Stück, F-298).
const schreibStueck = 1 << 20

// Gelesen ist das Ergebnis von LeseArbeitsdatei.
type Gelesen struct {
	// Dokument mit Original, Objekten, Zuschnitt, Streifen und Kantenform; der Verlauf ist leer. Bei einem fremden PNG ohne
	// Schnapp-Blöcke ist das sichtbare Bild das Original.
	Dokument *document.Dokument
	Meta     Metadaten
	// Fremd meldet ein PNG ohne Schnapp-Blöcke.
	Fremd bool
}

// LeseArbeitsdatei liest eine Arbeitsdatei oder ein gewöhnliches PNG. Fehlt einer der beiden Schnapp-Blöcke, ist
// die Datei beschädigt (ErrBeschaedigt) — der Aufrufer kann sie mit LeseBild als einfaches Bild öffnen.
func LeseArbeitsdatei(r io.Reader) (_ *Gelesen, fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
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
		d, err := neuesDokument(bild) // frisch dekodiert: ohne zweite Kopie (F-334)
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
		return nil, imBlock(blockOriginal, err)
	}
	if b := original.Bounds(); b.Dx() != in.breite || b.Dy() != in.hoehe {
		return nil, fmt.Errorf("%w: Original %v passt nicht zu %dx%d", ErrBeschaedigt, b, in.breite, in.hoehe)
	}
	d, err := neuesDokument(original) // frisch dekodiert: ohne zweite Kopie (F-334)
	if err != nil {
		return nil, err
	}
	if err := d.VorgabeKante(in.kante); err != nil { // vor jeder Änderung: kein Rückgängig-Schritt
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
func LeseBild(r io.Reader) (_ image.Image, fehler error) {
	defer errprefix.Defer("fileformat", &fehler)
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
