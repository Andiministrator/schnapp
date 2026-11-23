// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"math"
	"testing"
)

// TestHuelle (F-576): Go bleibt bei den Mindestwerten 1,5 / 1 / 1,25; die Testschrift mit 1,3 em Unterlänge verlangt
// mehr als 1 em nach unten (head-Rahmen 1,3 em, dazu 0,05 em Rand).
func TestHuelle(t *testing.T) {
	for _, s := range []Schnitt{Normal, Fett, Kursiv, FettKursiv} {
		o, u, l := Laden(Go, s).Huelle()
		if o != HuelleOben || u != HuelleUnten || l != HuelleSeitlich {
			t.Errorf("Go, Schnitt %v: %v / %v / %v, Soll %v / %v / %v", s, o, u, l, HuelleOben, HuelleUnten, HuelleSeitlich)
		}
		o, u, l = Laden(GoMono, s).Huelle()
		if o != HuelleOben || u != HuelleUnten || l != HuelleSeitlich {
			t.Errorf("Go Mono, Schnitt %v: %v / %v / %v", s, o, u, l)
		}
	}

	alt := ordner
	t.Cleanup(func() { ordner = alt })
	SetzeOrdner([]string{"../../testdata/fonts"})
	Suche()
	t.Cleanup(func() {
		mu.Lock()
		system, gesucht = nil, false
		mu.Unlock()
	})
	s := Laden("Schnapp Testkern", Normal)
	if s.Ersatz {
		t.Fatalf("Testschrift nicht gefunden: %v", Familien())
	}
	o, u, l := s.Huelle()
	if math.Abs(u-1.35) > 1e-9 || o != HuelleOben || l != HuelleSeitlich {
		t.Errorf("Testschrift: %v / %v / %v, Soll %v / 1,35 / %v", o, u, l, HuelleOben, HuelleSeitlich)
	}
}
