// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// zeileZaehlen zeichnet eine Zeile „g“ im Grad 100 auf eine Ansicht von 100×100 und liefert die Zahl der Umrisse.
func zeileZaehlen(ts *textsatz, s *fonts.Schrift, x, y float64) int {
	gtx := layout.Context{Ops: new(op.Ops)}
	return ts.zeichneZeile(gtx, s, "g", x, y, 100, color.NRGBA{A: 255}, image.Rect(0, 0, 100, 100))
}

// F-590: Der Editor lässt Zeilen und Glyphen außerhalb der Ansicht nach derselben Hülle der Schrift aus wie der
// Renderer (fonts.Schrift.Huelle, F-576): unten 1 em (Go) bzw. 1,35 em (Testschrift mit Unterlänge), oben 1,5 em,
// seitlich 1,25 em. Gezählt werden die gezeichneten Glyphenumrisse, ohne Fenster.
func TestZeichneZeileHuelle(t *testing.T) {
	fonts.SetzeOrdner([]string{"../../testdata/fonts"})
	fonts.SucheZuruecksetzen()
	fonts.Suche()
	t.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	g := fonts.Laden(fonts.Go, fonts.Normal)
	k := fonts.Laden("Schnapp Testkern", fonts.Normal)
	if k.Ersatz {
		t.Fatalf("Testschrift nicht gefunden: %v", fonts.Familien())
	}
	var ts textsatz
	if n := zeileZaehlen(&ts, g, 0, 50); n != 1 {
		t.Fatalf("Grundfall: %d Umrisse, Soll 1", n)
	}
	// Unten: Grundlinie 1,1 Schriftgrade über der Ansicht. Go reicht nur 1 em unter die Grundlinie, die Testschrift
	// mit 1,35 em Unterlänge noch ins Bild.
	if n := zeileZaehlen(&ts, g, 0, -110); n != 0 {
		t.Errorf("unten, Go: %d Umrisse, Soll 0", n)
	}
	if n := zeileZaehlen(&ts, k, 0, -110); n != 1 {
		t.Errorf("unten, Testschrift: %d Umrisse, Soll 1 (Unterlänge reicht ins Bild)", n)
	}
	// Oben: Grenze bei 1,5 Schriftgraden über der Grundlinie.
	if a, b := zeileZaehlen(&ts, g, 0, 249), zeileZaehlen(&ts, g, 0, 251); a != 1 || b != 0 {
		t.Errorf("oben: bei 249 %d, bei 251 %d Umrisse, Soll 1 und 0", a, b)
	}
	// Rechts: Glyphenursprung bis 1,25 Schriftgrade hinter dem Rand.
	if a, b := zeileZaehlen(&ts, g, 220, 50), zeileZaehlen(&ts, g, 230, 50); a != 1 || b != 0 {
		t.Errorf("rechts: bei 220 %d, bei 230 %d Umrisse, Soll 1 und 0", a, b)
	}
	// Links: ebenso vor dem Rand.
	if a, b := zeileZaehlen(&ts, g, -120, 50), zeileZaehlen(&ts, g, -130, 50); a != 1 || b != 0 {
		t.Errorf("links: bei -120 %d, bei -130 %d Umrisse, Soll 1 und 0", a, b)
	}
}
