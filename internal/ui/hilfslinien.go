// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"math"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
)

// zeichneHilfslinien zeichnet die Linien, an denen der laufende Zug gerade einrastet (ADR-0021 Nachtrag 12): dünn, in
// der Akzentfarbe, über die ganze Breite oder Höhe des sichtbaren Bilds, nur während des Ziehens.
func (e *editor) zeichneHilfslinien(gtx layout.Context) {
	linien := e.s.Hilfslinien()
	if len(linien) == 0 {
		return
	}
	g := e.s.D.Groesse()
	a := e.s.Abb.ZuFenster(document.Pt(0, 0))
	b := e.s.Abb.ZuFenster(document.Pt(float64(g.X), float64(g.Y)))
	w := max(1, gtx.Dp(1))
	for _, l := range linien {
		var r image.Rectangle
		if l.Waagrecht {
			y := int(math.Round(e.s.Abb.ZuFenster(e.s.D.ZuAnsicht(document.Pt(0, l.Lage))).Y))
			r = image.Rect(int(math.Round(a.X)), y, int(math.Round(b.X)), y+w)
		} else {
			x := int(math.Round(e.s.Abb.ZuFenster(e.s.D.ZuAnsicht(document.Pt(l.Lage, 0))).X))
			r = image.Rect(x, int(math.Round(a.Y)), x+w, int(math.Round(b.Y)))
		}
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(r).Op())
	}
}
