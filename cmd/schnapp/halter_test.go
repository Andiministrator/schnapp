// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"testing"

	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// halter greift nur bei genau einem Helfer-Argument (ADR-0024, Textmodus Nachtrag 9); jeder andere Aufruf geht an das
// normale Programm. Der Lauf bekommt das Argument, damit er die Art wählt.
func TestHalterNurMitHelferArgument(t *testing.T) {
	gerufen, mit := 0, ""
	lauf := func(argument string) int { gerufen++; mit = argument; return 7 }
	for _, args := range [][]string{nil, {"--region"}, {wlclip.HelferArgument, "x"}, {"x", wlclip.HelferArgument},
		{wlclip.HelferArgumentText, "x"}} {
		var fehler bytes.Buffer
		if _, ok := halter(args, &fehler, lauf); ok || gerufen != 0 {
			t.Errorf("%q: ok %v, Halter %d-mal gerufen", args, ok, gerufen)
		}
	}
	if probeGuard() != nil {
		return // Probe-Build außerhalb des virtuellen KWin: dort prüft TestProbeHalterEndetVorDerZwischenablage
	}
	for i, arg := range []string{wlclip.HelferArgument, wlclip.HelferArgumentText} {
		var fehler bytes.Buffer
		if code, ok := halter([]string{arg}, &fehler, lauf); !ok || code != 7 || gerufen != i+1 || mit != arg {
			t.Errorf("%s: ok %v, Exit %d, Halter %d-mal gerufen mit %q", arg, ok, code, gerufen, mit)
		}
	}
}
