// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/fileformat"
)

// ADR-0022 Punkt 4 (F-126): Entfernen löscht Datei, Vorschau und Index-Eintrag endgültig (kein Papierkorb).
func TestEntfernen(t *testing.T) {
	b, heim := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Fatalf("Vorschau fehlt vor dem Test: %v", err)
	}
	if err := b.Entfernen(e.Pfad, standIn(t, b, e.Pfad)); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{abs, b.VorschauPfad(e.Pfad)} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s noch da: %v", filepath.Base(p), err)
		}
	}
	idx, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Eintraege) != 0 {
		t.Errorf("Index %+v", idx.Eintraege)
	}
	// Nichts im Papierkorb: Dort läge der ungeschwärzte Stand weiter.
	if n, _ := filepath.Glob(filepath.Join(heim, ".local", "share", "Trash", "files", "*")); len(n) != 0 {
		t.Errorf("Papierkorb: %v", n)
	}
	// Schon fehlende Datei: Vorschau und Eintrag gehen trotzdem, kein Fehler.
	if err := b.Entfernen(e.Pfad, Stand{}); err != nil {
		t.Errorf("zweites Entfernen: %v", err)
	}
}

// Eine inzwischen anders beschriebene Datei bleibt stehen (ErrGeaendert); ein Pfad außerhalb der Ablage ist ErrPfad.
func TestEntfernenNichtFremdes(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	alt := standIn(t, b, e.Pfad)
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if err := os.WriteFile(abs, []byte("anders"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := b.Entfernen(e.Pfad, alt); !errors.Is(err, ErrGeaendert) {
		t.Errorf("geänderte Datei: %v, want ErrGeaendert", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("geänderte Datei entfernt: %v", err)
	}
	if err := b.Entfernen("../ausserhalb.png", alt); !errors.Is(err, ErrPfad) {
		t.Errorf("Pfad außerhalb: %v, want ErrPfad", err)
	}
}
