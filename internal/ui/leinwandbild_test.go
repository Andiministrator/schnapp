// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"image"
	"slices"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// grauerEditorMitNaht ist editorMitNaht über einem deckend grauen Bild: Dann ist der durchsichtige Spalt (Alpha 0) und
// jede Mischung der Kanten im Bild abzulesen.
func grauerEditorMitNaht(t *testing.T) *editor {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 60, 60))
	for i := 0; i < len(b.Pix); i += 4 {
		copy(b.Pix[i:i+4], []byte{0x80, 0x80, 0x80, 0xFF})
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 26}}); err != nil {
		t.Fatal(err)
	}
	e := testEditor(t)
	e.s.D = d
	return e
}

// F-407 (Kritik A-1, B-2, Mutanten Q2/E14): Das Leinwandbild trägt Fasersaum und Abdunklung der Nähte und einen
// durchsichtigen Spalt, solange das Werkzeug nicht Zuschneiden ist; beim Zuschneiden ist es die unveränderte
// Vollansicht.
func TestLeinwandBildMischtKanten(t *testing.T) {
	e := grauerEditorMitNaht(t)
	img, stand := leinwandBild(e.s)
	roh := e.s.D.Ansichtsbild()
	if !stand.im {
		t.Fatalf("Stand %+v: Saum nicht im Bild", stand)
	}
	n := e.s.D.Naehte()[0]
	if c := img.NRGBAAt(30, n.Lage); c.A != 0 {
		t.Errorf("Spaltpixel (30,%d) = %v, Soll durchsichtig (Alpha 0)", n.Lage, c)
	}
	if c := roh.NRGBAAt(30, n.Lage); c.A != 0xFF {
		t.Fatalf("Vorbedingung: das Ansichtsbild ist deckend, Pixel %v", c)
	}
	unterschiedlich, imSaum := 0, 0
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			if a, b := img.NRGBAAt(x, y), roh.NRGBAAt(x, y); a != b && a.A != 0 {
				unterschiedlich++
				if a.R > b.R { // der Fasersaum ist weiß
					imSaum++
				}
			}
		}
	}
	if unterschiedlich == 0 || imSaum == 0 {
		t.Errorf("außerhalb des Spalts weicht kein Pixel vom Ansichtsbild ab (%d, davon heller %d): Fasersaum fehlt", unterschiedlich, imSaum)
	}

	e.s.Werkzeug = canvas.WerkzeugZuschneiden
	voll, standZ := leinwandBild(e.s)
	if standZ.im {
		t.Errorf("Zuschneiden: Saum im Bild")
	}
	if !bytes.Equal(voll.Pix, e.s.D.VollAnsichtsbild().Pix) {
		t.Errorf("Zuschneiden: das Bild weicht von der Vollansicht ab")
	}
	if c := voll.NRGBAAt(30, n.Lage); c.A != 0xFF {
		t.Errorf("Zuschneiden: Pixel (30,%d) = %v, Soll unverändert deckend", n.Lage, c)
	}
}

// F-407 (Kritik A-1, B-2, Mutanten Q3/E15): bildPruefen baut das Bild neu, wenn sich nur der Stand der Kanten ändert
// (Kantenform, Lage der Nähte, Werkzeug Zuschneiden), ohne dass Pixel anders wurden.
func TestBildPruefenFolgtKantenstand(t *testing.T) {
	e := grauerEditorMitNaht(t)
	e.bildNeu()
	e.bildStand = e.s.BildAenderung()
	if !e.kantenStand.im || e.kantenStand.form != document.KanteAbriss {
		t.Fatalf("Ausgangsstand %+v", e.kantenStand)
	}
	vorbedingung := func(was string) {
		t.Helper()
		if e.s.BildAenderung() != e.bildStand {
			t.Fatalf("%s: Vorbedingung verletzt, die Pixel haben sich geändert (BildAenderung)", was)
		}
	}
	if err := e.s.D.SetzeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	vorbedingung("Kantenform")
	e.bildPruefen()
	if e.kantenStand.form != document.KanteZacken {
		t.Errorf("nach dem Wechsel der Kantenform hält das Bild die Form %v", e.kantenStand.form)
	}
	if err := e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 30, Bis: 36}}); err != nil {
		t.Fatal(err)
	}
	vorbedingung("Lage der Naht")
	e.bildPruefen()
	if !slices.Equal(e.kantenStand.naehte, e.s.D.Naehte()) {
		t.Errorf("nach dem Verschieben der Naht hält das Bild die Nähte %v, Soll %v", e.kantenStand.naehte, e.s.D.Naehte())
	}
	e.s.Werkzeug = canvas.WerkzeugZuschneiden
	vorbedingung("Werkzeug")
	e.bildPruefen()
	if e.kantenStand.im {
		t.Errorf("beim Zuschneiden liegt der Saum noch im Bild")
	}
	if e.bildOrt != e.s.Bildflaeche().Min {
		t.Errorf("Bildort %v, Soll %v", e.bildOrt, e.s.Bildflaeche().Min)
	}
}

