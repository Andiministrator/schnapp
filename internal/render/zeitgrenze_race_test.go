// SPDX-License-Identifier: GPL-3.0-or-later

//go:build race

package render

import "time"

// zeichenZeit ist unter dem Race-Detektor 0: Er bremst Rechenarbeit um ein Vielfaches (wie F-171), die Zeitmessung
// sagt dort nichts über den Code. Ablehnung und Ergebnis prüfen die Tests weiter.
const zeichenZeit time.Duration = 0

// pruefeNahtZeit ist unter dem Race-Detektor 0: TestPruefeVieleAbrissNaehte (F-418) entfällt dort ganz.
const pruefeNahtZeit time.Duration = 0

// pruefeStreifenZeit ist unter dem Race-Detektor 0: TestPruefeVieleStreifenUndPfeile loggt die Dauer nur (F-802).
const pruefeStreifenZeit time.Duration = 0

// musterZerlegZeit und musterPruefeZeit sind unter dem Race-Detektor 0: die Muster-Tests loggen die Dauer nur.
const (
	musterZerlegZeit time.Duration = 0
	musterPruefeZeit time.Duration = 0
)
