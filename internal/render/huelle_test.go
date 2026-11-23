// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// ladeTestkern macht die Testschrift (testdata/fonts, „g“ mit 1,3 em Unterlänge) bekannt und stellt danach den Zustand
// von TestMain (keine Systemschriften) wieder her.
func ladeTestkern(t *testing.T) {
	t.Helper()
	fonts.SetzeOrdner([]string{"../../testdata/fonts"})
	fonts.SucheZuruecksetzen()
	fonts.Suche()
	t.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	if !fonts.Bekannt("Schnapp Testkern") {
		t.Fatalf("Testschrift nicht gefunden: %v", fonts.Familien())
	}
}

// zeile0Tinte zählt die nicht weißen Pixel in der obersten Zeile.
func zeile0Tinte(b *image.NRGBA) int {
	n := 0
	for x := b.Rect.Min.X; x < b.Rect.Max.X; x++ {
		if c := b.NRGBAAt(x, b.Rect.Min.Y); c != weiss {
			n++
		}
	}
	return n
}

// TestUnterlaengeGrenze (F-576): Ein „g“ mit 1,3 em Unterlänge, dessen Grundlinie 1,1 Schriftgrade über dem Bild
// liegt, reicht mit der Unterlänge noch 0,2 Schriftgrade ins Bild. Die feste Grenze von 1 Schriftgrad unter der
// Grundlinie ließ die Zeile aus; mit der Hülle der Schrift steht die Tinte in Zeile 0. Go (Unterlänge unter 1 em)
// bleibt dabei unverändert ausgelassen.
func TestUnterlaengeGrenze(t *testing.T) {
	ladeTestkern(t)
	const grad = 100.0
	// Grundlinie = Ort.Y + Grad → -110.
	text := func(schrift string) *document.Dokument {
		d := weissesDokument(t, 100, 100)
		hinzu(t, d, document.Text{Ort: document.Pt(0, -210), Inhalt: "g", Schriftgrad: grad, Farbe: rot, Schrift: schrift})
		return d
	}
	if n := zeile0Tinte(bild(t, text("Schnapp Testkern"))); n < 10 {
		t.Errorf("Testschrift: %d Tintenpixel in Zeile 0, Soll mindestens 10 (g: 30 Pixel breit)", n)
	}
	// Dasselbe Zeichen in Go reicht nicht 1 em unter die Grundlinie: nichts im Bild, und es kostet nichts.
	d := text("")
	if n := zeile0Tinte(bild(t, d)); n != 0 {
		t.Errorf("Go: %d Tintenpixel in Zeile 0, Soll 0", n)
	}
	if a := trockenlauf(d, false); a.glyph != 0 {
		t.Errorf("Go: Kosten %v, Soll 0", a.glyph)
	}
	// Der Trockenlauf zählt die Zeile der Testschrift mit.
	dt := text("Schnapp Testkern")
	if a := trockenlauf(dt, false); a.glyph == 0 {
		t.Error("Testschrift: der Trockenlauf zählt die sichtbare Zeile nicht")
	}
}

// TestZeileSichtbarHuelle: zeileSichtbar folgt der Hülle der Schrift in allen drei Richtungen.
func TestZeileSichtbarHuelle(t *testing.T) {
	ladeTestkern(t)
	z := zeichner{rand: image.Rect(0, 0, 100, 100)}
	g := fonts.Laden(fonts.Go, fonts.Normal)
	k := fonts.Laden("Schnapp Testkern", fonts.Normal)
	// Grundlinie -110 bei Grad 100: Go reicht bis -10, die Testschrift bis +25.
	if z.zeileSichtbar(g, 0, -110, 100) || !z.zeileSichtbar(k, 0, -110, 100) {
		t.Errorf("unten: Go %v, Testschrift %v", z.zeileSichtbar(g, 0, -110, 100), z.zeileSichtbar(k, 0, -110, 100))
	}
	// Oben und seitlich bleibt es bei den Mindestwerten.
	if !z.zeileSichtbar(k, 0, 249, 100) || z.zeileSichtbar(k, 0, 251, 100) {
		t.Error("oben: Grenze nicht bei 1,5 Schriftgraden")
	}
	if !z.zeileSichtbar(k, 224, 50, 100) || z.zeileSichtbar(k, 226, 50, 100) {
		t.Error("rechts: Grenze nicht bei 1,25 Schriftgraden")
	}
}
