// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
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

// zackenDokument ist grauesDokument mit der Kantenform Zacken: Ihre Ecken liegen auf ganzen Pixeln, so lassen sich
// Pixel finden, die eine Schicht ganz bedeckt.
func zackenDokument(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	d := grauesDokument(t, w, h)
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
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

// ueber legt Farben mit voller Deckung über c, wie der Renderer (einmischer, draw.Over).
func ueber(c color.NRGBA, schichten ...color.NRGBA) color.NRGBA {
	p := []byte{c.R, c.G, c.B, c.A}
	for _, s := range schichten {
		e := neuerEinmischer(nil, s)
		e.misch(p, 0xffff)
	}
	return color.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}
}

// schicht ist die Farbe der Schicht von..bis (Faktor 1) aus document.KantenSchichten.
func schicht(t *testing.T, von, bis float64) color.NRGBA {
	t.Helper()
	for _, s := range document.KantenSchichten() {
		if s.Von == von && s.Bis == bis {
			return s.Farbe
		}
	}
	t.Fatalf("keine Schicht %v..%v", von, bis)
	return color.NRGBA{}
}

// ADR-0023 Nachtrag 6: Kante der Form Zacken an der senkrechten Naht bei x = 20 (Spalten 20 bis 40 heraus). Die
// Mittellinie läuft von x 16 (y 0) nach x 24 (y 8) und zurück nach 16 (y 16); in Zeile 4 liegt sie bei x 20 bis 21, in
// Zeile 12 bei 19 bis 20. Das Pixel darauf liegt quer zur Mittellinie zwischen −1 und +1: ganz im Spalt und im
// breitesten Schattenband des linken Teils (−5 bis 2), die Grenzen der übrigen Schatten berühren es nur an einer Ecke.
// Sieben Pixel daneben (quer −8..−6 bzw. +6..+8) bleibt das Bild. Die Kante liegt auch über Objekten.
func TestKanteSollwerte(t *testing.T) {
	d := zackenDokument(t, 100, 40)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(0, 30), document.Pt(100, 40)), Fuellung: blau})
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	b := bild(t, d)
	mitte := ueber(document.KantenSpalt, schicht(t, -5, 2))
	pruefe(t, b, map[image.Point]color.NRGBA{
		{20, 4}: mitte, {19, 12}: mitte,
		{20, 36}: mitte, // y 36: wie y 4 (Zahnbreite 16), über dem blauen Kasten
		{13, 4}:  grau, {27, 4}: grau, {12, 12}: grau, {26, 12}: grau, {13, 36}: blau, {27, 36}: blau,
	})
	// Im Spalt: links (oben) dunkler als rechts (unten), Licht von oben links; alles heller als der Schatten selbst.
	links, rechts := b.NRGBAAt(17, 4), b.NRGBAAt(23, 4)
	if links.R >= rechts.R || rechts.R >= document.KantenSpalt.R {
		t.Errorf("Schatten: links %v, rechts %v, Spalt %v", links, rechts, document.KantenSpalt)
	}
}

// Die Abdunklung der Bildkante (1 px bei Faktor 1, 2 px bei Faktor 2) und das Pixel auf der Mittellinie bei Faktor 2:
// Spalten 100 bis 120 heraus, Naht bei x = 100, Mittellinie von x 92 (y 0) nach x 108 (y 16), in Zeile 8 bei x 100
// bis 101. Quer −12..−10 (Pixel 89) ist ganz abgedunkelt, −1..1 (Pixel 100) wie bei Faktor 1, −14..−12 (Pixel 87)
// bleibt das Bild.
func TestKanteFaktor2(t *testing.T) {
	d := zackenDokument(t, 2200, 2200)
	streifen(t, d, document.Streifen{Von: 100, Bis: 120})
	if f := d.ZackenFaktor(); f != 2 {
		t.Fatalf("Faktor %d, Soll 2", f)
	}
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{
		{100, 8}: ueber(document.KantenSpalt, schicht(t, -5, 2)),
		{89, 8}:  ueber(grau, schicht(t, -6, -5)),
		{87, 8}:  grau, {60, 8}: grau,
	})
}

// Ohne Naht zeichnet der Renderer keine Kante: ohne Streifen, und bei einem Streifen am Rand der Ansicht.
func TestKeineKanteOhneNaht(t *testing.T) {
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

// Die drei Formen zeichnen verschieden, der Abriss ist die Voreinstellung; alle bleiben innerhalb der halben Bandhöhe
// (11 Pixel um die Naht bei y 50).
func TestKanteFormen(t *testing.T) {
	bilder := map[document.Kante][]byte{}
	for _, k := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		d := grauesDokument(t, 200, 100)
		if k != document.KanteAbriss {
			if err := d.VorgabeKante(k); err != nil {
				t.Fatal(err)
			}
		}
		streifen(t, d, document.Streifen{Zeilen: true, Von: 50, Bis: 60})
		b := bild(t, d)
		for x := 0; x < 200; x++ {
			for _, y := range []int{0, 38, 62, 89} {
				if c := b.NRGBAAt(x, y); c != grau {
					t.Fatalf("%v: Pixel (%d,%d) = %v außerhalb der Kante", k, x, y, c)
				}
			}
		}
		bilder[k] = b.Pix
	}
	if bytes.Equal(bilder[document.KanteAbriss], bilder[document.KanteWelle]) ||
		bytes.Equal(bilder[document.KanteWelle], bilder[document.KanteZacken]) ||
		bytes.Equal(bilder[document.KanteAbriss], bilder[document.KanteZacken]) {
		t.Error("zwei Formen zeichnen gleich")
	}
}

// Die Kante zählt im Kostenmodell (ADR-0020 Nachtrag 3, ADR-0023 Nachtrag 6): Fläche je Schicht aus dem
// umschließenden Rechteck. Waagrechte Naht der Form Zacken bei y 20 auf 100 Pixel Breite: Eine Schicht von..bis reicht
// von y 16+von bis 24+bis, ihr Rechteck hat 9+(bis−von) Zeilen (ganze Grenzen); also 25·100·Σ(9+bis−von). Trockenlauf
// und Rendern zählen dasselbe.
func TestKanteImAufwand(t *testing.T) {
	d := zackenDokument(t, 100, 40)
	if a := trockenlauf(d, false); a.summe() != 0 {
		t.Fatalf("ohne Naht Aufwand %v", a.summe())
	}
	streifen(t, d, document.Streifen{Zeilen: true, Von: 20, Bis: 25})
	a := trockenlauf(d, false)
	zeilen := 0.0
	for _, s := range document.KantenSchichten() {
		zeilen += 9 + s.Bis - s.Von
	}
	if soll := 25 * 100 * zeilen; a.flaeche != soll || a.kanten <= 0 || a.glyph != 0 {
		t.Errorf("Aufwand %+v, Fläche Soll %v", a, soll)
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
	if trockenlauf(d, false).summe() != trockenlauf(ohne, false).summe() {
		t.Error("verborgener Kasten kostet")
	}
	// Reicht er über die Naht hinaus (bis x 60, Ansicht 40), erscheint er rechts neben der Kante (die höchstens 11
	// Pixel über die Naht ragt).
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(25, 20), document.Pt(60, 30)), Fuellung: rot})
	if c := bild(t, d).NRGBAAt(34, 25); c != rot {
		t.Errorf("angeschnittener Kasten fehlt: %v", c)
	}
}
