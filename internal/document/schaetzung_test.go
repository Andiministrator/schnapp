// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"testing"
)

// Rückfall ohne Schrift (schaetzung): Ist er ein Obermaß der echten Tinte? Erst die echte Reichweite messen, dann die
// Schrift „nicht ladbar“ stellen und vergleichen.
func TestSchaetzungIstObermass(t *testing.T) {
	var texte []Text
	// jedes Zeichen der Schrift einzeln (BMP), dazu ein paar Zeilen
	schriftMu.Lock()
	s := schriftFuer(false)
	var runen []rune
	for r := rune(0x20); r < 0xFFFE; r++ {
		if gi, err := s.f.GlyphIndex(&schriftPuffer, r); err == nil && gi != 0 {
			runen = append(runen, r)
		}
	}
	schriftMu.Unlock()
	for _, r := range runen {
		texte = append(texte, Text{Ort: Pt(100, 100), Inhalt: string(r), Schriftgrad: 20})
	}
	for _, in := range []string{"Hallo Welt", "jgyQ_Åß|", "W\nfj\n\tTab", "AVAVAV", "ÄÖÜ\n\n.", "MMMMMMMMMMMM", "\tx", "a\r\nb"} {
		texte = append(texte, Text{Ort: Pt(100, 100), Inhalt: in, Schriftgrad: 20})
	}
	echt := make([]Rechteck, len(texte))
	for i, tx := range texte {
		echt[i] = Reichweite(tx)
	}
	var nummern []Nummer
	for _, w := range []int{0, 1, 8, 10, 99, 100, 888, 12345, 888888, 1000000} {
		nummern = append(nummern, Nummer{Mitte: Pt(100, 100), Wert: w, Radius: 10})
	}
	echtN := make([]Rechteck, len(nummern))
	for i, n := range nummern {
		echtN[i] = Reichweite(n)
	}

	schriftMu.Lock()
	ohneSchrift = true
	schriftMu.Unlock()
	defer func() {
		schriftMu.Lock()
		ohneSchrift = false
		schriftMu.Unlock()
	}()

	drin := func(aussen, innen Rechteck) bool {
		return innen.Min.X >= aussen.Min.X && innen.Min.Y >= aussen.Min.Y && innen.Max.X <= aussen.Max.X && innen.Max.Y <= aussen.Max.Y
	}
	fehler := 0
	for i, tx := range texte {
		g := Reichweite(tx)
		if _, _, geschaetzt := textTinte(tx); !geschaetzt {
			t.Fatal("Schrift ist trotz Abschalten geladen")
		}
		if !drin(g, echt[i]) {
			fehler++
			if fehler <= 12 {
				t.Errorf("Text %q (U+%04X): Schätzung %v enthält die echte Tinte %v nicht", tx.Inhalt, []rune(tx.Inhalt)[0], g, echt[i])
			}
		}
	}
	for i, n := range nummern {
		if g := Reichweite(n); !drin(g, echtN[i]) {
			fehler++
			t.Errorf("Nummer %d: Schätzung %v enthält die echte Reichweite %v nicht", n.Wert, g, echtN[i])
		}
	}
	t.Logf("%d Texte (%d Einzelzeichen), %d Nummern geprüft, %d Verstöße", len(texte), len(runen), len(nummern), fehler)
}
