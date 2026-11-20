// SPDX-License-Identifier: GPL-3.0-or-later

//go:build race

package render

import "time"

// zeichenZeit ist unter dem Race-Detektor 0: Er bremst Rechenarbeit um ein Vielfaches (wie F-171), die Zeitmessung
// sagt dort nichts über den Code. Ablehnung und Ergebnis prüfen die Tests weiter.
const zeichenZeit time.Duration = 0
