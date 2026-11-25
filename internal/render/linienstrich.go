// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// bogenSehne ist der größte Abstand der Bögen an Gelenken und Enden eines Strichs von ihrer Sehne, in Pixeln (bis zur
// halben Breite 4250; darüber begrenzt bogenSchritt die Eckenzahl auf 1024 je Vollkreis).
const bogenSehne = 0.02

// bogenSchritt ist der Winkel zwischen zwei Ecken eines Bogens vom Radius r: so klein, dass die Sehne höchstens
// bogenSehne vom Bogen abweicht, höchstens π/4 und mindestens 2π/1024.
func bogenSchritt(r float64) float64 {
	s := math.Pi / 4
	if x := 1 - bogenSehne/r; x > math.Cos(s/2) {
		s = 2 * math.Acos(x)
	}
	return max(s, 2*math.Pi/1024)
}

// linienStrich sind die Teile des Strichs um einen Linienzug mit runden Gelenken und Enden (ADR-0023 Nachtrag 22,
// F-796): je Strecke ein Sechseck zwischen ihren Enden, je Gelenk ein Fächer auf der Außenseite, je offenem Ende eine
// halbe Scheibe. Die Teile überlappen nicht und schließen lückenlos aneinander (gemeinsame Kanten aus denselben
// Punkten); in einer Füllung heben sich ihre gemeinsamen Kanten im Rasterizer genau auf, und jedes Pixel ist so gedeckt
// wie die Fläche des Strichs. Vorher füllte der Renderer je Strecke ein Rechteck und je Punkt einen Kreis; an jedem
// Gelenk addierte sich deren Kantenglättung (äußere Randpixel bis 255 statt rund 128).
//
// Das Sechseck der Strecke i (pts[i] → pts[i+1]) läuft je Seite vom Ende bei pts[i] zum Ende bei pts[i+1] und über die
// beiden Punkte selbst zurück. Ein Ende liegt
//   - auf der Außenseite eines Gelenks und an einem offenen Ende senkrecht um die halbe Breite hw neben dem Punkt (dort
//     schließen Fächer bzw. Kappe an),
//   - auf der Innenseite eines Gelenks auf dem Schnittpunkt der beiden inneren Strichkanten (Gehrung, hw/cos(Knick/2)
//     weit auf der Winkelhalbierenden); beide Nachbarn teilen sich diese Kante.
//
// Faltung: Reichen die Gehrungen beider Enden einer Seite zusammen weiter als die Strecke lang ist (Krümmungsradius
// unter hw, ein Schwalbenschwanz der inneren Kante), liefen die Sechsecke gegeneinander und höben sich auf (ein Loch
// wie F-783). An einem solchen Gelenk enden beide Nachbarn innen senkrecht wie außen; sie überlappen dann auf der
// Innenseite und addieren dort die Kantenglättung wie früher, aber nur an gefalteten Gelenken.
type linienStrich struct {
	pts      []document.Punkt // ohne doppelte Punkte
	teile    [][]document.Punkt
	strecke  []int  // Teil je Strecke i (pts[i] → pts[(i+1) % n])
	gelenk   []int  // Fächer bzw. Kappe je Punkt, −1 ohne (gerades Gelenk)
	gefaltet []bool // je Punkt: Gelenk gefaltet, die Nachbarn enden innen senkrecht
	zu       bool   // geschlossen (mindestens drei Punkte)
}

