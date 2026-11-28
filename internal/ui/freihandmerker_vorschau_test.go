// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// QA #333 (Fable): Leinwand und Vorschau der Ebenenliste teilen sich den Merker unter derselben Kennung. Die Vorschau
// verbreitert dünne Striche (F-1042) — verdrängen sich beide Fassungen gegenseitig?
func TestQAFreihandMerkerMitEbenenVorschau(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	ids := spiralStriche(t, d, 5, 4096)
	e := &editor{s: canvas.NeueSitzung(d)}
	bild := func() {
		gtx := layout.Context{Ops: new(op.Ops)}
		e.zeichneObjekte(gtx)
		for _, id := range ids {
			o, _ := d.Objekt(id)
			e.zeichneVorschau(gtx, o, image.Rect(0, 0, 40, 30))
		}
	}
	bild()
	nach1 := e.freihandNeu
	bild()
	bild()
	t.Logf("Zerlegungen nach Bild 1: %d, nach Bild 3: %d (5 Striche)", nach1, e.freihandNeu)
	if e.freihandNeu != nach1 {
		t.Errorf("Bilder 2 und 3 ohne Änderung zerlegten %d-mal neu", e.freihandNeu-nach1)
	}
}

// QA: ein breiter Strich (Vorschau ändert die Breite nicht) als Gegenprobe.
func TestQAFreihandMerkerBreiterStrich(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(10, 10), document.Pt(30, 20)},
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 12}))
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	for i := 0; i < 3; i++ {
		gtx := layout.Context{Ops: new(op.Ops)}
		e.zeichneObjekte(gtx)
		o, _ := d.Objekt(id)
		e.zeichneVorschau(gtx, o, image.Rect(0, 0, 40, 30))
	}
	t.Logf("breiter Strich: %d Zerlegungen in 3 Bildern", e.freihandNeu)
}
