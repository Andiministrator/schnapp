// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"slices"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0039 Nachtrag 4 Punkt 5: Umschalt+Klick in der Ebenenliste wählt den Bereich, Strg+Klick schaltet; jede gewählte
// Zeile ist hervorgehoben, der Kopf der Leiste nennt die Zahl (Punkt 4).
func TestEbenenlisteMehrfachUndKopf(t *testing.T) {
	l, k := ebenenLauf(t)
	k2, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(30, 2), document.Pt(50, 20)),
		Strich: document.Strich{Farbe: canvas.Schwaerzfarbe, Breite: 9}})
	if err != nil {
		t.Fatal(err)
	}
	l.setle()
	eb := &l.e.ebenen
	if len(eb.zeilen) < 3 {
		t.Fatalf("%d Zeilen", len(eb.zeilen))
	}
	// Zeilen von oben: k2, k, Text, Bild.
	l.klick(mitte(eb.zeilen[0].rahmen))
	l.klickMit(mitte(eb.zeilen[2].rahmen), key.ModShift)
	if got := l.e.s.Gewaehlt(); !slices.Equal(got, []int{l.id, k, k2}) || l.e.s.Auswahl != l.id {
		t.Fatalf("Umschalt: gewählt %v, Haupt %d", got, l.e.s.Auswahl)
	}
	if l.e.seite.kopf != "SELECTION: 3 OBJECTS" {
		t.Errorf("Kopf %q", l.e.seite.kopf)
	}
	if l.e.seite.strichText != gemischtText() {
		t.Errorf("Strichstärke %q, Soll gemischt", l.e.seite.strichText)
	}
	n := 0
	for _, z := range l.e.s.Ebenen() {
		if z.Gewaehlt {
			n++
		}
	}
	if n != 3 {
		t.Errorf("%d Zeilen hervorgehoben", n)
	}
	l.klickMit(mitte(eb.zeilen[0].rahmen), key.ModShortcut)
	if got := l.e.s.Gewaehlt(); !slices.Equal(got, []int{l.id, k}) {
		t.Fatalf("Strg heraus: %v", got)
	}
	if l.e.seite.strichText == gemischtText() {
		t.Error("ein Kasten übrig: Strichstärke noch gemischt")
	}
	// Strichstärke in der Leiste wirkt auf den Kasten, der Text bleibt; ein Schritt.
	if !l.e.s.SetzeBreite(14) {
		t.Fatal("SetzeBreite")
	}
	o, _ := l.e.s.D.Objekt(k)
	if o.(document.Kasten).Strich.Breite != 14 {
		t.Errorf("Kasten %v", o.(document.Kasten).Strich.Breite)
	}
}
