// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/vector"

	"github.com/Andiministrator/schnapp/internal/raster16"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

var (
	weiss = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	rot   = color.NRGBA{R: 255, A: 255}
	blau  = color.NRGBA{B: 255, A: 255}
)

func weissesDokument(t testing.TB, w, h int) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range b.Pix {
		b.Pix[i] = 255
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func hinzu(t testing.TB, d *document.Dokument, o document.Objekt) {
	t.Helper()
	if _, err := d.Hinzufuegen(o); err != nil {
		t.Fatal(err)
	}
}

func bild(t *testing.T, d *document.Dokument) *image.NRGBA {
	t.Helper()
	b, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// pruefe vergleicht Sollpixel; nur deckende Pixel in der Mitte von Flächen, Kanten sind geglättet.
func pruefe(t *testing.T, b *image.NRGBA, soll map[image.Point]color.NRGBA) {
	t.Helper()
	for p, c := range soll {
		if got := b.NRGBAAt(p.X, p.Y); got != c {
			t.Errorf("Pixel %v = %v, Soll %v", p, got, c)
		}
	}
}

func TestKasten(t *testing.T) {
	d := weissesDokument(t, 60, 60)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(30, 30)), Fuellung: rot})
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(35, 35), document.Pt(55, 55)),
		Strich: document.Strich{Farbe: blau, Breite: 4}})
	pruefe(t, bild(t, d), map[image.Point]color.NRGBA{
		{20, 20}: rot, {5, 5}: weiss, {31, 20}: weiss, // gefüllt, außen
		{35, 45}: blau, {45, 36}: blau, {45, 45}: weiss, // Rand um die Kante, Inneres frei
		{40, 32}: weiss, // außerhalb der halben Strichbreite
	})
}

func TestPfeilMitSpitze(t *testing.T) {
	d := weissesDokument(t, 60, 40)
	hinzu(t, d, document.GeraderPfeil(document.Pt(5, 20), document.Pt(55, 20), document.Strich{Farbe: rot, Breite: 2}))
	// Spitze (ADR-0021 Nachtrag 19): Länge 3·2+5 = 11, halbe Breite 7,5 am Fuß bei x = 44.
	pruefe(t, bild(t, d), map[image.Point]color.NRGBA{
		{20, 20}: rot, {20, 23}: weiss, // Schaft 2 px breit
		{48, 22}: rot, {48, 17}: rot, // Spitze breiter als der Schaft
		{30, 10}: weiss, {57, 20}: weiss,
	})
}

func TestGebogenerPfeilFolgtDerKurve(t *testing.T) {
	d := weissesDokument(t, 100, 100)
	p := document.Pfeil{A: document.Pt(10, 80), B: document.Pt(90, 80), Strich: document.Strich{Farbe: rot, Breite: 4}}
	p = p.MitGriff(document.Pt(50, 20))
	hinzu(t, d, p)
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{{50, 20}: rot, {50, 80}: weiss})
}

func TestNummer(t *testing.T) {
	d := weissesDokument(t, 60, 60)
	hinzu(t, d, document.Nummer{Mitte: document.Pt(30, 30), Radius: 20, Wert: 7, Farbe: blau})
	b := bild(t, d)
	pruefe(t, b, map[image.Point]color.NRGBA{{30, 14}: blau, {14, 30}: blau, {5, 5}: weiss})
	// Die Ziffer steht weiß in der Mitte: im inneren Quadrat gibt es helle Pixel, im Kreisrand nicht.
	hell := 0
	for y := 22; y < 38; y++ {
		for x := 22; x < 38; x++ {
			if c := b.NRGBAAt(x, y); c.R > 200 && c.G > 200 {
				hell++
			}
		}
	}
	if hell < 10 {
		t.Errorf("keine Ziffer in der Mitte: %d helle Pixel", hell)
	}
}

func TestTextMitHintergrund(t *testing.T) {
	d := weissesDokument(t, 200, 60)
	tx := document.Text{Ort: document.Pt(10, 10), Inhalt: "Hallo", Schriftgrad: 20, Farbe: weiss, Hintergrund: blau}
	hinzu(t, d, tx)
	b := bild(t, d)
	g := tx.Grenzen() // 10..70 × 10..36
	pruefe(t, b, map[image.Point]color.NRGBA{{11, 11}: blau, {int(g.Max.X) - 1, int(g.Max.Y) - 1}: blau, {150, 50}: weiss})
	// Schrift: im Hintergrund stehen helle Glyphen-Pixel, rechts davon nicht.
	hell := func(x0, x1 int) int {
		n := 0
		for y := 12; y < 34; y++ {
			for x := x0; x < x1; x++ {
				if c := b.NRGBAAt(x, y); c.R > 200 && c.B > 200 {
					n++
				}
			}
		}
		return n
	}
	if n := hell(12, 68); n < 30 {
		t.Errorf("Text fehlt: %d helle Pixel", n)
	}
}

