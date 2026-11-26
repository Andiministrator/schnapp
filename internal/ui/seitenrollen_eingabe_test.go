// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
)

// Eine schwebende Eingabe wird beim Klick auf ein Farbfeld im gerollten Bild verworfen, nicht übernommen (Verify V-1):
// inFarbwahl rechnet die Lage des Klicks in Inhaltskoordinaten um. Übernommen rückte sie in die zuletzt benutzten Farben.
func TestEingabeVerworfenBeiFarbfeldImGerolltenBild(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 330)
	l.setle()
	se, roll := &l.e.seite, &l.e.seite.roll
	l.rolle(1 << 20)
	l.klick(mitte(se.fw.hex.rahmen).Sub(image.Pt(0, roll.rollen)))
	l.tippe("00ff00ff")
	feld := se.ff.rahmen[1]
	if !l.imSichtfeld(feld) {
		t.Fatalf("Kontrolle: Farbfeld %v nicht im Bild %v (Versatz %d)", feld, roll.sicht, roll.rollen)
	}
	l.klick(mitte(feld).Sub(image.Pt(0, roll.rollen)))
	for _, c := range l.e.s.Zuletzt() {
		if c == qaGruen {
			t.Errorf("die schwebende Eingabe wurde übernommen und steht in den zuletzt benutzten Farben: %v", l.e.s.Zuletzt())
		}
	}
}
