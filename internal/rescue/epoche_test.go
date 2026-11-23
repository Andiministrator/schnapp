// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-406 (Mutant S2, F-417, F-446): Eine Kopie einer späteren Epoche darf der Auftrag einer früheren nicht an sich
// ziehen. Ohne Uhr und ohne Schlafen, nur über die Reihenfolge der Kette: Ein hängendes Schreiben (A) hält die Kette
// an; währenddessen steht ein zweiter Auftrag der Epoche 1 (B) hinter ihm, dann kommt Verwirf (Epoche 2, löscht) und
// eine neue Kopie (D). Nach dem Freigeben muss B sie überspringen (es ist nicht seine Epoche) und D sie schreiben.
// Zöge B sie an sich, verwürfe er sie als veraltet, und D fände nichts mehr: Die jüngste Kopie ginge verloren.
func TestSicherungKopieDerSpaeterenEpocheBleibt(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	s := NeueSicherung(ort, k)
	n, frei := haltenderSchreiber(t, s)
	d, _ := document.Neu(bild(64, 64))
	text := func() {
		t.Helper()
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(5, 20), Inhalt: "t", Schriftgrad: 10}); err != nil {
			t.Fatal(err)
		}
	}

	text()
	s.Sichere(d) // Auftrag A: beginnt zu schreiben und hängt
	wartetAuf(t, func() bool { return n.Load() == 1 })
	text()
	s.Sichere(d) // Auftrag B (Epoche 1): steht hinter A, hält die wartende Kopie
	s.Verwirf()  // Epoche 2, Auftrag C löscht
	text()
	text()
	s.Sichere(d) // Auftrag D (Epoche 2): vier Texte
	frei()
	if !s.Ende() {
		t.Fatal("Ende nicht fertig")
	}
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatalf("die jüngste Kopie fehlt: %v", err)
	}
	neu, _ := document.Neu(bild(64, 64))
	if err := Anwenden(neu, kp, nil); err != nil || len(neu.Objekte()) != 4 {
		t.Errorf("Kopie mit %d Objekten (%v), Soll 4 aus Epoche 2", len(neu.Objekte()), err)
	}
}
