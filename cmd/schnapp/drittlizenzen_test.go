// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
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
