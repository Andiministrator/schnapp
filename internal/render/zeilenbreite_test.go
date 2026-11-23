// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"math"
	"os"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// F-577: Die Breite einer Nummer misst mit der Unterschneidung im Schriftgrad (document.KernPaar), wie text sie setzt —
// nicht mit font.MeasureString, dessen Unterschneidung nicht mit dem Grad skaliert. Die Testschrift hat das Paar (A,V)
// = -120 Einheiten bei 1000 je em.
func TestZeilenBreiteMitUnterschneidung(t *testing.T) {
	roh, err := os.ReadFile("../../testdata/fonts/schnapp-testkern.ttf")
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(roh)
	if err != nil {
		t.Fatal(err)
	}
	s := &fonts.Schrift{Font: f}
	for _, grad := range []float64{10, 40, 100} {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			t.Fatal(err)
		}
		adv := func(r rune) float64 { a, _ := face.GlyphAdvance(r); return float64(a) / 64 }
		if got, want := zeilenBreite(face, s, "AV", grad), adv('A')+adv('V')-0.12*grad; math.Abs(got-want) > 0.05 {
			t.Errorf("Grad %v: Breite von \"AV\" %v, erwartet %v", grad, got, want)
		}
		if got, want := zeilenBreite(face, s, "VA", grad), adv('V')+adv('A'); math.Abs(got-want) > 1e-9 {
			t.Errorf("Grad %v: Breite von \"VA\" (ohne Paar) %v, erwartet %v", grad, got, want)
		}
		if got := zeilenBreite(face, s, "", grad); got != 0 {
			t.Errorf("leer: %v", got)
		}
	}
}
