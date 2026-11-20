// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// ADR-0022 Punkt 2: Rauschen verschiebt jeden Block um höchstens ±Rauschen, Alpha bleibt; Werte außerhalb werden
// abgelehnt.
func TestVerpixelnMitRauschen(t *testing.T) {
	alt := zufall
	defer func() { zufall = alt }()
	// Bytes 0 → -r, 2r → +r: abwechselnd größte Verschiebung nach unten und oben.
	n := 0
	zufall = func(b []byte) {
		for i := range b {
			if n%2 == 0 {
				b[i] = 0
			} else {
				b[i] = 2 * 24
			}
			n++
		}
	}
	d := neu(t, 16, 8)
	ohne := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	copy(ohne.Pix, d.Original().Pix)
	d2 := neu(t, 16, 8)
	if err := d2.Schwaerzen(image.Rect(0, 0, 16, 8), Schwaerzung{Block: 8}); err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 16, 8), Schwaerzung{Block: 8, Rauschen: 24}); err != nil {
		t.Fatal(err)
	}
	mit, mittel := d.Original(), d2.Original()
	for _, p := range []image.Point{{0, 0}, {8, 0}} {
		a, b := mit.NRGBAAt(p.X, p.Y), mittel.NRGBAAt(p.X, p.Y)
		if a == b {
			t.Errorf("Block bei %v ohne Rauschen: %v", p, a)
		}
		for _, k := range [][2]uint8{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
			if diff := int(k[0]) - int(k[1]); diff < -24 || diff > 24 {
				t.Errorf("Block bei %v: Abweichung %d über 24", p, diff)
			}
		}
		if a.A != b.A {
			t.Errorf("Alpha bei %v verändert: %d statt %d", p, a.A, b.A)
		}
	}
	for _, r := range []int{-1, MaxRauschen + 1} {
		if d.Schwaerzen(image.Rect(0, 0, 16, 8), Schwaerzung{Block: 8, Rauschen: r}) == nil {
			t.Errorf("Rauschen %d ohne Fehler", r)
		}
	}
}

// ADR-0022 Punkte 3 und 4: Geschwaerzt zählt wirksame Schwärzungen über Rückgängig und VergissVerlauf hinweg;
// VerlaufMitSchwaerzung sieht sie im Verlauf, auch zurückgenommen.
func TestGeschwaerztUndVerlauf(t *testing.T) {
	d := neu(t, 20, 20)
	schwarz := Schwaerzung{Farbe: color.NRGBA{A: 255}}
	if d.Geschwaerzt() || d.VerlaufMitSchwaerzung() {
		t.Fatal("frisches Dokument gilt als geschwärzt")
	}
	if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(9, 9)), Strich: Strich{Farbe: color.NRGBA{A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	if d.VerlaufMitSchwaerzung() {
		t.Error("Kasten zählt als Schwärzung")
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 10, 10), schwarz); err != nil {
		t.Fatal(err)
	}
	if !d.Geschwaerzt() || !d.VerlaufMitSchwaerzung() {
		t.Error("nach Schwärzen: Geschwaerzt/VerlaufMitSchwaerzung falsch")
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if d.Geschwaerzt() {
		t.Error("nach Rückgängig noch geschwärzt")
	}
	if !d.VerlaufMitSchwaerzung() {
		t.Error("zurückgenommene Schwärzung hält alte Pixel, wird aber nicht gemeldet")
	}
	if err := d.Wiederholen(); err != nil {
		t.Fatal(err)
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	if !d.Geschwaerzt() || d.VerlaufMitSchwaerzung() {
		t.Error("nach VergissVerlauf: Geschwaerzt muss bleiben, der Verlauf leer sein")
	}
}
