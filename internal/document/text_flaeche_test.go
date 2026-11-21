// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image/color"
	"math"
	"testing"
)

// tinteVon misst die Tinte einer einzelnen Zeile roh (ohne Rand), in Pixeln des Originals.
func tinteVon(t *testing.T, tx Text) Rechteck {
	t.Helper()
	schriftMu.Lock()
	defer schriftMu.Unlock()
	s := schriftFuer(false)
	if s == nil {
		t.Fatal("Schrift nicht ladbar")
	}
	_, ink, ok := s.zeile(tx.Inhalt, tx.Schriftgrad)
	if !ok {
		t.Fatal("keine Tinte")
	}
	g := Pt(tx.Ort.X, tx.Grundlinie(0))
	return Rechteck{Min: g.Plus(ink.Min), Max: g.Plus(ink.Max)}
}

// F-149: Zwischen den Glyphen und dem Rand des Schilds liegen mindestens 0,25 Schriftgrade, rundum.
func TestTextSchildInnenabstand(t *testing.T) {
	for _, grad := range []float64{12, 40, 100} {
		tx := Text{Ort: Pt(30, 20), Inhalt: "Hg", Schriftgrad: grad, Hintergrund: color.NRGBA{A: 255}}
		ink, g := tinteVon(t, tx), tx.Grenzen()
		min := 0.25*grad - 1e-9
		for name, abstand := range map[string]float64{
			"links": ink.Min.X - g.Min.X, "rechts": g.Max.X - ink.Max.X,
			"oben": ink.Min.Y - g.Min.Y, "unten": g.Max.Y - ink.Max.Y,
		} {
			if abstand < min {
				t.Errorf("Grad %v, %s: Abstand %.3f, Soll mindestens %.3f (Grenzen %v, Tinte %v)", grad, name, abstand, min, g, ink)
			}
		}
	}
}

// F-172: Die Grenzen messen die gezeichnete Fläche, auch für breite Zeichen und mehrere Zeilen; die alte Schätzung (0,6
// em je Zeichen) war für „W“ viel zu schmal.
func TestTextGrenzenUmfassenTinte(t *testing.T) {
	for _, inhalt := range []string{"WWWWWWWWWW", "Hg", "ÅÖj", "MMMM\nWW\n\njjj"} {
		tx := Text{Ort: Pt(10, 10), Inhalt: inhalt, Schriftgrad: 40}
		g, r := tx.Grenzen(), Reichweite(tx)
		if r.Vereint(g) != g {
			t.Errorf("%q: Grenzen %v umfassen die Tinte %v nicht", inhalt, g, r)
		}
	}
	w := Text{Ort: Pt(10, 10), Inhalt: "WWWWWWWWWW", Schriftgrad: 40}
	if b := w.Grenzen().Breite(); b < 10*0.85*40 {
		t.Errorf("Breite %.1f für zehn W im Grad 40, Soll mindestens 340", b)
	}
	// Eine Zeile mehr macht das Schild eine Zeile höher (1,3 Schriftgrade).
	e := Text{Ort: Pt(0, 0), Inhalt: "H", Schriftgrad: 40}
	z := Text{Ort: Pt(0, 0), Inhalt: "H\nH", Schriftgrad: 40}
	if d := z.Grenzen().Hoehe() - e.Grenzen().Hoehe(); math.Abs(d-1.3*40) > 1e-6 {
		t.Errorf("zweite Zeile macht das Schild %.3f höher, Soll %.3f", d, 1.3*40)
	}
	// Der Rahmen springt nicht, wenn eine Oberlänge dazukommt: gleiche Höhe für „a“ und „H“.
	a := Text{Ort: Pt(0, 0), Inhalt: "a", Schriftgrad: 40}
	if a.Grenzen().Hoehe() != e.Grenzen().Hoehe() {
		t.Errorf("Höhe für „a“ %.3f, für „H“ %.3f", a.Grenzen().Hoehe(), e.Grenzen().Hoehe())
	}
}

// F-172: Ziffern, die breiter sind als der Kreis, gehören zu den Grenzen der Nummer.
func TestNummerGrenzenUmfassenZiffern(t *testing.T) {
	n := Nummer{Mitte: Pt(50, 50), Wert: 88888, Radius: 6}
	g := n.Grenzen()
	z, ok := NummerTinte(n)
	if !ok {
		t.Fatal("keine Tinte")
	}
	if g.Vereint(z) != g || g.Breite() <= 12 {
		t.Errorf("Grenzen %v umfassen Ziffern %v nicht", g, z)
	}
	k := Nummer{Mitte: Pt(50, 50), Wert: 7, Radius: 10}
	if k.Grenzen() != R(Pt(40, 40), Pt(60, 60)) {
		t.Errorf("einstellige Nummer: Grenzen %v, Soll der Kreis", k.Grenzen())
	}
}

// F-108: Die Grundlinie liegt einen Schriftgrad unter Ort.Y, je Zeile 1,3 weiter; die der Ziffern eine halbe Versalhöhe
// (Go Bold, rund 0,72 em) unter der Mitte.
func TestGrundlinien(t *testing.T) {
	tx := Text{Ort: Pt(3, 7), Schriftgrad: 20}
	if tx.Grundlinie(0) != 27 || tx.Grundlinie(2) != 7+20*(1+2.6) {
		t.Errorf("Grundlinien %v %v", tx.Grundlinie(0), tx.Grundlinie(2))
	}
	n := Nummer{Mitte: Pt(0, 100), Wert: 5, Radius: 10}
	if d := n.Grundlinie() - 100; d < 12*0.30 || d > 12*0.42 {
		t.Errorf("Ziffern-Grundlinie %.3f unter der Mitte, Soll rund 0,36 · 12", d)
	}
}
