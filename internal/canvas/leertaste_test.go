// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0039 Nachtrag 1: Solange die Leertaste gehalten wird, verschiebt Ziehen mit der linken Taste die Ansicht, in jedem
// Werkzeug; es entsteht kein Objekt, keine Auswahl ändert sich, und der Zeiger zeigt die Hand.
func TestLeertasteZiehenVerschiebtAnsicht(t *testing.T) {
	for _, w := range []Werkzeug{WerkzeugAuswahl, WerkzeugPfeil, WerkzeugKasten, WerkzeugSchwaerzen, WerkzeugZuschneiden} {
		s := dreiKaesten(t)
		s.SetzeWerkzeug(w)
		if w == WerkzeugAuswahl {
			klick(s, f(15, 15), false) // Kasten 1 gewählt; Ziehen an ihm verschiebt sonst den Kasten
		}
		vorAuswahl, vorZahl, vorX := s.Auswahl, len(s.D.Objekte()), minX(t, s, 1)
		s.SetzeLeertaste(true)
		p := f(15, 15)
		if z := s.Zeiger(&p); z != ZeigerGreifen {
			t.Errorf("%v: Zeiger mit Leertaste %v, Soll greifen", w, z)
		}
		start := s.Abb.ZuFenster(document.Pt(0, 0))
		s.Druecken(p, TastePrimaer)
		s.Ziehen(p.Plus(document.Pt(30, 20)))
		if z := s.Zeiger(&p); z != ZeigerGreifend {
			t.Errorf("%v: Zeiger beim Ziehen %v, Soll greifend", w, z)
		}
		s.Loslassen(p.Plus(document.Pt(30, 20)))
		nach := s.Abb.ZuFenster(document.Pt(0, 0))
		if d := nach.Minus(start); d.X != 30 || d.Y != 20 {
			t.Errorf("%v: Ansicht um %v verschoben, Soll (30, 20)", w, d)
		}
		if len(s.D.Objekte()) != vorZahl || s.Auswahl != vorAuswahl || minX(t, s, 1) != vorX || s.Zieht() {
			t.Errorf("%v: Objekte %d (vorher %d), Auswahl %d (vorher %d), Kasten bei %v (vorher %v), zieht %v",
				w, len(s.D.Objekte()), vorZahl, s.Auswahl, vorAuswahl, minX(t, s, 1), vorX, s.Zieht())
		}
		s.SetzeLeertaste(false)
		if z := s.Zeiger(&p); z == ZeigerGreifen {
			t.Errorf("%v: Zeiger nach dem Loslassen der Leertaste noch greifen", w)
		}
	}
}

// Ohne Leertaste bleibt alles wie vorher: Ziehen im Bild mit Auswählen zieht einen Rahmen, nicht die Ansicht.
func TestOhneLeertasteKeinVerschieben(t *testing.T) {
	s := dreiKaesten(t)
	start := s.Abb.ZuFenster(document.Pt(0, 0))
	s.Druecken(f(5, 50), TastePrimaer)
	s.Ziehen(f(150, 90))
	if _, ok := s.AuswahlRahmen(); !ok {
		t.Error("kein Rahmen ohne Leertaste")
	}
	s.Loslassen(f(150, 90))
	if s.Abb.ZuFenster(document.Pt(0, 0)) != start {
		t.Error("Ansicht verschoben ohne Leertaste")
	}
}

// Während der Texteingabe ist die Leertaste Text: Ein Druck bestätigt die Eingabe nicht über den Umweg der Ansicht,
// sondern wirkt wie ohne Leertaste, und der Zeiger zeigt keine Hand.
func TestLeertasteNichtInTexteingabe(t *testing.T) {
	s := dreiKaesten(t)
	s.SetzeWerkzeug(WerkzeugText)
	klick(s, f(50, 60), false)
	if !s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	s.SetzeLeertaste(true)
	p := f(50, 60)
	if z := s.Zeiger(&p); z == ZeigerGreifen {
		t.Error("Hand während der Texteingabe")
	}
	start := s.Abb.ZuFenster(document.Pt(0, 0))
	s.Druecken(f(150, 80), TastePrimaer)
	s.Ziehen(f(170, 90))
	s.Loslassen(f(170, 90))
	if s.Abb.ZuFenster(document.Pt(0, 0)) != start {
		t.Error("Ansicht verschoben aus der Texteingabe heraus")
	}
}
