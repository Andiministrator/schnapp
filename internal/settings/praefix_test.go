// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"strings"
	"testing"
)

// F-379 (ADR-0019 Nachtrag 2): Pfad ist die einzige exportierte Funktion mit Fehlerrückgabe; ohne Heimverzeichnis
// beginnt der Fehlertext mit "settings: " (genau einmal), die Ursache aus os bleibt hinter dem Präfix.
func TestPfadFehlerPraefix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	_, err := Pfad()
	if err == nil {
		t.Fatal("Pfad ohne Heimverzeichnis: kein Fehler")
	}
	if s := err.Error(); !strings.HasPrefix(s, "settings: ") || strings.Count(s, "settings:") != 1 {
		t.Errorf("Fehlertext %q ohne genau ein Präfix \"settings: \"", s)
	}
}
