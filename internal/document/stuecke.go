// SPDX-License-Identifier: GPL-3.0-or-later

package document

import "sort"

// Text an der Naht (F-739, ADR-0023 Nachtrag 13): Die Ansicht ist das Original, aus dem Streifen herausgeschnitten und
// ein Zuschnitt abgeschnitten sind; zwischen zwei Nähten ist sie also eine reine Verschiebung. Rechtecke, Linien und
// Schilder bilden sich über die Ecken richtig ab, Glyphen aber nicht: Eine Textzeile stand starr ab ihrem
// abgebildeten Anfang, und ein Text, der nur knapp aus einem Streifen ragte, stand ganz hinter der Naht. Mit Stuecke
// behandeln Renderer und Editor Text wie Bildinhalt: je sichtbarem Bereich einmal gezeichnet, auf den Bereich
// beschnitten und um dessen Versatz verschoben — wie der SVG-Export mit seinen Kacheln (eigene Rechnung `intervalle`).

// Stueck ist ein sichtbarer Bereich der Ansicht: Quelle ist ein Rechteck im Original zwischen Zuschnittkanten und
// Streifen (ganzzahlige Kanten), in dem Ansicht = Original + Versatz gilt.
type Stueck struct {
	Quelle  Rechteck
	Versatz Punkt
}

// Ziel ist die Lage des Stücks in der Ansicht.
func (s Stueck) Ziel() Rechteck {
	return Rechteck{Min: s.Quelle.Min.Plus(s.Versatz), Max: s.Quelle.Max.Plus(s.Versatz)}
}

// abschnitt ist ein sichtbarer Bereich [a, b) einer Achse mit seinem Versatz zur Ansicht.
type abschnitt struct {
	a, b    int
	versatz float64
}

// abschnitte liefert die sichtbaren Bereiche der Achse, die [lo, hi] mit positiver Länge schneiden (Regel wie
// sichtbar), in aufsteigender Reihenfolge.
func (m abbildung) abschnitte(lo, hi float64) []abschnitt {
	var as []abschnitt
	nimm := func(a, b, k int) {
		if a < b && lo < float64(b) && hi > float64(a) {
			as = append(as, abschnitt{a, b, -float64(m.von + m.kum[k])})
		}
	}
	start := m.von
	for i, w := range m.weg {
		nimm(start, w[0], i)
		start = w[1]
	}
	if len(m.kum) > 0 {
		nimm(start, m.bis, len(m.weg))
	}
	return as
}

// Stuecke liefert die sichtbaren Bereiche, die r (Original) mit positiver Fläche schneiden; die Quelle jedes Stücks ist
// der ganze Bereich, nicht nur sein Schnitt mit r (Kantenglättung und Überhänge einer Glyphe bleiben so erhalten, nur an
// einer Naht oder Zuschnittkante wird beschnitten). Ohne Stück ist r nirgends sichtbar (wie Sichtbar).
func (d *Dokument) Stuecke(r Rechteck) []Stueck {
	xs := d.abbX.abschnitte(r.Min.X, r.Max.X)
	ys := d.abbY.abschnitte(r.Min.Y, r.Max.Y)
	st := make([]Stueck, 0, len(xs)*len(ys))
	for _, y := range ys {
		for _, x := range xs {
			st = append(st, Stueck{
				Quelle:  Rechteck{Min: Pt(float64(x.a), float64(y.a)), Max: Pt(float64(x.b), float64(y.b))},
				Versatz: Pt(x.versatz, y.versatz),
			})
		}
	}
	return st
}

// TextStuecke sind die Stücke, in denen die Glyphen von t gezeichnet werden: die sichtbaren Bereiche, die Tinte oder
// Schild von t berühren (Reichweite ohne Spitze, wie TextSichtbar). Bei genau einem Stück steht der Text wie bisher,
// verschoben um den Versatz dieses Bereichs; bei mehreren zeichnet jeder Ausgang die Zeilen je Stück, beschnitten auf
// dessen Ziel. Ein Text mit nur sichtbarer Spitze hat keine Stücke (NurSpitzeSichtbar).
func (d *Dokument) TextStuecke(t Text) []Stueck {
	return d.Stuecke(textReichweite(t, t.Grenzen()))
}

// Abschnitt ist ein sichtbarer Bereich [A, B) einer Achse des Originals; dort gilt Ansicht = Original + Versatz
// (ganzzahlig, die Nähte und der Zuschnitt liegen auf ganzen Pixeln).
type Abschnitt struct {
	A, B, Versatz int
}

// Abschnitte liefert die sichtbaren Bereiche je Achse, die r mit positiver Länge schneiden, aufsteigend; Stuecke(r) ist
// ihr Kreuzprodukt. Renderer und Editor zeichnen Text damit zeilenweise statt je Stück (F-747, F-750, ADR-0023
// Nachtrag 16): Eine Zeile berührt nur die waagerechten Abschnitte ihrer Höhe, und jede Glyphe wird einmal gesetzt und
// je Bereich nur verschoben.
func (d *Dokument) Abschnitte(r Rechteck) (xs, ys []Abschnitt) {
	umwandeln := func(as []abschnitt) []Abschnitt {
		out := make([]Abschnitt, len(as))
		for i, a := range as {
			out[i] = Abschnitt{A: a.a, B: a.b, Versatz: int(a.versatz)}
		}
		return out
	}
	return umwandeln(d.abbX.abschnitte(r.Min.X, r.Max.X)), umwandeln(d.abbY.abschnitte(r.Min.Y, r.Max.Y))
}

