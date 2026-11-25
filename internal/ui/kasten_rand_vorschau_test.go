// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// F-784 (Verify Sprint 13, V-4), Lücken zu F-765: Mutanten B6 und B9 in scripts/mutanten/text-naht.json.

// B6: Liegt die Mittellinie einer Seite des runden Kastens im Streifen der Naht, ragt ihr Strich seitlich in die
// sichtbaren Bereiche. Der Editor zeichnet ihn dort wie ohne Stauchen (der Rand von strichJeStueck rechnet den halben
// Strich ein: ohne ihn fehlte die Seite, weil sie keinen Bereich „berührte“).
func TestEditorRundKastenSeiteImStreifen(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	k := document.Kasten{Bereich: document.R(document.Pt(160, 40), document.Pt(330, 170)), Eckradius: 30,
		Strich: document.Strich{Farbe: gruen, Breite: 90}}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 200)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d), thema: material.NewTheme()}
	voll := headlessBild(t, image.Pt(400, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	zus := image.Rect(10, 14, 390, 190)
	sx0, sx1 := 150, 250
	if err := d.SetzeAnsicht(zus, []document.Streifen{{Von: sx0, Bis: sx1}}); err != nil {
		t.Fatal(err)
	}
	g := headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	band := int(document.KantenBandHalb) + 4
	ab, tinte := 0, 0
	for oy := zus.Min.Y; oy < zus.Max.Y; oy++ {
		for ox := zus.Min.X; ox < zus.Max.X; ox++ {
			if ox >= sx0-band && ox < sx1+band {
				continue
			}
			ax, ay := ox-zus.Min.X, oy-zus.Min.Y
			if ox >= sx1 {
				ax -= sx1 - sx0
			}
			v := voll.RGBAAt(ox, oy)
			if !rgbaNah(g.RGBAAt(ax, ay), v) {
				ab++
			}
			if ox < sx0 && v.A > 0 {
				tinte++
			}
		}
	}
	if tinte < 100 {
		t.Fatalf("zu wenig Tinte links der Naht (%d), der Test prüft nichts", tinte)
	}
	if ab != 0 {
		t.Errorf("%d Pixel weichen vom ungestauchten Bild ab (Tinte links der Naht %d)", ab, tinte)
	}
}

// B9: Die Vorschau der Ebenenliste zeichnet den Strich eines runden Kastens über den Gio-Strich in den Koordinaten, die
// die Vorschau vorgibt (Original ohne Ansicht), nicht über strichJeStueck, das in Ansichtskoordinaten des Dokuments
// arbeitet: Hier hat das Dokument einen Zuschnitt, der beide Wege auseinanderhält.
func TestVorschauRundKastenStrichImOriginal(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	k := document.Kasten{Bereich: document.R(document.Pt(100, 50), document.Pt(260, 150)), Eckradius: 20,
		Strich: document.Strich{Farbe: gruen, Breite: 6}}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 200)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAnsicht(image.Rect(20, 30, 380, 190), nil); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d), thema: material.NewTheme()}
	an := func(p document.Punkt) f32.Point { return f32.Pt(float32(p.X), float32(p.Y)) }
	b := headlessBild(t, image.Pt(400, 200), func(gtx layout.Context) {
		e.zeichneObjekt(gtx, k, an, image.Rect(0, 0, 400, 200), true)
	})
	// Mitte der linken und der oberen Seite im Original.
	for _, p := range []image.Point{{100, 100}, {180, 50}, {260, 100}, {180, 150}} {
		if c := b.RGBAAt(p.X, p.Y); c.G < 120 || c.A < 200 {
			t.Errorf("Vorschau: Pixel %v nicht im Strich: %v", p, c)
		}
	}
	if zaehleFarbe(b, color.NRGBA{G: 160, A: 255}) < 800 {
		t.Errorf("Vorschau: zu wenig Strichpixel")
	}
}
