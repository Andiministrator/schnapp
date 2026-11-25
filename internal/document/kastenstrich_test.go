// SPDX-License-Identifier: GPL-3.0-or-later

package document

import "testing"

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
