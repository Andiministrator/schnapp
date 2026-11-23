// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/fonts/fonttest"
)

// F-577: Scheitert die Glyphe (Umrisse unlesbar, fonttest.Defekt), steht ihr Vorschub trotzdem in der Messung: Editor
// (font.Face.GlyphAdvance) und Renderer setzen die nächste Glyphe hinter ihn, die Messung muss es ebenso tun —
// sonst liegt die Schreibmarke, das Schild und die Mitte/rechts-Ausrichtung um die Breite der Glyphe daneben.
func TestMessungSetztVorschubBeiGlyphenfehler(t *testing.T) {
	f := fonttest.Defekt(t)
	const grad = 40.0
	var puf sfnt.Buffer
	ppem := fixed.Int26_6(math.Round(grad * 64))
	adv := func(r rune) float64 {
		gi, _ := f.GlyphIndex(&puf, r)
		a, err := f.GlyphAdvance(&puf, gi, ppem, font.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		return float64(a) / 64
	}
	want := adv('A') + adv('B') + float64(KernPaar(f, &puf, 'A', 'B', grad))/64
	schriftMu.Lock()
	s := massVon(&fonts.Schrift{Font: f})
	vor, tinte, hat := s.zeile("AB", grad)
	schriftMu.Unlock()
	if math.Abs(vor-want) > 0.5 {
		t.Errorf("Vorschub von \"AB\" %.2f, erwartet %.2f (A %.2f + B %.2f)", vor, want, adv('A'), adv('B'))
	}
	// Das B steht hinter dem Vorschub des A: seine Tinte beginnt nicht vor ihm.
	if !hat || tinte.Min.X < adv('A')-1 {
		t.Errorf("Tinte %v (hat %v): beginnt vor dem Vorschub des A (%.2f)", tinte, hat, adv('A'))
	}
}
