// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// F-974: Die Liste blättert zur Hauptauswahl, so wenig wie nötig; eine sichtbare Zeile lässt den Versatz stehen.
func TestBlaetternZurHauptauswahl(t *testing.T) {
	zeilen := []canvas.EbenenZeile{{}, {Gewaehlt: true}, {}, {}, {Gewaehlt: true, Haupt: true}, {}}
	i := zeileZumZeigen(zeilen)
	if i != 4 {
		t.Fatalf("Zeile %d, erwartet 4 (Hauptauswahl, nicht die erste gewählte)", i)
	}
	for _, f := range []struct{ versatz, soll int }{
		{0, 70},  // unterhalb: Zeile 4 (80–100) unten bündig in 30 hoher Liste
		{90, 80}, // oberhalb: oben bündig
		{75, 75}, // schon sichtbar: bleibt
	} {
		if got := blaetternZu(f.versatz, i, 20, 30); got != f.soll {
			t.Errorf("Versatz %d: %d, erwartet %d", f.versatz, got, f.soll)
		}
	}
	if zeileZumZeigen([]canvas.EbenenZeile{{Gewaehlt: true}}) != -1 {
		t.Error("ohne Hauptauswahl eine Zeile")
	}
}
