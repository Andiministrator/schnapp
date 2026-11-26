// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// ADR-0021 Nachtrag 25: Namen, Maße, Ausgleich und Überlappungsfreiheit der Stricharten.

func TestStrichartNamen(t *testing.T) {
	for _, a := range Stricharten {
		b, ok := StrichartAus(a.Name())
		if !ok || b != a {
			t.Errorf("%v: Name %q liest sich als %v, %v", a, a.Name(), b, ok)
		}
	}
	if _, ok := StrichartAus("wavy"); ok {
		t.Error("unbekannter Name gilt als Strichart")
	}
	if Strichart(9).Name() != "solid" {
		t.Error("unbekannte Art heißt nicht solid")
	}
	if (Strich{Breite: 2, Art: 3}).pruefe() == nil {
		t.Error("Strichart 3 gilt als gültig")
	}
	if (Strich{Breite: 2, Art: StrichGepunktet}).pruefe() != nil {
		t.Error("gepunktet gilt als ungültig")
	}
}

func TestStrichMasse(t *testing.T) {
	if s, l := StrichMasse(StrichGestrichelt, 6); s != 20 || l != 14 {
		t.Errorf("gestrichelt 6: %v/%v", s, l)
	}
	if s, l := StrichMasse(StrichGepunktet, 6); s != 0 || l != 14 {
		t.Errorf("gepunktet 6: %v/%v", s, l)
	}
}

// Offener Zug: ganzer Strich an beiden Enden, alle Striche gleich lang, alle Lücken gleich, Verhältnis wie vorgegeben.
func TestMusterOffen(t *testing.T) {
	for _, s := range []float64{21, 55, 100, 257.3, 1000} {
		for _, w := range []float64{1, 6, 13} {
			ab := musterAbschnitte(s, StrichGestrichelt, w, false)
			if ab[0].von != 0 || ab[len(ab)-1].bis != s {
				t.Errorf("s %v w %v: Enden %v, %v", s, w, ab[0], ab[len(ab)-1])
			}
			strich, luecke := StrichMasse(StrichGestrichelt, w)
			if s <= strich {
				if len(ab) != 1 {
					t.Errorf("s %v w %v: kurzer Zug nicht voll: %v", s, w, ab)
				}
				continue
			}
			l0 := ab[0].bis - ab[0].von
			for i, a := range ab {
				if math.Abs(a.bis-a.von-l0) > 1e-9 {
					t.Errorf("s %v w %v: Strich %d %v statt %v", s, w, i, a.bis-a.von, l0)
				}
				if i > 0 {
					g := a.von - ab[i-1].bis
					if math.Abs(g/l0-luecke/strich) > 1e-9 {
						t.Errorf("s %v w %v: Lücke %v zu Strich %v", s, w, g, l0)
					}
				}
			}
			if k := l0 / strich; k < 0.6 || k > 1.6 {
				t.Errorf("s %v w %v: Faktor %v", s, w, k)
			}
		}
	}
	p := musterAbschnitte(100, StrichGepunktet, 6, false)
	if p[0].von != 0 || p[len(p)-1].von != 100 {
		t.Errorf("gepunktet: Enden %v", p)
	}
	if p := musterAbschnitte(3, StrichGepunktet, 6, false); len(p) != 1 || p[0].von != 1.5 {
		t.Errorf("gepunktet kürzer als Strichstärke: %v", p)
	}
}

// Zwischen zwei Ecken: halber Strich an beiden Enden, symmetrisch.
func TestMusterZwischenEcken(t *testing.T) {
	for _, s := range []float64{40, 97, 300} {
		ab := musterAbschnitte(s, StrichGestrichelt, 6, true)
		if ab[0].von != 0 || ab[len(ab)-1].bis != s {
			t.Fatalf("s %v: Enden %v", s, ab)
		}
		for i := range ab {
			j := len(ab) - 1 - i
			if math.Abs(ab[i].von-(s-ab[j].bis)) > 1e-9 {
				t.Errorf("s %v: nicht symmetrisch: %v / %v", s, ab[i], ab[j])
			}
		}
		if len(ab) > 2 && math.Abs(2*(ab[0].bis-ab[0].von)-(ab[1].bis-ab[1].von)) > 1e-9 {
			t.Errorf("s %v: Eckstrich %v nicht halb so lang wie %v", s, ab[0], ab[1])
		}
	}
	// Eine Seite kürzer als ein Strich ist voll, ohne Lücke in der Mitte.
	if ab := musterAbschnitte(15, StrichGestrichelt, 6, true); len(ab) != 1 || ab[0].von != 0 || ab[0].bis != 15 {
		t.Errorf("kurze Seite: %v", ab)
	}
}

