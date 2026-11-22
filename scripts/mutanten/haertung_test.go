// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Sitzungs-QA 22.11.2026 (Kritiker Fable, Q-3 und Q-4); beide Tests waren rot.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func kritikWurzel(t *testing.T, testInhalt, mutanten string) string {
	t.Helper()
	q := t.TempDir()
	schreibe(t, filepath.Join(q, "go.mod"), "module probe\n\ngo 1.27\n")
	schreibe(t, filepath.Join(q, "p", "p.go"), "package p\n\n// Kommentar\nfunc F() int { return 1 }\n")
	schreibe(t, filepath.Join(q, "p", "p_test.go"), testInhalt)
	schreibe(t, filepath.Join(q, "mutanten", "g.json"), mutanten)
	return q
}

// Q-3: Das Paket ist schon ohne Mutation rot (hier: ein Test, der immer scheitert). Die Probe prüft die Ausgangslage
// nicht und wertet einen Mutanten, der nur einen Kommentar ändert, als ROT, Exit 0. Ein roter oder wackliger Test im
// Paket lässt so jede Mutante als „getötet“ erscheinen.
func TestRoteAusgangslageIstFehler(t *testing.T) {
	q := kritikWurzel(t,
		"package p\n\nimport \"testing\"\n\nfunc TestImmerRot(t *testing.T) { t.Fatal(\"rot ohne Mutation\") }\n",
		`[{"name":"kommentar","datei":"p/p.go","paket":"./p","alt":"// Kommentar","neu":"// anderer Kommentar"}]`)
	var aus bytes.Buffer
	code := lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten")}, &aus)
	if code == 0 || strings.Contains(aus.String(), "kommentar: ROT") {
		t.Errorf("Exit %d, Mutant ohne Wirkung gilt als getötet:\n%s", code, aus.String())
	}
}

// Q-4: „datei“ mit ../ verlässt die Kopie: Die Ersetzung (und das Zurückschreiben) trifft den Arbeitsbaum, entgegen
// „arbeitet nie im Arbeitsbaum“.
func TestDateiAusserhalbDesBaumsAbgelehnt(t *testing.T) {
	q := kritikWurzel(t,
		"package p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n",
		`[{"name":"raus","datei":"../../../../p/p.go","paket":"./p","alt":"return 1","neu":"return 2"}]`)
	quelle := filepath.Join(q, "p", "p.go")
	alt := time.Now().Add(-time.Hour)
	if err := os.Chtimes(quelle, alt, alt); err != nil {
		t.Fatal(err)
	}
	var aus bytes.Buffer
	lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten")}, &aus)
	st, err := os.Stat(quelle)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ModTime().Equal(alt) {
		t.Errorf("Quelldatei im Arbeitsbaum wurde beschrieben (mtime %v statt %v):\n%s", st.ModTime(), alt, aus.String())
	}
}
