// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"testing"
)

// F-683: MonitoreMitSkala ordnet die Skala über den Namen zu; scheitert die Abfrage der Skalen, bleibt die Liste gültig
// ohne Skala; scheitert die Monitorliste, kommt ihr Fehler.
func TestMonitoreMitSkala(t *testing.T) {
	altM, altS := monitoreAbfrage, skalenAbfrage
	t.Cleanup(func() { monitoreAbfrage, skalenAbfrage = altM, altS })
	monitoreAbfrage = func() ([]Monitor, error) {
		return []Monitor{{Name: "A", Breite: 2560, Hoehe: 1440, Skala: 2}, {Name: "B", Breite: 1920, Hoehe: 1080, Skala: 1}}, nil
	}
	skalenAbfrage = func() (map[string]float64, error) { return map[string]float64{"A": 1.25, "C": 2}, nil }
	ms, err := MonitoreMitSkala()
	if err != nil || len(ms) != 2 || ms[0].Bruch != 1.25 || ms[1].Bruch != 0 {
		t.Errorf("mit Skalen: %+v, %v", ms, err)
	}
	skalenAbfrage = func() (map[string]float64, error) {
		return map[string]float64{"A": 1.25}, errors.New("kein xdg_output")
	}
	if ms, err := MonitoreMitSkala(); err != nil || len(ms) != 2 || ms[0].Bruch != 0 {
		t.Errorf("Skalen gescheitert: %+v, %v", ms, err)
	}
	fehler := errors.New("keine Verbindung")
	monitoreAbfrage = func() ([]Monitor, error) { return nil, fehler }
	if _, err := MonitoreMitSkala(); !errors.Is(err, fehler) {
		t.Errorf("Monitorliste gescheitert: %v", err)
	}
}
