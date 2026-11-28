// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ADR-0049 Punkt 2: Der Editor zeichnet Freihand-Striche aus denselben Teilen wie der Renderer — halbdurchsichtig,
// gemustert, als freier Textmarker und über einer Naht.
func TestEditorZeichnetFreihandWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 240, 160))
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if (y/12)%2 == 0 && x%50 < 30 {
				c = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	var welle []document.Punkt
	for i := 0; i <= 60; i++ {
		x := 10 + 3.5*float64(i)
		welle = append(welle, document.Pt(x, 40+20*math.Sin(x/15)))
	}
	schleife := []document.Punkt{document.Pt(20, 90), document.Pt(200, 140), document.Pt(200, 90), document.Pt(20, 140)}
	m := document.NeuerFreihand([]document.Punkt{document.Pt(10, 120), document.Pt(120, 118), document.Pt(230, 122)},
		document.Strich{Farbe: canvas.TextmarkerGelb, Breite: 14})
	m.Ebene.Mischung = document.MischMultiplizieren
	for _, o := range []document.Objekt{
		document.NeuerFreihand(welle, document.Strich{Farbe: color.NRGBA{R: 200, B: 40, A: 255}, Breite: 6}),
		document.NeuerFreihand(schleife, document.Strich{Farbe: color.NRGBA{B: 220, A: 128}, Breite: 10}),
		document.NeuerFreihand(welle[10:40], document.Strich{Farbe: color.NRGBA{G: 150, A: 255}, Breite: 4,
			Art: document.StrichGepunktet}),
		m,
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	for _, naht := range []bool{false, true} {
		if naht {
			if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 130}}); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		soll, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		g := soll.Bounds().Size()
		ist := headlessBild(t, g, func(gtx layout.Context) {
			paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			e.zeichneObjekte(gtx)
		})
		gefaerbt, grob := 0, 0
		for y := 0; y < g.Y; y++ {
			for x := 0; x < g.X; x++ {
				if o := d.ZuOriginal(document.Pt(float64(x)+0.5, float64(y)+0.5)); naht && math.Abs(o.X-100) < 24 ||
					naht && math.Abs(o.X-130) < 24 {
					continue // Kantenband der Naht zeichnen Editor und Renderer verschieden
				}
				a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
				if b != (color.NRGBA{255, 255, 255, 255}) && b != (color.NRGBA{30, 30, 30, 255}) {
					gefaerbt++
				}
				if absInt(int(a.R)-int(b.R)) > 60 || absInt(int(a.G)-int(b.G)) > 60 || absInt(int(a.B)-int(b.B)) > 60 {
					grob++
				}
			}
		}
		t.Logf("Naht %v: %d gefärbte Pixel, %d grob verschieden", naht, gefaerbt, grob)
		if gefaerbt < 1000 || grob > gefaerbt/500 {
			t.Errorf("Naht %v: %d gefärbte Pixel, %d grob verschieden (Soll höchstens 0,2 %%)", naht, gefaerbt, grob)
		}
	}
}

// ADR-0049 Punkte 4 und 5: F wählt Freihand; beim Textmarker zeigt die Leiste das Kästchen „Freehand“, ein Klick
// schaltet den Stil des Textmarkers.
func TestTasteFUndKaestchenFreihand(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.taste("F", 0)
	if l.e.s.Werkzeug != canvas.WerkzeugFreihand || werkzeugSymbol(canvas.WerkzeugFreihand).Name != "freehand" {
		t.Fatalf("Werkzeug %v", l.e.s.Werkzeug)
	}
	l.setle()
	if !l.e.frei.rahmen.Empty() {
		t.Error("Kästchen beim Werkzeug Freihand")
	}
	l.taste("H", 0)
	l.setle()
	r := l.e.frei.rahmen
	if r.Empty() {
		t.Fatal("kein Kästchen beim Textmarker")
	}
	l.klick(r.Min.Add(r.Max).Div(2))
	if !l.e.s.TextmarkerFrei() {
		t.Error("Klick auf das Kästchen schaltet nicht")
	}
	if h := l.e.hinweis(); h != i18n.T("Drag along the text to highlight it freely. Shift: horizontal line.") { // F-1039
		t.Errorf("Hinweis im freien Modus: %q", h)
	}
	l.klick(r.Min.Add(r.Max).Div(2))
	if l.e.s.TextmarkerFrei() {
		t.Error("zweiter Klick schaltet nicht zurück")
	}
}

// F-1042 (Kritik #333 K-8, Mutant FH42): Die Vorschau der Ebenenliste zeichnet auch einen sehr dünnen Strich auf großer
// Fläche mit mindestens 1,5 Pixeln der Vorschau, wie Pfeil, Kasten und Lupe.
func TestEbenenVorschauFreihandMindeststrich(t *testing.T) {
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2000, 2000)))
	if err != nil {
		t.Fatal(err)
	}
	o := document.NeuerFreihand([]document.Punkt{document.Pt(0, 0), document.Pt(1000, 1000), document.Pt(2000, 2000)},
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1})
	if _, err := d.Hinzufuegen(o); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	v := image.Rect(0, 0, 46, 46)
	ist := headlessBild(t, v.Size(), func(gtx layout.Context) {
		e.zeichneVorschau(gtx, o, v)
	})
	dunkel := 0
	for i := 4; i < 42; i++ { // entlang der Diagonale
		if c := ist.RGBAAt(i, i); c.A > 100 && c.R < 100 {
			dunkel++
		}
	}
	if dunkel < 30 {
		t.Errorf("nur %d von 38 Diagonalpunkten gezeichnet; der dünne Strich fehlt in der Vorschau", dunkel)
	}
}
