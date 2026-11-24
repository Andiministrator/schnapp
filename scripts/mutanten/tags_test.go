// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-614 (KS-11): Das Feld „tags“ reicht zusätzliche Build-Tags an go test durch, damit auch Pakete mit Tag (Probe-Build,
// cmd/schnapp-fernbedienung) gemutet werden können.

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoFlags(t *testing.T) {
	if got := goFlags(""); got != "-tags=novulkan,nox11" {
		t.Errorf("ohne Tags: %q", got)
	}
	if got := goFlags("probe"); got != "-tags=novulkan,nox11,probe" {
		t.Errorf("mit Tag: %q", got)
	}
}

func TestUngueltigeTagsScheitern(t *testing.T) {
	for _, wert := range []string{"a b", "-x", "a,,b", ",a", "a;b", "a\\n"} {
		code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","tags":"`+wert+`"}]`)
		if code == 0 || !strings.Contains(aus, "tags") {
			t.Errorf("tags %q: Exit %d, erwartet Fehler:\n%s", wert, code, aus)
		}
	}
	if code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","tags":"probe,zwei"}]`); code != 0 {
		t.Errorf("gültige tags: Exit %d:\n%s", code, aus)
	}
}

// Der Grundlauf unterscheidet Paket-Aufrufe mit und ohne Tags.
func TestGrundlaufSchluesselTraegtTags(t *testing.T) {
	if ausgangSchluessel(Mutant{Paket: "./p"}) == ausgangSchluessel(Mutant{Paket: "./p", Tags: "probe"}) {
		t.Error("Schlüssel muss Tags unterscheiden")
	}
}

// Ein Test, der nur mit Tag „probe“ gebaut wird: mit tags ROT, ohne tags wird nichts getestet (ÜBERLEBT, Exit 1).
func TestTagsMachenGetaggtesPaketMutierbar(t *testing.T) {
	test := "//go:build probe\n\npackage p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n"
	mit := kritikWurzel(t, test, `[{"name":"rot","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","tags":"probe"}]`)
	var aus bytes.Buffer
	if code := lauf([]string{"-wurzel", mit, "-mutanten", filepath.Join(mit, "mutanten")}, &aus); code != 0 || !strings.Contains(aus.String(), "rot: ROT") {
		t.Errorf("mit tags: Exit %d, erwartet ROT:\n%s", code, aus.String())
	}
	ohne := kritikWurzel(t, test, `[{"name":"stumm","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2"}]`)
	aus.Reset()
	if code := lauf([]string{"-wurzel", ohne, "-mutanten", filepath.Join(ohne, "mutanten")}, &aus); code == 0 {
		t.Errorf("ohne tags: Exit 0, der Mutant hätte überleben müssen:\n%s", aus.String())
	}
}

// Das Feld „modul“ lässt go test in einem Unterordner mit eigenem go.mod laufen (schnapp-ocr, F-643); ohne modul fände
// go test das Paket des Untermoduls nicht. Ein modul außerhalb des Baums scheitert beim Laden.
func TestModulMachtUntermodulMutierbar(t *testing.T) {
	test := "package p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n"
	w := kritikWurzel(t, test, `[{"name":"rot","datei":"sub/p/p.go","modul":"sub","paket":"./p","alt":"return 1","neu":"return 2"}]`)
	schreibe(t, filepath.Join(w, "sub", "go.mod"), "module sub\n\ngo 1.27\n")
	schreibe(t, filepath.Join(w, "sub", "p", "p.go"), "package p\n\nfunc F() int { return 1 }\n")
	schreibe(t, filepath.Join(w, "sub", "p", "p_test.go"), test)
	var aus bytes.Buffer
	if code := lauf([]string{"-wurzel", w, "-mutanten", filepath.Join(w, "mutanten")}, &aus); code != 0 || !strings.Contains(aus.String(), "rot: ROT") {
		t.Errorf("mit modul: Exit %d, erwartet ROT:\n%s", code, aus.String())
	}
	if ausgangSchluessel(Mutant{Paket: "./p"}) == ausgangSchluessel(Mutant{Paket: "./p", Modul: "sub"}) {
		t.Error("Schlüssel muss das Modul unterscheiden")
	}
	if code, aus := trockenMit(t, `[{"name":"x","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2","modul":"../x"}]`); code == 0 || !strings.Contains(aus, "modul") {
		t.Errorf("modul außerhalb: Exit %d:\n%s", code, aus)
	}
}