// F-407 (Kritik B-2, Mutant E17): Zum Stand des Leinwandbilds gehört die Lage der Nähte, nicht nur ihre Zahl.
func TestKantenBildGleichBeachtetLageDerNaehte(t *testing.T) {
	e := grauerEditorMitNaht(t)
	k := kantenBildStand(e.s)
	if !k.gleich(kantenBildStand(e.s)) {
		t.Fatal("derselbe Stand ist ungleich")
	}
	if err := e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 30, Bis: 36}}); err != nil {
		t.Fatal(err)
	}
	k2 := kantenBildStand(e.s)
	if len(k2.naehte) != len(k.naehte) {
		t.Fatalf("Vorbedingung: gleiche Zahl Nähte (%d, %d)", len(k.naehte), len(k2.naehte))
	}
	if k.gleich(k2) {
		t.Errorf("Naht verschoben (%v, %v), Stand gleich geblieben", k.naehte, k2.naehte)
	}
}

// F-407, F-521 (Mutanten E16 und ein Mutant, der den Ablauf nicht aufruft): zeichneNaehte führt den Ablauf von Renderer
// und SVG-Export (fuehreNahtAblaufAus) mit den Nähten und dem Faktor des Dokuments aus und sagt ihm, ob der Saum schon im
// Bild liegt.
func TestZeichneNaehteRuftDenAblaufAuf(t *testing.T) {
	alt := nahtAblauf
	defer func() { nahtAblauf = alt }()
	type aufruf struct {
		naehte []document.Naht
		faktor int
		saum   bool
	}
	var rufe []aufruf
	nahtAblauf = func(naehte []document.Naht, faktor int, saumImBild bool, spalte func(), durchgang func(func(document.Schicht) bool)) {
		rufe = append(rufe, aufruf{slices.Clone(naehte), faktor, saumImBild})
		alt(naehte, faktor, saumImBild, spalte, durchgang)
	}
	e := grauerEditorMitNaht(t)
	e.bildNeu()
	gtx := layout.Context{Ops: new(op.Ops)}
	e.zeichneNaehte(gtx)
	if len(rufe) != 1 {
		t.Fatalf("%d Aufrufe des Ablaufs, Soll 1", len(rufe))
	}
	if r := rufe[0]; !slices.Equal(r.naehte, e.s.D.Naehte()) || r.faktor != e.s.D.KantenFaktor() || !r.saum {
		t.Errorf("Aufruf %+v, Soll Nähte %v, Faktor %d, Saum im Bild", r, e.s.D.Naehte(), e.s.D.KantenFaktor())
	}
	e.zeichneNaehte(gtx)
	if len(rufe) != 1 {
		t.Errorf("zweites Zeichnen baute die Aufzeichnung neu (%d Aufrufe)", len(rufe))
	}
	// Beim Zuschneiden liegt der Saum nicht im Bild: Der Ablauf bekommt false, die Aufzeichnung wird neu gebaut.
	e.s.Werkzeug = canvas.WerkzeugZuschneiden
	e.bildPruefen()
	e.zeichneNaehte(gtx)
	if len(rufe) != 2 || rufe[1].saum {
		t.Errorf("Zuschneiden: Aufrufe %+v, Soll ein zweiter ohne Saum im Bild", rufe)
	}
	// Faktor 2 (kurze Seite 2160): Der Ablauf bekommt den Faktor des Dokuments, nicht 1.
	grossD, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160)))
	if err != nil {
		t.Fatal(err)
	}
	if err := grossD.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 300, Bis: 340}}); err != nil {
		t.Fatal(err)
	}
	if grossD.KantenFaktor() != 2 {
		t.Fatalf("Faktor %d, Soll 2", grossD.KantenFaktor())
	}
	e2 := testEditor(t)
	e2.s.D = grossD
	rufe = nil
	e2.zeichneNaehte(layout.Context{Ops: new(op.Ops)})
	if len(rufe) != 1 || rufe[0].faktor != 2 {
		t.Errorf("Faktor 2: Aufrufe %+v, Soll einer mit Faktor 2", rufe)
	}
}
