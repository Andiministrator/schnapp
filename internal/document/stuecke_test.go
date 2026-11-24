// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

func stueckDokument(t *testing.T, z image.Rectangle, st []Streifen) *Dokument {
	t.Helper()
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAnsicht(z, st); err != nil {
		t.Fatal(err)
	}
	return d
}

// Die Stücke decken genau die sichtbaren Bereiche, und ihr Versatz ist ZuAnsicht in jedem Bereich (F-739).
func TestStueckeWieZuAnsicht(t *testing.T) {
	d := stueckDokument(t, image.Rect(10, 5, 390, 290), []Streifen{
		{Von: 100, Bis: 140}, {Von: 200, Bis: 210}, {Zeilen: true, Von: 50, Bis: 80},
	})
	st := d.Stuecke(R(Pt(0, 0), Pt(400, 300)))
	if len(st) != 3*2 {
		t.Fatalf("%d Stücke, erwartet 6: %+v", len(st), st)
	}
	soll := []Rechteck{
		R(Pt(10, 5), Pt(100, 50)), R(Pt(140, 5), Pt(200, 50)), R(Pt(210, 5), Pt(390, 50)),
		R(Pt(10, 80), Pt(100, 290)), R(Pt(140, 80), Pt(200, 290)), R(Pt(210, 80), Pt(390, 290)),
	}
	for i, s := range st {
		if s.Quelle != soll[i] {
			t.Errorf("Stück %d: Quelle %v, erwartet %v", i, s.Quelle, soll[i])
		}
		for _, p := range []Punkt{s.Quelle.Min, s.Quelle.Max.Minus(Pt(0.5, 0.5)), Pt((s.Quelle.Min.X+s.Quelle.Max.X)/2, s.Quelle.Min.Y+1)} {
			if got, want := p.Plus(s.Versatz), d.ZuAnsicht(p); got != want {
				t.Errorf("Stück %d, %v: Versatz ergibt %v, ZuAnsicht %v", i, p, got, want)
			}
		}
	}
	// Ziel liegt in der Ansicht und die Ziele stoßen ohne Lücke aneinander.
	g := d.Groesse()
	if st[2].Ziel().Max != Pt(float64(g.X), 45) || st[5].Ziel().Max != Pt(float64(g.X), float64(g.Y)) {
		t.Errorf("Ziele: %v, %v; Ansicht %v", st[2].Ziel(), st[5].Ziel(), g)
	}
	if st[0].Ziel().Max.X != st[1].Ziel().Min.X || st[1].Ziel().Max.X != st[2].Ziel().Min.X {
		t.Errorf("Ziele stoßen nicht aneinander: %v %v %v", st[0].Ziel(), st[1].Ziel(), st[2].Ziel())
	}
}

// Nur Bereiche, die das Rechteck mit positiver Fläche schneiden, zählen; ein Rechteck ganz im Streifen hat keine.
func TestStueckeNurBeruehrte(t *testing.T) {
	d := stueckDokument(t, image.Rect(0, 0, 400, 300), []Streifen{{Von: 100, Bis: 140}})
	for _, f := range []struct {
		r Rechteck
		n int
	}{
		{R(Pt(20, 20), Pt(90, 40)), 1},
		{R(Pt(90, 20), Pt(150, 40)), 2},
		{R(Pt(105, 20), Pt(150, 40)), 1},
		{R(Pt(105, 20), Pt(135, 40)), 0},
		{R(Pt(20, 20), Pt(100, 40)), 1}, // endet genau an der Naht
		{R(Pt(140, 20), Pt(150, 40)), 1},
	} {
		if n := len(d.Stuecke(f.r)); n != f.n {
			t.Errorf("%v: %d Stücke, erwartet %d", f.r, n, f.n)
		}
		if (len(d.Stuecke(f.r)) > 0) != d.Sichtbar(f.r) {
			t.Errorf("%v: Stücke und Sichtbar widersprechen sich", f.r)
		}
	}
	if st := d.Stuecke(R(Pt(105, 20), Pt(150, 40))); st[0].Versatz != Pt(-40, 0) {
		t.Errorf("Versatz hinter der Naht: %v", st[0].Versatz)
	}
}

// Ein Text, der knapp aus einem Streifen ragt, hat zwei Stücke; einer ganz davor eins.
func TestTextStuecke(t *testing.T) {
	d := stueckDokument(t, image.Rect(0, 0, 400, 300), []Streifen{{Von: 100, Bis: 200}})
	text := Text{Ort: Pt(40, 50), Inhalt: "Geheim lang", Schriftgrad: 20, Farbe: color.NRGBA{A: 255}}
	r := textReichweite(text, text.Grenzen())
	if !(r.Min.X < 100 && r.Max.X > 100) {
		t.Fatalf("Testtext reicht nicht über die Naht: %v", r)
	}
	if n := len(d.TextStuecke(text)); n != 1 {
		t.Errorf("Text bis in den Streifen: %d Stücke, erwartet 1", n)
	}
	text.Ort.X = 190 - (r.Max.X - text.Ort.X) + 30 // ragt 30 px über das Streifenende hinaus
	st := d.TextStuecke(text)
	if len(st) != 1 || st[0].Quelle.Min.X != 200 {
		t.Errorf("Text aus dem Streifen: %+v", st)
	}
	text.Ort.X, text.Inhalt = 60, "Geheim lang und noch viel länger"
	if st := d.TextStuecke(text); len(st) != 2 {
		t.Errorf("Text über den ganzen Streifen: %d Stücke, erwartet 2", len(st))
	}
}
