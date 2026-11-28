// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-1023 (Kritik #328 E-1): Riesige Ellipsen, deren Ring das Bild nicht berührt, kosten nichts; gemusterte Riesen
// zerlegen in linearer Zeit; der Trockenlauf zählt auch den vollen Strich.

func TestEllipseRiesigAusserhalbSchnell(t *testing.T) {
	d := weissesDokument(t, 300, 220)
	for i := range document.MaxObjekte {
		v := float64(i % 1000)
		hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(-1e6+v, -1e6+v), document.Pt(1e6-v, 1e6-v)),
			Form: document.FormEllipse, Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6}})
	}
	t0 := time.Now()
	if err := Pruefe(d); err != nil {
		t.Errorf("Pruefe: %v", err)
	}
	if dt := time.Since(t0); ellipseZeitPruefen && dt > 3*time.Second {
		t.Errorf("Pruefe von %d Ellipsen außerhalb des Bilds: %v", document.MaxObjekte, dt)
	}
}

func TestEllipseRiesigDurchsBildBegrenzt(t *testing.T) {
	d := weissesDokument(t, 300, 220)
	for range document.MaxObjekte { // linker Scheitel bei (100, 110): der Ring kreuzt das Bild
		hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(100, 110-9e5), document.Pt(1e6, 110+9e5)),
			Form: document.FormEllipse, Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6}})
	}
	t0 := time.Now()
	err := Pruefe(d)
	if err == nil {
		t.Error("10 000 riesige Ellipsen durchs Bild ohne Budgetfehler")
	}
	if dt := time.Since(t0); ellipseZeitPruefen && dt > 5*time.Second {
		t.Errorf("Pruefe: %v", dt)
	}
}

func TestEllipseMusterRiesigLinear(t *testing.T) {
	r := document.R(document.Pt(-1e6, -1e6), document.Pt(1e6, 1e6))
	t0 := time.Now()
	sz := ZerlegeEllipse(r, document.Strich{Breite: 76, Art: document.StrichGestrichelt})
	if dt := time.Since(t0); ellipseZeitPruefen && dt > time.Second || len(sz.Teile) == 0 {
		t.Errorf("Zerlegen: %v, %d Teile", dt, len(sz.Teile))
	}
}

// abstandUmriss ist der Abstand von p zum geschlossenen Linienzug u.
func abstandUmriss(u []document.Punkt, p document.Punkt) float64 {
	best := math.Inf(1)
	for i := range u {
		a, b := u[i], u[(i+1)%len(u)]
		d := b.Minus(a)
		l2 := d.X*d.X + d.Y*d.Y
		t := 0.0
		if l2 > 0 {
			t = math.Min(1, math.Max(0, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l2))
		}
		best = math.Min(best, math.Hypot(p.X-(a.X+t*d.X), p.Y-(a.Y+t*d.Y)))
	}
	return best
}

// Ellipse über Stauch-Nähten und Zuschnitt, voll und gestrichelt, deckend und halb durchsichtig: Fläche und Außen jedes
// Pixels stimmen mit der Geometrie im Original (Kritik #328 E-2, Gegenprobe des Kritikers übernommen).
func TestEllipseNahtUndZuschnitt(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichVoll, document.StrichGestrichelt} {
		for _, deck := range []int{100, 50} {
			d := weissesDokument(t, 300, 220)
			r := document.R(document.Pt(40, 30), document.Pt(260, 190))
			k := document.Kasten{Bereich: r, Form: document.FormEllipse, Fuellung: color.NRGBA{B: 255, A: 255},
				Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 6, Art: art}}
			if deck < 100 {
				e := k.Ebenenwerte()
				e.Transparenz = uint8(100 - deck)
				k.Ebene = e
			}
			hinzu(t, d, k)
			if err := d.SetzeStreifen([]document.Streifen{{Von: 120, Bis: 180}, {Zeilen: true, Von: 80, Bis: 110}}); err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeZuschnitt(image.Rect(50, 10, 290, 180)); err != nil {
				t.Fatal(err)
			}
			b := bild(t, d)
			u := document.EllipsenUmriss(r)
			falsch := 0
			for y := b.Rect.Min.Y; y < b.Rect.Max.Y; y++ {
				for x := b.Rect.Min.X; x < b.Rect.Max.X; x++ {
					o := d.ZuOriginal(document.Pt(float64(x)+0.5, float64(y)+0.5))
					if math.Abs(o.X-120) < 24 || math.Abs(o.X-180) < 24 || math.Abs(o.Y-80) < 24 || math.Abs(o.Y-110) < 24 ||
						abstandUmriss(u, o) < 4.5 {
						continue // Kantenband der Nähte und Strichzone
					}
					c := b.NRGBAAt(x, y)
					innen := document.EllipseEnthaelt(r, o)
					var ok bool
					switch {
					case innen && deck == 100:
						ok = c.B == 255 && c.R == 0
					case innen:
						ok = c.B == 255 && c.R > 110 && c.R < 145
					default:
						ok = c.R == 255 && c.G == 255 && c.B == 255
					}
					if !ok {
						falsch++
					}
				}
			}
			if falsch > 0 {
				t.Errorf("%s Deckkraft %d: %d Pixel falsch", art.Name(), deck, falsch)
			}
		}
	}
}
