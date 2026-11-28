// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !race

package canvas

import "time"

// Zeitgrenzen der Zeittests über viele gewählte Objekte im Normalbau; unter -race entfallen sie
// (zeitgrenze_race_test.go, wie internal/ui, F-791).
const (
	auswahlZeit    = 3 * time.Second // 20 Bilder mit 2000 gewählten (F-944)
	auswahlZugZeit = 5 * time.Second // Breitenzug über 2000 gewählte (F-953)
	gruppenZeit    = 3 * time.Second // Rahmen und Menüzustand bei einer Gruppe aus 2000 (F-961)
)
