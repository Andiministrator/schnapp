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
