// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"testing"
)

// PruefeInhalt urteilt und misst wie das Laden über Neu, SetzeZuschnitt und SetzeStreifen (F-272).
func TestPruefeInhaltWieLaden(t *testing.T) {
	d := neu(t, 40, 30)
	z := image.Rect(2, 3, 38, 27)
	st := []Streifen{{Von: 10, Bis: 14}, {Zeilen: true, Von: 5, Bis: 8}}
	if err := d.SetzeZuschnitt(z); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	g, err := PruefeInhalt(image.Pt(40, 30), z, st, nil)
	if err != nil || g != d.Groesse() {
		t.Errorf("Größe %v (%v), Dokument %v", g, err, d.Groesse())
	}
	for name, f := range map[string]func() error{
		"Zuschnitt außerhalb": func() error { _, e := PruefeInhalt(image.Pt(40, 30), image.Rect(0, 0, 99, 99), nil, nil); return e },
		"Zuschnitt leer":      func() error { _, e := PruefeInhalt(image.Pt(40, 30), image.Rect(5, 5, 5, 5), nil, nil); return e },
		"Streifen ungültig": func() error {
			_, e := PruefeInhalt(image.Pt(40, 30), image.Rect(0, 0, 40, 30), []Streifen{{Von: -1, Bis: 3}}, nil)
			return e
		},
		"Bild zu groß": func() error {
			_, e := PruefeInhalt(image.Pt(MaxSeite+1, 3), image.Rect(0, 0, MaxSeite+1, 3), nil, nil)
			return e
		},
	} {
		if f() == nil {
			t.Errorf("%s: kein Fehler", name)
		}
	}
}
