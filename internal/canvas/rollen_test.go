// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0040 Punkt 1: Rad nach unten (dy > 0) schiebt den Inhalt nach oben, nach rechts (dx > 0) nach links; Zoom bleibt,
// Einpassen endet.
func TestRollenVerschiebtAnsicht(t *testing.T) {
	s := dreiKaesten(t)
	zoom, start := s.Abb.Zoom, s.Abb.ZuFenster(document.Pt(0, 0))
	s.Rollen(30, 150)
	if d := s.Abb.ZuFenster(document.Pt(0, 0)).Minus(start); d.X != -30 || d.Y != -150 {
		t.Errorf("Ansicht um %v verschoben, Soll (-30, -150)", d)
	}
	if s.Abb.Zoom != zoom || s.Eingepasst() {
		t.Errorf("Zoom %v (vorher %v), eingepasst %v", s.Abb.Zoom, zoom, s.Eingepasst())
	}
}

// Ungültige Werte und ein laufender Zug ändern nichts; Ausreißer werden auf RollGrenze begrenzt.
func TestRollenGrenzen(t *testing.T) {
	s := dreiKaesten(t)
	start := s.Abb
	for _, w := range [][2]float64{{math.NaN(), 0}, {0, math.Inf(1)}, {0, 0}} {
		s.Rollen(w[0], w[1])
	}
	if s.Abb != start || !s.Eingepasst() {
		t.Errorf("ungültige Werte änderten die Ansicht: %v", s.Abb)
	}
	s.Rollen(0, 1e9)
	if d := s.Abb.ZuFenster(document.Pt(0, 0)).Minus(start.ZuFenster(document.Pt(0, 0))); d.Y != -RollGrenze {
		t.Errorf("Ausreißer verschob um %v, Soll −%v", d.Y, RollGrenze)
	}
	s.Druecken(f(150, 80), TastePrimaer) // Rahmen beginnt
	s.Ziehen(f(170, 90))
	vor := s.Abb
	s.Rollen(0, 100)
	if s.Abb != vor {
		t.Error("Rollen während eines Zugs")
	}
	s.Loslassen(f(170, 90))
}

// Kritik 2 B-3 (F-922): Klemme in beide Richtungen und für dx, NaN auch in dy, Rollen während der Texteingabe (ADR-0040
// Punkt 1), nichts während die mittlere Taste die Ansicht verschiebt.
func TestRollenGrenzfaelle(t *testing.T) {
	s := dreiKaesten(t)
	null := s.Abb.ZuFenster(document.Pt(0, 0))
	weg := func() document.Punkt { return s.Abb.ZuFenster(document.Pt(0, 0)).Minus(null) }
	s.Rollen(-1e9, 0)
	if d := weg(); d.X != RollGrenze || d.Y != 0 {
		t.Errorf("negativer Ausreißer in dx: %v, Soll (%v, 0)", d, RollGrenze)
	}
	s.Rollen(1e9, -1e9)
	if d := weg(); d.X != 0 || d.Y != RollGrenze {
		t.Errorf("Ausreißer in dx und negativ in dy: %v, Soll (0, %v)", d, RollGrenze)
	}
	vor := s.Abb
	s.Rollen(5, math.NaN())
	if s.Abb != vor {
		t.Error("NaN in dy änderte die Ansicht")
	}
	s = dreiKaesten(t) // frisch: die Ansicht oben ist weggerollt
	s.SetzeWerkzeug(WerkzeugText)
	klick(s, f(50, 60), false)
	if !s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	vor = s.Abb
	s.Rollen(0, 40)
	if s.Abb == vor {
		t.Error("Rollen während der Texteingabe wirkt nicht")
	}
	s.TextBestaetigen()
	s.Druecken(f(5, 5), TasteMitte)
	s.Ziehen(f(15, 15))
	vor = s.Abb
	s.Rollen(0, 40)
	if s.Abb != vor {
		t.Error("Rollen während die mittlere Taste zieht")
	}
	s.Loslassen(f(15, 15))
}
