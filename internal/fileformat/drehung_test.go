// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"path/filepath"
	"testing"
)

// Kennfarben der sechs Felder des Prüfbilds (3 breit, 2 hoch): A B C / D E F, weit auseinander, damit die JPEG-
// Verluste nichts verwischen.
var feldFarben = map[byte]color.NRGBA{
	'A': {R: 255, A: 255}, 'B': {G: 255, A: 255}, 'C': {B: 255, A: 255},
	'D': {R: 255, G: 255, A: 255}, 'E': {R: 255, B: 255, A: 255}, 'F': {G: 255, B: 255, A: 255},
}

// erwartet je Ausrichtung das Bild nach dem Drehen, zeilenweise; unabhängig von der Formel in richteAus aufgeschrieben.
var erwartetNachAusrichtung = map[int][]string{
	1: {"ABC", "DEF"},
	2: {"CBA", "FED"},
	3: {"FED", "CBA"},
	4: {"DEF", "ABC"},
	5: {"AD", "BE", "CF"},
	6: {"DA", "EB", "FC"},
	7: {"FC", "EB", "DA"},
	8: {"CF", "BE", "AD"},
}

// pruefbild zeichnet die Felder A bis F als Blöcke der Seite blk Pixel.
func pruefbild(blk int) *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, 3*blk, 2*blk))
	zeilen := []string{"ABC", "DEF"}
	for y := 0; y < 2*blk; y++ {
		for x := 0; x < 3*blk; x++ {
			b.SetNRGBA(x, y, feldFarben[zeilen[y/blk][x/blk]])
		}
	}
	return b
}

func naheFarbe(a, b color.NRGBA) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) < 60 && d(a.G, b.G) < 60 && d(a.B, b.B) < 60
}

// pruefeFelder vergleicht die Feldmitten von bild mit dem Soll.
func pruefeFelder(t *testing.T, bild image.Image, soll []string, blk int) {
	t.Helper()
	b := bild.Bounds()
	if b.Dx() != len(soll[0])*blk || b.Dy() != len(soll)*blk {
		t.Fatalf("Größe %dx%d, erwartet %dx%d", b.Dx(), b.Dy(), len(soll[0])*blk, len(soll)*blk)
	}
	for zy, zeile := range soll {
		for zx := 0; zx < len(zeile); zx++ {
			r, g, bl, a := bild.At(b.Min.X+zx*blk+blk/2, b.Min.Y+zy*blk+blk/2).RGBA()
			ist := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: uint8(a >> 8)}
			if !naheFarbe(ist, feldFarben[zeile[zx]]) {
				t.Errorf("Feld (%d,%d): %v, erwartet Feld %c %v", zx, zy, ist, zeile[zx], feldFarben[zeile[zx]])
			}
		}
	}
}

// exifSegment baut ein APP1-Segment mit einem IFD0, der nur das Ausrichtungsfeld trägt.
func exifSegment(bo binary.AppendByteOrder, mm bool, ausrichtung uint16) []byte {
	t := []byte{'I', 'I'}
	if mm {
		t = []byte{'M', 'M'}
	}
	t = bo.AppendUint16(t, 42)
	t = bo.AppendUint32(t, 8)
	t = bo.AppendUint16(t, 1)      // ein Eintrag
	t = bo.AppendUint16(t, 0x0112) // Tag
	t = bo.AppendUint16(t, 3)      // SHORT
	t = bo.AppendUint32(t, 1)      // Anzahl
	t = bo.AppendUint16(t, ausrichtung)
	t = append(t, 0, 0)       // Rest des Wertfeldes
	t = bo.AppendUint32(t, 0) // kein weiterer IFD
	return app1(append([]byte("Exif\x00\x00"), t...))
}

func app1(inhalt []byte) []byte {
	s := []byte{0xFF, 0xE1}
	s = binary.BigEndian.AppendUint16(s, uint16(len(inhalt)+2))
	return append(s, inhalt...)
}

