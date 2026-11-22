// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// KA-4: SchreibeAtomarPrivat (Kommandozeile --output) legt eine neue Datei mit 0600 an, auch bei offener umask, und
// schon während des Schreibens; eine vorhandene Zieldatei behält ihre Rechte. SchreibeAtomar bleibt bei der umask
// (TestSchreibeAtomarNeueDatei).
func TestSchreibeAtomarPrivat(t *testing.T) {
	alt := syscall.Umask(0o022)
	defer syscall.Umask(alt)
	ordner := t.TempDir()
	neu := filepath.Join(ordner, "neu.png")
	var waehrend os.FileMode
	err := SchreibeAtomarPrivat(neu, func(w io.Writer) error {
		st, err := w.(*os.File).Stat()
		if err != nil {
			return err
		}
		waehrend = st.Mode().Perm()
		_, err = io.WriteString(w, "inhalt")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(neu)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 || waehrend != 0o600 {
		t.Errorf("neue Datei %v, während des Schreibens %v; Soll 0600", st.Mode().Perm(), waehrend)
	}
	// Vorhandene Datei mit 0644: bleibt 0644 (der Nutzer hat sie so gewollt).
	vorhanden := filepath.Join(ordner, "vorhanden.png")
	if err := os.WriteFile(vorhanden, []byte("alt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(vorhanden, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeAtomarPrivat(vorhanden, func(w io.Writer) error { _, err := io.WriteString(w, "neu"); return err }); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(vorhanden); err != nil || st.Mode().Perm() != 0o644 {
		t.Errorf("vorhandene Datei: %v, %v; Soll 0644", st.Mode().Perm(), err)
	}
	if b, _ := os.ReadFile(vorhanden); string(b) != "neu" {
		t.Errorf("Inhalt %q", b)
	}
}
