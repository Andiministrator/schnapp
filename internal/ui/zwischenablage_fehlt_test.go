// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import "testing"

// F-177, F-331: Meldet der Gio-Patch einen Schreibfehler der Zwischenablage, zeigt der Editor genau einmal den Fehler
// und nimmt die Warnung vor dem Schließen zurück; die Marke wird dabei zurückgesetzt.
func TestZwischenablageFehltMeldetEinmal(t *testing.T) {
	e := testEditor(t)
	zwischenablageFehlt.Store(false)
	t.Cleanup(func() { zwischenablageFehlt.Store(false) })
	if e.zwischenablageFehltMelden() {
		t.Fatal("meldete ohne Fehler")
	}
	e.fensterHaeltKopie = true
	zwischenablageFehlt.Store(true)
	if !e.zwischenablageFehltMelden() {
		t.Fatal("meldete den Fehler nicht")
	}
	if !e.meldung.Fehler || e.meldung.Text != "Could not copy: no clipboard is available." {
		t.Errorf("Meldung %q (Fehler %v)", e.meldung.Text, e.meldung.Fehler)
	}
	if e.fensterHaeltKopie {
		t.Error("fensterHaeltKopie blieb gesetzt")
	}
	if zwischenablageFehlt.Load() {
		t.Error("Marke nicht zurückgesetzt")
	}
	e.melde("anderes", false)
	if e.zwischenablageFehltMelden() {
		t.Error("meldete zweimal, obwohl nur einmal gemeldet wurde")
	}
	if e.meldung.Text != "anderes" {
		t.Errorf("Meldung überschrieben: %q", e.meldung.Text)
	}
}
