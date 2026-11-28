// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !race

package ui

import "time"

// Zeitgrenzen der Editor-Zeittests im Normalbau; unter -race entfallen sie (zeitgrenze_race_test.go, F-791).
const (
	// editorBildZeit: ein ganzes Bild über die Zeichenfläche ohne Fenster (llvmpipe), großzügig (ADR-0023 Nachtrag 18).
	editorBildZeit = 2 * time.Second
	// editorGrenzBildZeit: der Fall knapp unter bereichGrenze, das teuerste Muster, das der Editor noch selbst zeichnet
	// (Füllung je Bereich, 189 585 Bereiche). Gemessen 0,4 bis 0,56 s auf 16 Kernen, 2,12 s auf dem CI-Runner (Lauf
	// #389, F-812); großzügig für Runner mit wenigen Kernen.
	editorGrenzBildZeit = 5 * time.Second
	// editorAufbauZeit: nur der Befehlsaufbau von zeichneObjekte (ADR-0023 Nachtrag 16).
	editorAufbauZeit = time.Second
	// ebeneAufbauZeit: erstes Bild, das die Objektebene des Renderers baut (F-787, ADR-0023 Nachtrag 20); gemessen
	// 0,7 bis 1,9 s auf 16 Kernen (2 Kerne: 2,3 s; F-801), der Renderer rastert auf bis zu acht. Großzügig für
	// Runner mit wenigen Kernen.
	ebeneAufbauZeit = 10 * time.Second
	// freihandZweitesBildZeit: Befehlsaufbau eines zweiten Bilds ohne Änderung mit 20 Freihand-Strichen zu 4096
	// Punkten (F-1038; ohne Merker 56 ms).
	freihandZweitesBildZeit = 10 * time.Millisecond
)
