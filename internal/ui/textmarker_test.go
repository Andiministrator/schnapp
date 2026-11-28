// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ADR-0048 Punkt 3: Der Editor zeichnet den Textmarker über Gios multiplizierende Ebene (Gio-Patch paint.PushMultiply)
// wie der Renderer: heller Grund nimmt die Farbe an, dunkle Schrift bleibt dunkel, bunter Grund wird je Kanal
// malgenommen, mit Deckkraft der Ebene und mit Schatten.
func TestEditorZeichnetTextmarkerWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 120; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			switch {
			case y >= 14 && y < 20 && x >= 30 && x < 70: // „Schrift“
				c = color.NRGBA{R: 20, G: 20, B: 20, A: 255}
			case x >= 90: // bunter Streifen
				c = color.NRGBA{R: uint8(40 + 3*y), G: 180, B: uint8(200 - 2*y), A: 255}
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	st := document.Strich{Farbe: canvas.TextmarkerGelb, Breite: 14}
	if _, err := d.Hinzufuegen(canvas.NeuerTextmarker(document.Pt(8, 17), document.Pt(112, 17), st, 0)); err != nil {
		t.Fatal(err)
	}
	halb := canvas.NeuerTextmarker(document.Pt(8, 42), document.Pt(112, 42), document.Strich{Farbe: canvas.TextmarkerRosa, Breite: 14}, 0)
	halb.Ebene = halb.Ebene.MitDeckkraft(50)
	if _, err := d.Hinzufuegen(halb); err != nil {
		t.Fatal(err)
	}
	sch := canvas.NeuerTextmarker(document.Pt(8, 64), document.Pt(80, 64), document.Strich{Farbe: canvas.TextmarkerGruen, Breite: 10}, 0)
	sch.Ebene.Schatten = document.Schatten{An: true, Winkel: 90, Laenge: 6, Farbe: color.NRGBA{B: 160, A: 255}}
	if _, err := d.Hinzufuegen(sch); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	soll, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	ist := headlessBild(t, image.Pt(120, 80), func(gtx layout.Context) {
		paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		e.zeichneObjekte(gtx)
	})
	pruefe := func(name string, r image.Rectangle) {
		t.Helper()
		groesst := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
				groesst = max(groesst, absInt(int(a.R)-int(b.R)), absInt(int(a.G)-int(b.G)), absInt(int(a.B)-int(b.B)))
			}
		}
		t.Logf("%s: größte Abweichung %d", name, groesst)
		if groesst > 2 {
			t.Errorf("%s: größte Abweichung %d je Kanal, Soll höchstens 2", name, groesst)
		}
	}
	pruefe("weißer Grund", image.Rect(14, 12, 28, 22))
	pruefe("Schrift", image.Rect(32, 15, 68, 19))
	pruefe("bunter Grund", image.Rect(92, 12, 110, 22))
	pruefe("Deckkraft 50 %", image.Rect(14, 37, 110, 47))
	pruefe("mit Schatten", image.Rect(14, 60, 74, 74))
	if c := ist.RGBAAt(50, 17); c.R > 30 || c.G > 30 {
		t.Errorf("Schrift unter dem Marker im Editor: %v, Soll dunkel", c)
	}
}

// Der Gio-Patch: PushMultiply mischt die Ebene als Ganzes multiplizierend, auch mit Deckkraft; überlappende Teile in
// der Ebene zählen einmal (Gruppe), und PushOpacity bleibt „über“.
func TestPushMultiplyMischtAlsGruppe(t *testing.T) {
	grund := color.NRGBA{R: 200, G: 120, B: 60, A: 255}
	ist := headlessBild(t, image.Pt(60, 20), func(gtx layout.Context) {
		paint.Fill(gtx.Ops, grund)
		m := paint.PushMultiply(gtx.Ops, 1)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 128, G: 255, B: 255, A: 255}, clip.Rect(image.Rect(0, 0, 20, 20)).Op())
		paint.FillShape(gtx.Ops, color.NRGBA{R: 128, G: 255, B: 255, A: 255}, clip.Rect(image.Rect(10, 0, 20, 20)).Op())
		m.Pop()
		h := paint.PushMultiply(gtx.Ops, 0.5)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, clip.Rect(image.Rect(20, 0, 40, 20)).Op())
		h.Pop()
		o := paint.PushOpacity(gtx.Ops, 1)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 255, A: 255}, clip.Rect(image.Rect(40, 0, 60, 20)).Op())
		o.Pop()
	})
	for _, f := range []struct {
		name string
		x    int
		soll color.NRGBA
	}{
		{"mal (128,255,255)", 5, color.NRGBA{R: 100, G: 120, B: 60}},
		{"überlappend, einmal gezählt", 15, color.NRGBA{R: 100, G: 120, B: 60}},
		{"Schwarz mit 50 %", 30, color.NRGBA{R: 100, G: 60, B: 30}},
		{"PushOpacity bleibt über", 50, color.NRGBA{B: 255}},
	} {
		c := ist.RGBAAt(f.x, 10)
		if absInt(int(c.R)-int(f.soll.R)) > 1 || absInt(int(c.G)-int(f.soll.G)) > 1 || absInt(int(c.B)-int(f.soll.B)) > 1 {
			t.Errorf("%s: %v, Soll %v", f.name, c, f.soll)
		}
	}
}

