// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"fmt"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/library"
)

// eintraege liefert n Aufnahmen, neueste zuerst (wie library.Suchen).
func eintraege(n int) []library.Eintrag {
	es := make([]library.Eintrag, n)
	for i := range es {
		es[i] = library.Eintrag{Pfad: fmt.Sprintf("2026-11/a%03d.png", i), Erstellt: time.Unix(int64(1e9-i), 0), Breite: 10, Hoehe: 10}
	}
	return es
}

// m: Rand 10, Kachel 100, Lücke 10, sichtbar 340 px — drei Kacheln und ein Stück der vierten.
var m = Masse{Rand: 10, Kachel: 100, Luecke: 10, Breite: 340}

// ADR-0036 Punkt 1: Agenten-Aufnahmen fehlen, die Liste ist begrenzt, die Reihenfolge bleibt (neueste links).
func TestLeisteSetze(t *testing.T) {
	es := eintraege(5)
	es[1].Pfad = "agents/2026-11/x.png"
	l := NeueLeiste()
	l.Setze(es)
	if got := l.Eintraege(); len(got) != 4 || got[0].Pfad != es[0].Pfad || got[1].Pfad != es[2].Pfad {
		t.Errorf("ohne Agenten: %v", got)
	}
	l.Setze(eintraege(MaxEintraege + 5))
	if n := len(l.Eintraege()); n != MaxEintraege {
		t.Errorf("Grenze: %d", n)
	}
	if l.Eintraege()[0].Pfad != "2026-11/a000.png" {
		t.Error("die neuesten müssen bleiben")
	}
}

// Der Fokus bleibt bei derselben Aufnahme, wenn der Index neu gelesen wird (eine neue kommt links dazu); verschwindet
// sie, rückt er an die letzte vorhandene Stelle.
func TestLeisteFokusBleibt(t *testing.T) {
	l := NeueLeiste()
	es := eintraege(5)
	l.Setze(es[1:])
	l.SetzeFokus(1) // a002
	l.Setze(es)
	if l.Fokus() != 2 {
		t.Errorf("Fokus nach neuer Aufnahme: %d", l.Fokus())
	}
	l.SetzeFokus(4)
	l.Setze(es[:2])
	if l.Fokus() != 1 {
		t.Errorf("Fokus nach Wegfall: %d", l.Fokus())
	}
	l.Setze(nil)
	if l.Fokus() != -1 {
		t.Errorf("Fokus bei leerer Liste: %d", l.Fokus())
	}
	l2 := NeueLeiste()
	l2.Setze(es)
	if l2.Fokus() != -1 {
		t.Error("ohne Fokus entsteht keiner")
	}
	l2.SetzeFokus(9)
	if l2.Fokus() != -1 {
		t.Error("Fokus außerhalb der Liste")
	}
}

func TestLeisteAktuell(t *testing.T) {
	l := NeueLeiste()
	l.Setze(eintraege(3))
	if l.IstAktuell(0) {
		t.Error("ohne offene Aufnahme ist keine aktuell")
	}
	l.SetzeAktuell("2026-11/a001.png")
	if !l.IstAktuell(1) || l.IstAktuell(0) || l.IstAktuell(5) || l.Aktuell() != "2026-11/a001.png" {
		t.Error("aktuell falsch")
	}
}

func TestLeisteBewege(t *testing.T) {
	l := NeueLeiste()
	if l.Bewege(0, SchrittRechts) != -1 {
		t.Error("leere Liste")
	}
	l.Setze(eintraege(4))
	for _, f := range []struct {
		von  int
		s    Schritt
		soll int
	}{{0, SchrittLinks, 0}, {0, SchrittRechts, 1}, {3, SchrittRechts, 3}, {2, SchrittLinks, 1}, {2, SchrittAnfang, 0}, {1, SchrittEnde, 3}} {
		if ist := l.Bewege(f.von, f.s); ist != f.soll {
			t.Errorf("von %d mit %d: %d, Soll %d", f.von, f.s, ist, f.soll)
		}
	}
}

