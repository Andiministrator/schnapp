// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"fmt"
	"math"

	"github.com/Andiministrator/schnapp/draw/geo"
	"github.com/Andiministrator/schnapp/draw/svg"
)

// strichArt: wie ein Strich gezogen wird (SVG stroke-*).
type strichArt struct {
	breite      float64
	ende        string // butt, round, square
	ecke        string // miter, round, bevel
	gehrung     float64
	strichelung []float64
	// nurEcke: das erste Stück des Zugs nicht zeichnen und dort kein Ende setzen; es liefert nur die Richtung für die
	// Ecke am zweiten Punkt (Anfang eines Strichs auf einem geschlossenen Pfad).
	nurEcke bool
}

// strichPolygone baut den Umriss eines Strichs als Vereinigung einzelner Polygone (Stücke, Ecken, Enden), alle
// gleich herum orientiert, damit sie sich unter nonzero zu einer Fläche fügen. schritt: größter Abstand zweier
// Punkte auf Rundungen (Benutzereinheiten).
func strichPolygone(tps []svg.Teilpfad, art strichArt, schritt float64) ([][]geo.P, error) {
	var aus [][]geo.P
	hw := art.breite / 2
	if hw <= 0 {
		return nil, nil
	}
	nullFehler := fmt.Errorf("Strich der Länge 0 mit stroke-linecap square nicht unterstützt (rsvg zeichnet ihn je nach Maßstab oder nicht)")
	for _, tp := range tps {
		pts := ohneDoppelte(tp.Punkte, tp.Geschlossen)
		if len(pts) == 1 {
			// Teilpfad der Länge 0: bei round ein Punkt wie bei rsvg
			if art.ende == "square" {
				return nil, nullFehler
			}
			aus = append(aus, punktEnde(pts[0], hw, art.ende, schritt)...)
			continue
		}
		if len(pts) < 2 {
			continue
		}
		if len(art.strichelung) > 0 {
			for _, stueck := range stricheln(pts, tp.Geschlossen, art.strichelung) {
				st := ohneDoppelte(stueck.pts, false)
				if len(st) == 1 {
					if art.ende == "square" {
						return nil, nullFehler
					}
					aus = append(aus, punktEnde(st[0], hw, art.ende, schritt)...)
					continue
				}
				if stueck.vorher != nil && geo.Dist(*stueck.vorher, st[0]) > 1e-9 && knick(*stueck.vorher, st[0], st[1]) {
					mitEcke := art
					mitEcke.nurEcke = true
					aus = append(aus, zug(append([]geo.P{*stueck.vorher}, st...), false, hw, mitEcke, schritt)...)
					continue
				}
				aus = append(aus, zug(st, false, hw, art, schritt)...)
			}
			continue
		}
		aus = append(aus, zug(pts, tp.Geschlossen, hw, art, schritt)...)
	}
	return aus, nil
}

// punktEnde: Enden eines Strichs der Länge 0: bei round ein Punkt, bei butt nichts (wie rsvg; square meldet der
// Aufrufer als Fehler).
func punktEnde(p geo.P, hw float64, ende string, schritt float64) [][]geo.P {
	if ende == "round" {
		if q := gerichtet(kreis(p, hw, schritt)); q != nil {
			return [][]geo.P{q}
		}
	}
	return nil
}

func ohneDoppelte(pts []geo.P, geschlossen bool) []geo.P {
	var aus []geo.P
	for _, p := range pts {
		if len(aus) > 0 && geo.Dist(aus[len(aus)-1], p) < 1e-9 {
			continue
		}
		aus = append(aus, p)
	}
	if geschlossen && len(aus) > 2 && geo.Dist(aus[0], aus[len(aus)-1]) < 1e-9 {
		aus = aus[:len(aus)-1]
	}
	return aus
}

// knick: biegt der Zug in b ab (a -> b -> c)? Gerade Kanten sind in dichte Punkte zerlegt; an einem Zwischenpunkt
// einer Geraden ist die Ecke leer und würde nur das Ende verschlucken.
func knick(a, b, c geo.P) bool {
	d1x, d1y := b.X-a.X, b.Y-a.Y
	d2x, d2y := c.X-b.X, c.Y-b.Y
	l := math.Hypot(d1x, d1y) * math.Hypot(d2x, d2y)
	if l == 0 {
		return false
	}
	kreuz := (d1x*d2y - d1y*d2x) / l
	skalar := (d1x*d2x + d1y*d2y) / l
	return math.Abs(kreuz) > 1e-6 || skalar < 0
}

