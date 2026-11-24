// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// TestMain macht die Tests unabhängig von den Schriften des Rechners (Sitzungs-QA B-12): Eine fremde Familie in einem
// Test löst sonst die Suche in HOME und /usr/share/fonts aus; Ergebnis und Dauer hingen vom Rechner ab.
//
// Auch die Konfiguration des Rechners bleibt außen vor: Der Editor liest styles.ini beim Öffnen und schreibt sie bei „+“
// und Entfernen der Schnellstile (ADR-0037); die Tests bekommen einen eigenen, leeren Konfigurationsordner.
func TestMain(m *testing.M) {
	fonts.SetzeOrdner(nil)
	ordner, err := os.MkdirTemp("", "schnapp-ui-konfig-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", ordner)
	code := m.Run()
	os.RemoveAll(ordner)
	os.Exit(code)
}
