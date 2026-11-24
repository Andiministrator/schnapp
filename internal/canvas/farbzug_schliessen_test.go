// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Regressionstests aus der Sitzungs-QA 23.11.2026 abends (Kritiker A, Karte #189): Farbzug bei Schließen, Signal
// und Esc (A-1, A-4), Pipette in der Vollansicht (A-6), Sperre während des Speicherns.

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// A-1: Schließen-Anfrage während eines Farbzugs am gewählten Objekt.
func TestSitzungsQAFarbZugSchliessenAnfrage(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2]) // ungesicherte Änderung
	if !s.Geaendert() {
		t.Fatal("Vorbedingung: nicht geändert")
	}
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	z, abzug := s.SchliessenAnfrage()
	t.Logf("SchliessenAnfrage im Farbzug: Schluss %v, Abzug da %v, FarbZugLaeuft %v, D.ZugLaeuft %v",
		z, abzug != nil, s.FarbZugLaeuft(), s.D.ZugLaeuft())
	if z == SchlussFehler {
		t.Errorf("Schließen im Farbzug meldet SchlussFehler (Verwerfen-Weg) statt zu speichern")
	}
}

// A-1: Signal während eines Farbzugs am gewählten Objekt.
func TestSitzungsQAFarbZugSignal(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	zu, abzug := s.SignalAnfrage()
	t.Logf("SignalAnfrage im Farbzug: zu %v, Abzug da %v, geändert %v, D.ZugLaeuft %v", zu, abzug != nil, s.Geaendert(), s.D.ZugLaeuft())
	if zu && abzug == nil && s.Geaendert() {
		t.Errorf("Signal im Farbzug: sofort schließen ohne Abzug, obwohl geändert")
	}
}

// A-4: Esc während des Farbzugs.
func TestSitzungsQAFarbZugEsc(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	s.Esc()
	t.Logf("nach Esc: Auswahl %d, FarbZugLaeuft %v, D.ZugLaeuft %v", s.Auswahl, s.FarbZugLaeuft(), s.D.ZugLaeuft())
	if s.Auswahl == 0 && s.FarbZugLaeuft() {
		t.Errorf("Esc im Farbzug hebt die Auswahl auf und lässt den Zug laufen")
	}
	s.BeendeFarbZug()
	o, _ := s.D.Objekt(p)
	t.Logf("nach Loslassen: Pfeil %v, Schritte %d", o.(document.Pfeil).Strich.Farbe, schritte(t, s))
}

// A-6: Pipette beim Werkzeug Zuschneiden (Vollansicht) mit einem Streifen außerhalb des Zuschnitts.
func TestSitzungsQAPipetteVollansicht(t *testing.T) {
	o := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			o.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), A: 255})
		}
	}
	d, err := document.Neu(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 20, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(100, 0, 200, 100)); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(400, 300))
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	t.Logf("Streifen %v, Zuschnitt %v, Bildfläche %v, VollVersatz %v", d.Streifen(), d.Zuschnitt(), s.Bildflaeche(), d.VollVersatz())
	a := document.Pt(-50.5, 10.5) // Ansichtspunkt links vom Zuschnitt, vor dem Streifen
	ort, ok := s.ZeigerOrt(a)
	if !ok {
		t.Fatal("Punkt nicht im Bild")
	}
	if !s.SchaltePipette() {
		t.Fatal("Pipette geht nicht an")
	}
	p := s.Abb.ZuFenster(a)
	s.Druecken(p, TastePrimaer)
	s.Loslassen(p)
	want := o.NRGBAAt(ort.X, ort.Y)
	t.Logf("ZeigerOrt %v (Farbe %v), Pipette nahm %v", ort, want, s.Stil.Farbe)
	if s.Stil.Farbe != want {
		t.Errorf("Pipette nimmt %v, unter dem Zeiger liegt laut ZeigerOrt %v: %v", s.Stil.Farbe, ort, want)
	}
}

// Lücke: Sperre während des Speicherns in der Sitzung (Farbzug, Pipette) — erwartet grün.
func TestSitzungsQAFarbwahlWaehrendSpeichern(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	if _, err := s.SpeichernJetzt(); err != nil {
		t.Fatal(err)
	}
	if s.BeginneFarbZug() || s.SchaltePipette() || s.SetzeHex("#123456") || s.SetzeDeckkraftText("50") {
		t.Errorf("Farbwahl wirkt während des Speicherns")
	}
}
