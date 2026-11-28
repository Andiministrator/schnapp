// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"testing"
)

// ADR-0044 Punkt 6: Eine Gruppe ist 0 bis MaxGruppe; anderes lehnt das Dokument ab.
func TestEbeneGruppeGrenzen(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 40, 40)))
	if err != nil {
		t.Fatal(err)
	}
	k := Kasten{Bereich: R(Pt(2, 2), Pt(10, 10)), Strich: Strich{Breite: 1}}
	for _, f := range []struct {
		g  int
		ok bool
	}{{0, true}, {1, true}, {MaxGruppe, true}, {-1, false}, {MaxGruppe + 1, false}} {
		k.Gruppe = f.g
		_, err := d.Hinzufuegen(k)
		if (err == nil) != f.ok {
			t.Errorf("Gruppe %d: Fehler %v", f.g, err)
		}
	}
}
