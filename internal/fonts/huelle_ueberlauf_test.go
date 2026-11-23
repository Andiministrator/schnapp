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

// rahmenSchrift lädt die Testschrift und setzt im head unitsPerEm und den Rahmen aller Glyphen neu (F-593).
func rahmenSchrift(t *testing.T, em uint16, xMin, yMin, xMax, yMax int16) *Schrift {
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
	for i, v := range []int16{xMin, yMin, xMax, yMax} {
		binary.BigEndian.PutUint16(b[head+36+2*i:], uint16(v))
	}
	f, err := sfnt.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return &Schrift{Font: f}
}

// F-593: Die Hülle folgt dem head-Rahmen in allen drei Richtungen (oben, links, rechts), nie unter die Mindestwerte,
// und HuelleMax deckelt jede Richtung.
func TestHuelleRichtungenUndDeckel(t *testing.T) {
	nah := func(name string, ist, soll float64) {
		t.Helper()
		if math.Abs(ist-soll) > 0.01 {
			t.Errorf("%s: %.3f em, Soll %.3f em", name, ist, soll)
		}
	}
	// Rahmen von 3 em nach oben; sonst innerhalb der Mindestwerte.
	o, u, l := rahmenSchrift(t, 1000, 0, 0, 500, 3000).Huelle()
	nah("oben", o, 3.05)
	nah("unten (Mindestwert)", u, HuelleUnten)
	nah("seitlich (Mindestwert)", l, HuelleSeitlich)
	// 2,5 em rechts des Ursprungs.
	o, u, l = rahmenSchrift(t, 1000, 0, 0, 2500, 500).Huelle()
	nah("rechts", l, 2.55)
	nah("oben (Mindestwert)", o, HuelleOben)
	nah("unten (Mindestwert)", u, HuelleUnten)
	// 3 em links des Ursprungs.
	_, _, l = rahmenSchrift(t, 1000, -3000, 0, 500, 500).Huelle()
	nah("links", l, 3.05)
	// Kaputter Rahmen von mehr als 30 em: auf HuelleMax gedeckelt, in jeder Richtung.
	o, u, l = rahmenSchrift(t, 1000, -32000, -32000, 32000, 32000).Huelle()
	nah("oben gedeckelt", o, HuelleMax)
	nah("unten gedeckelt", u, HuelleMax)
	nah("seitlich gedeckelt", l, HuelleMax)
	_, _, l = rahmenSchrift(t, 1000, 0, 0, 32000, 500).Huelle()
	nah("rechts gedeckelt", l, HuelleMax)
}
