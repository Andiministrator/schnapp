// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"testing"

	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// halter greift nur bei genau dem Helfer-Argument (ADR-0024); jeder andere Aufruf geht an das normale Programm.
func TestHalterNurMitHelferArgument(t *testing.T) {
	gerufen := 0
	lauf := func() int { gerufen++; return 7 }
	for _, args := range [][]string{nil, {"--region"}, {wlclip.HelferArgument, "x"}, {"x", wlclip.HelferArgument}} {
		var fehler bytes.Buffer
		if _, ok := halter(args, &fehler, lauf); ok || gerufen != 0 {
			t.Errorf("%q: ok %v, Halter %d-mal gerufen", args, ok, gerufen)
		}
	}
	if probeSchutz() != nil {
		return // Probe-Build außerhalb des virtuellen KWin: dort prüft TestProbeHalterEndetVorDerZwischenablage
	}
	var fehler bytes.Buffer
	if code, ok := halter([]string{wlclip.HelferArgument}, &fehler, lauf); !ok || code != 7 || gerufen != 1 {
		t.Errorf("Helfer-Argument: ok %v, Exit %d, Halter %d-mal gerufen", ok, code, gerufen)
	}
}
