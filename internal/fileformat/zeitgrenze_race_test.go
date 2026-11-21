// SPDX-License-Identifier: GPL-3.0-or-later

//go:build race

package fileformat

import "time"

// kachelgrenzeZeit ist unter dem Race-Detektor 0: Er bremst Rechenarbeit um ein Vielfaches (F-171: 6,3 s statt unter
// 2 s), die Zeitmessung sagt dort nichts über den Code. Inhalt und Kachelzahl prüft der Test weiter.
const kachelgrenzeZeit time.Duration = 0
