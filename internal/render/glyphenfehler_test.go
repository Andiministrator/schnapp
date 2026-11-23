// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/fonts/fonttest"
)

// zeileBild setzt text im Grad 40 ab x auf die Grundlinie 60 eines 200×100-Bilds.
func zeileBild(t *testing.T, s *fonts.Schrift, text string, x float64) *image.NRGBA {
	t.Helper()
	face, err := opentype.NewFace(s.Font, &opentype.FaceOptions{Size: 40, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	b := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	z := zeichner{bild: b, rand: b.Bounds(), auf: &aufwand{}}
	z.text(face, s, text, x, 60, rot, 40)
	return b
}

// F-577: Eine Glyphe mit unlesbaren Umrissen (Face.Glyph meldet Vorschub 0) schiebt die nächste trotzdem um ihren
// Vorschub weiter, wie die Messung und der Editor: „AB“ setzt das B an dieselbe Stelle wie „B“ hinter dem Vorschub des A.
func TestTextVorschubBeiUnlesbarerGlyphe(t *testing.T) {
	s := &fonts.Schrift{Font: fonttest.Defekt(t)}
	face, err := opentype.NewFace(s.Font, &opentype.FaceOptions{Size: 40, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	adv, _ := face.GlyphAdvance('A')
	if adv == 0 {
		t.Fatal("Vorbedingung: A hat keinen Vorschub")
	}
	zwei := zeileBild(t, s, "AB", 10)
	nurB := zeileBild(t, s, "B", 10+float64(adv)/64)
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			if zwei.NRGBAAt(x, y) != nurB.NRGBAAt(x, y) {
				t.Fatalf("Pixel (%d,%d): „AB“ %v, „B“ hinter dem Vorschub %v — das B sitzt an anderer Stelle", x, y,
					zwei.NRGBAAt(x, y), nurB.NRGBAAt(x, y))
			}
		}
	}
}
