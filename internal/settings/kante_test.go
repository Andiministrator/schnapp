// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// [capture] seam (ADR-0023 Nachtrag 6, ADR-0018 Nachtrag 3): Voreinstellung tear; tear, wave, zigzag; jeder andere Wert
// (auch andere Groß-/Kleinschreibung) warnt und lässt den bisherigen Stand.
func TestKantenEinstellung(t *testing.T) {
	pfad := umgebung(t)
	if Voreinstellungen().Kante != document.KanteAbriss {
		t.Errorf("Voreinstellung %v, Soll tear", Voreinstellungen().Kante)
	}
	for _, tc := range []struct {
		inhalt string
		kante  document.Kante
		warn   string
	}{
		{"[capture]\n", document.KanteAbriss, ""},
		{"[capture]\nseam = tear\n", document.KanteAbriss, ""},
		{"[capture]\nseam = wave\n", document.KanteWelle, ""},
		{"[capture]\nseam =  zigzag \n", document.KanteZacken, ""},
		{"[capture]\nseam = Wave\n", document.KanteAbriss, `:2: Invalid value "Wave" for key "seam"`},
		{"[capture]\nseam = wave\nseam = zickzack\n", document.KanteWelle, `:3: Invalid value "zickzack"`},
		{"[capture]\nseam = wave\nseam = zigzag\n", document.KanteZacken, `:3: Key "seam" in section [capture] is set again`},
		{"[library]\nseam = wave\n", document.KanteAbriss, `:2: Unknown key "seam" in section [library]`},
	} {
		schreibe(t, pfad, tc.inhalt)
		e, warn := Lade()
		if e.Kante != tc.kante {
			t.Errorf("%q: Kante %v, Soll %v", tc.inhalt, e.Kante, tc.kante)
		}
		switch {
		case tc.warn == "" && len(warn) != 0:
			t.Errorf("%q: Warnungen %q", tc.inhalt, warn)
		case tc.warn != "" && (len(warn) != 1 || !strings.Contains(warn[0], tc.warn)):
			t.Errorf("%q: Warnungen %q, Soll eine mit %q", tc.inhalt, warn, tc.warn)
		}
	}
}
