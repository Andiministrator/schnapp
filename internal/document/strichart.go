// SPDX-License-Identifier: GPL-3.0-or-later

package document

import "math"

// Strichart ist das Muster eines Strichs (ADR-0021 Nachtrag 25): voll (Nullwert), gestrichelt oder gepunktet. Sie gilt
// für den Schaft des Pfeils, den Strich des Kastens und Rahmen samt Linie der Lupe; Spitzen bleiben voll.
type Strichart uint8

const (
	StrichVoll Strichart = iota
	StrichGestrichelt
	StrichGepunktet
)

// Stricharten ist die Reihenfolge der Bedienung.
var Stricharten = []Strichart{StrichVoll, StrichGestrichelt, StrichGepunktet}

// strichartNamen sind die Namen in Datei, --annotate und MCP (englisch wie s:kante).
var strichartNamen = [...]string{"solid", "dashed", "dotted"}

// Name ist der Name der Strichart in Datei, --annotate und MCP; eine unbekannte Art heißt „solid“.
func (a Strichart) Name() string {
	if int(a) < len(strichartNamen) {
		return strichartNamen[a]
	}
	return strichartNamen[0]
}

// StrichartAus liefert die Strichart zum Namen s.
func StrichartAus(s string) (Strichart, bool) {
	for i, n := range strichartNamen {
		if n == s {
			return Strichart(i), true
		}
	}
	return StrichVoll, false
}

// StrichartNamen sind alle Namen in der Reihenfolge der Bedienung.
func StrichartNamen() []string { return strichartNamen[:] }

// StrichMasse sind Länge eines Strichs und Lücke (gestrichelt) bzw. der Abstand der Punkte von Mitte zu Mitte
// (gepunktet, strich = 0) zur Strichstärke w, vor dem Ausgleich (ADR-0021 Nachtrag 25 Punkt 2).
func StrichMasse(a Strichart, w float64) (strich, luecke float64) {
	switch a {
	case StrichGestrichelt:
		return 3*w + 2, 2*w + 2
	case StrichGepunktet:
		return 0, 2*w + 2
	}
	return 0, 0
}

// Strichmuster sind die Stücke eines gemusterten Strichs: Linienzüge der Striche (gestrichelt) bzw. die Mitten der
// Punkte (gepunktet, Scheiben vom Durchmesser der Strichstärke). Die Stücke überlappen sich nicht (außer bei enger
// Krümmung, ADR-0021 Nachtrag 25 Punkt 4).
type Strichmuster struct {
	Striche [][]Punkt
	Punkte  []Punkt
}

// musterStueck ist ein Stück [Von, Bis] eines Wegs nach Länge; bei Punkten ist Von = Bis.
type musterStueck struct{ von, bis float64 }

// musterAbschnitte teilt einen Weg der Länge s nach Strichart a und Strichstärke w (ADR-0021 Nachtrag 25 Punkt 3).
// Offen (ecken = false): ganzer Strich bzw. Punkt an beiden Enden. Zwischen zwei Ecken (ecken = true): halber Strich
// bzw. Punkt an beiden Enden. Die Zahl der Striche wird gerundet, alle Längen um denselben Faktor gestreckt.
func musterAbschnitte(s float64, a Strichart, w float64, ecken bool) []musterStueck {
	strich, luecke := StrichMasse(a, w)
	switch a {
	case StrichGestrichelt:
		if s <= strich {
			return []musterStueck{{0, s}}
		}
		p := strich + luecke
		if !ecken {
			n := max(1, math.Round((s+luecke)/p))
			k := s / (n*strich + (n-1)*luecke)
			out := make([]musterStueck, 0, int(n))
			for j := range int(n) {
				v := float64(j) * p * k
				out = append(out, musterStueck{v, min(v+strich*k, s)})
			}
			out[len(out)-1].bis = s
			return out
		}
		n := max(1, math.Round(s/p))
		k := s / (n * p)
		out := make([]musterStueck, 0, int(n)+1)
		for j := 0; j <= int(n); j++ {
			c := float64(j) * p * k
			out = append(out, musterStueck{max(0, c-strich*k/2), min(s, c+strich*k/2)})
		}
		out[0].von, out[len(out)-1].bis = 0, s
		return out
	case StrichGepunktet:
		if s < w { // die Punkte an beiden Enden lägen übereinander: einer in der Mitte
			return []musterStueck{{s / 2, s / 2}}
		}
		n := max(1, math.Round(s/luecke))
		out := make([]musterStueck, 0, int(n)+1)
		for j := 0; j <= int(n); j++ {
			c := s * float64(j) / n
			out = append(out, musterStueck{c, c})
		}
		return out
	}
	return []musterStueck{{0, s}}
}

