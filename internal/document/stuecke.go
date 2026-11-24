// SPDX-License-Identifier: GPL-3.0-or-later

package document

// Text an der Naht (F-739, ADR-0023 Nachtrag 13): Die Ansicht ist das Original, aus dem Streifen herausgeschnitten und
// ein Zuschnitt abgeschnitten sind; zwischen zwei Nähten ist sie also eine reine Verschiebung. Rechtecke, Linien und
// Schilder bilden sich über die Ecken richtig ab, Glyphen aber nicht: Eine Textzeile stand starr ab ihrem
// abgebildeten Anfang, und ein Text, der nur knapp aus einem Streifen ragte, stand ganz hinter der Naht. Stuecke ist die
// eine Rechnung, mit der Renderer, Editor und SVG-Export Text wie Bildinhalt behandeln: je sichtbarem Bereich einmal
// gezeichnet, auf den Bereich beschnitten und um dessen Versatz verschoben.

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
