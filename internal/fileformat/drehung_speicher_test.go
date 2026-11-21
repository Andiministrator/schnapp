// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"runtime"
	"testing"
	"testing/iotest"
)

// richteAusReferenz ist die frühere Umsetzung (zwei Vollbild-Kopien neben der Quelle, F-323 a); sie dient nur als
// Bezug für die Bitgleichheit.
func richteAusReferenz(bild image.Image, o int) *image.NRGBA {
	b := bild.Bounds()
	w, h := b.Dx(), b.Dy()
	quelle := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(quelle, quelle.Bounds(), bild, b.Min, draw.Src)
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	ziel := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			var sx, sy int
			switch o {
			case 2:
				sx, sy = w-1-dx, dy
			case 3:
				sx, sy = w-1-dx, h-1-dy
			case 4:
				sx, sy = dx, h-1-dy
			case 5:
				sx, sy = dy, dx
			case 6:
				sx, sy = dy, h-1-dx
			case 7:
				sx, sy = w-1-dy, h-1-dx
			case 8:
				sx, sy = w-1-dy, dx
			}
			copy(ziel.Pix[ziel.PixOffset(dx, dy):][:4], quelle.Pix[quelle.PixOffset(sx, sy):][:4])
		}
	}
	return ziel
}

// Bitgleich zur früheren Umsetzung für alle Ausrichtungen und die Quelltypen JPEG-YCbCr (4:2:0 aus dem Decoder, 4:4:4,
// 4:2:2, 4:4:0 und 4:2:0 von Hand, auch mit Ursprung ungleich (0, 0) und als Teilbild), Gray, Gray16, RGBA mit
// Durchsichtigkeit und mit A = 0, RGBA64, CMYK, Paletted mit halbdurchsichtigen Einträgen, NRGBA und Teilbild
// (F-332, F-335).
func TestRichteAusBitgleichZurReferenz(t *testing.T) {
	const w, h = 37, 23 // ungerade Maße: Farbunterabtastung und Ränder
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	gray := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{R: uint8(x * 7), G: uint8(y * 11), B: uint8(x*y + 3), A: uint8(255 - (x+y)*3)}
			nrgba.SetNRGBA(x, y, c)
			rgba.Set(x, y, c)
			gray.SetGray(x, y, color.Gray{Y: uint8(x*5 + y*3)})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, nrgba, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	ycc, err := jpeg.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ycc.(*image.YCbCr); !ok {
		t.Fatalf("JPEG liefert %T, erwartet *image.YCbCr", ycc)
	}
	teil := image.NewNRGBA(image.Rect(5, 7, 5+w, 7+h))
	copy(teil.Pix, nrgba.Pix)

	quellen := map[string]image.Image{"YCbCr": ycc, "Gray": gray, "RGBA": rgba, "NRGBA": nrgba, "Teilbild": teil}
	// jpeg.Encode schreibt nur 4:2:0; die anderen Unterabtastungen und Ursprung ungleich (0, 0) kommen von Hand.
	for name, ratio := range map[string]image.YCbCrSubsampleRatio{"444": image.YCbCrSubsampleRatio444, "422": image.YCbCrSubsampleRatio422, "440": image.YCbCrSubsampleRatio440, "420": image.YCbCrSubsampleRatio420} {
		y := image.NewYCbCr(image.Rect(-3, -5, 61, 43), ratio)
		for i := range y.Y {
			y.Y[i] = uint8(i * 7)
		}
		for i := range y.Cb {
			y.Cb[i], y.Cr[i] = uint8(i*13+5), uint8(i*29+11)
		}
		quellen["YCbCr"+name] = y
		quellen["YCbCr"+name+"-Teil"] = y.SubImage(image.Rect(7, 9, 7+37, 9+23))
	}
	// Quellen, bei denen die Vormultiplikation und A = 0 zählen (F-332).
	rgbaNull := image.NewRGBA(image.Rect(0, 0, w, h)) // A = 0 mit Farbe ungleich 0
	rgba64 := image.NewRGBA64(image.Rect(0, 0, w, h))
	gray16 := image.NewGray16(image.Rect(0, 0, w, h))
	cmyk := image.NewCMYK(image.Rect(0, 0, w, h))
	pal := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{
		color.NRGBA{10, 200, 30, 1}, color.NRGBA{250, 3, 99, 77}, color.NRGBA{1, 2, 3, 128}, color.NRGBA{255, 255, 0, 254}, color.NRGBA{9, 8, 7, 255}, color.NRGBA{5, 6, 7, 0},
	})
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := rgbaNull.PixOffset(x, y)
			a := uint8((x + y) % 4 * 85) // 0, 85, 170, 255
			rgbaNull.Pix[i], rgbaNull.Pix[i+1], rgbaNull.Pix[i+2], rgbaNull.Pix[i+3] = uint8(x*3%(int(a)+1)), uint8(y*5%(int(a)+1)), uint8((x+y)%(int(a)+1)), a
			if a == 0 {
				rgbaNull.Pix[i], rgbaNull.Pix[i+1], rgbaNull.Pix[i+2] = 200, 100, 50
			}
			rgba64.SetRGBA64(x, y, color.RGBA64{R: uint16(x*997) % 0x4000, G: uint16(y*1999) % 0x4000, B: uint16(x*y*31) % 0x4000, A: uint16(0x4000 + (x*y*131+x*17)%0xc000)})
			gray16.SetGray16(x, y, color.Gray16{Y: uint16(x*1777 + y*911)})
			cmyk.SetCMYK(x, y, color.CMYK{C: uint8(x * 7), M: uint8(y * 9), Y: uint8(x + y), K: uint8(x * y)})
			pal.SetColorIndex(x, y, uint8((x*3+y)%6))
		}
	}
	quellen["RGBA-A0"], quellen["RGBA64"], quellen["Gray16"], quellen["CMYK"], quellen["Paletted"] = rgbaNull, rgba64, gray16, cmyk, pal
	for name, q := range quellen {
		for o := 2; o <= 8; o++ {
			got, ok := richteAus(q, o).(*image.NRGBA)
			if !ok {
				t.Fatalf("%s, Ausrichtung %d: kein NRGBA", name, o)
			}
			want := richteAusReferenz(q, o)
			if got.Rect != want.Rect || !bytes.Equal(got.Pix, want.Pix) {
				t.Errorf("%s, Ausrichtung %d: weicht von der Referenz ab", name, o)
			}
		}
	}
}

