// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-433: Nähte in beiden Achsen auf einem nicht quadratischen Bild. Das Soll kommt von document.Kantenband mit der
// richtigen Länge je Achse (Breite der Ansicht für eine Zeilen-Naht, Höhe für eine Spalten-Naht), nicht aus dem
// Renderer: Entlang jeder Mittellinie ist das Pixel auf der Linie durchsichtig (Spalt ±8), das Pixel 12 daneben
// (außerhalb von Spalt und Rändern, 8 + 2) bleibt Bild. Fiele die Linie einer Achse zu kurz aus (gemeinsame Linie für
// beide Achsen, falsche Länge), bliebe der Spalt dort leer. 300×120 deckt „Spalten-Naht mit der Breite gerechnet“ nicht
// auf (die Breite ist die größere), 120×300 nicht „gemeinsame Linie“ (die erste Linie ist die längere) — beide Lagen
// zusammen fangen beide Fehler.
func TestNaehteBeiderAchsenNichtQuadratisch(t *testing.T) {
	for _, gr := range []image.Point{{300, 120}, {120, 300}} {
		for _, form := range []document.Kante{document.KanteWelle, document.KanteZacken} {
			t.Run(fmt.Sprintf("%dx%d_%v", gr.X, gr.Y, form), func(t *testing.T) {
				d := grauesDokument(t, gr.X, gr.Y)
				if err := d.VorgabeKante(form); err != nil {
					t.Fatal(err)
				}
				// Je ein Streifen zwischen 45 und 50 in jeder Achse: Ansicht 5 Pixel kleiner je Achse.
				streifen(t, d, document.Streifen{Zeilen: true, Von: 45, Bis: 50}, document.Streifen{Zeilen: false, Von: 45, Bis: 50})
				naehte := d.Naehte()
				if len(naehte) != 2 {
					t.Fatalf("%d Nähte, Soll 2", len(naehte))
				}
				g, faktor := d.Groesse(), d.ZackenFaktor()
				if g.X != gr.X-5 || g.Y != gr.Y-5 {
					t.Fatalf("Ansicht %v", g)
				}
				b := bild(t, d)
				vonSpalt, bisSpalt := document.Spaltband()
				for _, n := range naehte {
					laenge := g.X // Länge der Naht: Breite der Ansicht für eine Zeilen-Naht
					if !n.Zeilen {
						laenge = g.Y
					}
					band := document.Kantenband(form, n, laenge, faktor, vonSpalt, bisSpalt)
					anz := len(band) / 2
					geprueft := 0
					for i := 0; i < anz; i++ {
						p, q := band[i], band[len(band)-1-i]
						mx, my := (p.X+q.X)/2, (p.Y+q.Y)/2
						// Entlang der Naht liegt die Stützstelle bei t; quer dazu die Mitte.
						t0 := my
						if n.Zeilen {
							t0 = mx
						}
						if t0 < 2 || t0 > float64(laenge)-3 {
							continue
						}
						// Nur wo keine Naht der anderen Achse nahe liegt (Abstand ≥ 25): Ihre Schichten lägen sonst über
						// dem Spalt dieser Naht.
						nahAnderer := func(x, y int) bool {
							for _, m := range naehte {
								if m.Zeilen == n.Zeilen {
									continue
								}
								pos := x
								if m.Zeilen {
									pos = y
								}
								if math.Abs(float64(pos-m.Lage)) < 25 {
									return true
								}
							}
							return false
						}
						if mxi, myi := int(math.Floor(mx)), int(math.Floor(my)); !nahAnderer(mxi, myi) {
							if a := b.NRGBAAt(mxi, myi).A; a != 0 {
								t.Errorf("Naht %+v: Mittellinie bei (%.1f, %.1f) nicht durchsichtig (Alpha %d)", n, mx, my, a)
							}
						}
						// 12 Pixel neben der Mittellinie (außerhalb von Spalt und Rändern), im Bild.
						for _, s := range []float64{-12, 12} {
							ox, oy := mx, my
							if n.Zeilen {
								oy += s
							} else {
								ox += s
							}
							x, y := int(math.Floor(ox)), int(math.Floor(oy))
							if x < 0 || y < 0 || x >= g.X || y >= g.Y {
								continue
							}
							if nahAnderer(x, y) {
								continue
							}
							if a := b.NRGBAAt(x, y).A; a == 0 {
								t.Errorf("Naht %+v: Pixel (%d, %d) neben der Mittellinie durchsichtig", n, x, y)
							}
						}
						geprueft++
					}
					if geprueft < laenge/10 {
						t.Errorf("Naht %+v: nur %d Stützstellen geprüft", n, geprueft)
					}
				}
			})
		}
	}
}
