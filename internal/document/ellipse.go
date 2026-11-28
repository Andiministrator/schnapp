// SPDX-License-Identifier: GPL-3.0-or-later

package document

// Ellipse als Form des Kastens (ADR-0047): eine Rechnung des Umrisses für Renderer, Editor, Treffer und Muster.

import "math"

// Kastenform ist die Form eines Kastens: Rechteck (Nullwert) oder die Ellipse in seinem Bereich (ADR-0047 Punkt 1).
type Kastenform uint8

const (
	FormRechteck Kastenform = iota
	FormEllipse
)

// Ellipse meldet, ob der Kasten als Ellipse gezeichnet wird: Form Ellipse und ein Bereich mit Breite und Höhe. Ohne
// Fläche zeichnen alle Wege den Balken des Kastens ohne Fläche (ADR-0047 Punkt 2).
func (k Kasten) Ellipse() bool {
	return k.Form == FormEllipse && k.Bereich.Breite() > 0 && k.Bereich.Hoehe() > 0
}

// Grenzen der Abtastung (ADR-0047 Punkt 2): Sehne höchstens ellipseSehne vom Bogen, höchstens ellipseViertelMax Punkte
// je Viertel (ein Viertel mit Halbachse 32767 bräuchte rund 1000).
const (
	ellipseSehne      = 0.02
	ellipseViertelMax = 1024
)

// ellipseSchritte ist die Zahl der Schritte je Viertel für die Halbachsen a, b: Die Sehne zum Parameterschritt dt
// weicht höchstens max(a, b)·dt²/8 vom Bogen ab.
func ellipseSchritte(a, b float64) int {
	r := math.Max(a, b)
	if !(r > 0) {
		return 1
	}
	n := int(math.Ceil((math.Pi / 2) / math.Sqrt(8*ellipseSehne/r)))
	return min(max(n, 2), ellipseViertelMax)
}

// EllipsenViertel sind die vier Viertelbögen der Ellipse in r als offene Linienzüge von Scheitel zu Scheitel (rechts,
// unten, links, oben, im Uhrzeigersinn auf dem Bildschirm); das Ende eines Viertels ist der Anfang des nächsten.
func EllipsenViertel(r Rechteck) [][]Punkt {
	c := Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	a, b := r.Breite()/2, r.Hoehe()/2
	n := ellipseSchritte(a, b)
	viertel := make([][]Punkt, 4)
	for q := range 4 {
		pts := make([]Punkt, 0, n+1)
		for i := 0; i <= n; i++ {
			var cos, sin float64
			switch {
			case i == 0: // Scheitel exakt
				cos, sin = scheitel(q)
			case i == n:
				cos, sin = scheitel(q + 1)
			default:
				t := (float64(q) + float64(i)/float64(n)) * math.Pi / 2
				cos, sin = math.Cos(t), math.Sin(t)
			}
			pts = append(pts, Pt(c.X+a*cos, c.Y+b*sin))
		}
		viertel[q] = pts
	}
	return viertel
}

// scheitel ist (cos, sin) des Scheitels q (0 rechts, 1 unten, 2 links, 3 oben, 4 = 0).
func scheitel(q int) (float64, float64) {
	switch q % 4 {
	case 1:
		return 0, 1
	case 2:
		return -1, 0
	case 3:
		return 0, -1
	}
	return 1, 0
}

// EllipsenUmriss ist der geschlossene Umriss der Ellipse in r (ohne den Anfangspunkt am Ende zu wiederholen).
func EllipsenUmriss(r Rechteck) []Punkt {
	var pts []Punkt
	for _, v := range EllipsenViertel(r) {
		pts = append(pts, v[:len(v)-1]...)
	}
	return pts
}

// EllipsenAbschnitte teilt den Strich der Ellipse in r nach Strichart a und Strichstärke w in vier Viertel von
// Scheitel zu Scheitel, mit halbem Strich bzw. Punkt an jedem Scheitel (ADR-0047 Punkt 2, wie RundRahmenAbschnitte).
func EllipsenAbschnitte(r Rechteck, a Strichart, w float64) Strichmuster {
	return rahmenMuster(EllipsenViertel(r), a, w)
}

// EllipseEnthaelt meldet, ob p in der Ellipse in r liegt.
func EllipseEnthaelt(r Rechteck, p Punkt) bool {
	a, b := r.Breite()/2, r.Hoehe()/2
	if !(a > 0) || !(b > 0) {
		return false
	}
	x, y := (p.X-(r.Min.X+a))/a, (p.Y-(r.Min.Y+b))/b
	return x*x+y*y <= 1
}
