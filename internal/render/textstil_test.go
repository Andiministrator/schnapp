// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// tinteIn zählt Bildpunkte in r, die vom Weiß abweichen.
func tinteIn(b *image.NRGBA, r image.Rectangle) int {
	n := 0
	r = r.Intersect(b.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := b.NRGBAAt(x, y); c.R < 250 || c.G < 250 || c.B < 250 {
				n++
			}
		}
	}
	return n
}

// TestTextStilImRenderer (Sitzungs-QA B-12): Der Renderer setzt Zeilen nach der Ausrichtung, zeichnet Unter- und
// Durchstrich und nimmt die Schrift des Texts.
func TestTextStilImRenderer(t *testing.T) {
	tx := document.Text{Ort: document.Pt(10, 10), Inhalt: "WWWWWW\nx", Schriftgrad: 30, Farbe: rot,
		Ausrichtung: document.AusrichtungRechts, Unterstrichen: true}
	d := weissesDokument(t, 260, 120)
	hinzu(t, d, tx)
	b, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	z := tx.Zeilen()
	// Zweite Zeile rechtsbündig: links von ihrem Anfang (bis kurz davor) keine Tinte in ihrer Zeilenhöhe.
	links := image.Rect(10, int(z[1].Grundlinie-20), int(z[1].X)-3, int(z[1].Grundlinie))
	if n := tinteIn(b, links); n != 0 {
		t.Errorf("%d Bildpunkte Tinte links der rechtsbündigen Zeile (Anfang %v)", n, z[1].X)
	}
	if n := tinteIn(b, image.Rect(int(z[1].X), int(z[1].Grundlinie-20), int(z[1].X+z[1].Vorschub)+1, int(z[1].Grundlinie))); n == 0 {
		t.Error("keine Tinte an der rechtsbündigen Zeile")
	}
	// Unterstrich: eine durchgehende Linie unter der ersten Zeile.
	st := tx.Striche()[0]
	y := int(math.Floor((st.Min.Y + st.Max.Y) / 2))
	if n := tinteIn(b, image.Rect(int(st.Min.X)+1, y, int(st.Max.X)-1, y+1)); n < int(st.Breite())-3 {
		t.Errorf("Unterstrich: %d von %d Bildpunkten", n, int(st.Breite())-2)
	}
	// Schrift: Go Mono setzt anders als Go.
	mono := tx
	mono.Schrift, mono.Unterstrichen = fonts.GoMono, false
	normal := tx
	normal.Unterstrichen = false
	bild := func(t2 document.Text) *image.NRGBA {
		d := weissesDokument(t, 260, 120)
		hinzu(t, d, t2)
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if a, c := bild(normal), bild(mono); string(a.Pix) == string(c.Pix) {
		t.Error("Go Mono und Go ergeben dasselbe Bild")
	}
	// Die Glyphen kommen aus Go Mono: schmale Zeichen reichen bis ans Ende ihres (breiten) Vorschubs.
	schmal := document.Text{Ort: document.Pt(10, 10), Inhalt: "iiiiiiii", Schriftgrad: 30, Farbe: rot, Schrift: fonts.GoMono}
	sz := schmal.Zeilen()[0]
	b2 := bild(schmal)
	ende := image.Rect(int(sz.X+sz.Vorschub*0.85), int(sz.Grundlinie-25), int(sz.X+sz.Vorschub)+2, int(sz.Grundlinie)+2)
	if n := tinteIn(b2, ende); n == 0 {
		t.Errorf("keine Tinte im letzten Achtel des Vorschubs %v: Glyphen nicht aus Go Mono", sz.Vorschub)
	}
}
