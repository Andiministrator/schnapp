package main

import (
	"math"

	"gioui.org/f32"
)

// pfeil ist eine quadratische Bézier-Kurve von A nach B mit Kontrollpunkt K, in Bildkoordinaten.
// Der Nutzer zieht nicht am Kontrollpunkt, sondern am Griff: dem Kurvenpunkt bei t = 0,5.
type pfeil struct {
	A, B, K f32.Point
}

// gerade liefert einen ungebogenen Pfeil von a nach b.
func gerade(a, b f32.Point) pfeil {
	return pfeil{A: a, B: b, K: mitte(a, b)}
}

func mitte(a, b f32.Point) f32.Point { return a.Add(b).Mul(0.5) }

// punkt liefert den Kurvenpunkt bei t ∈ [0,1].
func (p pfeil) punkt(t float32) f32.Point {
	u := 1 - t
	return p.A.Mul(u * u).Add(p.K.Mul(2 * u * t)).Add(p.B.Mul(t * t))
}

// griff ist der Kurvenpunkt bei t = 0,5: (A + 2K + B) / 4.
func (p pfeil) griff() f32.Point { return p.punkt(0.5) }

// setzeGriff legt den Kontrollpunkt so, dass die Kurve durch h läuft: K = 2h − (A+B)/2.
func (p *pfeil) setzeGriff(h f32.Point) {
	p.K = h.Mul(2).Sub(mitte(p.A, p.B))
}

// richtungAmEnde ist die normierte Tangente bei B (Richtung der Pfeilspitze).
func (p pfeil) richtungAmEnde() f32.Point {
	d := p.B.Sub(p.K)
	if laenge(d) < 1e-3 {
		d = p.B.Sub(p.A)
	}
	l := laenge(d)
	if l < 1e-3 {
		return f32.Pt(1, 0)
	}
	return d.Mul(1 / l)
}

func laenge(p f32.Point) float32 {
	return float32(math.Hypot(float64(p.X), float64(p.Y)))
}
