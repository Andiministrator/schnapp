// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui ist die Schicht zu Gio (Fenster, Overlay, Editor; ADR-0003). Sie bindet das gepatchte Gio aus
// third_party/gio ein (ADR-0015); alles andere im Programm kennt Gio nicht.
package ui

import (
	"gioui.org/app"

	"github.com/Andiministrator/schnapp/internal/wloutputs"
)

// Monitor beschreibt einen Bildschirm (Wayland-Ausgabe).
type Monitor struct {
	Name          string
	Breite, Hoehe int // aktueller Modus in physischen Pixeln
	Skala         int // ganzzahlige Pufferskala
	// Bruch ist die wirksame Skala (auch gebrochen, etwa 1,25) aus physischer und logischer Größe; 0 = unbekannt.
	// Nur MonitoreMitSkala setzt sie (F-668).
	Bruch float64
}

// monitoreAbfrage und skalenAbfrage sind die Abfragen von MonitoreMitSkala; Tests setzen Attrappen ein (F-683).
var (
	monitoreAbfrage = Monitore
	skalenAbfrage   = wloutputs.Skalen
)

// MonitoreMitSkala ist Monitore mit der wirksamen Skala je Monitor (eigene kurze Wayland-Verbindung über
// xdg_output, ADR-0017 Nachtrag 9). Kennt der Compositor xdg_output nicht, bleibt Bruch 0; die Liste gilt trotzdem.
// Nur --list-screens und das MCP-Werkzeug list_screens fragen sie ab, die Aufnahme braucht sie nicht.
func MonitoreMitSkala() ([]Monitor, error) {
	liste, err := monitoreAbfrage()
	if err != nil {
		return nil, err
	}
	if skalen, err := skalenAbfrage(); err == nil {
		for i := range liste {
			liste[i].Bruch = skalen[liste[i].Name]
		}
	}
	return liste, nil
}

// Monitore fragt die Wayland-Ausgaben ab, nach Name sortiert. Ausgaben ohne Namen fehlen, weil sie nicht
// ansprechbar sind (Gio-Patch, ADR-0015).
func Monitore() ([]Monitor, error) {
	ausgaben, err := app.WaylandOutputs()
	if err != nil {
		return nil, err
	}
	res := make([]Monitor, 0, len(ausgaben))
	for _, o := range ausgaben {
		b, h := gedrehteGroesse(o.Width, o.Height, o.Transform)
		res = append(res, Monitor{Name: o.Name, Breite: b, Hoehe: h, Skala: o.Scale})
	}
	return res, nil
}

// gedrehteGroesse liefert die Größe des Modus, wie der Monitor sie zeigt (F-415): wl_output meldet den Modus ohne
// Drehung; bei einer Drehung um 90 oder 270 Grad (wl_output.transform 1, 3 und gespiegelt 5, 7) sind Breite und Höhe
// vertauscht. 0, 2, 4 und 6 (keine Drehung, 180 Grad, gespiegelt) lassen sie. Das Bild einer Aufnahme hat diese Größe.
func gedrehteGroesse(breite, hoehe, transform int) (int, int) {
	if transform&1 == 1 {
		return hoehe, breite
	}
	return breite, hoehe
}
