// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/ui"
)

// ADR-0046 Punkt 7, Kritik K-7: Nur genau das interne Argument schließt geerbte Deskriptoren ab 3 und nimmt dann auf
// wie ohne Argumente; jede andere Liste bleibt unberührt.
func TestNeueAufnahmeArgs(t *testing.T) {
	var ab []int
	schliesse := func(n int) []int { ab = append(ab, n); return nil }
	if got := neueAufnahmeArgs([]string{ui.NeueAufnahmeArgument}, schliesse); len(got) != 0 || !slices.Equal(ab, []int{3}) {
		t.Errorf("internes Argument: %v, geschlossen ab %v", got, ab)
	}
	// ADR-0042 Nachtrag 2: genau ein --delay=… danach geht an die gewöhnliche Prüfung weiter.
	ab = nil
	if got := neueAufnahmeArgs([]string{ui.NeueAufnahmeArgument, "--delay=5"}, schliesse); !slices.Equal(got, []string{"--delay=5"}) ||
		!slices.Equal(ab, []int{3}) {
		t.Errorf("mit Verzögerung: %v, geschlossen ab %v", got, ab)
	}
	for _, args := range [][]string{nil, {"--region"}, {ui.NeueAufnahmeArgument, "x"}, {"mcp"},
		{ui.NeueAufnahmeArgument, "--delay=5", "--window"}, {ui.NeueAufnahmeArgument, "--delay"}, {"--delay=5", ui.NeueAufnahmeArgument}} {
		ab = nil
		if got := neueAufnahmeArgs(args, schliesse); !slices.Equal(got, args) || ab != nil {
			t.Errorf("%v: %v, geschlossen %v", args, got, ab)
		}
	}
}
