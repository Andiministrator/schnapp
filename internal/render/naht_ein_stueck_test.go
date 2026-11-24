// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Regressionstests aus der Sitzungs-Kritik zu Sprint 13 (Teil A): Text, der in einem Streifen endet, steht nicht hinter
// der Naht (A-1, [S]), waagrecht und senkrecht; kreuzende Nähte mit Zuschnitt und Deckkraft wie ungestaucht (A-6).

var qaSchwarz = color.NRGBA{A: 255}

func qaDunkel(b *image.NRGBA, x0, x1, y0, y1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if c := b.NRGBAAt(x, y); c.A > 0 && c.R < 200 {
				n++
			}
		}
	}
	return n
}

// A-1: Text beginnt sichtbar und endet im Streifen (nur EIN Stück) — steht der Rest hinter der Naht?
func TestTextEndetImStreifen(t *testing.T) {
	for _, deck := range []uint8{0, 50} {
		text := document.Text{Ort: document.Pt(12, 30), Inhalt: "WWW Geheimnis 1234567 WWW", Schriftgrad: 22, Farbe: qaSchwarz,
			Ebene: document.Ebene{Transparenz: deck}}
		d := weissesDokument(t, 700, 90)
		hinzu(t, d, text)
		voll, _ := Bild(d)
		ende := 0
		for x := 699; x >= 0 && ende == 0; x-- {
			if qaDunkel(voll, x, x+1, 0, 90) > 0 {
				ende = x
			}
		}
		if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: ende + 30}}); err != nil {
			t.Fatal(err)
		}
		n := len(d.TextStuecke(text))
		g, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		band := int(document.KantenBandHalb) + 4
		dunkel := qaDunkel(g, 100+band, g.Bounds().Dx(), 0, 90)
		t.Logf("Transparenz %d: Textende %d, Stücke %d, dunkle Pixel rechts der Naht (x>=%d): %d", deck, ende, n, 100+band, dunkel)
		if dunkel != 0 {
			t.Errorf("Transparenz %d: %d dunkle Pixel des im Streifen liegenden Textteils hinter der Naht", deck, dunkel)
		}
	}
}

// A-1b: dasselbe senkrecht (Zeilenstreifen): mehrzeiliger Text, untere Zeilen im Streifen.
func TestTextZeilenEndenImStreifen(t *testing.T) {
	text := document.Text{Ort: document.Pt(12, 10), Inhalt: "Oben\nGeheim1\nGeheim2\nGeheim3", Schriftgrad: 22, Farbe: qaSchwarz}
	d := weissesDokument(t, 300, 400)
	hinzu(t, d, text)
	g0 := text.Grenzen()
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 40, Bis: int(g0.Max.Y) + 40}}); err != nil {
		t.Fatal(err)
	}
	g, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	band := int(document.KantenBandHalb) + 4
	dunkel := qaDunkel(g, 0, 300, 40+band, g.Bounds().Dy())
	t.Logf("Grenzen %v, Stücke %d, dunkle Pixel unter der Naht: %d", g0, len(d.TextStuecke(text)), dunkel)
	if dunkel != 0 {
		t.Errorf("%d dunkle Pixel der weggestauchten Zeilen unter der Naht", dunkel)
	}
}

// Kreuzende Nähte + Zuschnitt: Vergleich gegen ungestaucht, je Quadrant.
func TestKreuzendeNaehteMitZuschnitt(t *testing.T) {
	sp := document.Pt(330, 170)
	objekte := map[string]document.Objekt{
		"Text":        document.Text{Ort: document.Pt(30, 60), Inhalt: "WWW Geheimnis 1234567 WWW\nzweite Zeile mit Text MMMM\ndritte Zeile gjpqy XXXXXX\nvierte", Schriftgrad: 30, Farbe: qaSchwarz},
		"TextDeck":    document.Text{Ort: document.Pt(30, 60), Inhalt: "WWW Geheimnis 1234567 WWW\nzweite Zeile mit Text MMMM\ndritte Zeile gjpqy XXXXXX\nvierte", Schriftgrad: 30, Farbe: qaSchwarz, Ebene: document.Ebene{Transparenz: 40}},
		"Nummer":      document.Nummer{Mitte: document.Pt(170, 100), Wert: 7, Radius: 45, Farbe: qaSchwarz},
		"NummerDeck":  document.Nummer{Mitte: document.Pt(170, 100), Wert: 7, Radius: 45, Farbe: qaSchwarz, Ebene: document.Ebene{Transparenz: 40}},
		"Sprechblase": document.Text{Ort: document.Pt(40, 40), Inhalt: "Hi", Schriftgrad: 40, Farbe: qaSchwarz, Hintergrund: color.NRGBA{R: 30, G: 90, B: 100, A: 255}, Spitze: &sp},
		"BlaseDeck":   document.Text{Ort: document.Pt(40, 40), Inhalt: "Hi", Schriftgrad: 40, Farbe: qaSchwarz, Hintergrund: color.NRGBA{R: 30, G: 90, B: 100, A: 255}, Spitze: &sp, Ebene: document.Ebene{Transparenz: 40}},
		"Lupe": document.Lupe{Quelle: document.R(document.Pt(30, 30), document.Pt(60, 60)),
			Ziel: document.R(document.Pt(280, 110), document.Pt(340, 170)), Strich: document.Strich{Farbe: qaSchwarz, Breite: 3}},
	}
	zus := image.Rect(10, 14, 390, 190)
	sx0, sx1, sy0, sy1 := 150, 190, 80, 110
	for name, o := range objekte {
		baue := func() *document.Dokument {
			d := weissesDokument(t, 400, 200)
			hinzu(t, d, o)
			return d
		}
		voll, err := Bild(baue())
		if err != nil {
			t.Fatal(err)
		}
		d := baue()
		if err := d.SetzeAnsicht(zus, []document.Streifen{{Von: sx0, Bis: sx1}, {Zeilen: true, Von: sy0, Bis: sy1}}); err != nil {
			t.Fatal(err)
		}
		g, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		band := int(document.KantenBandHalb) + 4
		ab, tinte := 0, 0
		// Originalpunkt (ox, oy) sichtbar -> Ansicht
		for oy := zus.Min.Y; oy < zus.Max.Y; oy++ {
			if oy >= sy0-band && oy < sy1+band {
				continue
			}
			for ox := zus.Min.X; ox < zus.Max.X; ox++ {
				if ox >= sx0-band && ox < sx1+band {
					continue
				}
				ax, ay := ox-zus.Min.X, oy-zus.Min.Y
				if ox >= sx1 {
					ax -= sx1 - sx0
				}
				if oy >= sy1 {
					ay -= sy1 - sy0
				}
				v := voll.NRGBAAt(ox, oy)
				if !fastGleich(g.NRGBAAt(ax, ay), v) {
					ab++
				}
				if v.R < 200 {
					tinte++
				}
			}
		}
		t.Logf("%s: abweichend %d, Tinte im Vergleich %d, Größe %v", name, ab, tinte, g.Bounds())
		if ab != 0 {
			t.Errorf("%s: %d Pixel weichen vom ungestauchten Bild ab", name, ab)
		}
	}
}
