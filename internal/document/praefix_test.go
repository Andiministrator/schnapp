// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"strings"
	"testing"
)

func praefixDok(t *testing.T) *Dokument {
	t.Helper()
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 64, 64)))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// F-362: Fehler aus pruefe() verlassen das Paket über die exportierten Funktionen Hinzufuegen, BeginneNeuenZug und
// ZugSetze mit dem Präfix „document: “ (ADR-0019 Nachtrag 2).
func TestPraefixPruefeFehler(t *testing.T) {
	faelle := map[string]Objekt{
		"Text zu lang":     Text{Ort: Pt(1, 1), Inhalt: strings.Repeat("a", MaxTextZeichen+1), Schriftgrad: 12},
		"Text Steuerzeich": Text{Ort: Pt(1, 1), Inhalt: "a\x01b", Schriftgrad: 12},
		"Text kein UTF-8":  Text{Ort: Pt(1, 1), Inhalt: "a\xffb", Schriftgrad: 12},
		"Text Grad 0":      Text{Ort: Pt(1, 1), Inhalt: "a", Schriftgrad: 0},
		"Kasten Strich":    Kasten{Bereich: Rechteck{Min: Pt(1, 1), Max: Pt(5, 5)}, Strich: Strich{Breite: -1}},
		"Nummer Radius":    Nummer{Mitte: Pt(5, 5), Radius: -1, Wert: 1},
	}
	einmal := func(name, funktion string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: %s nimmt das Objekt an", name, funktion)
			return
		}
		if s := err.Error(); !strings.HasPrefix(s, "document: ") || strings.Count(s, "document:") != 1 {
			t.Errorf("%s: %s liefert %q ohne genau ein Präfix „document: “", name, funktion, s)
		}
	}
	for name, o := range faelle {
		d := praefixDok(t)
		_, err := d.Hinzufuegen(o)
		einmal(name, "Hinzufuegen", err)
		_, err = d.BeginneNeuenZug(o)
		einmal(name, "BeginneNeuenZug", err)

		// ZugSetze: erst ein gültiger Zug, dann der ungültige Wert.
		d = praefixDok(t)
		gut := Kasten{Bereich: Rechteck{Min: Pt(1, 1), Max: Pt(5, 5)}, Strich: Strich{Breite: 2}}
		id, err := d.BeginneNeuenZug(gut)
		if err != nil {
			t.Fatal(err)
		}
		schlecht := o.mitKennung(id)
		einmal(name, "ZugSetze", d.ZugSetze(schlecht))
	}
	// Pfeil: ungültige Koordinaten (NaN) über Hinzufuegen.
	nan := Pt(nanWert(), 1)
	_, err := praefixDok(t).Hinzufuegen(Pfeil{A: nan, B: Pt(3, 3), K: Pt(2, 2), Strich: Strich{Breite: 2}})
	einmal2 := err
	if einmal2 == nil || !strings.HasPrefix(einmal2.Error(), "document: ") {
		t.Errorf("Pfeil mit NaN: %v ohne Präfix", einmal2)
	}
}

func nanWert() float64 {
	var null float64
	return null / null
}
