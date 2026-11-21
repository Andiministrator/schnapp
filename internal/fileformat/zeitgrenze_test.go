// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !race

package fileformat

import "time"

// kachelgrenzeZeit ist die Zeitgrenze für TestExportSVGKachelgrenze (F-141); unter -race entfällt sie, siehe
// zeitgrenze_race_test.go.
const kachelgrenzeZeit = 2 * time.Second
