// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// TestTextStilWerkzeugUndNeuerText: Ohne Text ändert die Wahl nur das Werkzeug; ein neuer Text übernimmt alles.
func TestTextStilWerkzeugUndNeuerText(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	if !s.Seitenleiste().Text {
		t.Fatal("Leiste zeigt beim Text-Werkzeug keine Schrift")
	}
	vor := s.D.Stand()
	if s.SetzeSchrift(fonts.GoMono) || s.SchalteStil(StilFett) || s.SchalteStil(StilUnterstrichen) ||
		s.SetzeAusrichtung(document.AusrichtungRechts) || s.SetzeSchriftgrad(32) || s.D.Stand() != vor {
		t.Fatal("Wahl ohne Text änderte das Dokument")
	}
	st := s.AktuellerTextStil()
	if st.Schrift != fonts.GoMono || !st.Fett || st.Kursiv || !st.Unterstrichen || st.Ausrichtung != document.AusrichtungRechts ||
		st.Grad != 32 || st.Fehlt {
		t.Fatalf("Werkzeugstil %+v", st)
	}
	s.Druecken(f(120, 30), TastePrimaer)
	s.Loslassen(f(120, 30))
	s.Tippe("ab")
	s.Enter(false)
	tx := letztes(t, s).(document.Text)
	if tx.Schrift != fonts.GoMono || !tx.Fett || !tx.Unterstrichen || tx.Ausrichtung != document.AusrichtungRechts ||
		tx.Schriftgrad != 32 {
		t.Errorf("neuer Text %+v", tx)
	}
	if s.SetzeSchrift(fonts.Go); s.Stil.Schrift != "" {
		t.Errorf("Go heißt im Werkzeug %q statt leer", s.Stil.Schrift)
	}
}

// TestTextStilGewaehlterText: Jede Wahl ist ein Rückgängig-Schritt am gewählten Text, dieselbe Wahl keiner.
func TestTextStilGewaehlterText(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(10, 60), Inhalt: "x", Schriftgrad: 20})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	if s.AktuellerTextStil().Grad != 20 {
		t.Fatal("Leiste zeigt nicht den gewählten Text")
	}
	for i, f := range []func() bool{
		func() bool { return s.SchalteStil(StilKursiv) },
		func() bool { return s.SchalteStil(StilDurchgestrichen) },
		func() bool { return s.SetzeAusrichtung(document.AusrichtungMitte) },
		func() bool { return s.SetzeSchrift("Fehlt Hier") },
	} {
		if !f() {
			t.Fatalf("Wahl %d änderte den Text nicht", i)
		}
	}
	if s.SetzeAusrichtung(document.AusrichtungMitte) || s.SetzeSchrift("a\nb") || s.SetzeSchriftgrad(0) ||
		s.SetzeSchriftgrad(document.MaxSchrift+1) || s.SetzeAusrichtung(9) {
		t.Error("gleiche oder ungültige Wahl ergab einen Schritt")
	}
	o, _ := s.D.Objekt(id)
	tx := o.(document.Text)
	if !tx.Kursiv || !tx.Durchgestrichen || tx.Ausrichtung != document.AusrichtungMitte || tx.Schrift != "Fehlt Hier" {
		t.Errorf("Text %+v", tx)
	}
	if st := s.AktuellerTextStil(); !st.Fehlt || st.Schrift != "Fehlt Hier" {
		t.Errorf("fehlende Schrift nicht gemeldet: %+v", st)
	}
	if n := schritte(t, s); n != 1+4 { // Hinzufügen des Texts, dann die vier Wahlen
		t.Errorf("%d Schritte, erwartet 5", n)
	}
	// Stufen der Größe: von 20 hoch auf 24, von 21 runter auf 20; oben Schluss.
	if g, ok := s.SchriftgradStufe(1); !ok || g != 24 {
		t.Errorf("Stufe hoch: %v %v", g, ok)
	}
	s.SetzeSchriftgrad(21)
	if g, ok := s.SchriftgradStufe(-1); !ok || g != 20 {
		t.Errorf("Stufe runter von 21: %v %v", g, ok)
	}
	s.SetzeSchriftgrad(SchriftStufen[len(SchriftStufen)-1])
	if _, ok := s.SchriftgradStufe(1); ok {
		t.Error("über der höchsten Stufe gibt es noch eine")
	}
	s.SetzeSchriftgrad(SchriftStufen[0])
	if _, ok := s.SchriftgradStufe(-1); ok {
		t.Error("unter der kleinsten Stufe gibt es noch eine")
	}
}

// TestTextStilInDerEingabe: Während der Eingabe gehört die Wahl zur Eingabe (kein eigener Schritt).
func TestTextStilInDerEingabe(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 30), TastePrimaer)
	s.Loslassen(f(120, 30))
	s.Tippe("ab")
	vor := schritte(t, s)
	s.SchalteStil(StilFett)
	s.SetzeSchriftgrad(40)
	if tx, _ := s.TextObjekt(); !tx.Fett || tx.Schriftgrad != 40 || !s.TextLaeuft() {
		t.Fatalf("Eingabe %+v", tx)
	}
	s.Enter(false)
	if n := schritte(t, s); n != vor+1 {
		t.Errorf("%d Schritte, erwartet %d", n, vor+1)
	}
}
