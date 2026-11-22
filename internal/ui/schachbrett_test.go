// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0023 Nachtrag 7: Das Brett deckt das ganze Band ±16·Faktor um die Naht und liegt im festen Raster der Ansicht:
// Das Feld bei Ansicht (x, y) ist hell, wenn x/8 + y/8 gerade ist — gleich, wo die Naht liegt und welche Achse.
func TestSchachbrettRaster(t *testing.T) {
	for _, c := range []struct {
		n      document.Naht
		laenge int
		faktor int
	}{
		{document.Naht{Zeilen: true, Lage: 100}, 300, 1}, {document.Naht{Zeilen: true, Lage: 37}, 300, 1},
		{document.Naht{Zeilen: false, Lage: 9}, 200, 1}, {document.Naht{Zeilen: false, Lage: 500}, 900, 3},
	} {
		erstes, quer, lang := schachLage(c.n, c.laenge, c.faktor)
		halb := int(document.ZackenBandHalb) * c.faktor
		if erstes*schachFeld > c.n.Lage-halb || (erstes+quer)*schachFeld < c.n.Lage+halb {
			t.Errorf("%+v: Brett %d..%d deckt das Band %d..%d nicht", c.n, erstes*schachFeld, (erstes+quer)*schachFeld,
				c.n.Lage-halb, c.n.Lage+halb)
		}
		if lang*schachFeld < c.laenge+8*c.faktor {
			t.Errorf("%+v: Brett %d lang, Soll mindestens %d", c.n, lang*schachFeld, c.laenge+8*c.faktor)
		}
		b := schachBild(lang, quer+1, c.n.Zeilen, erstes%2 != 0)
		for i := 0; i < lang; i++ {
			for j := 0; j <= quer; j++ {
				fx, fy := i, erstes+j // Feld in der Ansicht
				px, py := i, j        // Pixel im Bild
				if !c.n.Zeilen {
					fx, fy = erstes+j, i
					px, py = j, i
				}
				soll := schachHell
				if (fx+fy)%2 != 0 {
					soll = schachDunkel
				}
				if got := b.NRGBAAt(px, py); got != soll {
					t.Fatalf("%+v: Feld (%d,%d) = %v, Soll %v", c.n, fx, fy, got, soll)
				}
			}
		}
	}
	// male legt je Achse und Parität höchstens ein Bild an.
	var s schachbretter
	var o op.Ops
	for _, lage := range []int{100, 108, 116, 124} {
		s.male(&o, document.Naht{Zeilen: true, Lage: lage}, 300, 1)
	}
	if !s.da[0][0] || !s.da[0][1] || s.da[1][0] || s.da[1][1] {
		t.Errorf("Bilder %v", s.da)
	}
}
