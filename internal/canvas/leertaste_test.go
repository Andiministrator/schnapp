// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
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

// F-906: Auch ein Doppelklick verschiebt bei gehaltener Leertaste nur die Ansicht: Er hebt keinen Zuschnitt auf und
// beginnt keine Texteingabe.
func TestLeertasteDoppelklickVerschiebt(t *testing.T) {
	s := dreiKaesten(t)
	if err := s.D.SetzeZuschnitt(image.Rect(0, 0, 150, 80)); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.SetzeLeertaste(true)
	vor, start := s.D.Stand(), s.Abb.ZuFenster(document.Pt(0, 0))
	s.Doppelklick(f(5, 5))
	s.Ziehen(f(25, 15))
	s.Loslassen(f(25, 15))
	if s.D.Stand() != vor {
		t.Error("Doppelklick mit Leertaste änderte das Dokument")
	}
	if d := s.Abb.ZuFenster(document.Pt(0, 0)).Minus(start); d.X != 20 || d.Y != 10 {
		t.Errorf("Ansicht um %v verschoben, Soll (20, 10)", d)
	}
}

// Zeiger mit gehaltener Leertaste (Kritik #334, F-911): Hand auch gedrückt unter der Ziehschwelle, keine Hand ohne
// Zeiger über der Leinwand, während der Ruhe (Speichern, Laden) und im Zug des Bereichsmodus.
func TestLeertasteZeigerGrenzfaelle(t *testing.T) {
	s := dreiKaesten(t)
	s.SetzeLeertaste(true)
	p := f(150, 80)
	s.Druecken(p, TastePrimaer)
	if z := s.Zeiger(&p); z != ZeigerGreifen {
		t.Errorf("gedrückt, unter der Ziehschwelle: %v, Soll greifen", z)
	}
	s.Loslassen(p)
	if z := s.Zeiger(nil); z == ZeigerGreifen {
		t.Error("Hand ohne Zeiger über der Leinwand")
	}
	s.SetzeRuhe(true)
	if z := s.Zeiger(&p); z == ZeigerGreifen {
		t.Error("Hand während der Ruhe")
	}
	s.SetzeRuhe(false)
	s.SetzeLeertaste(false)
	if !s.BeginneTextBereich() {
		t.Fatal("Bereichsmodus")
	}
	s.Druecken(p, TastePrimaer)
	s.SetzeLeertaste(true)
	if z := s.Zeiger(&p); z == ZeigerGreifen {
		t.Error("Hand im Zug des Bereichsmodus")
	}
}
