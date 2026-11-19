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
		res = append(res, Monitor{Name: o.Name, Breite: o.Width, Hoehe: o.Height, Skala: o.Scale})
	}
	return res, nil
}
