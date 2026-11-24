// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
)

// Karte #186: Die Schilder des Overlays sind eckig, keine Ecke ist gerundet.
func TestSchildUmrissEckig(t *testing.T) {
	r := schildUmriss(image.Pt(120, 30))
	if r.Rect != image.Rect(0, 0, 120, 30) {
		t.Errorf("Rechteck %v", r.Rect)
	}
	if r.NE != 0 || r.NW != 0 || r.SE != 0 || r.SW != 0 {
		t.Errorf("Ecken gerundet: %+v", r)
	}
}
