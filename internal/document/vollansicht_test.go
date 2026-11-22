// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"
)

// ADR-0023 Nachtrag 5: Die Ansicht ist ein verschobener Ausschnitt der Vollansicht. Für zufällige Zuschnitte und
// Streifen (auch Zuschnittskanten mitten in einem Streifen) gilt: Ansichtspunkt v und Vollansichtspunkt v + VollVersatz
// liegen im selben Originalpixel, und die Pixel der Ansicht stehen in der Vollansicht bei VollVersatz.
func TestVollansichtVerschobenerAusschnitt(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 60, 40))
	for y := range 40 {
		for x := range 60 {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x * y), 255})
		}
	}
	zf := rand.New(rand.NewPCG(386, 5))
	for fall := range 400 {
		d, err := Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		var st []Streifen
		for range zf.IntN(4) {
			zeilen := zf.IntN(2) == 0
			n := 60
			if zeilen {
				n = 40
			}
			von := zf.IntN(n - 1)
			st = append(st, Streifen{Zeilen: zeilen, Von: von, Bis: von + 1 + zf.IntN(min(6, n-von-1)+1)})
		}
		x0, y0 := zf.IntN(30), zf.IntN(20)
		z := image.Rect(x0, y0, x0+1+zf.IntN(60-x0), y0+1+zf.IntN(40-y0))
		var ok []Streifen
		for _, s := range st { // nur gültige Kombinationen (keine Überlappung, Ansicht nicht leer)
			if d.SetzeAnsicht(z, append(append([]Streifen(nil), ok...), s)) == nil {
				ok = append(ok, s)
			}
		}
		if d.SetzeAnsicht(z, ok) != nil {
			continue
		}
		v, g, vg := d.VollVersatz(), d.Groesse(), d.VollGroesse()
		if !(image.Rectangle{Max: g}).Add(v).In(image.Rectangle{Max: vg}) {
			t.Fatalf("Fall %d: Ansicht %v bei %v liegt nicht in der Vollansicht %v", fall, g, v, vg)
		}
		for x := 0; x < g.X; x++ {
			a := d.ZuOriginal(Pt(float64(x), 0)).X
			w := d.VollZuOriginal(Pt(float64(x+v.X), 0)).X
			if a != w {
				t.Fatalf("Fall %d (z %v, st %v): Spalte %d → %v, Vollansicht → %v", fall, z, ok, x, a, w)
			}
		}
		for y := 0; y < g.Y; y++ {
			a := d.ZuOriginal(Pt(0, float64(y))).Y
			w := d.VollZuOriginal(Pt(0, float64(y+v.Y))).Y
			if a != w {
				t.Fatalf("Fall %d (z %v, st %v): Zeile %d → %v, Vollansicht → %v", fall, z, ok, y, a, w)
			}
		}
		an, voll := d.Ansichtsbild(), d.VollAnsichtsbild()
		for y := 0; y < g.Y; y++ {
			for x := 0; x < g.X; x++ {
				if an.NRGBAAt(x, y) != voll.NRGBAAt(x+v.X, y+v.Y) {
					t.Fatalf("Fall %d: Pixel (%d,%d) weicht ab", fall, x, y)
				}
			}
		}
	}
}

// Ohne Zuschnitt ist die Vollansicht die Ansicht.
func TestVollansichtOhneZuschnitt(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 30, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Von: 5, Bis: 10}}); err != nil {
		t.Fatal(err)
	}
	if d.VollVersatz() != (image.Point{}) || d.VollGroesse() != d.Groesse() || d.VollGroesse() != image.Pt(25, 20) {
		t.Errorf("Versatz %v, Vollansicht %v, Ansicht %v", d.VollVersatz(), d.VollGroesse(), d.Groesse())
	}
}
