// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/zeichnen/svg"
)

// Randfall-Proben (testdata/rasterproben): jede muss sich ohne Fehler zeichnen lassen und darf kein leeres Bild
// ergeben. Den Pixelvergleich gegen rsvg-convert macht dieser Test nicht (braucht ein externes Programm); er fängt
// Abstürze und still verschluckte Formen.
func TestRasterproben(t *testing.T) {
	dateien, err := filepath.Glob("testdata/rasterproben/*.svg")
	if err != nil || len(dateien) == 0 {
		t.Fatalf("keine Proben gefunden: %v", err)
	}
	for _, d := range dateien {
		t.Run(filepath.Base(d), func(t *testing.T) {
			daten, err := os.ReadFile(d)
			if err != nil {
				t.Fatal(err)
			}
			w, err := svg.Parse(daten, d)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Zeichnen(w, 200)
			if err != nil {
				t.Fatal(err)
			}
			for i := 3; i < len(b.Pix); i += 4 {
				if b.Pix[i] != 0 {
					return
				}
			}
			t.Fatal("Bild ist vollständig leer")
		})
	}
}
