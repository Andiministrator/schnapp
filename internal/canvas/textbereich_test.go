// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// koordinatenBild ist ein Bild, dessen Pixel ihre Lage im Original tragen: R = x, G = y.
func koordinatenSitzung(t *testing.T, w, h int) *Sitzung {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 7, A: 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return NeueSitzung(d)
}

// ziehe fährt einen Zug der linken Taste in Fensterpunkten.
func ziehe(s *Sitzung, a, b document.Punkt) {
	s.Druecken(a, TastePrimaer)
	s.Ziehen(document.Pt((a.X+b.X)/2, (a.Y+b.Y)/2))
	s.Ziehen(b)
	s.Loslassen(b)
}

// Der Modus wählt einen Bereich, ohne Objekt und Werkzeug anzufassen; danach gilt das vorige Werkzeug und der Modus
// ist aus. Der Bereich kommt genau einmal.
func TestTextBereichZiehenWaehltBereich(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	s.Abb = Abbildung{Zoom: 1}
	s.eingepasst = false
	s.SetzeWerkzeug(WerkzeugKasten)
	if !s.BeginneTextBereich() || !s.TextBereichLaeuft() {
		t.Fatal("Modus beginnt nicht")
	}
	vor := s.D.Stand()
	s.Druecken(document.Pt(10, 10), TastePrimaer)
	s.Ziehen(document.Pt(40, 30))
	if !s.Zieht() {
		t.Error("Zieht meldet den Zug des Modus nicht")
	}
	a, b, ok := s.TextBereichRahmen()
	if !ok || a != document.Pt(10, 10) || b != document.Pt(40, 30) {
		t.Errorf("Rahmen %v %v %v", a, b, ok)
	}
	s.Loslassen(document.Pt(40.5, 30.2))
	if s.D.Stand() != vor || len(s.D.Objekte()) != 0 {
		t.Error("der Zug hat ein Objekt angelegt")
	}
	if s.Werkzeug != WerkzeugKasten || s.TextBereichLaeuft() || s.Zieht() {
		t.Errorf("nach dem Zug: Werkzeug %v, Modus %v, Zieht %v", s.Werkzeug, s.TextBereichLaeuft(), s.Zieht())
	}
	r, ok := s.NimmTextBereich()
	if !ok || r != image.Rect(10, 10, 41, 31) {
		t.Errorf("Bereich %v %v, erwartet (10,10)-(41,31)", r, ok)
	}
	if _, ok := s.NimmTextBereich(); ok {
		t.Error("Bereich kam zweimal")
	}
}

// Ein Klick (unter der Ziehschwelle) und ein Zug ganz neben dem Bild wählen nichts; der Modus wartet weiter.
func TestTextBereichKlickUndDaneben(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	s.Abb = Abbildung{Zoom: 1}
	s.eingepasst = false
	s.BeginneTextBereich()
	ziehe(s, document.Pt(20, 20), document.Pt(21, 21))
	if _, ok := s.NimmTextBereich(); ok || !s.TextBereichLaeuft() {
		t.Errorf("Klick wählte einen Bereich oder beendete den Modus (%v)", s.TextBereichLaeuft())
	}
	ziehe(s, document.Pt(120, 10), document.Pt(160, 40)) // rechts neben dem Bild
	if _, ok := s.NimmTextBereich(); ok || !s.TextBereichLaeuft() {
		t.Error("Zug neben dem Bild wählte einen Bereich oder beendete den Modus")
	}
	ziehe(s, document.Pt(90, 70), document.Pt(130, 120)) // halb drin: auf das Bild begrenzt
	if r, ok := s.NimmTextBereich(); !ok || r != image.Rect(90, 70, 100, 80) {
		t.Errorf("Bereich %v %v, erwartet (90,70)-(100,80)", r, ok)
	}
}

