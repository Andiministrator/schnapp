// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// kastenDeckung misst die Abweichung des Strichs eines runden Kastens von der analytischen Deckung (32×32 Stichproben je
// Pixel an der Kante, abseits der Kante voll oder leer); wie TestKastenStrichDecktWieAnalytisch, ohne dessen Grenze.
func kastenDeckung(t *testing.T, x0, y0, x1, y1, radius, breite float64) (kante, abseits float64) {
	t.Helper()
	r := document.R(document.Pt(x0, y0), document.Pt(x1, y1))
	d := weissesDokument(t, 200, 180)
	hinzu(t, d, document.Kasten{Bereich: r, Eckradius: radius,
		Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: breite}})
	b := bild(t, d)
	umriss := document.RundesRechteck(r, radius)
	hw := breite / 2
	const n = 32
	for y := 0; y < 180; y++ {
		for x := 0; x < 200; x++ {
			m := abstandZumUmriss(document.Pt(float64(x)+0.5, float64(y)+0.5), umriss)
			ist := 1 - float64(b.NRGBAAt(x, y).R)/255
			if math.Abs(m-hw) > 1 {
				soll := 0.0
				if m < hw {
					soll = 1
				}
				abseits = math.Max(abseits, math.Abs(ist-soll))
				continue
			}
			treffer := 0
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					p := document.Pt(float64(x)+(float64(sx)+0.5)/n, float64(y)+(float64(sy)+0.5)/n)
					if abstandZumUmriss(p, umriss) <= hw {
						treffer++
					}
				}
			}
			kante = math.Max(kante, math.Abs(ist-float64(treffer)/(n*n)))
		}
	}
	return kante, abseits
}

// Bewusste Grenzen (ADR-0023 Nachträge 21 und 22): Diese Fälle weichen mehr als die allgemeine Toleranz ab, das ist
// hingenommen. Der Test hält die gemessenen Werte fest, damit sie nicht unbemerkt wachsen (Sitzungs-Kritik 25.11.2026,
// F-814). Gemessen: Kreis R8 0,062, Kreis R1 0,082 (F-807); Fast-Kreis 40,1×40,2 an (30,5/30,5) 0,136 (F-809); doppelt überstrichene
// Strecke einer Kehrtwende 0,41 bei Kante auf 0,4 px, 0,50 bei Kante auf halbem Pixel (F-810, F-814).
func TestBewussteGrenzenBleibenKlein(t *testing.T) {
	for _, f := range []struct {
		name                           string
		x0, y0, x1, y1, radius, breite float64
		grenze                         float64
	}{
		{"Kreis R8 Strich 15,5 (F-807)", 30.5, 30.5, 46.5, 46.5, 8, 15.5, 0.09},
		{"Kreis R1 Strich 1,5 (F-807)", 30.5, 30.5, 32.5, 32.5, 1, 1.5, 0.10},
		{"Fast-Kreis 40,1×40,2 R20 Strich 39,5 (F-809)", 30.5, 30.5, 70.6, 70.7, 20, 39.5, 0.16},
	} {
		kante, abseits := kastenDeckung(t, f.x0, f.y0, f.x1, f.y1, f.radius, f.breite)
		t.Logf("%s: Kante %.4f, abseits %.4f", f.name, kante, abseits)
		if kante > f.grenze || abseits > 0.01 {
			t.Errorf("%s: Kante %.4f (bewusste Grenze %.2f), abseits %.4f (≤ 0,01)", f.name, kante, f.grenze, abseits)
		}
	}
	for _, f := range []struct {
		name   string
		y, br  float64
		grenze float64
	}{
		{"Kehrtwende, Kante auf 0,4 px (F-814)", 100.4, 10, 0.45},
		{"Kehrtwende, Kante auf halbem Pixel (F-814)", 100, 9, 0.55},
	} {
		pts := []document.Punkt{document.Pt(60, f.y), document.Pt(200, f.y), document.Pt(120, f.y)}
		ls := neuerLinienStrich(pts, false, f.br/2)
		var falten []document.Punkt
		for k, p := range ls.pts {
			if ls.gefaltet[k] {
				falten = append(falten, p)
			}
		}
		if len(falten) == 0 {
			t.Fatalf("%s: keine Faltung erkannt", f.name)
		}
		nah := func(p document.Punkt) bool {
			for _, q := range falten {
				if p.Minus(q).Laenge() < f.br/2+2 {
					return true
				}
			}
			return false
		}
		a := linienAbweichung(strichBild(t, pts, f.br), pts, f.br/2, nah)
		t.Logf("%s: Kante %.4f, abseits %.4f", f.name, a.kante, a.innen)
		if a.kante > f.grenze || a.innen > 0.01 {
			t.Errorf("%s: Kante %.4f (bewusste Grenze %.2f), abseits %.4f (≤ 0,01)", f.name, a.kante, f.grenze, a.innen)
		}
	}
}
