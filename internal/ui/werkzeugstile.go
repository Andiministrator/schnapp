// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Stil je Werkzeug über Sitzungen (ADR-0041 Punkt 4): tools.ini wird beim Öffnen des Editors gelesen, beim Wechsel der
// Aufnahme in die neue Sitzung mitgenommen und beim Schließen des Fensters geschrieben, wenn sich etwas geändert hat.

import (
	"fmt"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Lesen und Schreiben der Datei; Tests tauschen sie nicht, sie setzen XDG_CONFIG_HOME (TestMain).
var (
	ladeWerkzeugStile     = settings.LadeWerkzeugStile
	schreibeWerkzeugStile = settings.SchreibeWerkzeugStile
)

// werkzeugStand ist der Inhalt, den tools.ini nach dem Lesen hätte (Vergleich beim Schließen: nur Änderungen
// schreiben).
func werkzeugStand(s *canvas.Sitzung) string {
	inhalt, err := settings.FormatWerkzeugStile(s.WerkzeugStile())
	if err != nil {
		return ""
	}
	return inhalt
}

// werkzeugStileLaden liest tools.ini beim Öffnen; Warnungen gehen auf die Fehlerausgabe wie bei styles.ini.
func (e *editor) werkzeugStileLaden() {
	roh, warn := ladeWerkzeugStile()
	for _, w := range warn {
		fmt.Fprintln(fehlerAusgabe, w)
	}
	e.s.SetzeWerkzeugStile(roh)
	e.werkzeugStand = werkzeugStand(e.s)
}

// werkzeugStileMitnehmen gibt der neuen Sitzung beim Wechsel der Aufnahme die Stile der alten (auf Faktor 1
// umgerechnet, dann mal dem Faktor der neuen).
func (e *editor) werkzeugStileMitnehmen(alt *canvas.Sitzung) {
	e.s.SetzeWerkzeugStile(alt.WerkzeugStile())
}

// werkzeugStileSchreiben schreibt tools.ini beim Schließen, wenn sich seit dem Lesen etwas geändert hat. Ein Fehler
// geht auf die Fehlerausgabe; das Fenster schließt trotzdem (ADR-0041 Punkt 4).
func (e *editor) werkzeugStileSchreiben() {
	if werkzeugStand(e.s) == e.werkzeugStand {
		return
	}
	if err := schreibeWerkzeugStile(e.s.WerkzeugStile()); err != nil {
		fmt.Fprintln(fehlerAusgabe, err)
		return
	}
	e.werkzeugStand = werkzeugStand(e.s)
}
