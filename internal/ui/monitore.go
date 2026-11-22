// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui ist die Schicht zu Gio (Fenster, Overlay, Editor; ADR-0003). Sie bindet das gepatchte Gio aus
// third_party/gio ein (ADR-0015); alles andere im Programm kennt Gio nicht.
package ui

import "gioui.org/app"

// Monitor beschreibt einen Bildschirm (Wayland-Ausgabe).
type Monitor struct {
	Name          string
	Breite, Hoehe int // aktueller Modus in physischen Pixeln
	Skala         int // ganzzahlige Pufferskala
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
