// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-108, F-498: Ein Kasten mit Eckradius > 0 hat runde Außenecken (Strich mit runden Ecken); das SVG sagt dasselbe mit
// stroke-linejoin="round" (TestSVGMasseWieRenderer). Ohne Radius gilt die Gehrung (TestKastenEckeSpitz).
func TestKastenEckeRund(t *testing.T) {
	d := weissesDokument(t, 100, 100)
	strich := document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 12}
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(30, 30), document.Pt(80, 70)), Strich: strich, Eckradius: 6})
	b := bild(t, d)
	// Abstand des Pixels (24,24) zur Ecke (30,30): 8,5 > 6 — eine Gehrung deckte es ab, ein runder Strich nicht.
	if c := b.NRGBAAt(24, 24); c != weiss {
		t.Errorf("Eckpixel %v, Soll weiß (runde Ecke)", c)
	}
	if c := b.NRGBAAt(28, 50); c.R > 20 {
		t.Errorf("Pixel im Strich %v, Soll dunkel", c)
	}
}

// F-498: Ein Kasten mit Eckradius 0 hat spitze Außenecken (Gehrung); das SVG sagt stroke-linejoin="miter". Strich 8:
// das Eckpixel (Rand − 4, Rand − 4) ist gedeckt, die Innenecke und das Innere bleiben weiß, Ausdehnung wie Grenzen().
func TestKastenEckeSpitz(t *testing.T) {
	d := weissesDokument(t, 100, 100)
	strich := document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 8}
	k := document.Kasten{Bereich: document.R(document.Pt(30, 30), document.Pt(80, 70)), Strich: strich}
	hinzu(t, d, k)
	b := bild(t, d)
	for _, p := range [][2]int{{26, 26}, {83, 26}, {26, 73}, {83, 73}} {
		if c := b.NRGBAAt(p[0], p[1]); c.A != 255 || c.R > 20 {
			t.Errorf("Außeneckpixel %v: %v, Soll dunkel (spitze Ecke)", p, c)
		}
	}
	if c := b.NRGBAAt(25, 25); c != weiss {
		t.Errorf("Pixel außerhalb der Grenzen %v, Soll weiß", c)
	}
	if c := b.NRGBAAt(40, 40); c != weiss {
		t.Errorf("Inneres %v, Soll weiß", c)
	}
	if g := k.Grenzen(); g.Min.X != 26 || g.Max.Y != 74 {
		t.Errorf("Grenzen %v", g)
	}
}

// F-498: Ist der Innenraum durch den Strich verschwunden, füllt der eckige Rahmen das ganze Rechteck.
func TestKastenEckeSpitzSchmal(t *testing.T) {
	d := weissesDokument(t, 100, 100)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(40, 40), document.Pt(46, 80)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 8}})
	b := bild(t, d)
	if c := b.NRGBAAt(43, 60); c.R > 20 {
		t.Errorf("Mitte des schmalen Kastens %v, Soll dunkel", c)
	}
	if c := b.NRGBAAt(37, 37); c.R > 20 {
		t.Errorf("Außenecke %v, Soll dunkel", c)
	}
}

// F-108: Der Eckradius wird wie im SVG begrenzt: höchstens die halbe kürzere Seite, kreisförmig. Ein 40×20-Kasten mit
// Radius 100 hat Ecken mit Radius 10 (ein Halbkreis an beiden schmalen Seiten), keine Ellipse.
func TestKastenEckradiusBegrenzt(t *testing.T) {
	d := weissesDokument(t, 100, 100)
	hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(20, 40), document.Pt(60, 60)), Fuellung: rot, Eckradius: 100})
	b := bild(t, d)
	if c := b.NRGBAAt(20, 40); c != weiss {
		t.Errorf("Ecke %v, Soll weiß (gerundet)", c)
	}
	if c := b.NRGBAAt(30, 41); c != rot {
		t.Errorf("Pixel an der oberen Kante nach dem Bogen %v, Soll rot", c)
	}
	if c := b.NRGBAAt(21, 50); c.G > 60 {
		t.Errorf("Pixel an der linken Mitte %v, Soll rot (Halbkreis reicht bis zur Kante)", c)
	}
}

// F-108: Die Ziffern einer Nummer stehen auf der Grundlinie, die document.Nummer.Grundlinie angibt (Editor und
// SVG-Export nutzen dieselbe Zahl). „1“ hat keine Unterlänge: die unterste weiße Zeile liegt direkt über der Grundlinie.
func TestNummerGrundlinie(t *testing.T) {
	d := weissesDokument(t, 200, 200)
	n := document.Nummer{Mitte: document.Pt(100, 100), Wert: 1, Radius: 60, Farbe: rot}
	hinzu(t, d, n)
	b := bild(t, d)
	unten := -1
	for y := 60; y < 140; y++ {
		for x := 80; x < 120; x++ {
			if c := b.NRGBAAt(x, y); c.G > 200 && c.B > 200 {
				unten = y
			}
		}
	}
	if unten < 0 {
		t.Fatal("keine Ziffer gefunden")
	}
	if soll := n.Grundlinie(); math.Abs(float64(unten+1)-soll) > 1 {
		t.Errorf("Ziffer endet bei %d, Grundlinie laut Dokument %.2f", unten+1, soll)
	}
}
