// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"math"
	"testing"
)

// F-142: Festgelegtes Verhalten von ZuOriginal an den Schnittkanten, mit Ansichtswerten ein ulp vor und nach jeder
// Kante (nicht nur 0,5-Schritten). Auf der Kante: das Streifenende; ein ulp davor: der Streifenanfang (linksseitiger
// Grenzwert, nie ein Wert im Streifen); ein ulp danach: das Streifenende. Die frühere Gleitkomma-Rechnung
// (refZuOriginal) rundete einen Wert knapp vor der Kante auf die Kante und lieferte das Streifenende.
func TestZuOriginalAnDenKanten(t *testing.T) {
	d := neu(t, 300, 200)
	if err := d.SetzeZuschnitt(image.Rect(7, 3, 293, 197)); err != nil {
		t.Fatal(err)
	}
	st := []Streifen{{false, 20, 35}, {false, 60, 61}, {false, 100, 140}, {false, 200, 290},
		{true, 11, 12}, {true, 40, 77}, {true, 150, 160}}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	abweichungen := 0
	for _, zeilen := range []bool{false, true} {
		von := float64(d.Zuschnitt().Min.X)
		if zeilen {
			von = float64(d.Zuschnitt().Min.Y)
		}
		kum := 0
		for _, s := range d.Streifen() {
			if s.Zeilen != zeilen {
				continue
			}
			a, b := float64(s.Von), float64(s.Bis)
			kante := a - float64(kum) - von // Ansichtswert der Schnittkante
			um := func(v float64) float64 {
				if zeilen {
					return d.ZuOriginal(Pt(0, v)).Y
				}
				return d.ZuOriginal(Pt(v, 0)).X
			}
			vor, nach := math.Nextafter(kante, math.Inf(-1)), math.Nextafter(kante, math.Inf(1))
			if got := um(kante); got != b {
				t.Errorf("Streifen %v: ZuOriginal(Kante %v) = %v, Soll Streifenende %v", s, kante, got, b)
			}
			if got := um(vor); got > a || got < a-1e-9 {
				t.Errorf("Streifen %v: ZuOriginal(ein ulp vor %v) = %v, Soll Streifenanfang %v (nicht im Streifen)", s, kante, got, a)
			}
			if got := um(nach); got < b || got > b+1e-9 {
				t.Errorf("Streifen %v: ZuOriginal(ein ulp nach %v) = %v, Soll Streifenende %v", s, kante, got, b)
			}
			// Gegenstück zur alten Rechnung: Sie weicht genau an diesen Stellen ab (mindestens einmal muss das vorkommen,
			// sonst prüft der Test die Kante nicht).
			ref := refZuOriginal(d.Zuschnitt(), d.Streifen(), Pt(vor, vor))
			if (zeilen && ref.Y != um(vor)) || (!zeilen && ref.X != um(vor)) {
				abweichungen++
			}
			kum += s.Bis - s.Von
		}
	}
	if abweichungen == 0 {
		t.Error("die alte Rechnung weicht an keiner Kante ab: der Test trifft die Rundungsstelle nicht")
	}
	t.Logf("%d von 7 Kanten (ein ulp davor) weichen von der alten Rechnung ab", abweichungen)
}

// Der Nullwert Dokument{} ist nicht nutzbar (nur Neu liefert ein gültiges Dokument), Groesse darf aber nicht
// abstürzen: 0×0. (Die Umrechnungen ZuAnsicht/ZuOriginal des Nullwerts sind nicht Teil der Zusage.)
func TestNullwertDokumentGroesse(t *testing.T) {
	var d Dokument
	if g := d.Groesse(); g != image.Pt(0, 0) {
		t.Errorf("Groesse des Nullwerts %v, Soll 0×0", g)
	}
}
