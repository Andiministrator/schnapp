// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// verlaufSitzung liefert eine Sitzung über einem Farbverlauf 200×100 (Zoom 1, Versatz 24 wie sitzung).
func verlaufSitzung(t *testing.T) *Sitzung {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(2 * y), uint8(x ^ y), 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	return s
}

// ADR-0022 Punkt 1: Ein Rahmen mit dem Werkzeug Schwärzen überschreibt genau die Pixel darin (nach außen gerundet),
// legt kein Objekt an und ist ein Rückgängig-Schritt.
func TestSchwaerzenWerkzeug(t *testing.T) {
	s := verlaufSitzung(t)
	if !s.SetzeWerkzeug(WerkzeugSchwaerzen) || s.Voll() != AblehnungKeine {
		t.Fatal("Werkzeug")
	}
	s.Druecken(f(10.5, 20.5), TastePrimaer)
	s.Ziehen(f(30, 30))
	if a, b, ok := s.Rahmen(); !ok || a != f(10.5, 20.5) || b != f(30, 30) {
		t.Errorf("Rahmen beim Ziehen: %v %v %v", a, b, ok)
	}
	s.Loslassen(f(40.2, 50.2))
	if _, _, ok := s.Rahmen(); ok {
		t.Error("Rahmen nach dem Loslassen noch da")
	}
	if len(s.D.Objekte()) != 0 {
		t.Error("Schwärzen hat ein Objekt angelegt")
	}
	if s.BildAenderung() != 1 || !s.Geaendert() || !s.D.Geschwaerzt() {
		t.Errorf("BildAenderung %d, Geaendert %v", s.BildAenderung(), s.Geaendert())
	}
	o := s.D.Original()
	for _, p := range []image.Point{{10, 20}, {40, 50}, {25, 35}} {
		if c := o.NRGBAAt(p.X, p.Y); c != Schwaerzfarbe {
			t.Errorf("Pixel %v nicht geschwärzt: %v", p, c)
		}
	}
	for _, p := range []image.Point{{9, 20}, {41, 50}, {10, 19}, {40, 51}} {
		if c := o.NRGBAAt(p.X, p.Y); c == Schwaerzfarbe {
			t.Errorf("Pixel %v außerhalb geschwärzt", p)
		}
	}
	if n := schritte(t, s); n != 1 {
		t.Errorf("%d Rückgängig-Schritte, want 1", n)
	}
}

// Ein Klick oder ein winziger Rahmen schwärzt nichts.
func TestSchwaerzenZuKlein(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, f(10, 10), f(14, 40))
	s.Druecken(f(50, 50), TastePrimaer)
	s.Loslassen(f(50, 50))
	if s.Geaendert() || s.BildAenderung() != 0 {
		t.Error("zu kleiner Rahmen hat geschwärzt")
	}
}

// ADR-0022 Punkt 2: Verpixeln lässt kein Pixel des Bereichs stehen; ist er zu schmal, wird geschwärzt und gemeldet.
func TestVerpixelnWerkzeug(t *testing.T) {
	s := verlaufSitzung(t)
	vorher := s.D.Original()
	s.SetzeWerkzeug(WerkzeugVerpixeln)
	zieh(s, f(20, 20), f(80, 60))
	if s.NimmVerpixelnErsetzt() {
		t.Error("großer Bereich als zu schmal gemeldet")
	}
	o := s.D.Original()
	gleich := 0
	for y := 20; y < 60; y++ {
		for x := 20; x < 80; x++ {
			if o.NRGBAAt(x, y) == vorher.NRGBAAt(x, y) {
				gleich++
			}
		}
	}
	// Der Verlauf hat in jedem Block verschiedene Farben; ein unverändertes Pixel wäre Zufall (Mosaikfarbe
	// trifft es), mehr als eine Handvoll ein Fehler.
	if gleich > 10 {
		t.Errorf("%d Pixel unverändert", gleich)
	}
	zieh(s, f(100, 10), f(106, 90)) // 6 px breit: zu schmal zum Verpixeln, aber über MinObjekt
	if !s.NimmVerpixelnErsetzt() {
		t.Error("schmaler Bereich nicht als ersetzt gemeldet")
	}
	if s.NimmVerpixelnErsetzt() {
		t.Error("Meldung „zu schmal“ kommt ein zweites Mal (F-170)")
	}
	if c := s.D.Original().NRGBAAt(103, 50); c != Schwaerzfarbe {
		t.Errorf("schmaler Bereich nicht geschwärzt: %v", c)
	}
}

// Mit Stauch-Streifen im Rahmen wird der verborgene Teil des Originals mitgeschwärzt.
func TestSchwaerzenUeberStreifen(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 50, Bis: 70}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeFenster(image.Pt(228, 148)) // Ansicht 180×100, eingepasst Zoom 1, Versatz 24
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, f(40, 10), f(60, 30)) // Ansicht 40..60 = Original 40..50 und 70..80
	o := s.D.Original()
	for _, x := range []int{40, 49, 55, 69, 79} {
		if c := o.NRGBAAt(x, 20); c != Schwaerzfarbe {
			t.Errorf("Original x=%d nicht geschwärzt: %v", x, c)
		}
	}
}

// ADR-0022 Punkt 3: Nach dem Speichern lässt sich eine Schwärzung nicht mehr zurücknehmen; ohne Schwärzung bleibt der
// Verlauf.
func TestNachSpeichernKeinRueckgaengigDerSchwaerzung(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	zieh(s, f(10, 10), f(40, 40))
	s.Gespeichert()
	if !s.D.KannRueckgaengig() {
		t.Error("Verlauf ohne Schwärzung nach dem Speichern geleert")
	}
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, f(50, 10), f(90, 40))
	s.Gespeichert()
	if s.D.KannRueckgaengig() || s.Geaendert() {
		t.Errorf("nach dem Speichern: KannRueckgaengig %v, Geaendert %v", s.D.KannRueckgaengig(), s.Geaendert())
	}
	if c := s.D.Original().NRGBAAt(60, 20); c != Schwaerzfarbe {
		t.Errorf("Schwärzung verloren: %v", c)
	}
}
