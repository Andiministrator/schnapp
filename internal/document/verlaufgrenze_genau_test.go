// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// F-336: Genau an der Grenze wird nichts verworfen (`>`, nicht `>=`); knapp darüber fällt der älteste Schritt weg.
func TestVerlaufGrenzeGenauErreicht(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 32 // zwei Schwärzungen 2 x 2 halten genau 32 Bytes
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 12, 4)))
	if err != nil {
		t.Fatal(err)
	}
	schwarz := Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}
	for i := 0; i < 2; i++ {
		if err := d.Schwaerzen(image.Rect(2*i, 0, 2*i+2, 2), schwarz); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(d.verlauf.erledigt); n != 2 || d.verlauf.basis != 0 {
		t.Fatalf("genau an der Grenze: %d Schritte, basis %d; erwartet 2, 0", n, d.verlauf.basis)
	}
	erster := d.verlauf.erledigt[0].stand
	if err := d.Schwaerzen(image.Rect(4, 0, 6, 2), schwarz); err != nil {
		t.Fatal(err)
	}
	if n := len(d.verlauf.erledigt); n != 2 || d.verlauf.basis != erster {
		t.Errorf("über der Grenze: %d Schritte, basis %d; erwartet 2, %d", n, d.verlauf.basis, erster)
	}
}
