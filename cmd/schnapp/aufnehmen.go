// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"

	"github.com/Andiministrator/schnapp/aufnahme"
	"github.com/Andiministrator/schnapp/auswahl"
	"github.com/Andiministrator/schnapp/bibliothek"
	"github.com/Andiministrator/schnapp/datei"
	"github.com/Andiministrator/schnapp/dokument"
	"github.com/Andiministrator/schnapp/einstellungen"
	"github.com/Andiministrator/schnapp/oberflaeche"
	"github.com/Andiministrator/schnapp/sprache"
)

// exitAbgebrochen: Der Nutzer hat die Auswahl abgebrochen (Esc im Overlay, Abbruch der Fensterwahl).
const exitAbgebrochen = 3

// Ausfuehrung führt einen Aufnahme-Auftrag aus und liefert den Exit-Code; Tests ersetzen sie.
type Ausfuehrung func(t Auftrag, aus, fehler io.Writer) int

// anschluesse sind die Außenanschlüsse eines Aufnahme-Laufs. Das Programm nimmt die echten (echteAnschluesse),
// Tests ihre Attrappen: keine echte Aufnahme, kein Fenster, keine Datei außerhalb von t.TempDir().
type anschluesse struct {
	quelle      func() (aufnahme.Quelle, error)
	monitore    func() ([]oberflaeche.Monitor, error)
	overlay     func([]oberflaeche.MonitorBild, auswahl.Modus) (oberflaeche.OverlayErgebnis, error)
	ablegen     func(d *dokument.Dokument) (string, error) // liefert den absoluten Pfad der Datei
	letzterPfad string
}

func echteAnschluesse() anschluesse {
	return anschluesse{
		quelle:   func() (aufnahme.Quelle, error) { return aufnahme.NeuKWin() },
		monitore: oberflaeche.Monitore,
		overlay:  oberflaeche.Overlay,
		ablegen: func(d *dokument.Dokument) (string, error) {
			orte, err := bibliothek.OrteAus(os.Getenv)
			if err != nil {
				return "", err
			}
			e, err := bibliothek.Neu(orte).Ablegen(d, d.Ansichtsbild(), datei.Metadaten{})
			if e.Pfad == "" {
				return "", err
			}
			// Index oder Vorschau gescheitert: Die Datei liegt trotzdem in der Ablage, das ist der Erfolg.
			return filepath.Join(orte.Ablage, filepath.FromSlash(e.Pfad)), nil
		},
		letzterPfad: auswahl.LetzterPfad(os.Getenv),
	}
}

