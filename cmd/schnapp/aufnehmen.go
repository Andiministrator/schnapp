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
	"time"

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
	// aufraeumen räumt die Bibliothek auf; tagesmarke ist die Marke für „höchstens einmal am Tag“. Ohne
	// aufraeumen (nil) entfällt das Aufräumen.
	aufraeumen func(e bibliothek.Einstellung, jetzt time.Time) (bibliothek.AufraeumErgebnis, error)
	tagesmarke string
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
		aufraeumen: func(e bibliothek.Einstellung, jetzt time.Time) (bibliothek.AufraeumErgebnis, error) {
			orte, err := bibliothek.OrteAus(os.Getenv)
			if err != nil {
				return bibliothek.AufraeumErgebnis{}, err
			}
			return bibliothek.Neu(orte).Aufraeumen(jetzt, e)
		},
		tagesmarke: bibliothek.TagesmarkePfad(os.Getenv),
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
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t.Aufraeumen, aus, fehler)

	case t.Modus == ModusWiederholen:
		l, err := auswahl.LadeLetzten(an.letzterPfad)
		if err != nil {
			if errors.Is(err, auswahl.ErrKeinZustandsordner) {
				fmt.Fprintln(fehler, sprache.T("No state directory (HOME is not set), so there is no last area to repeat."))
				return exitFehler
			}
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
		return an.speichere(b.Pixel, l.Bereich, t.Aufraeumen, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Name != "":
		b, err := q.Bildschirm(ctx, t.Ziel.Name, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		an.merke(t.Ziel.Name, b.Pixel, b.Pixel.Bounds(), fehler)
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t.Aufraeumen, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Wahl == einstellungen.WahlZeiger:
		b, err := q.AktiverBildschirm(ctx, aufnahme.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err)
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t.Aufraeumen, aus, fehler)

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
	start := time.Now()
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
	diagnose(fehler, "Aufnahme aller Monitore: %v", time.Since(start).Round(time.Millisecond))
	start = time.Now()
	e, err := an.overlay(bilder, modus)
	diagnose(fehler, "Overlay bis Ergebnis und Fenster zu: %v", time.Since(start).Round(time.Millisecond))
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
	return an.speichere(pixel[e.Monitor], e.Bereich, t.Aufraeumen, aus, fehler)
}

// speichere legt den Bereich als neue Aufnahme in der Bibliothek ab und schreibt den Pfad nach aus. Das Original
// der Arbeitsdatei ist nur der gewählte Bereich, nicht der ganze Monitor: Was der Nutzer nicht gewählt hat, gehört
// nicht in die Datei (Kernprinzip 3).
func (an anschluesse) speichere(b *image.NRGBA, bereich image.Rectangle, e bibliothek.Einstellung, aus, fehler io.Writer) int {
	start := time.Now()
	defer func() { diagnose(fehler, "Speichern: %v", time.Since(start).Round(time.Millisecond)) }()
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
	an.raeumeAuf(e, time.Now(), fehler)
	return exitOK
}

// raeumeAuf räumt nach dem Speichern die Bibliothek auf, höchstens einmal am Tag (ADR-0016 Nachtrag 2). Die Aufnahme
// ist dann schon gespeichert: Was hier scheitert, ist nur eine Warnung und ändert den Exit-Code nicht. Auf stdout
// steht weiter nur der Pfad.
func (an anschluesse) raeumeAuf(e bibliothek.Einstellung, jetzt time.Time, fehler io.Writer) {
	if !e.An || an.aufraeumen == nil {
		return
	}
	heute, err := bibliothek.BeanspruchenHeute(an.tagesmarke, jetzt)
	switch {
	case errors.Is(err, bibliothek.ErrKeinZustandsordner):
		fmt.Fprintln(fehler, sprache.T("Automatic cleanup of the library skipped: no state directory (HOME is not set)."))
		return
	case err != nil:
		fmt.Fprintf(fehler, sprache.T("Automatic cleanup of the library skipped: %v")+"\n", err)
		return
	case !heute:
		return
	}
	erg, err := an.aufraeumen(e, jetzt)
	if err != nil {
		fmt.Fprintf(fehler, sprache.T("Automatic cleanup of the library failed: %v")+"\n", err)
	}
	if n := len(erg.Uebersprungen); n > 0 {
		fmt.Fprintf(fehler, sprache.N("Automatic cleanup could not move %d screenshot to the trash.",
			"Automatic cleanup could not move %d screenshots to the trash.", n)+"\n", n)
	}
	for _, u := range erg.Uebersprungen {
		diagnose(fehler, "Aufräumen übersprungen: %s: %v", u.Pfad, u.Grund)
	}
	diagnose(fehler, "Aufräumen: %d in den Papierkorb", len(erg.Verschoben))
}

// merke speichert den Bereich für --repeat. Scheitert das, ist die Aufnahme trotzdem gelungen: nur eine Warnung.
func (an anschluesse) merke(monitor string, b *image.NRGBA, bereich image.Rectangle, fehler io.Writer) {
	l := auswahl.Letzter{Monitor: monitor, Groesse: b.Bounds().Size(), Bereich: bereich}
	err := auswahl.SpeichereLetzten(an.letzterPfad, l)
	switch {
	case errors.Is(err, auswahl.ErrKeinZustandsordner):
		// Fehlertexte des Pakets sind nicht übersetzt; den bekannten Fall übersetzt hier (F-100).
		fmt.Fprintln(fehler, sprache.T("Could not remember the area for --repeat: no state directory (HOME is not set)."))
	case err != nil:
		fmt.Fprintf(fehler, sprache.T("Could not remember the area for --repeat: %v")+"\n", err)
	}
}

// diagnose schreibt nur mit gesetztem SCHNAPP_DEBUG (Fehlersuche bei echten Läufen).
func diagnose(fehler io.Writer, format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		fmt.Fprintf(fehler, "[schnapp] "+format+"\n", args...)
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
