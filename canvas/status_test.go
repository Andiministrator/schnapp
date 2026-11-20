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
		meldung := f.m
		text, fe, m := StatusZeile(&meldung, f.stand, f.w, hinweis, false)
		if text != f.text || fe != f.istFehler || m != f.istMeldung {
			t.Errorf("%s: (%q, %v, %v), Soll (%q, %v, %v)", f.name, text, fe, m, f.text, f.istFehler, f.istMeldung)
		}
	}
	// Mit Werkzeug Auswählen (kein Hinweis) zeigt die Leiste nach dem Ende der Meldung nichts.
	if text, _, m := StatusZeile(&gespeichert, 8, WerkzeugAuswahl, "", false); text != "" || m {
		t.Errorf("Auswählen nach Änderung: %q, %v", text, m)
	}
}

// F-151 zusammen mit der Sitzung: Speichern, dann Zeichnen beendet die Meldung; ein gescheiterter Versuch bleibt bis dahin.
func TestStatusZeileMitSitzung(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugNummer)
	m := NeueMeldung("In der Bibliothek gespeichert", false, s.D.Stand(), s.Werkzeug)
	if text, _, ist := StatusZeile(&m, s.D.Stand(), s.Werkzeug, "hinweis", false); !ist || text != "In der Bibliothek gespeichert" {
		t.Fatalf("frisch: %q %v", text, ist)
	}
	s.Druecken(f(150, 80), TastePrimaer)
	s.Loslassen(f(150, 80))
	if text, _, ist := StatusZeile(&m, s.D.Stand(), s.Werkzeug, "hinweis", false); ist || text != "hinweis" {
		t.Errorf("nach Anlegen: %q %v", text, ist)
	}
	// F-160: Rückgängig führt auf den alten Stand, die abgelöste Meldung kehrt nicht zurück.
	if !s.Rueckgaengig() {
		t.Fatal("Rückgängig")
	}
	if text, _, ist := StatusZeile(&m, s.D.Stand(), s.Werkzeug, "hinweis", false); ist || text != "hinweis" {
		t.Errorf("nach Rückgängig kehrt die Meldung zurück: %q %v", text, ist)
	}
}

// F-160 (2): Ein Fehler, den eine Änderung abgelöst hat, kehrt nach Rückgängig nicht zurück („Close again to discard
// the changes“ stimmte dann nicht mehr).
func TestStatusFehlerKehrtNichtZurueck(t *testing.T) {
	m := NeueMeldung("Could not save. Close again to discard the changes.", true, 7, WerkzeugAuswahl)
	if text, fe, ist := StatusZeile(&m, 7, WerkzeugAuswahl, "h", false); text != m.Text || !fe || !ist {
		t.Fatalf("frisch: %q %v %v", text, fe, ist)
	}
	if text, _, ist := StatusZeile(&m, 8, WerkzeugAuswahl, "h", false); text != "h" || ist {
		t.Fatalf("nach Änderung: %q %v", text, ist)
	}
	if text, fe, ist := StatusZeile(&m, 7, WerkzeugAuswahl, "h", false); text != "h" || fe || ist {
		t.Errorf("nach Rückgängig kehrt der Fehler zurück: %q %v %v", text, fe, ist)
	}
	// Auch wenn Änderung und Rückgängig zwischen zwei Bildern liegen, genügt die Abfrage nach jedem Ereignis.
	m2 := NeueMeldung("Fehler", true, 7, WerkzeugAuswahl)
	m2.Frisch(8, WerkzeugAuswahl)
	if m2.Frisch(7, WerkzeugAuswahl) {
		t.Error("Frisch: abgelöster Fehler gilt wieder")
	}
}

// F-160 (3): Eine gewöhnliche Meldung kehrt nach Werkzeugwechsel hin und zurück nicht zurück; ein Fehler überdauert den
// Werkzeugwechsel (F-151) und bleibt dann auch nach dem Zurückwechseln stehen.
func TestStatusMeldungKehrtNachWerkzeugwechselNichtZurueck(t *testing.T) {
	m := NeueMeldung("Gespeichert", false, 7, WerkzeugNummer)
	if text, _, ist := StatusZeile(&m, 7, WerkzeugPfeil, "h", false); text != "h" || ist {
		t.Fatalf("anderes Werkzeug: %q %v", text, ist)
	}
	if text, _, ist := StatusZeile(&m, 7, WerkzeugNummer, "h", false); text != "h" || ist {
		t.Errorf("zurück zum alten Werkzeug: %q %v (Soll Hinweis)", text, ist)
	}
	fehler := NeueMeldung("Fehler", true, 7, WerkzeugNummer)
	StatusZeile(&fehler, 7, WerkzeugPfeil, "h", false)
	if text, fe, ist := StatusZeile(&fehler, 7, WerkzeugNummer, "h", false); text != "Fehler" || !fe || !ist {
		t.Errorf("Fehler nach Hin und Zurück: %q %v %v", text, fe, ist)
	}
}

// F-160 (4): Während der Texteingabe geht der Hinweis vor „Gespeichert“; ein Fehler bleibt sichtbar.
func TestStatusTexteingabeHinweisVorGespeichert(t *testing.T) {
	m := NeueMeldung("Gespeichert", false, 7, WerkzeugText)
	if text, _, ist := StatusZeile(&m, 7, WerkzeugText, "Type the text", true); text != "Type the text" || ist {
		t.Fatalf("während der Eingabe: %q %v", text, ist)
	}
	if text, _, ist := StatusZeile(&m, 7, WerkzeugText, "h", false); text != "h" || ist {
		t.Errorf("nach der Eingabe kehrt „Gespeichert“ zurück: %q %v", text, ist)
	}
	fehler := NeueMeldung("Could not save", true, 7, WerkzeugText)
	if text, fe, ist := StatusZeile(&fehler, 7, WerkzeugText, "Type the text", true); text != "Could not save" || !fe || !ist {
		t.Errorf("Fehler während der Eingabe: %q %v %v", text, fe, ist)
	}
}
