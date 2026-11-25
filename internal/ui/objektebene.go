// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// bereichGrenze ist die Zahl abgespielter Bereiche je Bild, über der der Editor die Objekte nicht mehr selbst zeichnet,
// sondern die Objektebene des Renderers zeigt (ADR-0023 Nachtrag 20, F-787). Ein Bereich ist ein Abspielen einer
// Aufzeichnung in einem sichtbaren Bereich zwischen Nähten (jeStueckIn, Lauf einer Textzeile) oder ein Bereich des
// Strichs eines runden Kastens (strichJeStueck). Gio braucht mit llvmpipe je Bereich rund 1,3 µs (Strich eines runden
// Kastens: 2000 über 4096 Kacheln, 488 000 Bereiche, 0,63 s je Bild) bis 3,3 µs (Füllung eines runden Kastens: 120
// gefüllte über 4096 Kacheln, knapp unter dem Budget von render.Pruefe, 506 000 Bereiche, 1,66 s je Bild); an der Grenze
// also höchstens rund 0,7 s je Bild. Im Alltag sind es wenige Hundert bis Tausend Bereiche.
const bereichGrenze = 200_000

// objektebeneRendern ist der Renderaufruf von baue; ein Test ersetzt ihn, weil sich ein Fehler des Renderers sonst nicht
// erzeugen lässt (F-805).
var objektebeneRendern = render.Objektebene

// objektEbene ist die zwischengespeicherte Objektebene des Renderers (render.Objektebene) samt dem Stand, für den sie
// gilt: Dokument, Dokumentstand, Pixelstand, Ansichtsgröße und die Objekte selbst (während eines Zugs ändern sie sich
// ohne Rückgängig-Schritt). Das Dokument zählt mit, weil zwei frisch geöffnete Dokumente denselben Stand und Pixelstand
// tragen (Wechsel der Aufnahme, F-799).
type objektEbene struct {
	an       bool
	dok      *document.Dokument
	bild     paint.ImageOp
	groesse  image.Point
	stand    uint64
	pixel    uint64
	objekte  []document.Objekt
	gebaut   int // Zahl der Aufbauten (Tests)
	bereiche int // gezählte Bereiche des letzten eigenen Zeichnens (Tests, Grenze)
}

// passt meldet, dass die Ebene den Stand von e zeigt.
func (o *objektEbene) passt(e *editor) bool {
	d := e.s.D
	if !o.an || d != o.dok || d.Stand() != o.stand || e.s.BildAenderung() != o.pixel || d.Groesse() != o.groesse {
		return false
	}
	obj := d.Objekte()
	if len(obj) != len(o.objekte) {
		return false
	}
	for i := range obj {
		if !document.Gleich(obj[i], o.objekte[i]) {
			return false
		}
	}
	return true
}

// baue rastert die Ebene für den Stand von e; scheitert der Renderer (Schrift nicht zu laden), bleibt der Editor beim
// eigenen Zeichnen.
func (o *objektEbene) baue(e *editor) bool {
	d := e.s.D
	b, err := objektebeneRendern(d)
	if err != nil {
		diagnoseEditor("Objektebene: %v", err)
		o.frei()
		return false
	}
	o.an, o.dok, o.bild, o.groesse = true, d, paint.NewImageOp(b), d.Groesse()
	o.stand, o.pixel = d.Stand(), e.s.BildAenderung()
	o.objekte = append(o.objekte[:0], d.Objekte()...)
	o.gebaut++
	return true
}

// frei schaltet die Ebene ab und gibt Bild, Dokument und Objekte frei (F-799: das Bild in Ansichtsgröße, bis rund 262 MiB
// bei 32768×2100, blieb sonst bis zum nächsten Aufbau liegen). Die Zähler für Tests bleiben.
func (o *objektEbene) frei() {
	o.an, o.dok, o.bild, o.objekte = false, nil, paint.ImageOp{}, nil
}

// zeige legt die Ebene in Ansichtskoordinaten auf die Zeichenfläche.
func (o *objektEbene) zeige(gtx layout.Context) {
	c := clip.Rect(image.Rectangle{Max: o.groesse}).Push(gtx.Ops)
	o.bild.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	c.Pop()
}
