// SPDX-License-Identifier: GPL-3.0-or-later

//go:build race

package canvas

import "time"

// Unter dem Race-Detektor sind die Zeitgrenzen 0: gemessen und geloggt, nicht geprüft. Der Detektor bremst um den
// Faktor 9 bis 22 (knowledge/topics/lehren.md); die CI lief damit rot (Breitenzug über 2000 Objekte, 4,2 s lokal).
const (
	auswahlZeit    time.Duration = 0
	auswahlZugZeit time.Duration = 0
	gruppenZeit    time.Duration = 0
)
