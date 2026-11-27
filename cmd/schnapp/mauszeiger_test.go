// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/selection"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
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

// ADR-0043 Punkt 3: Jede Aufnahmeart reicht die Wahl an KWin weiter (include-cursor), bei jedem Aufruf (Kritik #338
// T-1, F-940: Bereich, Wiederholen, Monitor unter dem Zeiger, alle Monitore, Fenster, benannter Monitor).
func TestAufnahmeReichtMauszeigerWeiter(t *testing.T) {
	for _, f := range []struct {
		name string
		a    Auftrag
		vor  func(l *lauf)
	}{
		{"Fenster", Auftrag{Modus: ModusFenster}, nil},
		{"benannter Monitor", Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Name: "A"}}, nil},
		{"Monitor unter dem Zeiger", Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Wahl: settings.WahlZeiger}}, nil},
		{"alle Monitore", Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Wahl: settings.WahlAlle}}, nil},
		{"Bereich", Auftrag{Modus: ModusBereich}, func(l *lauf) {
			l.overlayErg = ui.OverlayErgebnis{Monitor: 1, Bereich: image.Rect(1, 1, 4, 3)}
		}},
		{"Wiederholen", Auftrag{Modus: ModusWiederholen}, func(l *lauf) {
			if err := selection.SpeichereLetzten(l.an.letzterPfad, selection.Letzter{Monitor: "B", Groesse: image.Pt(6, 4),
				Skala: 1, Bereich: image.Rect(2, 0, 5, 2)}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		for _, zeiger := range []bool{true, false} {
			l := neuerLauf(t)
			if f.vor != nil {
				f.vor(l)
			}
			a := f.a
			a.Mauszeiger = zeiger
			l.fuehre(a)
			if len(l.q.optionen) == 0 {
				t.Errorf("%s: keine Aufnahme", f.name)
			}
			for i, o := range l.q.optionen {
				if o.Mauszeiger != zeiger {
					t.Errorf("%s (Zeiger %v): Aufruf %d mit %+v", f.name, zeiger, i, o)
				}
			}
		}
	}
}

// Kritik #338 T-3 (F-940): Die Hilfe nennt --delay und --cursor/--no-cursor.
func TestHilfeNenntDelayUndCursor(t *testing.T) {
	for _, o := range []string{"--delay=SEC", "--cursor, --no-cursor"} {
		if !strings.Contains(hilfe(), o) {
			t.Errorf("Hilfe ohne %s", o)
		}
	}
}

// ADR-0043 Punkt 4 (Kritik #338 T-2, F-940): Der MCP-Server nimmt nie mit Zeiger auf, auf keinem Weg (benannter
// Monitor, aktiver, alle). Eine Verzögerung kennt er nicht: er geht nicht über plane/anschluesse.aufnehmen.
func TestMcpOhneZeiger(t *testing.T) {
	l := neuerMcpLauf(t)
	for _, args := range []string{`{"screen":"DP-1"}`, `{}`, `{"all":true}`} {
		if e := l.rufe(t, "capture_screen", args); e.Fehler {
			t.Fatalf("%s: %+v", args, e)
		}
	}
	if len(l.q.optionen) < 3 {
		t.Fatalf("Kontrolle: %d Aufnahmen", len(l.q.optionen))
	}
	for i, o := range l.q.optionen {
		if o.Mauszeiger {
			t.Errorf("MCP-Aufnahme %d mit Zeiger", i)
		}
	}
}
