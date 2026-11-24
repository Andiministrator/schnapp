// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// F-719: THIRD-PARTY-LICENSES/README.md nennt genau die Fremdmodule, die in schnapp eingebaut werden, mit derselben
// Fassung, und jede genannte Datei liegt im Ordner. Ein neues oder gehobenes Modul lässt den Test scheitern, bis
// Lizenztext und Tabelle nachgezogen sind.
func TestDrittlizenzenWieModule(t *testing.T) {
	gehe, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go nicht gefunden: %v", err)
	}
	cmd := exec.Command(gehe, "list", "-deps", "-f", "{{with .Module}}{{.Path}} {{.Version}}{{end}}", ".")
	cmd.Env = append(os.Environ(), "GOFLAGS=-tags=novulkan,nox11")
	aus, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	module := map[string]string{}
	for _, z := range strings.Split(string(aus), "\n") {
		f := strings.Fields(z)
		if len(f) == 2 { // das eigene Modul hat keine Fassung
			module[f[0]] = f[1]
		}
	}
	if len(module) == 0 {
		t.Fatal("go list lieferte keine Fremdmodule")
	}

	ordner := filepath.Join("..", "..", "THIRD-PARTY-LICENSES")
	readme, err := os.ReadFile(filepath.Join(ordner, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	zeile := regexp.MustCompile("^\\| ([a-z0-9./-]+) \\| (v[^ ]+) [^|]*\\|[^|]*\\| (.+) \\|$")
	datei := regexp.MustCompile("`([^`]+)`")
	tabelle := map[string]string{}
	for _, z := range strings.Split(string(readme), "\n") {
		if strings.HasPrefix(z, "| Go standard library") {
			for _, d := range datei.FindAllStringSubmatch(z, -1) {
				if strings.HasSuffix(d[1], ".txt") {
					pruefeDatei(t, ordner, d[1])
				}
			}
			continue
		}
		m := zeile.FindStringSubmatch(z)
		if m == nil {
			continue
		}
		tabelle[m[1]] = m[2]
		dateien := datei.FindAllStringSubmatch(m[3], -1)
		if len(dateien) == 0 {
			t.Errorf("%s: keine Datei genannt", m[1])
		}
		for _, d := range dateien {
			pruefeDatei(t, ordner, d[1])
		}
	}
	for p, v := range module {
		if tv, ok := tabelle[p]; !ok {
			t.Errorf("Modul %s %s fehlt in THIRD-PARTY-LICENSES/README.md", p, v)
		} else if tv != v {
			t.Errorf("Modul %s: Fassung %s, in der Tabelle %s", p, v, tv)
		}
	}
	for p := range tabelle {
		if _, ok := module[p]; !ok {
			t.Errorf("Tabelle nennt %s, das nicht mehr eingebaut wird", p)
		}
	}
}

func pruefeDatei(t *testing.T, ordner, name string) {
	t.Helper()
	if st, err := os.Stat(filepath.Join(ordner, name)); err != nil || st.Size() == 0 {
		t.Errorf("Lizenzdatei %s fehlt oder ist leer: %v", name, err)
	}
}

// Kritik C-3 (F-719): Lizenz-, Copying-, Notice- und Patents-Dateien liegen auch unterhalb der Modulwurzel (etwa
// go-text/typesetting/harfbuzz, MIT; x/exp/shiny/materialdesign/icons, Apache-2.0). Jede solche Datei in der Ordnerkette
// eines eingebauten Pakets bis zur Modulwurzel muss inhaltsgleich in THIRD-PARTY-LICENSES liegen.
func TestDrittlizenzenAufPaketebene(t *testing.T) {
	gehe, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go nicht gefunden: %v", err)
	}
	cmd := exec.Command(gehe, "list", "-deps", "-f", "{{.Dir}}|{{with .Module}}{{if .Version}}{{.Dir}}{{end}}{{end}}", ".")
	cmd.Env = append(os.Environ(), "GOFLAGS=-tags=novulkan,nox11")
	aus, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	ordner := filepath.Join("..", "..", "THIRD-PARTY-LICENSES")
	eintraege, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatal(err)
	}
	var vorhanden [][]byte
	for _, e := range eintraege {
		if b, err := os.ReadFile(filepath.Join(ordner, e.Name())); err == nil {
			vorhanden = append(vorhanden, b)
		}
	}
	lizenz := regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|patents|unlicense)`)
	geprueft := map[string]bool{}
	for _, z := range strings.Split(string(aus), "\n") {
		dir, modul, ok := strings.Cut(z, "|")
		if !ok || modul == "" {
			continue // Standardbibliothek und eigenes Modul (go-LICENSE.txt bzw. GPL)
		}
		for d := dir; ; d = filepath.Dir(d) {
			if !geprueft[d] {
				geprueft[d] = true
				dateien, _ := os.ReadDir(d)
				for _, f := range dateien {
					if f.IsDir() || !lizenz.MatchString(f.Name()) {
						continue
					}
					inhalt, err := os.ReadFile(filepath.Join(d, f.Name()))
					if err != nil {
						t.Fatal(err)
					}
					gefunden := false
					for _, v := range vorhanden {
						if bytes.Equal(v, inhalt) {
							gefunden = true
							break
						}
					}
					if !gefunden {
						t.Errorf("Lizenzdatei %s fehlt in THIRD-PARTY-LICENSES", filepath.Join(d, f.Name()))
					}
				}
			}
			if d == modul || len(d) <= len(modul) {
				break
			}
		}
	}
	if len(geprueft) == 0 {
		t.Fatal("keine Fremdpakete geprüft")
	}
}