// TextAbschnitte sind die Abschnitte zu TextStuecke (dieselbe Reichweite).
func (d *Dokument) TextAbschnitte(t Text) (xs, ys []Abschnitt) {
	return d.Abschnitte(textReichweite(t, t.Grenzen()))
}

// Beschneide schneidet die Abschnitte einer Achse auf [lo, hi) der Ansicht zu; leere fallen weg, die Reihenfolge bleibt
// aufsteigend und überlappungsfrei.
func Beschneide(as []Abschnitt, lo, hi int) []Abschnitt {
	out := make([]Abschnitt, 0, len(as))
	for _, a := range as {
		b := Abschnitt{A: max(a.A, lo-a.Versatz), B: min(a.B, hi-a.Versatz), Versatz: a.Versatz}
		if b.A < b.B {
			out = append(out, b)
		}
	}
	return out
}

// AbschnitteIn liefert die Abschnitte, die [lo, hi] im Original berühren (geschlossene Grenzen), per Binärsuche; as
// ist aufsteigend und überlappungsfrei.
func AbschnitteIn(as []Abschnitt, lo, hi float64) []Abschnitt {
	i := sort.Search(len(as), func(k int) bool { return float64(as[k].B) >= lo })
	j := i
	for j < len(as) && float64(as[j].A) <= hi {
		j++
	}
	return as[i:j]
}

// StreckenStueck ist ein sichtbarer Bereich mit den Strecken einer Linie, die ihn berühren (Indizes i der Strecke
// pts[i] → pts[i+1], bei geschlossener Linie auch die letzte zurück zum Anfang).
type StreckenStueck struct {
	Stueck
	Strecken []int
}

// StreckenStuecke ordnet die Strecken der Linie pts (Original) den sichtbaren Bereichen zu, die sie mit dem Rand rand
// (halbe Strichbreite und Glättung) berühren, in der Reihenfolge von Stuecke. So zeichnen Renderer und Editor einen
// Strich über einer Naht abgeschnitten wie der SVG-Export, und jeder Bereich baut nur die Strecken in seiner Nähe (F-748,
// ADR-0023 Nachtrag 17): Ein Strich je Bereich ganz hätte über vielen Nähten jedes Mal die ganze Linie gebaut (F-136).
// Aufwand: Strecken × berührte Bereiche.
func (d *Dokument) StreckenStuecke(pts []Punkt, geschlossen bool, rand float64) []StreckenStueck {
	if len(pts) == 0 {
		return nil
	}
	g := Rechteck{Min: pts[0], Max: pts[0]}
	for _, p := range pts[1:] {
		g = g.Vereint(Rechteck{Min: p, Max: p})
	}
	g = Rechteck{Min: g.Min.Minus(Pt(rand, rand)), Max: g.Max.Plus(Pt(rand, rand))}
	xs, ys := d.Abschnitte(g)
	if len(xs) == 0 || len(ys) == 0 {
		return nil
	}
	index := func(as []Abschnitt, lo, hi float64) (int, int) {
		i := sort.Search(len(as), func(k int) bool { return float64(as[k].B) > lo })
		j := i
		for j < len(as) && float64(as[j].A) < hi {
			j++
		}
		return i, j
	}
	n := len(pts)
	strecken := n - 1
	if geschlossen && n > 1 {
		strecken = n
	}
	if n == 1 {
		strecken = 1 // ein Punkt: eine Strecke der Länge 0 (Kreis)
	}
	faecher := map[[2]int][]int{}
	for i := 0; i < strecken; i++ {
		a, b := pts[i], pts[(i+1)%n]
		lx, hx := min(a.X, b.X)-rand, max(a.X, b.X)+rand
		ly, hy := min(a.Y, b.Y)-rand, max(a.Y, b.Y)+rand
		x0, x1 := index(xs, lx, hx)
		y0, y1 := index(ys, ly, hy)
		for yi := y0; yi < y1; yi++ {
			for xi := x0; xi < x1; xi++ {
				k := [2]int{yi, xi}
				faecher[k] = append(faecher[k], i)
			}
		}
	}
	// Nur die berührten Bereiche, in der Reihenfolge von Stuecke (Zeile, dann Spalte); nicht alle Bereiche ablaufen.
	schluessel := make([][2]int, 0, len(faecher))
	for k := range faecher {
		schluessel = append(schluessel, k)
	}
	sort.Slice(schluessel, func(i, j int) bool {
		if schluessel[i][0] != schluessel[j][0] {
			return schluessel[i][0] < schluessel[j][0]
		}
		return schluessel[i][1] < schluessel[j][1]
	})
	out := make([]StreckenStueck, 0, len(schluessel))
	for _, k := range schluessel {
		y, x, st := ys[k[0]], xs[k[1]], faecher[k]
		out = append(out, StreckenStueck{
			Stueck: Stueck{
				Quelle:  Rechteck{Min: Pt(float64(x.A), float64(y.A)), Max: Pt(float64(x.B), float64(y.B))},
				Versatz: Pt(float64(x.Versatz), float64(y.Versatz)),
			},
			Strecken: st,
		})
	}
	return out
}
