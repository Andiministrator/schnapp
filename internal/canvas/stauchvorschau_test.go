// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"
)

// F-240: Die Vorschau des Stauch-Bands folgt derselben Mindestrest-Regel wie das Ergebnis beim Loslassen. Reicht das
// Band mit einem Rest unter der halben Bandhöhe (11, ADR-0023 Nachtrag 6) bis an den Rand, zeigt die Vorschau es dort;
// sonst beginnt sie an der Naht. Verglichen wird, was die Vorschau zeigt, mit dem, was nach dem Loslassen im Dokument
// steht.
func TestStauchenVorschauGleichErgebnis(t *testing.T) {
	faelle := []struct {
		name      string
		von, bis  float64 // Zug in Bildzeilen
		vorVon    float64 // erwartete Vorschau (Bildzeilen)
		vorBis    float64
		naht      bool            // Ergebnis ist eine Naht (sonst Zuschnitt)
		zuschnitt image.Rectangle // erwarteter Zuschnitt, wenn keine Naht
	}{
		{"oben Rest 10", 10, 60, 0, 60, false, image.Rect(0, 60, 200, 100)},
		{"unten Rest 10", 40, 90, 40, 100, false, image.Rect(0, 0, 200, 40)},
		{"oben Rest 11", 11, 40, 11, 40, true, image.Rect(0, 0, 200, 100)},
	}
	for _, c := range faelle {
		s := verlaufSitzung(t)
		s.SetzeWerkzeug(WerkzeugStauchen)
		s.Druecken(f(5, c.von), TastePrimaer)
		s.Ziehen(f(6, (c.von+c.bis)/2))
		s.Ziehen(f(6, c.bis))
		a, b, ok := s.Rahmen()
		if !ok {
			t.Fatalf("%s: keine Vorschau", c.name)
		}
		if want := fv(s, 0, c.vorVon); a.Y != want.Y {
			t.Errorf("%s: Vorschau beginnt bei Fensterzeile %v, Soll %v (Bildzeile %v)", c.name, a.Y, want.Y, c.vorVon)
		}
		if want := fv(s, 0, c.vorBis); b.Y != want.Y {
			t.Errorf("%s: Vorschau endet bei Fensterzeile %v, Soll %v (Bildzeile %v)", c.name, b.Y, want.Y, c.vorBis)
		}
		s.Loslassen(f(6, c.bis))
		st := s.D.Streifen()
		if c.naht {
			if len(st) != 1 || float64(st[0].Von) != c.vorVon || float64(st[0].Bis) != c.vorBis {
				t.Errorf("%s: Ergebnis %v, Soll Naht %v bis %v wie die Vorschau", c.name, st, c.vorVon, c.vorBis)
			}
		} else if len(st) != 0 || s.D.Zuschnitt() != c.zuschnitt {
			t.Errorf("%s: Ergebnis Streifen %v, Zuschnitt %v, Soll %v wie die Vorschau", c.name, st, s.D.Zuschnitt(), c.zuschnitt)
		}
	}
}
