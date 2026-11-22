// SPDX-License-Identifier: GPL-3.0-or-later

package settings

import (
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// ADR-0018 Nachtrag 2: [general] language = auto | Sprachcode. auto und das Fehlen ergeben „“ (Umgebung); ein Wert in
// anderer Form wird gewarnt und lässt den bisherigen Stand.
func TestSprache(t *testing.T) {
	i18n.Setze(nil)
	if Voreinstellungen().Sprache != "" {
		t.Error("Voreinstellung ist nicht auto")
	}
	for _, fall := range []struct {
		wert, soll string
		warnt      bool
	}{
		{"de", "de", false}, {"en", "en", false}, {"pt_BR", "pt_BR", false}, {"fil", "fil", false},
		{"auto", "", false},
		{"German", "", true}, {"de_at", "", true}, {"de.UTF-8", "", true}, {"d", "", true}, {"deutsch", "", true},
		{"", "", true}, {"de_", "", true},
	} {
		e, w := Parse("settings.ini", "[general]\nlanguage = "+fall.wert+"\n")
		if e.Sprache != fall.soll || (len(w) > 0) != fall.warnt {
			t.Errorf("%q: Sprache %q, Warnungen %q", fall.wert, e.Sprache, w)
		}
		if fall.warnt && (len(w) != 1 || !strings.Contains(w[0], "Invalid value")) {
			t.Errorf("%q: Warnung %q", fall.wert, w)
		}
	}
	// Ein ungültiger späterer Wert lässt den früheren gültigen stehen; ein zweiter gültiger gilt mit Warnung.
	e, w := Parse("s", "[general]\nlanguage = de\nlanguage = xx-YY\n")
	if e.Sprache != "de" || len(w) != 1 {
		t.Errorf("Sprache %q, Warnungen %q", e.Sprache, w)
	}
	e, w = Parse("s", "[general]\nlanguage = de\nlanguage = auto\n")
	if e.Sprache != "" || len(w) != 1 || !strings.Contains(w[0], "set again") {
		t.Errorf("Sprache %q, Warnungen %q", e.Sprache, w)
	}
	// Der Abschnitt ist bekannt, ein fremder Schlüssel darin wird gewarnt.
	if _, w := Parse("s", "[general]\ntheme = dark\n"); len(w) != 1 || !strings.Contains(w[0], "Unknown key") {
		t.Errorf("Warnungen %q", w)
	}
}
