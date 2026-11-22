// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Schnittstelle für Skripte und Agenten, Fassung 1 (ADR-0030): --json, --output=DATEI, --export=ZIEL DATEI.

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// ergebnis ist die JSON-Zeile einer Aufnahme oder eines Exports (ADR-0030 Punkt 1). Die Schlüssel sind Oberfläche.
type ergebnis struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// gibJSONAus schreibt v als eine Zeile JSON nach aus. Scheitert das Schreiben, ist das ein Fehler: Ein Skript mit
// --json wertet nur diese Zeile aus.
func gibJSONAus(aus, fehler io.Writer, v any) int {
	b, err := json.Marshal(v)
	if err == nil {
		_, err = fmt.Fprintf(aus, "%s\n", b)
	}
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Could not write to standard output: %v")+"\n", err)
		return exitFehler
	}
	return exitOK
}

// gibPfadAus schreibt das Ergebnis einer Aufnahme oder eines Exports: den Pfad als Zeile, mit alsJSON die JSON-Zeile.
// Scheitert das Schreiben, meldet es fehler (die Datei liegt trotzdem vor, F-162) und liefert false.
func gibPfadAus(aus, fehler io.Writer, alsJSON bool, pfad string, g image.Point) bool {
	var err error
	if alsJSON {
		var b []byte
		if b, err = json.Marshal(ergebnis{Path: pfad, Width: g.X, Height: g.Y}); err == nil {
			_, err = fmt.Fprintf(aus, "%s\n", b)
		}
	} else {
		_, err = fmt.Fprintln(aus, pfad)
	}
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Could not write the path to standard output: %v")+"\n", err)
		return false
	}
	return true
}

// schreibeBild schreibt das sichtbare Bild von d nach pfad, Format nach der Endung (ADR-0030 Punkte 2 und 3), atomar
// (ADR-0025 Punkt 5). Liefert den absoluten Pfad.
func schreibeBild(d *document.Dokument, pfad string) (string, error) {
	f, ok := export.AusEndung(pfad)
	if !ok {
		// Auswerten prüft die Endung schon; hier nur gegen einen Aufruf von anderer Stelle.
		return "", fmt.Errorf("schnapp: unbekannte Endung %q", filepath.Ext(pfad))
	}
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return "", err
	}
	return abs, export.SchreibeAtomar(abs, func(w io.Writer) error { return export.Schreibe(d, f, w) })
}

// schreibFehler übersetzt einen Fehler beim Schreiben der Ausgabe für die Meldung. Andere Fehler bleiben, wie sie sind:
// Sie nennen höchstens das Ziel, das der Aufrufer selbst angegeben hat.
func schreibFehler(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return i18n.T("No permission to write there.")
	case errors.Is(err, fs.ErrNotExist):
		return i18n.T("The folder does not exist.")
	}
	return err.Error()
}

// exportiere öffnet die Datei ohne Fenster und schreibt ihr sichtbares Bild nach a.Ziel (--export, ADR-0030 Punkt 3).
// Das Rechenbudget gilt wie beim Öffnen im Editor, außer mit --ignore-limit. Die Datei selbst bleibt unverändert.
func exportiere(a Aufruf, aus, fehler io.Writer) int {
	_, g, _, err := ladeFuerEditor(a.Datei, a.OhneGrenze)
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Cannot open %s: %s")+"\n", a.Datei, nutzerFehler(err))
		return exitFehler
	}
	abs, err := schreibeBild(g.Dokument, a.Ziel)
	if err != nil {
		diagnose(fehler, "Ursache des Exports: %v", err)
		fmt.Fprintf(fehler, i18n.T("Could not write %s: %s")+"\n", a.Ziel, schreibFehler(err))
		return exitFehler
	}
	// Wie bei der Aufnahme (F-180): Die Datei liegt vor; eine verlorene Ergebniszeile ist nur gemeldet.
	gibPfadAus(aus, fehler, a.JSON, abs, g.Dokument.Groesse())
	return exitOK
}
