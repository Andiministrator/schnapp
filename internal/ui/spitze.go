// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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

// zeichneSpitze zeichnet die Zeile „Spitze [ ]“ bei y (Beschriftung links, Schalter rechts) und liefert das neue y.
func (e *editor) zeichneSpitze(gtx layout.Context, x, y, spalte int) int {
	k := &e.spitze
	hoch := gtx.Dp(26)
	e.seiteText(gtx, i18n.T("Tail"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	r := image.Rect(x+spalte, y, x+spalte+hoch, y+hoch)
	k.rahmen = r
	an := e.s.SpitzeAn()
	rnd := farbeLinie
	if k.knopf.Hovered() {
		rnd = farbeNebentext
	}
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	if an { // eckiges Kästchen, innen gefüllt (Corporate Design: eckig)
		paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r.Inset(gtx.Dp(6))).Op())
	}
	seiteKnopf(gtx, &k.knopf, r)
	return y + hoch + gtx.Dp(10)
}
