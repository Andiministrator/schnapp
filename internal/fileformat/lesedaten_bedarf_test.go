// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"testing"
	"testing/iotest"
)

// Z-5 (Verify Fable zu #115): leseDaten fordert bei einem wirklich vorhandenen großen Block höchstens das Doppelte
// der Länge an (Stücke plus Endpuffer) und behält genau die Länge, nicht die doppelte Kapazität eines
// verdoppelnden Puffers.
func TestLeseDatenBedarfBeiGrossemBlock(t *testing.T) {
	for _, mb := range []int{2, 24, 65} {
		n := mb<<20 + 1
		roh := make([]byte, n)
		for i := 0; i < n; i += 4099 {
			roh[i] = byte(i >> 8)
		}
		var got []byte
		var err error
		alloc := angefordert(func() { got, err = leseDaten(bytes.NewReader(roh), uint32(n)) })
		if err != nil || !bytes.Equal(got, roh) {
			t.Fatalf("%d MB: Fehler %v, gleich: %v", mb, err, bytes.Equal(got, roh))
		}
		if alloc > uint64(n)*21/10 {
			t.Errorf("%d MB: %d Bytes angefordert (%.2fx, Soll höchstens ~2x)", mb, alloc, float64(alloc)/float64(n))
		}
		if cap(got) != n {
			t.Errorf("%d MB: Kapazität %d, Soll genau %d", mb, cap(got), n)
		}
	}
}

// Z-5: Ein langsamer Strom und eine angekündigte Länge weit über dem Gelieferten kosten nur das Gelieferte (F-273).
func TestLeseDatenLangsamerStromReserviertNichts(t *testing.T) {
	roh := make([]byte, 3<<20)
	var err error
	alloc := angefordert(func() { _, err = leseDaten(iotest.OneByteReader(bytes.NewReader(roh)), 500<<20) })
	if err == nil {
		t.Error("kein Fehler bei zu kurzem Strom")
	}
	if alloc > 32<<20 {
		t.Errorf("%d MB angefordert (Soll nur etwa das Gelieferte)", alloc>>20)
	}
}
