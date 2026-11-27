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