// ADR-0048 Punkt 4: H wählt den Textmarker, Strg+H blendet weiter aus; Knopf, Symbol und Name sind da, und die Leiste
// zeichnet die Kacheln des Textmarkers (multiplizierend, ohne Fehler).
func TestTasteHWaehltTextmarker(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	id := l.e.s.Auswahl
	if id == 0 {
		t.Fatal("kein gewähltes Objekt im Lauf")
	}
	l.taste("H", key.ModShortcut)
	if o, _ := l.e.s.D.Objekt(id); !o.Ebenenwerte().Ausgeblendet {
		t.Error("Strg+H blendet nicht mehr aus")
	}
	l.taste("H", 0)
	if l.e.s.Werkzeug != canvas.WerkzeugTextmarker {
		t.Errorf("Werkzeug %v", l.e.s.Werkzeug)
	}
	if werkzeugSymbol(canvas.WerkzeugTextmarker).Name != "highlighter" || l.e.werkzeugName(canvas.WerkzeugTextmarker) != "Highlighter" {
		t.Error("Symbol oder Name fehlt")
	}
	l.setle()
	if a, ok := l.e.s.SchnellArt(); !ok || a != canvas.SchnellTextmarker {
		t.Errorf("Schnellstil-Art %v", a)
	}
	if !strings.Contains(l.e.hinweis(), "highlight") {
		t.Errorf("Hinweis %q", l.e.hinweis())
	}
}

// Über Stauch-Nähten und Zuschnitt zeichnet der Editor den Textmarker wie der Renderer (je Bereich beschnitten).
func TestEditorZeichnetTextmarkerAnNaehtenWieRenderer(t *testing.T) {
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if (y/10)%3 == 0 && x%40 < 30 {
				c = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []document.Objekt{
		canvas.NeuerTextmarker(document.Pt(10, 35), document.Pt(290, 35), document.Strich{Farbe: canvas.TextmarkerGelb, Breite: 16}, 0),
		canvas.NeuerTextmarker(document.Pt(20, 180), document.Pt(280, 60), document.Strich{Farbe: canvas.TextmarkerGruen, Breite: 12}, 0),
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 120, Bis: 160}, {Zeilen: true, Von: 90, Bis: 110}}); err != nil {
		t.Fatal(err)
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
			o := d.ZuOriginal(document.Pt(float64(x)+0.5, float64(y)+0.5))
			if math.Abs(o.X-120) < 24 || math.Abs(o.X-160) < 24 || math.Abs(o.Y-90) < 24 || math.Abs(o.Y-110) < 24 {
				continue // Kantenband und Schatten der Naht zeichnen Editor und Renderer verschieden (ADR-0023 Nachtrag 11)
			}
			a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
			if b.B < 128 && b.R > 128 {
				gefaerbt++
			}
			if absInt(int(a.R)-int(b.R)) > 60 || absInt(int(a.G)-int(b.G)) > 60 || absInt(int(a.B)-int(b.B)) > 60 {
				grob++
			}
		}
	}
	t.Logf("%d gefärbte Pixel, %d grob verschieden", gefaerbt, grob)
	if gefaerbt < 500 || grob > gefaerbt/500 {
		t.Errorf("%d gefärbte Pixel, %d grob verschieden (Soll höchstens 0,2 %%)", gefaerbt, grob)
	}
}

// Die Kachel des Textmarkers zeigt die Mischung: Die dunkle „Zeile“ unter dem Strich bleibt dunkel (ADR-0048 Punkt 6).
func TestTextmarkerKachelMultipliziert(t *testing.T) {
	e := &editor{}
	st := canvas.EingebauteSchnellstile(canvas.SchnellTextmarker)[0]
	ist := headlessBild(t, image.Pt(40, 40), func(gtx layout.Context) {
		paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		e.zeichneMini(gtx, st, image.Rect(0, 0, 40, 40))
	})
	if c := ist.RGBAAt(15, 24); c.R > 60 || c.G > 60 {
		t.Errorf("Zeile unter dem Strich: %v, Soll dunkel", c)
	}
	if c := ist.RGBAAt(30, 20); c.R < 200 || c.B > 80 {
		t.Errorf("Strich über Weiß: %v, Soll gelb", c)
	}
}