// Objekte folgen dem Zuschnitt: Sie liegen im Original und werden in die Ansicht abgebildet.
func TestZuschnittVerschiebtObjekte(t *testing.T) {
	d := weissesDokument(t, 80, 80)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(40, 40), document.Pt(60, 60)), Fuellung: rot})
	if err := d.SetzeZuschnitt(image.Rect(20, 20, 80, 80)); err != nil {
		t.Fatal(err)
	}
	b := bild(t, d)
	if b.Bounds().Size() != image.Pt(60, 60) {
		t.Fatalf("Größe %v", b.Bounds().Size())
	}
	pruefe(t, b, map[image.Point]color.NRGBA{{30, 30}: rot, {50, 50}: weiss, {15, 15}: weiss})
}

// Halbdurchsichtige Füllung mischt mit dem Bild darunter.
func TestDeckkraft(t *testing.T) {
	d := weissesDokument(t, 20, 20)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(20, 20)), Fuellung: color.NRGBA{A: 128}})
	c := bild(t, d).NRGBAAt(10, 10)
	if c.A != 255 || c.R < 120 || c.R > 135 {
		t.Errorf("Mischung %v, Soll Grau um 127", c)
	}
}

// Ohne Objekte ist das Ergebnis genau das Ansichtsbild.
func TestOhneObjekte(t *testing.T) {
	d := weissesDokument(t, 7, 5)
	b := bild(t, d)
	if b.Bounds() != image.Rect(0, 0, 7, 5) || b.NRGBAAt(3, 3) != weiss {
		t.Errorf("%v %v", b.Bounds(), b.NRGBAAt(3, 3))
	}
}

// BenchmarkGrossesBild: 4K mit einer Handvoll Objekten (Maß für Speichern im Editor).
func BenchmarkGrossesBild(b *testing.B) {
	bild := image.NewNRGBA(image.Rect(0, 0, 3840, 2160))
	d, _ := document.Neu(bild)
	for i := 0; i < 10; i++ {
		f := float64(i * 300)
		_, _ = d.Hinzufuegen(document.GeraderPfeil(document.Pt(100+f, 100), document.Pt(300+f, 900), document.Strich{Farbe: rot, Breite: 6}))
		_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(f, 1200), document.Pt(f+250, 1500)), Strich: document.Strich{Farbe: blau, Breite: 4}})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
}

// Messungen hinter dem Kostenmodell und den Budget-Zahlen (F-140). Sie laufen nur mit -bench, nicht in go test, und
// brauchen Speicher (8000×8000: 256 MB Bild plus 512 MB Puffer des Rasterizers). Jede meldet ns/Einheit (Rechenzeit je
// Einheit des Kostenmodells) und Budget/Aufwand. Aufruf etwa:
//
//	GOFLAGS=-tags=novulkan,nox11 go test -run '^$' -bench 'Aufwand' -benchtime 1x ./internal/render
func messeAufwand(b *testing.B, d *document.Dokument) {
	b.Helper()
	a := Aufwand(d)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Bild(d); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/a, "ns/Einheit")
	b.ReportMetric(Budget/a, "Budget/Aufwand")
}

// Das alltagsnahe große Dokument (alltagsDokument, F-131): Messung 20.11.2026 rund 10,5 s, seit ADR-0028 rund 1,9 s; Budget/Aufwand 3,1.
func BenchmarkAufwandAlltag(b *testing.B) {
	messeAufwand(b, alltagsDokument(b))
}

// Füllen (F-130): zwei gefüllte, umrandete Kästen auf 8000×8000. Messung 20.11.2026 rund 1,9 s (vorher 7,8 s), seit ADR-0028 rund 0,49 s.
func BenchmarkAufwandZweiKaesten8000(b *testing.B) {
	d := weissesDokument(b, 8000, 8000)
	for i := 0; i < 2; i++ {
		hinzu(b, d, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(8000, 8000)), Fuellung: rot,
			Strich: document.Strich{Farbe: blau, Breite: 4}})
	}
	messeAufwand(b, d)
}

// Ein diagonaler Pfeil über 8000×8000: Das umschließende Rechteck ist bildgroß. Messung 20.11.2026 rund 0,55 s
// (vorher 2,4 s), seit ADR-0028 rund 0,21 s.
func BenchmarkAufwandPfeilDiagonal8000(b *testing.B) {
	d := weissesDokument(b, 8000, 8000)
	hinzu(b, d, document.GeraderPfeil(document.Pt(5, 5), document.Pt(7990, 7990), document.Strich{Farbe: rot, Breite: 6}))
	messeAufwand(b, d)
}