// neuerLinienStrich zerlegt den Strich der halben Breite hw um den Linienzug roh (geschlossen: auch die Strecke vom
// letzten zum ersten Punkt). Punkte ohne Abstand zum vorigen fallen weg; bleibt einer, ist der Strich eine Scheibe.
func neuerLinienStrich(roh []document.Punkt, geschlossen bool, hw float64) linienStrich {
	var ls linienStrich
	if !(hw > 0) {
		return ls
	}
	for _, p := range roh {
		if len(ls.pts) == 0 || p.Minus(ls.pts[len(ls.pts)-1]).Laenge() > 1e-9 {
			ls.pts = append(ls.pts, p)
		}
	}
	for geschlossen && len(ls.pts) > 1 && ls.pts[0].Minus(ls.pts[len(ls.pts)-1]).Laenge() <= 1e-9 {
		ls.pts = ls.pts[:len(ls.pts)-1]
	}
	n := len(ls.pts)
	if n == 0 {
		return ls
	}
	ls.gelenk = make([]int, n)
	if n == 1 {
		p := ls.pts[0]
		ls.gelenk[0] = 0
		ls.teile = [][]document.Punkt{bogen(p, hw, document.Pt(hw, 0), document.Pt(hw, 0), 2*math.Pi)}
		return ls
	}
	ls.zu = geschlossen && n >= 3
	strecken := n - 1
	if ls.zu {
		strecken = n
	}
	richtung := make([]document.Punkt, strecken)
	normale := make([]document.Punkt, strecken) // links der Richtung
	laenge := make([]float64, strecken)
	for i := range strecken {
		d := ls.pts[(i+1)%n].Minus(ls.pts[i])
		laenge[i] = d.Laenge()
		richtung[i] = d.Mal(1 / laenge[i])
		normale[i] = document.Pt(-richtung[i].Y, richtung[i].X)
	}
	// vor und nach sind die Strecken an Punkt k (−1 ohne).
	vor := func(k int) int {
		if k > 0 {
			return k - 1
		}
		if ls.zu {
			return n - 1
		}
		return -1
	}
	nach := func(k int) int {
		if k < strecken {
			return k
		}
		return -1
	}
	// Je Gelenk: Innenseite (+1 links, −1 rechts, 0 gerade), Knickwinkel (mit Vorzeichen), Gehrung (links) und wie weit
	// sie entlang der Strecken reicht (hw·tan(|Knick|/2)).
	seite := make([]float64, n)
	knick := make([]float64, n)
	gehrung := make([]document.Punkt, n)
	reicht := make([]float64, n)
	gefaltet := make([]bool, n)
	ls.gefaltet = gefaltet
	for k := range n {
		a, b := vor(k), nach(k)
		if a < 0 || b < 0 {
			continue
		}
		cos := richtung[a].X*richtung[b].X + richtung[a].Y*richtung[b].Y
		sin := richtung[a].X*richtung[b].Y - richtung[a].Y*richtung[b].X
		knick[k] = math.Atan2(sin, cos)
		switch {
		case sin > 0:
			seite[k] = 1
		case sin < 0:
			seite[k] = -1
		case cos < 0:
			seite[k] = 1 // Kehrtwende: eine Seite wählen, das Gelenk ist gefaltet
		}
		if 1+cos < 1e-9 {
			gefaltet[k] = true
			continue
		}
		gehrung[k] = normale[a].Plus(normale[b]).Mal(hw / (1 + cos))
		reicht[k] = hw * math.Abs(sin) / (1 + cos)
	}
	// Faltung: Reichen die inneren Gehrungen einer Seite zusammen über die Strecke, enden ihre Gelenke senkrecht. Das
	// verkürzt nur, die Schleife endet.
	for geaendert := true; geaendert; {
		geaendert = false
		for i := range strecken {
			for _, s := range []float64{1, -1} {
				summe := 0.0
				var beteiligt []int
				for _, k := range []int{i, (i + 1) % n} {
					if !gefaltet[k] && seite[k] == s && reicht[k] > 0 {
						summe += reicht[k]
						beteiligt = append(beteiligt, k)
					}
				}
				if summe > laenge[i] {
					for _, k := range beteiligt {
						gefaltet[k] = true
					}
					geaendert = true
				}
			}
		}
	}
	// ende ist die Lage des Endes der Strecke i an Punkt k auf Seite s, relativ zum Punkt.
	ende := func(k, i int, s float64) document.Punkt {
		if vor(k) >= 0 && nach(k) >= 0 && !gefaltet[k] && seite[k] == s {
			return gehrung[k].Mal(s)
		}
		return normale[i].Mal(s * hw)
	}
	ls.strecke = make([]int, strecken)
	for i := range strecken {
		j := (i + 1) % n
		a, b := ls.pts[i], ls.pts[j]
		ls.strecke[i] = len(ls.teile)
		ls.teile = append(ls.teile, []document.Punkt{a.Plus(ende(i, i, 1)), b.Plus(ende(j, i, 1)), b,
			b.Plus(ende(j, i, -1)), a.Plus(ende(i, i, -1)), a})
	}
	for k := range n {
		ls.gelenk[k] = -1
		p := ls.pts[k]
		a, b := vor(k), nach(k)
		switch {
		case a < 0: // Anfang: halbe Scheibe von links über rückwärts nach rechts
			ls.gelenk[k] = len(ls.teile)
			ls.teile = append(ls.teile, bogen(p, hw, normale[b].Mal(hw), normale[b].Mal(-hw), math.Pi))
		case b < 0: // Ende: von rechts über vorwärts nach links
			ls.gelenk[k] = len(ls.teile)
			ls.teile = append(ls.teile, bogen(p, hw, normale[a].Mal(-hw), normale[a].Mal(hw), math.Pi))
		case seite[k] != 0: // Fächer auf der Außenseite, von der Normalen der vorigen zu der der nächsten Strecke
			o := -seite[k]
			f := bogen(p, hw, normale[a].Mal(o*hw), normale[b].Mal(o*hw), knick[k])
			ls.gelenk[k] = len(ls.teile)
			ls.teile = append(ls.teile, append([]document.Punkt{p}, f...))
		}
	}
	return ls
}

// bogen sind die Ecken des Bogens um m vom Radius r, von m+von um den Winkel w (mit Vorzeichen, Drehsinn wie von der
// x- zur y-Achse) bis m+bis; Anfang und Ende sind genau m+von und m+bis, damit Nachbarteile dieselben Punkte teilen.
// Ein Vollkreis (|w| = 2π) endet vor seinem Anfang.
func bogen(m document.Punkt, r float64, von, bis document.Punkt, w float64) []document.Punkt {
	schritte := max(1, int(math.Ceil(math.Abs(w)/bogenSchritt(r))))
	voll := math.Abs(w) >= 2*math.Pi
	pts := make([]document.Punkt, 0, schritte+1)
	pts = append(pts, m.Plus(von))
	a0 := math.Atan2(von.Y, von.X)
	for j := 1; j < schritte; j++ {
		a := a0 + w*float64(j)/float64(schritte)
		pts = append(pts, document.Pt(m.X+r*math.Cos(a), m.Y+r*math.Sin(a)))
	}
	if !voll {
		pts = append(pts, m.Plus(bis))
	}
	return pts
}
