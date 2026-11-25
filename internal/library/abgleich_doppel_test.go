// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// F-773: Doppelte Indexzeilen aus der Zeit vor F-755 heilt der Abgleich (die letzte Zeile je Pfad bleibt).
func TestAbgleichHeiltDoppelteZeilen(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	alt := e
	alt.Schlagworte = []string{"alt"}
	neu := e
	neu.Schlagworte = []string{"neu"}
	f, err := os.OpenFile(b.Orte.Index, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []Eintrag{alt, neu} {
		z, _ := json.Marshal(x)
		if _, err := f.Write(append(z, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if g, _ := b.Laden(); len(g.Eintraege) != 3 {
		t.Fatalf("Vorbereitung: %d Zeilen statt 3", len(g.Eintraege))
	}
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
	g, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Eintraege) != 1 {
		t.Fatalf("nach dem Abgleich %d Zeilen statt 1", len(g.Eintraege))
	}
	if s := g.Eintraege[0].Schlagworte; len(s) != 1 || s[0] != "neu" {
		t.Errorf("nicht die letzte Zeile behalten: %v", s)
	}
	// Zweiter Abgleich: nichts mehr zu tun.
	if _, err := b.Abgleich(); err != nil {
		t.Fatal(err)
	}
}

// F-781 (Verify Sprint 13 V-1): Eine Doppelzeile für A und eine Aufnahme B ohne Indexzeile — gleich viele Zeilen fallen
// weg wie dazukommen. Der Abgleich muss trotzdem schreiben: danach je eine Zeile für A und B.
func TestAbgleichHeiltDoppelzeilePlusNeueAufnahme(t *testing.T) {
	b, _ := neueBibliothek(t)
	a := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-01T10:00:00Z")})
	bb := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-10-02T10:00:00Z")})
	z, _ := json.Marshal(a)
	zeile := append(z, '\n')
	if err := os.WriteFile(b.Orte.Index, append(append([]byte{}, zeile...), zeile...), 0o600); err != nil {
		t.Fatal(err)
	}
	if g, _ := b.Laden(); len(g.Eintraege) != 2 || g.Eintraege[0].Pfad != a.Pfad || g.Eintraege[1].Pfad != a.Pfad {
		t.Fatalf("Vorbereitung: %+v", g.Eintraege)
	}
	for lauf := 1; lauf <= 3; lauf++ {
		if _, err := b.Abgleich(); err != nil {
			t.Fatal(err)
		}
		g, err := b.Laden()
		if err != nil {
			t.Fatal(err)
		}
		var pfade []string
		for _, e := range g.Eintraege {
			pfade = append(pfade, e.Pfad)
		}
		t.Logf("nach Abgleich %d: %d Zeilen %v (A=%s, B=%s)", lauf, len(pfade), pfade, a.Pfad, bb.Pfad)
		if lauf == 3 && (len(pfade) != 2 || pfade[0] == pfade[1]) {
			t.Errorf("nach drei Abgleichen: Doppelzeile nicht geheilt oder B ohne Indexzeile: %v", pfade)
		}
	}
}
