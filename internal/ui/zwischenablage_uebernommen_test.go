// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import "testing"

// F-328: Übernimmt ein anderes Programm die Zwischenablage (Gio-Patch: app.ClipboardCancelled), hält das Fenster die
// Kopie nicht mehr: Das Schließen warnt nicht.
func TestUebernommenOhneWarnungBeimSchliessen(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	zwischenablageUebernommen.Store(true) // das täte der Gio-Patch
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	if !e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete die Übernahme nicht")
	}
	if e.fensterHaeltKopie {
		t.Error("fensterHaeltKopie blieb gesetzt")
	}
	if e.fensterWarnen() {
		t.Error("Schließen warnt trotz Übernahme")
	}
	if e.meldung.Text == warnungKopie() {
		t.Error("Warnung gezeigt")
	}
}

// F-328: Eine bereits gezeigte Fenster-Warnung verschwindet, wenn die Übernahme danach eintrifft.
func TestUebernommenLoestGezeigteWarnungAuf(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	if !e.fensterWarnen() || e.meldung.Text != warnungKopie() || e.meldungHaelt != haltFenster {
		t.Fatal("Vorbedingung: Warnung nicht gezeigt")
	}
	zwischenablageUebernommen.Store(true)
	t.Cleanup(func() { zwischenablageUebernommen.Store(false) })
	if !e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete die Übernahme nicht")
	}
	if e.haltendeMeldung() || e.meldungHaelt != keinHalt || e.meldung.Text != "" {
		t.Errorf("Warnung blieb: %q (Halt %v)", e.meldung.Text, e.meldungHaelt)
	}
	if e.zwischenablageUebernommenMelden() {
		t.Error("meldete zweimal")
	}
}

// F-328 Gegenprobe: Ohne Übernahme bleibt die Warnung.
func TestOhneUebernahmeBleibtWarnung(t *testing.T) {
	e, _ := editorMitFensterKopie(t)
	zwischenablageUebernommen.Store(false)
	if e.zwischenablageUebernommenMelden() {
		t.Fatal("meldete ohne Übernahme")
	}
	if !e.fensterWarnen() || e.meldung.Text != warnungKopie() {
		t.Errorf("Warnung fehlt: %q", e.meldung.Text)
	}
}
