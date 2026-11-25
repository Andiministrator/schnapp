// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// F-789, Mutant B8: Ist der halbe Strich mindestens so breit wie der Radius, liegen die inneren Ecken einer Rundung auf
// einer Stelle; ihre Strecken bilden einen Fächer (ein konvexes Vieleck), keine einzelnen Dreiecke, die in einem Punkt
// zusammenlaufen (dort verlöre Gio Deckung). Ein Kasten 120×80 mit Radius 20 hat dann je Rundung einen Fächer, je Seite
// ein Viereck und an jedem Übergang Seite–Rundung ein Viereck. Jeder Teil ist konvex und läuft gleich herum. Endet die
// Gehrung weniger als mittelpunktFang vor dem Mittelpunkt (Strich 39,6 und 39,9), gilt dasselbe (F-798).
func TestKastenStrichFaecherJeRundung(t *testing.T) {
	r := R(Pt(10, 10), Pt(130, 90))
	for _, breite := range []float64{39.6, 39.9, 40, 41, 60} { // 39,6 und 39,9: F-798 (Fang vor dem Mittelpunkt)
		ks := NeuerKastenStrich(r, 20, breite/2)
		faecher := 0
		for i, teil := range ks.Teile {
			if len(teil) > 4 {
				faecher++
			}
			for k := range teil {
				a, b, c := teil[k], teil[(k+1)%len(teil)], teil[(k+2)%len(teil)]
				kreuz := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
				if kreuz*ks.Umlauf < -1e-9 {
					t.Errorf("Strich %v: Teil %d nicht konvex oder falsch herum an Ecke %d", breite, i, k)
				}
			}
		}
		if faecher != 4 || len(ks.Teile) > 16 {
			t.Errorf("Strich %v: %d Teile, davon %d Fächer; erwartet je Rundung einen Fächer", breite, len(ks.Teile), faecher)
		}
	}
}

// F-804 (ADR-0023 Nachtrag 21, Zusatz): Im Band des Fangs (die Gehrung im Bogen endet weniger als mittelpunktFang vor
// dem Mittelpunkt) bleibt die innere Kante jeder geraden Seite genau hw vom Umriss; vorher sprangen auch die
// Tangentenpunkte auf den Mittelpunkt, und die Kante rückte um Radius − hw ins Loch. Zugleich hat kein Teil eine Kante,
// die kürzer als mittelpunktFang ist (keine Splitter am Mittelpunkt, F-798), auch nicht bei Pille und Kreis, deren
// Rundungen einen Mittelpunkt teilen. Außerhalb des Bands greift der Fang nicht: Bei Strich 39,3 (die Gehrung endet
// 0,31 px vor dem Mittelpunkt) hat jede Strecke der Rundung ihr eigenes Viereck.
func TestKastenStrichFangHaeltSeitenkante(t *testing.T) {
	for _, fall := range []struct {
		r      Rechteck
		breite float64
	}{
		{R(Pt(30, 30), Pt(150, 110)), 39.42},
		{R(Pt(30, 30), Pt(150, 110)), 39.5},
		{R(Pt(30, 30), Pt(150, 110)), 39.6},
		{R(Pt(30, 30), Pt(150, 110)), 39.9},
		{R(Pt(30.3, 30.7), Pt(150.3, 110.7)), 39.5},
		{R(Pt(30.3, 30.7), Pt(150.3, 70.7)), 39.42}, // Pille
		{R(Pt(30.3, 30.7), Pt(150.3, 70.7)), 39.6},
		{R(Pt(30.3, 30.7), Pt(150.3, 70.7)), 39.9},
		{R(Pt(30.5, 30.5), Pt(70.5, 70.5)), 39.5}, // Kreis
	} {
		hw := fall.breite / 2
		ks := NeuerKastenStrich(fall.r, 20, hw)
		seiten := 0
		for i, a := range ks.Umriss {
			b := ks.Umriss[(i+1)%len(ks.Umriss)]
			if a.Minus(b).Laenge() < 10 || (a.X != b.X && a.Y != b.Y) {
				continue // keine gerade Seite
			}
			seiten++
			teil := ks.Teile[ks.TeilVon[i]]
			if len(teil) != 4 {
				t.Errorf("Strich %v, Seite %v–%v: Teil mit %d Ecken statt eines Vierecks", fall.breite, a, b, len(teil))
				continue
			}
			for _, e := range teil[2:] {
				if d := abstandZurGeraden(e, a, b); math.Abs(d-hw) > 1e-9 {
					t.Errorf("Strich %v, Seite %v–%v: innere Ecke %v liegt %.4f statt %.4f vom Umriss", fall.breite, a, b, e,
						d, hw)
				}
			}
		}
		if seiten == 0 && fall.r.Breite() != fall.r.Hoehe() {
			t.Errorf("Strich %v: keine gerade Seite gefunden", fall.breite)
		}
		kuerzeste := math.Inf(1)
		for _, teil := range ks.Teile {
			for k, e := range teil {
				if l := teil[(k+1)%len(teil)].Minus(e).Laenge(); l > 0 {
					kuerzeste = min(kuerzeste, l)
				}
			}
		}
		if kuerzeste < mittelpunktFang-1e-9 {
			t.Errorf("Strich %v, %v: kürzeste Kante eines Teils %.4f, unter mittelpunktFang (Splitter)", fall.breite, fall.r,
				kuerzeste)
		}
	}
	// Obergrenze des Fangs: außerhalb des Bands keine Fächer.
	if ks := NeuerKastenStrich(R(Pt(30, 30), Pt(150, 110)), 20, 39.3/2); len(ks.Teile) != 52 {
		t.Errorf("Strich 39,3: %d Teile statt 52; der Fang greift über mittelpunktFang hinaus", len(ks.Teile))
	}
}

// abstandZurGeraden ist der Abstand von p zur Geraden durch a und b.
func abstandZurGeraden(p, a, b Punkt) float64 {
	d := b.Minus(a)
	return math.Abs(d.X*(p.Y-a.Y)-d.Y*(p.X-a.X)) / d.Laenge()
}
