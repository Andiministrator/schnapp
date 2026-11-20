// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/draw/svg"
)

// Sollwerte der Randfall-Proben (testdata/rasterproben) bei Breite 200: Pixelzahl und Summen der Kanäle
// (vormultipliziertes RGBA). Gemessen am 19.11.2026 mit dem Stand, dessen Ausgaben im Schwesterprojekt byte-gleich
// zu den früheren Ergebnissen waren. Ein abweichender Wert heißt: das Zeichenergebnis hat sich geändert — gewollt
// nur mit Begründung und neuem Sollwert. Den Pixelvergleich gegen rsvg-convert ersetzt das nicht.
var probenSoll = []struct {
	datei          string
	alpha, r, g, b int
	pixel          int
}{
	{"clip-varianten.svg", 5100000, 3026815, 3100667, 3865777, 20000},
	{"dash-krumm.svg", 5100000, 4496070, 4496070, 4496070, 20000},
	{"dash-null.svg", 5100000, 4732800, 4732800, 4732800, 20000},
	{"dash-rect.svg", 5100000, 4820010, 4820010, 4820010, 20000},
	{"dash-rund-gerade.svg", 5100000, 4459480, 4459480, 4459480, 20000},
	{"hoehe.svg", 4371400, 0, 2924000, 0, 17200},
	{"maske-erbe.svg", 5100000, 2550000, 2550000, 5100000, 20000},
	{"negativ.svg", 5100000, 4130514, 4130514, 4130514, 20000},
	{"nulllaenge.svg", 5100000, 5019888, 5019888, 5019888, 20000},
	{"punkte.svg", 5100000, 4946160, 4946160, 4946160, 20000},
	{"strich-ecke.svg", 755600, 742330, 742330, 742330, 3000},
	{"uwende-a.svg", 5100000, 4651944, 4651944, 4651944, 20000},
	{"uwende-b.svg", 5100000, 4651944, 4651944, 4651944, 20000},
}

func TestRasterproben(t *testing.T) {
	dateien, err := filepath.Glob("testdata/rasterproben/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	if len(dateien) != len(probenSoll) {
		t.Fatalf("%d Proben im Ordner, %d Sollwerte — Tabelle nachziehen", len(dateien), len(probenSoll))
	}
	for _, soll := range probenSoll {
		t.Run(soll.datei, func(t *testing.T) {
			pfad := filepath.Join("testdata", "rasterproben", soll.datei)
			daten, err := os.ReadFile(pfad)
			if err != nil {
				t.Fatal(err)
			}
			w, err := svg.Parse(daten, pfad)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Zeichnen(w, 200)
			if err != nil {
				t.Fatal(err)
			}
			var alpha, r, g, bl int
			for i := 0; i < len(b.Pix); i += 4 {
				r += int(b.Pix[i])
				g += int(b.Pix[i+1])
				bl += int(b.Pix[i+2])
				alpha += int(b.Pix[i+3])
			}
			if n := len(b.Pix) / 4; n != soll.pixel || alpha != soll.alpha || r != soll.r || g != soll.g || bl != soll.b {
				t.Fatalf("Pixel %d, Summen A/R/G/B %d/%d/%d/%d; Soll %d, %d/%d/%d/%d",
					n, alpha, r, g, bl, soll.pixel, soll.alpha, soll.r, soll.g, soll.b)
			}
		})
	}
}