// Knapp unter Budget, fast nur Fläche: 39 gefüllte, umrandete Kästen auf 8000×8000. Die Wartezeit, die eine
// präparierte Datei beim Speichern erzwingen kann (F-135). Messung 20.11.2026 rund 36 s, seit ADR-0028 rund 7 s.
func BenchmarkAufwandNaheBudgetFlaeche(b *testing.B) {
	d := weissesDokument(b, 8000, 8000)
	for i := 0; i < 39; i++ {
		hinzu(b, d, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(8000, 8000)), Fuellung: rot,
			Strich: document.Strich{Farbe: blau, Breite: 4}})
	}
	messeAufwand(b, d)
}

// Glyphen knapp unter GlyphBudget, sichtbar auf großem Bild (177 × „W“ Grad 4096 auf 8000×8000, 80 Orte, darüber
// übereinander; vor F-589 59 Stück bei GlyphBudget 1e9).
func BenchmarkAufwandNaheGlyphBudget(b *testing.B) {
	d := weissesDokument(b, 8000, 8000)
	for i := 0; i < 177; i++ {
		hinzu(b, d, document.Text{Ort: document.Pt(float64((i%80)*100), 0), Inhalt: "W", Schriftgrad: 4096, Farbe: rot})
	}
	messeAufwand(b, d)
}

// Kanten knapp unter KantenBudget: sehr breite Pfeile weit außerhalb eines winzigen Bildes (F-132). 429 Pfeile sind
// 9,984e9 Kanten-Einheiten (99,84 % des Budgets; Aufwand gesamt 9,986e9), 430 lehnt Pruefe ab (gemessen 25.11.2026, F-806; bis ADR-0023
// Nachtrag 22 lag die Grenze bei 171, weil die Kreise an den Gelenken viel zu teuer zählten). Bild rund 3,6 bis 3,8 s.
func BenchmarkAufwandNaheKantenBudget(b *testing.B) {
	d := weissesDokument(b, 10, 10)
	for i := 0; i < 429; i++ {
		hinzu(b, d, document.Pfeil{A: document.Pt(-50, -50+float64(i%7)), B: document.Pt(70, 70), K: document.Pt(35, 3),
			Strich: document.Strich{Farbe: rot, Breite: 1e5}})
	}
	messeAufwand(b, d)
}

// referenzEinmischen ist der Weg vor F-130: vector zeichnet über den allgemeinen Weg mit draw.Over ins NRGBA-Bild.
// Seit ADR-0028 baut der Test die Pfade in x/image/vector selbst nach, der Renderer nutzt raster16.
func referenzEinmischen(_ *raster16.Rasterizer, pfade func(pfad), bild *image.NRGBA, ausschnitt image.Rectangle, farbe color.NRGBA) {
	// Mindestens 513 hoch: vector rechnet dann im Gleitkomma-Weg wie raster16 seit F-764 immer, bei gleicher Breite in
	// derselben Reihenfolge; gezeichnet wird nur der Ausschnitt.
	r := vector.NewRasterizer(ausschnitt.Dx(), max(ausschnitt.Dy(), 513))
	pfade(r)
	r.DrawOp = draw.Over
	r.Draw(bild, ausschnitt, image.NewUniform(farbe), image.Point{})
}

