// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"encoding/binary"
	"image"
	"image/color"
	"path/filepath"
	"runtime"
	"testing"
)

// F-334: Öffnen eines gedrehten JPEG (4:2:0, Ausrichtung 6) fordert neben der Quelle (YCbCr, 1,5 Byte je Pixel) nur
// eine Vollbild-Kopie an (das gedrehte NRGBA, 4 Byte je Pixel), die das Dokument ohne weitere Kopie übernimmt. Früher
// kopierte document.Neu sie noch einmal (zusammen 9,5 Byte je Pixel). Erlaubt sind 6,5 Byte je Pixel: Quelle,
// Ergebnis und Spielraum für Decoder und Dateipuffer.
func TestOeffneGedrehtesJPEGEineKopie(t *testing.T) {
	const w, h = 1024, 768
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: 255})
		}
	}
	pfad := filepath.Join(t.TempDir(), "gedreht.jpg")
	schreibeDatei(t, pfad, mitSegment(kodiere(t, b), exifSegment(binary.LittleEndian, false, 6)))

	var a, z runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&a)
	g, err := Oeffne(pfad)
	runtime.ReadMemStats(&z)
	if err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(g)
	if s := g.Dokument.Groesse(); s != image.Pt(h, w) {
		t.Fatalf("Größe %v, erwartet %dx%d (gedreht)", s, h, w)
	}
	erlaubt := uint64(w * h * 13 / 2)
	if d := z.TotalAlloc - a.TotalAlloc; d > erlaubt {
		t.Errorf("%d Bytes angefordert (%.2f je Pixel), erlaubt höchstens %d (Quelle und eine Kopie)",
			d, float64(d)/float64(w*h), erlaubt)
	}
}
