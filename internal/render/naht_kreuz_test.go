// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-438 (ADR-0023 Nachtrag 8): Kreuzen sich eine Zeilen- und eine Spalten-Naht, liegt der weiße Fasersaum der einen
// nicht im durchsichtigen Spalt der anderen. Dort bleibt Alpha 0 oder höchstens der dunkle Schatten. Zackenform, Naht
// bei Zeile 50 und bei Spalte 50 auf einem deckend weißen Bild; im Spalt der Spaltennaht (x 48 bis 52) liegt der Saum
// der Zeilennaht (y 36 bis 46) und umgekehrt.
func TestKreuzendeNaehteSaumNichtImSpalt(t *testing.T) {
	d := opakDokument(t, 100, 100, document.KanteZacken, 0)
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 50, Bis: 51}, {Von: 50, Bis: 51}}); err != nil {
		t.Fatal(err)
	}
	if n := d.Naehte(); len(n) != 2 {
		t.Fatalf("Naehte %v", n)
	}
	b := bild(t, d)
	pruefe := func(name string, xs, ys [2]int) {
		weiss := 0
		for y := ys[0]; y <= ys[1]; y++ {
			for x := xs[0]; x <= xs[1]; x++ {
				p := b.NRGBAAt(x, y)
				if p.A != 0 && p.R > 0x40 { // weder durchsichtig noch Schatten: Saum oder Bild
					weiss++
				}
			}
		}
		if weiss != 0 {
			t.Errorf("%s: %d Pixel mit hellem Saum im Spalt der anderen Naht", name, weiss)
		}
	}
	pruefe("Saum der Zeilennaht im Spalt der Spaltennaht", [2]int{48, 52}, [2]int{30, 46})
	pruefe("Saum der Spaltennaht im Spalt der Zeilennaht", [2]int{30, 46}, [2]int{48, 52})
	// Der Schatten der Zeilennaht bleibt im eigenen Spalt (y 43 bis 49, weit weg von der Spaltennaht).
	schatten := 0
	for y := 40; y <= 49; y++ {
		if p := b.NRGBAAt(20, y); p.A != 0 && p.R <= 0x40 {
			schatten++
		}
	}
	if schatten == 0 {
		t.Error("Schatten der Zeilennaht fehlt im eigenen Spalt")
	}
	// Zur Kontrolle: Ohne die andere Naht steht der Saum dort sehr wohl.
	e := opakDokument(t, 100, 100, document.KanteZacken, 0)
	if err := e.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 50, Bis: 51}}); err != nil {
		t.Fatal(err)
	}
	c := bild(t, e)
	helle := 0
	for y := 36; y <= 46; y++ {
		if p := c.NRGBAAt(50, y); p.A != 0 && p.R > 0x40 {
			helle++
		}
	}
	if helle == 0 {
		t.Error("Kontrolle: Saum der einzelnen Zeilennaht nicht zu sehen")
	}
}

// F-430: Bild zählt die Kosten der Füllungen nicht mit; das Ergebnis ist trotzdem bitgleich zu dem Weg, der zählt
// (bildMitAufwand), mit Nähten in allen drei Formen, kreuzenden Nähten, Objekten und halbdurchsichtigem Grund.
func TestBildOhneZaehlenBitgleich(t *testing.T) {
	for _, form := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		d := opakDokument(t, 300, 200, form, 0)
		if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 60, Bis: 70}, {Zeilen: true, Von: 80, Bis: 90},
			{Von: 100, Bis: 120}}); err != nil {
			t.Fatal(err)
		}
		hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(5, 5), document.Pt(250, 150)), Fuellung: rot,
			Strich: document.Strich{Farbe: blau, Breite: 3}})
		a, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		b, auf, err := bildMitAufwand(d)
		if err != nil {
			t.Fatal(err)
		}
		if auf.summe() == 0 || string(a.Pix) != string(b.Pix) {
			t.Errorf("%v: Bild und bildMitAufwand unterscheiden sich (Aufwand %v)", form, auf.summe())
		}
	}
}

// F-438: Parallele Nähte rücken zusammen — der Saum der einen liegt im Spalt der nächsten, wenn sie näher als die
// ganze Bandhöhe (32) beieinanderliegen; die Entscheidung trifft document.NaehteBeruehren.
func TestNaehteBeruehren(t *testing.T) {
	z := func(l int) document.Naht { return document.Naht{Zeilen: true, Lage: l} }
	s := func(l int) document.Naht { return document.Naht{Lage: l} }
	faelle := []struct {
		name   string
		naehte []document.Naht
		faktor int
		soll   bool
	}{
		{"keine", nil, 1, false},
		{"eine", []document.Naht{z(50)}, 1, false},
		{"beide Achsen", []document.Naht{z(50), s(500)}, 1, true},
		{"parallel weit (32)", []document.Naht{z(50), z(82)}, 1, false},
		{"parallel eng (31)", []document.Naht{z(50), z(81)}, 1, true},
		{"parallel eng, ungeordnet", []document.Naht{z(81), z(10), z(50)}, 1, true},
		{"Faktor 2: 63 ist eng", []document.Naht{s(100), s(163)}, 2, true},
		{"Faktor 2: 64 ist weit", []document.Naht{s(100), s(164)}, 2, false},
	}
	for _, c := range faelle {
		if got := document.NaehteBeruehren(c.naehte, c.faktor); got != c.soll {
			t.Errorf("%s: %v, Soll %v", c.name, got, c.soll)
		}
	}
}
