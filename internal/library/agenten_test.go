// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// TestAblegenAgent: Agenten-Aufnahmen liegen unter agents/<JJJJ-MM>/ mit den Rechten der Ablage, tragen den
// Index-Pfad agents/… und lassen sich in der Suche ausblenden; der Neuaufbau findet sie (ADR-0030 Nachtrag 2).
func TestAblegenAgent(t *testing.T) {
	b, _ := neueBibliothek(t)
	ablage := b.Orte.Ablage
	d := neuesDokument(t, 40, 30)
	e, err := b.AblegenAgent(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	if err != nil {
		t.Fatal(err)
	}
	if want := "agents/2026-11/2026-11-19_14-35-12.png"; e.Pfad != want || !e.Agent() {
		t.Fatalf("Pfad %q (Agent %v), erwartet %q", e.Pfad, e.Agent(), want)
	}
	for _, o := range []string{filepath.Join(ablage, "agents"), filepath.Join(ablage, "agents", "2026-11")} {
		if m := modus(t, o); m != 0o700 {
			t.Errorf("%s: Rechte %o, erwartet 700", o, m)
		}
	}
	if m := modus(t, filepath.Join(ablage, "agents", "2026-11", "2026-11-19_14-35-12.png")); m != 0o600 {
		t.Errorf("Datei: Rechte %o, erwartet 600", m)
	}
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Errorf("Vorschau fehlt: %v", err)
	}

	n, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T13:00:00Z")})
	if err != nil || n.Agent() {
		t.Fatalf("normale Aufnahme: %+v, %v", n, err)
	}
	alle, err := b.Suchen(Anfrage{})
	if err != nil || len(alle) != 2 {
		t.Fatalf("Suche ohne Filter: %d Einträge, %v", len(alle), err)
	}
	ohne, err := b.Suchen(Anfrage{OhneAgenten: true})
	if err != nil || len(ohne) != 1 || ohne[0].Pfad != n.Pfad {
		t.Fatalf("Suche ohne Agenten: %+v, %v", ohne, err)
	}

	erg, err := b.Neuaufbau()
	if err != nil || erg.Aufgenommen != 2 {
		t.Fatalf("Neuaufbau: %+v, %v", erg, err)
	}
	neu, _ := b.Suchen(Anfrage{})
	if len(neu) != 2 || !neu[1].Agent() && !neu[0].Agent() {
		t.Errorf("nach dem Neuaufbau fehlt die Agenten-Aufnahme: %+v", neu)
	}
}

// TestAgentNurMitPraefix: Ein Monatsordner, der zufällig mit „agents“ beginnt, ist keine Agenten-Aufnahme.
func TestAgentNurMitPraefix(t *testing.T) {
	for pfad, want := range map[string]bool{
		"agents/2026-11/a.png": true, "agentsx/2026-11/a.png": false, "2026-11/agents.png": false, "agents": false,
	} {
		if got := (Eintrag{Pfad: pfad, Erstellt: time.Now()}).Agent(); got != want {
			t.Errorf("%q: Agent %v, erwartet %v", pfad, got, want)
		}
	}
}
