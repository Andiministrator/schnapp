// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// spitzenLeiste ist der Schalter „Spitze“ der rechten Leiste (ADR-0021 Nachtrag 13 Punkt 6): Bei einem gewählten Text
// macht er ihn zur Sprechblase (Standard-Spitze) oder nimmt die Spitze weg, je ein Rückgängig-Schritt.
type spitzenLeiste struct {
	knopf  widget.Clickable
	rahmen image.Rectangle // Lage des Schalters beim letzten Zeichnen (Inhaltskoordinaten der Leiste), leer: nicht gezeigt
}

// spitzeAuswerten verarbeitet einen Klick auf den Schalter; während des Speicherns ruht das Bearbeiten.
func (e *editor) spitzeAuswerten(gtx layout.Context) {
	if e.spitze.knopf.Clicked(gtx) && e.seiteAn() {
		e.s.SchalteSpitze()
	}
}

// zeichneSpitze zeichnet die Zeile „[ ] Spitze“ bei y (Kästchen mit Wort) und liefert das neue y.
func (e *editor) zeichneSpitze(gtx layout.Context, x, y, spalte int) int {
	k := &e.spitze
	hoch := gtx.Dp(26)
	// Kästchen mit Wort wie „Shadow“ (ADR-0031 Nachtrag 13 Punkt 2, Karte #369): Kästchen und Wort sind eine Klickfläche.
	q := gtx.Dp(18)
	kr := image.Rect(x, y+(hoch-q)/2, x+q, y+(hoch-q)/2+q)
	zeichneKaestchen(gtx, kr, e.s.SpitzeAn())
	e.seiteText(gtx, i18n.T("Tail"), 13, farbeText, image.Pt(kr.Max.X+gtx.Dp(8), y+gtx.Dp(4)), spalte, 1)
	r := image.Rect(x, y, x+spalte+gtx.Dp(40), y+hoch)
	k.rahmen = r
	seiteKnopf(gtx, &k.knopf, r)
	return y + hoch + gtx.Dp(10)
}
