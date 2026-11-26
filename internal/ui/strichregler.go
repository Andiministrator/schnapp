// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Regler der Strichstärke in der rechten Leiste (Karte #352, ADR-0021 Nachtrag 22): unter − / + eine Schiene über die
// volle Breite, logarithmisch von 1 bis 32 px (canvas.BreiteAusRegler); Ziehen ist ein Rückgängig-Schritt
// (canvas.Sitzung.BeginneBreiteZug). − / + bleiben für die Tastatur und genaue Stufen.

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// zeichneStrichRegler zeichnet den Regler ab (x, y) über breite und meldet ihn an; liefert das neue y.
func (e *editor) zeichneStrichRegler(gtx layout.Context, x, y, breite int) int {
	k := &e.seite
	hoch, schiene := gtx.Dp(16), gtx.Dp(4)
	k.strRegler = image.Rect(x, y, x+breite, y+hoch)
	mitte := y + hoch/2
	px := x + int(math.Round(canvas.ReglerLage(e.s.Breite())*float64(breite)))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(x, mitte-schiene/2, x+breite, mitte+schiene/2)).Op())
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(image.Rect(x, mitte-schiene/2, px, mitte+schiene/2)).Op())
	b := gtx.Dp(3)
	marke(gtx, image.Rect(max(x, px-b), y+gtx.Dp(1), min(x+breite, px+b), y+hoch-gtx.Dp(1)))
	reglerAnmelden(gtx, &k.strRegler, k.strRegler)
	return y + hoch
}

// strichReglerAuswerten verarbeitet Drücken, Ziehen, Loslassen und Abbruch am Regler.
func (e *editor) strichReglerAuswerten(gtx layout.Context) {
	k := &e.seite
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &k.strRegler,
			Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			return
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch pe.Kind {
		case pointer.Press:
			if pe.Buttons != pointer.ButtonPrimary || k.strZieht || !e.seiteAn() {
				continue
			}
			if e.s.BeginneBreiteZug() {
				k.strZieht = true
				e.strichZiehen(pe.Position)
			}
		case pointer.Drag, pointer.Release:
			if !k.strZieht {
				continue
			}
			e.strichZiehen(pe.Position)
			if pe.Kind == pointer.Release {
				e.s.BeendeBreiteZug()
				k.strZieht = false
			}
		case pointer.Cancel:
			if k.strZieht {
				e.s.BrecheBreiteZugAb()
				k.strZieht = false
			}
		}
	}
}

// strichZiehen setzt die Stärke aus der Lage p (relativ zum Regler).
func (e *editor) strichZiehen(p f32.Point) {
	r := e.seite.strRegler
	if r.Dx() <= 0 {
		return
	}
	e.s.BreiteZugSetze(canvas.BreiteAusRegler(float64(p.X) / float64(r.Dx())))
}
