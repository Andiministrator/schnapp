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

// schattenSpeicher begrenzt die Pixel aller gemerkten Schattenbilder (F-843). Darüber fallen die am längsten nicht
// gezeichneten weg, nie eines aus dem laufenden Bild (V-3): Brauchen die sichtbaren Schatten eines Bilds zusammen mehr,
// bleiben sie trotzdem gemerkt, sonst rechnete jedes Bild alle neu.
var schattenSpeicher = 256 << 20 // Variable nur für Tests

// schattenAnsicht ist, was außer dem Objekt das Schattenbild bestimmt.
// Bei einer Lupe gehören Stand und Kantenform dazu (V-2): Ihr Inhalt hängt an den Pixeln des Originals (auch nach
// Rückgängig einer Schwärzung) und am Spalt der Nähte; BildAenderung erfasst beides nicht.
type schattenAnsicht struct {
	d         *document.Dokument
	zuschnitt image.Rectangle
	streifen  []document.Streifen
	bild      uint64
	stand     uint64
	kante     document.Kante
}

func (a schattenAnsicht) gleich(b schattenAnsicht) bool {
	return a.d == b.d && a.zuschnitt == b.zuschnitt && a.bild == b.bild && a.stand == b.stand && a.kante == b.kante &&
		slices.Equal(a.streifen, b.streifen)
}

// schattenBild ist ein gemerktes Schattenbild.
type schattenBild struct {
	o       document.Objekt
	ansicht schattenAnsicht
	op      paint.ImageOp
	rahmen  image.Rectangle // Ansichtskoordinaten
	bytes   int
	runde   int // zuletzt gezeichnet in dieser Runde (editor.schattenRunde)
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
	if _, lupe := o.(document.Lupe); lupe {
		ansicht.stand, ansicht.kante = e.s.D.Stand(), e.s.D.Kante()
	}
	b, ok := e.schatten[o.Kennung()]
	if ok && b.ansicht.gleich(ansicht) && document.Gleich(b.o, o) {
		b.runde = e.schattenRunde
		e.schatten[o.Kennung()] = b
		return b
	}
	if ok {
		e.schattenBytes -= b.bytes
	}
	e.schattenNeu++
	delete(e.schatten, o.Kennung())
	b = schattenBild{o: o, ansicht: ansicht, runde: e.schattenRunde}
	if img, err := render.Schattenbild(e.s.D, o); err == nil && img != nil {
		b.op, b.rahmen, b.da, b.bytes = paint.NewImageOp(img), img.Rect, true, len(img.Pix)
	}
	e.schattenVerdraengen(b.bytes)
	e.schatten[o.Kennung()] = b
	e.schattenBytes += b.bytes
	return b
}

// schattenVerdraengen macht Platz für neu Bytes: die am längsten nicht gezeichneten Bilder fallen weg, keines aus der
// laufenden Runde (V-3).
func (e *editor) schattenVerdraengen(neu int) {
	if e.schattenBytes+neu <= schattenSpeicher {
		return
	}
	alt := make([]int, 0, len(e.schatten))
	for id, b := range e.schatten {
		if b.runde < e.schattenRunde {
			alt = append(alt, id)
		}
	}
	slices.SortFunc(alt, func(a, b int) int { return e.schatten[a].runde - e.schatten[b].runde })
	for _, id := range alt {
		if e.schattenBytes+neu <= schattenSpeicher {
			break
		}
		e.schattenBytes -= e.schatten[id].bytes
		delete(e.schatten, id)
	}
}

// schattenAufraeumen verwirft nach jedem Bild die Schattenbilder von Objekten, die es nicht mehr gibt, und beginnt die
// nächste Runde.
func (e *editor) schattenAufraeumen() {
	e.schattenRunde++
	for id, b := range e.schatten {
		if _, ok := e.s.D.Objekt(id); !ok {
			e.schattenBytes -= b.bytes
			delete(e.schatten, id)
		}
	}
}
