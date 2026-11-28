// SPDX-License-Identifier: GPL-3.0-or-later

//go:build race

package ui

import "time"

// Unter dem Race-Detektor sind die Zeitgrenzen der Editor-Zeittests 0: gemessen und geloggt, nicht geprüft. Der Detektor
// bremst Rechenarbeit um den Faktor 9 bis 22 (knowledge/topics/lehren.md); die CI lief damit rot, obwohl der Normalbau
// die Grenze hielt (F-791). Bild und Ergebnis prüfen die Tests weiter, nur die Dauer nicht.
const (
	editorBildZeit      time.Duration = 0 // TestEditorGanzesBildUeberVielenNaehten
	editorGrenzBildZeit time.Duration = 0 // dort der Fall knapp unter bereichGrenze (F-812)
	editorAufbauZeit    time.Duration = 0 // TestEditorBildUeberVielenNaehten (Befehlsaufbau)
	ebeneAufbauZeit     time.Duration = 0 // erstes Bild mit Objektebene (F-787)
	// TestFreihandMerkerZweitesBild (F-1038)
	freihandZweitesBildZeit time.Duration = 0
)
