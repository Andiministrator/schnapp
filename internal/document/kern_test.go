// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// testkern ist die Familie der Testschrift testdata/fonts/schnapp-testkern.ttf: A und V mit kern-Paar (A,V) = -120
// Einheiten bei 1000 je em, ein „g“ mit 1,3 em Unterlänge (F-575, F-576).
const testkern = "Schnapp Testkern"

// ladeTestkern macht die Testschrift als einzige Systemschrift bekannt und stellt danach den Zustand von TestMain
// (keine Systemschriften) wieder her.
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
	if !fonts.Bekannt(testkern) {
		t.Fatalf("Testschrift %q nicht gefunden: %v", testkern, fonts.Familien())
	}
}

// TestKernPaarSkaliertMitGrad (F-575): Die Unterschneidung eines Paars wirkt im Schriftgrad — der Vorschub von „AV“
// ist um -0,12 em kleiner als der von „A“ und „V“ einzeln, im Grad 10 wie im Grad 100; Paare ohne Eintrag und
// Go (ohne Unterschneidung) bleiben unverändert.
func TestKernPaarSkaliertMitGrad(t *testing.T) {
	ladeTestkern(t)
	for _, grad := range []float64{10, 40, 100, 400} {
		tx := Text{Inhalt: "AV", Schriftgrad: grad, Schrift: testkern}
		einzeln := Text{Schriftgrad: grad, Schrift: testkern}
		diff := tx.Vorschub("AV") - einzeln.Vorschub("A") - einzeln.Vorschub("V")
		if soll := -0.12 * grad; math.Abs(diff-soll) > 0.05 {
			t.Errorf("Grad %v: Differenz %v, Soll %v", grad, diff, soll)
		}
		// „VA“ und „AA“ haben kein Paar.
		for _, z := range []string{"VA", "AA", "HH"} {
			a, b := z[:1], z[1:]
			if d := einzeln.Vorschub(z) - einzeln.Vorschub(a) - einzeln.Vorschub(b); math.Abs(d) > 1e-9 {
				t.Errorf("Grad %v, %q: Differenz %v, Soll 0", grad, z, d)
			}
		}
	}
	// Proportional: Verdoppeln des Grads verdoppelt die Differenz.
	d := func(grad float64) float64 {
		e := Text{Schriftgrad: grad, Schrift: testkern}
		return e.Vorschub("AV") - e.Vorschub("A") - e.Vorschub("V")
	}
	if a, b := d(50), d(100); math.Abs(b-2*a) > 0.05 {
		t.Errorf("Differenz bei 50: %v, bei 100: %v, Soll das Doppelte", a, b)
	}
	// Die Tinte folgt dem Vorschub: Das „V“ rückt um dieselbe Strecke nach links.
	tx := Text{Inhalt: "AV", Schriftgrad: 100, Schrift: testkern}
	r, ok, _ := textTinte(tx)
	if !ok {
		t.Fatal("keine Tinte")
	}
	// A (600) + Paar (-120) + V (Tinte bis 600) = 1080 Einheiten = 108 Pixel, dazu der Rand der Tinte.
	if soll := 108 + mitRand(Rechteck{}, 2).Max.X; math.Abs(r.Max.X-soll) > 0.05 {
		t.Errorf("rechte Kante der Tinte %v, Soll %v", r.Max.X, soll)
	}
}