// mitSegment setzt seg unmittelbar hinter den SOI-Marker.
func mitSegment(jpg, seg []byte) []byte {
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func kodiere(t *testing.T, b image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, b, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// F-113: alle acht Ausrichtungen, beide Bytereihenfolgen, durch Oeffne hindurch.
func TestOeffneJPEGAusrichtung(t *testing.T) {
	const blk = 16
	roh := kodiere(t, pruefbild(blk))
	for o := 1; o <= 8; o++ {
		for _, mm := range []bool{false, true} {
			var bo binary.AppendByteOrder = binary.LittleEndian
			if mm {
				bo = binary.BigEndian
			}
			pfad := filepath.Join(t.TempDir(), "a.jpg")
			schreibeDatei(t, pfad, mitSegment(roh, exifSegment(bo, mm, uint16(o))))
			g, err := Oeffne(pfad)
			if err != nil {
				t.Fatalf("Ausrichtung %d: %v", o, err)
			}
			pruefeFelder(t, g.Dokument.Ansichtsbild(), erwartetNachAusrichtung[o], blk)
		}
	}
}

// Ohne Exif oder mit unbrauchbarem Wert bleibt das Bild unverändert.
func TestOeffneJPEGOhneAusrichtung(t *testing.T) {
	const blk = 16
	roh := kodiere(t, pruefbild(blk))
	for name, daten := range map[string][]byte{
		"ohne Exif":    roh,
		"Wert 0":       mitSegment(roh, exifSegment(binary.LittleEndian, false, 0)),
		"Wert 9":       mitSegment(roh, exifSegment(binary.BigEndian, true, 9)),
		"Wert 65535":   mitSegment(roh, exifSegment(binary.LittleEndian, false, 0xFFFF)),
		"XMP statt Ex": mitSegment(roh, app1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))),
	} {
		pfad := filepath.Join(t.TempDir(), "a.jpg")
		schreibeDatei(t, pfad, daten)
		g, err := Oeffne(pfad)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pruefeFelder(t, g.Dokument.Ansichtsbild(), erwartetNachAusrichtung[1], blk)
	}
}

// Ein XMP-Segment vor dem Exif-Segment verdeckt dieses nicht.
func TestExifNachXMPSegment(t *testing.T) {
	roh := kodiere(t, pruefbild(8))
	xmp := app1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
	daten := mitSegment(mitSegment(roh, exifSegment(binary.LittleEndian, false, 6)), xmp)
	if o := exifAusrichtung(daten); o != 6 {
		t.Errorf("Ausrichtung %d, erwartet 6", o)
	}
}

// richteAus direkt, ohne JPEG-Verluste: Pixel bleiben bitgleich.
func TestRichteAusTabelle(t *testing.T) {
	const blk = 3
	src := pruefbild(blk)
	for o := 1; o <= 8; o++ {
		pruefeFelder(t, richteAus(src, o), erwartetNachAusrichtung[o], blk)
	}
	// Ursprung ungleich (0,0): Teilbild.
	teil := image.NewNRGBA(image.Rect(5, 7, 5+3*blk, 7+2*blk))
	copy(teil.Pix, src.Pix)
	pruefeFelder(t, richteAus(teil, 6), erwartetNachAusrichtung[6], blk)
}

