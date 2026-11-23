// SPDX-License-Identifier: GPL-3.0-or-later

// Package fonttest baut kaputte und besondere Schriftdateien für Tests aus den eingebauten Go-Schriften (F-577): eine
// Schrift mit unlesbaren Umrissen eines Zeichens, eine mit Farb-Bitmap-Glyphen und eine Sammlung (.ttc). Nur Tests
// binden das Paket ein.
package fonttest

import (
	"encoding/binary"
	"sort"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// tabellen liefert die Offsets der Tabellen einer Einzelschrift roh.
func tabellen(b []byte) map[string]int {
	t := map[string]int{}
	for i := 0; i < int(binary.BigEndian.Uint16(b[4:])); i++ {
		r := b[12+16*i:]
		t[string(r[:4])] = int(binary.BigEndian.Uint32(r[8:]))
	}
	return t
}

// DefektRoh sind die Bytes von Go Regular, bei der die Umrisse des „A“ unlesbar sind: Der Vorschub steht in hmtx,
// aber LoadGlyph und damit GlyphBounds scheitern (numberOfContours 32767 mit zu kurzen Daten). Alle anderen Glyphen
// sind heil.
func DefektRoh(tb testing.TB) []byte {
	tb.Helper()
	b := append([]byte(nil), goregular.TTF...)
	ok, err := sfnt.Parse(b)
	if err != nil {
		tb.Fatal(err)
	}
	var puf sfnt.Buffer
	gi, err := ok.GlyphIndex(&puf, 'A')
	if err != nil || gi == 0 {
		tb.Fatalf("kein A: %v", err)
	}
	t := tabellen(b)
	loca, glyf, head := t["loca"], t["glyf"], t["head"]
	var von int
	if binary.BigEndian.Uint16(b[head+50:]) == 0 { // kurze Offsets: Wort · 2
		von = int(binary.BigEndian.Uint16(b[loca+2*int(gi):])) * 2
	} else {
		von = int(binary.BigEndian.Uint32(b[loca+4*int(gi):]))
	}
	binary.BigEndian.PutUint16(b[glyf+von:], 0x7fff) // numberOfContours
	f, err := sfnt.Parse(b)
	if err != nil {
		tb.Fatal(err)
	}
	if _, _, err := f.GlyphBounds(&puf, gi, fixed.I(16), font.HintingNone); err == nil {
		tb.Fatal("Vorbedingung: GlyphBounds des A scheitert nicht")
	}
	return b
}

// Defekt ist DefektRoh geparst.
func Defekt(tb testing.TB) *sfnt.Font {
	tb.Helper()
	f, err := sfnt.Parse(DefektRoh(tb))
	if err != nil {
		tb.Fatal(err)
	}
	return f
}

// FarbBitmapRoh macht aus der Schrift quelle (Bytes einer Einzelschrift) eine Farb-Bitmap-Schrift: Die Tabelle „loca“ heißt „CBLC“, sfnt sieht
// dann nur leere Glyphen und meldet ErrColoredGlyph aus LoadGlyph (wie Emoji-Schriften mit CBDT).
func FarbBitmapRoh(tb testing.TB, quelle []byte) []byte {
	tb.Helper()
	b := append([]byte(nil), quelle...)
	n := int(binary.BigEndian.Uint16(b[4:]))
	ok := false
	var recs [][]byte
	for i := 0; i < n; i++ {
		r := append([]byte(nil), b[12+16*i:12+16*i+16]...)
		if string(r[:4]) == "loca" {
			copy(r[:4], "CBLC")
			ok = true
		}
		recs = append(recs, r)
	}
	if !ok {
		tb.Fatal("keine loca-Tabelle")
	}
	// Das Verzeichnis muss nach Tag sortiert sein.
	sort.Slice(recs, func(i, j int) bool { return string(recs[i][:4]) < string(recs[j][:4]) })
	for i, r := range recs {
		copy(b[12+16*i:], r)
	}
	f, err := sfnt.Parse(b)
	if err != nil {
		tb.Fatal(err)
	}
	var puf sfnt.Buffer
	if _, err := f.LoadGlyph(&puf, 1, fixed.I(16), nil); err == nil {
		tb.Fatal("Vorbedingung: LoadGlyph meldet keine Farb-Glyphe")
	}
	return b
}

// Sammlung baut aus Einzelschriften eine TrueType-Sammlung (.ttc): Kopf „ttcf“, Offsets der Schriften, die Tabellenoffsets
// jeder Schrift um ihre Lage in der Sammlung verschoben (in einer Sammlung zählen sie ab dem Dateianfang).
func Sammlung(tb testing.TB, roh ...[]byte) []byte {
	tb.Helper()
	kopf := 12 + 4*len(roh)
	out := make([]byte, kopf)
	copy(out, "ttcf")
	binary.BigEndian.PutUint32(out[4:], 0x00010000)
	binary.BigEndian.PutUint32(out[8:], uint32(len(roh)))
	for i, r := range roh {
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		basis := len(out)
		binary.BigEndian.PutUint32(out[12+4*i:], uint32(basis))
		b := append([]byte(nil), r...)
		for k := 0; k < int(binary.BigEndian.Uint16(b[4:])); k++ {
			e := b[12+16*k:]
			binary.BigEndian.PutUint32(e[8:], binary.BigEndian.Uint32(e[8:])+uint32(basis))
		}
		out = append(out, b...)
	}
	if _, err := sfnt.ParseCollection(out); err != nil {
		tb.Fatalf("Sammlung nicht lesbar: %v", err)
	}
	return out
}
