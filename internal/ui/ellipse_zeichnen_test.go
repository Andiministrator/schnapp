// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ADR-0047 Punkt 2: Der Editor zeichnet die Ellipse aus denselben Teilen wie der Renderer — gefüllt, voll, gestrichelt
// und gepunktet, als Kreis und flach mit dickem Strich; höchstens 0,2 % der gefärbten Pixel grob verschieden.
func TestEditorZeichnetEllipseWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	for _, art := range []document.Strichart{document.StrichVoll, document.StrichGestrichelt, document.StrichGepunktet} {
		bild := image.NewNRGBA(image.Rect(0, 0, 240, 160))
		draw.Draw(bild, bild.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
		d, err := document.Neu(bild)
		if err != nil {
			t.Fatal(err)
		}
		st := document.Strich{Farbe: color.NRGBA{R: 200, B: 40, A: 255}, Breite: 5, Art: art}
		dick := st
		dick.Breite = 16
		for _, o := range []document.Objekt{
			document.Kasten{Bereich: document.R(document.Pt(12, 10), document.Pt(110, 70)), Form: document.FormEllipse,
				Strich: st, Fuellung: color.NRGBA{R: 30, G: 60, B: 200, A: 255}},
			document.Kasten{Bereich: document.R(document.Pt(130, 10), document.Pt(200, 80)), Form: document.FormEllipse, Strich: st},
			document.Kasten{Bereich: document.R(document.Pt(20, 105), document.Pt(220, 135)), Form: document.FormEllipse, Strich: dick},
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
		gefaerbt, grob := 0, 0
		for y := 0; y < 160; y++ {
			for x := 0; x < 240; x++ {
				a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
				if b.G < 128 {
					gefaerbt++
				}
				if absInt(int(a.G)-int(b.G)) > 100 || absInt(int(a.B)-int(b.B)) > 100 {
					grob++
					if grob <= 10 {
						t.Logf("grob bei %d,%d: Editor %v, Renderer %v", x, y, a, b)
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

// Taste E wählt das Werkzeug Ellipse (ADR-0047 Punkt 4).
func TestTasteEWaehltEllipse(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.taste("E", 0)
	if l.e.s.Werkzeug != canvas.WerkzeugEllipse {
		t.Errorf("Werkzeug %v", l.e.s.Werkzeug)
	}
	if werkzeugSymbol(canvas.WerkzeugEllipse).Name == "" {
		t.Error("ohne Symbol")
	}
}

// Editor gegen Renderer über Stauch-Nähten und Zuschnitt, mit Ebenen-Deckkraft und halb durchsichtigem Strich
// (Kritik #328 E-2, Gegenprobe des Kritikers übernommen): höchstens 0,2 % der gefärbten Pixel grob verschieden.
func TestEditorZeichnetEllipseAnNaehtenWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	for _, art := range []document.Strichart{document.StrichVoll, document.StrichGestrichelt, document.StrichGepunktet} {
		for _, halb := range []bool{false, true} {
			bild := image.NewNRGBA(image.Rect(0, 0, 420, 320))
			draw.Draw(bild, bild.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
			d, err := document.Neu(bild)
			if err != nil {
				t.Fatal(err)
			}
			st := document.Strich{Farbe: color.NRGBA{R: 200, B: 40, A: 255}, Breite: 7, Art: art}
			if halb {
				st.Farbe.A = 128
			}
			k1 := document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(400, 300)), Form: document.FormEllipse,
				Strich: st, Fuellung: color.NRGBA{R: 30, G: 60, B: 200, A: 255}}
			k2 := document.Kasten{Bereich: document.R(document.Pt(60, 120), document.Pt(360, 150)), Form: document.FormEllipse, Strich: st}
			if halb {
				e := k1.Ebenenwerte()
				e.Transparenz = 40
				k1.Ebene = e
			}
			for _, o := range []document.Objekt{k1, k2} {
				if _, err := d.Hinzufuegen(o); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.SetzeStreifen([]document.Streifen{{Von: 170, Bis: 230}, {Zeilen: true, Von: 200, Bis: 240}}); err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeZuschnitt(image.Rect(30, 10, 410, 290)); err != nil {
				t.Fatal(err)
			}
			e := &editor{s: canvas.NeueSitzung(d)}
			soll, err := render.Bild(d)
			if err != nil {
				t.Fatal(err)
			}
			gr := soll.Rect.Size()
			ist := headlessBild(t, gr, func(gtx layout.Context) {
				paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				e.zeichneObjekte(gtx)
			})
			gefaerbt, grob := 0, 0
			for y := 0; y < gr.Y; y++ {
				for x := 0; x < gr.X; x++ {
					o := d.ZuOriginal(document.Pt(float64(x)+0.5, float64(y)+0.5))
					if math.Abs(o.X-170) < 24 || math.Abs(o.X-230) < 24 || math.Abs(o.Y-200) < 24 || math.Abs(o.Y-240) < 24 {
						continue // Kantenband und Schatten der Naht
					}
					a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
					if b.G < 250 {
						gefaerbt++
					}
					if absInt(int(a.G)-int(b.G)) > 100 || absInt(int(a.B)-int(b.B)) > 100 || absInt(int(a.R)-int(b.R)) > 100 {
						grob++
					}
				}
			}
			if gefaerbt < 500 || grob > gefaerbt/500 {
				t.Errorf("%s halb=%v: %d gefärbte Pixel, %d grob verschieden", art.Name(), halb, gefaerbt, grob)
			}
		}
	}
}
