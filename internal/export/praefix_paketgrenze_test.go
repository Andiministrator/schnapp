// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
)

// F-362 Teil 3: Rohe Fehler von Betriebssystem und Schreib-Rückruf verlassen das Paket mit Präfix „export: “ und bleiben
// per errors.Is erkennbar.
func TestPraefixPaketgrenze(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "gibt-es-nicht", "a.png")

	// Zielordner fehlt: die temporäre Datei lässt sich nicht anlegen (fs.ErrNotExist, nicht der Rückfall).
	err := SchreibeAtomar(ziel, func(io.Writer) error { return nil })
	einmalPraefix(t, "SchreibeAtomar", err)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("SchreibeAtomar: errors.Is(fs.ErrNotExist) gilt nicht für %v", err)
	}
	err = SchreibeOhneErsetzen(ziel, func(io.Writer) error { return nil })
	einmalPraefix(t, "SchreibeOhneErsetzen", err)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("SchreibeOhneErsetzen: errors.Is(fs.ErrNotExist) gilt nicht für %v", err)
	}

	// Fehler des Rückrufs (fremd, ohne Präfix) kommen gehüllt, die Ursache bleibt.
	ausfall := errors.New("Rückruf kaputt")
	err = SchreibeAtomar(filepath.Join(ordner, "b.png"), func(io.Writer) error { return ausfall })
	einmalPraefix(t, "SchreibeAtomar Rückruf", err)
	if !errors.Is(err, ausfall) {
		t.Errorf("SchreibeAtomar: Ursache verloren: %v", err)
	}
}
