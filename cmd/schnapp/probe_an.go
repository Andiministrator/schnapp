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
