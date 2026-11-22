// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// F-412: VerlaufPlatz sagt genau, wie viele Bytes Pixelkopien der Verlauf noch ohne Verwerfen aufnimmt — wer bis
// dahin schwärzt, verliert keine Schwärzung (SchwaerzungenImVerlauf bleibt vollständig), ein Pixel mehr verwirft eine.
func TestVerlaufPlatz(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 64 // 16 Pixel
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 16, 4)))
	if err != nil {
		t.Fatal(err)
	}
	schwarz := Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}
	if p := d.VerlaufPlatz(); p != 64 {
		t.Fatalf("leerer Verlauf: Platz %d, erwartet 64", p)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 2, 2), schwarz); err != nil { // 16 Bytes
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(3, 3)), Strich: Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	if p := d.VerlaufPlatz(); p != 48 {
		t.Fatalf("nach 2x2: Platz %d, erwartet 48 (Objekte halten keine Pixel)", p)
	}
	if err := d.Schwaerzen(image.Rect(4, 0, 10, 2), schwarz); err != nil { // 48 Bytes: genau der Platz
		t.Fatal(err)
	}
	if _, voll := d.SchwaerzungenImVerlauf(); !voll || d.VerlaufPlatz() != 0 {
		t.Fatalf("genau den Platz belegt: vollständig %v, Platz %d", voll, d.VerlaufPlatz())
	}
	if err := d.Schwaerzen(image.Rect(12, 0, 13, 1), schwarz); err != nil { // 4 Bytes zu viel
		t.Fatal(err)
	}
	if _, voll := d.SchwaerzungenImVerlauf(); voll {
		t.Error("über den Platz hinaus geschwärzt, Liste trotzdem vollständig")
	}
	if p := d.VerlaufPlatz(); p < 0 {
		t.Errorf("Platz negativ: %d", p)
	}
}

// F-412: PruefeSchwaerzung urteilt wie Schwaerzen, ohne Pixel oder Verlauf anzufassen, und liefert den Bereich im Bild.
func TestPruefeSchwaerzungWieSchwaerzen(t *testing.T) {
	schwarz := color.NRGBA{0, 0, 0, 255}
	faelle := []struct {
		r image.Rectangle
		s Schwaerzung
	}{
		{image.Rect(-5, -5, 10, 10), Schwaerzung{Farbe: schwarz}},
		{image.Rect(40, 40, 50, 50), Schwaerzung{Farbe: schwarz}},
		{image.Rect(0, 0, 10, 10), Schwaerzung{Farbe: color.NRGBA{}}},
		{image.Rect(0, 0, 20, 20), Schwaerzung{Block: 4, Farbe: schwarz}},
		{image.Rect(0, 0, 4, 20), Schwaerzung{Block: 8, Farbe: schwarz}},
		{image.Rect(0, 0, 20, 20), Schwaerzung{Block: 8, Mosaik: MaxMosaik + 1, Farbe: schwarz}},
		{image.Rect(0, 0, 20, 20), Schwaerzung{Block: 1 << 30, Farbe: schwarz}},
	}
	for i, f := range faelle {
		d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 32, 32)))
		if err != nil {
			t.Fatal(err)
		}
		r, perr := d.PruefeSchwaerzung(f.r, f.s)
		if d.Geschwaerzt() || d.Stand() != 0 {
			t.Fatalf("Fall %d: PruefeSchwaerzung hat das Dokument verändert", i)
		}
		serr := d.Schwaerzen(f.r, f.s)
		if (perr == nil) != (serr == nil) {
			t.Errorf("Fall %d: PruefeSchwaerzung %v, Schwaerzen %v", i, perr, serr)
		}
		if perr == nil {
			if liste, _ := d.SchwaerzungenImVerlauf(); len(liste) != 1 || liste[0].Bereich != r {
				t.Errorf("Fall %d: Bereich %v, Schwaerzen nahm %v", i, r, liste)
			}
		}
	}
}
