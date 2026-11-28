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
	for _, args := range [][]string{nil, {"--region"}, {ui.NeueAufnahmeArgument, "x"}, {"mcp"}} {
		ab = nil
		if got := neueAufnahmeArgs(args, schliesse); !slices.Equal(got, args) || ab != nil {
			t.Errorf("%v: %v, geschlossen %v", args, got, ab)
		}
	}
}
