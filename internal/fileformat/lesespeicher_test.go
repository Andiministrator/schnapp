// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/png"
	"math/rand"
	"testing"
)

// F-64: Speicherbedarf beim Lesen eines fremden PNG. Das Bild ist verrauscht, damit die Datei kaum schrumpft und
// die Kopien der Datei den Bedarf bestimmen (nicht das dekodierte Bild).
func rauschPNG(t testing.TB, b, h int) []byte {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	rand.New(rand.NewSource(1)).Read(bild.Pix)
	for i := 3; i < len(bild.Pix); i += 4 {
		bild.Pix[i] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, bild); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func BenchmarkLeseFremdPNG(b *testing.B) {
	daten := rauschPNG(b, 1500, 1000)
	b.SetBytes(int64(len(daten)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := LeseArbeitsdatei(bytes.NewReader(daten)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeseBild(b *testing.B) {
	daten := rauschPNG(b, 1500, 1000)
	b.SetBytes(int64(len(daten)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := LeseBild(bytes.NewReader(daten)); err != nil {
			b.Fatal(err)
		}
	}
}

// Das Lesen eines fremden PNG legt neben dem dekodierten Bild höchstens die Blöcke der Datei (und ihre
// Zwischenkopie in leseDaten) an, nicht noch eine zusammengesetzte Kopie der ganzen Datei (F-64).
func TestLeseFremdPNGOhneZusatzkopie(t *testing.T) {
	daten := rauschPNG(t, 1500, 1000)
	bild := uint64(1500 * 1000 * 4) // dekodiertes Bild
	allocs := func(f func()) uint64 {
		var a, b runtimeStats
		a.lies()
		f()
		b.lies()
		return b.gesamt - a.gesamt
	}
	got := allocs(func() {
		if _, err := LeseBild(bytes.NewReader(daten)); err != nil {
			t.Fatal(err)
		}
	})
	// Obergrenze: Bild + Blöcke (1 Kopie, leseDaten legt bis zu 2 an) + Spielraum; mit der Zusammenbau-Kopie
	// (weitere ganze Datei samt Wachstumsstufen) wäre sie überschritten.
	grenze := bild + 2*uint64(len(daten)) + 1<<20
	if got > grenze {
		t.Errorf("LeseBild legte %d Bytes an, Grenze %d (Datei %d)", got, grenze, len(daten))
	}
}
