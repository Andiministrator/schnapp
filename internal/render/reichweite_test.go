// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// tintenRahmen liefert das kleinste Rechteck um alle Pixel von b, die nicht grund sind.
func tintenRahmen(b *image.NRGBA, grund color.NRGBA) (image.Rectangle, bool) {
	var r image.Rectangle
	gefunden := false
	for y := b.Rect.Min.Y; y < b.Rect.Max.Y; y++ {
		for x := b.Rect.Min.X; x < b.Rect.Max.X; x++ {
			if b.NRGBAAt(x, y) == grund {
				continue
			}
			p := image.Rect(x, y, x+1, y+1)
			if gefunden {
				r = r.Union(p)
			} else {
				r, gefunden = p, true
			}
		}
	}
	return r, gefunden
}

// F-181: Die Reichweite von Text und Nummer (document.Reichweite, Grundlage von ObjektSichtbar) umfasst jedes Pixel,
// das der Renderer zeichnet, und liegt höchstens 2,5 Pixel (plus Rundung je Glyphe) daneben — so entscheiden Editor,
// Renderer, SVG-Export und Zähler nach der echten Breite, nicht nach einer Schätzung.
func TestReichweiteWieGerastert(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for name, fall := range map[string]struct {
		o     document.Objekt
		grund color.NRGBA
	}{
		"Satz":        {document.Text{Ort: document.Pt(20, 30), Inhalt: "Kennwort hunter2 nicht zeigen!", Schriftgrad: 20, Farbe: schwarz}, weiss},
		"schmal":      {document.Text{Ort: document.Pt(40, 30), Inhalt: "iiiiiiiiil", Schriftgrad: 30, Farbe: schwarz}, weiss},
		"breit":       {document.Text{Ort: document.Pt(10, 10), Inhalt: "WWMM@@", Schriftgrad: 25, Farbe: schwarz}, weiss},
		"Zeilen+Tab":  {document.Text{Ort: document.Pt(15, 12), Inhalt: "ab\tc\n\nÄgypten jq", Schriftgrad: 14, Farbe: schwarz}, weiss},
		"ohne Glyphe": {document.Text{Ort: document.Pt(15, 12), Inhalt: "x中文y", Schriftgrad: 18, Farbe: schwarz}, weiss},
		// Weiße Ziffern auf rotem Kreis und schwarzem Grund: Kreis und Ziffern sind Tinte, die Ziffern auch neben dem Kreis.
		"Nummer":  {document.Nummer{Mitte: document.Pt(150, 60), Wert: 88888, Radius: 8, Farbe: rot}, schwarz},
		"Nummer1": {document.Nummer{Mitte: document.Pt(150, 60), Wert: 1, Radius: 20, Farbe: rot}, schwarz},
	} {
		t.Run(name, func(t *testing.T) {
			b := image.NewNRGBA(image.Rect(0, 0, 400, 140))
			for i := 0; i < len(b.Pix); i += 4 {
				b.Pix[i], b.Pix[i+1], b.Pix[i+2], b.Pix[i+3] = fall.grund.R, fall.grund.G, fall.grund.B, 255
			}
			d, err := document.Neu(b)
			if err != nil {
				t.Fatal(err)
			}
			hinzu(t, d, fall.o)
			ist, ok := tintenRahmen(bild(t, d), fall.grund)
			if !ok {
				t.Fatal("keine Tinte")
			}
			r := document.Reichweite(fall.o)
			if float64(ist.Min.X) < r.Min.X || float64(ist.Min.Y) < r.Min.Y || float64(ist.Max.X) > r.Max.X ||
				float64(ist.Max.Y) > r.Max.Y {
				t.Errorf("Tinte %v ragt über die Reichweite %v", ist, r)
			}
			const spiel = 2.5 + 0.5 // Kantenglättung, Rundung, Anschnitt der Kantenpixel
			if float64(ist.Min.X)-r.Min.X > spiel || float64(ist.Min.Y)-r.Min.Y > spiel ||
				r.Max.X-float64(ist.Max.X) > spiel || r.Max.Y-float64(ist.Max.Y) > spiel {
				t.Errorf("Reichweite %v viel größer als die Tinte %v", r, ist)
			}
		})
	}
}
