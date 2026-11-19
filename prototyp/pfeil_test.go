package main

import (
	"testing"

	"gioui.org/f32"
)

func nah(a, b f32.Point) bool { return laenge(a.Sub(b)) < 1e-4 }

func TestGriff(t *testing.T) {
	p := gerade(f32.Pt(0, 0), f32.Pt(100, 0))
	if !nah(p.griff(), f32.Pt(50, 0)) {
		t.Fatalf("Griff eines geraden Pfeils %v, Soll (50,0)", p.griff())
	}
	// Griff nach (50,40) ziehen: K = 2·(50,40) − (50,0) = (50,80).
	p.setzeGriff(f32.Pt(50, 40))
	if !nah(p.K, f32.Pt(50, 80)) {
		t.Errorf("Kontrollpunkt %v, Soll (50,80)", p.K)
	}
	if !nah(p.griff(), f32.Pt(50, 40)) {
		t.Errorf("Kurve läuft nicht durch den Griff: %v", p.griff())
	}
	if !nah(p.punkt(0), p.A) || !nah(p.punkt(1), p.B) {
		t.Errorf("Endpunkte verschoben: %v %v", p.punkt(0), p.punkt(1))
	}
}

func TestRichtungAmEnde(t *testing.T) {
	p := gerade(f32.Pt(0, 0), f32.Pt(0, 10))
	if !nah(p.richtungAmEnde(), f32.Pt(0, 1)) {
		t.Errorf("Richtung %v, Soll (0,1)", p.richtungAmEnde())
	}
	// K fällt auf B: Richtung von A nach B.
	q := pfeil{A: f32.Pt(0, 0), B: f32.Pt(3, 4), K: f32.Pt(3, 4)}
	if !nah(q.richtungAmEnde(), f32.Pt(0.6, 0.8)) {
		t.Errorf("Richtung %v, Soll (0.6,0.8)", q.richtungAmEnde())
	}
	// Alles auf einem Punkt: feste Richtung statt NaN.
	r := gerade(f32.Pt(5, 5), f32.Pt(5, 5))
	if !nah(r.richtungAmEnde(), f32.Pt(1, 0)) {
		t.Errorf("Richtung %v, Soll (1,0)", r.richtungAmEnde())
	}
}