// strichStueck: ein Strich der Strichelung; vorher ist der Stützpunkt vor seinem Anfang, wenn die Lücke davor genau
// an einem Stützpunkt endete. Dann setzt Cairo dort eine Ecke statt eines Endes (gemessen 18.11.2026 an rsvg).
type strichStueck struct {
	pts    []geo.P
	vorher *geo.P
}

// stricheln zerlegt einen Zug nach dem Muster (Strich, Lücke, ...) ohne Versatz in offene Züge. Fällt ein Wechsel
// genau auf einen Stützpunkt, gilt er als Anfang des nächsten Stücks (sonst entstehen Striche der
// Länge 0 mitten im Zug).
func stricheln(pts []geo.P, geschlossen bool, muster []float64) []strichStueck {
	if len(muster)%2 == 1 {
		muster = append(append([]float64{}, muster...), muster...)
	}
	gesamt := 0.0
	for _, v := range muster {
		gesamt += v
	}
	if gesamt <= 0 {
		return []strichStueck{{pts: pts}}
	}
	if geschlossen {
		pts = append(append([]geo.P{}, pts...), pts[0])
	}
	var aus []strichStueck
	idx, rest, an := 0, muster[0], true
	akt := []geo.P{pts[0]}
	var vorher *geo.P
	letzterWechsel := -1 // Index des Stücks, an dessen Ende zuletzt eine Lücke genau endete
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		l := geo.Dist(a, b)
		pos := 0.0
		for l-pos >= rest && !(l-pos == rest && i+2 < len(pts)) {
			pos += rest
			t := pos / l
			p := geo.P{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
			if pos == 0 {
				p = a
			}
			if an {
				akt = append(akt, p)
				aus = append(aus, strichStueck{pts: akt, vorher: vorher})
				akt, vorher = nil, nil
			} else {
				akt = []geo.P{p}
				vorher = nil
				if pos == 0 && i > 0 {
					v := pts[i-1]
					vorher = &v
				}
				if pos == l && i+2 == len(pts) {
					letzterWechsel = i
				}
			}
			an = !an
			idx = (idx + 1) % len(muster)
			rest = muster[idx]
		}
		rest -= l - pos
		if an {
			akt = append(akt, b)
		}
	}
	if an && len(akt) > 0 {
		switch {
		case geschlossen && len(aus) > 0 && letzterWechsel >= 0 && len(akt) <= 2:
			// Lücke endet genau am Schlusspunkt: der erste Strich bekommt dort eine Ecke
			v := pts[len(pts)-2]
			aus[0].vorher = &v
		case geschlossen && len(aus) > 0:
			// Strich läuft über den Startpunkt: letzter und erster Strich sind einer
			aus[0].pts = append(akt, aus[0].pts[1:]...)
		default:
			aus = append(aus, strichStueck{pts: akt, vorher: vorher})
		}
	}
	return aus
}

func flaeche(p []geo.P) float64 {
	a := 0.0
	for i := range p {
		q := p[(i+1)%len(p)]
		a += p[i].X*q.Y - q.X*p[i].Y
	}
	return a / 2
}

// gerichtet: alle Polygone gleich herum (positive Fläche); entartete fallen weg.
func gerichtet(p []geo.P) []geo.P {
	a := flaeche(p)
	if math.Abs(a) < 1e-12 {
		return nil
	}
	if a < 0 {
		for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
			p[i], p[j] = p[j], p[i]
		}
	}
	return p
}

// bogenPunkte: Punkte auf dem Kreis um m mit Radius r von Winkel w0 nach w1 (ohne Anfang, mit Ende).
func bogenPunkte(m geo.P, r, w0, w1, schritt float64) []geo.P {
	n := max(1, int(math.Ceil(math.Abs(w1-w0)*r/schritt)))
	aus := make([]geo.P, 0, n)
	for k := 1; k <= n; k++ {
		w := w0 + (w1-w0)*float64(k)/float64(n)
		aus = append(aus, geo.P{X: m.X + r*math.Cos(w), Y: m.Y + r*math.Sin(w)})
	}
	return aus
}

func kreis(m geo.P, r, schritt float64) []geo.P {
	return bogenPunkte(m, r, 0, 2*math.Pi, schritt)
}

