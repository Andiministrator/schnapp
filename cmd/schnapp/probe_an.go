// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"os"

	"github.com/Andiministrator/schnapp/internal/probeguard"
)

// probeGuard (F-229): Der Probe-Build nimmt ohne Eingabe den ganzen ersten Monitor auf (ui.probeAutoWahl) und legt ihn
// in der Bibliothek ab. Er läuft darum nur im virtuellen KWin von scripts/probe.sh; sonst Fehler vor jeder Aufnahme.
func probeGuard() error { return probeguard.NurVirtuell(os.Getenv) }

// Im Probe-Build ist HOME das Heim des Laufs unter tmp/probe (scripts/probe.sh); Einstellungen und Bibliothek des echten
// Heims gehören nicht in eine Probe. Der MCP-Server nimmt hier darum HOME statt der Benutzerdatenbank (ADR-0030
// Nachtrag 4 Punkt 1); außerhalb des virtuellen KWin läuft er gar nicht (probeGuard).
func init() {
	benutzerHeim = func() (string, error) { return os.Getenv("HOME"), nil }
}
