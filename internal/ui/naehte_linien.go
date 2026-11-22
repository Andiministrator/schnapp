// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"

	"github.com/Andiministrator/schnapp/internal/document"
)

// nahtLinien rechnet die Mittellinie der Kante je Achse einmal (F-418) und liefert sie mit der Länge der Naht: die
// Breite der Ansicht für eine Zeilen-Naht, die Höhe für eine Spalten-Naht. Aus zeichneNaehte herausgelöst, damit ein
// Test ohne Fenster die Wahl von Achse und Länge prüfen kann (F-433); der Renderer macht es ebenso.
type nahtLinien struct {
	g      image.Point
	form   document.Kante
	faktor int
	linien [2]*document.Kantenlinie
}

// fuer liefert die Linie der Achse von n und die Länge der Naht.
func (l *nahtLinien) fuer(n document.Naht) (*document.Kantenlinie, int) {
	achse, laenge := 0, l.g.X
	if !n.Zeilen {
		achse, laenge = 1, l.g.Y
	}
	if l.linien[achse] == nil {
		l.linien[achse] = document.NeueKantenlinie(l.form, laenge, l.faktor)
	}
	return l.linien[achse], laenge
}
