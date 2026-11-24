// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// fastGleich lässt 2 Stufen je Kanal zu: Der Rasterkern rechnet relativ zum Ausschnitt in float32; ein auf den Bereich
// beschnittener Ausschnitt beginnt anderswo und rundet die Kantenglättung um eine Stufe anders. Eine falsch gezogene
// Kante weicht um viele Stufen ab.
func fastGleich(a, b color.NRGBA) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= 2 && int(y)-int(x) <= 2 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

// vergleicheUmNaht rendert o einmal ohne und einmal mit dem Spaltenstreifen [von, bis) und zählt die abweichenden
// Pixel links und rechts der Naht außerhalb des Kantenbands; rechts wird um die Streifenbreite verschoben verglichen.
// farbig zählt die Pixel in Objektfarbe rechts der Naht (der Vergleich soll das Objekt dort sehen).
func vergleicheUmNaht(t *testing.T, o document.Objekt, von, bis int, farbe color.NRGBA) (links, rechts, farbig int) {
	t.Helper()
	baue := func() *document.Dokument {
		b := image.NewNRGBA(image.Rect(0, 0, 400, 200))
		draw.Draw(b, b.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
		return d
	}
	voll, err := Bild(baue())
	if err != nil {
		t.Fatal(err)
	}
	d := baue()
	if err := d.SetzeStreifen([]document.Streifen{{Von: von, Bis: bis}}); err != nil {
		t.Fatal(err)
	}
	gestaucht, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	band, breite := int(document.KantenBandHalb)+4, bis-von
	for y := 0; y < 200; y++ {
		for x := 0; x < von-band; x++ {
			if !fastGleich(gestaucht.NRGBAAt(x, y), voll.NRGBAAt(x, y)) {
				links++
			}
		}
		for x := von + band; x < 400-breite; x++ {
			g := gestaucht.NRGBAAt(x, y)
			if !fastGleich(g, voll.NRGBAAt(x+breite, y)) {
				rechts++
			}
			if g == farbe {
				farbig++
			}
		}
	}
	return
}

// F-695 (ADR-0023 Nachtrag 14): Das Dreieck einer Sprechblase, das über eine Naht reicht, ist dort abgeschnitten wie im
// SVG-Export und im Treffer; vorher zog der Renderer es zwischen den abgebildeten Ecken neu (841 von 896 Pixeln
// abweichend, Kritik A-4).
func TestSprechblasenDreieckUeberNaht(t *testing.T) {
	fuell := color.NRGBA{R: 30, G: 90, B: 200, A: 255}
	sp := document.Pt(330, 170)
	for _, mitFuellung := range []bool{true, false} {
		tx := document.Text{Ort: document.Pt(20, 20), Inhalt: "Hi", Schriftgrad: 40, Farbe: color.NRGBA{A: 255},
			Hintergrund: fuell, Spitze: &sp}
		farbe := fuell
		if !mitFuellung {
			tx.Hintergrund = color.NRGBA{}
			farbe = tx.Farbe
		}
		links, rechts, farbig := vergleicheUmNaht(t, tx, 150, 190, farbe)
		if farbig == 0 {
			t.Fatalf("Füllung %v: rechts der Naht kein Dreieck", mitFuellung)
		}
		if links != 0 || rechts != 0 {
			t.Errorf("Füllung %v: Dreieck über der Naht nicht abgeschnitten: links %d, rechts %d Pixel abweichend",
				mitFuellung, links, rechts)
		}
	}
}

// F-695: Die Verbindungslinie einer Lupe über einer Naht ist dort abgeschnitten wie im SVG-Export.
func TestLupenLinieUeberNaht(t *testing.T) {
	gruen := color.NRGBA{G: 160, A: 255}
	l := document.Lupe{Quelle: document.R(document.Pt(20, 20), document.Pt(50, 50)),
		Ziel: document.R(document.Pt(280, 110), document.Pt(340, 170)), Strich: document.Strich{Farbe: gruen, Breite: 3}}
	links, rechts, farbig := vergleicheUmNaht(t, l, 150, 190, gruen)
	if farbig == 0 {
		t.Fatal("rechts der Naht keine Lupe")
	}
	if links != 0 || rechts != 0 {
		t.Errorf("Lupen-Linie über der Naht nicht abgeschnitten: links %d, rechts %d Pixel abweichend", links, rechts)
	}
}

// F-745 (ADR-0023 Nachtrag 15): Pfeil und Schritt-Nummer über einer Naht sind dort abgeschnitten wie im SVG-Export,
// nicht gestaucht bzw. starr an der abgebildeten Mitte.
func TestPfeilUndNummerUeberNaht(t *testing.T) {
	rot := color.NRGBA{R: 200, A: 255}
	for name, o := range map[string]document.Objekt{
		"Pfeil":  document.GeraderPfeil(document.Pt(20, 40), document.Pt(380, 160), document.Strich{Farbe: rot, Breite: 4}),
		"Nummer": document.Nummer{Mitte: document.Pt(170, 100), Wert: 7, Radius: 45, Farbe: rot},
	} {
		links, rechts, farbig := vergleicheUmNaht(t, o, 150, 190, rot)
		if name == "Pfeil" && farbig == 0 {
			t.Fatalf("%s: rechts der Naht nichts", name)
		}
		if links != 0 || rechts != 0 {
			t.Errorf("%s über der Naht nicht abgeschnitten: links %d, rechts %d Pixel abweichend", name, links, rechts)
		}
	}
}
