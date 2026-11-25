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

// F-751: Gegenstück zu TestKreuzendeNaehteMitZuschnitt im Renderer. Der Editor zeichnet die Beschriftungen über einer
// Spalten- und einer Zeilennaht mit Zuschnitt so, wie sie ohne Stauchen an derselben Stelle des Originals stünden:
// außerhalb der Kantenbänder Pixel für Pixel gleich (Toleranz 2 Stufen für llvmpipe), auch mit Deckkraft unter 100 %
// und für Pfeil und runden Kasten (F-748, F-749, ADR-0023 Nachträge 13 bis 17).
func TestEditorKreuzendeNaehteMitZuschnitt(t *testing.T) {
	sp := document.Pt(330, 170)
	schwarz := color.NRGBA{A: 255}
	gruen := color.NRGBA{G: 160, A: 255}
	blau := color.NRGBA{R: 30, G: 90, B: 100, A: 255}
	text := "WWW Geheimnis 1234567 WWW\nzweite Zeile mit Text MMMM\ndritte Zeile gjpqy XXXXXX\nvierte"
	objekte := map[string]document.Objekt{
		"Text":        document.Text{Ort: document.Pt(30, 60), Inhalt: text, Schriftgrad: 30, Farbe: schwarz},
		"TextDeck":    document.Text{Ort: document.Pt(30, 60), Inhalt: text, Schriftgrad: 30, Farbe: schwarz, Ebene: document.Ebene{Transparenz: 40}},
		"Nummer":      document.Nummer{Mitte: document.Pt(170, 150), Wert: 7, Radius: 70, Farbe: gruen},
		"NummerDeck":  document.Nummer{Mitte: document.Pt(170, 150), Wert: 7, Radius: 70, Farbe: gruen, Ebene: document.Ebene{Transparenz: 40}},
		"Sprechblase": document.Text{Ort: document.Pt(40, 40), Inhalt: "Hi", Schriftgrad: 40, Farbe: schwarz, Hintergrund: blau, Spitze: &sp},
		"Lupe": document.Lupe{Quelle: document.R(document.Pt(30, 30), document.Pt(60, 60)),
			Ziel: document.R(document.Pt(280, 110), document.Pt(340, 170)), Strich: document.Strich{Farbe: gruen, Breite: 3}},
		"Pfeil":     document.GeraderPfeil(document.Pt(20, 30), document.Pt(380, 180), document.Strich{Farbe: gruen, Breite: 12}),
		"PfeilDeck": document.GeraderPfeil(document.Pt(20, 30), document.Pt(380, 180), document.Strich{Farbe: gruen, Breite: 12}),
		// Spitze im Streifen: Über die abgebildeten Stützpunkte bekäme die Spitze außerhalb des Kantenbands eine andere
		// Form (F-748, Mutant P5).
		"PfeilSpitze": document.GeraderPfeil(document.Pt(20, 160), document.Pt(170, 170), document.Strich{Farbe: gruen, Breite: 16}),
		"Kasten": document.Kasten{Bereich: document.R(document.Pt(40, 30), document.Pt(260, 170)), Eckradius: 40,
			Strich: document.Strich{Farbe: gruen, Breite: 6}, Fuellung: blau},
	}
	pd := objekte["PfeilDeck"].(document.Pfeil)
	pd.Ebene = document.Ebene{Transparenz: 40}
	objekte["PfeilDeck"] = pd
	zus := image.Rect(10, 14, 390, 190)
	sx0, sx1, sy0, sy1 := 150, 190, 80, 110
	for name, o := range objekte {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 200)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
		e := &editor{s: canvas.NeueSitzung(d), thema: material.NewTheme()}
		voll := headlessBild(t, image.Pt(400, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		if err := d.SetzeAnsicht(zus, []document.Streifen{{Von: sx0, Bis: sx1}, {Zeilen: true, Von: sy0, Bis: sy1}}); err != nil {
			t.Fatal(err)
		}
		g := headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		band := int(document.KantenBandHalb) + 4
		ab, tinte := 0, 0
		for oy := zus.Min.Y; oy < zus.Max.Y; oy++ {
			if oy >= sy0-band && oy < sy1+band {
				continue
			}
			for ox := zus.Min.X; ox < zus.Max.X; ox++ {
				if ox >= sx0-band && ox < sx1+band {
					continue
				}
				ax, ay := ox-zus.Min.X, oy-zus.Min.Y
				if ox >= sx1 {
					ax -= sx1 - sx0
				}
				if oy >= sy1 {
					ay -= sy1 - sy0
				}
				v := voll.RGBAAt(ox, oy)
				if !rgbaNah(g.RGBAAt(ax, ay), v) {
					ab++
				}
				if v.A > 0 {
					tinte++
				}
			}
		}
		if tinte == 0 {
			t.Errorf("%s: keine Tinte im Vergleich, der Test prüft nichts", name)
		}
		if ab != 0 {
			t.Errorf("%s: %d Pixel weichen vom ungestauchten Bild ab (Tinte im Vergleich %d)", name, ab, tinte)
		}
	}
}
