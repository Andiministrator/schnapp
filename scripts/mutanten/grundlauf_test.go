// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-489: HÄNGT zählt nur, wenn der Grundlauf mit demselben timeout durchläuft. F-495: eindeutiger Laufordner.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Der Mutant ändert das Verhalten nicht (überlebt in Wahrheit); der Test ist nur langsamer als der timeout. Ohne
// Grundlauf mit diesem timeout hieße das HÄNGT (rot, Exit 0); gewollt ist ein Fehler der Konfiguration, Exit 1.
func TestUeberlebenderMutantMitKnappemTimeoutIstFehler(t *testing.T) {
	q := kritikWurzel(t,
		"package p\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nfunc TestF(t *testing.T) {\n\ttime.Sleep(4 * time.Second)\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n",
		`[{"name":"gleichwertig","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2 - 1","timeout":"1s"}]`)
	var aus bytes.Buffer
	code := lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten")}, &aus)
	s := aus.String()
	if code == 0 || strings.Contains(s, "HÄNGT (Test-Timeout") || !strings.Contains(s, "TIMEOUT-ZU-KNAPP") {
		t.Errorf("Exit %d, erwartet FEHLER mit TIMEOUT-ZU-KNAPP, kein HÄNGT:\n%s", code, s)
	}
}

// Der Grundlauf läuft einmal je (Paket, timeout), nicht je Mutant.
func TestGrundlaufEinmalJeKombination(t *testing.T) {
	a := Mutant{Paket: "./p", Timeout: "5s"}
	b := Mutant{Paket: "./p", Timeout: "5s", Name: "anders"}
	c := Mutant{Paket: "./p"}
	if ausgangSchluessel(a) != ausgangSchluessel(b) || ausgangSchluessel(a) == ausgangSchluessel(c) {
		t.Error("Schlüssel muss Paket und timeout unterscheiden, nicht den Namen")
	}
}

// Zwei Läufe in derselben Sekunde (feste Uhr) und gleichzeitig: beide legen ihren Ordner an und laufen.
func TestZweiLaeufeGleicheSekunde(t *testing.T) {
	alt := jetzt
	jetzt = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	defer func() { jetzt = alt }()
	q := kritikWurzel(t,
		"package p\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n",
		`[{"name":"rot","datei":"p/p.go","paket":"./p","alt":"return 1","neu":"return 2"}]`)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	ausgabe := make([]bytes.Buffer, 2)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = lauf([]string{"-wurzel", q, "-mutanten", filepath.Join(q, "mutanten")}, &ausgabe[i])
		}()
	}
	wg.Wait()
	for i := range codes {
		if codes[i] != 0 || strings.Contains(ausgabe[i].String(), "file exists") {
			t.Errorf("Lauf %d: Exit %d:\n%s", i, codes[i], ausgabe[i].String())
		}
	}
	laeufe, err := os.ReadDir(filepath.Join(q, "tmp", "mutanten"))
	if err != nil || len(laeufe) != 2 {
		t.Errorf("erwartet 2 Laufordner, gefunden %d (%v)", len(laeufe), err)
	}
}
