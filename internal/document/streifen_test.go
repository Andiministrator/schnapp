// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"testing"
)

// Referenz für die Abbildung: die lineare Rechnung vor F-136, unverändert nachgebaut.
func refZuAnsicht(zuschnitt image.Rectangle, st []Streifen, p Punkt) Punkt {
	um := func(v float64, zeilen bool) float64 {
		von, bis := zuschnitt.Min.X, zuschnitt.Max.X
		if zeilen {
			von, bis = zuschnitt.Min.Y, zuschnitt.Max.Y
		}
		r := v - float64(von)
		for _, s := range st {
			if s.Zeilen != zeilen {
				continue
			}
			a, b := float64(max(s.Von, von)), float64(min(s.Bis, bis))
			if a >= b {
				continue
			}
			switch {
			case v >= b:
				r -= b - a
			case v > a:
				r -= v - a
			}
		}
		return r
	}
	return Punkt{um(p.X, false), um(p.Y, true)}
}

func refZuOriginal(zuschnitt image.Rectangle, st []Streifen, p Punkt) Punkt {
	um := func(v float64, zeilen bool) float64 {
		von := zuschnitt.Min.X
		if zeilen {
			von = zuschnitt.Min.Y
		}
		bis := zuschnitt.Max.X
		if zeilen {
			bis = zuschnitt.Max.Y
		}
		r := v + float64(von)
		for _, s := range st {
			if s.Zeilen != zeilen {
				continue
			}
			a, b := max(s.Von, von), min(s.Bis, bis)
			if a < b && r >= float64(a) {
				r += float64(b - a)
			}
		}
		return r
	}
	return Punkt{um(p.X, false), um(p.Y, true)}
}

// F-136: Die zwischengespeicherte Abbildung liefert dasselbe wie die lineare Rechnung — für Punkte auf, neben und
// in den Streifen, auch halbe Pixel, mit Zuschnitt, und nach Setzen, Rückgängig und Wiederholen.
func TestAbbildungGleichReferenz(t *testing.T) {
	d := neu(t, 200, 120)
	pruefe := func(name string) {
		t.Helper()
		z, st := d.Zuschnitt(), d.Streifen()
		for x := -2.0; x <= 202; x += 0.5 {
			for _, y := range []float64{-1, 0, 3.5, 7, 29.5, 30, 60, 100, 119, 121} {
				p := Pt(x, y)
				if got, want := d.ZuAnsicht(p), refZuAnsicht(z, st, p); got != want {
					t.Fatalf("%s: ZuAnsicht(%v) = %v, want %v", name, p, got, want)
				}
				if got, want := d.ZuOriginal(p), refZuOriginal(z, st, p); got != want {
					t.Fatalf("%s: ZuOriginal(%v) = %v, want %v", name, p, got, want)
				}
			}
		}
	}
	pruefe("leer")
	if err := d.SetzeZuschnitt(image.Rect(10, 5, 190, 110)); err != nil {
		t.Fatal(err)
	}
	pruefe("Zuschnitt")
	a := []Streifen{{false, 0, 12}, {false, 30, 40}, {false, 41, 42}, {false, 60, 65}, {false, 150, 200},
		{true, 0, 6}, {true, 20, 30}, {true, 31, 32}}
	if err := d.SetzeStreifen(a); err != nil {
		t.Fatal(err)
	}
	pruefe("a")
	if err := d.SetzeStreifen([]Streifen{{false, 100, 101}}); err != nil {
		t.Fatal(err)
	}
	pruefe("b")
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	pruefe("zurück auf a")
	if got, want := d.ZuAnsicht(Pt(66, 40)), refZuAnsicht(d.Zuschnitt(), sortiert(a), Pt(66, 40)); got != want {
		t.Errorf("veralteter Stand nach Rückgängig: %v, want %v", got, want)
	}
	if err := d.Wiederholen(); err != nil {
		t.Fatal(err)
	}
	pruefe("wieder b")
	if err := d.Rueckgaengig(); err != nil { // b -> a
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // Streifen a, dann der Zuschnitt zurück
		if err := d.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
	}
	pruefe("Anfang")
	if g := d.Groesse(); g != image.Pt(200, 120) {
		t.Errorf("Groesse %v, want 200x120", g)
	}
}

// F-136: Genau MaxStreifen sind erlaubt, einer mehr wird abgelehnt (Zustand bleibt unverändert).
func TestGrenzeStreifen(t *testing.T) {
	d := neu(t, 4*MaxStreifen, 10)
	liste := func(n int) []Streifen {
		st := make([]Streifen, n)
		for i := range st {
			st[i] = Streifen{Zeilen: false, Von: 2 * i, Bis: 2*i + 1}
		}
		return st
	}
	if err := d.SetzeStreifen(liste(MaxStreifen)); err != nil {
		t.Fatalf("MaxStreifen abgelehnt: %v", err)
	}
	if err := d.SetzeStreifen(liste(MaxStreifen + 1)); err == nil {
		t.Error("MaxStreifen+1 ohne Fehler")
	}
	if len(d.Streifen()) != MaxStreifen {
		t.Errorf("%d Streifen, soll %d", len(d.Streifen()), MaxStreifen)
	}
}

// Zeilen- und Spaltenstreifen zusammen dürfen höchstens MaxKacheln Kacheln ergeben (F-141): 63+63 → 4096 erlaubt,
// 64+63 und 63+64 → 4160 abgelehnt (beide Richtungen, Verify F-141), und eine Achse allein darf bis MaxStreifen.
func TestGrenzeKacheln(t *testing.T) {
	d := neu(t, 400, 400)
	liste := func(spalten, zeilen int) []Streifen {
		var st []Streifen
		for i := 0; i < spalten; i++ {
			st = append(st, Streifen{Zeilen: false, Von: 3 * i, Bis: 3*i + 1})
		}
		for i := 0; i < zeilen; i++ {
			st = append(st, Streifen{Zeilen: true, Von: 3 * i, Bis: 3*i + 1})
		}
		return st
	}
	if err := d.SetzeStreifen(liste(63, 63)); err != nil {
		t.Fatalf("63+63 Streifen (4096 Kacheln) abgelehnt: %v", err)
	}
	for _, f := range [][2]int{{64, 63}, {63, 64}, {100, 100}} {
		if err := d.SetzeStreifen(liste(f[0], f[1])); err == nil {
			t.Errorf("%d+%d Streifen ohne Fehler", f[0], f[1])
		}
	}
	if n := len(d.Streifen()); n != 126 {
		t.Errorf("%d Streifen nach Ablehnung, soll 126", n)
	}
	if err := d.SetzeStreifen(liste(0, 130)); err != nil {
		t.Errorf("130 Zeilenstreifen allein abgelehnt: %v", err)
	}
}