// zugLaengen sind die Bogenlängen bis zu jedem Punkt von pts.
func zugLaengen(pts []Punkt) []float64 {
	l := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		l[i] = l[i-1] + pts[i].Minus(pts[i-1]).Laenge()
	}
	return l
}

// zugPunkt ist der Punkt des Linienzugs pts bei Bogenlänge s (cum aus zugLaengen).
func zugPunkt(pts []Punkt, cum []float64, s float64) Punkt {
	for i := 1; i < len(pts); i++ {
		if s <= cum[i] || i == len(pts)-1 {
			d := cum[i] - cum[i-1]
			if d <= 0 {
				return pts[i]
			}
			t := min(max((s-cum[i-1])/d, 0), 1)
			return pts[i-1].Plus(pts[i].Minus(pts[i-1]).Mal(t))
		}
	}
	return pts[0]
}

// teilzug ist der Linienzug pts zwischen den Bogenlängen von und bis, samt der Punkte dazwischen.
func teilzug(pts []Punkt, cum []float64, von, bis float64) []Punkt {
	out := []Punkt{zugPunkt(pts, cum, von)}
	for i := 1; i < len(pts)-1; i++ {
		if cum[i] > von && cum[i] < bis {
			out = append(out, pts[i])
		}
	}
	return append(out, zugPunkt(pts, cum, bis))
}

// StrichAbschnitte teilt den offenen Linienzug pts (Original) nach Strichart a und Strichstärke w: ganzer Strich bzw.
// Punkt an beiden Enden (ADR-0021 Nachtrag 25 Punkt 3). Bei StrichVoll ist es ein Strich über den ganzen Zug.
func StrichAbschnitte(pts []Punkt, a Strichart, w float64) Strichmuster {
	if len(pts) == 0 {
		return Strichmuster{}
	}
	cum := zugLaengen(pts)
	var m Strichmuster
	for _, ab := range musterAbschnitte(cum[len(cum)-1], a, w, false) {
		if a == StrichGepunktet {
			m.Punkte = append(m.Punkte, zugPunkt(pts, cum, ab.von))
		} else {
			m.Striche = append(m.Striche, teilzug(pts, cum, ab.von, ab.bis))
		}
	}
	return m
}

// rahmenMuster teilt einen geschlossenen Umriss in Abschnitte von Ecke zu Ecke (jeder ein offener Linienzug; das Ende
// eines Abschnitts ist der Anfang des nächsten) mit halbem Strich bzw. Punkt an jeder Ecke. Striche über eine Ecke werden
// ein Linienzug, jeder Eckpunkt ist ein Punkt.
func rahmenMuster(abschnitte [][]Punkt, a Strichart, w float64) Strichmuster {
	var m Strichmuster
	anEnde := false       // der letzte Strich reicht bis ans Ende seines Abschnitts
	ersterAnfang := false // der erste Strich beginnt am Anfang des ersten Abschnitts
	for ai, pts := range abschnitte {
		cum := zugLaengen(pts)
		s := cum[len(cum)-1]
		abs := musterAbschnitte(s, a, w, true)
		if a == StrichGepunktet {
			for _, ab := range abs {
				if ab.von < s { // der Punkt an der Ecke am Ende gehört dem nächsten Abschnitt
					m.Punkte = append(m.Punkte, zugPunkt(pts, cum, ab.von))
				}
			}
			continue
		}
		for j, ab := range abs {
			z := teilzug(pts, cum, ab.von, ab.bis)
			if j == 0 && ab.von == 0 && anEnde && len(m.Striche) > 0 {
				letzter := m.Striche[len(m.Striche)-1]
				m.Striche[len(m.Striche)-1] = append(letzter, z[1:]...)
			} else {
				if ai == 0 && j == 0 && ab.von == 0 {
					ersterAnfang = true
				}
				m.Striche = append(m.Striche, z)
			}
			anEnde = ab.bis == s
		}
	}
	if a != StrichGepunktet && anEnde && ersterAnfang && len(m.Striche) > 1 {
		letzter := m.Striche[len(m.Striche)-1]
		m.Striche[0] = append(letzter, m.Striche[0][1:]...)
		m.Striche = m.Striche[:len(m.Striche)-1]
	}
	return m
}

