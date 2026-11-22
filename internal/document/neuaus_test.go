// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"image"
	"testing"
)

// F-334: NeuAus übernimmt ein geschlossenes NRGBA ab (0, 0) ohne Kopie; Neu kopiert weiterhin.
func TestNeuAusUebernimmtOhneKopie(t *testing.T) {
	o := image.NewNRGBA(image.Rect(0, 0, 8, 5))
	o.Pix[0] = 42
	d, err := NeuAus(o)
	if err != nil {
		t.Fatal(err)
	}
	if &d.OriginalBild().Pix[0] != &o.Pix[0] {
		t.Error("NeuAus hat kopiert")
	}
	if g := d.Groesse(); g != image.Pt(8, 5) {
		t.Errorf("Größe %v", g)
	}
	n, err := Neu(o)
	if err != nil {
		t.Fatal(err)
	}
	if &n.OriginalBild().Pix[0] == &o.Pix[0] {
		t.Error("Neu teilt die Pixel des Aufrufers")
	}
}

// Ein Teilbild, ein Ursprung ungleich (0, 0) oder ein abweichender Zeilenabstand wird kopiert, nicht übernommen: Sonst
// teilte das Dokument Pixel mit dem umgebenden Bild.
func TestNeuAusKopiertTeilbilder(t *testing.T) {
	gross := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for i := range gross.Pix {
		gross.Pix[i] = uint8(i)
	}
	versetzt := image.NewNRGBA(image.Rect(3, 4, 9, 8))
	faelle := map[string]*image.NRGBA{
		"Teilbild ab 0":     gross.SubImage(image.Rect(0, 0, 6, 4)).(*image.NRGBA),
		"Teilbild innen":    gross.SubImage(image.Rect(2, 3, 7, 9)).(*image.NRGBA),
		"Ursprung versetzt": versetzt,
	}
	for name, o := range faelle {
		d, err := NeuAus(o)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		org := d.OriginalBild()
		if &org.Pix[0] == &o.Pix[0] {
			t.Errorf("%s: Pixel übernommen statt kopiert", name)
		}
		r := o.Rect
		if org.Rect != image.Rect(0, 0, r.Dx(), r.Dy()) {
			t.Errorf("%s: Original %v", name, org.Rect)
		}
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				if org.NRGBAAt(x, y) != o.NRGBAAt(r.Min.X+x, r.Min.Y+y) {
					t.Fatalf("%s: Pixel (%d, %d) weicht ab", name, x, y)
				}
			}
		}
	}
}

// Dieselben Grenzen wie Neu; nil ist ein Fehler, keine Panik.
func TestNeuAusGrenzen(t *testing.T) {
	if _, err := NeuAus(nil); !errors.Is(err, ErrBildGroesse) {
		t.Errorf("nil: %v", err)
	}
	if _, err := NeuAus(image.NewNRGBA(image.Rect(0, 0, 0, 3))); !errors.Is(err, ErrBildGroesse) {
		t.Errorf("leer: %v", err)
	}
	breit := &image.NRGBA{Rect: image.Rect(0, 0, MaxSeite+1, 1), Stride: 4 * (MaxSeite + 1)}
	breit.Pix = make([]uint8, breit.Stride)
	if _, err := NeuAus(breit); !errors.Is(err, ErrBildGroesse) {
		t.Errorf("zu breit: %v", err)
	}
}
