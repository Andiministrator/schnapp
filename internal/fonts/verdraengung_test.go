// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// familienAus legt n Schriftdateien (Kopien der Testschrift) unter verschiedenen Namen an und trägt je eine Familie
// F00, F01, … in die Suchergebnisse ein; die Namensliste der Familien kommt zurück.
func familienAus(t *testing.T, n int) []string {
	t.Helper()
	// Die Verdrängungstests laden in schneller Folge; ohne Ruhefrist (sie prüft TestRuhefristVerdraengtNichtImGebrauch).
	alt := ruhefrist
	ruhefrist = 0
	t.Cleanup(func() { ruhefrist = alt })
	roh, err := os.ReadFile("../../testdata/fonts/schnapp-testkern.ttf")
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	mitOrdnern(t, d) // leere Suche, räumt danach auf
	var namen []string
	mu.Lock()
	defer mu.Unlock()
	for i := range n {
		name := fmt.Sprintf("F%02d", i)
		pfad := filepath.Join(d, name+".ttf")
		if err := os.WriteFile(pfad, roh, 0o600); err != nil {
			t.Fatal(err)
		}
		system = append(system, familie{name: name, schnitte: [4]quelle{{da: true, pfad: pfad}}})
		namen = append(namen, name)
	}
	return namen
}

// TestGrenzeVerdraengtStattZuSperren (F-697): Die Grenze von MaxGeladen Dateien ist ein Zwischenspeicher. Wer nacheinander
// mehr Schriftdateien lädt (Wechsel der Aufnahme im selben Fenster), bekommt auch die letzte, nicht Go als Ersatz.
func TestGrenzeVerdraengtStattZuSperren(t *testing.T) {
	namen := familienAus(t, MaxGeladen+5)
	for _, n := range namen {
		if s := Laden(n, Normal); s.Ersatz || s.Familie != n {
			t.Fatalf("%s: %+v", n, s)
		}
		mu.Lock()
		if len(dateien) > MaxGeladen {
			t.Errorf("%s: %d Dateien im Speicher, Grenze %d", n, len(dateien), MaxGeladen)
		}
		mu.Unlock()
	}
	// Eine schon ausgegebene Schrift bleibt benutzbar, auch wenn sie verdrängt wurde.
	mu.Lock()
	_, da := dateien[system[0].schnitte[0].pfad]
	mu.Unlock()
	if da {
		t.Error("die älteste Datei ist nicht verdrängt worden")
	}
	if s := Laden(namen[0], Normal); s.Ersatz || s.Font == nil {
		t.Errorf("verdrängte Familie lädt nicht wieder: %+v", s)
	}
}

// TestVerdraengtDieLangUnbenutzte (F-697): Die Verdrängung trifft die am längsten unbenutzte Datei, nicht die, die
// gerade gebraucht wird.
func TestVerdraengtDieLangUnbenutzte(t *testing.T) {
	namen := familienAus(t, MaxGeladen+1)
	for _, n := range namen[:MaxGeladen] {
		Laden(n, Normal)
	}
	Laden(namen[0], Normal) // F00 ist jetzt die jüngste, F01 die älteste
	Laden(namen[MaxGeladen], Normal)
	mu.Lock()
	defer mu.Unlock()
	if _, da := dateien[system[0].schnitte[0].pfad]; !da {
		t.Error("die zuletzt benutzte Datei wurde verdrängt")
	}
	if _, da := dateien[system[1].schnitte[0].pfad]; da {
		t.Error("die am längsten unbenutzte Datei blieb")
	}
}

// TestVerdraengenNebenlaeufig (F-697): Paralleles Laden und Verdrängen läuft unter -race ohne Ersatz und ohne Absturz.
func TestVerdraengenNebenlaeufig(t *testing.T) {
	namen := familienAus(t, 2*MaxGeladen)
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 3 * len(namen) {
				n := namen[(i*7+g*5)%len(namen)]
				if s := Laden(n, Normal); s == nil || s.Ersatz || s.Font == nil {
					t.Errorf("%s: %+v", n, s)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// Kritik B-1: Eine Datei, die vor weniger als ruhefrist benutzt wurde, wird nicht verdrängt. Braucht ein Bild mehr als
// MaxGeladen Dateien zugleich, bekommt die überzählige Ersatz Go, statt dass sich das Laden je Bild im Kreis dreht und
// jedes Mal eine neue Fassung der Datei liest.
func TestRuhefristVerdraengtNichtImGebrauch(t *testing.T) {
	namen := familienAus(t, MaxGeladen+1)
	ruhefrist = time.Hour
	erste := map[string]*Schrift{}
	for _, n := range namen[:MaxGeladen] {
		erste[n] = Laden(n, Normal)
	}
	if s := Laden(namen[MaxGeladen], Normal); !s.Ersatz {
		t.Errorf("Datei %d bei %d Dateien in Gebrauch geladen statt Ersatz", MaxGeladen+1, MaxGeladen)
	}
	for _, n := range namen[:MaxGeladen] {
		if s := Laden(n, Normal); s.Font != erste[n].Font {
			t.Errorf("%s: in Gebrauch verdrängt und neu gelesen", n)
		}
	}
}