// Kaputte und präparierte Segmente: kein Absturz, keine Hänger, Ergebnis 1 (oder eine gültige Zahl).
func TestExifKaputteEingaben(t *testing.T) {
	gut := exifSegment(binary.LittleEndian, false, 6)
	soi := []byte{0xFF, 0xD8}
	tiff := func(teile ...[]byte) []byte {
		return app1(append([]byte("Exif\x00\x00"), bytes.Join(teile, nil)...))
	}
	le := binary.LittleEndian
	faelle := map[string][]byte{
		"leer":                nil,
		"nur SOI":             soi,
		"kein SOI":            {0x00, 0x01, 0x02, 0x03, 0x04},
		"Segment zu kurz":     append(append([]byte{}, soi...), 0xFF, 0xE1, 0x00),
		"Länge 0":             append(append([]byte{}, soi...), 0xFF, 0xE1, 0x00, 0x00),
		"Länge 1":             append(append([]byte{}, soi...), 0xFF, 0xE1, 0x00, 0x01, 0xFF, 0xD9),
		"Länge über das Ende": append(append([]byte{}, soi...), 0xFF, 0xE1, 0xFF, 0xFF, 'E', 'x'),
		"abgeschnitten":       append(append([]byte{}, soi...), gut[:len(gut)-9]...),
		"abgeschnitten kurz":  append(append([]byte{}, soi...), gut[:12]...),
		"nur Exif-Kennung":    append(append([]byte{}, soi...), app1([]byte("Exif\x00\x00"))...),
		"falsche Bytefolge":   append(append([]byte{}, soi...), tiff([]byte("XX"), []byte{42, 0, 8, 0, 0, 0})...),
		"falsche Magie":       append(append([]byte{}, soi...), tiff([]byte("II"), []byte{43, 0, 8, 0, 0, 0})...),
		"IFD-Offset riesig":   append(append([]byte{}, soi...), tiff([]byte("II"), []byte{42, 0, 0xFF, 0xFF, 0xFF, 0xFF})...),
		"IFD-Offset am Ende":  append(append([]byte{}, soi...), tiff([]byte("II"), []byte{42, 0, 8, 0, 0, 0})...),
		"Zahl der Einträge 65535": append(append([]byte{}, soi...),
			tiff([]byte("II"), []byte{42, 0, 8, 0, 0, 0}, le.AppendUint16(nil, 0xFFFF), make([]byte, 24))...),
		"Typ falsch": append(append([]byte{}, soi...), tiff([]byte("II"), []byte{42, 0, 8, 0, 0, 0}, le.AppendUint16(nil, 1),
			le.AppendUint16(nil, 0x0112), le.AppendUint16(nil, 4), le.AppendUint32(nil, 1), le.AppendUint16(nil, 6), []byte{0, 0})...),
		"Anzahl falsch": append(append([]byte{}, soi...), tiff([]byte("II"), []byte{42, 0, 8, 0, 0, 0}, le.AppendUint16(nil, 1),
			le.AppendUint16(nil, 0x0112), le.AppendUint16(nil, 3), le.AppendUint32(nil, 0xFFFFFFFF), le.AppendUint16(nil, 6), []byte{0, 0})...),
		"Marke ohne Längenfeld am Ende": append(append([]byte{}, soi...), 0xFF, 0xFF, 0xFF, 0xFF, 0xFF),
		"nur Füllbytes":                 append(append([]byte{}, soi...), bytes.Repeat([]byte{0xFF}, 5000)...),
		"Marken ohne Länge":             append(append([]byte{}, soi...), bytes.Repeat([]byte{0xFF, 0xD0}, 5000)...),
	}
	for name, d := range faelle {
		if o := exifAusrichtung(d); o < 1 || o > 8 {
			t.Errorf("%s: Ausrichtung %d außerhalb 1..8", name, o)
		} else if o != 1 {
			t.Errorf("%s: Ausrichtung %d, erwartet 1", name, o)
		}
	}
}

// Jede Kürzung und jede Bytevertauschung eines gültigen Segments bleibt ohne Absturz.
func TestExifVerstuemmelt(t *testing.T) {
	gut := append([]byte{0xFF, 0xD8}, exifSegment(binary.BigEndian, true, 8)...)
	for n := 0; n <= len(gut); n++ {
		exifAusrichtung(gut[:n])
	}
	for i := range gut {
		for _, v := range []byte{0x00, 0x01, 0x7F, 0x80, 0xFF} {
			k := append([]byte{}, gut...)
			k[i] = v
			if o := exifAusrichtung(k); o < 1 || o > 8 {
				t.Fatalf("Byte %d = %#x: Ausrichtung %d", i, v, o)
			}
		}
	}
}

func FuzzExifAusrichtung(f *testing.F) {
	f.Add(append([]byte{0xFF, 0xD8}, exifSegment(binary.LittleEndian, false, 6)...))
	f.Add(append([]byte{0xFF, 0xD8}, exifSegment(binary.BigEndian, true, 3)...))
	f.Fuzz(func(t *testing.T, d []byte) {
		if o := exifAusrichtung(d); o < 1 || o > 8 {
			t.Fatalf("Ausrichtung %d", o)
		}
	})
}
