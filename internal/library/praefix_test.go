// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-362): Präfix „library: “ am Anfang, genau einmal.
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	if s := err.Error(); !strings.HasPrefix(s, "library: ") || strings.Count(s, "library:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix „library: “ am Anfang", name, s)
	}
}

// F-362 Teil 3: Rohe Fehler des Betriebssystems verlassen das Paket mit Präfix und bleiben per errors.Is erkennbar.
func TestPraefixPaketgrenze(t *testing.T) {
	_, err := StandVon(filepath.Join(t.TempDir(), "gibt-es-nicht.png"))
	einmalPraefix(t, "StandVon", err)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("StandVon: errors.Is(fs.ErrNotExist) gilt nicht für %v", err)
	}

	b, heim := neueBibliothek(t)
	d := neuesDokument(t, 16, 16)

	// Speichern einer Datei, die es nicht gibt.
	_, _, err = b.Speichern("2026-11/gibt-es-nicht.png", Stand{}, d, d.Ansichtsbild())
	einmalPraefix(t, "Speichern", err)

	// Ablegen: Die Ablage ist eine gewöhnliche Datei, kein Ordner.
	datei := filepath.Join(heim, "ablage-ist-datei")
	if err := os.WriteFile(datei, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.Orte.Ablage = datei
	_, err = b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	einmalPraefix(t, "Ablegen", err)
	if !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("Ablegen: errors.Is(ENOTDIR) gilt nicht für %v", err)
	}
}