// RundRahmenAbschnitte teilt den Strich eines runden Kastens (Umriss document.RundesRechteck mit wirksamer Rundung) je
// Abschnitt von der Mitte einer Rundung zur Mitte der nächsten (ADR-0021 Nachtrag 25 Punkt 3).
func RundRahmenAbschnitte(r Rechteck, radius float64, a Strichart, w float64) Strichmuster {
	roh := RundesRechteck(r, radius)
	if len(roh) < 4*13 {
		return Strichmuster{Striche: [][]Punkt{append(roh, roh[0])}}
	}
	const je, mitte = 13, 6 // Punkte je Viertelkreis, Mitte des Bogens
	var abschnitte [][]Punkt
	for c := range 4 {
		var pts []Punkt
		for i := c*je + mitte; i <= (c+1)*je+mitte; i++ {
			p := roh[i%len(roh)]
			if len(pts) == 0 || p.Minus(pts[len(pts)-1]).Laenge() > 1e-9 {
				pts = append(pts, p)
			}
		}
		abschnitte = append(abschnitte, pts)
	}
	return rahmenMuster(abschnitte, a, w)
}

// EckigerRahmenMuster zerlegt den Strich eines eckigen Rahmens (Kasten ohne Rundung, Rahmen der Lupe) nach Strichart:
// gestrichelt in überlappungsfreie Rechtecke (die waagerechten Striche besitzen die Ecken wie bei EckigerRahmen),
// gepunktet in die Mitten der Punkte (Ecken eingeschlossen). StrichVoll und ein Rahmen ohne Innenraum liefern
// EckigerRahmen.
func EckigerRahmenMuster(r Rechteck, w float64, a Strichart) (balken []Rechteck, punkte []Punkt) {
	h := w / 2
	ia, ib := r.Min.Plus(Pt(h, h)), r.Max.Minus(Pt(h, h))
	if a == StrichVoll || ia.X >= ib.X || ia.Y >= ib.Y {
		return EckigerRahmen(r, w), nil
	}
	ecken := []Punkt{r.Min, Pt(r.Max.X, r.Min.Y), r.Max, Pt(r.Min.X, r.Max.Y)}
	if a == StrichGepunktet {
		var abschnitte [][]Punkt
		for c := range 4 {
			abschnitte = append(abschnitte, []Punkt{ecken[c], ecken[(c+1)%4]})
		}
		return nil, rahmenMuster(abschnitte, a, w).Punkte
	}
	bw, bh := r.Breite(), r.Hoehe()
	for _, y := range []float64{r.Min.Y, r.Max.Y} { // oben und unten, mit den Ecken
		for _, ab := range musterAbschnitte(bw, a, w, true) {
			x0, x1 := r.Min.X+ab.von, r.Min.X+ab.bis
			if ab.von == 0 {
				x0 = r.Min.X - h
			}
			if ab.bis == bw {
				x1 = r.Max.X + h
			}
			balken = append(balken, Rechteck{Min: Pt(x0, y-h), Max: Pt(x1, y+h)})
		}
	}
	for _, x := range []float64{r.Min.X, r.Max.X} { // links und rechts, zwischen den Ecken
		for _, ab := range musterAbschnitte(bh, a, w, true) {
			y0, y1 := max(r.Min.Y+ab.von, ia.Y), min(r.Min.Y+ab.bis, ib.Y)
			if y1 > y0 {
				balken = append(balken, Rechteck{Min: Pt(x-h, y0), Max: Pt(x+h, y1)})
			}
		}
	}
	return balken, nil
}
