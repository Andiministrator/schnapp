// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-420, F-154: Ein Formwechsel ist eine neue Bearbeitung (merkeAenderung) und hebt die Verwerfen-Marke eines
// gescheiterten Speicherns auf. Kommt danach Rückgängig auf den Stand der Marke zurück, verlangt der Stand einen neuen
// Speicherversuch, statt still verworfen zu werden. Ohne merkeAenderung im Wechsel bliebe die Marke stehen und
// Ungesichert wäre falsch.
func TestWechsleKanteHebtVerwerfenMarkeAuf(t *testing.T) {
	s := randSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 100, Bis: 120}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.WechsleKante(); !ok {
		t.Fatal("erster Wechsel misslungen")
	}
	// Das Speichern bei diesem Stand ist gescheitert, der nächste Schließen-Versuch dürfte ihn verwerfen.
	s.verwerfen, s.verwerfenBei = true, s.D.Stand()
	if s.Ungesichert() {
		t.Fatal("Ausgangslage: Stand zum Verwerfen freigegeben, Ungesichert soll falsch sein")
	}
	if _, ok := s.WechsleKante(); !ok {
		t.Fatal("zweiter Wechsel misslungen")
	}
	if s.verwerfen {
		t.Error("Verwerfen-Marke nach dem Formwechsel noch gesetzt")
	}
	if !s.Rueckgaengig() {
		t.Fatal("Rückgängig misslungen")
	}
	if s.D.Stand() != s.verwerfenBei {
		t.Fatal("Rückgängig führt nicht auf den Stand der Marke")
	}
	if !s.Ungesichert() {
		t.Error("Stand nach Wechsel und Rückgängig gilt als zum Verwerfen freigegeben")
	}
}
