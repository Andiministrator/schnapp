// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// abstandZumUmriss ist der Abstand von p zum geschlossenen Vieleck pts (Strecken, nicht Fläche).
func abstandZumUmriss(p document.Punkt, pts []document.Punkt) float64 {
	best := math.Inf(1)
	for i, a := range pts {
		b := pts[(i+1)%len(pts)]
		d := b.Minus(a)
		l := d.X*d.X + d.Y*d.Y
		t := 0.0
		if l > 0 {
			t = min(max(((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l, 0), 1)
		}
		best = min(best, p.Minus(a.Plus(d.Mal(t))).Laenge())
	}
	return best
}

// F-788 (ADR-0023 Nachtrag 21): Der Strich eines runden Kastens deckt jedes Pixel so, wie die Fläche aller Punkte
// höchstens eine halbe Strichbreite vom Umriss (document.RundesRechteck) es analytisch deckt: geschätzt mit 32×32
// Stichproben je Pixel (Fehler höchstens rund 1/32 an einer Kante). Vorher füllte der Renderer je Strecke ein Rechteck und
// je Gelenk einen Kreis in einer Füllung; deren Kantenglättung addierte sich (Abweichung bis 0,45 bis 0,82 in diesen
// Fällen). Die Gehrung der Teile (Knick 7,5°) reicht höchstens 0,22 % der halben Breite über den runden Strich hinaus.
// Geprüft werden nur Pixel nahe einer Kante; die übrigen sind ganz oder gar nicht gedeckt. Fälle: Strich 24 (Befund),
// Radius 0,5 (F-793), halber Strich gleich Radius (F-792) und breiter als der Radius (F-783).
func TestKastenStrichDecktWieAnalytisch(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for _, fall := range []struct {
		x0, y0, x1, y1 float64
		radius, breite float64
	}{
		{20.3, 20.7, 140.3, 100.7, 20, 24},
		{30, 30, 110, 90, 12, 24},
		{25.5, 25.25, 65.5, 65.25, 0.5, 30},
		{30, 30, 150, 110, 20, 40},
		{30, 30, 70, 70, 20, 60},
		// F-804: Band knapp unter 2·Radius, in dem die inneren Ecken der Rundung auf den Fangpunkt springen
		// (mittelpunktFang, F-798); die innere Kante der geraden Seiten bleibt dabei, wo sie ist.
		{30, 30, 150, 110, 20, 39.42},
		{30, 30, 150, 110, 20, 39.5},
		{30, 30, 150, 110, 20, 39.6},
		{30, 30, 150, 110, 20, 39.9},
		{30.3, 30.7, 150.3, 110.7, 20, 39.42},
		{30.3, 30.7, 150.3, 110.7, 20, 39.5},
		{30.5, 30.5, 150.5, 110.5, 20, 39.45},
		{40, 40, 60, 60, 3, 5.5},
		{40, 40, 60, 60, 3, 5.6},
		{40.5, 40.5, 60.5, 60.5, 3, 5.5},
		{40, 40, 60, 60, 1, 1.5},
		{40.5, 40.5, 60.5, 60.5, 1, 1.6},
		// Pille und Kreis: Die Mittelpunkte benachbarter Rundungen fallen zusammen, ihre Fangpunkte rücken auseinander.
		{30.3, 30.7, 150.3, 70.7, 20, 39.5},
		{30.3, 30.7, 150.3, 70.7, 20, 39.6},
		{30.7, 30.3, 70.7, 140.3, 20, 39.5}, // hohe Pille: Schmal-Loch-Regel in X (Verify 5, V5-4)
		{30.7, 30.3, 70.7, 140.3, 20, 39.6},
		{30.7, 30.3, 70.7, 140.3, 20, 39.8}, // Loch 0,2 px breit: Fangpunkte auf seine Mitte
		{30.7, 30.3, 70.7, 140.3, 20, 39.9},
		{30.3, 30.7, 140.3, 70.7, 20, 39.8}, // dasselbe quer
		{30.5, 30.5, 70.5, 70.5, 20, 39.9},
	} {
		name := fmt.Sprintf("Kasten (%v,%v)–(%v,%v) Radius %v Strich %v", fall.x0, fall.y0, fall.x1, fall.y1, fall.radius, fall.breite)
		r := document.R(document.Pt(fall.x0, fall.y0), document.Pt(fall.x1, fall.y1))
		d := weissesDokument(t, 200, 150)
		hinzu(t, d, document.Kasten{Bereich: r, Eckradius: fall.radius, Strich: document.Strich{Farbe: schwarz, Breite: fall.breite}})
		b := bild(t, d)
		umriss := document.RundesRechteck(r, fall.radius)
		hw := fall.breite / 2
		const n = 32
		schlimmst, geprueft, abseits := 0.0, 0, 0.0
		var wo, woAb [2]int
		for y := 0; y < 150; y++ {
			for x := 0; x < 200; x++ {
				m := abstandZumUmriss(document.Pt(float64(x)+0.5, float64(y)+0.5), umriss)
				ist := 1 - float64(b.NRGBAAt(x, y).R)/255 // schwarz auf weiß
				if math.Abs(m-hw) > 1 {                   // ganz drinnen oder draußen: voll bzw. leer
					soll := 0.0
					if m < hw {
						soll = 1
					}
					if f := math.Abs(ist - soll); f > abseits {
						abseits, woAb = f, [2]int{x, y}
					}
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
				soll := float64(treffer) / (n * n)
				geprueft++
				if f := math.Abs(ist - soll); f > schlimmst {
					schlimmst, wo = f, [2]int{x, y}
				}
			}
		}
		if geprueft < 100 {
			t.Fatalf("%s: nur %d Kantenpixel", name, geprueft)
		}
		t.Logf("%s: höchstens %.4f Abweichung über %d Kantenpixel (Pixel %v), abseits %.4f", name, schlimmst, geprueft, wo,
			abseits)
		if abseits > 0.01 {
			t.Errorf("%s: abseits der Kante weicht die Deckung bis %.3f ab (Pixel %v): Loch oder Deckung daneben", name,
				abseits, woAb)
		}
		// Zulässig: Stichprobenfehler (1/32), die Gehrung über den runden Strich hinaus (hw·(1/cos 3,75° − 1)) und Rundung.
		grenze := 1.0/n + hw*(1/math.Cos(3.75*math.Pi/180)-1) + 0.01
		if schlimmst > grenze {
			t.Errorf("%s: Deckung weicht bis %.3f von der analytischen ab, zulässig %.3f (Pixel %v, %d Kantenpixel)", name,
				schlimmst, grenze, wo, geprueft)
		}
	}
}
