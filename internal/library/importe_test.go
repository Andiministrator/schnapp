// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-0016 Nachtrag 10 Punkt 2: Die Liste der Übernahmen findet die letzte passende Zeile, übergeht kaputte Zeilen und
// Pfade, die aus der Ablage führen, und nennt nur eine reguläre Datei, die noch in der Ablage liegt.
func TestImportListe(t *testing.T) {
	b, _ := neueBibliothek(t)
	summe := []byte{1, 2, 3}
	if _, ok := b.Importiert(summe); ok {
		t.Fatal("ohne Liste gefunden")
	}
	for _, rel := range []string{"2026-11/a.png", "2026-11/b.png"} {
		p := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.MerkeImport(summe, "2026-11/a.png"); err != nil {
		t.Fatal(err)
	}
	if err := b.MerkeImport(summe, "2026-11/b.png"); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(b.importPfad(), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("kaputt\n")
	_ = f.Close()
	if rel, ok := b.Importiert(summe); !ok || rel != "2026-11/b.png" {
		t.Errorf("letzte Zeile: %q %v", rel, ok)
	}
	if fi, err := os.Stat(b.importPfad()); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Rechte der Liste: %v %v", fi.Mode(), err)
	}
	// Aus der Ablage hinaus und fehlende Dateien gelten nicht; die Datei neben der Ablage gibt es wirklich.
	if err := os.WriteFile(filepath.Join(filepath.Dir(b.Orte.Ablage), "aussen.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../aussen.png", "/etc/passwd", "2026-11/fehlt.png"} {
		if err := b.MerkeImport(summe, rel); err != nil {
			t.Fatal(err)
		}
		if got, ok := b.Importiert(summe); ok {
			t.Errorf("%q: gefunden %q", rel, got)
		}
	}
}
