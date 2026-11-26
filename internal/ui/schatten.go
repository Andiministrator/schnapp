// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Schatten auf der Leinwand (ADR-0021 Nachtrag 26 Punkt 4): dasselbe Bild wie im Renderer (render.Schattenbild), je
// Objekt gemerkt, solange Objekt, Stand des Dokuments und Pixel gleich bleiben, und unter dem Objekt gezeichnet —
// innerhalb der Deckkraft der Ebene, die zeichneObjekt davor setzt.

// schattenBild ist ein gemerktes Schattenbild.
type schattenBild struct {
	o           document.Objekt
	stand, bild uint64
	op          paint.ImageOp
	rahmen      image.Rectangle // Ansichtskoordinaten
	da          bool
}

// zeichneSchatten zeichnet den Schatten von o in Ansichtskoordinaten (zeichneObjekte).
func (e *editor) zeichneSchatten(gtx layout.Context, o document.Objekt) {
	if !o.Ebenenwerte().Schatten.Sichtbar() {
		return
	}
	if e.schatten == nil {
		e.schatten = map[int]schattenBild{}
	}
	stand, bild := e.s.D.Stand(), e.s.BildAenderung()
	b, ok := e.schatten[o.Kennung()]
	if !ok || b.stand != stand || b.bild != bild || !document.Gleich(b.o, o) {
		b = schattenBild{o: o, stand: stand, bild: bild}
		if img, err := render.Schattenbild(e.s.D, o); err == nil && img != nil {
			b.op, b.rahmen, b.da = paint.NewImageOp(img), img.Rect, true
		}
		e.schatten[o.Kennung()] = b
	}
	if !b.da {
		return
	}
	t := op.Offset(b.rahmen.Min).Push(gtx.Ops)
	b.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()
}

// schattenAufraeumen verwirft nach jedem Bild die Schattenbilder von Objekten, die es nicht mehr gibt.
func (e *editor) schattenAufraeumen() {
	for id := range e.schatten {
		if _, ok := e.s.D.Objekt(id); !ok {
			delete(e.schatten, id)
		}
	}
}
