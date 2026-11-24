// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// F-725 (CONTRIBUTING.md): Jede eigene Quelldatei (Go, Shell) beginnt mit dem SPDX-Kopf, und Testdateien heißen nach
// dem, was sie prüfen, nicht nach einer Karte, Kritik oder einem Prüflauf. Fremdcode (third_party/, internal/raster16/
// mit eigener BSD-Lizenz) und der Arbeitsordner tmp/ zählen nicht.
func TestRepoRegelnSPDXUndTestnamen(t *testing.T) {
	wurzel := filepath.Join("..", "..")
	auslassen := map[string]bool{"third_party": true, "tmp": true, ".git": true, ".claude": true, "raster16": true}
	arbeitsschritt := regexp.MustCompile(`(kritik|_qa|verify|_k[0-9]+|_[0-9]{3}|_s[0-9]+)[a-z0-9]*_test\.go$`)
	geprueft := 0
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if auslassen[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".sh") {
			return nil
		}
		rel, _ := filepath.Rel(wurzel, pfad)
		if arbeitsschritt.MatchString(name) {
			t.Errorf("%s: Testdatei nach Arbeitsschritt benannt, nicht nach ihrem Gegenstand", rel)
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		kopf := string(inhalt[:min(len(inhalt), 400)])
		if !strings.Contains(kopf, "SPDX-License-Identifier: GPL-3.0-or-later") {
			t.Errorf("%s: SPDX-Kopf fehlt", rel)
		}
		geprueft++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if geprueft < 100 {
		t.Fatalf("nur %d Dateien geprüft: Wurzel falsch?", geprueft)
	}
}
