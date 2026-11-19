// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/render"
	"github.com/Andiministrator/schnapp/ui"
)

// oeffneImEditor öffnet eine Datei im Editor (ADR-0020). Speichern schreibt eine Aufnahme der Bibliothek an ihren
// Platz zurück; alles andere (Datei außerhalb, fremdes oder beschädigtes Bild, neuere Fassung) kommt beim ersten
// Speichern als neue Aufnahme in die Bibliothek, danach wird diese fortgeschrieben. Die geöffnete Datei selbst bleibt
// dann unberührt.
func oeffneImEditor(pfad string) error {
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return err
	}
	g, err := fileformat.Oeffne(abs)
	if err != nil {
		return err
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		return err
	}
	return ui.Editor(filepath.Base(abs), g.Dokument, speichernIn(library.Neu(orte), abs, g))
}

// speichernIn liefert die Speicherfunktion für den Editor. Gescheiterte Nebenschritte (Index, Vorschau) sind kein
// Fehler: Die Datei liegt dann in der Ablage.
func speichernIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen) ui.Speichern {
	rel, inBibliothek := b.Relativ(abs)
	inBibliothek = inBibliothek && !g.Fremd
	meta := g.Meta
	return func(d *document.Dokument) error {
		sichtbar, err := render.Bild(d)
		if err != nil {
			return err
		}
		var zurueck error // Fehler beim Zurückschreiben, falls auch das Ausweichen scheitert
		if inBibliothek {
			_, err := b.Speichern(rel, d, sichtbar)
			switch {
			case err == nil || errors.Is(err, library.ErrIndex) || errors.Is(err, library.ErrVorschau):
				return nil
			case errors.Is(err, fs.ErrNotExist) || errors.Is(err, library.ErrFremd) ||
				errors.Is(err, library.ErrPfad) || errors.Is(err, fileformat.ErrFormatNeuer):
				// Die Datei ist weg, ersetzt oder nicht mehr unsere (gelöscht, aufgeräumt, neuere Fassung): nicht
				// verlieren, sondern als neue Aufnahme ablegen (Kritik A-1).
			default:
				// Anderer Fehler (z. B. keine Schreibberechtigung): trotzdem eine neue Aufnahme versuchen, statt die
				// Bearbeitung zu verlieren (Verify V-1).
				zurueck = err
			}
		}
		m := meta
		m.Bearbeitet = true
		m.Erstellt = b.Jetzt() // neue Aufnahme der Bibliothek, Zeitpunkt des Speicherns
		e, err := b.Ablegen(d, sichtbar, m)
		if e.Pfad == "" {
			return errors.Join(zurueck, err)
		}
		rel, inBibliothek = e.Pfad, true
		return nil
	}
}

// nutzerFehler übersetzt bekannte Fehler beim Öffnen und im Editor für die Ausgabe; Unbekanntes bleibt, wie es ist
// (Paketfehler sind nicht übersetzt, Kritik A-9).
func nutzerFehler(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return i18n.T("The file does not exist.")
	case errors.Is(err, fs.ErrPermission):
		return i18n.T("No permission to read the file.")
	case errors.Is(err, fileformat.ErrKeineDatei):
		return i18n.T("This is not a regular file.")
	case errors.Is(err, fileformat.ErrKeinPNG):
		return i18n.T("This is neither a PNG nor a JPEG image.")
	case errors.Is(err, fileformat.ErrZuGross):
		return i18n.T("The image is too large.")
	case errors.Is(err, fileformat.ErrBeschaedigt):
		return i18n.T("The image is damaged.")
	case errors.Is(err, ui.ErrOverlayOffen):
		return i18n.T("The selection overlay did not close in time.")
	case errors.Is(err, library.ErrKeinHeim):
		return i18n.T("No home directory (HOME is not set).")
	}
	return err.Error()
}
