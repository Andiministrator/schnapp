// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"

	"github.com/Andiministrator/schnapp/document"
)

// Treffer liefert die Kennung des obersten Objekts, das den Originalpunkt p trifft; toleranz ist in Pixeln des
// Originals (Fenster-Toleranz durch Zoom). Oben liegt, was zuletzt eingefügt wurde.
func Treffer(objekte []document.Objekt, p document.Punkt, toleranz float64) (int, bool) {
	for i := len(objekte) - 1; i >= 0; i-- {
		if trifft(objekte[i], p, toleranz) {
			return objekte[i].Kennung(), true
		}
	}
	return 0, false
}

func trifft(o document.Objekt, p document.Punkt, tol float64) bool {
	switch v := o.(type) {
	case document.Pfeil:
		halb := v.Strich.Breite/2 + tol
		if abstandZurKurve(v, p) <= halb {
			return true
		}
		// Die Spitze ragt über B hinaus (Länge wie beim Zeichnen, mindestens 8 px).
		g := spitzenLaenge(v.Strich.Breite)
		if p.Minus(v.B).Laenge() <= g+tol {
			return true
		}
		// Die Ecken der Spitze liegen bis g·√5/2 von B: die Kanten des Dreiecks zählen mit.
		r := v.RichtungAmEnde()
		n := document.Pt(-r.Y, r.X)
		fuss := v.B.Minus(r.Mal(g))
		e1, e2 := fuss.Plus(n.Mal(g/2)), fuss.Minus(n.Mal(g/2))
		return abstandZurStrecke(p, v.B, e1) <= tol || abstandZurStrecke(p, v.B, e2) <= tol ||
			abstandZurStrecke(p, e1, e2) <= tol
	case document.Kasten:
		r := v.Bereich
		halb := v.Strich.Breite/2 + tol
		aussen := document.Rechteck{Min: r.Min.Minus(document.Pt(halb, halb)), Max: r.Max.Plus(document.Pt(halb, halb))}
		if !aussen.Enthaelt(p) {
			return false
		}
		if v.Fuellung.A > 0 {
			return true
		}
		innen := document.Rechteck{Min: r.Min.Plus(document.Pt(halb, halb)), Max: r.Max.Minus(document.Pt(halb, halb))}
		return innen.Min.X >= innen.Max.X || innen.Min.Y >= innen.Max.Y || !innen.Enthaelt(p)
	case document.Nummer:
		return p.Minus(v.Mitte).Laenge() <= v.Radius+tol
	default:
		g := o.Grenzen()
		return document.Rechteck{Min: g.Min.Minus(document.Pt(tol, tol)), Max: g.Max.Plus(document.Pt(tol, tol))}.Enthaelt(p)
	}
}

// spitzenLaenge ist die Länge der Pfeilspitze: 4·Breite, mindestens 8 Pixel.
func spitzenLaenge(breite float64) float64 { return math.Max(8, 4*breite) }

// abstandZurKurve nähert den Abstand von p zur Bézier-Kurve über 32 Strecken an.
func abstandZurKurve(pf document.Pfeil, p document.Punkt) float64 {
	const n = 32
	best := math.Inf(1)
	a := pf.Punkt(0)
	for i := 1; i <= n; i++ {
		b := pf.Punkt(float64(i) / n)
		best = math.Min(best, abstandZurStrecke(p, a, b))
		a = b
	}
	return best
}

func abstandZurStrecke(p, a, b document.Punkt) float64 {
	ab := b.Minus(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	if l2 == 0 {
		return p.Minus(a).Laenge()
	}
	t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / l2
	t = math.Max(0, math.Min(1, t))
	return p.Minus(a.Plus(ab.Mal(t))).Laenge()
}

// Verschiebe liefert das um d verschobene Objekt (gleiche Kennung). Unbekannte Arten bleiben unverändert.
func Verschiebe(o document.Objekt, d document.Punkt) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		v.A, v.B, v.K = v.A.Plus(d), v.B.Plus(d), v.K.Plus(d)
		return v
	case document.Kasten:
		v.Bereich = document.Rechteck{Min: v.Bereich.Min.Plus(d), Max: v.Bereich.Max.Plus(d)}
		return v
	case document.Text:
		v.Ort = v.Ort.Plus(d)
		return v
	case document.Nummer:
		v.Mitte = v.Mitte.Plus(d)
		return v
	}
	return o
}
