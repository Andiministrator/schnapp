// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"slices"
)

// KastenStrich sind die konvexen, überlappungsfreien und gleich herum laufenden Teile des Strichs eines Kastens mit
// Eckradius (ADR-0023 Nachträge 19 und 21). Ausnahme, bewusst hingenommen (F-809): Ist eine gerade Seite kürzer als
// rund 0,065·hw, kreuzen sich die inneren Gehrungen ihrer Tangentenpunkte (höchstens ein Pixel in der Mitte eines
// Fast-Kreises). Renderer, Editor und die Vorschau der Ebenenliste füllen genau diese Teile:
// Weil sie sich nicht überlappen, addiert keine Füllung die Kantenglättung zweier Teile (F-788), und weil ihre inneren
// Ecken nie gegeneinander laufen, entsteht kein Loch (F-783, F-792, F-789).
type KastenStrich struct {
	// Umriss ist der Umriss des Kastens (RundesRechteck ohne doppelte Punkte); Strecke i läuft von Umriss[i] nach
	// Umriss[(i+1) % n].
	Umriss []Punkt
	// Teile sind die konvexen Teile, TeilVon nennt den Teil je Strecke, Grenzen umschließt jeden Teil.
	Teile   [][]Punkt
	TeilVon []int
	Grenzen []Rechteck
	// Umlauf ist das Vorzeichen der Fläche aller Teile (+1 oder −1).
	Umlauf float64
	// Rand ist der größte Abstand eines Punkts eines Teils von der nächsten seiner Strecken, plus ein Pixel: Wer die
	// Bereiche sucht, die ein Teil berührt (StreckenStuecke), sucht mit diesem Rand.
	Rand float64
}

// mittelpunktFang (F-798, F-804): Endet die Gehrung im Bogen einer Rundung weniger als so viele Pixel vor ihrem
// Mittelpunkt, liegen alle inneren Ecken im Bogen auf einem Punkt, dem Fangpunkt. Sonst lägen sie auf einem Kreis von
// weniger als 0,25 px Radius um den Mittelpunkt, und dort liefen zwölf Teile mit Kanten von Bruchteilen eines Pixels
// zusammen; solche Splitter rastert llvmpipe je nach Mesa-Fassung verschieden (Strich 39,9 bei Radius 20: 26 Pixel
// Abweichung zwischen Editor und Renderer in der CI, lokal keine). Der Fangpunkt ist die wahre innere Ecke der mittleren
// Ecke des Bogens (auf der Diagonale, auf dem Rand des Lochs); die Fläche, die dazukommt, liegt im Loch zwischen den
// wahren inneren Ecken und den beiden Kanten zum Fangpunkt (gemessen gegen die analytische Deckung höchstens 0,012 mehr
// als ohne Fang, bei Pillen 0,033; beim Kreis bis 0,07, F-807). Kanten unter 0,25 px gibt es im Band erst unter Radius
// rund 8 (rund 0,033·Radius, F-808), die Teile daran sind breit. Bis F-804 sprangen auch die Tangentenpunkte auf den
// Mittelpunkt, und die innere Kante jeder geraden Seite rückte um Radius − hw (bis 0,25 px) ins Loch.
const mittelpunktFang = 0.25

