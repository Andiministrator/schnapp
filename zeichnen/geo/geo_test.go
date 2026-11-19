// SPDX-License-Identifier: GPL-3.0-or-later

package geo

import (
	"math"
	"testing"
)

func TestGrundwerte(t *testing.T) {
	if Zahl(-0.001) != "0" || Zahl(12.5) != "12.5" || Zahl(100) != "100" || Zahl(1.005) != "1" {
		t.Fatalf("Zahl: %s %s %s %s", Zahl(-0.001), Zahl(12.5), Zahl(100), Zahl(1.005))
	}
	if Zahl1(-0.01) != "-0" || Zahl1(3.25) != "3.2" {
		t.Fatalf("Zahl1: %s %s", Zahl1(-0.01), Zahl1(3.25))
	}
	if Mod(-190, 360) != 170 || Mod(190, -360) != -170 {
		t.Fatalf("Mod")
	}
	if Hypot(3, 4) != 5 {
		t.Fatalf("Hypot")
	}
}

func TestRepr(t *testing.T) {
	for _, f := range []struct {
		v float64
		s string
	}{{1, "1.0"}, {0.1, "0.1"}, {math.Copysign(0, -1), "-0.0"}, {1e16, "1e+16"}, {1e15, "1000000000000000.0"}, {0.0001, "0.0001"},
		{0.00001, "1e-05"}, {math.Inf(1), "inf"}, {math.Inf(-1), "-inf"}, {math.NaN(), "nan"}, {2.5e-7, "2.5e-07"}} {
		if Repr(f.v) != f.s {
			t.Errorf("Repr(%v) = %s, erwartet %s", f.v, Repr(f.v), f.s)
		}
	}
}
