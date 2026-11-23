// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// F-422: Vereinfacht lässt Stützpunkte weg, ohne die Linie um mehr als die Toleranz zu verschieben; Anfang und Ende
// bleiben, und die Linie wird beim Abriss merklich kürzer.
func TestKantenlinieVereinfacht(t *testing.T) {
	const tol = 0.4
	for _, k := range []Kante{KanteAbriss, KanteWelle, KanteZacken} {
		for _, f := range []int{1, 2, 3} {
			voll := NeueKantenlinie(k, 1920, f)
			v := voll.Vereinfacht(tol)
			if v.t[0] != voll.t[0] || v.t[len(v.t)-1] != voll.t[len(voll.t)-1] || v.f != voll.f {
				t.Fatalf("%v f=%d: Anfang oder Ende verändert", k, f)
			}
			// Jeder Stützpunkt der vollen Linie liegt höchstens tol neben dem Linienzug der vereinfachten.
			j := 0
			for i, x := range voll.t {
				for j+2 < len(v.t) && v.t[j+1] < x {
					j++
				}
				soll := float64(v.m[j]) + float64(v.m[j+1]-v.m[j])*(x-v.t[j])/(v.t[j+1]-v.t[j])
				if d := math.Abs(float64(voll.m[i]) - soll); d > tol*100+1e-6 {
					t.Fatalf("%v f=%d Punkt %d: Abweichung %.1f Hundertstel > %.0f", k, f, i, d, tol*100)
				}
			}
			t.Logf("%v f=%d: %d von %d Stützpunkten", k, f, len(v.t), len(voll.t))
			if k == KanteAbriss && f == 1 && len(v.t)*10 > len(voll.t)*6 {
				t.Errorf("Abriss f=%d: %d von %d Stützpunkten, Soll höchstens 60 Prozent", f, len(v.t), len(voll.t))
			}
		}
	}
	// Toleranz 0 lässt jeden Punkt stehen, der von der Verbindung seiner Nachbarn abweicht (hier: alle beim Abriss).
	if v := NeueKantenlinie(KanteAbriss, 200, 1).Vereinfacht(0); len(v.t) < 150 {
		t.Errorf("Toleranz 0: %d Stützpunkte, Soll fast alle", len(v.t))
	}
	// Eine Linie aus zwei Punkten bleibt.
	if v := NeueKantenlinie(KanteZacken, 1, 1).Vereinfacht(0.4); len(v.t) != 2 {
		t.Errorf("zwei Stützpunkte: %d", len(v.t))
	}
}

// F-422: KantenFlach malt dasselbe wie KantenSchichten (je Tiefe quer zur Mittellinie dieselbe zusammengesetzte Farbe
// über einem Untergrund, auf 1/255 genau), ohne Überlappung je Farbe, mit höchstens fünf verschiedenen Breiten.
func TestKantenFlachGleichKantenSchichten(t *testing.T) {
	komponiere := func(schichten []Schicht, tiefe float64, grund [3]float64) [3]float64 {
		c := grund
		for _, s := range schichten {
			if s.Von <= tiefe && tiefe < s.Bis {
				a := float64(s.Farbe.A) / 255
				for i, v := range [3]uint8{s.Farbe.R, s.Farbe.G, s.Farbe.B} {
					c[i] = float64(v)*a + c[i]*(1-a)
				}
			}
		}
		return c
	}
	flach := KantenFlach()
	breiten := map[int]bool{}
	for _, s := range flach {
		breiten[hundertstel(s.Bis)-hundertstel(s.Von)] = true
	}
	if len(breiten) > 5 {
		t.Errorf("%d verschiedene Breiten, Soll höchstens 5: %v", len(breiten), breiten)
	}
	for _, grund := range [][3]float64{{0, 0, 0}, {128, 128, 128}, {255, 255, 255}} {
		for tiefe := -12.0; tiefe < 12; tiefe += 0.05 {
			a, b := komponiere(KantenSchichten(), tiefe, grund), komponiere(flach, tiefe, grund)
			for i := range a {
				if math.Abs(a[i]-b[i]) > 1.01 { // Deckkraft auf 1/255 gerundet: höchstens eine Stufe
					t.Fatalf("Tiefe %.2f Untergrund %v: %v gegen %v", tiefe, grund, a, b)
				}
			}
		}
	}
	// Je Farbe überlappen sich die Stücke nicht.
	for i, a := range flach {
		for _, b := range flach[i+1:] {
			if mitDeckkraft(a.Farbe, 0) == mitDeckkraft(b.Farbe, 0) && hundertstel(a.Von) < hundertstel(b.Bis) && hundertstel(b.Von) < hundertstel(a.Bis) {
				t.Errorf("Stücke überlappen: %v und %v", a, b)
			}
		}
	}
	// Die Reihenfolge der Farben bleibt (Saum, Abdunklung, Schatten): spätere Farben liegen oben.
	var farben []Schicht
	for _, s := range flach {
		if len(farben) == 0 || mitDeckkraft(farben[len(farben)-1].Farbe, 0) != mitDeckkraft(s.Farbe, 0) {
			farben = append(farben, s)
		}
	}
	if len(farben) != 3 || farben[0].Farbe.R != 255 || farben[1].Farbe.R != 0 || farben[2].Farbe.R != 0x10 {
		t.Errorf("Farbfolge: %v", farben)
	}
	// Der Schatten liegt ganz im Spalt, Saum und Abdunklung nicht (ImSpalt wie bei den Schichten).
	for _, s := range flach {
		if s.ImSpalt() != (s.Farbe.R == 0x10) {
			t.Errorf("ImSpalt %v für %v", s.ImSpalt(), s)
		}
	}
}