// NeuerKastenStrich zerlegt den Strich der Breite 2·hw um das Rechteck r mit Eckradius radius. Ohne wirksame Rundung
// (WirksamerEckradius = 0) oder ohne Strich ist das Ergebnis leer; dann gilt EckigerRahmen.
//
// Je Strecke i→i+1 des Umrisses ein Viereck: außen i, außen i+1, innen i+1, innen i. Außen liegt jede Ecke um die
// Gehrung 2·hw·(n₁+n₂)/|n₁+n₂|² verschoben (hw/cos(Knick/2) weit; der Umriss knickt höchstens um 7,5°), innen ebenso
// nach innen, mit drei Begrenzungen:
//   - Reicht die Gehrung g einer Ecke bis zum Mittelpunkt ihrer Rundung oder darüber (g ≥ Radius), liegt die innere
//     Ecke auf dem Mittelpunkt. Sonst läge sie jenseits davon, die inneren Ecken der Rundung in umgekehrter Reihenfolge,
//     und jedes Viereck wäre eine Schleife (F-783). Das gilt schon ab hw ≥ Radius·cos(3,75°), nicht erst ab hw ≥ Radius
//     (F-792); zwischen beiden ist die wahre innere Kante ein Kreis vom Radius r − hw < 0,0022·r um den Mittelpunkt.
//   - Endet die Gehrung im Bogen weniger als mittelpunktFang vor dem Mittelpunkt, liegen die inneren Ecken im Bogen
//     alle auf dem Fangpunkt der Rundung (F-798, F-804; siehe mittelpunktFang). Die beiden Tangentenpunkte, an denen
//     eine gerade Seite anschließt, behalten ihre Gehrung: Die innere Kante der Seite bleibt genau hw vom Umriss.
//   - Die innere Ecke liegt höchstens auf dem Rand des Lochs, des um hw verkleinerten umschließenden Rechtecks (bei zu
//     kleinem Rechteck seine Mittellinie bzw. Mitte): genau die Punkte weiter als hw vom Umriss.
//
// Strecken, deren beide inneren Ecken auf derselben Stelle liegen, gehören zu einem Fächer (äußere Ecken, dann die
// Spitze) statt zu einzelnen Dreiecken: Liefen viele schmale Dreiecke in einem Punkt zusammen, verlöre Gio dort durch
// die gegeneinander aufgerechneten Kanten Deckung. Liegen alle inneren Ecken auf einer Stelle, ist der Strich der äußere
// Umriss.
func NeuerKastenStrich(r Rechteck, radius, hw float64) KastenStrich {
	rad := WirksamerEckradius(radius, r.Breite(), r.Hoehe())
	if rad <= 0 || !(hw > 0) {
		return KastenStrich{}
	}
	// Umriss wie RundesRechteck, mit dem Mittelpunkt der Rundung je Punkt; doppelte Punkte (Seite ohne geraden Teil)
	// fallen weg, eine Strecke ohne Länge hat keine Richtung.
	roh := RundesRechteck(r, radius)
	const je = 13 // Punkte je Viertelkreis (RundesRechteck: schritte + 1)
	mitten := []Punkt{Pt(r.Max.X-rad, r.Min.Y+rad), Pt(r.Max.X-rad, r.Max.Y-rad), Pt(r.Min.X+rad, r.Max.Y-rad),
		Pt(r.Min.X+rad, r.Min.Y+rad)}
	var pts, mitte []Punkt
	var ecke []int
	for i, p := range roh {
		if len(pts) == 0 || p.Minus(pts[len(pts)-1]).Laenge() > 1e-9 {
			pts = append(pts, p)
			mitte = append(mitte, mitten[i/je])
			ecke = append(ecke, i/je)
		}
	}
	for len(pts) > 1 && pts[0].Minus(pts[len(pts)-1]).Laenge() <= 1e-9 {
		pts, mitte, ecke = pts[:len(pts)-1], mitte[:len(mitte)-1], ecke[:len(ecke)-1]
	}
	n := len(pts)
	if n < 3 {
		return KastenStrich{}
	}
	normale := make([]Punkt, n)
	for i := range pts {
		d := pts[(i+1)%n].Minus(pts[i])
		normale[i] = Pt(-d.Y, d.X).Mal(1 / d.Laenge())
	}
	gehrung := make([]Punkt, n)
	g := Rechteck{Min: pts[0], Max: pts[0]}
	for k, p := range pts {
		g = g.Vereint(Rechteck{Min: p, Max: p})
		sum := normale[(k+n-1)%n].Plus(normale[k])
		q := sum.X*sum.X + sum.Y*sum.Y
		if q < 1e-6 { // Kehrtwende, kommt im Umriss nicht vor
			gehrung[k] = normale[k].Mal(hw)
			continue
		}
		gehrung[k] = sum.Mal(2 * hw / q)
	}
	loch := Rechteck{Min: g.Min.Plus(Pt(hw, hw)), Max: g.Max.Minus(Pt(hw, hw))}
	if loch.Min.X > loch.Max.X {
		loch.Min.X, loch.Max.X = (g.Min.X+g.Max.X)/2, (g.Min.X+g.Max.X)/2
	}
	if loch.Min.Y > loch.Max.Y {
		loch.Min.Y, loch.Max.Y = (g.Min.Y+g.Max.Y)/2, (g.Min.Y+g.Max.Y)/2
	}
	mittelpunkt := Pt((g.Min.X+g.Max.X)/2, (g.Min.Y+g.Max.Y)/2)
	vorz := 1.0 // die linke Normale zeigt je nach Umlaufsinn nach außen oder innen
	if pts[0].Plus(gehrung[0]).Minus(mittelpunkt).Laenge() < pts[0].Minus(gehrung[0]).Minus(mittelpunkt).Laenge() {
		vorz = -1
	}
	aussen := make([]Punkt, n)
	innen := make([]Punkt, n)
	klemme := func(i Punkt) Punkt {
		return Pt(min(max(i.X, loch.Min.X), loch.Max.X), min(max(i.Y, loch.Min.Y), loch.Max.Y))
	}
	// Fangpunkt je Rundung (F-798, F-804): die wahre innere Ecke der mittleren Ecke des Bogens, ρ = Radius − Gehrung
	// im Bogen vor dem Mittelpunkt auf der Diagonale nach außen. Liegt der Mittelpunkt der Nachbarrundung näher als
	// mittelpunktFang (Seite ohne geraden Teil, Pille, Kreis), rücken beide Fangpunkte mindestens so weit auseinander;
	// ist das Loch in einer Richtung schmaler als mittelpunktFang, fallen sie dort auf seine Mitte zusammen.
	gBogen := hw / math.Cos(math.Pi/48) // Gehrung im Bogen (Knick 7,5°); an den Tangentenpunkten hw/cos(1,875°)
	imBogen := hw * (1/math.Cos(math.Pi/96) + 1/math.Cos(math.Pi/48)) / 2
	fangen := gBogen >= rad-mittelpunktFang
	rho := max(rad-gBogen, 0)
	nah := func(c int) bool {
		return mitten[c].Minus(mitten[(c+1)%4]).Laenge() < mittelpunktFang ||
			mitten[c].Minus(mitten[(c+3)%4]).Laenge() < mittelpunktFang
	}
	s2 := 1 / math.Sqrt2
	diag := [4]Punkt{Pt(s2, -s2), Pt(s2, s2), Pt(-s2, s2), Pt(-s2, -s2)} // nach außen, Reihenfolge wie mitten
	var fang [4]Punkt
	for c := range 4 {
		r := rho
		if r > 0 && nah(c) {
			r = max(r, mittelpunktFang/math.Sqrt2)
		}
		fang[c] = klemme(mitten[c].Plus(diag[c].Mal(r)))
		if loch.Max.X-loch.Min.X < mittelpunktFang {
			fang[c].X = (loch.Min.X + loch.Max.X) / 2
		}
		if loch.Max.Y-loch.Min.Y < mittelpunktFang {
			fang[c].Y = (loch.Min.Y + loch.Max.Y) / 2
		}
	}
	for k, p := range pts {
		aussen[k] = p.Plus(gehrung[k].Mal(vorz))
		i := p.Minus(gehrung[k].Mal(vorz))
		switch {
		case fangen && gehrung[k].Laenge() > imBogen:
			i = fang[ecke[k]] // F-798, F-804: nur Ecken im Bogen, die Tangentenpunkte halten die Kante der Seite
		case gehrung[k].Laenge() >= rad:
			i = mitte[k] // F-792: nicht über den Mittelpunkt der Rundung hinaus
		}
		innen[k] = klemme(i)
	}
	faecher := func(i int) bool { return innen[i] == innen[(i+1)%n] }
	start := 0 // erste Strecke eines Teils: nicht Fortsetzung des Fächers der vorigen
	for start < n {
		vor := (start + n - 1) % n
		if !(faecher(start) && faecher(vor) && innen[start] == innen[vor]) {
			break
		}
		start++
	}
	s := KastenStrich{Umriss: pts, TeilVon: make([]int, n)}
	if start == n {
		s.Teile = append(s.Teile, slices.Clone(aussen))
	} else {
		for k := 0; k < n; {
			i := (start + k) % n
			teil := []Punkt{aussen[i]}
			s.TeilVon[i] = len(s.Teile)
			k++
			if faecher(i) {
				for k < n && faecher((start+k)%n) && innen[(start+k)%n] == innen[i] {
					teil = append(teil, aussen[(start+k)%n])
					s.TeilVon[(start+k)%n] = len(s.Teile)
					k++
				}
				teil = append(teil, aussen[(start+k)%n], innen[i])
			} else {
				j := (i + 1) % n
				teil = append(teil, aussen[j], innen[j], innen[i])
			}
			s.Teile = append(s.Teile, teil)
		}
	}
	for _, teil := range s.Teile {
		b := Rechteck{Min: teil[0], Max: teil[0]}
		for _, e := range teil[1:] {
			b = b.Vereint(Rechteck{Min: e, Max: e})
		}
		s.Grenzen = append(s.Grenzen, b)
		if f := VieleckFlaeche(teil); s.Umlauf == 0 && f != 0 {
			s.Umlauf = math.Copysign(1, f)
		}
	}
	// Rand: Jeder Teil ist die Vereinigung der Vierecke (außen i, außen i+1, innen i+1, innen i) seiner Strecken, und der
	// Abstand zu einer Strecke ist konvex: Die Ecken jedes Vierecks bestimmen ihn.
	for i := range pts {
		j := (i + 1) % n
		for _, q := range []Punkt{aussen[i], aussen[j], innen[i], innen[j]} {
			s.Rand = max(s.Rand, abstandZurStrecke(q, pts[i], pts[j]))
		}
	}
	s.Rand++
	return s
}

// abstandZurStrecke ist der Abstand von p zur Strecke a–b.
func abstandZurStrecke(p, a, b Punkt) float64 {
	d := b.Minus(a)
	l := d.X*d.X + d.Y*d.Y
	if l == 0 {
		return p.Minus(a).Laenge()
	}
	t := min(max(((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l, 0), 1)
	return p.Minus(a.Plus(d.Mal(t))).Laenge()
}

// VieleckFlaeche ist die vorzeichenbehaftete Fläche des Vielecks p (Schnürsenkelformel).
func VieleckFlaeche(p []Punkt) float64 {
	f := 0.0
	for i, a := range p {
		b := p[(i+1)%len(p)]
		f += a.X*b.Y - b.X*a.Y
	}
	return f / 2
}