// Ausfuehren ist die echte Ausführung.
func (an anschluesse) Ausfuehren(t Auftrag, aus, fehler io.Writer) int {
	q, err := an.quelle()
	if err != nil {
		return meldeFehler(fehler, err)
	}
	defer q.Close()
	ctx := context.Background()

	switch {
	case t.Modus == ModusFenster:
		b, err := q.Fenster(ctx, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), aus, fehler)

	case t.Modus == ModusWiederholen:
		l, err := auswahl.LadeLetzten(an.letzterPfad)
		if err != nil {
			if errors.Is(err, auswahl.ErrKeinLetzter) {
				fmt.Fprintln(fehler, sprache.T("No area to repeat yet. Select one with --region first."))
				return exitFehler
			}
			return meldeFehler(fehler, err)
		}
		b, err := q.Bildschirm(ctx, l.Monitor, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		if b.Pixel.Bounds().Size() != l.Groesse {
			fmt.Fprintln(fehler, sprache.T("The monitor of the last area has changed its size. Select a new area with --region."))
			return exitFehler
		}
		return an.speichere(b.Pixel, l.Bereich, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Name != "":
		b, err := q.Bildschirm(ctx, t.Ziel.Name, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		an.merke(t.Ziel.Name, b.Pixel, b.Pixel.Bounds(), fehler)
		return an.speichere(b.Pixel, b.Pixel.Bounds(), aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Wahl == einstellungen.WahlZeiger:
		b, err := q.AktiverBildschirm(ctx, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Wahl == einstellungen.WahlAlle:
		fmt.Fprintln(fehler, sprache.T("Capturing all monitors at once is not implemented yet."))
		return exitFehler
	}

	// Overlay: --region (Bereich) oder --screen mit der Einstellung pick (Monitor per Klick).
	modus := auswahl.ModusBereich
	if t.Modus == ModusBildschirm {
		modus = auswahl.ModusMonitor
	}
	monitore, err := an.monitore()
	if err != nil {
		fmt.Fprintf(fehler, sprache.T("Could not query the monitors: %v")+"\n", err)
		return exitFehler
	}
	if len(monitore) == 0 {
		fmt.Fprintln(fehler, sprache.T("No monitors found."))
		return exitFehler
	}
	// Erst alle Monitore aufnehmen, dann im eingefrorenen Bild wählen (knowledge/topics/kwin-screenshot.md).
	bilder := make([]oberflaeche.MonitorBild, 0, len(monitore))
	pixel := make([]*image.NRGBA, 0, len(monitore))
	for _, m := range monitore {
		b, err := q.Bildschirm(ctx, m.Name, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		bilder = append(bilder, oberflaeche.MonitorBild{Name: m.Name, Bild: b.Pixel})
		pixel = append(pixel, b.Pixel)
	}
	e, err := an.overlay(bilder, modus)
	if errors.Is(err, oberflaeche.ErrAbgebrochen) {
		return exitAbgebrochen
	}
	if err != nil {
		return meldeFehler(fehler, err)
	}
	if e.Monitor < 0 || e.Monitor >= len(pixel) || !e.Bereich.In(pixel[e.Monitor].Bounds()) || e.Bereich.Empty() {
		fmt.Fprintln(fehler, sprache.T("The selection is outside the captured image."))
		return exitFehler
	}
	an.merke(monitore[e.Monitor].Name, pixel[e.Monitor], e.Bereich, fehler)
	return an.speichere(pixel[e.Monitor], e.Bereich, aus, fehler)
}

// speichere legt den Bereich als neue Aufnahme in der Bibliothek ab und schreibt den Pfad nach aus. Das Original
// der Arbeitsdatei ist nur der gewählte Bereich, nicht der ganze Monitor: Was der Nutzer nicht gewählt hat, gehört
// nicht in die Datei (Kernprinzip 3).
func (an anschluesse) speichere(b *image.NRGBA, bereich image.Rectangle, aus, fehler io.Writer) int {
	d, err := dokument.Neu(b.SubImage(bereich))
	if err != nil {
		return meldeFehler(fehler, err)
	}
	pfad, err := an.ablegen(d)
	if err != nil {
		fmt.Fprintf(fehler, sprache.T("Could not save the screenshot: %v")+"\n", err)
		return exitFehler
	}
	fmt.Fprintln(aus, pfad)
	return exitOK
}

// merke speichert den Bereich für --repeat. Scheitert das, ist die Aufnahme trotzdem gelungen: nur eine Warnung.
func (an anschluesse) merke(monitor string, b *image.NRGBA, bereich image.Rectangle, fehler io.Writer) {
	l := auswahl.Letzter{Monitor: monitor, Groesse: b.Bounds().Size(), Bereich: bereich}
	if err := auswahl.SpeichereLetzten(an.letzterPfad, l); err != nil {
		fmt.Fprintf(fehler, sprache.T("Could not remember the area for --repeat: %v")+"\n", err)
	}
}

// meldeFehler schreibt einen Aufnahmefehler und liefert den Exit-Code. Ein Abbruch durch den Nutzer ist kein Fehler.
func meldeFehler(fehler io.Writer, err error) int {
	if errors.Is(err, aufnahme.ErrAbgebrochen) {
		return exitAbgebrochen
	}
	fmt.Fprintln(fehler, err)
	return exitFehler
}
