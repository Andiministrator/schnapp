// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0043: --cursor und --no-cursor nur bei Aufnahmen, nicht zusammen; mehrfach dasselbe ist erlaubt.
func TestAuswertenCursor(t *testing.T) {
	for _, f := range []struct {
		args   []string
		soll   int
		fehler bool
	}{
		{[]string{"--cursor"}, 1, false},
		{[]string{"--screen", "--no-cursor"}, -1, false},
		{[]string{"--cursor", "--window", "--cursor"}, 1, false},
		{[]string{"--cursor", "--no-cursor"}, 0, true},
		{[]string{"--cursor", "bild.png"}, 0, true},
		{[]string{"--no-cursor", "--version"}, 0, true},
		{[]string{"--cursor=yes"}, 0, true},
	} {
		a, err := Auswerten(f.args)
		if f.fehler != (err != nil) || (err == nil && a.Mauszeiger != f.soll) {
			t.Errorf("%v: %+v, %v", f.args, a, err)
		}
	}
}

// ADR-0043: Befehlszeile vor [capture] cursor.
func TestPlaneMauszeiger(t *testing.T) {
	an, aus := settings.Voreinstellungen(), settings.Voreinstellungen()
	an.Mauszeiger = true
	for _, f := range []struct {
		e    settings.Einstellungen
		a    int
		soll bool
	}{
		{aus, 0, false}, {an, 0, true}, {aus, 1, true}, {an, -1, false}, {an, 1, true}, {aus, -1, false},
	} {
		if got := plane(Aufruf{Modus: ModusBereich, Mauszeiger: f.a}, f.e).Mauszeiger; got != f.soll {
			t.Errorf("Einstellung %v, Befehlszeile %d: %v", f.e.Mauszeiger, f.a, got)
		}
	}
	if plane(Aufruf{Modus: ModusOeffnen, Datei: "x.png"}, an).Mauszeiger {
		t.Error("beim Öffnen einer Datei gilt der Mauszeiger")
	}
}

// ADR-0043: Jede Aufnahmeart reicht die Wahl an KWin weiter (include-cursor).
func TestAufnahmeReichtMauszeigerWeiter(t *testing.T) {
	for _, m := range []Modus{ModusFenster, ModusBildschirm} {
		l := neuerLauf(t)
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		ziel := Bildschirmziel{}
		if m == ModusBildschirm {
			ziel = Bildschirmziel{Name: "A"}
		}
		l.fuehre(Auftrag{Modus: m, Ziel: ziel, Mauszeiger: true})
		if len(l.q.optionen) == 0 || !l.q.optionen[0].Mauszeiger {
			t.Errorf("Modus %v: Optionen %+v", m, l.q.optionen)
		}
	}
}
