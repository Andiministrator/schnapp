// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0049 Punkt 2: Der Freihand-Strich deckt seine Pixel, halbdurchsichtig jedes höchstens einmal, auch wo er sich
// kreuzt; Punkte außerhalb bleiben weiß.
func TestFreihandZeichnetUeberlappungsfrei(t *testing.T) {
	d := weissesDokument(t, 120, 100)
	halb := color.NRGBA{R: 200, A: 128}
	// Eine Schleife: kreuzt sich bei (60, 50).
	pts := []document.Punkt{document.Pt(20, 20), document.Pt(100, 80), document.Pt(100, 20), document.Pt(20, 80)}
	hinzu(t, d, document.NeuerFreihand(pts, document.Strich{Farbe: halb, Breite: 10}))
	b := bild(t, d)
	einmal := b.NRGBAAt(40, 35) // auf der ersten Strecke
	kreuz := b.NRGBAAt(60, 50)  // Kreuzung
	if einmal.G == 255 {
		t.Fatalf("Strich nicht gezeichnet: %v", einmal)
	}
	if kreuz != einmal {
		t.Errorf("Kreuzung %v, einfach %v: doppelt gedeckt", kreuz, einmal)
	}
	if c := b.NRGBAAt(60, 10); c != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("außerhalb %v", c)
	}
	// Gemustert gibt es Lücken.
	d2 := weissesDokument(t, 120, 40)
	hinzu(t, d2, document.NeuerFreihand([]document.Punkt{document.Pt(10, 20), document.Pt(110, 20)},
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 4, Art: document.StrichGestrichelt}))
	b2 := bild(t, d2)
	luecke := false
	for x := 10; x < 110; x++ {
		if b2.NRGBAAt(x, 20).R == 255 {
			luecke = true
		}
	}
	if !luecke {
		t.Error("gestrichelt ohne Lücke")
	}
}
