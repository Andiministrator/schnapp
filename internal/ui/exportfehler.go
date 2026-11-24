// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// exportFehlerText übersetzt einen Fehler des Exports für die Fußzeile (A-6): Bekannte Fehler der Pakete portal und
// export kommen als übersetzter Text; bei Dateifehlern erscheint nur die Ursache des Betriebssystems, nie der Pfad
// (er enthielte Ordner- und Benutzernamen). dateiname ist der Name der Zieldatei ohne Ordner.
func exportFehlerText(err error, dateiname string) string {
	switch {
	case errors.Is(err, export.ErrZielVorhanden):
		return fmt.Sprintf(i18n.T("A file with this name already exists: %s"), dateiname)
	case errors.Is(err, export.ErrZielBeschaedigt):
		return fmt.Sprintf(i18n.T("The file %s is damaged: writing failed and the old content could not be restored."), dateiname)
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
	return ohnePaketpraefix(err.Error())
}

// paketPraefixe sind die Pakete, deren Fehlertexte mit „<paket>: “ beginnen (internal/errprefix, ADR-0019 Nachtrag 3,
// F-379, F-416). Ein Test vergleicht die Liste mit den Aufrufen von errprefix im Quelltext.
var paketPraefixe = []string{"export", "fileformat", "document", "render", "library", "rescue", "capture", "canvas", "portal", "settings", "mcp", "notify", "ocr"}

// ohnePaketpraefix nimmt die Präfixe „<paket>: “ der internen Pakete vom Anfang eines Fehlertexts (ADR-0019 Nachtrag 2):
// In der Fußzeile steht schon „Could not export:“, die Paketnamen sagen dem Nutzer nichts (Sitzungs-QA 22.11., Q-9).
func ohnePaketpraefix(text string) string {
	for {
		rest, ok := "", false
		for _, p := range paketPraefixe {
			if r, gefunden := strings.CutPrefix(text, p+": "); gefunden {
				rest, ok = r, true
				break
			}
		}
		if !ok {
			return text
		}
		text = rest
	}
}

// Meldungen des Exports, die nur ins Debug-Log gehören (etwa nicht übernommene Rechte auf exFAT).
func init() { export.Diagnose = diagnoseEditor }
