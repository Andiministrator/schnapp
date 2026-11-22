// SPDX-License-Identifier: GPL-3.0-or-later

package i18n_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/po"
)

// ADR-0014 Nachtrag 2: Datum und Zoom folgen der Sprache über den Katalog — englisch „2026-11-22 15:04“ und „100%“,
// deutsch „22.11.2026 15:04“ und „100 %“. Dieselben Literale wie in cmd/schnapp/rettung.go und internal/ui/editor.go.
func TestFormateJeSprache(t *testing.T) {
	zeit := time.Date(2026, 11, 22, 15, 4, 0, 0, time.UTC)
	for _, fall := range []struct{ sprache, datum, status string }{
		{"en", "2026-11-22 15:04", "1200 × 800 · 100%"},
		{"de", "22.11.2026 15:04", "1200 × 800 · 100 %"},
	} {
		k, err := i18n.Lade(po.Dateien, fall.sprache)
		if err != nil {
			t.Fatal(err)
		}
		i18n.Setze(k)
		if d := zeit.Format(i18n.TK("date format", "2006-01-02 15:04")); d != fall.datum {
			t.Errorf("%s: Datum %q, Soll %q", fall.sprache, d, fall.datum)
		}
		if s := fmt.Sprintf(i18n.TK("status bar", "%d × %d · %d%%"), 1200, 800, 100); s != fall.status {
			t.Errorf("%s: Status %q, Soll %q", fall.sprache, s, fall.status)
		}
	}
	i18n.Setze(nil)
}
