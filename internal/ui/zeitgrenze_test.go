// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !race

package ui

import "time"

// Zeitgrenzen der Editor-Zeittests im Normalbau; unter -race entfallen sie (zeitgrenze_race_test.go, F-791).
const (
	// editorBildZeit: ein ganzes Bild über die Zeichenfläche ohne Fenster (llvmpipe), großzügig (ADR-0023 Nachtrag 18).
	editorBildZeit = 2 * time.Second
	// editorAufbauZeit: nur der Befehlsaufbau von zeichneObjekte (ADR-0023 Nachtrag 16).
	editorAufbauZeit = time.Second
)
