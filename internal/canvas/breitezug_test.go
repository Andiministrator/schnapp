// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Karte #352: Skala des Reglers — Enden, ganze Pixel, Lage und Stärke passen zueinander.
func TestBreiteReglerSkala(t *testing.T) {
	if BreiteAusRegler(0) != 1 || BreiteAusRegler(1) != 32 || BreiteAusRegler(-3) != 1 || BreiteAusRegler(7) != 32 {
		t.Errorf("Enden: %v %v", BreiteAusRegler(0), BreiteAusRegler(1))
	}
	if b := BreiteAusRegler(0.5); b != math.Round(math.Sqrt(32)) {
		t.Errorf("Mitte %v, Soll %v (logarithmisch)", b, math.Round(math.Sqrt(32)))
	}
	for _, b := range StrichStufen {
		if got := BreiteAusRegler(ReglerLage(b)); got != b {
			t.Errorf("Stufe %v: zurück %v", b, got)
		}
	}
	if ReglerLage(0.5) != 0 || ReglerLage(100) != 1 {
		t.Errorf("außerhalb: %v %v", ReglerLage(0.5), ReglerLage(100))
	}
	for f := 0.0; f <= 1; f += 0.01 {
		if b := BreiteAusRegler(f); b != math.Round(b) {
			t.Fatalf("Lage %v: %v keine ganze Zahl", f, b)
		}
	}
}

// Karte #352: Ziehen am Regler ändert den gewählten Pfeil laufend und wird ein Rückgängig-Schritt; ohne Änderung keiner;
// Abbruch stellt das Objekt her, die Stärke des Werkzeugs bleibt die gezogene.
func TestBreiteZug(t *testing.T) {
	s := randSitzung(t)
	id, err := s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(10, 10), document.Pt(80, 60),
		document.Strich{Farbe: Beschriftungsfarben[0], Breite: 5}))
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	vor := s.D.Stand()
	if !s.BeginneBreiteZug() || !s.BreiteZugLaeuft() || !s.Zieht() {
		t.Fatal("Zug beginnt nicht")
	}
	if s.BeginneBreiteZug() {
		t.Error("zweiter Zug während des ersten")
	}
	for _, b := range []float64{7, 11, 20} {
		s.BreiteZugSetze(b)
		if s.Breite() != b || s.Stil.Breite != b {
			t.Fatalf("während des Zugs: Objekt %v, Stil %v, Soll %v", s.Breite(), s.Stil.Breite, b)
		}
	}
	if s.SetzeBreite(3) {
		t.Error("− / + während des Zugs angenommen")
	}
	s.BeendeBreiteZug()
	if s.BreiteZugLaeuft() || s.Zieht() || s.Breite() != 20 || s.D.Stand() == vor {
		t.Fatalf("nach dem Zug: läuft %v, Breite %v", s.BreiteZugLaeuft(), s.Breite())
	}
	if !s.Rueckgaengig() || s.Breite() != 5 {
		t.Errorf("ein Rückgängig: %v, Soll 5", s.Breite())
	}
	if _, ok := s.D.Objekt(id); !ok { // ein Schritt zurück ist vor dem Zug, der Pfeil besteht noch
		t.Error("ein Rückgängig nahm mehr als den Zug zurück")
	}
	// Ohne Änderung kein Schritt.
	nach := s.D.Stand()
	s.BeginneBreiteZug()
	s.BreiteZugSetze(5)
	s.BeendeBreiteZug()
	if s.D.Stand() != nach {
		t.Error("gleiche Stärke ist ein Schritt")
	}
	// Esc bricht ab: Objekt wie vorher, Werkzeug behält die gezogene Stärke.
	s.BeginneBreiteZug()
	s.BreiteZugSetze(24)
	s.Esc()
	if s.BreiteZugLaeuft() || s.Breite() != 5 || s.Stil.Breite != 24 || s.D.Stand() != nach {
		t.Errorf("nach Esc: läuft %v, Objekt %v, Stil %v", s.BreiteZugLaeuft(), s.Breite(), s.Stil.Breite)
	}
	// Abbrechen (Zeiger-Cancel) ebenso.
	s.BeginneBreiteZug()
	s.BreiteZugSetze(2)
	s.Abbrechen()
	if s.BreiteZugLaeuft() || s.Breite() != 5 {
		t.Errorf("nach Abbrechen: läuft %v, Objekt %v", s.BreiteZugLaeuft(), s.Breite())
	}
}

// Karte #352: Ohne gewähltes Objekt setzt der Zug nur die Stärke des Werkzeugs, ohne Rückgängig-Schritt.
func TestBreiteZugOhneAuswahl(t *testing.T) {
	s := randSitzung(t)
	vor := s.D.Stand()
	if !s.BeginneBreiteZug() {
		t.Fatal("Zug beginnt nicht")
	}
	s.BreiteZugSetze(16)
	s.BeendeBreiteZug()
	if s.Stil.Breite != 16 || s.Breite() != 16 || s.D.Stand() != vor {
		t.Errorf("Stil %v, Stand geändert %v", s.Stil.Breite, s.D.Stand() != vor)
	}
	s.BreiteZugSetze(3) // nach dem Ende wirkungslos
	if s.Stil.Breite != 16 {
		t.Errorf("nach dem Ende gesetzt: %v", s.Stil.Breite)
	}
}

// Karte #352: Während einer Texteingabe beginnt kein Zug am Regler; die Eingabe läuft weiter.
func TestBreiteZugNichtInDerTexteingabe(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 80), TastePrimaer)
	s.Loslassen(f(120, 80))
	s.Tippe("x")
	if s.BeginneBreiteZug() || s.BreiteZugLaeuft() || !s.TextLaeuft() {
		t.Errorf("Zug in der Texteingabe: läuft %v, Text %v", s.BreiteZugLaeuft(), s.TextLaeuft())
	}
}

// Karte #352: Eine Schließen-Anfrage und ein Signal brechen einen laufenden Zug am Regler ab, wie im Farbwähler.
func TestBreiteZugSchliessenUndSignal(t *testing.T) {
	for _, signal := range []bool{false, true} {
		s := randSitzung(t)
		s.BeginneBreiteZug()
		s.BreiteZugSetze(20)
		if signal {
			s.SignalAnfrage()
		} else {
			s.SchliessenAnfrage()
		}
		if s.BreiteZugLaeuft() || s.Zieht() {
			t.Errorf("Signal %v: Zug läuft weiter", signal)
		}
	}
}
