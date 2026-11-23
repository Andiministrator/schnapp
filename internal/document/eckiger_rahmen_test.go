package document

import "testing"

// F-498 (Kritik A-2, B-6): EckigerRahmen liefert überlappungsfreie Rechtecke. Ist der Innenraum verschwunden (Breite oder
// Höhe höchstens eine Strichbreite), bleibt genau ein Rechteck gleich Grenzen(); sonst vier, deren Flächen sich zu
// „Außen minus Innen“ summieren und die sich nicht überlappen.
func TestEckigerRahmenSchmal(t *testing.T) {
	for _, c := range []struct {
		name    string
		bereich Rechteck
		eins    bool
	}{
		{"breit", R(Pt(30, 30), Pt(80, 70)), false},
		{"schmaler als der Strich in x", R(Pt(40, 30), Pt(44, 80)), true},
		{"schmaler als der Strich in y", R(Pt(30, 40), Pt(80, 44)), true},
		{"genau eine Strichbreite in x", R(Pt(30, 30), Pt(38, 70)), true},
		{"genau eine Strichbreite in y", R(Pt(30, 30), Pt(80, 38)), true},
		{"knapp mit Innenraum", R(Pt(30, 30), Pt(38.5, 70)), false},
		{"ohne Fläche", R(Pt(100, 100), Pt(300, 100)), true},
	} {
		k := Kasten{Bereich: c.bereich, Strich: Strich{Breite: 8}}
		rs := EckigerRahmen(c.bereich, 8)
		if c.eins {
			if len(rs) != 1 || rs[0] != k.Grenzen() {
				t.Errorf("%s: %v, Soll genau ein Rechteck gleich Grenzen() %v", c.name, rs, k.Grenzen())
			}
			continue
		}
		if len(rs) != 4 {
			t.Fatalf("%s: %d Rechtecke, Soll 4", c.name, len(rs))
		}
		summe := 0.0
		for i, r := range rs {
			if r.Breite() <= 0 || r.Hoehe() <= 0 {
				t.Errorf("%s: Rechteck %d ohne Fläche: %v", c.name, i, r)
			}
			summe += r.Breite() * r.Hoehe()
			for j := i + 1; j < len(rs); j++ {
				s := rs[j]
				if r.Min.X < s.Max.X && s.Min.X < r.Max.X && r.Min.Y < s.Max.Y && s.Min.Y < r.Max.Y {
					t.Errorf("%s: Rechtecke %d und %d überlappen (%v, %v)", c.name, i, j, r, s)
				}
			}
		}
		g := k.Grenzen()
		innen := (c.bereich.Breite() - 8) * (c.bereich.Hoehe() - 8)
		if soll := g.Breite()*g.Hoehe() - innen; summe < soll-1e-9 || summe > soll+1e-9 {
			t.Errorf("%s: Flächensumme %v, Soll Außen minus Innen %v", c.name, summe, soll)
		}
	}
}
