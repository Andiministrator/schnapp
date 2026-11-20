// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/io/key"
)

// F-286, ADR-0026 Nachtrag 4: Die Tastenschleife meldet außer den Tasten der Tabelle nur den Fokus der Leinwand und die
// Navigation im Menü an. Eine weitere Taste, die am Test TestJedeTasteAuchPerMaus vorbei in die Schleife käme, macht
// diesen Test rot.
func TestTastenfilterNurTabelleUndMenueNavigation(t *testing.T) {
	e := testEditor(t)
	tabelle := map[key.Filter]bool{}
	for _, f := range tastenFilter(e.s.TextLaeuft()) {
		tabelle[f] = true
	}
	// Genau diese fünf: Pfeile, Pos1, Ende ohne Tag, Leertaste am Menüknopf.
	soll := map[key.Filter]bool{
		{Name: key.NameUpArrow}: true, {Name: key.NameDownArrow}: true, {Name: key.NameHome}: true,
		{Name: key.NameEnd}: true, {Focus: &e.menue.knopf, Name: key.NameSpace}: true,
	}
	nav := map[key.Filter]bool{}
	for _, f := range menueNavigationsFilter(&e.menue.knopf) {
		if tabelle[f] {
			t.Errorf("Navigation %q steht auch in der Tabelle", tastenName(f))
		}
		nav[f] = true
	}
	if len(nav) != len(soll) {
		t.Errorf("%d Navigationsfilter (Soll %d)", len(nav), len(soll))
	}
	for f := range soll {
		if !nav[f] {
			t.Errorf("Navigationsfilter %q fehlt", tastenName(f))
		}
	}
	// Die Schleife meldet genau Tabelle, Navigation und den Fokus der Leinwand an.
	fokus := 0
	gesehen := map[key.Filter]bool{}
	for _, f := range e.alleTastenFilter() {
		switch f := f.(type) {
		case key.Filter:
			if !tabelle[f] && !nav[f] {
				t.Errorf("Filter %q weder in der Tabelle noch Menü-Navigation", tastenName(f))
			}
			gesehen[f] = true
		case key.FocusFilter:
			fokus++
		default:
			t.Errorf("unbekannter Filter %T", f)
		}
	}
	if fokus != 1 {
		t.Errorf("%d Fokusfilter (Soll 1)", fokus)
	}
	for f := range tabelle {
		if !gesehen[f] {
			t.Errorf("Tabellenfilter %q nicht angemeldet", tastenName(f))
		}
	}
	for f := range nav {
		if !gesehen[f] {
			t.Errorf("Navigationsfilter %q nicht angemeldet", tastenName(f))
		}
	}
}
