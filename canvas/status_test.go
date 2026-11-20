// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "testing"

// F-151: Eine frische Meldung hat Vorrang vor dem Werkzeughinweis, bis zur nächsten Änderung am Dokument oder zum
// Werkzeugwechsel; ein Fehler bleibt bis zur nächsten Änderung.
func TestStatusZeile(t *testing.T) {
	const hinweis = "Klick setzt Nummer 3"
	gespeichert := NeueMeldung("gespeichert", false, 7, WerkzeugNummer)
	fehler := NeueMeldung("gescheitert", true, 7, WerkzeugNummer)
	for _, f := range []struct {
		name       string
		m          Meldung
		stand      uint64
		w          Werkzeug
		text       string
		istFehler  bool
		istMeldung bool
	}{
		{"frisch, gleiches Werkzeug", gespeichert, 7, WerkzeugNummer, "gespeichert", false, true},
		{"nach Änderung am Dokument", gespeichert, 8, WerkzeugNummer, hinweis, false, false},
		{"Rückgängig auf früheren Stand", gespeichert, 6, WerkzeugNummer, hinweis, false, false},
		{"nach Werkzeugwechsel", gespeichert, 7, WerkzeugPfeil, hinweis, false, false},
		{"Fehler frisch", fehler, 7, WerkzeugNummer, "gescheitert", true, true},
		{"Fehler übersteht Werkzeugwechsel", fehler, 7, WerkzeugText, "gescheitert", true, true},
		{"Fehler endet mit Änderung", fehler, 8, WerkzeugNummer, hinweis, false, false},
		{"keine Meldung", Meldung{}, 0, WerkzeugNummer, hinweis, false, false},
		{"leere Meldung", NeueMeldung("", true, 7, WerkzeugNummer), 7, WerkzeugNummer, hinweis, false, false},
	} {
		text, fe, m := StatusZeile(f.m, f.stand, f.w, hinweis)
		if text != f.text || fe != f.istFehler || m != f.istMeldung {
			t.Errorf("%s: (%q, %v, %v), Soll (%q, %v, %v)", f.name, text, fe, m, f.text, f.istFehler, f.istMeldung)
		}
	}
	// Mit Werkzeug Auswählen (kein Hinweis) zeigt die Leiste nach dem Ende der Meldung nichts.
	if text, _, m := StatusZeile(gespeichert, 8, WerkzeugAuswahl, ""); text != "" || m {
		t.Errorf("Auswählen nach Änderung: %q, %v", text, m)
	}
}

// F-151 zusammen mit der Sitzung: Speichern, dann Zeichnen beendet die Meldung; ein gescheiterter Versuch bleibt bis dahin.
func TestStatusZeileMitSitzung(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugNummer)
	m := NeueMeldung("In der Bibliothek gespeichert", false, s.D.Stand(), s.Werkzeug)
	if text, _, ist := StatusZeile(m, s.D.Stand(), s.Werkzeug, "hinweis"); !ist || text != "In der Bibliothek gespeichert" {
		t.Fatalf("frisch: %q %v", text, ist)
	}
	s.Druecken(f(150, 80), TastePrimaer)
	s.Loslassen(f(150, 80))
	if text, _, ist := StatusZeile(m, s.D.Stand(), s.Werkzeug, "hinweis"); ist || text != "hinweis" {
		t.Errorf("nach Anlegen: %q %v", text, ist)
	}
}
