// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// F-789 (ADR-0023 Nachtrag 21): Die Vorschau der Ebenenliste zeichnet den Strich eines runden Kastens aus denselben
// Teilen wie Leinwand und Renderer (document.NeuerKastenStrich), nicht mit Gios Strich: Der hatte bei Strich > 2·Radius
// dasselbe Loch wie F-783. Verglichen wird mit dem Renderer wie in TestEditorDickerStrichKleinerKastenWieRenderer: kein
// Pixel voll gegen leer, abseits der Kanten dieselbe Farbe. Die Vorschau bildet das Original ohne Ansicht ab; das
// Dokument hat hier keine Ansicht, beide Wege zeigen dieselben Koordinaten.
func TestVorschauRundKastenWieRenderer(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	for _, fall := range []struct {
		seite          int
		radius, breite float64
	}{{40, 20, 60}, {30, 10, 50}, {40, 0.5, 30}, {120, 20, 40}, {60, 12, 20}} {
		name := fmt.Sprintf("Kasten %d Radius %v Strich %v", fall.seite, fall.radius, fall.breite)
		k := document.Kasten{Bereich: document.R(document.Pt(100, 80), document.Pt(float64(100+fall.seite),
			float64(80+fall.seite))), Eckradius: fall.radius, Strich: document.Strich{Farbe: gruen, Breite: fall.breite}}
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 360, 300)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(k); err != nil {
			t.Fatal(err)
		}
		e := &editor{s: canvas.NeueSitzung(d), thema: material.NewTheme()}
		an := func(p document.Punkt) f32.Point { return f32.Pt(float32(p.X), float32(p.Y)) }
		vor := headlessBild(t, image.Pt(360, 300), func(gtx layout.Context) {
			e.zeichneObjekt(gtx, k, an, image.Rect(0, 0, 360, 300), true)
		})
		rend, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		rgba := func(x, y int) color.RGBA { return color.RGBAModel.Convert(rend.At(x, y)).(color.RGBA) }
		innen, ab, loch := 0, 0, 0
		for y := 2; y < 298; y++ {
			for x := 2; x < 358; x++ {
				v, r := vor.RGBAAt(x, y), rgba(x, y)
				if (v.A < 64 && r.A > 192) || (v.A > 192 && r.A < 64) {
					loch++
				}
				gleich := r.A == 0 || r.A == 255
				for dy := -2; dy <= 2 && gleich; dy++ {
					for dx := -2; dx <= 2 && gleich; dx++ {
						gleich = rgba(x+dx, y+dy).A == r.A
					}
				}
				if gleich {
					innen++
					if !rgbaNah(v, r) {
						ab++
					}
				}
			}
		}
		m := 100 + fall.seite/2
		if innen == 0 || ab != 0 || loch != 0 {
			t.Errorf("%s: %d von %d Pixeln abseits der Kanten weichen vom Renderer ab, %d stehen voll gegen leer "+
				"(Mitte: Vorschau %d, Renderer %d)", name, ab, innen, loch, vor.RGBAAt(m, m-20).A, rgba(m, m-20).A)
		}
	}
}
