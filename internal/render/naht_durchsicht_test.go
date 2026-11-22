// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/raster16"
)

// ADR-0023 Nachtrag 7, Ausgaben: Der Spalt trägt Alpha durch alle Wege. Das PNG der Zwischenablage (PNG) und der
// PNG-Export (fileformat.ExportierePNG) behalten ihn durchsichtig und den Schatten halbdurchsichtig; JPEG kennt keine
// Durchsicht und legt den Spalt auf Weiß, den Schatten darauf gemischt. Naht wie in TestKanteSollwerte (Zacken bei x 20,
// Zeile 4): Pixel 22 liegt im Spalt ohne Schatten, Pixel 15 in zwei Schattenbändern, Pixel 9 im Bild.
func TestNahtDurchsichtigInDenAusgaben(t *testing.T) {
	d := zackenDokument(t, 100, 40)
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	schatten := ueber(durchsichtig, schatten08, schatten10)

	daten, err := PNG(d)
	if err != nil {
		t.Fatal(err)
	}
	var export bytes.Buffer
	b := bild(t, d)
	if err := fileformat.ExportierePNG(&export, b); err != nil {
		t.Fatal(err)
	}
	for name, roh := range map[string][]byte{"Zwischenablage": daten, "PNG-Export": export.Bytes()} {
		img, err := png.Decode(bytes.NewReader(roh))
		if err != nil {
			t.Fatal(name, err)
		}
		n := color.NRGBAModel.Convert(img.At(22, 4)).(color.NRGBA)
		s := color.NRGBAModel.Convert(img.At(15, 4)).(color.NRGBA)
		g := color.NRGBAModel.Convert(img.At(9, 4)).(color.NRGBA)
		if n.A != 0 || s != schatten || g != grau {
			t.Errorf("%s: Spalt %v, Schatten %v (Soll %v), Bild %v", name, n, s, schatten, g)
		}
	}

	var j bytes.Buffer
	if err := fileformat.ExportiereJPG(&j, b, 100); err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(&j)
	if err != nil {
		t.Fatal(err)
	}
	// Soll: Weiß mit dem Schatten darüber gemischt; JPEG verlustbehaftet, daher mit Spielraum.
	weissSchatten := ueber(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, schatten08, schatten10)
	for _, c := range []struct {
		p    image.Point
		soll color.NRGBA
	}{{image.Pt(22, 4), color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}}, {image.Pt(15, 4), weissSchatten},
		{image.Pt(9, 4), grau}} {
		got := color.NRGBAModel.Convert(img.At(c.p.X, c.p.Y)).(color.NRGBA)
		if !nahe(got, c.soll, 6) {
			t.Errorf("JPEG Pixel %v = %v, Soll etwa %v", c.p, got, c.soll)
		}
	}
}

// nahe vergleicht zwei Farben je Kanal mit Spielraum.
func nahe(a, b color.NRGBA, spiel int) bool {
	ab := func(x, y uint8) bool { d := int(x) - int(y); return d >= -spiel && d <= spiel }
	return ab(a.R, b.R) && ab(a.G, b.G) && ab(a.B, b.B) && a.A == b.A
}

// ausradieren für sich (ohne die Schichten, die danach über den Spalt gehen): Ganz bedeckte Pixel werden (0,0,0,0),
// ein halb bedecktes behält seine Farbe und halbiert die Deckung, unbedeckte bleiben.
func TestAusradieren(t *testing.T) {
	farbe := color.NRGBA{R: 0x40, G: 0x80, B: 0xC0, A: 0xFF}
	b := image.NewNRGBA(image.Rect(0, 0, 6, 2))
	for i := 0; i < len(b.Pix); i += 4 {
		copy(b.Pix[i:i+4], []uint8{farbe.R, farbe.G, farbe.B, farbe.A})
	}
	z := zeichner{bild: b, rand: b.Bounds(), auf: &aufwand{}, ras: &raster16.Rasterizer{}}
	// Rechteck x 1 bis 3,5 über beide Zeilen: Pixel 1 und 2 ganz, Pixel 3 halb.
	z.radiere([][]document.Punkt{{document.Pt(1, 0), document.Pt(3.5, 0), document.Pt(3.5, 2), document.Pt(1, 2)}})
	halb := b.NRGBAAt(3, 0)
	pruefe(t, b, map[image.Point]color.NRGBA{{0, 0}: farbe, {1, 0}: durchsichtig, {2, 1}: durchsichtig, {4, 1}: farbe})
	if halb.R != farbe.R || halb.G != farbe.G || halb.B != farbe.B || halb.A < 0x7E || halb.A > 0x81 {
		t.Errorf("halb bedeckt %v, Soll Farbe %v mit Deckung um 0x80", halb, farbe)
	}
	if z.auf.flaeche <= 0 || z.auf.kanten <= 0 {
		t.Errorf("Radieren kostet nichts: %+v", *z.auf)
	}
}

// Der Spalt nimmt auch Pixel mit, die schon durchsichtig waren, ohne Farbreste: ein ganz durchsichtiges Pixel ist
// (0,0,0,0), wie color.NRGBAModel es schreibt. Und ein halbdurchsichtiges Bild bleibt neben dem Spalt unverändert.
func TestRadierenDurchsichtigesBild(t *testing.T) {
	halb := color.NRGBA{R: 0x40, G: 0x80, B: 0xC0, A: 0x80}
	b := image.NewNRGBA(image.Rect(0, 0, 100, 40))
	for i := 0; i < len(b.Pix); i += 4 {
		copy(b.Pix[i:i+4], []uint8{halb.R, halb.G, halb.B, halb.A})
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	pruefe(t, bild(t, d), map[image.Point]color.NRGBA{{22, 4}: durchsichtig, {26, 4}: durchsichtig, {9, 4}: halb})
}
