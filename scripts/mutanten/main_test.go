// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func schreibe(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(inhalt), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMusterZaehlung(t *testing.T) {
	for _, c := range []struct {
		text, alt string
		n         int
	}{
		{"a b a", "a", 2},
		{"a b", "x", 0},
		{"a b", "b", 1},
		{"abc", "", 0},
	} {
		if n := zaehle(c.text, c.alt); n != c.n {
			t.Errorf("zaehle(%q, %q) = %d, will %d", c.text, c.alt, n, c.n)
		}
	}
	if _, err := wende("a b a", Mutant{Alt: "a", Neu: "c"}); err == nil || !strings.Contains(err.Error(), "2-mal gefunden") {
		t.Errorf("zweimaliges Muster: Fehler %v", err)
	}
	if _, err := wende("a b", Mutant{Alt: "x", Neu: "c"}); err == nil || !strings.Contains(err.Error(), "0-mal gefunden") {
		t.Errorf("fehlendes Muster: Fehler %v", err)
	}
	if s, err := wende("a b a", Mutant{Alt: "b", Neu: "c"}); err != nil || s != "a c a" {
		t.Errorf("wende = %q, %v", s, err)
	}
}

func TestKopieLaesstOrdnerAus(t *testing.T) {
	q := t.TempDir()
	schreibe(t, filepath.Join(q, "go.mod"), "module x\n")
	schreibe(t, filepath.Join(q, "internal", "a", "a.go"), "package a\n")
	schreibe(t, filepath.Join(q, "third_party", "gio", "go.mod"), "module g\n")
	schreibe(t, filepath.Join(q, "draw", "go.mod"), "module d\n")
	schreibe(t, filepath.Join(q, "tmp", "mutanten", "alt.txt"), "x")
	schreibe(t, filepath.Join(q, ".git", "HEAD"), "x")
	schreibe(t, filepath.Join(q, ".claude", "settings.json"), "x")
	z := filepath.Join(t.TempDir(), "kopie")
	if err := kopiereBaum(q, z); err != nil {
		t.Fatal(err)
	}
	for _, da := range []string{"go.mod", "internal/a/a.go", "third_party/gio/go.mod", "draw/go.mod"} {
		if _, err := os.Stat(filepath.Join(z, da)); err != nil {
			t.Errorf("fehlt in der Kopie: %s", da)
		}
	}
	for _, weg := range []string{"tmp", ".git", ".claude"} {
		if _, err := os.Stat(filepath.Join(z, weg)); !os.IsNotExist(err) {
			t.Errorf("%s gehört nicht in die Kopie (Fehler %v)", weg, err)
		}
	}
}

func TestErsetzungNurInDerKopie(t *testing.T) {
	q := t.TempDir()
	schreibe(t, filepath.Join(q, "go.mod"), "module probe\n\ngo 1.27\n")
	schreibe(t, filepath.Join(q, "p", "p.go"), "package p\n\nfunc F() int { return 1 }\n")
	schreibe(t, filepath.Join(q, "p", "p_test.go"),
		"package p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n")
	schreibe(t, filepath.Join(q, "mutanten", "g.json"), `[
 {"name":"rot","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2"},
 {"name":"lebt","datei":"p/p.go","paket":"./p","alt":"func F()","neu":"func F() "},
 {"name":"bau","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return \"x\""},
 {"name":"muster","datei":"p/p.go","paket":"./p","alt":"nirgends","neu":"x"}
]`)
	vorher, _ := os.ReadFile(filepath.Join(q, "p", "p.go"))

	alle, err := ladeGruppen(filepath.Join(q, "mutanten"))
	if err != nil || len(alle) != 4 {
		t.Fatalf("ladeGruppen: %v, %d Einträge", err, len(alle))
	}
	lauf := filepath.Join(q, "tmp", "lauf")
	baum := filepath.Join(lauf, "baum")
	if err := os.MkdirAll(filepath.Join(lauf, "g"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := kopiereBaum(q, baum); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"rot": Rot, "lebt": Ueberlebt, "bau": Fehler, "muster": Fehler}
	for _, m := range alle {
		st, _, detail := testeMutant(baum, lauf, m)
		if st != want[m.Name] {
			t.Errorf("%s: %s (%s), will %s", m.Name, st, detail, want[m.Name])
		}
	}
	nachher, _ := os.ReadFile(filepath.Join(q, "p", "p.go"))
	if string(vorher) != string(nachher) {
		t.Error("der Quellbaum wurde verändert")
	}
	inKopie, _ := os.ReadFile(filepath.Join(baum, "p", "p.go"))
	if string(inKopie) != string(vorher) {
		t.Error("die Kopie wurde nicht wiederhergestellt")
	}
}

func TestWaehle(t *testing.T) {
	alle := []Mutant{{Name: "a", Gruppe: "x"}, {Name: "b", Gruppe: "x"}, {Name: "a", Gruppe: "y"}}
	if n := len(waehle(alle, "x", "")); n != 2 {
		t.Errorf("Gruppe x: %d", n)
	}
	if n := len(waehle(alle, "", "a")); n != 2 {
		t.Errorf("Name a: %d", n)
	}
	if n := len(waehle(alle, "y", "a")); n != 1 {
		t.Errorf("y/a: %d", n)
	}
}