// Enter nimmt das ganze sichtbare Bild (Ansicht nach Zuschnitt und Stauchen); Esc bricht ab, auch mitten im Zug.
func TestTextBereichEnterUndEsc(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	if err := s.D.SetzeAnsicht(image.Rect(10, 5, 90, 75), []document.Streifen{{Von: 30, Bis: 40}}); err != nil {
		t.Fatal(err)
	}
	if s.Enter(false) {
		t.Error("Enter ohne Modus wählte etwas")
	}
	s.BeginneTextBereich()
	if !s.Enter(false) {
		t.Fatal("Enter im Modus wirkt nicht")
	}
	if r, ok := s.NimmTextBereich(); !ok || r != image.Rect(0, 0, 70, 70) {
		t.Errorf("ganzes Bild %v %v, erwartet (0,0)-(70,70)", r, ok)
	}
	if s.TextBereichLaeuft() {
		t.Error("Modus läuft nach Enter weiter")
	}

	s.SetzeWerkzeug(WerkzeugPfeil)
	s.BeginneTextBereich()
	s.Druecken(document.Pt(10, 10), TastePrimaer)
	s.Ziehen(document.Pt(30, 30))
	s.Esc()
	if s.TextBereichLaeuft() || s.Zieht() || s.Werkzeug != WerkzeugPfeil {
		t.Errorf("nach Esc: Modus %v, Zieht %v, Werkzeug %v", s.TextBereichLaeuft(), s.Zieht(), s.Werkzeug)
	}
	s.Loslassen(document.Pt(30, 30))
	if _, ok := s.NimmTextBereich(); ok {
		t.Error("nach Esc kam ein Bereich")
	}
	if len(s.D.Objekte()) != 0 {
		t.Error("das Loslassen nach Esc legte einen Pfeil an")
	}
}

