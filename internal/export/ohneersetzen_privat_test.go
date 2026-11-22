// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestSchreibeOhneErsetzenPrivat: neue Datei mit 0600; ein vorhandenes Ziel bleibt unverändert (ADR-0030 Nachtrag 4
// Punkt 2, F-462).
func TestSchreibeOhneErsetzenPrivat(t *testing.T) {
	dir := t.TempDir()
	ziel := filepath.Join(dir, "neu.png")
	schreib := func(w io.Writer) error { _, err := io.WriteString(w, "neu"); return err }
	if err := SchreibeOhneErsetzenPrivat(ziel, schreib); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(ziel); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("neue Datei: %v, %v", fi, err)
	}
	alt := filepath.Join(dir, "alt.png")
	if err := os.WriteFile(alt, []byte("alt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeOhneErsetzenPrivat(alt, schreib); !errors.Is(err, ErrZielVorhanden) {
		t.Errorf("vorhandenes Ziel: %v, erwartet ErrZielVorhanden", err)
	}
	if b, _ := os.ReadFile(alt); string(b) != "alt" {
		t.Errorf("vorhandenes Ziel verändert: %q", b)
	}
	eintraege, _ := os.ReadDir(dir)
	if len(eintraege) != 2 {
		t.Errorf("Reste im Ordner: %v", eintraege)
	}
}
