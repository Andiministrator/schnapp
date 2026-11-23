// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"math"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// zeichneHilfslinien zeichnet die Linien, an denen der laufende Zug gerade einrastet (ADR-0021 Nachtrag 12): dünn, in
// der Akzentfarbe, über die ganze Breite oder Höhe des sichtbaren Bilds, nur während des Ziehens.
func (e *editor) zeichneHilfslinien(gtx layout.Context) {
	for _, r := range hilfslinienFlaechen(e.s, max(1, gtx.Dp(1))) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(r).Op())
	}
}

// hilfslinienFlaechen liefert die Flächen der Hilfslinien des laufenden Zugs in Fensterpixeln der Leinwand, w Pixel
// dick: Die Lage (Original) geht über die Ansicht (Zuschnitt, Stauchen) und die Abbildung (Zoom, Versatz) ins Fenster,
// die Länge reicht über das sichtbare Bild.
func hilfslinienFlaechen(s *canvas.Sitzung, w int) []image.Rectangle {
	linien := s.Hilfslinien()
	if len(linien) == 0 {
		return nil
	}
	g := s.D.Groesse()
	a := s.Abb.ZuFenster(document.Pt(0, 0))
	b := s.Abb.ZuFenster(document.Pt(float64(g.X), float64(g.Y)))
	var res []image.Rectangle
	for _, l := range linien {
		if l.Waagrecht {
			y := int(math.Round(s.Abb.ZuFenster(s.D.ZuAnsicht(document.Pt(0, l.Lage))).Y))
			res = append(res, image.Rect(int(math.Round(a.X)), y, int(math.Round(b.X)), y+w))
		} else {
			x := int(math.Round(s.Abb.ZuFenster(s.D.ZuAnsicht(document.Pt(l.Lage, 0))).X))
			res = append(res, image.Rect(x, int(math.Round(a.Y)), x+w, int(math.Round(b.Y))))
		}
	}
	return res
}
