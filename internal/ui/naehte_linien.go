// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"slices"

	"github.com/Andiministrator/schnapp/internal/canvas"
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
//
// Mit saumImBild liegen Fasersaum und Abdunklung schon im Bild der Leinwand (render.KantenOhneSchatten, F-407, auch die
// zweite Leerung bei berührenden Nähten): Der Editor legt dann nur das Schachbrett des Spalts und darüber den Schatten.
func fuehreNahtAblaufAus(naehte []document.Naht, faktor int, saumImBild bool, spalte func(), durchgang func(auswahl func(document.Schicht) bool)) {
	spalte()
	if saumImBild {
		durchgang(document.Schicht.ImSpalt)
		return
	}
	if !document.NaehteBeruehren(naehte, faktor) {
		durchgang(func(document.Schicht) bool { return true })
		return
	}
	durchgang(func(s document.Schicht) bool { return !s.ImSpalt() })
	spalte()
	durchgang(document.Schicht.ImSpalt)
}

// kantenBild ist der Stand, mit dem das Bild der Leinwand gebaut wurde: ob Fasersaum und Abdunklung der Nähte darin
// liegen (im) und wovon sie abhängen. Ändert sich einer der Werte (Kantenform per Menü, Stauchen, Rückgängig), baut
// leinwand das Bild neu (F-407).
type kantenBild struct {
	im      bool
	groesse image.Point
	form    document.Kante
	faktor  int
	naehte  []document.Naht
}

// kantenBildStand ist der Stand, den das Bild der Leinwand jetzt haben muss: Fasersaum und Abdunklung liegen im Bild,
// wenn es Nähte gibt und das Werkzeug nicht Zuschneiden ist (dort zeigt die Leinwand die Vollansicht, und die Kanten
// bleiben, wie sie waren, in Gio gezeichnet).
func kantenBildStand(s *canvas.Sitzung) kantenBild {
	d := s.D
	k := kantenBild{groesse: d.Groesse(), form: d.Kante(), faktor: d.KantenFaktor(), naehte: slices.Clone(d.Naehte())}
	k.im = len(k.naehte) > 0 && s.Werkzeug != canvas.WerkzeugZuschneiden
	return k
}

func (k kantenBild) gleich(a kantenBild) bool {
	return k.im == a.im && k.groesse == a.groesse && k.form == a.form && k.faktor == a.faktor && slices.Equal(k.naehte, a.naehte)
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
