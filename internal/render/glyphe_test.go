// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"math/rand"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TestGlyphMischenBitgleich: glyphMischen ergibt dieselben Bytes wie draw.DrawMask mit draw.Over (der Weg von
// font.Drawer), auf deckendem, halb und ganz durchsichtigem Grund, mit deckender und halb durchsichtiger Farbe, und
// mit Glyphen, die über den Bildrand ragen (Zuschnitt). F-147.
func TestGlyphMischenBitgleich(t *testing.T) {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 37, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()
	rnd := rand.New(rand.NewSource(147))
	grund := image.NewNRGBA(image.Rect(0, 0, 120, 80))
	for i := 0; i < len(grund.Pix); i += 4 {
		grund.Pix[i], grund.Pix[i+1], grund.Pix[i+2] = byte(rnd.Intn(256)), byte(rnd.Intn(256)), byte(rnd.Intn(256))
		switch i / 4 % 3 {
		case 0:
			grund.Pix[i+3] = 0xff
		case 1:
			grund.Pix[i+3] = byte(rnd.Intn(256))
		default:
			grund.Pix[i+3] = 0
		}
	}
	farben := []color.NRGBA{{200, 30, 40, 0xff}, {10, 200, 90, 0x80}, {255, 255, 255, 3}}
	// Lagen innen, links/oben über den Rand, rechts/unten über den Rand, mit Teilpixeln.
	lagen := []fixed.Point26_6{{X: 10 << 6, Y: 50 << 6}, {X: -9<<6 + 17, Y: 12<<6 + 40}, {X: 100<<6 + 5, Y: 90 << 6}}
	geprueft := 0
	for _, farbe := range farben {
		for _, lage := range lagen {
			for _, c := range "Wgé@" {
				gr, maske, mp, _, _ := face.Glyph(lage, c)
				if gr.Empty() {
					continue
				}
				soll := image.NewNRGBA(grund.Rect)
				copy(soll.Pix, grund.Pix)
				draw.DrawMask(soll, gr, image.NewUniform(farbe), image.Point{}, maske, mp, draw.Over)
				ist := image.NewNRGBA(grund.Rect)
				copy(ist.Pix, grund.Pix)
				glyphMischen(ist, gr, maske, mp, farbe)
				if !bytes.Equal(soll.Pix, ist.Pix) {
					n := 0
					for i := range soll.Pix {
						if soll.Pix[i] != ist.Pix[i] {
							n++
						}
					}
					t.Errorf("Farbe %v, Lage %v, %q: %d Bytes weichen von draw.DrawMask ab", farbe, lage, c, n)
				}
				geprueft++
			}
		}
	}
	if geprueft < 30 {
		t.Fatalf("nur %d Glyphen geprüft", geprueft)
	}
}

// TestGlyphMischenKleineMaske: Ist die Maske kleiner als das Zielrechteck oder versetzt, schneidet glyphMischen wie
// draw.DrawMask auch auf die Maske zu (opentype liefert immer passende Masken, andere Gesichter nicht unbedingt).
func TestGlyphMischenKleineMaske(t *testing.T) {
	maske := image.NewAlpha(image.Rect(3, 2, 13, 9))
	for i := range maske.Pix {
		maske.Pix[i] = byte(40 + i*7)
	}
	grund := image.NewNRGBA(image.Rect(0, 0, 30, 30))
	for i := range grund.Pix {
		grund.Pix[i] = byte(i * 13)
	}
	for _, f := range []struct {
		gr image.Rectangle
		mp image.Point
	}{
		{image.Rect(5, 5, 25, 25), image.Pt(0, 0)},
		{image.Rect(5, 5, 25, 25), image.Pt(6, 4)},
		{image.Rect(-4, -3, 12, 10), image.Pt(3, 2)},
	} {
		soll := image.NewNRGBA(grund.Rect)
		copy(soll.Pix, grund.Pix)
		draw.DrawMask(soll, f.gr, image.NewUniform(color.NRGBA{9, 99, 199, 0xc0}), image.Point{}, maske, f.mp, draw.Over)
		ist := image.NewNRGBA(grund.Rect)
		copy(ist.Pix, grund.Pix)
		glyphMischen(ist, f.gr, maske, f.mp, color.NRGBA{9, 99, 199, 0xc0})
		if !bytes.Equal(soll.Pix, ist.Pix) {
			t.Errorf("%v/%v: weicht von draw.DrawMask ab", f.gr, f.mp)
		}
	}
}