func TestStrichAbschnitteAufDemZug(t *testing.T) {
	pts := []Punkt{Pt(0, 0), Pt(100, 0), Pt(100, 80)}
	m := StrichAbschnitte(pts, StrichGestrichelt, 4)
	if len(m.Striche) < 3 || len(m.Punkte) != 0 {
		t.Fatalf("Striche %d, Punkte %d", len(m.Striche), len(m.Punkte))
	}
	if m.Striche[0][0] != pts[0] || m.Striche[len(m.Striche)-1][len(m.Striche[len(m.Striche)-1])-1] != pts[2] {
		t.Error("Muster beginnt oder endet nicht an den Enden des Zugs")
	}
	for _, z := range m.Striche {
		for _, p := range z {
			if !(p.Y == 0 && p.X >= 0 && p.X <= 100 || p.X == 100 && p.Y >= 0 && p.Y <= 80) {
				t.Errorf("Punkt %v liegt nicht auf dem Zug", p)
			}
		}
	}
	g := StrichAbschnitte(pts, StrichGepunktet, 4)
	if len(g.Striche) != 0 || g.Punkte[0] != pts[0] || g.Punkte[len(g.Punkte)-1] != pts[2] {
		t.Errorf("gepunktet: %v", g)
	}
}

// Eckiger Rahmen gestrichelt: Rechtecke überlappen nicht, liegen im vollen Rahmen, decken die vier Ecken, lassen Lücken.
func TestEckigerRahmenMuster(t *testing.T) {
	for _, w := range []float64{1, 6, 20} {
		r := R(Pt(10.5, 20.25), Pt(210, 150.75))
		balken, punkte := EckigerRahmenMuster(r, w, StrichGestrichelt)
		if len(punkte) != 0 || len(balken) < 8 {
			t.Fatalf("w %v: %d Balken, %d Punkte", w, len(balken), len(punkte))
		}
		voll := 0.0
		for _, q := range EckigerRahmen(r, w) {
			voll += q.Breite() * q.Hoehe()
		}
		summe := 0.0
		for i, a := range balken {
			summe += a.Breite() * a.Hoehe()
			for _, b := range balken[i+1:] {
				if a.Min.X < b.Max.X-1e-9 && b.Min.X < a.Max.X-1e-9 && a.Min.Y < b.Max.Y-1e-9 && b.Min.Y < a.Max.Y-1e-9 {
					t.Errorf("w %v: %v und %v überlappen", w, a, b)
				}
			}
			if a.Min.X < r.Min.X-w/2-1e-9 || a.Max.X > r.Max.X+w/2+1e-9 || a.Min.Y < r.Min.Y-w/2-1e-9 || a.Max.Y > r.Max.Y+w/2+1e-9 {
				t.Errorf("w %v: %v ragt aus dem Rahmen", w, a)
			}
		}
		if summe >= voll || summe < voll*0.4 {
			t.Errorf("w %v: Fläche %v von %v", w, summe, voll)
		}
		for _, e := range []Punkt{r.Min, Pt(r.Max.X, r.Min.Y), r.Max, Pt(r.Min.X, r.Max.Y)} {
			for _, d := range []Punkt{Pt(-w/2+1e-3, -w/2+1e-3), Pt(w/2-1e-3, w/2-1e-3), Pt(-w/2+1e-3, w/2-1e-3)} {
				q := e.Plus(d)
				drin := false
				for _, a := range balken {
					drin = drin || a.Enthaelt(q)
				}
				if !drin {
					t.Errorf("w %v: Ecke %v bei %v nicht gedeckt", w, e, q)
				}
			}
		}
	}
	// Gepunktet: Ecken als Punkte, jeder einmal; Abstand mindestens die Strichstärke.
	r := R(Pt(0, 0), Pt(100, 60))
	_, punkte := EckigerRahmenMuster(r, 4, StrichGepunktet)
	for _, e := range []Punkt{r.Min, Pt(r.Max.X, r.Min.Y), r.Max, Pt(r.Min.X, r.Max.Y)} {
		n := 0
		for _, p := range punkte {
			if p == e {
				n++
			}
		}
		if n != 1 {
			t.Errorf("Ecke %v %d-mal", e, n)
		}
	}
	for i, a := range punkte {
		for _, b := range punkte[i+1:] {
			if a.Minus(b).Laenge() < 4 {
				t.Errorf("Punkte %v und %v zu nah", a, b)
			}
		}
	}
	// Voll und ohne Innenraum: EckigerRahmen.
	if b, p := EckigerRahmenMuster(r, 4, StrichVoll); len(b) != 4 || p != nil {
		t.Errorf("voll: %v %v", b, p)
	}
	if b, _ := EckigerRahmenMuster(R(Pt(0, 0), Pt(3, 50)), 4, StrichGestrichelt); len(b) != 1 {
		t.Errorf("ohne Innenraum: %v", b)
	}
}

