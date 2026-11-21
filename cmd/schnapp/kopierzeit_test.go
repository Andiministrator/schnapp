// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Messung zu F-281: Wie lange brauchen Abzug und render.PNG (Fenster für „Strg+C, sofort schließen“)? Läuft nur mit
// SCHNAPP_MESSUNG=1 (dauert, ohne Prüfung; ohne -race laufen lassen).
func TestKopierzeitPNG(t *testing.T) {
	if os.Getenv("SCHNAPP_MESSUNG") == "" {
		t.Skip("Messung: SCHNAPP_MESSUNG=1 setzen")
	}
	for _, fall := range []struct {
		name    string
		w, h    int
		rausch  bool
		flaeche bool
	}{
		{"1920x1080 flächig", 1920, 1080, false, true},
		{"3840x2160 flächig", 3840, 2160, false, true},
		{"3840x2160 Verlauf+Rauschen", 3840, 2160, true, false},
		{"7680x2160 flächig", 7680, 2160, false, true},
	} {
		b := image.NewNRGBA(image.Rect(0, 0, fall.w, fall.h))
		r := rand.New(rand.NewSource(1))
		for y := 0; y < fall.h; y++ {
			for x := 0; x < fall.w; x++ {
				i := b.PixOffset(x, y)
				if fall.rausch {
					b.Pix[i], b.Pix[i+1], b.Pix[i+2] = uint8(x*255/fall.w)+uint8(r.Intn(24)), uint8(y*255/fall.h)+uint8(r.Intn(24)), uint8(r.Intn(24))
				} else {
					v := uint8(240)
					if (x/200+y/120)%2 == 0 {
						v = 30
					}
					b.Pix[i], b.Pix[i+1], b.Pix[i+2] = v, v, v
				}
				b.Pix[i+3] = 255
			}
		}
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		a, _ := d.Abzug()
		tAbzug := time.Since(start)
		daten, err := render.PNG(a)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: Abzug %v, Abzug+PNG %v, %d Byte", fall.name, tAbzug.Round(time.Millisecond), time.Since(start).Round(time.Millisecond), len(daten))
	}
}
