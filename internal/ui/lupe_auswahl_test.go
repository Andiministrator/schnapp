// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-606 (Karte #231): Bei einer gewählten Lupe zeigt die Auswahl je Rechteck (Quelle, Ziel) einen eigenen Rahmen, nicht
// eine Hülle über beide: In der leeren Fläche zwischen weit auseinanderliegenden Rechtecken liegt kein Rahmenpixel.
func TestLupeAuswahlJeRechteckEinRahmen(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	id, err := l.e.s.D.Hinzufuegen(document.Lupe{
		Quelle: document.R(document.Pt(1, 1), document.Pt(6, 5)),
		Ziel:   document.R(document.Pt(32, 22), document.Pt(39, 29)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 1}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	b := headlessBild(t, image.Pt(1200, 800), func(gtx layout.Context) { l.e.zeichneAuswahl(gtx) })
	fe := func(x, y float64) image.Point {
		q := l.e.s.Abb.ZuFenster(l.e.s.D.ZuAnsicht(document.Pt(x, y)))
		return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	}
	ist := func(x, y int) bool {
		p := b.RGBAAt(x, y)
		return p.R == farbeAuswahl.R && p.G == farbeAuswahl.G && p.B == farbeAuswahl.B && p.A == farbeAuswahl.A
	}
	// Die Hülle beider Rechtecke hätte ihre obere Kante über dem Ziel und ihre linke Kante neben dem Ziel; dort liegt
	// nichts von einem der beiden Rechtecke. (Original 31..40 über dem Ziel, links daneben 21..30: abseits der Griffe.)
	for _, c := range []struct {
		n              string
		x0, y0, x1, y1 float64
	}{{"obere Hüllkante über dem Ziel", 31, 0, 40, 3}, {"linke Hüllkante neben dem Ziel", 0, 20, 3, 31}} {
		a, z := fe(c.x0, c.y0), fe(c.x1, c.y1)
		r := image.Rectangle{Min: a, Max: z}.Canon().Inset(-4)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if ist(x, y) {
					t.Fatalf("%s: Rahmenpixel (%d,%d) dort, wo nur eine Hülle über beide Rechtecke einen Rahmen hätte", c.n, x, y)
				}
			}
		}
	}
	// Dafür hat jedes Rechteck seinen Rahmen: Pixel nahe der oberen Kante von Quelle und Ziel.
	for _, c := range []struct {
		n    string
		x, y float64
	}{{"Quelle", 3.5, 1}, {"Ziel", 35.5, 22}} {
		p := fe(c.x, c.y)
		n := 0
		for dy := -4; dy <= 1; dy++ {
			for dx := -2; dx <= 2; dx++ {
				q := b.RGBAAt(p.X+dx, p.Y+dy)
				if q.R == farbeAuswahl.R && q.G == farbeAuswahl.G && q.B == farbeAuswahl.B && q.A == farbeAuswahl.A {
					n++
				}
			}
		}
		if n == 0 {
			t.Errorf("%s: kein Rahmenpixel an der oberen Kante", c.n)
		}
	}
}
