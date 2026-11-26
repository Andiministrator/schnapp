// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0038 Punkt 2: KontextBei wählt das Objekt unter dem Zeiger gleich welchen Werkzeugs (das Werkzeug bleibt), hebt
// sonst die Auswahl auf; Zuschneiden wählt nichts; eine Texteingabe wird bestätigt; gesperrt öffnet es nicht.
func TestKontextBei(t *testing.T) {
	s, k, _ := sitzung(t) // Bild bei (24,24), Zoom 1 (f rechnet den Versatz ein): Kasten (10,10)–(50,50) gefüllt
	s.SetzeWerkzeug(WerkzeugPfeil)
	if offen, obj := s.KontextBei(f(30, 30)); !offen || !obj || s.Auswahl != k || s.Werkzeug != WerkzeugPfeil {
		t.Fatalf("auf dem Kasten: offen %v, objekt %v, Auswahl %d, Werkzeug %v", offen, obj, s.Auswahl, s.Werkzeug)
	}
	if offen, obj := s.KontextBei(f(150, 30)); !offen || obj || s.Auswahl != 0 {
		t.Errorf("leer: offen %v, objekt %v, Auswahl %d", offen, obj, s.Auswahl)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if _, obj := s.KontextBei(f(30, 30)); obj || s.Auswahl != 0 {
		t.Errorf("Zuschneiden wählt den Kasten (%v)", obj)
	}
	// Texteingabe: bestätigt, dann gewählt.
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(150, 100), TastePrimaer)
	s.Loslassen(f(150, 100))
	s.Tippe("hi")
	if offen, _ := s.KontextBei(f(30, 30)); !offen || s.TextLaeuft() {
		t.Errorf("Texteingabe nicht bestätigt (offen %v)", offen)
	}
	if tx, ok := letztes(t, s).(document.Text); !ok || tx.Inhalt != "hi" {
		t.Errorf("Text nicht angelegt: %+v", letztes(t, s))
	}
	s.SetzeRuhe(true)
	if offen, _ := s.KontextBei(f(30, 30)); offen {
		t.Error("gesperrt geöffnet")
	}
	s.SetzeRuhe(false)
}

// ADR-0038 Punkt 6: Der Anker per Tastatur ist die Mitte des gewählten Objekts im Fenster, sonst die Mitte der Leinwand.
func TestKontextAnker(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	if p, obj := s.KontextAnker(); !obj || !nah(p, f(30, 30)) {
		t.Errorf("mit Kasten: %v %v, Soll (54,54)", p, obj)
	}
	s.Auswahl = 0
	if p, obj := s.KontextAnker(); obj || !nah(p, document.Pt(124, 74)) {
		t.Errorf("ohne Auswahl: %v %v, Soll (124,74)", p, obj)
	}
}