// Lage, Rollen, sichtbarer Bereich und Zeigen in Pixeln.
func TestLeisteLageUndRollen(t *testing.T) {
	l := NeueLeiste()
	l.Setze(eintraege(10)) // Spur: 2·10 + 10·100 + 9·10 = 1110
	if r := l.RollMax(m); r != 1110-340 {
		t.Errorf("RollMax %d", r)
	}
	if x0, x1 := l.Lage(2, m); x0 != 230 || x1 != 330 {
		t.Errorf("Lage 2: %d %d", x0, x1)
	}
	if v, b := l.Sichtbar(m); v != 0 || b != 5 {
		t.Errorf("sichtbar am Anfang: %d bis %d", v, b)
	}
	l.Rad(225.5, m)
	if l.Rollen(m) != 225 {
		t.Errorf("Rad: %d", l.Rollen(m))
	}
	l.Rad(0.5, m) // der Rest zählt
	if l.Rollen(m) != 226 {
		t.Errorf("Rad mit Rest: %d", l.Rollen(m))
	}
	if x0, _ := l.Lage(2, m); x0 != 230-226 {
		t.Errorf("Lage nach dem Rollen: %d", x0)
	}
	// Sichtbar 226 … 566: Kacheln 2 (230–330) bis 5 (560–660); Vorlauf 1 und 6.
	if v, b := l.Sichtbar(m); v > 1 || b < 7 || v < 0 || b > 8 {
		t.Errorf("sichtbar: %d bis %d", v, b)
	}
	l.Rad(-5000, m)
	if l.Rollen(m) != 0 {
		t.Errorf("Anschlag links: %d", l.Rollen(m))
	}
	l.Rad(5000, m)
	if l.Rollen(m) != l.RollMax(m) {
		t.Errorf("Anschlag rechts: %d", l.Rollen(m))
	}
	if v, b := l.Sichtbar(m); b != 10 || v > 6 {
		t.Errorf("sichtbar am Ende: %d bis %d", v, b)
	}
	l.Zeige(0, m)
	if l.Rollen(m) != 0 {
		t.Errorf("Zeige 0: %d", l.Rollen(m))
	}
	l.Zeige(5, m) // rechte Kante 660 + Rand 10 − 340
	if l.Rollen(m) != 330 {
		t.Errorf("Zeige 5: %d", l.Rollen(m))
	}
	l.Zeige(4, m) // schon sichtbar: nichts
	if l.Rollen(m) != 330 {
		t.Errorf("Zeige 4: %d", l.Rollen(m))
	}
	l.Zeige(3, m) // linke Kante 340 − Rand
	if l.Rollen(m) != 330 {
		t.Errorf("Zeige 3: %d", l.Rollen(m))
	}
	l.Zeige(2, m)
	if l.Rollen(m) != 220 {
		t.Errorf("Zeige 2: %d", l.Rollen(m))
	}
	// Eine kürzere Liste begrenzt den Versatz.
	l.Setze(eintraege(3))
	if l.Rollen(m) != 0 || l.RollMax(m) != 0 {
		t.Errorf("kurze Liste: %d %d", l.Rollen(m), l.RollMax(m))
	}
	if v, b := l.Sichtbar(m); v != 0 || b != 3 {
		t.Errorf("kurze Liste sichtbar: %d bis %d", v, b)
	}
	l.Setze(nil)
	if v, b := l.Sichtbar(m); v != 0 || b != 0 {
		t.Errorf("leer: %d bis %d", v, b)
	}
}

// Der Sichtbereich enthält jede Kachel, die den sichtbaren Ausschnitt berührt, bei jedem Versatz.
func TestLeisteSichtbarDeckt(t *testing.T) {
	l := NeueLeiste()
	l.Setze(eintraege(20))
	for r := 0; r <= l.RollMax(m); r += 7 {
		l.rollen = r
		v, b := l.Sichtbar(m)
		for i := range 20 {
			x0, x1 := l.Lage(i, m)
			if x1 > 0 && x0 < m.Breite && (i < v || i >= b) {
				t.Fatalf("Versatz %d: Kachel %d sichtbar, aber nicht in %d bis %d", r, i, v, b)
			}
		}
		if b-v > 6 {
			t.Fatalf("Versatz %d: zu viele (%d bis %d)", r, v, b)
		}
	}
}
