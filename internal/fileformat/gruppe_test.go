// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"strings"
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
	k := document.Kasten{Ebene: document.Ebene{Gruppe: 2}, Bereich: document.R(document.Pt(0, 0), document.Pt(4, 4)),
		Strich: document.Strich{Breite: 1}}
	if v := schreibVersion([]document.Objekt{k, k}); v != 2 {
		t.Errorf("Version mit Gruppe %d, Soll 2", v)
	}
	// Ein einzelnes Restmitglied ist keine Gruppe: Version 1, kein s:gruppe (F-962).
	if v := schreibVersion([]document.Objekt{k}); v != 1 {
		t.Errorf("Version mit einzelner Kennung %d, Soll 1", v)
	}
	s := schreiber{rundlauf: true}
	s.ebene(k.Ebene)
	if strings.Contains(s.String(), "s:gruppe") {
		t.Errorf("einzelne Kennung geschrieben: %s", s.String())
	}
	s = schreiber{rundlauf: true, gruppen: wirksameGruppen([]document.Objekt{k, k})}
	s.ebene(k.Ebene)
	if !strings.Contains(s.String(), `s:gruppe="2"`) {
		t.Errorf("wirksame Gruppe nicht geschrieben: %s", s.String())
	}
}

// F-963: Die Gruppe überlebt die Angaben der Rettungskopie (AngabenSVG → LeseAngaben).
func TestGruppeRundlaufAngaben(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 40)))
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Ebene: document.Ebene{Gruppe: 7}, Bereich: document.R(document.Pt(2, 2), document.Pt(10, 10)),
		Strich: document.Strich{Breite: 1}}
	for i := 0; i < 2; i++ {
		if _, err := d.Hinzufuegen(k); err != nil {
			t.Fatal(err)
		}
	}
	a, err := LeseAngaben(bytes.NewReader(AngabenSVG(d)))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Objekte) != 2 || a.Objekte[0].Ebenenwerte().Gruppe != 7 || a.Objekte[1].Ebenenwerte().Gruppe != 7 {
		t.Errorf("Gruppe nach dem Rundlauf: %+v", a.Objekte)
	}
}

// F-969: Über den echten Schreibweg — ein einzelnes Objekt mit Kennung schreibt kein s:gruppe und bleibt Version 1.
func TestGruppeEinzelneKennungSchreibweg(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 40)))
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Ebene: document.Ebene{Gruppe: 3}, Bereich: document.R(document.Pt(2, 2), document.Pt(10, 10)),
		Strich: document.Strich{Breite: 1}}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}
	svg := string(AngabenSVG(d))
	if strings.Contains(svg, "s:gruppe") || !strings.Contains(svg, `s:format="1"`) {
		t.Errorf("einzelne Kennung im Schreibweg: %s", svg)
	}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}
	svg = string(AngabenSVG(d))
	if strings.Count(svg, `s:gruppe="3"`) != 2 || !strings.Contains(svg, `s:format="2"`) {
		t.Errorf("wirksame Gruppe im Schreibweg: %s", svg)
	}
}
