// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0044 Punkt 6: s:gruppe gilt ab Version 2; fehlend, ungültig oder in Version 1 heißt keine Gruppe.
func TestLeseEbeneGruppe(t *testing.T) {
	for _, f := range []struct {
		wert    string
		version int
		soll    int
	}{
		{"3", 2, 3}, {"3", 1, 0}, {"0", 2, 0}, {"-1", 2, 0}, {"x", 2, 0}, {"+4", 2, 0},
		{"1073741824", 2, document.MaxGruppe}, {"1073741825", 2, 0},
	} {
		if g := leseEbene(map[string]string{"gruppe": f.wert}, f.version).Gruppe; g != f.soll {
			t.Errorf("s:gruppe=%q Version %d: %d, Soll %d", f.wert, f.version, g, f.soll)
		}
	}
	if v := schreibVersion([]document.Objekt{document.Kasten{Ebene: document.Ebene{Gruppe: 2},
		Bereich: document.R(document.Pt(0, 0), document.Pt(4, 4)), Strich: document.Strich{Breite: 1}}}); v != 2 {
		t.Errorf("Version mit Gruppe %d, Soll 2", v)
	}
}
