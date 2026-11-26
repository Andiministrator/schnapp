// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"slices"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Schatten auf der Leinwand (ADR-0021 Nachtrag 26 Punkt 4): dasselbe Bild wie im Renderer (render.Schattenbild), je
// Objekt gemerkt, solange Objekt und Ansicht (Dokument, Zuschnitt, Streifen, Pixel) gleich bleiben, und unter dem Objekt
// gezeichnet — innerhalb der Deckkraft der Ebene, die zeichneObjekt davor setzt. Der Stand des Dokuments gehört nicht
// zum Schlüssel: Sonst rechnete jede Änderung an irgendeinem Objekt alle Schatten neu (F-843).

// schattenSpeicher begrenzt die Pixel aller gemerkten Schattenbilder (F-843); darüber fängt der Speicher neu an.
const schattenSpeicher = 256 << 20

// schattenAnsicht ist, was außer dem Objekt das Schattenbild bestimmt.
type schattenAnsicht struct {
	d         *document.Dokument
	zuschnitt image.Rectangle
	streifen  []document.Streifen
	bild      uint64
}

func (a schattenAnsicht) gleich(b schattenAnsicht) bool {
	return a.d == b.d && a.zuschnitt == b.zuschnitt && a.bild == b.bild && slices.Equal(a.streifen, b.streifen)
}

// schattenBild ist ein gemerktes Schattenbild.
type schattenBild struct {
	o       document.Objekt
	ansicht schattenAnsicht
	op      paint.ImageOp
	rahmen  image.Rectangle // Ansichtskoordinaten
	bytes   int
	da      bool
}

// zeichneSchatten zeichnet den Schatten von o in Ansichtskoordinaten (zeichneObjekte).
func (e *editor) zeichneSchatten(gtx layout.Context, o document.Objekt) {
	if !o.Ebenenwerte().Schatten.Sichtbar() {
		return
	}
	b := e.schattenFuer(o)
	if !b.da {
		return
	}
	t := op.Offset(b.rahmen.Min).Push(gtx.Ops)
	b.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	t.Pop()
}

// schattenFuer liefert das gemerkte Schattenbild von o und rechnet es neu, wenn Objekt oder Ansicht sich geändert haben.
func (e *editor) schattenFuer(o document.Objekt) schattenBild {
	if e.schatten == nil {
		e.schatten = map[int]schattenBild{}
	}
	ansicht := schattenAnsicht{d: e.s.D, zuschnitt: e.s.D.Zuschnitt(), streifen: e.s.D.Streifen(), bild: e.s.BildAenderung()}
	b, ok := e.schatten[o.Kennung()]
	if ok && b.ansicht.gleich(ansicht) && document.Gleich(b.o, o) {
		return b
	}
	if ok {
		e.schattenBytes -= b.bytes
	}
	e.schattenNeu++
	b = schattenBild{o: o, ansicht: ansicht}
	if img, err := render.Schattenbild(e.s.D, o); err == nil && img != nil {
		b.op, b.rahmen, b.da, b.bytes = paint.NewImageOp(img), img.Rect, true, len(img.Pix)
	}
	if e.schattenBytes+b.bytes > schattenSpeicher { // Grenze: neu anfangen, das aktuelle Bild bleibt
		e.schatten, e.schattenBytes = map[int]schattenBild{}, 0
	}
	e.schatten[o.Kennung()] = b
	e.schattenBytes += b.bytes
	return b
}

// schattenAufraeumen verwirft nach jedem Bild die Schattenbilder von Objekten, die es nicht mehr gibt.
func (e *editor) schattenAufraeumen() {
	for id, b := range e.schatten {
		if _, ok := e.s.D.Objekt(id); !ok {
			e.schattenBytes -= b.bytes
			delete(e.schatten, id)
		}
	}
}
