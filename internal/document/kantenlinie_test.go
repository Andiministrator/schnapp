// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// kantenbandVorher ist Kantenband, wie es bis F-418 je Schicht die Mittellinie neu rechnete — die Vorlage, an der
// Kantenlinie bitgleich bleiben muss.
func kantenbandVorher(k Kante, n Naht, laenge, faktor int, von, bis float64) []Punkt {
	f := max(faktor, 1)
	s := k.schritt()
	anz := int(math.Ceil(float64(max(laenge, 1)) / (s * float64(f))))
	vonH, bisH := hundertstel(von)*f, hundertstel(bis)*f
	vorn := make([]Punkt, 0, 2*(anz+1))
	hinten := make([]Punkt, anz+1)
	for i := 0; i <= anz; i++ {
		tb := float64(i) * s
		t := tb * float64(f)
		m := n.Lage*100 + hundertstel(k.mitte(tb))*f
		vorn = append(vorn, quer(n, t, float64(m+vonH)/100))
		hinten[anz-i] = quer(n, t, float64(m+bisH)/100)
	}
	return append(vorn, hinten...)
}

// F-418: Eine einmal gerechnete Kantenlinie liefert für jede Naht und Schicht genau die Zahlen des früheren
// Kantenbands (bitgleich, auch über einen wiederverwendeten Puffer hinweg), in allen Formen, Faktoren und Längen.
func TestKantenlinieWieVorher(t *testing.T) {
	naehte := []Naht{{Zeilen: true, Lage: 0}, {Zeilen: true, Lage: 1234}, {Zeilen: false, Lage: 7}, {Lage: -3}}
	for _, k := range []Kante{KanteAbriss, KanteWelle, KanteZacken} {
		for _, f := range []int{1, 2, 3, 7} {
			for _, laenge := range []int{0, 1, 17, 300, 1921} {
				l := NeueKantenlinie(k, laenge, f)
				var puffer []Punkt
				for _, n := range naehte {
					for _, s := range KantenSchichten() {
						soll := kantenbandVorher(k, n, laenge, f, s.Von, s.Bis)
						puffer = l.Band(puffer, n, s.Von, s.Bis)
						neu := Kantenband(k, n, laenge, f, s.Von, s.Bis)
						if len(puffer) != len(soll) || len(neu) != len(soll) {
							t.Fatalf("%v f%d l%d %v: Längen %d/%d, Soll %d", k, f, laenge, n, len(puffer), len(neu), len(soll))
						}
						for i := range soll {
							if puffer[i] != soll[i] || neu[i] != soll[i] {
								t.Fatalf("%v f%d l%d %v Schicht %v: Punkt %d %v/%v, Soll %v", k, f, laenge, n, s, i,
									puffer[i], neu[i], soll[i])
							}
						}
					}
				}
			}
		}
	}
}
