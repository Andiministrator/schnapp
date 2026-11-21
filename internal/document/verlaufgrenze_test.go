// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// F-64: Der Rückgängig-Verlauf hält Pixelkopien nur bis zu einer Gesamtgröße. Darüber fallen die ältesten Schritte
// weg, der jüngste bleibt, und Rückgängig liefert danach genau den Stand vor dem jüngsten verbliebenen Schritt.
func TestVerlaufGrenze(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 40 // jede Schwärzung unten hält 2 x 2 x 4 = 16 Bytes: zwei passen, drei nicht

	weiss := image.NewNRGBA(image.Rect(0, 0, 10, 2))
	for i := range weiss.Pix {
		weiss.Pix[i] = 255
	}
	d, err := Neu(weiss)
	if err != nil {
		t.Fatal(err)
	}
	schwarz := Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}
	var staende []uint64
	for x := 0; x < 8; x += 2 { // vier Schwärzungen bei x = 0, 2, 4, 6
		if err := d.Schwaerzen(image.Rect(x, 0, x+2, 2), schwarz); err != nil {
			t.Fatal(err)
		}
		staende = append(staende, d.Stand())
	}
	if n := len(d.verlauf.erledigt); n != 2 {
		t.Fatalf("Verlauf hält %d Schritte, erwartet 2", n)
	}
	if err := d.Rueckgaengig(); err != nil { // jüngster Schritt (x = 6) bleibt zurücknehmbar
		t.Fatal(err)
	}
	if err := d.Rueckgaengig(); err != nil { // x = 4
		t.Fatal(err)
	}
	if d.KannRueckgaengig() {
		t.Error("ältester Schritt hätte wegfallen müssen")
	}
	if err := d.Rueckgaengig(); err != ErrNichtsDa {
		t.Errorf("dritter Rückgängig: %v", err)
	}
	if d.Stand() != staende[1] {
		t.Errorf("Stand %d, erwartet der nach der zweiten Schwärzung (%d)", d.Stand(), staende[1])
	}
	// Erwartet: x = 0..3 geschwärzt (nicht mehr rückgängig), x = 4..9 wieder weiß.
	o := d.Original()
	for x := 0; x < 10; x++ {
		want := uint8(255)
		if x < 4 {
			want = 0
		}
		if got := o.NRGBAAt(x, 0).R; got != want {
			t.Errorf("Pixel x=%d: R=%d, erwartet %d", x, got, want)
		}
	}
	// Wiederholen funktioniert für die verbliebenen Schritte.
	if err := d.Wiederholen(); err != nil {
		t.Fatal(err)
	}
	if got := d.Original().NRGBAAt(4, 0).R; got != 0 {
		t.Errorf("Wiederholen: x=4 R=%d", got)
	}
}

// Ein einzelner Schritt über der Grenze bleibt (der jüngste fällt nie weg).
func TestVerlaufGrenzeJuengsterBleibt(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 1
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 4, 4)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 4, 4), Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
		t.Fatal(err)
	}
	if !d.KannRueckgaengig() {
		t.Error("jüngster Schritt fiel weg")
	}
}
