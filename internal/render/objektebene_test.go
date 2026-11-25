// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// objektebeneDokument trägt jede Objektart, auch halbdurchsichtig, mit Lupe und Sprechblase, auf deckendem Grund.
func objektebeneDokument(t *testing.T, st []document.Streifen) *document.Dokument {
	t.Helper()
	d := opakDokument(t, 300, 220, document.KanteZacken, 0)
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	gruen := color.NRGBA{G: 160, A: 200}
	spitze := document.Pt(30, 200)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(150, 120)), Eckradius: 18,
		Fuellung: color.NRGBA{R: 255, G: 200, A: 120}, Strich: document.Strich{Farbe: blau, Breite: 9}})
	hinzu(t, d, document.Kasten{Ebene: document.Ebene{Transparenz: 40}, Bereich: document.R(document.Pt(120, 40),
		document.Pt(280, 160)), Strich: document.Strich{Farbe: rot, Breite: 5}})
	hinzu(t, d, document.Pfeil{A: document.Pt(20, 180), B: document.Pt(270, 30), Strich: document.Strich{Farbe: gruen, Breite: 7}})
	hinzu(t, d, document.Text{Ort: document.Pt(60, 130), Inhalt: "Objektebene\nzwei Zeilen", Schriftgrad: 22,
		Farbe: color.NRGBA{A: 255}, Hintergrund: color.NRGBA{R: 255, G: 255, B: 200, A: 255}, Spitze: &spitze})
	hinzu(t, d, document.Nummer{Mitte: document.Pt(250, 190), Radius: 16, Wert: 7, Farbe: rot})
	hinzu(t, d, document.Lupe{Quelle: document.R(document.Pt(20, 20), document.Pt(60, 50)),
		Ziel: document.R(document.Pt(200, 170), document.Pt(280, 215)), Strich: document.Strich{Farbe: blau, Breite: 2}})
	return d
}

// F-787 (ADR-0023 Nachtrag 20): Die Objektebene über das Ansichtsbild gemischt ist das Bild des Renderers (ohne Naht,
// dort fehlen nur die Kanten); Mischen in zwei Schritten rundet anders, höchstens um 2 je Kanal.
func TestObjektebeneUeberAnsichtWieBild(t *testing.T) {
	d := objektebeneDokument(t, nil)
	ebene, err := Objektebene(d)
	if err != nil {
		t.Fatal(err)
	}
	soll := bild(t, d)
	ist := d.Ansichtsbild()
	draw.Draw(ist, ist.Rect, ebene, image.Point{}, draw.Over)
	abw, gefaerbt := 0, 0
	for i := 0; i < len(soll.Pix); i++ {
		if diff := int(soll.Pix[i]) - int(ist.Pix[i]); diff > 2 || diff < -2 {
			abw++
		}
	}
	for i := 3; i < len(ebene.Pix); i += 4 {
		if ebene.Pix[i] != 0 {
			gefaerbt++
		}
	}
	if abw != 0 {
		t.Errorf("%d Kanäle weichen um mehr als 2 vom Bild des Renderers ab", abw)
	}
	if gefaerbt < 10000 {
		t.Errorf("Objektebene fast leer: %d Pixel gedeckt", gefaerbt)
	}
}

// F-787: In Bändern gerastert ist die Objektebene dieselbe wie in einem Band, auch über kreuzenden Nähten; kein Band
// schreibt in die Zeilen eines anderen (unter -race geprüft). Bitgleich ist sie nicht: Der Rasterizer rechnet relativ
// zur oberen Kante seines Ausschnitts in float32, ein anderer Ausschnitt rundet die Kantenglättung um höchstens 1 anders.
func TestObjektebeneBaenderWieEinBand(t *testing.T) {
	st := []document.Streifen{{Von: 70, Bis: 80}, {Von: 160, Bis: 165}, {Zeilen: true, Von: 60, Bis: 66},
		{Zeilen: true, Von: 140, Bis: 150}}
	for _, streifen := range [][]document.Streifen{nil, st} {
		d := objektebeneDokument(t, streifen)
		eins, err := objektebene(d, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range []int{2, 5, 8} {
			viele, err := objektebene(d, n)
			if err != nil {
				t.Fatal(err)
			}
			abw, maxd := 0, 0
			for i := range eins.Pix {
				if dd := int(eins.Pix[i]) - int(viele.Pix[i]); dd != 0 {
					abw++
					maxd = max(maxd, dd, -dd)
				}
			}
			if maxd > 1 || abw > len(eins.Pix)/500 {
				t.Errorf("Nähte %d, %d Bänder: %d Kanäle weichen von einem Band ab, höchstens um %d", len(streifen), n,
					abw, maxd)
			}
		}
	}
}