// Die Umrechnung beachtet Zoom, Versatz, Zuschnitt und Streifen: Der Bereich liegt in Pixeln der Ansicht, und das Pixel
// dort ist das Original an der Stelle, die der Zeiger zeigte.
func TestTextBereichUmrechnung(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	if err := s.D.SetzeAnsicht(image.Rect(10, 5, 90, 75), []document.Streifen{{Von: 30, Bis: 40},
		{Zeilen: true, Von: 50, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	s.Abb = Abbildung{Zoom: 2, Versatz: document.Pt(10, 6)}
	s.eingepasst = false
	s.BeginneTextBereich()
	// Der Rahmen während des Zugs liegt in Fensterpunkten, auch bei Zoom und Versatz.
	s.Druecken(document.Pt(20, 12), TastePrimaer)
	s.Ziehen(document.Pt(60, 40))
	if a, b, ok := s.TextBereichRahmen(); !ok || a != document.Pt(20, 12) || b != document.Pt(60, 40) {
		t.Errorf("Rahmen %v %v %v, erwartet (20,12)–(60,40)", a, b, ok)
	}
	s.Esc()
	s.BeginneTextBereich()
	// Ansicht (5, 3) bis (40.25, 50.5): Fenster = Ansicht·2 + Versatz
	ziehe(s, document.Pt(20, 12), document.Pt(90.5, 107))
	r, ok := s.NimmTextBereich()
	if !ok || r != image.Rect(5, 3, 41, 51) {
		t.Fatalf("Bereich %v %v, erwartet (5,3)-(41,51)", r, ok)
	}
	bild := s.D.Ansichtsbild()
	for _, p := range []image.Point{r.Min, {r.Max.X - 1, r.Min.Y}, {r.Min.X, r.Max.Y - 1}, r.Max.Sub(image.Pt(1, 1))} {
		o := s.D.ZuOriginal(document.Pt(float64(p.X)+0.5, float64(p.Y)+0.5))
		c := bild.NRGBAAt(p.X, p.Y)
		if int(c.R) != int(math.Floor(o.X)) || int(c.G) != int(math.Floor(o.Y)) {
			t.Errorf("Ansichtspixel %v trägt Original (%d,%d), erwartet (%.0f,%.0f)", p, c.R, c.G, math.Floor(o.X), math.Floor(o.Y))
		}
	}
	// Zuschnitt (10, 5) und die Spalten 30–39 und Zeilen 50–59 fehlen: Ansicht (5,3) ist Original (15,8), Ansicht x = 40
	// ist Original 60, Ansicht y = 50 ist Original 65.
	if c := bild.NRGBAAt(5, 3); c.R != 15 || c.G != 8 {
		t.Errorf("Ansicht (5,3) = Original (%d,%d), erwartet (15,8)", c.R, c.G)
	}
	if c := bild.NRGBAAt(40, 50); c.R != 60 || c.G != 65 {
		t.Errorf("Ansicht (40,50) = Original (%d,%d), erwartet (60,65)", c.R, c.G)
	}
}

// Werkzeugwechsel beendet den Modus; während des Ziehens und des Speicherns beginnt er nicht; der zweite Druck eines
// Doppelklicks beginnt nur einen Zug (kein Aufheben des Zuschnitts).
func TestTextBereichGrenzen(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	s.Abb = Abbildung{Zoom: 1}
	s.eingepasst = false
	s.BeginneTextBereich()
	s.SetzeWerkzeug(WerkzeugText)
	if s.TextBereichLaeuft() {
		t.Error("Werkzeugwechsel beendet den Modus nicht")
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Druecken(document.Pt(10, 10), TastePrimaer) // Ansicht verschieben
	if s.BeginneTextBereich() {
		t.Error("Modus beginnt während eines Zugs")
	}
	s.Loslassen(document.Pt(10, 10))

	if err := s.D.SetzeZuschnitt(image.Rect(10, 10, 90, 70)); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.BeginneTextBereich()
	s.Doppelklick(document.Pt(20, 20))
	if s.D.Zuschnitt() != image.Rect(10, 10, 90, 70) {
		t.Error("Doppelklick im Modus hob den Zuschnitt auf")
	}
	if !s.Zieht() {
		t.Error("der zweite Druck beginnt keinen Zug des Modus")
	}
	s.Abbrechen() // Zeiger-Cancel: Zug weg, Modus wartet weiter
	if s.Zieht() || !s.TextBereichLaeuft() {
		t.Errorf("nach Cancel: Zieht %v, Modus %v", s.Zieht(), s.TextBereichLaeuft())
	}
	s.Esc()

	s.hg.laeuft = true // Speichern läuft
	if s.BeginneTextBereich() {
		t.Error("Modus beginnt während des Speicherns")
	}
	s.hg.laeuft = false
}

// Die mittlere Taste verschiebt im Modus die Ansicht wie sonst; der Modus bleibt.
func TestTextBereichMittlereTaste(t *testing.T) {
	s := koordinatenSitzung(t, 100, 80)
	s.Abb = Abbildung{Zoom: 1}
	s.eingepasst = false
	s.BeginneTextBereich()
	s.Druecken(document.Pt(10, 10), TasteMitte)
	s.Ziehen(document.Pt(30, 10))
	s.Loslassen(document.Pt(30, 10))
	if s.Abb.Versatz != document.Pt(20, 0) || !s.TextBereichLaeuft() {
		t.Errorf("Versatz %v, Modus %v", s.Abb.Versatz, s.TextBereichLaeuft())
	}
}

// SichtPixel rundet nach außen, begrenzt auf das Bild und lehnt leere und ungültige Bereiche ab.
func TestSichtPixel(t *testing.T) {
	g := image.Pt(50, 40)
	for _, fall := range []struct {
		a, b document.Punkt
		r    image.Rectangle
		ok   bool
	}{
		{document.Pt(1.5, 2.5), document.Pt(10.1, 3.9), image.Rect(1, 2, 11, 4), true},
		{document.Pt(10.1, 3.9), document.Pt(1.5, 2.5), image.Rect(1, 2, 11, 4), true}, // Ecken vertauscht
		{document.Pt(-5, -5), document.Pt(60, 60), image.Rect(0, 0, 50, 40), true},
		{document.Pt(-5, -5), document.Pt(-1, 10), image.Rectangle{}, false},
		{document.Pt(50, 0), document.Pt(70, 10), image.Rectangle{}, false},
		{document.Pt(3, 3), document.Pt(3, 9), image.Rectangle{}, false}, // keine Breite
		{document.Pt(math.NaN(), 3), document.Pt(9, 9), image.Rectangle{}, false},
	} {
		r, ok := SichtPixel(fall.a, fall.b, g)
		if r != fall.r || ok != fall.ok {
			t.Errorf("%v–%v: %v %v, erwartet %v %v", fall.a, fall.b, r, ok, fall.r, fall.ok)
		}
	}
}
