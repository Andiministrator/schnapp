// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font"

	"github.com/Andiministrator/schnapp/document"
)

var (
	weiss = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	rot   = color.NRGBA{R: 255, A: 255}
	blau  = color.NRGBA{B: 255, A: 255}
)

func weissesDokument(t *testing.T, w, h int) *document.Dokument {
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

func hinzu(t *testing.T, d *document.Dokument, o document.Objekt) {
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
	// Spitze: Länge max(8, 4·2) = 8, halbe Breite 4 am Fuß bei x = 47.
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

// F-118: Text weit außerhalb des Bildes wird nicht gerastert — schnell, und das Bild bleibt unverändert.
func TestTextAusserhalbWirdNichtGerastert(t *testing.T) {
	for _, ort := range []document.Punkt{document.Pt(50000, 50000), document.Pt(-50000, 5), document.Pt(5, -50000), document.Pt(500000, 5)} {
		d := weissesDokument(t, 10, 10)
		hinzu(t, d, document.Text{Ort: ort, Inhalt: strings.Repeat("Wg", 500), Schriftgrad: 4096, Farbe: rot})
		vor := append([]byte(nil), d.Ansichtsbild().Pix...)
		start := time.Now()
		b := bild(t, d)
		if dauer := time.Since(start); dauer > 200*time.Millisecond {
			t.Errorf("Text bei %v: %v gebraucht, soll unter 200 ms bleiben", ort, dauer)
		}
		if !bytes.Equal(b.Pix, vor) {
			t.Errorf("Text bei %v verändert das Bild", ort)
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
		f, err := z.schrift(false, 20)
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
