// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// ADR-0036 Zusatz Karte #356: Speicherbedarf zählt die Pixel des Originals und die Pixelkopien im Verlauf, erledigt und
// zurückgenommen; Objekte zählen nicht.
func TestSpeicherbedarf(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 16, 4)))
	if err != nil {
		t.Fatal(err)
	}
	if b := d.Speicherbedarf(); b != 16*4*4 {
		t.Fatalf("leer: %d, Soll %d", b, 16*4*4)
	}
	if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(3, 3)), Strich: Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	if b := d.Speicherbedarf(); b != 16*4*4 {
		t.Errorf("mit Kasten: %d, Soll %d", b, 16*4*4)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 2, 2), Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if b := d.Speicherbedarf(); b != 16*4*4+16 {
		t.Errorf("mit Schwärzung 2×2: %d, Soll %d", b, 16*4*4+16)
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if b := d.Speicherbedarf(); b != 16*4*4+16 {
		t.Errorf("zurückgenommen: %d, Soll %d (die Kopie bleibt für Wiederholen)", b, 16*4*4+16)
	}
}

// F-1041: Die Punkte eines Freihand-Strichs zählen mit, 16 Byte je Punkt.
func TestSpeicherbedarfFreihand(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 16, 4)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(NeuerFreihand([]Punkt{Pt(1, 1), Pt(5, 2), Pt(9, 1)}, Strich{Farbe: color.NRGBA{A: 255}, Breite: 1})); err != nil {
		t.Fatal(err)
	}
	if b, soll := d.Speicherbedarf(), int64(16*4*4+3*16); b != soll {
		t.Errorf("mit Freihand: %d, Soll %d", b, soll)
	}
}
