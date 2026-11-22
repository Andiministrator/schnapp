// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !race

package render

import "time"

// zeichenZeit ist die Zeitgrenze der Tests in zeichen_test.go (F-224); unter -race entfällt sie, siehe
// zeitgrenze_race_test.go.
const zeichenZeit = 20 * time.Millisecond

// pruefeNahtZeit ist die Zeitgrenze von TestPruefeVieleAbrissNaehte (F-418): großzügig, gemessen rund 2 s (vorher
// rund 52 s); unter -race entfällt der Test.
const pruefeNahtZeit = 15 * time.Second
