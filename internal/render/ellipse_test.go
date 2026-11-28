// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0047: Die Ellipse als Form des Kastens. Füllung nur im Oval, der Strich auf dem Umriss deckt halbdurchsichtig jedes
// Pixel höchstens einmal (voll und gemustert, auch flach mit dickem Strich), gemustert gibt es Lücken.

func TestEllipseFuelltNurDasOval(t *testing.T) {
	rot := color.NRGBA{R: 220, A: 255}
	k := document.Kasten{Bereich: document.R(document.Pt(40, 30), document.Pt(260, 190)), Form: document.FormEllipse,
		Fuellung: rot}
	d := weissesDokument(t, 300, 220)
	hinzu(t, d, k)
	b := bild(t, d)
	if c := b.NRGBAAt(150, 110); c.R != 220 || c.G != 0 {
		t.Errorf("Mitte %v", c)
	}
	for _, p := range [][2]int{{150, 32}, {258, 110}, {150, 188}, {42, 110}} { // knapp innen an den Scheiteln
		if c := b.NRGBAAt(p[0], p[1]); c.G > 40 {
			t.Errorf("innen am Scheitel %v: %v", p, c)
		}
	}
	for _, p := range [][2]int{{45, 35}, {255, 35}, {255, 185}, {45, 185}} { // Ecken des Bereichs: außerhalb des Ovals
		if c := b.NRGBAAt(p[0], p[1]); c.G != 255 {
			t.Errorf("Ecke %v gefüllt: %v", p, c)
		}
	}
}

func TestEllipseStrichDecktEinmal(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichVoll, document.StrichGestrichelt, document.StrichGepunktet} {
		for _, fall := range []struct {
			r      document.Rechteck
			breite float64
		}{
			{document.R(document.Pt(40.5, 30.25), document.Pt(260.25, 190.5)), 6},
			{document.R(document.Pt(60, 20), document.Pt(240, 200)), 13},   // Kreis
			{document.R(document.Pt(20, 95), document.Pt(280, 125)), 24},   // flach, Strich dicker als die Krümmung innen
			{document.R(document.Pt(140, 10), document.Pt(152.5, 210)), 3}, // schmal und hoch
		} {
			name := fmt.Sprintf("%s/%v/Strich %v", art.Name(), fall.r, fall.breite)
			k := document.Kasten{Bereich: fall.r, Form: document.FormEllipse,
				Strich: document.Strich{Farbe: halbSchwarz, Breite: fall.breite, Art: art}}
			d := weissesDokument(t, 300, 220)
			hinzu(t, d, k)
			b := bild(t, d)
			if g := dunkelstesPixel(b); g < 125 {
				t.Errorf("%s: dunkelstes Pixel %d: Teile überlappen", name, g)
			}
			// Am rechten Scheitel liegt Strich (voll und an jedem Scheitel ein halber Strich bzw. ein Punkt).
			x, y := int(math.Floor(fall.r.Max.X)), int(math.Floor((fall.r.Min.Y+fall.r.Max.Y)/2))
			if g := b.NRGBAAt(x, y).R; g > 140 {
				t.Errorf("%s: rechter Scheitel nicht gedeckt (%d)", name, g)
			}
		}
	}
}

func TestEllipseGemustertHatLuecken(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		r := document.R(document.Pt(40, 30), document.Pt(260, 190))
		k := document.Kasten{Bereich: r, Form: document.FormEllipse, Strich: document.Strich{Farbe: halbSchwarz, Breite: 6, Art: art}}
		d := weissesDokument(t, 300, 220)
		hinzu(t, d, k)
		b := bild(t, d)
		weiss, gedeckt := 0, 0
		for _, p := range document.EllipsenUmriss(r) {
			switch g := b.NRGBAAt(int(math.Floor(p.X)), int(math.Floor(p.Y))).R; {
			case g == 255:
				weiss++
			case g < 140:
				gedeckt++
			}
		}
		if weiss < 20 || gedeckt < 20 {
			t.Errorf("%s: auf dem Umriss %d weiß, %d gedeckt", art.Name(), weiss, gedeckt)
		}
	}
}

// Der volle Strich ist ein geschlossener Linienzug (kein Anfang mit Kappen), gemustert offene Stücke (ADR-0047 Punkt 2).
func TestZerlegeEllipseGeschlossen(t *testing.T) {
	r := document.R(document.Pt(10, 10), document.Pt(110, 70))
	if sz := ZerlegeEllipse(r, document.Strich{Breite: 4}); !sz.Zu || len(sz.Zug) < 16 {
		t.Errorf("voll: zu %v, %d Punkte", sz.Zu, len(sz.Zug))
	}
}

// Ohne Fläche zeichnet die Ellipse wie der Kasten ohne Fläche einen Balken (ADR-0047 Punkt 2).
func TestEllipseOhneFlaecheWieKasten(t *testing.T) {
	s := document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 4}
	r := document.R(document.Pt(40, 100), document.Pt(260, 100))
	d1, d2 := weissesDokument(t, 300, 220), weissesDokument(t, 300, 220)
	hinzu(t, d1, document.Kasten{Bereich: r, Strich: s})
	hinzu(t, d2, document.Kasten{Bereich: r, Strich: s, Form: document.FormEllipse})
	b1, b2 := bild(t, d1), bild(t, d2)
	for i := range b1.Pix {
		if b1.Pix[i] != b2.Pix[i] {
			t.Fatalf("Ellipse ohne Fläche anders als der Kasten (Byte %d)", i)
		}
	}
}
