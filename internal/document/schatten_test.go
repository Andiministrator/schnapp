// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image/color"
	"math"
	"testing"
)

// ADR-0021 Nachtrag 26 Punkt 2: Versatz entgegen dem Licht, Voreinstellung mal Faktor, Grenzen, Winkel normiert,
// unsichtbar ohne Deckung.
func TestSchattenWerte(t *testing.T) {
	v := Schatten{An: true, Winkel: 135, Laenge: 10 * math.Sqrt2}.Versatz()
	if math.Abs(v.X-10) > 1e-9 || math.Abs(v.Y-10) > 1e-9 {
		t.Errorf("135°: Versatz %v, Soll (10, 10) nach rechts unten", v)
	}
	if v := (Schatten{An: true, Winkel: 90, Laenge: 5}).Versatz(); math.Abs(v.X) > 1e-9 || math.Abs(v.Y-5) > 1e-9 {
		t.Errorf("90° (Licht von oben): %v", v)
	}
	if s := StandardSchatten(2); !s.An || s.Winkel != 135 || s.Laenge != 12 || s.Weich != 16 || s.Farbe != (color.NRGBA{A: 102}) {
		t.Errorf("Voreinstellung Faktor 2: %+v", s)
	}
	for w, soll := range map[float64]float64{-45: 315, 360: 0, 725: 5, 0: 0, -720: 0} {
		if got := NormWinkel(w); math.Abs(got-soll) > 1e-9 {
			t.Errorf("NormWinkel(%v) = %v, Soll %v", w, got, soll)
		}
	}
	if (Schatten{An: true, Farbe: color.NRGBA{R: 9}}).Sichtbar() || !(Schatten{An: true, Farbe: color.NRGBA{A: 1}}).Sichtbar() {
		t.Error("Sichtbar hängt nicht an der Deckung")
	}
	for _, s := range []Schatten{
		{An: true, Laenge: -1}, {An: true, Laenge: MaxSchattenLaenge + 1}, {An: true, Weich: -1},
		{An: true, Weich: MaxSchattenWeich + 1}, {An: true, Winkel: math.NaN()},
	} {
		if (Ebene{Schatten: s}).pruefe() == nil {
			t.Errorf("%+v gilt als gültig", s)
		}
	}
	if (Ebene{Schatten: Schatten{Laenge: -1}}).pruefe() != nil {
		t.Error("ausgeschalteter Schatten wird geprüft")
	}
	if r := (Schatten{An: true, Laenge: 4, Weich: 10, Farbe: color.NRGBA{A: 9}}).Rand(); r != 20 {
		t.Errorf("Rand %v", r)
	}
}
