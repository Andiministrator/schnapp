// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Text über einer Naht Pixel für Pixel wie abgeschnitten (ADR-0023 Nachträge 13 und 16): Außerhalb des Kantenbands
// gleicht das gestauchte Bild dem ungestauchten, links bzw. oben an derselben Stelle, rechts bzw. unten um die Breite
// des Streifens verschoben. Glyphen, die über die Naht reichen, erscheinen auf beiden Seiten mit ihrem Teil
// (textBaender: jede Glyphe in jeden Bereich, den sie berührt; Mutanten N3, N3b).
func TestTextUeberSpaltenNahtWieAbgeschnitten(t *testing.T) {
	rot := color.NRGBA{R: 200, A: 255}
	o := document.Text{Ort: document.Pt(20, 40), Inhalt: "MWMWMWMW\nWMWMWMWM", Schriftgrad: 60, Farbe: rot}
	links, rechts, farbig := vergleicheUmNaht(t, o, 150, 190, rot)
	if links != 0 || rechts != 0 {
		t.Errorf("abweichend links %d, rechts %d (Soll 0)", links, rechts)
	}
	if farbig == 0 {
		t.Error("rechts der Naht keine Schrift: der Vergleich prüft nichts")
	}
}

func TestTextUeberZeilenNahtWieAbgeschnitten(t *testing.T) {
	rot := color.NRGBA{R: 200, A: 255}
	const von, bis = 70, 85
	baue := func() *document.Dokument {
		b := image.NewNRGBA(image.Rect(0, 0, 300, 300))
		draw.Draw(b, b.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		// Grad 120: Der Streifen liegt mitten in der ersten Zeile, beide Teile der Glyphen ragen über das Kantenband hinaus.
		hinzu(t, d, document.Text{Ort: document.Pt(10, 20), Inhalt: "MWM\nWMW", Schriftgrad: 120, Farbe: rot})
		return d
	}
	voll, err := Bild(baue())
	if err != nil {
		t.Fatal(err)
	}
	d := baue()
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: von, Bis: bis}}); err != nil {
		t.Fatal(err)
	}
	g, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	band, breite := int(document.KantenBandHalb)+4, bis-von
	oben, unten, farbig := 0, 0, 0
	for x := 0; x < 300; x++ {
		for y := 0; y < von-band; y++ {
			if !fastGleich(g.NRGBAAt(x, y), voll.NRGBAAt(x, y)) {
				oben++
			}
		}
		for y := von + band; y < 300-breite; y++ {
			p := g.NRGBAAt(x, y)
			if !fastGleich(p, voll.NRGBAAt(x, y+breite)) {
				unten++
			}
			if p.R > 150 && p.G < 100 {
				farbig++
			}
		}
	}
	if oben != 0 || unten != 0 {
		t.Errorf("abweichend oben %d, unten %d (Soll 0)", oben, unten)
	}
	if farbig == 0 {
		t.Error("unter der Naht keine Schrift: der Vergleich prüft nichts")
	}
}
