// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// ADR-0022 Nachtrag 1 (F-163): Das Mosaik hängt nur vom Mittel des ganzen Bereichs ab, nicht vom Inhalt. Zwei Bilder
// mit gleichem Mittel und vertauschtem Muster ergeben bei gleichem Zufall dasselbe Ergebnis; jede Blockfarbe liegt
// höchstens 2·Mosaik vom Mittel entfernt. Werte außerhalb 0..MaxMosaik werden abgelehnt.
func TestMosaikOhneInhalt(t *testing.T) {
	alt := zufall
	defer func() { zufall = alt }()
	muster := func(tausch bool) *Dokument {
		b := image.NewNRGBA(image.Rect(0, 0, 48, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 48; x++ {
				hell := (x/3+y/5)%2 == 0
				if tausch {
					hell = !hell
				}
				c := color.NRGBA{20, 20, 20, 255}
				if hell {
					c = color.NRGBA{220, 220, 220, 255}
				}
				b.SetNRGBA(x, y, c)
			}
		}
		d, err := Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	ergebnis := func(d *Dokument) *image.NRGBA {
		n := 0
		zufall = func(m int) int { n++; return (n * 7) % m } // fest, aber je Block verschieden
		if err := d.Schwaerzen(image.Rect(0, 0, 48, 24), Schwaerzung{Block: 8, Mosaik: 24}); err != nil {
			t.Fatal(err)
		}
		return d.Original()
	}
	a, b := ergebnis(muster(false)), ergebnis(muster(true))
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Error("Mosaik hängt vom Inhalt ab")
	}
	verschieden := map[color.NRGBA]bool{}
	for y := 0; y < 24; y++ {
		for x := 0; x < 48; x++ {
			c := a.NRGBAAt(x, y)
			verschieden[c] = true
			if d := int(c.R) - 120; d < -48 || d > 48 || c.R != c.G || c.G != c.B {
				t.Fatalf("Pixel (%d,%d) = %v, Soll Grau 120±48", x, y, c)
			}
		}
	}
	if len(verschieden) < 2 {
		t.Error("Mosaik ohne Blockstruktur")
	}
	d := muster(false)
	for _, m := range []int{-1, MaxMosaik + 1} {
		if d.Schwaerzen(image.Rect(0, 0, 16, 8), Schwaerzung{Block: 8, Mosaik: m}) == nil {
			t.Errorf("Mosaik %d ohne Fehler", m)
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
