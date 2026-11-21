// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-172: Ein Klick rechts außen auf die Schrift trifft. Zehn „W“ im Grad 40 sind rund 390 Pixel breit, die alte
// Schätzung (0,6 em je Zeichen) endete bei 240.
func TestTrefferTextRechtsAussen(t *testing.T) {
	tx := document.Text{ID: 1, Ort: document.Pt(10, 10), Inhalt: "WWWWWWWWWW", Schriftgrad: 40}
	ink := document.Reichweite(tx)
	p := document.Pt(ink.Max.X-3, 10+40*0.6)
	if p.X < 10+300 {
		t.Fatalf("Voraussetzung: Tinte reicht nur bis %v", ink.Max.X)
	}
	if id, ok := Treffer([]document.Objekt{tx}, p, 0); !ok || id != 1 {
		t.Errorf("Klick auf die letzte Glyphe bei %v trifft nicht", p)
	}
	if _, ok := Treffer([]document.Objekt{tx}, document.Pt(ink.Max.X+60, 30), 0); ok {
		t.Error("Klick weit rechts neben dem Text trifft")
	}
}

// F-172: Die Ziffern einer fünfstelligen Nummer ragen über den Kreis hinaus und zählen als Treffer.
func TestTrefferNummerZiffernAusserhalbKreis(t *testing.T) {
	n := document.Nummer{ID: 1, Mitte: document.Pt(100, 100), Wert: 88888, Radius: 6}
	ink, ok := document.NummerTinte(n)
	if !ok || ink.Max.X <= 106+1 {
		t.Fatalf("Voraussetzung: Ziffern %v ragen nicht über den Kreis", ink)
	}
	p := document.Pt(ink.Max.X-1.5, 100)
	if _, ok := Treffer([]document.Objekt{n}, p, 0); !ok {
		t.Errorf("Klick auf die letzte Ziffer bei %v trifft nicht", p)
	}
}
