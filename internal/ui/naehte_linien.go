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

// fuehreNahtAblaufAus ruft spalte (alle Spalte leeren) und durchgang (die Schichten aller Nähte, die auswahl bejaht) in
// der Reihenfolge von Renderer und SVG-Export auf (ADR-0023 Nachtrag 9, F-438): zuerst die Spalte, dann alle Schichten;
// berühren sich Nähte (document.NaehteBeruehren), erst Fasersaum und Abdunklung, dann die Spalte noch einmal, dann der
// Schatten im Spalt. Aus zeichneNaehte herausgelöst, damit ein Test ohne Fenster die zweite Leerung prüfen kann (F-499).
func fuehreNahtAblaufAus(naehte []document.Naht, faktor int, spalte func(), durchgang func(auswahl func(document.Schicht) bool)) {
	spalte()
	if !document.NaehteBeruehren(naehte, faktor) {
		durchgang(func(document.Schicht) bool { return true })
		return
	}
	durchgang(func(s document.Schicht) bool { return !s.ImSpalt() })
	spalte()
	durchgang(document.Schicht.ImSpalt)
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
