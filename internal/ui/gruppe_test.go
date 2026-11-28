// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/io/key"
)

// ADR-0044 Punkt 2: Strg+G gruppiert die gewählten, Strg+Umschalt+G hebt auf; das Kontextmenü bietet beides nur, wo es
// wirkt; die Ebenenliste zeigt die Gruppe.
func TestGruppeKuerzelUndListe(t *testing.T) {
	l, k := ebenenLauf(t)
	l.fokusiere(l.e)
	l.taste("A", key.ModShortcut)
	if l.e.s.AnzahlGewaehlt() != 2 {
		t.Fatalf("Strg+A: %d gewählt", l.e.s.AnzahlGewaehlt())
	}
	l.e.kontext.objekt = true // wie nach einem Rechtsklick auf ein gewähltes Objekt
	if !hatEintrag(l.e.kontextEintraege(), aktGruppieren) || hatEintrag(l.e.kontextEintraege(), aktGruppierungAufheben) {
		t.Error("Kontextmenü vor dem Gruppieren")
	}
	l.taste("G", key.ModShortcut)
	if l.e.s.GruppeVon(k) == 0 || l.e.s.GruppeVon(k) != l.e.s.GruppeVon(l.id) {
		t.Fatalf("Strg+G gruppiert nicht: %d %d", l.e.s.GruppeVon(k), l.e.s.GruppeVon(l.id))
	}
	if hatEintrag(l.e.kontextEintraege(), aktGruppieren) || !hatEintrag(l.e.kontextEintraege(), aktGruppierungAufheben) {
		t.Error("Kontextmenü nach dem Gruppieren")
	}
	n := 0
	for _, z := range l.e.s.Ebenen() {
		if z.Gruppe != 0 {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d Zeilen mit Gruppe", n)
	}
	l.taste("G", key.ModShortcut|key.ModShift)
	if l.e.s.GruppeVon(k) != 0 {
		t.Error("Strg+Umschalt+G hebt nicht auf")
	}
}

func hatEintrag(en []menueEintrag, a aktion) bool {
	for _, e := range en {
		if e.akt == a {
			return true
		}
	}
	return false
}

// Paare der Übersicht F1 (F-705): gleicher Umschalter wird zusammengezogen.
func TestPaarTaste(t *testing.T) {
	for _, f := range [][3]string{
		{"Ctrl+PgUp", "Ctrl+PgDn", "Ctrl+PgUp/PgDn"},
		{"Alt+PgUp", "Ctrl+PgDn", "Alt+PgUp / Ctrl+PgDn"},
		{"F2", "F3", "F2 / F3"},
	} {
		if got := paarTaste(f[0], f[1]); got != f[2] {
			t.Errorf("paarTaste(%q, %q) = %q, Soll %q", f[0], f[1], got, f[2])
		}
	}
	if gruppenFarbe(1) == gruppenFarbe(2) || gruppenFarbe(1) != gruppenFarbe(1+len(gruppenFarben)) {
		t.Error("Farben der Gruppen")
	}
}