func zug(pts []geo.P, geschlossen bool, hw float64, art strichArt, schritt float64) [][]geo.P {
	var aus [][]geo.P
	add := func(p []geo.P) {
		if q := gerichtet(p); q != nil {
			aus = append(aus, q)
		}
	}
	n := len(pts)
	stuecke := n - 1
	if geschlossen {
		stuecke = n
	}
	richt := make([]geo.P, stuecke)
	for i := 0; i < stuecke; i++ {
		a, b := pts[i], pts[(i+1)%n]
		l := geo.Dist(a, b)
		if l < 1e-12 { // ohneDoppelte schließt das aus; Schutz gegen NaN
			l = 1e-12
		}
		d := geo.P{X: (b.X - a.X) / l, Y: (b.Y - a.Y) / l}
		richt[i] = d
		if i == 0 && art.nurEcke {
			continue
		}
		nx, ny := -d.Y*hw, d.X*hw
		add([]geo.P{{X: a.X + nx, Y: a.Y + ny}, {X: b.X + nx, Y: b.Y + ny}, {X: b.X - nx, Y: b.Y - ny}, {X: a.X - nx, Y: a.Y - ny}})
	}
	ecke := func(p geo.P, d1, d2 geo.P) {
		kreuz := d1.X*d2.Y - d1.Y*d2.X
		skalar := d1.X*d2.X + d1.Y*d2.Y
		if math.Abs(kreuz) < 1e-12 && skalar > 0 {
			return
		}
		vz := 1.0
		if kreuz > 0 {
			vz = -1
		}
		n1 := geo.P{X: -d1.Y * vz, Y: d1.X * vz}
		n2 := geo.P{X: -d2.Y * vz, Y: d2.X * vz}
		o1 := geo.P{X: p.X + n1.X*hw, Y: p.Y + n1.Y*hw}
		o2 := geo.P{X: p.X + n2.X*hw, Y: p.Y + n2.Y*hw}
		switch art.ecke {
		case "round":
			w0 := math.Atan2(n1.Y, n1.X)
			w1 := math.Atan2(n2.Y, n2.X)
			dw := w1 - w0
			for dw > math.Pi {
				dw -= 2 * math.Pi
			}
			for dw < -math.Pi {
				dw += 2 * math.Pi
			}
			if math.Abs(dw) > math.Pi-1e-9 {
				// U-Wende: der Bogen muss durch p + d1*hw laufen, sonst liegt er innen
				if math.Cos(w0+dw/2-math.Atan2(d1.Y, d1.X)) < 0 {
					dw = -dw
				}
			}
			poly := []geo.P{p, o1}
			poly = append(poly, bogenPunkte(p, hw, w0, w0+dw, schritt)...)
			add(poly)
		case "miter":
			// Drehwinkel zwischen den Stücken; Gehrungslänge / Strichbreite = 1 / cos(dreh/2)
			cosHalb := math.Sqrt(math.Max(0, (1+skalar)/2))
			if cosHalb > 1e-9 && 1/cosHalb <= art.gehrung {
				bx, by := n1.X+n2.X, n1.Y+n2.Y
				bl := math.Hypot(bx, by)
				if bl > 1e-12 {
					l := hw / cosHalb
					mp := geo.P{X: p.X + bx/bl*l, Y: p.Y + by/bl*l}
					add([]geo.P{p, o1, mp, o2})
					return
				}
			}
			add([]geo.P{p, o1, o2})
		default: // bevel
			add([]geo.P{p, o1, o2})
		}
	}
	for i := 1; i < n-1; i++ {
		ecke(pts[i], richt[i-1], richt[i])
	}
	if geschlossen {
		ecke(pts[n-1], richt[n-2], richt[n-1])
		ecke(pts[0], richt[n-1], richt[0])
		return aus
	}
	ende := func(p, d geo.P) { // d zeigt aus dem Zug hinaus
		switch art.ende {
		case "round":
			add(kreis(p, hw, schritt))
		case "square":
			nx, ny := -d.Y*hw, d.X*hw
			q := geo.P{X: p.X + d.X*hw, Y: p.Y + d.Y*hw}
			add([]geo.P{{X: p.X + nx, Y: p.Y + ny}, {X: q.X + nx, Y: q.Y + ny}, {X: q.X - nx, Y: q.Y - ny}, {X: p.X - nx, Y: p.Y - ny}})
		}
	}
	if !art.nurEcke {
		ende(pts[0], geo.P{X: -richt[0].X, Y: -richt[0].Y})
	}
	ende(pts[n-1], richt[stuecke-1])
	return aus
}
