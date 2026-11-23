// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"encoding/binary"
	"math"
	"os"
	"testing"

	"golang.org/x/image/font/sfnt"
)

// huelleSchrift lädt die Testschrift und setzt im head unitsPerEm und yMin neu (Vorlage Verify V23-4).
func huelleSchrift(t *testing.T, em uint16, yMin int16) *Schrift {
	t.Helper()
	b, err := os.ReadFile("../../testdata/fonts/schnapp-testkern.ttf")
	if err != nil {
		t.Fatal(err)
	}
	n := int(binary.BigEndian.Uint16(b[4:]))
	head := -1
	for i := 0; i < n; i++ {
		r := b[12+16*i:]
		if string(r[:4]) == "head" {
			head = int(binary.BigEndian.Uint32(r[8:]))
		}
	}
	if head < 0 {
		t.Fatal("kein head")
	}
	binary.BigEndian.PutUint16(b[head+18:], em)
	binary.BigEndian.PutUint16(b[head+38:], uint16(yMin))
	f, err := sfnt.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return &Schrift{Font: f}
}

// Verify V23-4: Auch bei großem unitsPerEm läuft die Rechnung der Hülle nicht über.
func TestHuelleOhneUeberlauf(t *testing.T) {
	for _, f := range []struct {
		em   uint16
		yMin int16
	}{{1000, -2200}, {2048, -4506}, {4096, -9011}, {8192, -18022}, {16384, -32000}} {
		soll := float64(-f.yMin)/float64(f.em) + 0.05
		_, unten, _ := huelleSchrift(t, f.em, f.yMin).Huelle()
		if math.Abs(unten-soll) > 0.01 {
			t.Errorf("unitsPerEm %d, yMin %d: unten %.3f em, Soll %.3f em", f.em, f.yMin, unten, soll)
		}
	}
}