// F-130: Der schnelle Weg (einmischer) liefert bitgleich dieselben Pixel wie vector mit draw.Over — auf deckendem und
// auf teils durchsichtigem Grund (auch Alpha 0 mit Farbe), mit halbdurchsichtigen Farben, angeschnittenen Formen,
// Text, Streifen und Ausschnitten über 512 Pixel (Gleitkomma-Weg von vector).
func TestFuellenBitgleich(t *testing.T) {
	alt, grenzeAlt := einmischen, maskenCacheGrenze
	defer func() { einmischen, maskenCacheGrenze = alt, grenzeAlt }()
	bunt := func(w, h int) *document.Dokument {
		b := image.NewNRGBA(image.Rect(0, 0, w, h))
		for i := range b.Pix {
			b.Pix[i] = uint8(i*7 + i/4*13 + i/(4*w)*29)
		}
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	gross := func(d *document.Dokument) *document.Dokument {
		w, h := float64(d.Groesse().X), float64(d.Groesse().Y)
		halb := color.NRGBA{R: 200, G: 30, B: 90, A: 77}
		for i := 0; i < 8; i++ {
			f := float64(i)
			hinzu(t, d, document.Pfeil{A: document.Pt(-20+f*13, 3+f*31), B: document.Pt(w-f*41, h-f*7), K: document.Pt(w/2, f*40),
				Strich: document.Strich{Farbe: []color.NRGBA{rot, halb, blau}[i%3], Breite: 0.7 + f*2.3}})
			hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(-30+f*90, -10+f*60), document.Pt(200+f*110, 150+f*70)),
				Eckradius: f * 7, Fuellung: []color.NRGBA{halb, rot, {G: 255, A: 1}, {A: 254}}[i%4],
				Strich: document.Strich{Farbe: halb, Breite: f * 1.5}})
			hinzu(t, d, document.Nummer{Mitte: document.Pt(30+f*95, 40+f*50), Radius: 4 + f*6, Wert: i * 37, Farbe: halb})
		}
		hinzu(t, d, document.Text{Ort: document.Pt(-10, 20), Inhalt: "Füllen\tbitgleich\nzweite Zeile", Schriftgrad: 27,
			Farbe: halb, Hintergrund: color.NRGBA{R: 10, G: 250, B: 250, A: 130}})
		return d
	}
	streifen := gross(weissesDokument(t, 900, 600))
	if err := streifen.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 100, Bis: 180}, {Zeilen: true, Von: 50, Bis: 90}}); err != nil {
		t.Fatal(err)
	}
	for name, d := range map[string]*document.Dokument{
		"gemischt":              gemischtesDokument(t),
		"deckend 900×600":       gross(weissesDokument(t, 900, 600)),
		"durchsichtig 300×200":  gross(bunt(300, 200)),
		"durchsichtig 1100×700": gross(bunt(1100, 700)),
		"Streifen":              streifen,
	} {
		einmischen = referenzEinmischen
		maskenCacheGrenze = 0 // der Zwischenspeicher (F-430) umginge die Nahtstelle bei großen Füllungen
		soll := bild(t, d)
		maskenCacheGrenze = grenzeAlt
		einmischen = alt
		ist := bild(t, d)
		if !bytes.Equal(soll.Pix, ist.Pix) {
			n := 0
			for i := range soll.Pix {
				if soll.Pix[i] != ist.Pix[i] {
					n++
				}
			}
			t.Errorf("%s: %d Bytes weichen von draw.Over ab", name, n)
		}
	}
}

// Tabulator und Wagenrücklauf erscheinen nicht als Ersatzkästchen (Kritik A-3).
func TestSteuerzeichenOhneKaestchen(t *testing.T) {
	leer := weissesDokument(t, 120, 40)
	mitTab := weissesDokument(t, 120, 40)
	hinzu(t, mitTab, document.Text{Ort: document.Pt(5, 5), Inhalt: "\t\r\x7f\u0085", Schriftgrad: 20, Farbe: rot})
	a, b := bild(t, leer), bild(t, mitTab)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatal("Tabulator/Wagenrücklauf zeichnen etwas")
		}
	}
}

// Ein langer Text, der links aus dem Bild ragt, zeigt seine sichtbaren Glyphen an derselben Stelle wie ohne Überspringen:
// Beginnt der Text weit links, steht an Position x dieselbe Glyphe wie bei einem Text, der um eine Zeichenfolge kürzer ist.
func TestTextAnschnittLinksBleibtAmSelbenOrt(t *testing.T) {
	d := weissesDokument(t, 60, 30)
	hinzu(t, d, document.Text{Ort: document.Pt(-300, 2), Inhalt: strings.Repeat("H", 20) + "AB", Schriftgrad: 20, Farbe: rot})
	b := bild(t, d)
	// Sollwert: ohne Überspringen der Text ab der 20. Glyphe (H, A, B), um den Vorschub der 19 H davor nach rechts versetzt, ergibt dieselben Pixel.
	adv := float64(0)
	{
		z := zeichner{d: d, faces: map[faceSchluessel]font.Face{}}
		defer z.schliessen()
		f, err := z.schrift(fonts.Laden(fonts.Go, fonts.Normal), 20)
		if err != nil {
			t.Fatal(err)
		}
		adv = float64(font.MeasureString(f, strings.Repeat("H", 19))) / 64
	}
	e := weissesDokument(t, 60, 30)
	hinzu(t, e, document.Text{Ort: document.Pt(-300+adv, 2), Inhalt: "HAB", Schriftgrad: 20, Farbe: rot})
	if !bytes.Equal(b.Pix, bild(t, e).Pix) {
		t.Error("sichtbarer Textrest weicht ab")
	}
	rot := 0
	for i := 0; i < len(b.Pix); i += 4 {
		if b.Pix[i+1] < 200 {
			rot++
		}
	}
	if rot < 20 {
		t.Errorf("Textrest fehlt: %d rote Pixel", rot)
	}
}
