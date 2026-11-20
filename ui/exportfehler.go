// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Andiministrator/schnapp/export"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/portal"
)

// exportFehlerText übersetzt einen Fehler des Exports für die Statusleiste (A-6): Bekannte Fehler der Pakete portal und
// export kommen als übersetzter Text; bei Dateifehlern erscheint nur die Ursache des Betriebssystems, nie der Pfad
// (er enthielte Ordner- und Benutzernamen). dateiname ist der Name der Zieldatei ohne Ordner.
func exportFehlerText(err error, dateiname string) string {
	switch {
	case errors.Is(err, export.ErrZielVorhanden):
		return fmt.Sprintf(i18n.T("A file with this name already exists: %s"), dateiname)
	case errors.Is(err, portal.ErrVerbindung):
		return i18n.T("The connection to the desktop portal was lost.")
	case errors.Is(err, portal.ErrAntwort):
		return i18n.T("The file dialog ended without a result.")
	case errors.Is(err, portal.ErrOhneDatei):
		return i18n.T("The file dialog returned no file.")
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le.Err.Error()
	}
	return err.Error()
}

// Meldungen des Exports, die nur ins Debug-Log gehören (etwa nicht übernommene Rechte auf exFAT).
func init() { export.Diagnose = diagnoseEditor }
