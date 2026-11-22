// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import "testing"

// F-415: Bei einer Drehung um 90 oder 270 Grad (auch gespiegelt) tauschen Breite und Höhe, sonst nicht.
func TestGedrehteGroesse(t *testing.T) {
	for _, c := range []struct {
		transform int
		tauscht   bool
	}{
		{0, false}, // normal
		{1, true},  // 90 Grad
		{2, false}, // 180 Grad
		{3, true},  // 270 Grad
		{4, false}, // gespiegelt
		{5, true},  // gespiegelt, 90 Grad
		{6, false}, // gespiegelt, 180 Grad
		{7, true},  // gespiegelt, 270 Grad
	} {
		b, h := gedrehteGroesse(2560, 1440, c.transform)
		wb, wh := 2560, 1440
		if c.tauscht {
			wb, wh = 1440, 2560
		}
		if b != wb || h != wh {
			t.Errorf("transform %d: %dx%d, erwartet %dx%d", c.transform, b, h, wb, wh)
		}
	}
}
