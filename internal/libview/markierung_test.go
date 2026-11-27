// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/library"
)

func markListe(pfade ...string) []library.Eintrag {
	var es []library.Eintrag
	for _, p := range pfade {
		es = append(es, library.Eintrag{Pfad: p})
	}
	return es
}

func TestMarkierungUmschalten(t *testing.T) {
	es := markListe("a", "b", "c")
	var m Markierung
	m.Umschalte("b")
	if !m.Enthaelt("b") || m.Anzahl() != 1 || m.Anker() != "b" {
		t.Fatalf("nach Umschalte(b): enthält %v, Anzahl %d, Anker %q", m.Enthaelt("b"), m.Anzahl(), m.Anker())
	}
	m.Umschalte("a")
	if got := m.Pfade(es); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("Pfade = %v, will [a b] in Reihenfolge der Liste", got)
	}
	m.Umschalte("b")
	if m.Enthaelt("b") || m.Anzahl() != 1 || m.Anker() != "b" {
		t.Fatalf("zweites Umschalte(b) nimmt heraus und setzt den Anker: enthält %v, Anzahl %d, Anker %q", m.Enthaelt("b"), m.Anzahl(), m.Anker())
	}
	m.Umschalte("")
	if m.Anzahl() != 1 || m.Anker() != "b" {
		t.Fatal("Umschalte(\"\") darf nichts ändern")
	}
	if m.Enthaelt("") {
		t.Fatal("leerer Pfad ist nie markiert")
	}
}

func TestMarkierungBereich(t *testing.T) {
	es := markListe("a", "b", "c", "d", "e")
	var m Markierung
	m.Bereich(es, "c", false) // ohne Anker: nur das Ziel, es wird Anker
	if got := m.Pfade(es); !slices.Equal(got, []string{"c"}) || m.Anker() != "c" {
		t.Fatalf("ohne Anker: %v, Anker %q", got, m.Anker())
	}
	m.Bereich(es, "e", false)
	if got := m.Pfade(es); !slices.Equal(got, []string{"c", "d", "e"}) {
		t.Fatalf("Anker c bis e: %v", got)
	}
	m.Bereich(es, "a", false) // ersetzt, vom selben Anker aus rückwärts
	if got := m.Pfade(es); !slices.Equal(got, []string{"a", "b", "c"}) || m.Anker() != "c" {
		t.Fatalf("Anker c bis a ersetzt: %v, Anker %q", got, m.Anker())
	}
	m.SetzeAnker("e")
	m.Bereich(es, "e", true) // dazu: e kommt hinzu, a–c bleiben
	if got := m.Pfade(es); !slices.Equal(got, []string{"a", "b", "c", "e"}) {
		t.Fatalf("dazu: %v", got)
	}
	m.Bereich(es, "x", false) // unbekanntes Ziel: nichts
	if m.Anzahl() != 4 {
		t.Fatalf("unbekanntes Ziel änderte die Markierung: %d", m.Anzahl())
	}
	m.SetzeAnker("weg")
	m.Bereich(es, "b", false) // Anker nicht in der Liste: Ziel wird Anker
	if got := m.Pfade(es); !slices.Equal(got, []string{"b"}) || m.Anker() != "b" {
		t.Fatalf("Anker außerhalb: %v, Anker %q", got, m.Anker())
	}
}

func TestMarkierungAlleUndLeere(t *testing.T) {
	es := markListe("a", "b", "c")
	var m Markierung
	m.SetzeAnker("b")
	m.Alle(es)
	if m.Anzahl() != 3 {
		t.Fatalf("Alle: %d", m.Anzahl())
	}
	m.Leere()
	if m.Anzahl() != 0 || m.Enthaelt("a") || m.Anker() != "b" {
		t.Fatalf("Leere: Anzahl %d, Anker %q (bleibt)", m.Anzahl(), m.Anker())
	}
}

func TestMarkierungBereinige(t *testing.T) {
	var m Markierung
	m.Alle(markListe("a", "b", "c"))
	m.SetzeAnker("c")
	m.Bereinige(markListe("a", "c", "d"))
	if got := m.Pfade(markListe("a", "b", "c", "d")); !slices.Equal(got, []string{"a", "c"}) || m.Anker() != "c" {
		t.Fatalf("Bereinige: %v, Anker %q", got, m.Anker())
	}
	m.Bereinige(markListe("a"))
	if m.Anzahl() != 1 || m.Anker() != "" {
		t.Fatalf("Anker außerhalb fällt weg: Anzahl %d, Anker %q", m.Anzahl(), m.Anker())
	}
}

// Liest die Leiste bzw. das Raster neu, gilt die Markierung nur noch für Gezeigtes (ADR-0036 Zusatz Karte #361 Punkt 1).
func TestMarkierungFolgtDerListe(t *testing.T) {
	l := NeueLeiste()
	l.Setze(markListe("a", "b", "c"))
	l.Markierung().Alle(l.Eintraege())
	l.Setze(markListe("a", "c"))
	if got := l.Markierung().Pfade(l.Eintraege()); !slices.Equal(got, []string{"a", "c"}) || l.Markierung().Anzahl() != 2 {
		t.Fatalf("Leiste: %v", got)
	}
	r := NeuesRaster(nil)
	r.Setze(markListe("a", "b", "c"))
	r.Markierung().Alle(r.Eintraege())
	r.Setze(markListe("b"))
	if r.Markierung().Anzahl() != 1 || !r.Markierung().Enthaelt("b") {
		t.Fatalf("Raster: Anzahl %d", r.Markierung().Anzahl())
	}
}
