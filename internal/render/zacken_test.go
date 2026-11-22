// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"math"
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

// Die Schichten der Kante als Literal (ADR-0023 Nachtrag 7, F-421): Sollpixel kommen von hier, nicht aus
// document.KantenSchichten — sonst bliebe eine geänderte Tabelle grün. Quer zur Mittellinie bei Faktor 1, negativ oben
// bzw. links; der Spalt ±8 ist durchsichtig und keine Schicht.
var (
	saum30, saum76       = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x30}, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x76}
	dunkelO08, dunkelO13 = color.NRGBA{A: 0x08}, color.NRGBA{A: 0x13}
	dunkelU05, dunkelU0D = color.NRGBA{A: 0x05}, color.NRGBA{A: 0x0D}
	schatten08           = color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0x08}
	schatten10           = color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0x10}
	schatten18           = color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0x18}
	schatten20           = color.NRGBA{R: 0x10, G: 0x12, B: 0x14, A: 0x20}
	durchsichtig         = color.NRGBA{}
)

// schichtTabelle ist die Tabelle aus ADR-0023 Nachtrag 7 (von, bis, Farbe) in Zeichenreihenfolge.
var schichtTabelle = []document.Schicht{
	{Von: -9.6, Bis: -8, Farbe: saum30}, {Von: -8.8, Bis: -8, Farbe: saum76},
	{Von: 8, Bis: 9.6, Farbe: saum30}, {Von: 8, Bis: 8.8, Farbe: saum76},
	{Von: -10, Bis: -8, Farbe: dunkelO08}, {Von: -9, Bis: -8, Farbe: dunkelO13},
	{Von: 8, Bis: 10, Farbe: dunkelU05}, {Von: 8, Bis: 9, Farbe: dunkelU0D},
	{Von: -8, Bis: -1, Farbe: schatten08}, {Von: -8, Bis: -4, Farbe: schatten10},
	{Von: -8, Bis: -6, Farbe: schatten18}, {Von: -8, Bis: -7, Farbe: schatten20},
}

// ADR-0023 Nachtrag 7: Kante der Form Zacken an der senkrechten Naht bei x = 20 (Spalten 20 bis 40 heraus). Die
// Mittellinie läuft von x 16 (y 0) nach x 24 (y 8) und zurück nach 16 (y 16); in Zeile 4 liegt sie bei x 20 bis 21,
// das Pixel x überdeckt dort quer zur Mittellinie x−21 bis x−19. Der Spalt (±8) ist durchsichtig, auch über Objekten
// (der blaue Kasten in Zeile 36 = Zeile 4 + 2 Zähne): Pixel 22 (quer 1..3) und 26 (5..7) haben Alpha 0, rechts
// fällt kein Schatten. Den Schatten wirft nur der linke Teil: Pixel 17 (−4..−2) liegt im Band −8..−1, Pixel 15
// (−6..−4) zusätzlich in −8..−4 (die übrigen Grenzen berühren es nur an einer Ecke). Pixel 9 (−12..−10) und 31
// (10..12) bleiben Bild.
func TestKanteSollwerte(t *testing.T) {
	d := zackenDokument(t, 100, 40)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(0, 30), document.Pt(100, 40)), Fuellung: blau})
	streifen(t, d, document.Streifen{Von: 20, Bis: 40})
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{
		{22, 4}: durchsichtig, {26, 4}: durchsichtig, {22, 36}: durchsichtig, {26, 36}: durchsichtig,
		{17, 4}: ueber(durchsichtig, schatten08), {17, 36}: ueber(durchsichtig, schatten08),
		{15, 4}: ueber(durchsichtig, schatten08, schatten10),
		{9, 4}:  grau, {31, 4}: grau, {9, 36}: blau, {31, 36}: blau,
	})
	if a := b.NRGBAAt(15, 4).A; a == 0 || a == 0xFF {
		t.Errorf("Schatten nicht halbdurchsichtig: Alpha %d", a)
	}
}

// Fasersaum und Abdunklung bei Faktor 3 (alle Maße dreifach, die Bänder sind breit genug, ganze Pixel zu decken):
// Spalten 300 bis 320 heraus, Naht bei x = 300; die Mittellinie läuft von x 288 (y 0) nach 312 (y 24), in Zeile 12
// bei x 300 bis 301, das Pixel x überdeckt quer x−301 bis x−299. Pixel 275 (−26..−24) liegt in beiden Saum- und beiden
// Abdunklungsbändern des linken Teils, Pixel 325 (24..26) in denen des rechten; Pixel 277 (−24..−22) im Spalt in allen
// vier Schattenbändern, Pixel 301 (0..2) im Spalt ohne Schatten. Pixel 269 und 331 (±30..±32) bleiben Bild.
func TestKanteFaktor3(t *testing.T) {
	d := zackenDokument(t, 3300, 3300)
	streifen(t, d, document.Streifen{Von: 300, Bis: 320})
	if f := d.ZackenFaktor(); f != 3 {
		t.Fatalf("Faktor %d, Soll 3", f)
	}
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{
		{275, 12}: ueber(grau, saum30, saum76, dunkelO08, dunkelO13),
		{325, 12}: ueber(grau, saum30, saum76, dunkelU05, dunkelU0D),
		{277, 12}: ueber(durchsichtig, schatten08, schatten10, schatten18, schatten20),
		{301, 12}: durchsichtig,
		{269, 12}: grau, {331, 12}: grau,
	})
	// Der Saum hellt auf, die Abdunklung des oberen bzw. linken Teils ist kräftiger als die des unteren bzw. rechten.
	l, r := b.NRGBAAt(275, 12), b.NRGBAAt(325, 12)
	if l.R <= grau.R || r.R <= l.R {
		t.Errorf("Saum links %v, rechts %v, Bild %v", l, r, grau)
	}
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
// (16 Pixel um die Naht bei y 50, ADR-0023 Nachtrag 7).
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
			for _, y := range []int{0, 33, 67, 89} {
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

// Die Kante zählt im Kostenmodell (ADR-0020 Nachtrag 3, ADR-0023 Nachträge 6 und 7): Fläche je Schicht und für den
// Spalt (±8) aus dem umschließenden Rechteck. Waagrechte Naht der Form Zacken bei y 20 auf 100 Pixel Breite: Ein Band
// von..bis reicht von y 16+von bis 24+bis, sein Rechteck von ⌊16+von⌋ bis ⌈24+bis⌉+1; also 25·100·Σ Zeilen.
// Trockenlauf und Rendern zählen dasselbe.
func TestKanteImAufwand(t *testing.T) {
	d := zackenDokument(t, 100, 40)
	if a := trockenlauf(d, false); a.summe() != 0 {
		t.Fatalf("ohne Naht Aufwand %v", a.summe())
	}
	streifen(t, d, document.Streifen{Zeilen: true, Von: 20, Bis: 25})
	a := trockenlauf(d, false)
	zeilen := 0.0
	for _, s := range append([]document.Schicht{{Von: -8, Bis: 8}}, schichtTabelle...) {
		zeilen += math.Ceil(24+s.Bis) + 1 - math.Floor(16+s.Von)
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
	// Reicht er über die Naht hinaus (bis x 60, Ansicht 40), erscheint er rechts neben der Kante (die höchstens 16
	// Pixel über die Naht ragt).
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(25, 20), document.Pt(60, 30)), Fuellung: rot})
	if c := bild(t, d).NRGBAAt(38, 25); c != rot {
		t.Errorf("angeschnittener Kasten fehlt: %v", c)
	}
}
