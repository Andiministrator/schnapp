// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ADR-0021 Nachtrag 25 Punkt 4: Der Editor zeichnet gemusterte Striche aus denselben Teilen wie der Renderer — Pfeil mit
// Köpfen, eckiger und runder Kasten, Lupe (Rahmen und Linie). Verglichen wird das Zeichnen ohne Fenster bei Zoom 100 %
// mit render.Bild; Kantenpixel weichen durch die Glättung leicht ab, ein fehlender oder versetzter Strich nicht. Voll
// ist dabei: Bis Punkt 11 stand im Editor am Ansatz der Spitzen eine helle Fuge (gegenläufige Vielecke).
func TestEditorZeichnetStrichartWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	for _, art := range []document.Strichart{document.StrichVoll, document.StrichGestrichelt, document.StrichGepunktet} {
		bild := image.NewNRGBA(image.Rect(0, 0, 240, 160))
		draw.Draw(bild, bild.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
		d, err := document.Neu(bild)
		if err != nil {
			t.Fatal(err)
		}
		st := document.Strich{Farbe: color.NRGBA{R: 200, B: 40, A: 255}, Breite: 5, Art: art}
		for _, o := range []document.Objekt{
			document.Pfeil{A: document.Pt(10, 20), B: document.Pt(220, 30), K: document.Pt(110, 0), Strich: st}.
				MitKoepfen(document.KopfPunkt, document.KopfDreieck),
			document.Kasten{Bereich: document.R(document.Pt(12, 50), document.Pt(100, 110)), Strich: st},
			document.Kasten{Bereich: document.R(document.Pt(120, 50), document.Pt(220, 110)), Eckradius: 14, Strich: st},
			document.Lupe{Quelle: document.R(document.Pt(14, 125), document.Pt(40, 150)),
				Ziel: document.R(document.Pt(150, 118), document.Pt(200, 150)), Strich: st},
		} {
			if _, err := d.Hinzufuegen(o); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		soll, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		ist := headlessBild(t, image.Pt(240, 160), func(gtx layout.Context) {
			paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			e.zeichneObjekte(gtx)
		})
		ziel := image.Rect(150, 118, 200, 150) // Bild im Ziel der Lupe: eigener Weg, hier nicht verglichen
		gefaerbt, grob := 0, 0
		for y := 0; y < 160; y++ {
			for x := 0; x < 240; x++ {
				if image.Pt(x, y).In(ziel.Inset(3)) {
					continue
				}
				a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
				if b.G < 128 {
					gefaerbt++
				}
				if absInt(int(a.G)-int(b.G)) > 100 {
					grob++
					if grob <= 10 {
						t.Logf("grob bei %d,%d: Editor %d, Renderer %d", x, y, a.G, b.G)
					}
				}
			}
		}
		t.Logf("%s: %d gefärbte Pixel, %d grob verschieden", art.Name(), gefaerbt, grob)
		if gefaerbt < 500 || grob > gefaerbt/500 {
			t.Errorf("%s: %d gefärbte Pixel, %d grob verschieden (Soll höchstens 0,2 %%)", art.Name(), gefaerbt, grob)
		}
	}
}
