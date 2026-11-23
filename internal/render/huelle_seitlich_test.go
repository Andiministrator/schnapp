// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"encoding/binary"
	"image"
	"os"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// rahmenSchrift lädt die Testschrift und setzt im head unitsPerEm und den Rahmen aller Glyphen neu (F-593).
func rahmenSchrift(t *testing.T, em uint16, xMin, yMin, xMax, yMax int16) *fonts.Schrift {
	t.Helper()
	b, err := os.ReadFile("../../testdata/fonts/schnapp-testkern.ttf")
	if err != nil {
		t.Fatal(err)
	}
	n := int(binary.BigEndian.Uint16(b[4:]))
	head := -1
	for i := 0; i < n; i++ {
		r := b[12+16*i:]
		if string(r[:4]) == "head" {
			head = int(binary.BigEndian.Uint32(r[8:]))
		}
	}
	if head < 0 {
		t.Fatal("kein head")
	}
	binary.BigEndian.PutUint16(b[head+18:], em)
	for i, v := range []int16{xMin, yMin, xMax, yMax} {
		binary.BigEndian.PutUint16(b[head+36+2*i:], uint16(v))
	}
	f, err := sfnt.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return &fonts.Schrift{Font: f}
}

// glyphenGezaehlt lässt text eine Zeile „g“ im Grad 100 ab x auf der Grundlinie 50 auf ein kleines Bild zählen
// und liefert die Zahl der gerechneten Glyphen.
func glyphenGezaehlt(t *testing.T, s *fonts.Schrift, x float64) int {
	t.Helper()
	face, err := opentype.NewFace(s.Font, &opentype.FaceOptions{Size: 100, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	z := zeichner{bild: image.NewNRGBA(image.Rect(0, 0, 100, 100)), rand: image.Rect(0, 0, 100, 100), auf: &aufwand{}}
	z.text(face, s, "g", x, 50, rot, 100)
	return z.auf.glyphen
}

// F-593: Die seitliche Grenze von text folgt der Hülle der Schrift: Eine Schrift mit Tinte 2,5 em rechts bzw. 3 em
// links des Ursprungs rechnet Glyphen, deren Ursprung 2 Schriftgrade hinter dem rechten bzw. 2,5 davor dem linken Rand
// liegt; mit dem Mindestwert von 1,25 Schriftgraden (Go) entfallen sie.
func TestTextSeitlicheGrenzeFolgtDerHuelle(t *testing.T) {
	schmal := rahmenSchrift(t, 1000, 0, 0, 500, 500)
	breitRechts := rahmenSchrift(t, 1000, 0, 0, 2500, 500)
	breitLinks := rahmenSchrift(t, 1000, -3000, 0, 500, 500)
	// Im Bereich der Mindestwerte (1,2 Schriftgrade neben dem Rand) zählt jede Schrift die Glyphe.
	for name, s := range map[string]*fonts.Schrift{"schmal": schmal, "rechts": breitRechts, "links": breitLinks} {
		if n := glyphenGezaehlt(t, s, 100+120); n != 1 {
			t.Errorf("%s, 1,2 Schriftgrade rechts: %d Glyphen, Soll 1", name, n)
		}
		if n := glyphenGezaehlt(t, s, -120); n != 1 {
			t.Errorf("%s, 1,2 Schriftgrade links: %d Glyphen, Soll 1", name, n)
		}
	}
	// Dahinter nur, wenn die Schrift dort noch Tinte haben kann.
	if n := glyphenGezaehlt(t, schmal, 100+200); n != 0 {
		t.Errorf("schmal, 2 Schriftgrade rechts: %d Glyphen, Soll 0", n)
	}
	if n := glyphenGezaehlt(t, breitRechts, 100+200); n != 1 {
		t.Errorf("breit rechts, 2 Schriftgrade rechts: %d Glyphen, Soll 1", n)
	}
	if n := glyphenGezaehlt(t, schmal, -250); n != 0 {
		t.Errorf("schmal, 2,5 Schriftgrade links: %d Glyphen, Soll 0", n)
	}
	if n := glyphenGezaehlt(t, breitLinks, -250); n != 1 {
		t.Errorf("breit links, 2,5 Schriftgrade links: %d Glyphen, Soll 1", n)
	}
}
