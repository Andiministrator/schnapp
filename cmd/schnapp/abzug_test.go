// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"image"
	"image/color"
	"sync"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// Abzug des Dokuments für Speichern und Kopieren im Hintergrund (ADR-0020 Nachtrag 6, F-282).

func abzugDokument(t *testing.T) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	for i := range b.Pix {
		b.Pix[i] = uint8(i*31 + i/7)
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	rot := document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 3}
	objekte := []document.Objekt{
		document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(80, 60)), Strich: rot, Eckradius: 4},
		document.GeraderPfeil(document.Pt(20, 150), document.Pt(250, 40), rot).MitGriff(document.Pt(100, 20)),
		document.Text{Ort: document.Pt(30, 70), Inhalt: "Zeile eins\nZeile zwei äöü", Schriftgrad: 18, Farbe: color.NRGBA{B: 255, A: 255},
			Hintergrund: color.NRGBA{R: 255, G: 255, A: 255}},
		document.Nummer{Mitte: document.Pt(200, 150), Wert: 7, Radius: 14, Farbe: color.NRGBA{G: 128, A: 255}},
	}
	for _, o := range objekte {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatalf("%T: %v", o, err)
		}
	}
	if err := d.Schwaerzen(image.Rect(100, 100, 160, 140), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAnsicht(image.Rect(5, 5, 295, 195), []document.Streifen{{Zeilen: true, Von: 80, Bis: 90}, {Von: 260, Bis: 270}}); err != nil {
		t.Fatal(err)
	}
	return d
}

// Vollständigkeit: Die Arbeitsdatei aus dem Abzug ist bytegleich zur Arbeitsdatei aus dem Dokument, ebenso PNG und
// Prüfung des Budgets.
func TestAbzugSchreibtDieselbeDatei(t *testing.T) {
	d := abzugDokument(t)
	a, err := d.Abzug()
	if err != nil {
		t.Fatal(err)
	}
	schreibe := func(x *document.Dokument) []byte {
		sichtbar, err := render.Bild(x)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := fileformat.SchreibeArbeitsdatei(&buf, x, sichtbar, fileformat.Metadaten{}); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	if !bytes.Equal(schreibe(d), schreibe(a)) {
		t.Error("Arbeitsdatei aus dem Abzug weicht ab")
	}
	p1, _ := render.PNG(d)
	p2, _ := render.PNG(a)
	if !bytes.Equal(p1, p2) {
		t.Error("PNG aus dem Abzug weicht ab")
	}
	if a.NaechsteNummer() != d.NaechsteNummer() || a.Geschwaerzt() != d.Geschwaerzt() || a.SchwaerzStand() != d.SchwaerzStand() || a.Stand() != d.Stand() {
		t.Error("Zähler weichen ab")
	}
	// Eine neue Schwärzung am Abzug bekommt keine Kennung, die das Dokument schon vergeben hat.
	_ = d.Rueckgaengig() // Ansicht
	_ = d.Rueckgaengig() // Schwärzung zurück: Kennung 1 ist vergeben, SchwaerzStand 0
	a2, _ := d.Abzug()
	if err := a2.Schwaerzen(image.Rect(0, 0, 20, 20), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if a2.SchwaerzStand() == 1 {
		t.Error("Abzug vergibt eine schon benutzte Schwärzungskennung")
	}
}

// Nebenläufigkeit wie im Editor: Speichern (Abzug 1) und Kopieren (Abzug 2) in Goroutinen, die Ereignisschleife
// bearbeitet und rendert das Dokument weiter (Export, Zeichnen, Reichweite). Mit -race.
func TestAbzugNebenDemDokument(t *testing.T) {
	i18n.Setze(nil)
	d := abzugDokument(t)
	b := testBibliothek(t)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := b.Orte.Ablage + "/" + e.Pfad
	a0, g, stand, err := ladeFuerEditor(abs, false)
	if err != nil {
		t.Fatal(err)
	}
	dok := g.Dokument
	speichern := speichernIn(b, a0, g, stand)
	for runde := range 6 {
		if _, err := dok.Hinzufuegen(document.Text{Ort: document.Pt(float64(10+runde*5), 20), Inhalt: "Runde " + string(rune('a'+runde)),
			Schriftgrad: 14, Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
		if err := dok.Schwaerzen(image.Rect(runde*10, 0, runde*10+12, 12), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
		a1, _ := dok.Abzug()
		a2, _ := dok.Abzug()
		var wg sync.WaitGroup
		var erg ui.Gespeichert
		var serr error
		wg.Add(2)
		go func() { defer wg.Done(); erg, serr = speichern(a1) }()
		go func() { defer wg.Done(); _, _ = render.PNG(a2) }()
		// „Ereignisschleife“: bearbeiten, rendern, vermessen
		for i := range 5 {
			id, err := dok.Hinzufuegen(document.Text{Ort: document.Pt(50, float64(40+i*10)), Inhalt: "neben dem Speichern", Schriftgrad: 12,
				Farbe: color.NRGBA{A: 255}})
			if err != nil {
				t.Fatal(err)
			}
			_ = dok.Schwaerzen(image.Rect(200, 100+i*5, 230, 104+i*5), document.Schwaerzung{Farbe: color.NRGBA{A: 255}})
			_, _ = render.Bild(dok)
			_ = render.Pruefe(dok)
			_ = dok.Ansichtsbild()
			_ = dok.Rueckgaengig()
			_ = dok.Entfernen(id)
		}
		wg.Wait()
		if serr != nil {
			t.Fatalf("Runde %d: %+v %v", runde, erg, serr)
		}
	}
}
