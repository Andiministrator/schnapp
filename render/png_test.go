// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// Karte #39: Die Zwischenablage bekommt ein reines PNG des gerenderten Bildes, ohne Bearbeitungs- und Metadatenblöcke.
func TestPNGOhneBearbeitungsdaten(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 6), G: uint8(y * 8), B: 90, A: 255})
		}
	}
	d, err := document.Neu(src)
	if err != nil {
		t.Fatal(err)
	}
	daten, err := PNG(d)
	if err != nil {
		t.Fatal(err)
	}
	var bloecke []string
	rest := daten[8:]
	for len(rest) >= 12 {
		n := int(binary.BigEndian.Uint32(rest[:4]))
		bloecke = append(bloecke, string(rest[4:8]))
		rest = rest[12+n:]
	}
	erlaubt := map[string]bool{"IHDR": true, "PLTE": true, "IDAT": true, "IEND": true}
	for _, b := range bloecke {
		if !erlaubt[b] {
			t.Errorf("unerwarteter Block %q in %v", b, bloecke)
		}
	}
	if len(bloecke) < 3 || bloecke[0] != "IHDR" || bloecke[len(bloecke)-1] != "IEND" {
		t.Errorf("Blockfolge %v", bloecke)
	}
	got, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	soll, _ := Bild(d)
	if got.Bounds() != soll.Bounds() || got.Bounds().Dx() != 40 || got.Bounds().Dy() != 30 {
		t.Errorf("Maße %v, Soll %v", got.Bounds(), soll.Bounds())
	}
}
