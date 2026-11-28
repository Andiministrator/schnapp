// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// freihandLeiste ist das Kästchen „Freehand“ beim Werkzeug Textmarker (ADR-0049 Punkt 5): an, zieht der Textmarker
// einen freien Strich statt einer Strecke. Es gilt für folgende Textmarker (Stil des Werkzeugs).
type freihandLeiste struct {
	knopf  widget.Clickable
	rahmen image.Rectangle // Lage beim letzten Zeichnen (Inhaltskoordinaten der Leiste), leer: nicht gezeigt
}

// freihandAuswerten verarbeitet einen Klick auf das Kästchen; während des Speicherns ruht das Bearbeiten.
func (e *editor) freihandAuswerten(gtx layout.Context) {
	if e.frei.knopf.Clicked(gtx) && e.seiteAn() {
		e.s.SchalteTextmarkerFrei()
	}
}

// zeichneFreihand zeichnet die Zeile „[ ] Freehand“ bei y und liefert das neue y.
func (e *editor) zeichneFreihand(gtx layout.Context, x, y, spalte int) int {
	k := &e.frei
	hoch := gtx.Dp(26)
	q := gtx.Dp(18)
	kr := image.Rect(x, y+(hoch-q)/2, x+q, y+(hoch-q)/2+q)
	zeichneKaestchen(gtx, kr, e.s.TextmarkerFrei())
	e.seiteText(gtx, i18n.T("Freehand"), 13, farbeText, image.Pt(kr.Max.X+gtx.Dp(8), y+gtx.Dp(4)), spalte, 1)
	r := image.Rect(x, y, x+spalte+gtx.Dp(40), y+hoch)
	k.rahmen = r
	seiteKnopf(gtx, &k.knopf, r)
	return y + hoch + gtx.Dp(10)
}
