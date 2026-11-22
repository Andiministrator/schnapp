// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Schnittstelle für Skripte und Agenten, Fassung 1 (ADR-0030): --json, --output=DATEI, --export=ZIEL DATEI.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"unicode/utf8"

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
// Scheitert das Schreiben, meldet es fehler (die Datei liegt trotzdem vor, F-162) und liefert false. Ein Pfad, der kein
// gültiges UTF-8 ist, lässt sich in JSON nicht treu wiedergeben (json.Marshal setzte still U+FFFD ein): Mit alsJSON
// steht dann keine Zeile da, sondern eine Meldung auf fehler (KA-8). Ziele von --output und --export lehnt Auswerten
// schon vorher ab; übrig bleiben ein Bibliotheksordner oder ein Arbeitsordner mit solchem Namen.
func gibPfadAus(aus, fehler io.Writer, alsJSON bool, pfad string, g image.Point) bool {
	var err error
	if alsJSON && !utf8.ValidString(pfad) {
		fmt.Fprintln(fehler, i18n.T("The path is not valid UTF-8 and cannot be written as JSON."))
		return false
	}
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
// (ADR-0025 Punkt 5). Liefert den absoluten Pfad. Das Bild wird zuerst vollständig im Speicher gerendert und kodiert;
// erst dann entsteht die temporäre Datei neben dem Ziel (KA-3): Ein Abbruch (etwa SIGTERM) während des langen
// Renderns hinterlässt so nichts, ein Signal während des kurzen Schreibens fängt mitSignalwache (F-396).
// privat (--output): eine neue Datei bekommt die Rechte 0600 (Bildschirminhalt, wie die Bibliothek; KA-4); sonst
// (--export) gilt die umask wie beim Export aus dem Editor (ADR-0025). Eine vorhandene Zieldatei behält ihre Rechte.
func schreibeBild(d *document.Dokument, pfad string, privat bool) (string, error) {
	f, ok := export.AusEndung(pfad)
	if !ok {
		// Auswerten prüft die Endung schon; hier nur gegen einen Aufruf von anderer Stelle.
		return "", fmt.Errorf("schnapp: unbekannte Endung %q", filepath.Ext(pfad))
	}
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return "", err
	}
	var inhalt bytes.Buffer
	if err := export.Schreibe(d, f, &inhalt); err != nil {
		return "", err
	}
	return abs, schreibeInhalt(abs, inhalt.Bytes(), privat, nil)
}

// schreibFehler übersetzt einen Fehler beim Schreiben der Ausgabe für die Meldung. Was keine bekannte Ursache hat,
// erscheint als allgemeine übersetzte Meldung; die Ursache selbst (deutsche Paketfehler, womöglich mit Pfaden) zeigt
// nur SCHNAPP_DEBUG, das schreiben die Aufrufer über diagnose (KA-8).
func schreibFehler(err error) string {
	switch {
	case errors.Is(err, export.ErrZielBeschaedigt):
		return i18n.T("Writing failed and the old content could not be restored; the target file is incomplete.")
	case errors.Is(err, fs.ErrPermission):
		return i18n.T("No permission to write there.")
	case errors.Is(err, fs.ErrNotExist):
		return i18n.T("The folder does not exist.")
	}
	return i18n.T("The file could not be written.")
}

// errZielIstQuelle: --export=ZIEL DATEI mit ZIEL = DATEI (auch über einen Link) überschriebe die Arbeitsdatei mit dem
// flachen Bild (KA-2).
var errZielIstQuelle = errors.New("schnapp: Exportziel ist die Datei selbst")

// pruefeZielNichtQuelle lehnt ein Exportziel ab, das dieselbe Datei ist wie die Quelle — gleich ob über denselben
// Namen, einen symbolischen oder einen harten Link (os.SameFile). Gibt es das Ziel noch nicht oder lässt es sich nicht
// prüfen, entscheidet das Schreiben selbst.
func pruefeZielNichtQuelle(quelle, ziel string) error {
	q, err := os.Stat(quelle)
	if err != nil {
		return nil // das Laden meldet den Fehler der Quelle
	}
	z, err := os.Stat(ziel)
	if err != nil {
		return nil
	}
	if os.SameFile(q, z) {
		return errZielIstQuelle
	}
	return nil
}

// exportiere öffnet die Datei ohne Fenster und schreibt ihr sichtbares Bild nach a.Ziel (--export, ADR-0030 Punkt 3).
// Das Rechenbudget gilt wie beim Öffnen im Editor, außer mit --ignore-limit. Die Datei selbst bleibt unverändert; ein
// Ziel, das die Datei selbst ist, wird abgelehnt (KA-2).
func exportiere(a Aufruf, aus, fehler io.Writer) int {
	if err := pruefeZielNichtQuelle(a.Datei, a.Ziel); err != nil {
		fmt.Fprintf(fehler, i18n.T("Cannot export %s onto itself; choose a different target.")+"\n", a.Datei)
		return exitFehler
	}
	_, g, _, err := ladeFuerEditor(a.Datei, a.OhneGrenze)
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Cannot open %s: %s")+"\n", a.Datei, nutzerFehler(err))
		return exitFehler
	}
	abs, err := schreibeBild(g.Dokument, a.Ziel, false)
	if err != nil {
		diagnose(fehler, "Ursache des Exports: %v", err)
		meldeSchreibFehler(fehler, a.Ziel, err)
		if errors.Is(err, errNachSchreiben) {
			gibPfadAus(aus, fehler, a.JSON, abs, g.Dokument.Groesse()) // die Datei liegt vor (F-441)
		}
		return exitFehler
	}
	// Wie bei der Aufnahme (F-180): Die Datei liegt vor; eine verlorene Ergebniszeile ist nur gemeldet.
	gibPfadAus(aus, fehler, a.JSON, abs, g.Dokument.Groesse())
	return exitOK
}
