// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

func gleich(a, b document.Punkt) bool { return math.Abs(a.X-b.X) < 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 }

// F-156: Mit Umschalt am Bildrand bleibt der Pfeil auf 45° (er wird entlang der Richtung gekürzt, nicht je Achse
// geklemmt) und der Kasten ein Quadrat. Bild 200×100, Fenster = Ansicht + (24, 24).
func TestUmschaltAmRandPfeilBleibtAuf45Grad(t *testing.T) {
	for _, fall := range []struct {
		name     string
		von, bis document.Punkt
		soll     document.Punkt
	}{
		{"unten rechts hinaus", f(100, 50), document.Pt(500, 130), document.Pt(150, 100)},
		{"oben links hinaus", f(100, 50), document.Pt(-300, -300), document.Pt(50, 0)},
		{"rechts hinaus: waagerecht", f(100, 50), document.Pt(900, 74), document.Pt(200, 50)},
		{"im Bild: nur gerastet", f(20, 20), f(60, 55), document.Pt(20+math.Hypot(40, 35)/math.Sqrt2, 20+math.Hypot(40, 35)/math.Sqrt2)},
	} {
		s, _, _ := sitzung(t)
		s.Umschalt = true
		s.SetzeWerkzeug(WerkzeugPfeil)
		zieh(s, fall.von, fall.bis)
		pf, ok := letztes(t, s).(document.Pfeil)
		if !ok {
			t.Fatalf("%s: kein Pfeil", fall.name)
		}
		if !gleich(pf.B, fall.soll) {
			t.Errorf("%s: B %v, Soll %v", fall.name, pf.B, fall.soll)
		}
		d := pf.B.Minus(pf.A)
		if w := math.Atan2(d.Y, d.X) / (math.Pi / 4); math.Abs(w-math.Round(w)) > 1e-9 {
			t.Errorf("%s: Winkel %.3f° nicht auf 45° gerastet", fall.name, w*45)
		}
	}
}

func TestUmschaltAmRandKastenBleibtQuadrat(t *testing.T) {
	for _, fall := range []struct {
		name     string
		von, bis document.Punkt
		soll     document.Rechteck
	}{
		// Platz nach rechts 80, nach unten 40: Seite 40
		{"unten rechts hinaus", f(120, 60), document.Pt(500, 160), document.R(document.Pt(120, 60), document.Pt(160, 100))},
		// Platz nach links 100, nach oben 50: Seite 50
		{"oben links hinaus", f(100, 50), document.Pt(-300, -300), document.R(document.Pt(50, 0), document.Pt(100, 50))},
		{"im Bild", f(20, 20), f(50, 40), document.R(document.Pt(20, 20), document.Pt(50, 50))},
	} {
		s, _, _ := sitzung(t)
		s.Umschalt = true
		s.SetzeWerkzeug(WerkzeugKasten)
		zieh(s, fall.von, fall.bis)
		k, ok := letztes(t, s).(document.Kasten)
		if !ok {
			t.Fatalf("%s: kein Kasten", fall.name)
		}
		if !gleich(k.Bereich.Min, fall.soll.Min) || !gleich(k.Bereich.Max, fall.soll.Max) {
			t.Errorf("%s: Bereich %v, Soll %v", fall.name, k.Bereich, fall.soll)
		}
		if math.Abs(k.Bereich.Breite()-k.Bereich.Hoehe()) > 1e-9 {
			t.Errorf("%s: kein Quadrat: %.3f × %.3f", fall.name, k.Bereich.Breite(), k.Bereich.Hoehe())
		}
	}
}

// F-157: Mit einem Stauch-Streifen rastet der Pfeil in der Ansicht; die Spitze liegt nie im herausgeschnittenen
// Streifen (Spalten 100 bis 150 des Originals), und in der Ansicht ist der Pfeil 45°.
func TestUmschaltRastenMitStreifen(t *testing.T) {
	s, _, _ := sitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 100, Bis: 150}}); err != nil {
		t.Fatal(err)
	}
	if err := s.D.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s.Abb = Abbildung{Zoom: 1, Versatz: document.Pt(24, 24)}
	s.Umschalt = true
	s.SetzeWerkzeug(WerkzeugPfeil)
	// Ansicht: Start (60, 10), Ziel (101, 50) — rechts der Schnittkante bei Ansichts-x 100 = Original 150.
	zieh(s, f(60, 10), f(101, 50))
	pf, ok := letztes(t, s).(document.Pfeil)
	if !ok {
		t.Fatal("kein Pfeil")
	}
	// Gerastet wird in der Ansicht: d = (41, 40) → 45°, Länge 57,28 → (40,51…; 40,51…).
	e := math.Hypot(41, 40) / math.Sqrt2
	soll := s.D.ZuOriginal(document.Pt(60+e, 10+e))
	if !gleich(pf.B, soll) || pf.B.X <= 150 {
		t.Errorf("B %v, Soll %v (rechts des Streifens, X > 150)", pf.B, soll)
	}
	if pf.B.X > 100 && pf.B.X < 150 {
		t.Errorf("Spitze im herausgeschnittenen Streifen: %v", pf.B)
	}
	d := s.D.ZuAnsicht(pf.B).Minus(s.D.ZuAnsicht(pf.A))
	if w := math.Atan2(d.Y, d.X) / (math.Pi / 4); math.Abs(w-math.Round(w)) > 1e-9 {
		t.Errorf("Ansichtswinkel %.3f° nicht auf 45° gerastet", w*45)
	}
	// Quadrat in der Ansicht: Original-Breite ist wegen des Streifens größer als die Höhe.
	s.Abwaehlen()
	s.SetzeWerkzeug(WerkzeugKasten)
	zieh(s, f(60, 10), f(101, 50))
	k := letztes(t, s).(document.Kasten)
	a, b := s.D.ZuAnsicht(k.Bereich.Min), s.D.ZuAnsicht(k.Bereich.Max)
	if math.Abs((b.X-a.X)-(b.Y-a.Y)) > 1e-9 || (k.Bereich.Max.X > 100 && k.Bereich.Max.X < 150) {
		t.Errorf("Kasten in der Ansicht kein Quadrat oder im Streifen: %v (Ansicht %v–%v)", k.Bereich, a, b)
	}
}
