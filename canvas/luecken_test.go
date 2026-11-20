// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// F-243: Testlücken der Nacharbeit Sprint 6 in der Leinwand.

// Ältere Dateien (vor F-222) tragen einen Streifen am Rand des Originals, ohne Naht. Ein Schwärzungsrahmen über den
// Rand der Ansicht hinaus schwärzt ihn nicht mit (F-223, ADR-0023 Punkt 8): weder unten noch oben.
func TestSchwaerzenUeberRandstreifenAlterDatei(t *testing.T) {
	for _, fall := range []struct {
		name        string
		streifen    document.Streifen
		von, bis    float64 // Ansicht, y
		erst, letzt int     // geschwärzte Zeilen des Originals
	}{
		{"unten", document.Streifen{Zeilen: true, Von: 150, Bis: 200}, 100, 900, 100, 149},
		{"oben", document.Streifen{Zeilen: true, Von: 0, Bis: 40}, 50, -300, 40, 89},
	} {
		t.Run(fall.name, func(t *testing.T) {
			s := randSitzung(t)
			if err := s.D.SetzeStreifen([]document.Streifen{fall.streifen}); err != nil {
				t.Fatal(err)
			}
			if err := s.D.VergissVerlauf(); err != nil { // wie frisch geöffnet
				t.Fatal(err)
			}
			if n := len(s.D.Naehte()); n != 0 {
				t.Fatalf("Vorbedingung: Randstreifen ohne Naht, %d Nähte", n)
			}
			s.Einpassen()
			s.SetzeWerkzeug(WerkzeugSchwaerzen)
			zieh(s, fv(s, 50, fall.von), fv(s, 80, fall.bis))
			z := schwarzeZeilen(s.D.Original(), 60)
			if len(z) == 0 || z[0] != fall.erst || z[len(z)-1] != fall.letzt || len(z) != fall.letzt-fall.erst+1 {
				t.Errorf("geschwärzte Zeilen %v, Soll %d..%d", z, fall.erst, fall.letzt)
			}
		})
	}
}

// Jeder Druck setzt die Meldung „Rahmen zu klein“ zurück (F-227): Wer sie nicht abholt und weiterarbeitet, bekommt
// sie nicht später zu einem anderen Zug gezeigt.
func TestDrueckenSetztRahmenZuKleinZurueck(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, f(10, 10), f(14, 40)) // zu klein: Marke gesetzt, nicht abgeholt
	// Die mittlere Taste verschiebt nur die Ansicht; Loslassen berührt die Marke dabei nicht.
	s.Druecken(f(50, 50), TasteMitte)
	s.Loslassen(f(50, 50))
	if s.NimmRahmenZuKlein() {
		t.Error("Meldung „Rahmen zu klein“ überlebt einen neuen Druck")
	}
}
