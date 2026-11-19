// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

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
// Fehler: Die Datei liegt dann in der Ablage. Bleibt dabei die Vorschau des vorigen Stands liegen, sagt es ein
// Hinweis (F-115). Weicht Speichern auf eine neue Aufnahme aus, nennt das Ergebnis die neue Datei; ist die alte
// stehen geblieben, weil das Zurückschreiben scheiterte, als Warnung (F-116). Fehler kommen übersetzt zurück
// (F-117).
func speichernIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen) ui.Speichern {
	rel, inBibliothek := b.Relativ(abs)
	inBibliothek = inBibliothek && !g.Fremd
	meta := g.Meta
	return func(d *document.Dokument) (ui.Gespeichert, error) {
		sichtbar, err := render.Bild(d)
		if err != nil {
			return ui.Gespeichert{}, speicherFehler(err)
		}
		var zurueck error // Fehler beim Zurückschreiben, falls auch das Ausweichen scheitert
		versucht := inBibliothek
		if inBibliothek {
			_, err := b.Speichern(rel, d, sichtbar)
			switch {
			case errors.Is(err, library.ErrVorschauAlt):
				return ui.Gespeichert{Hinweis: i18n.T("Saved, but the old preview of this capture could not be removed."),
					Warnung: true}, nil
			case err == nil || errors.Is(err, library.ErrIndex) || errors.Is(err, library.ErrVorschau):
				return ui.Gespeichert{}, nil
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
			return ui.Gespeichert{}, speicherFehler(errors.Join(zurueck, err))
		}
		rel, inBibliothek = e.Pfad, true
		name := filepath.Base(filepath.FromSlash(e.Pfad))
		erg := ui.Gespeichert{Datei: name, Hinweis: fmt.Sprintf(i18n.T("Saved as a new capture: %s"), name)}
		if versucht {
			// Die bisherige Datei wurde nicht aktualisiert; liegt sie noch, enthält sie den alten Stand.
			erg.Hinweis = fmt.Sprintf(i18n.T("The previous file could not be updated; saved as a new capture: %s"), name)
			erg.Warnung = true
		}
		if errors.Is(err, library.ErrVorschauAlt) {
			erg.Hinweis += " " + i18n.T("The old preview could not be removed.")
			erg.Warnung = true
		}
		return erg, nil
	}
}

// uebersetzt trägt einen fertig übersetzten Text für die Ausgabe und behält den ursprünglichen Fehler (errors.Is).
type uebersetzt struct {
	text string
	err  error
}

func (u uebersetzt) Error() string { return u.text }
func (u uebersetzt) Unwrap() error { return u.err }

// speicherFehler übersetzt einen Fehler beim Speichern (F-117): ohne Pfad und ohne Paketpräfix, Schreiben statt Lesen.
func speicherFehler(err error) error {
	var text string
	switch {
	case errors.Is(err, fs.ErrPermission):
		text = i18n.T("No permission to write the file.")
	case errors.Is(err, syscall.ENOSPC):
		text = i18n.T("Not enough disk space.")
	case errors.Is(err, library.ErrKeinHeim):
		text = i18n.T("No home directory (HOME is not set).")
	default:
		text = i18n.T("The file could not be written.")
	}
	return uebersetzt{text: text, err: err}
}

// nutzerFehler übersetzt bekannte Fehler beim Öffnen für die Ausgabe; Fehler des Speicherns sind schon übersetzt; Unbekanntes bleibt, wie es ist
// (Paketfehler sind nicht übersetzt, Kritik A-9).
func nutzerFehler(err error) string {
	var u uebersetzt
	if errors.As(err, &u) {
		return u.text // schon übersetzt (Speichern, F-117)
	}
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
