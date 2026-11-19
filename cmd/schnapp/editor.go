// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
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
		if inBibliothek {
			_, err := b.Speichern(rel, d, sichtbar)
			if err == nil || errors.Is(err, library.ErrIndex) || errors.Is(err, library.ErrVorschau) {
				return nil
			}
			return err
		}
		m := meta
		m.Bearbeitet = true
		m.Erstellt = b.Jetzt() // neue Aufnahme der Bibliothek, Zeitpunkt des Speicherns
		e, err := b.Ablegen(d, sichtbar, m)
		if e.Pfad == "" {
			return err
		}
		rel, inBibliothek = e.Pfad, true
		return nil
	}
}
