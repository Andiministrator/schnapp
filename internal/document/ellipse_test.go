// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image/color"
	"math"
	"testing"
)

// ADR-0047 Punkt 2: Die Viertel beginnen und enden exakt auf den Scheiteln und schließen lückenlos aneinander; die
// Abtastung bleibt in der Sehnengrenze und begrenzt.
func TestEllipsenViertelScheitel(t *testing.T) {
	r := R(Pt(10.5, 20.25), Pt(210.75, 120.5))
	c := Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	v := EllipsenViertel(r)
	soll := []Punkt{Pt(r.Max.X, c.Y), Pt(c.X, r.Max.Y), Pt(r.Min.X, c.Y), Pt(c.X, r.Min.Y)}
	for q := range 4 {
		if v[q][0] != soll[q] {
			t.Errorf("Viertel %d beginnt bei %v, Soll %v", q, v[q][0], soll[q])
		}
		if ende := v[q][len(v[q])-1]; ende != v[(q+1)%4][0] {
			t.Errorf("Viertel %d endet bei %v, das nächste beginnt bei %v", q, ende, v[(q+1)%4][0])
		}
	}
	a, b := r.Breite()/2, r.Hoehe()/2
	u := EllipsenUmriss(r)
	for i := range u {
		p, q := u[i], u[(i+1)%len(u)]
		m := Pt((p.X+q.X)/2-c.X, (p.Y+q.Y)/2-c.Y)
		// Abstand der Sehnenmitte zum Bogen, genähert über den normierten Radius.
		rho := math.Hypot(m.X/a, m.Y/b)
		if abw := (1 - rho) * math.Min(a, b); abw > ellipseSehne*1.5 {
			t.Fatalf("Sehne %d weicht %v vom Bogen ab", i, abw)
		}
	}
	if n := len(EllipsenViertel(R(Pt(0, 0), Pt(1e7, 1e7)))[0]); n != ellipseViertelMax+1 {
		t.Errorf("große Ellipse: %d Punkte je Viertel", n)
	}
}

// Eine unbekannte Form ist ein ungültiges Objekt (ADR-0047 Punkt 1).
func TestKastenformGeprueft(t *testing.T) {
	d, err := Neu(testbild(20, 20))
	if err != nil {
		t.Fatal(err)
	}
	k := Kasten{Bereich: R(Pt(1, 1), Pt(9, 9)), Strich: Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}, Form: FormEllipse + 1}
	if _, err := d.Hinzufuegen(k); err == nil {
		t.Error("unbekannte Form angenommen")
	}
	k.Form = FormEllipse
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Errorf("Ellipse abgelehnt: %v", err)
	}
}
