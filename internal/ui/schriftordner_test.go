// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/settings"
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
	// Die Einstellungstafel liest und schreibt den Reiter Agents am Ort des MCP-Servers im Heim der Benutzerdatenbank
	// (ADR-0034 Nachtrag 1): nie das echte Heim. Tests, die ein Heim brauchen, setzen es selbst (eigeneEinstellungen).
	settings.BenutzerHeim = func() (string, error) { return "", errors.New("tests: no home directory") }
	code := m.Run()
	os.RemoveAll(ordner)
	os.Exit(code)
}
