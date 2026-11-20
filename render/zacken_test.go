// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

var grau = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}

func grauesDokument(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(b.Pix); i += 4 {
		copy(b.Pix[i:i+4], []uint8{grau.R, grau.G, grau.B, grau.A})
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func streifen(t *testing.T, d *document.Dokument, st ...document.Streifen) {
	t.Helper()
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
}

// ADR-0023 Punkt 6: Zackenkante an der senkrechten Naht bei x = 20 (Spalten 20 bis 40 heraus). Die Zickzacklinie läuft
// von x 16 (y 0) nach x 24 (y 8) und zurück nach 16 (y 16); in Zeile 4 liegt sie bei x 20 bis 21, in Zeile 12 bei
// 19 bis 20. Dort ist das Pixel ganz von der dunklen Linie (±1) bedeckt, zwei Pixel daneben ganz vom hellen Rand (±3),
// weiter weg bleibt das Bild. Die Kante liegt auch über Objekten.
func TestZackenkanteSollwerte(t *testing.T) {
	d := grauesDokument(t, 100, 40)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(0, 30), document.Pt(100, 40)), Fuellung: blau})
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{
		{20, 4}: document.ZackenLinie, {18, 4}: document.ZackenRand, {22, 4}: document.ZackenRand,
		{19, 12}: document.ZackenLinie, {17, 12}: document.ZackenRand, {21, 12}: document.ZackenRand,
		{20, 36}: document.ZackenLinie, // y 36: wie y 4 (Zahnbreite 16), über dem blauen Kasten
		{10, 4}:  grau, {30, 4}: grau, {12, 12}: grau, {28, 12}: grau, {10, 36}: blau, {30, 36}: blau,
	})
}

// Ohne Naht zeichnet der Renderer keine Kante: ohne Streifen, und bei einem Streifen am Rand der Ansicht.
func TestKeineZackenOhneNaht(t *testing.T) {
	for name, st := range map[string][]document.Streifen{
		"ohne":            nil,
		"am linken Rand":  {{Von: 0, Bis: 10}},
		"am unteren Rand": {{Zeilen: true, Von: 30, Bis: 40}},
	} {
		d := grauesDokument(t, 100, 40)
		streifen(t, d, st...)
		if !bytes.Equal(bild(t, d).Pix, d.Ansichtsbild().Pix) {
			t.Errorf("%s: Bild weicht vom Ansichtsbild ab", name)
		}
	}
}

// Die Kante zählt im Kostenmodell (ADR-0020 Nachtrag 3): Fläche je Band aus dem umschließenden Rechteck. Waagrechte
// Naht bei y 20 auf 100 Pixel Breite: Rand y 13 bis 27 (Rechteck 13 bis 28, 15 Zeilen), Linie y 15 bis 25 (11 Zeilen),
// also 25·100·(15 + 11) = 65 000. Trockenlauf und Rendern zählen dasselbe.
func TestZackenkanteImAufwand(t *testing.T) {
	d := grauesDokument(t, 100, 40)
	if a := trockenlauf(d); a.summe() != 0 {
		t.Fatalf("ohne Naht Aufwand %v", a.summe())
	}
	streifen(t, d, document.Streifen{Zeilen: true, Von: 20, Bis: 25})
	a := trockenlauf(d)
	if a.flaeche != 65000 || a.kanten <= 0 || a.glyph != 0 {
		t.Errorf("Aufwand %+v, Fläche Soll 65000", a)
	}
	_, r, err := bildMitAufwand(d)
	if err != nil {
		t.Fatal(err)
	}
	if r.summe() != a.summe() {
		t.Errorf("Rendern zählt %v, Trockenlauf %v", r.summe(), a.summe())
	}
}

// ADR-0023 Punkt 7: Eine Beschriftung ganz in einem Streifen wird nicht mehr auf der Naht zusammengedrückt gezeichnet
// (vorher: der Rand eines Kastens als Strich auf der Naht); eine, die über die Naht reicht, schon. Kosten zählt die
// verborgene nicht.
func TestVerborgeneBeschriftungNichtGezeichnet(t *testing.T) {
	d := grauesDokument(t, 100, 40)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(25, 5), document.Pt(35, 15)),
		Strich: document.Strich{Farbe: rot, Breite: 2}})
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	b := bild(t, d)
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			if c := b.NRGBAAt(x, y); c.R > 200 && c.G < 100 {
				t.Fatalf("roter Pixel bei (%d,%d): verborgener Kasten gezeichnet", x, y)
			}
		}
	}
	ohne := grauesDokument(t, 100, 40)
	streifen(t, ohne, document.Streifen{Von: 20, Bis: 40})
	if trockenlauf(d).summe() != trockenlauf(ohne).summe() {
		t.Error("verborgener Kasten kostet")
	}
	// Reicht er über die Naht hinaus (bis x 60, Ansicht 40), erscheint er rechts neben der Kante.
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(25, 20), document.Pt(60, 30)), Fuellung: rot})
	if c := bild(t, d).NRGBAAt(32, 25); c != rot {
		t.Errorf("angeschnittener Kasten fehlt: %v", c)
	}
}
