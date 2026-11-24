// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// F-739 (ADR-0023 Nachtrag 13): Der Editor schneidet Text an einer Stauch-Naht ab wie der Renderer. Ein Text, dessen
// Anfang im Streifen liegt und der nur knapp herausragt, zeigt hinter der Naht nur den herausragenden Rest; vorher
// stand er ganz dahinter. Ein Text über dem ganzen Streifen zeigt links und rechts der Naht Tinte.
func TestEditorSchneidetTextAnDerNaht(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 360, 90)))
	if err != nil {
		t.Fatal(err)
	}
	schrift := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	tx := document.Text{Ort: document.Pt(70, 30), Inhalt: "GEHEIM-4711 Ende", Schriftgrad: 22, Farbe: schrift}
	if _, err := d.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	voll := headlessBild(t, image.Pt(360, 90), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	ende := 0
	for x := 359; x >= 0 && ende == 0; x-- {
		for y := 0; y < 90; y++ {
			if rgbaZuNRGBA(voll.RGBAAt(x, y)) == schrift {
				ende = x
				break
			}
		}
	}
	if ende < 150 {
		t.Fatalf("Aufbau: Textende bei %d", ende)
	}
	// Streifen von 60 bis 6 Spalten vor dem Textende: nur die letzten Spalten ragen heraus.
	if err := d.SetzeStreifen([]document.Streifen{{Von: 60, Bis: ende - 6}}); err != nil {
		t.Fatal(err)
	}
	b := headlessBild(t, image.Pt(360, 90), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	hinten := 0
	for y := 0; y < 90; y++ {
		for x := 60 + 12; x < 360; x++ {
			if c := rgbaZuNRGBA(b.RGBAAt(x, y)); c.R > 128 && c.G < 100 {
				hinten++
			}
		}
	}
	if hinten != 0 {
		t.Errorf("%d Pixel Schrift weiter als 12 Spalten hinter der Naht: Text starr hinter der Naht gezeichnet", hinten)
	}

	// Text über den ganzen Streifen: links und rechts der Naht Tinte, rechts genau um die Streifenbreite verschoben.
	tx.Ort.X = 20
	d2, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 360, 90)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d2.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	e = &editor{s: canvas.NeueSitzung(d2)}
	d = d2
	ohne := headlessBild(t, image.Pt(360, 90), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 140}}); err != nil {
		t.Fatal(err)
	}
	mit := headlessBild(t, image.Pt(360, 90), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	links, rechts, tinte := 0, 0, 0
	for y := 0; y < 90; y++ {
		for x := 0; x < 96; x++ {
			if !rgbaNah(mit.RGBAAt(x, y), ohne.RGBAAt(x, y)) {
				links++
			}
		}
		for x := 104; x < 320; x++ {
			if !rgbaNah(mit.RGBAAt(x, y), ohne.RGBAAt(x+40, y)) {
				rechts++
			}
			if rgbaZuNRGBA(mit.RGBAAt(x, y)) == schrift {
				tinte++
			}
		}
	}
	if tinte == 0 {
		t.Fatal("rechts der Naht keine Schrift")
	}
	if links != 0 || rechts != 0 {
		t.Errorf("Text an der Naht nicht wie ohne Stauchen beschnitten: links %d, rechts %d Pixel abweichend", links, rechts)
	}
}

func rgbaZuNRGBA(c color.RGBA) color.NRGBA { return color.NRGBAModel.Convert(c).(color.NRGBA) }

// F-695 (ADR-0023 Nachtrag 14): Dreieck einer Sprechblase und Verbindungslinie einer Lupe über einer Naht schneidet
// der Editor ab wie Renderer und SVG-Export: links der Naht wie ohne Stauchen, rechts um die Streifenbreite verschoben.
func TestEditorSchneidetSchraegeKantenAnDerNaht(t *testing.T) {
	sp := document.Pt(330, 170)
	fuell := color.NRGBA{R: 30, G: 90, B: 200, A: 255}
	gruen := color.NRGBA{G: 160, A: 255}
	for name, o := range map[string]document.Objekt{
		"Sprechblase": document.Text{Ort: document.Pt(20, 20), Inhalt: "Hi", Schriftgrad: 40,
			Farbe: color.NRGBA{A: 255}, Hintergrund: fuell, Spitze: &sp},
		"Lupe": document.Lupe{Quelle: document.R(document.Pt(20, 20), document.Pt(50, 50)),
			Ziel: document.R(document.Pt(280, 110), document.Pt(340, 170)), Strich: document.Strich{Farbe: gruen, Breite: 3}},
		"Pfeil":  document.GeraderPfeil(document.Pt(20, 40), document.Pt(380, 160), document.Strich{Farbe: gruen, Breite: 4}),
		"Nummer": document.Nummer{Mitte: document.Pt(170, 100), Wert: 7, Radius: 45, Farbe: gruen},
	} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 200)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
		e := &editor{s: canvas.NeueSitzung(d), thema: material.NewTheme()}
		ohne := headlessBild(t, image.Pt(400, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		if err := d.SetzeStreifen([]document.Streifen{{Von: 150, Bis: 190}}); err != nil {
			t.Fatal(err)
		}
		mit := headlessBild(t, image.Pt(400, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		links, rechts, sichtbar := 0, 0, 0
		nah := func(a, b color.RGBA) bool {
			d := func(x, y uint8) bool { return int(x)-int(y) <= 2 && int(y)-int(x) <= 2 }
			return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
		}
		for y := 0; y < 200; y++ {
			for x := 0; x < 146; x++ {
				if !nah(mit.RGBAAt(x, y), ohne.RGBAAt(x, y)) {
					links++
				}
			}
			for x := 154; x < 360; x++ {
				if !nah(mit.RGBAAt(x, y), ohne.RGBAAt(x+40, y)) {
					rechts++
				}
				if mit.RGBAAt(x, y).A > 0 {
					sichtbar++
				}
			}
		}
		if sichtbar == 0 && name != "Nummer" {
			t.Fatalf("%s: rechts der Naht nichts gezeichnet", name)
		}
		if links != 0 || rechts != 0 {
			t.Errorf("%s: über der Naht nicht abgeschnitten: links %d, rechts %d Pixel abweichend", name, links, rechts)
		}
	}
}

// rgbaNah lässt 2 Stufen je Kanal zu: Software-Rasterung (llvmpipe in der CI) glättet Kanten an anderer Lage um eine
// Stufe anders; ein falsch gesetzter oder unbeschnittener Text weicht um viele Stufen ab.
func rgbaNah(a, b color.RGBA) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= 2 && int(y)-int(x) <= 2 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}
