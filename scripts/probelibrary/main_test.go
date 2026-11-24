// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/library"
)

// Die Probe-Bibliothek hat 14 Aufnahmen ohne Agenten und eine Agenten-Aufnahme; geliefert wird die zweitneueste.
func TestLege(t *testing.T) {
	heim := t.TempDir()
	jetzt := time.Date(2026, 11, 24, 15, 0, 0, 0, time.Local)
	zweite, err := lege(heim, jetzt)
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(library.Orte{
		Ablage:     filepath.Join(heim, "Bilder", "schnapp"),
		Index:      filepath.Join(heim, ".local", "share", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(heim, ".cache", "schnapp", "vorschau"),
		Papierkorb: filepath.Join(heim, ".local", "share", "Trash"),
	})
	ohne, err := b.Suchen(library.Anfrage{OhneAgenten: true})
	if err != nil {
		t.Fatal(err)
	}
	alle, err := b.Suchen(library.Anfrage{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ohne) != 14 || len(alle) != 15 {
		t.Fatalf("%d ohne Agenten, %d insgesamt", len(ohne), len(alle))
	}
	if bug, _ := b.Suchen(library.Anfrage{Schlagwort: "login", OhneAgenten: true}); len(bug) != 5 {
		t.Errorf("Aufnahmen mit „login“: %d", len(bug))
	}
	if want := filepath.Join(b.Orte.Ablage, filepath.FromSlash(ohne[1].Pfad)); zweite != want {
		t.Errorf("zweitneueste %q, Soll %q", zweite, want)
	}
	t.Setenv("HOME", heim)
	if _, err := lege(heim, jetzt); err == nil {
		t.Error("das echte Heim wird nicht abgelehnt")
	}
}
