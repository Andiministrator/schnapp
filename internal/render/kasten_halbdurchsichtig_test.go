package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// schnittFlaeche ist die Fläche der Schnittmenge des Pixels (x, y) mit dem Rechteck r.
func schnittFlaeche(x, y int, r document.Rechteck) float64 {
	w := math.Min(float64(x+1), r.Max.X) - math.Max(float64(x), r.Min.X)
	h := math.Min(float64(y+1), r.Max.Y) - math.Max(float64(y), r.Min.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// F-498 (Kritik A-2, B-6): Der eckige Strich eines Kastens besteht aus überlappungsfreien Rechtecken. Mit einem
// halbdurchsichtigen Strich (A = 128) und Kanten auf halben Pixeln fällt jede doppelte Deckung auf, denn die Randpixel
// haben dann eine Teildeckung, die sich bei Überlappung addierte: Jedes Pixel hat die Deckung „Fläche im Strich“ (Außen
// minus Innen), Ecken sind nicht dunkler als Kanten. Fälle: breit, schmal in einer Achse, genau eine Strichbreite, knapp
// mit Innenraum, auf ganzen und halben Pixeln.
func TestKastenEckigHalbdurchsichtig(t *testing.T) {
	strich := document.Strich{Farbe: color.NRGBA{A: 128}, Breite: 8}
	faelle := []struct {
		name    string
		bereich document.Rechteck
	}{
		{"breit", document.R(document.Pt(30, 30), document.Pt(80, 70))},
		{"breit, halbe Pixel", document.R(document.Pt(30.5, 30.5), document.Pt(80.5, 70.5))},
		{"schmal in x", document.R(document.Pt(40, 30), document.Pt(44, 80))},
		{"schmal in x, halbe Pixel", document.R(document.Pt(40.5, 30.5), document.Pt(44.5, 80.5))},
		{"schmal in y, halbe Pixel", document.R(document.Pt(30.5, 40.5), document.Pt(80.5, 44.5))},
		{"genau eine Strichbreite", document.R(document.Pt(30.5, 30.5), document.Pt(38.5, 70.5))},
		{"knapp mit Innenraum", document.R(document.Pt(30.5, 30.5), document.Pt(40.5, 70.5))},
	}
	for _, f := range faelle {
		d := weissesDokument(t, 100, 100)
		k := document.Kasten{Bereich: f.bereich, Strich: strich}
		hinzu(t, d, k)
		b := bild(t, d)
		g := k.Grenzen()
		h := strich.Breite / 2
		ia, ib := f.bereich.Min.Plus(document.Pt(h, h)), f.bereich.Max.Minus(document.Pt(h, h))
		hatInnen := ia.X < ib.X && ia.Y < ib.Y
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				deckung := schnittFlaeche(x, y, g)
				if hatInnen {
					deckung -= schnittFlaeche(x, y, document.R(ia, ib))
				}
				soll := 255 * (1 - deckung*128.0/255)
				c := b.NRGBAAt(x, y)
				if math.Abs(float64(c.R)-soll) > 4 || c.A != 255 {
					t.Errorf("%s: Pixel (%d,%d) = %v, Soll R rund %.0f (Deckung %.2f)", f.name, x, y, c, soll, deckung)
				}
			}
		}
	}
}
