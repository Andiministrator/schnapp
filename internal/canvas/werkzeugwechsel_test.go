// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "testing"

// TestWerkzeugwechselHebtAuswahlAuf prüft ADR-0039 Nachtrag 3 (Karte #382, Andi 27.11.2026): Wer ein anderes Werkzeug
// als Auswählen nimmt, verliert jede Auswahl, und die rechte Leiste zeigt das Werkzeug; zu Auswählen bleibt die
// Hauptauswahl, dasselbe Werkzeug noch einmal ändert nichts.
func TestWerkzeugwechselHebtAuswahlAuf(t *testing.T) {
	s, k, p := sitzung(t)

	// Auswählen → Kasten: Haupt- und weitere Auswahl weg, Leiste zeigt das Werkzeug.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	if !s.SchalteGewaehlt(p) || s.AnzahlGewaehlt() != 2 {
		t.Fatalf("Kontrolle: %d gewählt", s.AnzahlGewaehlt())
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Auswahl != 0 || s.AnzahlGewaehlt() != 0 {
		t.Fatalf("nach dem Wechsel zu Kasten: Auswahl %d, %d gewählt", s.Auswahl, s.AnzahlGewaehlt())
	}
	if l := s.Seitenleiste(); l.Auswahl || l.Werkzeug != WerkzeugKasten {
		t.Fatalf("Leiste nach dem Wechsel: %+v", l)
	}

	// Mit dem Kasten gewählt (etwa per Rechtsklick, das Kontextmenü wählt mit jedem Werkzeug): dasselbe Werkzeug noch einmal lässt die Auswahl stehen, Auswählen behält sie.
	s.Auswahl = k
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Auswahl != k {
		t.Fatalf("dasselbe Werkzeug hebt die Auswahl auf: %d", s.Auswahl)
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	if s.Auswahl != k {
		t.Fatalf("Wechsel zu Auswählen hebt die Auswahl auf: %d", s.Auswahl)
	}

	// Auswählen → Pfeil, Text, Nummer, Lupe, Sprechblase, Schwärzen: jedes Mal ohne Auswahl.
	for _, w := range []Werkzeug{WerkzeugPfeil, WerkzeugText, WerkzeugNummer, WerkzeugLupe, WerkzeugSprechblase,
		WerkzeugSchwaerzen, WerkzeugVerpixeln} {
		s.SetzeWerkzeug(WerkzeugAuswahl)
		s.Auswahl = p
		s.SetzeWerkzeug(w)
		if s.Auswahl != 0 {
			t.Errorf("Wechsel zu %v lässt die Auswahl %d stehen", w, s.Auswahl)
		}
		if l := s.Seitenleiste(); l.Auswahl || l.Werkzeug != w {
			t.Errorf("Leiste nach dem Wechsel zu %v: %+v", w, l)
		}
	}

	// Auswählen → Auswählen behält die weiteren gewählten (kein Wechsel).
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	s.SchalteGewaehlt(p)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	if s.AnzahlGewaehlt() != 2 {
		t.Fatalf("Auswählen noch einmal hebt die Mehrfachauswahl auf: %d", s.AnzahlGewaehlt())
	}

	// Ein Zeichenwerkzeug mit Mehrfachauswahl (gezeichnet, dann Umschalt-Klick) → Auswählen: Hauptauswahl bleibt,
	// die weiteren fallen weg (ADR-0039 Punkt 6).
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Auswahl = k
	s.SchalteGewaehlt(p)
	if s.AnzahlGewaehlt() != 2 {
		t.Fatalf("Kontrolle: %d gewählt", s.AnzahlGewaehlt())
	}
	if haupt := s.Auswahl; haupt != 0 {
		s.SetzeWerkzeug(WerkzeugAuswahl)
		if s.Auswahl != haupt || s.AnzahlGewaehlt() != 1 {
			t.Fatalf("Kasten → Auswählen: Auswahl %d, %d gewählt", s.Auswahl, s.AnzahlGewaehlt())
		}
	}
}
