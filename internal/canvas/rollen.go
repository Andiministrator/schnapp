// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// RollGrenze ist die größte Strecke in Fensterpixeln, um die ein einzelnes Rad-Ereignis die Ansicht verschiebt; sie
// fängt Ausreißer ab (gebündelte Rastschritte nach einer Pause), ohne den üblichen Schritt von rund 150 px zu kürzen.
const RollGrenze = 2000.0

// Rollen verschiebt die Ansicht um das, was Mausrad oder Touchpad melden (ADR-0040 Punkt 1): dx, dy in Fensterpixeln,
// positiv heißt Rad nach unten bzw. rechts — der Inhalt wandert dann nach oben bzw. links. Während eines Zugs geschieht
// nichts, ebenso bei ungültigen Werten.
func (s *Sitzung) Rollen(dx, dy float64) {
	if s.Zieht() || math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) || (dx == 0 && dy == 0) {
		return
	}
	klemme := func(v float64) float64 { return math.Max(-RollGrenze, math.Min(RollGrenze, v)) }
	s.eingepasst = false
	s.Abb = s.Abb.Verschoben(document.Pt(-klemme(dx), -klemme(dy)))
}
