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

// KA-8: Die Warnung von Lade ist eine übersetzte Nutzerzeile; die Ursache steht darin ohne Paketpräfix.
func TestLadeWarnungOhnePraefix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	_, warn := Lade()
	if len(warn) != 1 || strings.Contains(warn[0], "settings:") || !strings.Contains(warn[0], "$HOME") {
		t.Errorf("Warnungen %q, Soll eine Zeile mit der Ursache ($HOME) ohne \"settings:\"", warn)
	}
}
