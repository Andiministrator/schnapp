// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"
)

// hilfeLauf öffnet die Übersicht bei der Fenstergröße f (mit oder ohne Leiste der letzten Aufnahmen) und zeichnet sie.
func hilfeLauf(t *testing.T, f image.Point, mitLeiste bool) *menueLauf {
	t.Helper()
	var l *menueLauf
	if mitLeiste {
		l = neuerLeistenLauf(t, f, 3).menueLauf
	} else {
		l = neuerMenueLauf(t, f)
	}
	l.e.hilfeOffen = true
	l.setle()
	return l
}

// Die Übersicht der Tastenkürzel (F1) zeigt im Startfenster 1200×800 alle Zeilen ohne Rollen, mit und ohne die Leiste
// der letzten Aufnahmen (F-705).
func TestHilfeZeigtImStartfensterAlleZeilen(t *testing.T) {
	for _, mitLeiste := range []bool{false, true} {
		l := hilfeLauf(t, image.Pt(1200, 800), mitLeiste)
		fl := l.e.leinwandFlaeche(l.gtx())
		k := &l.e.hilfeRoll
		n := len(l.e.hilfeZeilen())
		if k.rollMax != 0 {
			t.Errorf("Leiste %v: Leinwand %v, %d Zeilen: die Liste rollt um %d, alle Zeilen sollen sichtbar sein",
				mitLeiste, fl.Size(), n, k.rollMax)
		}
		if g := l.e.hilfeGroesse; g.X > fl.Dx() || g.Y > fl.Dy() {
			t.Errorf("Leiste %v: Tafel %v größer als die Leinwand %v", mitLeiste, g, fl.Size())
		}
	}
}

// In kleinen Fenstern rollt die Liste: Mausrad, Pfeile, Bild auf/ab, Pos1 und Ende; die Grenzen halten.
func TestHilfeRolltInKleinenFenstern(t *testing.T) {
	l := hilfeLauf(t, image.Pt(800, 500), true)
	k := &l.e.hilfeRoll
	if k.rollMax <= 0 {
		t.Fatalf("Vorbedingung: die Übersicht rollt bei 800×500 nicht (rollMax %d, Tafel %v)", k.rollMax, l.e.hilfeGroesse)
	}
	ende := k.rollMax
	l.taste(key.NameEnd, 0)
	if k.rollen != ende {
		t.Errorf("Ende: rollen %d, Soll %d", k.rollen, ende)
	}
	l.taste(key.NameDownArrow, 0)
	if k.rollen != ende {
		t.Errorf("Runter am Ende: rollen %d, Soll %d", k.rollen, ende)
	}
	l.taste(key.NameHome, 0)
	if k.rollen != 0 {
		t.Errorf("Pos1: rollen %d, Soll 0", k.rollen)
	}
	l.taste(key.NameDownArrow, 0)
	if k.rollen != hilfeZeileDp {
		t.Errorf("Runter: rollen %d, Soll %d", k.rollen, hilfeZeileDp)
	}
	l.taste(key.NamePageDown, 0)
	if k.rollen <= hilfeZeileDp {
		t.Errorf("Bild ab: rollen %d, hat sich nicht bewegt", k.rollen)
	}
	l.taste(key.NamePageUp, 0)
	l.taste(key.NameUpArrow, 0)
	if k.rollen != 0 {
		t.Errorf("zurück: rollen %d, Soll 0", k.rollen)
	}
	l.radSchritte(image.Pt(l.e.hilfeGroesse.X/2+l.e.leinwandFlaeche(l.gtx()).Min.X, l.fenst.Y/2), 5, 15)
	if k.rollen <= 0 {
		t.Errorf("Mausrad: rollen %d, Soll > 0", k.rollen)
	}
	if !l.e.hilfeOffen {
		t.Errorf("Die Rolltasten haben die Übersicht geschlossen")
	}
	l.taste(key.NameEscape, 0)
	l.setle()
	if l.e.hilfeOffen || k.rollen != 0 {
		t.Errorf("Esc: offen %v, rollen %d; Soll geschlossen und zurückgesetzt", l.e.hilfeOffen, k.rollen)
	}
}

// Die Rechnung der Rolltasten ohne Fenster.
func TestRollTaste(t *testing.T) {
	for _, c := range []struct {
		n                      key.Name
		rollen, rollMax        int
		zeile, seite, erwartet int
		rolle                  bool
	}{
		{key.NameDownArrow, 0, 100, 20, 200, 20, true},
		{key.NameDownArrow, 95, 100, 20, 200, 100, true},
		{key.NameUpArrow, 5, 100, 20, 200, 0, true},
		{key.NamePageDown, 0, 500, 20, 200, 180, true},
		{key.NamePageUp, 250, 500, 20, 200, 70, true},
		{key.NameHome, 70, 500, 20, 200, 0, true},
		{key.NameEnd, 70, 500, 20, 200, 500, true},
		{key.NameSpace, 70, 500, 20, 200, 70, false},
	} {
		got, rolle := rollTaste(c.n, c.rollen, c.rollMax, c.zeile, c.seite)
		if got != c.erwartet || rolle != c.rolle {
			t.Errorf("%s bei %d/%d: %d, %v; Soll %d, %v", c.n, c.rollen, c.rollMax, got, rolle, c.erwartet, c.rolle)
		}
	}
}
