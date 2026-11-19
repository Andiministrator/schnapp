// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/document"

// GriffArt bezeichnet einen Griff eines ausgewählten Objekts.
type GriffArt int

// Griffe: Pfeilenden und Bogen; beim Kasten Ecken und Kanten im Uhrzeigersinn ab links oben.
const (
	GriffAnfang GriffArt = iota + 1 // Pfeil: A
	GriffEnde                       // Pfeil: B (Spitze)
	GriffBogen                      // Pfeil: Kurvenpunkt bei t = 0,5
	GriffNW
	GriffN
	GriffNO
	GriffO
	GriffSO
	GriffS
	GriffSW
	GriffW
)

// Griff ist ein Griff mit seinem Ort im Original.
type Griff struct {
	Art GriffArt
	Ort document.Punkt
}

// Griffe liefert die Griffe eines Objekts; Text und Nummer haben keine (sie werden nur verschoben).
func Griffe(o document.Objekt) []Griff {
	switch v := o.(type) {
	case document.Pfeil:
		return []Griff{{GriffAnfang, v.A}, {GriffEnde, v.B}, {GriffBogen, v.Griff()}}
	case document.Kasten:
		r := v.Bereich
		mx, my := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
		return []Griff{
			{GriffNW, r.Min}, {GriffN, document.Pt(mx, r.Min.Y)}, {GriffNO, document.Pt(r.Max.X, r.Min.Y)},
			{GriffO, document.Pt(r.Max.X, my)}, {GriffSO, r.Max}, {GriffS, document.Pt(mx, r.Max.Y)},
			{GriffSW, document.Pt(r.Min.X, r.Max.Y)}, {GriffW, document.Pt(r.Min.X, my)},
		}
	}
	return nil
}

// GriffTreffer liefert den Griff von o, der p (Original) innerhalb von toleranz trifft; liegen mehrere gleich nah,
// gewinnt der zuerst genannte.
func GriffTreffer(o document.Objekt, p document.Punkt, toleranz float64) (GriffArt, bool) {
	var best GriffArt
	bestAbstand := toleranz
	for _, g := range Griffe(o) {
		if a := p.Minus(g.Ort).Laenge(); a <= bestAbstand && (best == 0 || a < bestAbstand) {
			best, bestAbstand = g.Art, a
		}
	}
	return best, best != 0
}

// ZieheGriff liefert o mit dem Griff art an p (Original). Beim Pfeil behält das Verschieben eines Endes die Biegung
// (Abstand des Bogengriffs zur Mitte der Sehne); beim Kasten bewegen Ecken zwei Seiten, Kanten eine, und der Bereich
// wird wieder geordnet, wenn eine Seite über die andere gezogen wurde.
// Die Oberfläche wendet ZieheGriff während eines Zugs immer auf das Objekt vom Zugbeginn an, nicht auf das Zwischen-
// ergebnis — so bleibt ein Griff derselbe, auch wenn der Kasten beim Überziehen umklappt.
func ZieheGriff(o document.Objekt, art GriffArt, p document.Punkt) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		switch art {
		case GriffBogen:
			return v.MitGriff(p)
		case GriffAnfang, GriffEnde:
			biegung := v.Griff().Minus(v.A.Plus(v.B).Mal(0.5))
			if art == GriffAnfang {
				v.A = p
			} else {
				v.B = p
			}
			return v.MitGriff(v.A.Plus(v.B).Mal(0.5).Plus(biegung))
		}
	case document.Kasten:
		r := v.Bereich
		switch art {
		case GriffNW:
			r.Min = p
		case GriffN:
			r.Min.Y = p.Y
		case GriffNO:
			r.Max.X, r.Min.Y = p.X, p.Y
		case GriffO:
			r.Max.X = p.X
		case GriffSO:
			r.Max = p
		case GriffS:
			r.Max.Y = p.Y
		case GriffSW:
			r.Min.X, r.Max.Y = p.X, p.Y
		case GriffW:
			r.Min.X = p.X
		default:
			return o
		}
		v.Bereich = document.R(r.Min, r.Max)
		return v
	}
	return o
}
