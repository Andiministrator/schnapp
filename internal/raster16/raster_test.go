// SPDX-License-Identifier: GPL-3.0-or-later

package raster16

import (
	"image"
	"image/draw"
	"math/rand"
	"testing"

	"golang.org/x/image/vector"
)

// TestMaskeGleichVector: Die Maske ist Wert für Wert die Deckung, die x/image/vector seinem allgemeinen Zeichenweg
// übergibt. Mit DrawOp Src, deckender Quelle und *image.RGBA64 als Ziel schreibt vector genau diese Deckung als Alpha.
// Beide Rechenwege: Festkomma bis 512 px, Gleitkomma darüber (dort auf amd64 die Summenfolge der SIMD-Fassung).
func TestMaskeGleichVector(t *testing.T) {
	for _, g := range []image.Point{{300, 200}, {512, 64}, {513, 7}, {1100, 700}, {37, 1300}} {
		rnd := rand.New(rand.NewSource(int64(g.X*10007 + g.Y)))
		var z Rasterizer
		z.Reset(g.X, g.Y)
		v := vector.NewRasterizer(g.X, g.Y)
		v.DrawOp = draw.Src
		for p := 0; p < 12; p++ {
			n := 3 + rnd.Intn(6)
			for k := 0; k < n; k++ {
				// Punkte auch außerhalb der Maske, mit Bruchteilen.
				x := rnd.Float32()*float32(g.X)*1.4 - float32(g.X)*0.2
				y := rnd.Float32()*float32(g.Y)*1.4 - float32(g.Y)*0.2
				if k == 0 {
					z.MoveTo(x, y)
					v.MoveTo(x, y)
				} else {
					z.LineTo(x, y)
					v.LineTo(x, y)
				}
			}
			z.ClosePath()
			v.ClosePath()
		}
		ziel := image.NewRGBA64(image.Rect(0, 0, g.X, g.Y))
		v.Draw(ziel, ziel.Bounds(), image.Opaque, image.Point{})
		maske := z.Maske()
		abw := 0
		for y := 0; y < g.Y; y++ {
			for x := 0; x < g.X; x++ {
				if uint32(ziel.RGBA64At(x, y).A) != maske[y*g.X+x] {
					abw++
				}
			}
		}
		if abw != 0 {
			t.Errorf("%v: %d Pixel weichen von x/image/vector ab", g, abw)
		}
	}
}

// TestBloeckeGleichMaske: Blockweise akkumuliert ergibt dieselben Werte wie die ganze Maske, auch mit Blöcken, deren
// Länge nicht durch die Zeilenbreite, und einer letzten Zeile, deren Länge nicht durch vier teilbar ist.
func TestBloeckeGleichMaske(t *testing.T) {
	for _, g := range []image.Point{{300, 201}, {1100, 703}, {513, 9}, {37, 1301}} {
		for _, n := range []int{4, 64} {
			zeichne := func(z *Rasterizer) {
				z.Reset(g.X, g.Y)
				rnd := rand.New(rand.NewSource(int64(g.X + g.Y)))
				for p := 0; p < 8; p++ {
					z.MoveTo(rnd.Float32()*float32(g.X), rnd.Float32()*float32(g.Y))
					for k := 0; k < 4; k++ {
						z.LineTo(rnd.Float32()*float32(g.X), rnd.Float32()*float32(g.Y))
					}
					z.ClosePath()
				}
			}
			var a, b Rasterizer
			zeichne(&a)
			ganz := append([]uint32(nil), a.Maske()...)
			zeichne(&b)
			naechste := 0
			b.Bloecke(n, func(y0 int, m []uint32) {
				if y0*g.X != naechste {
					t.Fatalf("%v/%d: Block bei Zeile %d, erwartet %d", g, n, y0, naechste/g.X)
				}
				for i, v := range m {
					if v != ganz[naechste+i] {
						t.Fatalf("%v/%d: Wert %d weicht ab: %d statt %d", g, n, naechste+i, v, ganz[naechste+i])
					}
				}
				naechste += len(m)
			})
			if naechste != len(ganz) {
				t.Errorf("%v/%d: %d von %d Werten geliefert", g, n, naechste, len(ganz))
			}
		}
	}
}

// TestWiederverwendet: Ein Rasterizer über mehrere Füllungen (groß, klein, Maske ohne Leeren, wieder groß) liefert
// dieselben Werte wie ein frischer. Bloecke leert den Puffer blockweise und Reset verlässt sich darauf (sauber).
func TestWiederverwendet(t *testing.T) {
	zeichne := func(z *Rasterizer, g image.Point, saat int64) {
		z.Reset(g.X, g.Y)
		rnd := rand.New(rand.NewSource(saat))
		for p := 0; p < 6; p++ {
			z.MoveTo(rnd.Float32()*float32(g.X), rnd.Float32()*float32(g.Y))
			for k := 0; k < 5; k++ {
				z.LineTo(rnd.Float32()*float32(g.X), rnd.Float32()*float32(g.Y))
			}
			z.ClosePath()
		}
	}
	bloecke := func(z *Rasterizer) []uint32 {
		var alle []uint32
		z.Bloecke(64, func(_ int, m []uint32) { alle = append(alle, m...) })
		return alle
	}
	var alt Rasterizer
	schritte := []struct {
		g     image.Point
		maske bool
	}{{image.Pt(1100, 700), false}, {image.Pt(600, 600), false}, {image.Pt(900, 800), true},
		{image.Pt(700, 600), false}, {image.Pt(1100, 700), false}, {image.Pt(1200, 800), false}}
	for i, sch := range schritte {
		var neu Rasterizer
		zeichne(&alt, sch.g, int64(i))
		zeichne(&neu, sch.g, int64(i))
		var a []uint32
		if sch.maske {
			a = alt.Maske()
		} else {
			a = bloecke(&alt)
		}
		b := bloecke(&neu)
		for k := range b {
			if a[k] != b[k] {
				t.Fatalf("Schritt %d %v: Wert %d ist %d, frisch %d", i, sch.g, k, a[k], b[k])
			}
		}
	}
}
