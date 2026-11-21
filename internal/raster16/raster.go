// Übernommen aus golang.org/x/image/vector v0.26.0 (vector.go), BSD-Lizenz siehe LICENSE in diesem Ordner. Geändert:
// nur der Rasterkern (Pfade aus Strecken, Akkumulation), dazu Maske, die die 16-Bit-Deckung je Pixel herausgibt;
// Zeichnen auf Bilder und Kurven entfallen; die SIMD-Akkumulation ist in Go mit derselben Summenfolge nachgebaut,
// dazu Bloecke (Maske je Zeilenblock) und Leeren des Puffers beim Lesen (ADR-0028).

// Original copyright notice:
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package raster16 ist der Rasterkern von golang.org/x/image/vector, der die Deckung je Pixel mit 16 Bit herausgibt.
// x/image/vector zeichnet auf *image.NRGBA nur über einen allgemeinen Weg mit zwei Schnittstellenaufrufen je Pixel;
// mit der Maske mischt internal/render die Farbe selbst ein, mit denselben Zahlen (ADR-0028, F-145).
package raster16

import "image"

// floatingPointMathThreshold: ab dieser Breite oder Höhe rechnet der Rasterizer in Gleitkomma (wie x/image/vector).
const floatingPointMathThreshold = 512

func lerp(t, px, py, qx, qy float32) (x, y float32) {
	return px + t*(qx-px), py + t*(qy-py)
}

func clamp(i, width int32) uint {
	if i < 0 {
		return 0
	}
	if i < width {
		return uint(i)
	}
	return uint(width)
}

// Rasterizer füllt Pfade aus Strecken mit Kantenglättung (nonzero, Beträge wie x/image/vector).
type Rasterizer struct {
	bufF32 []float32
	bufU32 []uint32

	useFloatingPointMath bool
	// sauber: bufF32 ist auf seiner ganzen Kapazität null (Bloecke leert jeden Block nach dem Lesen); Reset spart sich
	// dann das Leeren.
	sauber bool
	// restSauber: bufF32 ist jenseits der aktuellen Länge null (für sauber nach dem nächsten Bloecke).
	restSauber bool

	size   image.Point
	firstX float32
	firstY float32
	penX   float32
	penY   float32
}

// Reset leert den Rasterizer für eine Maske der Breite w und Höhe h.
func (z *Rasterizer) Reset(w, h int) {
	z.size = image.Point{w, h}
	z.firstX = 0
	z.firstY = 0
	z.penX = 0
	z.penY = 0
	z.setUseFloatingPointMath(w > floatingPointMathThreshold || h > floatingPointMathThreshold)
}

func (z *Rasterizer) setUseFloatingPointMath(b bool) {
	z.useFloatingPointMath = b
	if z.useFloatingPointMath {
		if n := z.size.X * z.size.Y; n > cap(z.bufF32) {
			z.bufF32 = make([]float32, n)
			z.restSauber = true
		} else {
			z.bufF32 = z.bufF32[:n]
			if z.sauber {
				z.restSauber = true
			} else {
				clear(z.bufF32)
				z.restSauber = n == cap(z.bufF32)
			}
		}
		z.sauber = false
	} else {
		if n := z.size.X * z.size.Y; n > cap(z.bufU32) {
			z.bufU32 = make([]uint32, n)
		} else {
			z.bufU32 = z.bufU32[:n]
			clear(z.bufU32)
		}
	}
}

// Size gibt Breite und Höhe der Maske zurück.
func (z *Rasterizer) Size() image.Point { return z.size }

// ClosePath schließt den aktuellen Pfad.
func (z *Rasterizer) ClosePath() { z.LineTo(z.firstX, z.firstY) }

// MoveTo beginnt einen neuen Pfad bei (ax, ay); die Punkte dürfen außerhalb der Maske liegen.
func (z *Rasterizer) MoveTo(ax, ay float32) {
	z.firstX = ax
	z.firstY = ay
	z.penX = ax
	z.penY = ay
}

// LineTo fügt eine Strecke vom Stift nach (bx, by) an.
func (z *Rasterizer) LineTo(bx, by float32) {
	if z.useFloatingPointMath {
		z.floatingLineTo(bx, by)
	} else {
		z.fixedLineTo(bx, by)
	}
}

// Maske akkumuliert die Pfade und gibt die Deckung je Pixel zurück, zeilenweise, Breite Size().X, Werte 0 bis 0xffff —
// dieselben Zahlen, die x/image/vector seinem allgemeinen Zeichenweg übergibt. Die Maske gehört dem Rasterizer und
// gilt bis zum nächsten Reset; Maske darf je Reset nur einmal aufgerufen werden.
func (z *Rasterizer) Maske() []uint32 {
	if z.useFloatingPointMath {
		if n := z.size.X * z.size.Y; n > cap(z.bufU32) {
			z.bufU32 = make([]uint32, n)
		} else {
			z.bufU32 = z.bufU32[:n]
		}
		floatingAccumulateMask(z.bufU32, z.bufF32)
	} else {
		fixedAccumulateMask(z.bufU32)
	}
	return z.bufU32
}

// Bloecke akkumuliert die Pfade wie Maske, aber in Blöcken von je n Zeilen (n ein Vielfaches von 4), und ruft f je
// Block mit dessen erster Zeile und der Deckung dieser Zeilen auf (Breite Size().X). Auf großen Flächen spart das den
// Umweg über eine bildgroße Maske: Im Gleitkommaweg ist der Puffer nur n Zeilen groß und bleibt im Cache (F-145). Die
// Werte sind dieselben wie bei Maske. Der Block gilt nur während des Aufrufs von f; je Reset nur einmal aufrufen.
func (z *Rasterizer) Bloecke(n int, f func(y0 int, maske []uint32)) {
	if n <= 0 || n%4 != 0 {
		panic("raster16: Blockhöhe kein positives Vielfaches von 4")
	}
	w, h := z.size.X, z.size.Y
	if !z.useFloatingPointMath {
		m := z.Maske()
		for y0 := 0; y0 < h; y0 += n {
			f(y0, m[y0*w:min(y0+n, h)*w])
		}
		return
	}
	if k := n * w; k > cap(z.bufU32) {
		z.bufU32 = make([]uint32, k)
	}
	acc := float32(0)
	for y0 := 0; y0 < h; y0 += n {
		von, bis := y0*w, min(y0+n, h)*w
		blk := z.bufU32[:bis-von]
		acc = floatingAccumulateMaskAb(blk, z.bufF32[von:bis], acc)
		clear(z.bufF32[von:bis]) // noch im Cache; spart Reset das Leeren
		f(y0, blk)
	}
	z.sauber = z.restSauber
}
