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
		// Mindestens 513 hoch: vector rechnet dann wie raster16 im Gleitkomma-Weg (seit F-764 immer, ADR-0028 Nachtrag 1);
		// verglichen werden die ersten g.Y Zeilen.
		v := vector.NewRasterizer(g.X, max(g.Y, 513))
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
		ziel := image.NewRGBA64(image.Rect(0, 0, g.X, max(g.Y, 513)))
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

// TestMaskeGleichVectorZufall: viele kleine Zufallsfälle im Gleitkommaweg (Breite über 512, wenige Zeilen), damit auch
// seltene Rundungsunterschiede der Summenfolge auffallen (Sitzungs-QA Sprint 10, K-4: eine vertauschte Klammer in p2
// fiel den festen Fällen nicht auf).
func TestMaskeGleichVectorZufall(t *testing.T) {
	rnd := rand.New(rand.NewSource(4711))
	for lauf := 0; lauf < 600; lauf++ {
		g := image.Pt(513+rnd.Intn(300), 1+rnd.Intn(6))
		var z Rasterizer
		z.Reset(g.X, g.Y)
		v := vector.NewRasterizer(g.X, g.Y)
		v.DrawOp = draw.Src
		for p := 0; p < 1+rnd.Intn(5); p++ {
			for k, n := 0, 3+rnd.Intn(8); k < n; k++ {
				x := rnd.Float32()*float32(g.X)*1.2 - float32(g.X)*0.1
				y := rnd.Float32()*float32(g.Y)*1.6 - float32(g.Y)*0.3
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
		for i, m := range maske {
			if uint32(ziel.RGBA64At(i%g.X, i/g.X).A) != m {
				t.Fatalf("Lauf %d %v: Pixel %d weicht von x/image/vector ab", lauf, g, i)
			}
		}
	}
}

// F-764: Dieselbe Form (Koordinaten wie im Renderer in float64 verschoben, dann float32) an einer um ganze Pixel verschobenen Stelle ergibt dieselbe Maske (bis auf Rundung), auch wenn der
// Ausschnitt sie links anschneidet (Grenze 256 von 65535, rund eine 8-Bit-Stufe). Der Festkomma-Weg wich hier entlang
// langer Kanten um bis 22 % ab.
func TestVerschiebungsfest(t *testing.T) {
	rnd := rand.New(rand.NewSource(764))
	for lauf := 0; lauf < 201; lauf++ {
		w, h := 60+rnd.Intn(300), 60+rnd.Intn(300)
		var pts [][2]float64
		for k := 0; k < 3+rnd.Intn(4); k++ {
			pts = append(pts, [2]float64{rnd.Float64() * float64(w), rnd.Float64() * float64(h)})
		}
		if lauf == 200 {
			// Der gemessene Fall: Spitze eines Pfeils, Breite 32, Ausschnitt ab 64 (Sitzung 25.11., Karte #286).
			w, h = 145, 116
			pts = [][2]float64{{380 - 236, 190 - 75}, {236.89164944001348 - 236, 190 - 75}, {294.1349896640081 - 236, 75.51331955201078 - 75}}
		}
		maske := func(ox int, breite int) []uint32 {
			var z Rasterizer
			z.Reset(breite, h)
			for k, p := range pts {
				if k == 0 {
					z.MoveTo(float32(p[0]-float64(ox)), float32(p[1]))
				} else {
					z.LineTo(float32(p[0]-float64(ox)), float32(p[1]))
				}
			}
			z.ClosePath()
			return z.Maske()
		}
		voll := maske(0, w)
		ox := 1 + rnd.Intn(w-2)
		if lauf == 200 {
			ox = 64
		}
		teil := maske(ox, w-ox)
		for y := 0; y < h; y++ {
			for x := 0; x < w-ox; x++ {
				a, b := int(voll[y*w+x+ox]), int(teil[y*(w-ox)+x])
				if a-b > 256 || b-a > 256 {
					t.Fatalf("Lauf %d, Ausschnitt ab %d: Pixel (%d,%d) %d statt %d", lauf, ox, x+ox, y, b, a)
				}
			}
		}
	}
}
