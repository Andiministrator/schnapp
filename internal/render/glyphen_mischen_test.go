// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// F-766 (ADR-0023 Nachtrag 18): Über vielen Spaltenstreifen mischte der Renderer jede Glyphe je berührtem Bereich mit
// einem eigenen Aufruf ein (54 000 „@“ Grad 74 über 1023 Spaltenstreifen 4,5 s statt 1,1 s), gezählt war jede Glyphe
// einmal. Jetzt mischt ein Aufruf je Glyphe alle Bereiche ein (glyphBaender): gleich viele Aufrufe mit und ohne
// Streifen.
func TestGlyphEinAufrufUeberVielenNaehten(t *testing.T) {
	inhalt := strings.TrimSuffix(strings.Repeat(strings.Repeat("@", 27)+"\n", 10), "\n")
	zaehlen := func(st []document.Streifen) aufwand {
		d := weissesDokument(t, 2048, 900)
		if st != nil {
			if err := d.SetzeStreifen(st); err != nil {
				t.Fatal(err)
			}
		}
		hinzu(t, d, document.Text{Ort: document.Pt(2, 2), Inhalt: inhalt, Schriftgrad: 74, Farbe: qaSchwarz})
		_, a, err := bildMitAufwand(d)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	ohne := zaehlen(nil)
	for name, st := range map[string][]document.Streifen{
		"1023 Spaltenstreifen": spaltenStreifen(1023, 2),
		"Kacheln":              kacheln(),
	} {
		mit := zaehlen(st)
		if mit.einmischungen > mit.glyphen {
			t.Errorf("%s: %d Aufrufe zum Einmischen für %d Glyphen (ohne Streifen %d für %d), Soll höchstens einer je Glyphe",
				name, mit.einmischungen, mit.glyphen, ohne.einmischungen, ohne.glyphen)
		}
	}
	if ohne.glyphen == 0 || ohne.einmischungen != ohne.glyphen {
		t.Errorf("ohne Streifen %d Aufrufe für %d Glyphen", ohne.einmischungen, ohne.glyphen)
	}
}

// F-766: Das gebündelte Einmischen ändert das Bild nicht. textBaender über schmalen Spalten- und Zeilenbändern zeigt in
// jedem Pixel eines Bands genau das Pixel, das ohne Bänder an seiner Stelle im Original steht (jede Glyphe sitzt an
// derselben Lage und ist nur um ganze Pixel verschoben), Bit für Bit; auch halbdurchsichtige Schrift auf durchsichtigem
// Grund. (Über ein ganzes Bild mit Streifen ginge das nicht: Die Kanten der Nähte decken dort fast alles.)
func TestGlyphenGebuendeltBitgleich(t *testing.T) {
	const w, h, grad = 700, 240, 61.0
	sch := fonts.Laden("Go", fonts.Normal)
	face, err := opentype.NewFace(sch.Font, &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	// baender liefert die Bänder in [0, n) ohne jede Stelle x mit x%luecke == rest, mit ihrem Versatz in der Ansicht.
	baender := func(n, luecke, rest int) []band {
		var bs []band
		a, v := 0, 0
		for x := 0; x <= n; x++ {
			if x == n || x%luecke == rest {
				if x > a {
					bs = append(bs, band{A: a, B: x, Versatz: v})
				}
				a = x + 1
				v--
			}
		}
		return bs
	}
	for _, fall := range []struct {
		name  string
		grund color.NRGBA
		farbe color.NRGBA
	}{
		{"deckend auf Weiß", color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 200, G: 20, B: 40, A: 255}},
		{"halbdurchsichtig auf durchsichtig", color.NRGBA{}, color.NRGBA{R: 20, G: 120, B: 240, A: 140}},
	} {
		leer := func() *image.NRGBA {
			b := image.NewNRGBA(image.Rect(0, 0, w, h))
			for i := 0; i < len(b.Pix); i += 4 {
				b.Pix[i], b.Pix[i+1], b.Pix[i+2], b.Pix[i+3] = fall.grund.R, fall.grund.G, fall.grund.B, fall.grund.A
			}
			return b
		}
		ohne, mit := leer(), leer()
		zo := zeichner{bild: ohne, rand: ohne.Bounds(), auf: &aufwand{}}
		zm := zeichner{bild: mit, rand: mit.Bounds(), auf: &aufwand{}}
		spalten, reihen := baender(w, 3, 1), baender(h, 15, 7)
		for i, s := range []string{"W@gAjy Wolke", "@@MWj quer"} {
			y := 70 + 90*float64(i)
			zo.textBaender(face, sch, s, 5.3, y, fall.farbe, grad, []band{{A: 0, B: w}}, []band{{A: 0, B: h}})
			zm.textBaender(face, sch, s, 5.3, y, fall.farbe, grad, spalten, reihen)
		}
		ab, tinte := 0, 0
		for _, ze := range reihen {
			for y := ze.A; y < ze.B; y++ {
				for _, sp := range spalten {
					for x := sp.A; x < sp.B; x++ {
						a, b := mit.NRGBAAt(x+sp.Versatz, y+ze.Versatz), ohne.NRGBAAt(x, y)
						if a != b {
							ab++
						}
						if a != fall.grund {
							tinte++
						}
					}
				}
			}
		}
		if tinte == 0 {
			t.Fatalf("%s: keine Schrift", fall.name)
		}
		if ab != 0 {
			t.Errorf("%s: %d Pixel weichen vom Original an ihrer Stelle ab", fall.name, ab)
		}
	}
}

// BenchmarkGlyphenSpaltenGrenze rastert „@“ Grad 74 über 1023 Spaltenstreifen (Abstand 2) knapp unter GlyphBudget: die
// Wartezeit, die dieses Muster beim Speichern erzwingen kann (F-766, ADR-0023 Nachtrag 18; Zeitversprechen rund 21 s,
// ADR-0020 Nachtrag 12), zum Vergleich dieselben Texte ohne Streifen.
func BenchmarkGlyphenSpaltenGrenze(b *testing.B) {
	inhalt := strings.TrimSuffix(strings.Repeat(strings.Repeat("@", 27)+"\n", 10), "\n")
	text := document.Text{Ort: document.Pt(2, 2), Inhalt: inhalt, Schriftgrad: 74, Farbe: qaSchwarz}
	for _, fall := range []struct {
		name string
		st   []document.Streifen
	}{{"1023 Spaltenstreifen", spaltenStreifen(1023, 2)}, {"ohne Streifen", nil}} {
		b.Run(fall.name, func(b *testing.B) {
			d := weissesDokument(b, 2048, 900)
			if fall.st != nil {
				if err := d.SetzeStreifen(fall.st); err != nil {
					b.Fatal(err)
				}
			}
			hinzu(b, d, text)
			je := trockenlauf(d, false).glyph
			n := int(GlyphBudget/je) - 1
			for i := 1; i < n; i++ {
				hinzu(b, d, text)
			}
			if err := Pruefe(d); err != nil {
				b.Fatalf("%d Texte: %v", n, err)
			}
			b.Logf("%d Texte, Glyphkosten %.3g von %.3g", n, trockenlauf(d, false).glyph, float64(GlyphBudget))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Bild(d); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
