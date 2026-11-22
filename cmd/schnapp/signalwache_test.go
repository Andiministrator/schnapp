// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-396: Ein Signal während des Schreibens von --output/--export lässt keine `.schnapp-export-*` und kein Ziel zurück.

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ordnerInhalt listet die Namen im Ordner.
func ordnerInhalt(t *testing.T, dir string) []string {
	t.Helper()
	eintraege, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var namen []string
	for _, e := range eintraege {
		namen = append(namen, e.Name())
	}
	return namen
}

// Ein echtes Signal an den eigenen Prozess (von der Wache gefangen, der Test lebt weiter) kommt an, bevor geschrieben
// wird: nichts liegt im Zielordner, weder Ziel noch temporäre Datei, und der Aufruf meldet den Abbruch.
func TestSignalwacheLaesstKeinenRest(t *testing.T) {
	fangbar(t, syscall.SIGHUP)
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		dir := t.TempDir()
		ziel := filepath.Join(dir, "aus.png")
		err := schreibeInhalt(ziel, []byte("inhalt"), true, func(abgebrochen func() bool) {
			if err := syscall.Kill(os.Getpid(), sig); err != nil {
				t.Error(err)
			}
			for stop := time.Now().Add(5 * time.Second); !abgebrochen() && time.Now().Before(stop); {
				time.Sleep(time.Millisecond)
			}
		})
		if !errors.Is(err, errAbgebrochen) {
			t.Fatalf("%v: Fehler %v, Soll errAbgebrochen", sig, err)
		}
		if namen := ordnerInhalt(t, dir); len(namen) != 0 {
			t.Fatalf("%v: Zielordner nicht leer: %v", sig, namen)
		}
	}
}

// Ohne Signal schreibt die Wache nichts dazwischen: Das Ziel entsteht vollständig mit 0600, kein Rest.
func TestSignalwacheOhneSignal(t *testing.T) {
	dir := t.TempDir()
	d, err := document.Neu(testbild(20, 10, 3))
	if err != nil {
		t.Fatal(err)
	}
	abs, err := schreibeBild(d, filepath.Join(dir, "aus.png"), true)
	if err != nil {
		t.Fatal(err)
	}
	if namen := ordnerInhalt(t, dir); len(namen) != 1 || namen[0] != "aus.png" {
		t.Fatalf("Zielordner: %v", namen)
	}
	st, err := os.Stat(abs)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("Rechte %v, %v", st, err)
	}
}