// Runder Rahmen: Striche über die Mitten der Rundungen sind ein Linienzug (auch über den Anfang), Punkte einmal.
func TestRundRahmenAbschnitte(t *testing.T) {
	r := R(Pt(0, 0), Pt(200, 120))
	m := RundRahmenAbschnitte(r, 20, StrichGestrichelt, 6)
	roh := RundesRechteck(r, 20)
	for c := range 4 {
		mitte := roh[c*13+6]
		n := 0
		for _, z := range m.Striche {
			for k, p := range z {
				if p.Minus(mitte).Laenge() < 1e-9 {
					n++
					if k == 0 || k == len(z)-1 {
						t.Errorf("Mitte der Rundung %d am Ende eines Strichs", c)
					}
				}
			}
		}
		if n != 1 {
			t.Errorf("Mitte der Rundung %d in %d Strichen", c, n)
		}
	}
	p := RundRahmenAbschnitte(r, 20, StrichGepunktet, 6)
	for i, a := range p.Punkte {
		for _, b := range p.Punkte[i+1:] {
			if a.Minus(b).Laenge() < 6 {
				t.Errorf("Punkte %v und %v zu nah", a, b)
			}
		}
	}
}

// F-840: Über MaxMusterStuecke Stücke wird der Strich voll (offen, rund, eckig).
func TestMusterObergrenze(t *testing.T) {
	m := StrichAbschnitte([]Punkt{Pt(-1e6, 0), Pt(1e6, 0)}, StrichGepunktet, 1)
	if !m.Ersatz || len(m.Striche) != 1 || len(m.Punkte) != 0 {
		t.Errorf("offen: %d Striche, %d Punkte, Ersatz %v", len(m.Striche), len(m.Punkte), m.Ersatz)
	}
	r := RundRahmenAbschnitte(R(Pt(-1e6, -1e6), Pt(1e6, 1e6)), 50, StrichGestrichelt, 1)
	if !r.Ersatz || len(r.Striche) != 1 {
		t.Errorf("rund: %d Striche, Ersatz %v", len(r.Striche), r.Ersatz)
	}
	if b, p := EckigerRahmenMuster(R(Pt(-1e6, -1e6), Pt(1e6, 1e6)), 1, StrichGepunktet); len(b) != 4 || p != nil {
		t.Errorf("eckig: %d Balken, %d Punkte", len(b), len(p))
	}
	if _, p := EckigerRahmenMuster(R(Pt(0, 0), Pt(8000, 8000)), 1, StrichGepunktet); len(p) < 7000 {
		t.Errorf("8000er-Rahmen gepunktet: nur %d Punkte", len(p))
	}
}
