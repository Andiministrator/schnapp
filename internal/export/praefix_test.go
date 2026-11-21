// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-339): Präfix "export: " genau einmal am Anfang.
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	s := err.Error()
	if !strings.HasPrefix(s, "export: ") || strings.Count(s, "export:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"export: \"", name, s)
	}
}

func TestFehlertextPraefix(t *testing.T) {
	einmalPraefix(t, "ErrZielVorhanden", ErrZielVorhanden)
	einmalPraefix(t, "ErrZielBeschaedigt", ErrZielBeschaedigt)
	einmalPraefix(t, "errZielGewechselt", errZielGewechselt)

	// Echter Weg: das Ziel mit angehängter Endung gibt es schon.
	ordner := t.TempDir()
	if err := os.WriteFile(filepath.Join(ordner, "a.png"), []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := PruefeAngehaengt(filepath.Join(ordner, "a.png"), true)
	if !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("Fehler %v", err)
	}
	einmalPraefix(t, "PruefeAngehaengt", err)
	// SchreibeOhneErsetzen auf ein vorhandenes Ziel.
	err = SchreibeOhneErsetzen(filepath.Join(ordner, "a.png"), func(io.Writer) error { return nil })
	if !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("Fehler %v", err)
	}
	einmalPraefix(t, "SchreibeOhneErsetzen", err)
}
