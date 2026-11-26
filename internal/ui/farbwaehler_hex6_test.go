// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #318 (ADR-0021 Nachtrag 17): Das Hex-Feld zeigt immer sechs Ziffern, die Deckkraft steht nur im Feld daneben;
// acht Ziffern werden weiter angenommen. Die Pipette steht rechts in der Zeile des Farbpaars, das freie Feld im Raster
// nimmt eine vierte zuletzt benutzte Farbe.

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

func TestHexFeldOhneDeckkraft(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.e.s.SetzeFarbe(color.NRGBA{R: 0xD9, G: 0x3A, B: 0x33, A: 0xFF})
	if !l.e.s.SetzeDeckkraftText("74") {
		t.Fatal("Deckkraft 74 % abgelehnt")
	}
	l.setle()
	if fw.hex.gezeigt != "#D93A33" {
		t.Errorf("Hex-Feld bei 74 %%: %q, Soll #D93A33", fw.hex.gezeigt)
	}
	if fw.deckkraft.gezeigt != "74%" && fw.deckkraft.gezeigt != "74 %" {
		t.Errorf("Deckkraft-Feld: %q", fw.deckkraft.gezeigt)
	}
	// Sechs Ziffern ändern die Farbe und lassen die Deckkraft stehen.
	if !l.e.s.SetzeHex("#2F6FDB") || l.e.s.AktuelleFarbe() != (color.NRGBA{R: 0x2F, G: 0x6F, B: 0xDB, A: 0xBD}) {
		t.Errorf("sechs Ziffern: %v", l.e.s.AktuelleFarbe())
	}
	// Acht Ziffern setzen beide.
	if !l.e.s.SetzeHex("#11223380") || l.e.s.AktuelleFarbe() != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x80}) {
		t.Errorf("acht Ziffern: %v", l.e.s.AktuelleFarbe())
	}
	l.setle()
	if fw.hex.gezeigt != "#112233" {
		t.Errorf("nach acht Ziffern zeigt das Feld %q", fw.hex.gezeigt)
	}
}

func TestPipetteInDerZeileDesFarbpaars(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	se := &l.e.seite
	pip, vorn, fuel := se.fw.pipRahmen, se.paarRahmen[paarVorn], se.paarRahmen[paarFuellung]
	if pip.Empty() || (pip.Min.Y+pip.Max.Y)/2 != (vorn.Min.Y+vorn.Max.Y)/2 || pip.Min.X < fuel.Max.X {
		t.Fatalf("Pipette %v, Farbe %v, Füllung %v: nicht rechts in derselben Zeile", pip, vorn, fuel)
	}
	if pip.Max.X > l.fenst.X {
		t.Errorf("Pipette ragt aus der Leiste: %v", pip)
	}
	for _, r := range se.farbRahmen {
		if r.Overlaps(pip) {
			t.Errorf("Farbfeld %v überdeckt die Pipette %v", r, pip)
		}
	}
	l.klick(mittenPunkt(pip))
	if !l.e.s.PipetteLaeuft() {
		t.Error("Klick auf die Pipette startet sie nicht")
	}
}

// Seit Karte #364 zeigt das Raster acht zuletzt benutzte Farben (vorher vier neben den Beschriftungsfarben).
func TestVierZuletztBenutzteFarben(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	for i := range canvas.ZuletztGrenze + 1 {
		l.e.s.SetzeHex(canvas.HexText(color.NRGBA{R: uint8(10 + i), A: 255}))
	}
	l.setle()
	se := &l.e.seite
	if n := len(l.e.s.Zuletzt()); n != farbPlaetze {
		t.Fatalf("%d zuletzt benutzte, Soll %d", n, farbPlaetze)
	}
	for i := range farbPlaetze {
		if se.farbRahmen[i].Empty() {
			t.Fatalf("zuletzt benutzte Farbe %d nicht gezeichnet (%v)", i+1, l.e.s.Zuletzt())
		}
	}
	// Das Raster bleibt zwei Spalten mal vier Reihen neben der Farbfläche.
	if a, b := se.farbRahmen[0], se.farbRahmen[farbPlaetze-1]; a.Min.Y != se.fw.flaeche.Min.Y ||
		b.Max.Y > se.fw.flaeche.Max.Y || b.Min.X == a.Min.X {
		t.Errorf("Raster: erstes Feld %v, letztes %v, Fläche %v", a, b, se.fw.flaeche)
	}
}