// Speichergrenze (F-323 a): Neben der Quelle entsteht höchstens eine Vollbild-Kopie (das Ergebnis). Gemessen wird die
// insgesamt angeforderte Speichermenge; früher kam eine zweite NRGBA-Kopie der Quelle hinzu, jetzt bleibt es unter
// 1,5 Ergebnisgrößen.
func TestRichteAusEineKopie(t *testing.T) {
	const w, h = 1024, 768
	quellen := map[string]image.Image{
		"NRGBA": image.NewNRGBA(image.Rect(0, 0, w, h)),
		"YCbCr": image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420),
		"Gray":  image.NewGray(image.Rect(0, 0, w, h)),
		"CMYK":  image.NewCMYK(image.Rect(0, 0, w, h)),
	}
	ergebnis := uint64(w * h * 4)
	for name, q := range quellen {
		for _, o := range []int{2, 3, 4, 5, 6, 7, 8} {
			var a, b runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&a)
			r := richteAus(q, o)
			runtime.ReadMemStats(&b)
			runtime.KeepAlive(r)
			if d := b.TotalAlloc - a.TotalAlloc; d > ergebnis+ergebnis/2 {
				t.Errorf("%s, Ausrichtung %d: %d Bytes angefordert, erlaubt höchstens %d (eine Kopie)", name, o, d, ergebnis+ergebnis/2)
			}
			// Der Rückfall über At fordert je Pixel ein bis zwei Objekte an (F-333); hier bleibt es bei einer Handvoll.
			if n := b.Mallocs - a.Mallocs; n > 50 {
				t.Errorf("%s, Ausrichtung %d: %d Anforderungen, erlaubt höchstens 50 (keine je Pixel)", name, o, n)
			}
		}
	}
}

// Laufzeit des Ausrichtens bei 12 Megapixeln (F-333): go test -run '^$' -bench RichteAus -benchmem ./internal/fileformat
func BenchmarkRichteAus(b *testing.B) {
	const w, h = 4000, 3000
	quellen := map[string]image.Image{
		"Gray": image.NewGray(image.Rect(0, 0, w, h)),
		"CMYK": image.NewCMYK(image.Rect(0, 0, w, h)),
	}
	for name, q := range quellen {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				richteAus(q, 6)
			}
		})
	}
}

// blockStrom: Lesen mit leerem Puffer liefert 0, nil und ändert nichts; iotest.TestReader ist grün; die Bytes sind
// dieselben wie bei schreibeBloecke (F-323 b).
func testBloecke() []block {
	return []block{
		{typ: "IHDR", daten: []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0}},
		{typ: "snAP", daten: bytes.Repeat([]byte{1, 2, 3}, 100)},
		{typ: "tEXt", daten: nil},
		{typ: "IEND", daten: nil},
	}
}

func TestBlockStromGleichSchreibeBloecke(t *testing.T) {
	bloecke := testBloecke()
	var want bytes.Buffer
	if err := schreibeBloecke(&want, bloecke); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(neuerBlockStrom(bloecke))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Errorf("Strom (%d Bytes) weicht von schreibeBloecke (%d Bytes) ab", len(got), want.Len())
	}
}

func TestBlockStromTestReader(t *testing.T) {
	bloecke := testBloecke()
	var want bytes.Buffer
	if err := schreibeBloecke(&want, bloecke); err != nil {
		t.Fatal(err)
	}
	if err := iotest.TestReader(neuerBlockStrom(bloecke), want.Bytes()); err != nil {
		t.Error(err)
	}
}

func TestBlockStromLeererPuffer(t *testing.T) {
	bloecke := testBloecke()
	var want bytes.Buffer
	if err := schreibeBloecke(&want, bloecke); err != nil {
		t.Fatal(err)
	}
	s := neuerBlockStrom(bloecke)
	// Leere Lesungen zwischen jedem Byte: nie ein Byte verschlucken.
	var got []byte
	one := make([]byte, 1)
	for {
		if n, err := s.Read(nil); n != 0 || err != nil {
			t.Fatalf("Read(nil) = %d, %v; erwartet 0, nil", n, err)
		}
		n, err := s.Read(one)
		got = append(got, one[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Error("Bytes nach leeren Lesungen weichen ab")
	}
}
