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
//
// ohneGrenze ist --ignore-limit (ADR-0017 Nachtrag 6): Die Datei wird auch über dem Rechenbudget geöffnet.
func oeffneImEditor(pfad string, ohneGrenze bool) error {
	abs, g, stand, err := ladeFuerEditor(pfad, ohneGrenze)
	if err != nil {
		return err
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		return err
	}
	return ui.Editor(filepath.Base(abs), g.Dokument, speichernIn(library.Neu(orte), abs, g, stand))
}

// ladeFuerEditor liest eine Datei für den Editor (von außen oder aus der Bibliothek) und lehnt ein Dokument ab, dessen
// Rastern das Rechenbudget überschritte (render.Pruefe, F-125, F-132): Gegen präparierte Dateien wird beim Öffnen
// geprüft, damit Speichern nie abbrechen muss (F-131, ADR-0020 Nachtrag 1). stand ist der Zustand der Datei, gemerkt
// vor dem Lesen (F-110): Ändert sie sich danach, merkt es das Speichern. Lässt er sich nicht lesen, bleibt der
// Nullwert, und das Zurückschreiben weicht auf eine neue Aufnahme aus.
//
// ohneGrenze (--ignore-limit, F-146) überspringt nur die Ablehnung durch render.Pruefe, etwa für eigene Arbeit, die
// über das Budget gewachsen ist (F-138). Die Grenzen beim Lesen (Objekte, Streifen, Kacheln, Bildgröße) und die
// Formatprüfung gelten weiter.
func ladeFuerEditor(pfad string, ohneGrenze bool) (abs string, g *fileformat.Gelesen, stand library.Stand, err error) {
	abs, err = filepath.Abs(pfad)
	if err != nil {
		return "", nil, library.Stand{}, err
	}
	stand, _ = library.StandVon(abs)
	g, err = fileformat.Oeffne(abs)
	if err != nil {
		return "", nil, library.Stand{}, err
	}
	if !ohneGrenze {
		if err := render.Pruefe(g.Dokument); err != nil {
			return "", nil, library.Stand{}, err
		}
	}
	return abs, g, stand, nil
}

// speichernIn liefert die Speicherfunktion für den Editor. Gescheiterte Nebenschritte (Index, Vorschau) sind kein
// Fehler: Die Datei liegt dann in der Ablage. Bleibt dabei die Vorschau des vorigen Stands liegen, sagt es ein
// Hinweis (F-115). Weicht Speichern auf eine neue Aufnahme aus, nennt das Ergebnis die neue Datei; ist die alte
// stehen geblieben, weil das Zurückschreiben scheiterte, als Warnung (F-116). Wurde die Datei seit dem Öffnen von
// anderer Seite geändert (stand, etwa ein zweiter Editor), wird sie nicht überschrieben, sondern ebenfalls eine neue
// Aufnahme angelegt und als Warnung gemeldet (F-110). Fehler kommen übersetzt zurück (F-117).
//
// Überschreitet das gespeicherte Dokument das Rechenbudget (render.Pruefe), warnt das Ergebnis zusätzlich (F-138):
// Gespeichert ist es trotzdem (F-131), aber ladeFuerEditor lehnte es beim nächsten Öffnen ab. Die Warnung hält das
// Fenster beim Schließen einmal offen (ADR-0020 Nachtrag 2), sodass sich Beschriftungen noch entfernen lassen.
func speichernIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand) ui.Speichern {
	speichern := ablegenIn(b, abs, g, stand)
	return func(d *document.Dokument) (ui.Gespeichert, error) {
		erg, err := speichern(d)
		if err != nil || render.Pruefe(d) == nil {
			return erg, err
		}
		text := i18n.T("Saved, but the file has too many or too large annotations to be opened again. Remove some before closing.")
		if erg.Hinweis != "" {
			text = erg.Hinweis + " " + text
		}
		erg.Hinweis, erg.Warnung = text, true
		return erg, nil
	}
}

// ablegenIn ist speichernIn ohne die Prüfung des Budgets.
func ablegenIn(b *library.Bibliothek, abs string, g *fileformat.Gelesen, stand library.Stand) ui.Speichern {
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
		konflikt := false
		if inBibliothek {
			_, neu, err := b.Speichern(rel, stand, d, sichtbar)
			if neu != (library.Stand{}) {
				stand = neu // geschrieben (auch wenn Index oder Vorschau scheiterten)
			}
			switch {
			case errors.Is(err, library.ErrVorschauAlt):
				return ui.Gespeichert{Hinweis: i18n.T("Saved, but the old preview of this capture could not be removed."),
					Warnung: true}, nil
			case err == nil || errors.Is(err, library.ErrIndex) || errors.Is(err, library.ErrVorschau):
				return ui.Gespeichert{}, nil
			case errors.Is(err, library.ErrGeaendert):
				// Ein anderer Editor oder ein anderes Programm hat die Datei inzwischen geschrieben: dessen Stand nicht
				// still überschreiben (F-110), sondern ausweichen.
				konflikt = true
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
		// Stand der neuen Aufnahme für das nächste Speichern; unlesbar bleibt der Nullwert (dann wiederum Ausweichen).
		stand, _ = library.StandVon(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
		name := filepath.Base(filepath.FromSlash(e.Pfad))
		erg := ui.Gespeichert{Datei: name, Hinweis: fmt.Sprintf(i18n.T("Saved as a new capture: %s"), name)}
		switch {
		case konflikt:
			erg.Hinweis = fmt.Sprintf(i18n.T("The file was changed elsewhere since it was opened; saved as a new capture: %s"), name)
			erg.Warnung = true
		case versucht:
			// Die bisherige Datei wurde nicht aktualisiert; liegt sie noch, enthält sie den alten Stand. Bewusst so
			// (ADR-0020 Nachtrag 1: Ausweichen statt Verlust). Mit dem Schwärzen darf dort kein ungeschwärzter Stand
			// stehen bleiben — offen als F-126 (Ausnahmepfad Sprint 4, Lösung vor oder mit der Schwärzen-Karte).
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
	case errors.Is(err, render.ErrZuAufwendig):
		return i18n.T("This file has too many or too large annotations to open. To open it anyway, use --ignore-limit.")
	}
	return err.Error()
}
