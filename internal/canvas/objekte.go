// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
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
		// Schaft und Köpfe wie gezeichnet (document.Pfeil.Form, ADR-0021 Nachtrag 19).
		f := v.Form()
		hw := v.Strich.Breite / 2
		for i := 1; i < len(f.Schaft); i++ {
			if abstandZurStrecke(p, f.Schaft[i-1], f.Schaft[i]) <= hw+tol {
				return true
			}
		}
		if len(f.Schaft) == 1 && p.Minus(f.Schaft[0]).Laenge() <= hw+tol {
			return true
		}
		for _, k := range f.Koepfe {
			if imVieleck(p, k.Umriss(hw), tol) {
				return true
			}
		}
		return false
	case document.Kasten:
		if v.Ellipse() { // ADR-0047 Punkt 3: mit Füllung im Oval, sonst auf dem Strich
			return trifftEllipse(v, p, tol)
		}
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
	case document.Lupe: // Quelle, Ziel oder Verbindungslinie (ADR-0033 Punkt 8)
		_, ok := lupenTeil(v, p, tol)
		return ok
	case document.Nummer:
		if p.Minus(v.Mitte).Laenge() <= v.Radius+tol {
			return true
		}
		// Ziffern, die über den Kreis hinausragen (ab drei Stellen), zählen mit (F-172).
		if z, ok := document.NummerTinte(v); ok {
			return document.Rechteck{Min: z.Min.Minus(document.Pt(tol, tol)), Max: z.Max.Plus(document.Pt(tol, tol))}.Enthaelt(p)
		}
		return false
	case document.Text:
		g := v.Grenzen()
		if (document.Rechteck{Min: g.Min.Minus(document.Pt(tol, tol)), Max: g.Max.Plus(document.Pt(tol, tol))}).Enthaelt(p) {
			return true
		}
		// Sprechblase: das Dreieck bis zur Spitze zählt mit (ADR-0021 Nachtrag 13 Punkt 4), die Spitze selbst mit Toleranz.
		return v.Spitze != nil && (v.Enthaelt(p) || p.Minus(*v.Spitze).Laenge() <= tol)
	default:
		g := o.Grenzen()
		return document.Rechteck{Min: g.Min.Minus(document.Pt(tol, tol)), Max: g.Max.Plus(document.Pt(tol, tol))}.Enthaelt(p)
	}
}

// trifftEllipse: mit Füllung im Umriss (oder höchstens halbe Strichbreite plus tol davon), ohne Füllung nur in diesem
// Abstand vom Umriss (ADR-0047 Punkt 3).
func trifftEllipse(k document.Kasten, p document.Punkt, tol float64) bool {
	halb := k.Strich.Breite/2 + tol
	r := k.Bereich
	aussen := document.Rechteck{Min: r.Min.Minus(document.Pt(halb, halb)), Max: r.Max.Plus(document.Pt(halb, halb))}
	if !aussen.Enthaelt(p) {
		return false
	}
	if k.Fuellung.A > 0 && document.EllipseEnthaelt(r, p) {
		return true
	}
	u := document.EllipsenUmriss(r)
	for i := range u {
		if abstandZurStrecke(p, u[i], u[(i+1)%len(u)]) <= halb {
			return true
		}
	}
	return false
}

// imVieleck meldet, ob p im einfachen Vieleck u liegt oder höchstens tol von seinem Rand.
func imVieleck(p document.Punkt, u []document.Punkt, tol float64) bool {
	innen := false
	for i, j := 0, len(u)-1; i < len(u); j, i = i, i+1 {
		a, b := u[i], u[j]
		if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			innen = !innen
		}
		if abstandZurStrecke(p, a, b) <= tol {
			return true
		}
	}
	return innen
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
		if v.Spitze != nil { // Schild und Spitze gemeinsam; ein neuer Zeiger, der alte gehört dem Rückgängig-Stand
			sp := v.Spitze.Plus(d)
			v.Spitze = &sp
		}
		return v
	case document.Nummer:
		v.Mitte = v.Mitte.Plus(d)
		return v
	case document.Lupe: // beide Rechtecke; einen Teil allein verschiebt zieheLupenTeil
		v.Quelle = document.Rechteck{Min: v.Quelle.Min.Plus(d), Max: v.Quelle.Max.Plus(d)}
		v.Ziel = document.Rechteck{Min: v.Ziel.Min.Plus(d), Max: v.Ziel.Max.Plus(d)}
		return v
	}
	return o
}
