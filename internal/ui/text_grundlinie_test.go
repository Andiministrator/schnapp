// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// F-108: Der Editor setzt Text auf die Grundlinie (Ort.Y + Schriftgrad) wie der Renderer, nicht mit der Oberkante des
// Textfelds auf Ort.Y. Die Grundlinie des echten Labels in Go Regular liegt unter der Oberkante, und textEcke
// legt sie genau auf die verlangte Höhe.
func TestEditorTextAufGrundlinie(t *testing.T) {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	for _, grad := range []float32{12, 40, 100} {
		gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(1<<20, 1<<20)}}
		lbl := material.Label(th, unit.Sp(grad), "H")
		lbl.MaxLines = 1
		dims := lbl.Layout(gtx)
		if dims.Baseline <= 0 || dims.Baseline >= dims.Size.Y {
			t.Fatalf("Grad %v: Grundlinie %d außerhalb des Felds %v", grad, dims.Baseline, dims.Size)
		}
		// Die Grundlinie liegt bei Ort.Y + Schriftgrad; das Feld darf also nicht an Ort.Y beginnen.
		p := f32.Pt(10, 50+grad)
		ecke := textEcke(p, dims, false)
		if got := ecke.Y + float32(dims.Baseline); math.Abs(float64(got-p.Y)) > 1e-4 {
			t.Errorf("Grad %v: Grundlinie liegt bei %v, Soll %v", grad, got, p.Y)
		}
		if ecke.Y == 50 {
			t.Errorf("Grad %v: Feld beginnt an der Oberkante wie vor F-108", grad)
		}
		if m := textEcke(p, dims, true); m.X != 10-float32(dims.Size.X)/2 {
			t.Errorf("Grad %v: Mitte bei %v", grad, m.X)
		}
	}
}
