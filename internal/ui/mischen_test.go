// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// mischDokument ist ein 96×64-Bild aus bunten Streifen (kein Grau, damit sRGB- und lineare Mischung weit auseinander
// liegen) mit halbdurchsichtigen Beschriftungen: ein Kasten mit Füllung und Strich, ein Kasten in einer Ebene mit
// Deckkraft 40 % und ein Text mit halbdurchsichtigem Schild.
func mischDokument(t *testing.T) *editor {
	t.Helper()
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			bild.SetNRGBA(x, y, color.NRGBA{R: uint8(40 + 20*(x/12)), G: uint8(220 - 25*(y/8)), B: uint8(30 + 25*((x+y)/16)), A: 255})
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	add := func(o document.Objekt) int {
		id, err := d.Hinzufuegen(o)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	add(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 28)),
		Fuellung: color.NRGBA{R: 255, G: 255, A: 128}, Strich: document.Strich{Farbe: color.NRGBA{R: 200, B: 255, A: 160}, Breite: 4}})
	k := add(document.Kasten{Bereich: document.R(document.Pt(50, 6), document.Pt(90, 40)), Fuellung: color.NRGBA{R: 255, A: 255}, Strich: document.Strich{Farbe: color.NRGBA{B: 255, A: 255}, Breite: 6}})
	if !e.s.SetzeDeckkraft(k, 40) {
		t.Fatal("Deckkraft der Ebene nicht gesetzt")
	}
	add(document.Text{Ort: document.Pt(6, 50), Inhalt: "M", Schriftgrad: 14, Farbe: color.NRGBA{A: 255},
		Hintergrund: color.NRGBA{R: 255, G: 255, B: 255, A: 100}})
	return e
}

// F-520, ADR-0020 Nachtrag 14: Der Editor mischt halbdurchsichtige Beschriftungen im sRGB-Raum wie der Renderer
// (vorher linear: bis zu 40 von 255 zu hell). Verglichen wird das Zeichnen des Editors (Gio ohne Fenster, Zoom 100 %)
// über dem Ansichtsbild mit render.Bild; Pixel an Kanten (Glättung) zählen nicht.
func TestEditorMischtWieRenderer(t *testing.T) {
	e := mischDokument(t)
	soll, err := render.Bild(e.s.D)
	if err != nil {
		t.Fatal(err)
	}
	ist := headlessBild(t, image.Pt(96, 64), func(gtx layout.Context) {
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
	pruefe("Füllung 50 %", image.Rect(10, 10, 34, 22))
	pruefe("Strich 63 %", image.Rect(5, 5, 7, 27))
	pruefe("Ebene 40 %", image.Rect(54, 10, 86, 36))
	pruefe("Ebene 40 %, Strich über Füllung (Gruppe)", image.Rect(51, 10, 53, 36))
	pruefe("Schild 39 %", image.Rect(4, 36, 9, 38))
}

// Deckendes bleibt, wie es war: Deckende Flächen schreiben ihre sRGB-Werte unverändert (Menü, Leisten, Schachbrett,
// Auswahlrahmen, ADR-0020 Nachtrag 14).
func TestEditorDeckendBleibtUnveraendert(t *testing.T) {
	farbe := color.NRGBA{R: 37, G: 142, B: 201, A: 255}
	b := headlessBild(t, image.Pt(8, 8), func(gtx layout.Context) { paint.Fill(gtx.Ops, farbe) })
	if p := b.RGBAAt(3, 3); p.R != 37 || p.G != 142 || p.B != 201 || p.A != 255 {
		t.Errorf("deckende Fläche: Pixel %v, Soll (37,142,201,255)", p)
	}
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
