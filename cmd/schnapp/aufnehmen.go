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

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/selection"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// exitAbgebrochen: Der Nutzer hat die Auswahl abgebrochen (Esc im Overlay, Abbruch der Fensterwahl).
const exitAbgebrochen = 3

// Ausfuehrung führt einen Aufnahme-Auftrag aus und liefert den Exit-Code; Tests ersetzen sie.
type Ausfuehrung func(t Auftrag, aus, fehler io.Writer) int

// anschluesse sind die Außenanschlüsse eines Aufnahme-Laufs. Das Programm nimmt die echten (echteAnschluesse),
// Tests ihre Attrappen: keine echte Aufnahme, kein Fenster, keine Datei außerhalb von t.TempDir().
type anschluesse struct {
	quelle   func() (capture.Quelle, error)
	monitore func() ([]ui.Monitor, error)
	overlay  func([]ui.MonitorBild, selection.Modus) (ui.OverlayErgebnis, error)
	// ablegen liefert den absoluten Pfad der Datei und ihren Stand unmittelbar nach dem Schreiben (F-110: Ändert sie
	// sich danach, etwa durch ein Skript, das den Pfad von stdout liest, merkt es das Speichern im Editor).
	ablegen     func(d *document.Dokument) (string, library.Stand, error)
	letzterPfad string
	// aufraeumen räumt die Bibliothek auf; tagesmarke ist die Marke für „höchstens einmal am Tag“. Ohne
	// aufraeumen (nil) entfällt das Aufräumen.
	aufraeumen func(e library.Einstellung, jetzt time.Time) (library.AufraeumErgebnis, error)
	tagesmarke string
	// editor öffnet eine Datei im Editor und kehrt nach dem Schließen zurück (ADR-0020); nil = kein Editor.
	// ohneGrenze ist --ignore-limit (nur bei schnapp DATEI, ADR-0017 Nachtrag 6).
	editor func(pfad string, ohneGrenze bool) error
	// bearbeiten öffnet eine eben abgelegte Aufnahme im Editor, mit dem Dokument aus dem Speicher statt die Datei neu
	// zu lesen (F-112). nil = kein Editor nach der Aufnahme.
	bearbeiten func(a abgelegt) error
	// schutz prüft vor jeder Aufnahme und vor dem Editor, ob der Lauf erlaubt ist (Probe-Build nur im virtuellen KWin,
	// F-229); ein Fehler beendet den Lauf mit exitFehler, bevor etwas aufgenommen oder geöffnet wird. nil = keine Prüfung.
	schutz func() error
}

func echteAnschluesse() anschluesse {
	return anschluesse{
		quelle:   func() (capture.Quelle, error) { return capture.NeuKWin() },
		monitore: ui.Monitore,
		overlay:  ui.Overlay,
		ablegen: func(d *document.Dokument) (string, library.Stand, error) {
			orte, err := library.OrteAus(os.Getenv)
			if err != nil {
				return "", library.Stand{}, err
			}
			sichtbar, err := render.Bild(d)
			if err != nil {
				return "", library.Stand{}, err
			}
			e, err := library.Neu(orte).Ablegen(d, sichtbar, fileformat.Metadaten{})
			if e.Pfad == "" {
				return "", library.Stand{}, err
			}
			// Index oder Vorschau gescheitert: Die Datei liegt trotzdem in der Ablage, das ist der Erfolg. Den Stand
			// sofort merken (F-110, Verify W-5); unlesbar bleibt der Nullwert, dann weicht Speichern auf eine neue
			// Aufnahme aus.
			pfad := filepath.Join(orte.Ablage, filepath.FromSlash(e.Pfad))
			stand, _ := library.StandVon(pfad)
			// Eine liegen gebliebene temporäre Kopie (F-274, Z-4) ist kein Fehler der Aufnahme, aber der Aufrufer
			// meldet sie: Zusammen mit dem Pfad kommt nur ErrTempRest zurück (alle anderen Folgefehler bleiben still).
			if errors.Is(err, library.ErrTempRest) {
				return pfad, stand, err
			}
			return pfad, stand, nil
		},
		letzterPfad: selection.LetzterPfad(os.Getenv),
		aufraeumen: func(e library.Einstellung, jetzt time.Time) (library.AufraeumErgebnis, error) {
			orte, err := library.OrteAus(os.Getenv)
			if err != nil {
				return library.AufraeumErgebnis{}, err
			}
			return library.Neu(orte).Aufraeumen(jetzt, e)
		},
		tagesmarke: library.TagesmarkePfad(os.Getenv),
		editor:     oeffneImEditor,
		bearbeiten: bearbeiteNeu,
		schutz:     probeGuard,
	}
}

// Ausfuehren ist die echte Ausführung.
func (an anschluesse) Ausfuehren(t Auftrag, aus, fehler io.Writer) int {
	if an.schutz != nil {
		if err := an.schutz(); err != nil {
			fmt.Fprintln(fehler, "schnapp (Probe-Build):", err)
			return exitFehler
		}
	}
	if t.Modus == ModusOeffnen {
		if an.editor == nil {
			return exitFehler
		}
		if err := an.editor(t.Datei, t.OhneGrenze); err != nil {
			diagnoseUrsache(fehler, err)
			if ungesichert(err) {
				// Geöffnet war die Datei; gescheitert ist das Speichern beim Schließen (F-302).
				fmt.Fprintf(fehler, i18n.T("The editor could not be opened or saved: %v")+"\n", nutzerFehler(err))
				return exitFehler
			}
			fmt.Fprintf(fehler, i18n.T("Cannot open %s: %s")+"\n", t.Datei, nutzerFehler(err))
			return exitFehler
		}
		return exitOK
	}
	code, neu := an.aufnehmen(t, aus, fehler)
	if neu != nil && t.Editor && an.bearbeiten != nil {
		// Erst hier, nach der Rückkehr aus aufnehmen: Die Verbindung zu KWin ist zu, die Monitorbilder hängen an
		// keinem Aufruf mehr und können freigegeben werden; der Editor bekommt nur das Dokument (F-112). Die Aufnahme
		// liegt schon in der Bibliothek, der Pfad steht auf stdout: Was der Editor meldet, ist nur noch eine Warnung
		// (ADR-0020).
		start := time.Now()
		err := an.bearbeiten(*neu)
		diagnose(fehler, "Editor-Sitzung bis Fenster zu: %v", time.Since(start).Round(time.Millisecond))
		if err != nil {
			diagnoseUrsache(fehler, err)
			fmt.Fprintf(fehler, i18n.T("The editor could not be opened or saved: %v")+"\n", nutzerFehler(err))
			// Die Aufnahme liegt in der Bibliothek, ein Fensterfehler allein bleibt eine Warnung (Exit 0). Gingen aber
			// Änderungen verloren, ist es ein Fehler (ADR-0020 Nachtrag 7, F-302).
			if code == exitOK && ungesichert(err) {
				code = exitFehler
			}
		}
	}
	return code
}

// abgelegt ist eine eben in der Bibliothek abgelegte Aufnahme, die der Editor danach öffnet (F-112).
type abgelegt struct {
	pfad  string             // absoluter Pfad der Datei
	d     *document.Dokument // das abgelegte Dokument; hält nur den gewählten Bereich, nicht die Monitorbilder
	stand library.Stand      // Stand der Datei unmittelbar nach dem Ablegen (F-110)
}

// aufnehmen nimmt auf und legt in der Bibliothek ab. Zurück kommen der Exit-Code und bei Erfolg die abgelegte
// Aufnahme. Alles, was die Aufnahme braucht (Verbindung zu KWin, Bilder aller Monitore), lebt nur bis zur Rückkehr.
func (an anschluesse) aufnehmen(t Auftrag, aus, fehler io.Writer) (int, *abgelegt) {
	q, err := an.quelle()
	if err != nil {
		return meldeFehler(fehler, err), nil
	}
	defer q.Close()
	ctx := context.Background()

	switch {
	case t.Modus == ModusFenster:
		b, err := q.Fenster(ctx, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t, aus, fehler)

	case t.Modus == ModusWiederholen:
		l, err := selection.LadeLetzten(an.letzterPfad)
		if err != nil {
			if errors.Is(err, selection.ErrKeinZustandsordner) {
				fmt.Fprintln(fehler, i18n.T("No state directory (HOME is not set), so there is no last area to repeat."))
				return exitFehler, nil
			}
			if errors.Is(err, selection.ErrKeinLetzter) {
				fmt.Fprintln(fehler, i18n.T("No area to repeat yet. Select one with --region first."))
				return exitFehler, nil
			}
			return meldeFehler(fehler, err), nil
		}
		b, err := q.Bildschirm(ctx, l.Monitor, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		if b.Pixel.Bounds().Size() != l.Groesse || !l.SkalaPasst(b.Skala) {
			fmt.Fprintln(fehler, i18n.T("The monitor of the last area has changed. Select a new area with --region."))
			return exitFehler, nil
		}
		return an.speichere(b.Pixel, l.Bereich, t, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Name != "":
		b, err := q.Bildschirm(ctx, t.Ziel.Name, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		an.merke(t.Ziel.Name, b.Pixel, b.Skala, b.Pixel.Bounds(), fehler)
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Wahl == settings.WahlZeiger:
		b, err := q.AktiverBildschirm(ctx, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t, aus, fehler)

	case t.Modus == ModusBildschirm && t.Ziel.Wahl == settings.WahlAlle:
		// Alle Monitore zusammengesetzt (CaptureWorkspace). Ohne Monitorname gibt es keinen „letzten Bereich“ zu merken.
		b, err := q.Alle(ctx, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		return an.speichere(b.Pixel, b.Pixel.Bounds(), t, aus, fehler)
	}

	// Overlay: --region (Bereich) oder --screen mit der Einstellung pick (Monitor per Klick).
	modus := selection.ModusBereich
	if t.Modus == ModusBildschirm {
		modus = selection.ModusMonitor
	}
	monitore, err := an.monitore()
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Could not query the monitors: %v")+"\n", err)
		return exitFehler, nil
	}
	if len(monitore) == 0 {
		fmt.Fprintln(fehler, i18n.T("No monitors found."))
		return exitFehler, nil
	}
	// Erst alle Monitore aufnehmen, dann im eingefrorenen Bild wählen (knowledge/topics/kwin-screenshot.md).
	start := time.Now()
	bilder := make([]ui.MonitorBild, 0, len(monitore))
	pixel := make([]*image.NRGBA, 0, len(monitore))
	skalen := make([]float64, 0, len(monitore))
	for _, m := range monitore {
		b, err := q.Bildschirm(ctx, m.Name, capture.Optionen{})
		if err != nil {
			return meldeFehler(fehler, err), nil
		}
		bilder = append(bilder, ui.MonitorBild{Name: m.Name, Bild: b.Pixel})
		pixel = append(pixel, b.Pixel)
		skalen = append(skalen, b.Skala)
	}
	diagnose(fehler, "Aufnahme aller Monitore: %v", time.Since(start).Round(time.Millisecond))
	start = time.Now()
	e, err := an.overlay(bilder, modus)
	diagnose(fehler, "Overlay bis Ergebnis und Fenster zu: %v", time.Since(start).Round(time.Millisecond))
	if errors.Is(err, ui.ErrAbgebrochen) {
		return exitAbgebrochen, nil
	}
	if err != nil {
		return meldeFehler(fehler, err), nil
	}
	if e.Monitor < 0 || e.Monitor >= len(pixel) || !e.Bereich.In(pixel[e.Monitor].Bounds()) || e.Bereich.Empty() {
		fmt.Fprintln(fehler, i18n.T("The selection is outside the captured image."))
		return exitFehler, nil
	}
	an.merke(monitore[e.Monitor].Name, pixel[e.Monitor], skalen[e.Monitor], e.Bereich, fehler)
	return an.speichere(pixel[e.Monitor], e.Bereich, t, aus, fehler)
}

// speichere legt den Bereich als neue Aufnahme in der Bibliothek ab und schreibt den Pfad nach aus. Das Original
// der Arbeitsdatei ist nur der gewählte Bereich, nicht der ganze Monitor: Was der Nutzer nicht gewählt hat, gehört
// nicht in die Datei (Kernprinzip 3). Den Editor öffnet nicht speichere, sondern Ausfuehren nach der Rückkehr (F-112);
// zurück kommt die abgelegte Aufnahme. document.Neu kopiert den Bereich: Das Dokument hält das Monitorbild nicht fest.
func (an anschluesse) speichere(b *image.NRGBA, bereich image.Rectangle, t Auftrag, aus, fehler io.Writer) (int, *abgelegt) {
	// Die Messung endet mit dem Ablegen; die Editor-Sitzung misst eine eigene Zeile (sonst stand die ganze
	// Bearbeitungszeit unter „Speichern“, Echtlauf #86).
	start := time.Now()
	d, err := document.Neu(b.SubImage(bereich))
	if err != nil {
		return meldeFehler(fehler, err), nil
	}
	if t.Ausgabe != "" {
		// --output (ADR-0030 Punkt 2): statt in die Bibliothek an den angegebenen Ort; kein Editor, kein Aufräumen.
		abs, err := schreibeBild(d, t.Ausgabe, true)
		diagnose(fehler, "Schreiben nach --output: %v", time.Since(start).Round(time.Millisecond))
		if err != nil {
			diagnose(fehler, "Ursache: %v", err)
			fmt.Fprintf(fehler, i18n.T("Could not write %s: %s")+"\n", t.Ausgabe, schreibFehler(err))
			return exitFehler, nil
		}
		gibPfadAus(aus, fehler, t.JSON, abs, d.Groesse()) // Exit 0 auch ohne Zeile, wie unten (F-180)
		return exitOK, nil
	}
	pfad, stand, err := an.ablegen(d)
	diagnose(fehler, "Ablegen in der Bibliothek: %v", time.Since(start).Round(time.Millisecond))
	if err != nil && !(pfad != "" && errors.Is(err, library.ErrTempRest)) {
		// Ohne Pfad und ohne Paketpräfix wie beim Editor (F-117, F-303, F-305): Der Nutzer sieht den übersetzten
		// Grund, die rohe Ursache nur mit SCHNAPP_DEBUG. Library.Ablegen liefert ErrTempRest nie ohne Pfad; ein
		// eigener Zweig dafür war toter Code (F-306) und entfällt, ein solcher Fall läuft hier über den
		// allgemeinen Text.
		diagnose(fehler, "Ursache des Ablegens: %v", err)
		fmt.Fprintf(fehler, i18n.T("Could not save the screenshot: %v")+"\n", speicherFehler(err).Error())
		return exitFehler, nil
	}
	if err != nil {
		// Abgelegt, aber eine temporäre Kopie blieb liegen (F-274, Z-4): Sie zeigt nach einem späteren Schwärzen noch
		// den alten Stand. Warnen, der Weg geht weiter. Der Text trägt weder Pfad noch Paketpräfix noch rohe Ursache
		// (F-303, wie F-117 beim Editor); die Ursache steht nur mit SCHNAPP_DEBUG im Debug-Log.
		diagnose(fehler, "Ursache der temporären Kopie: %v", err)
		fmt.Fprintln(fehler, i18n.T("Warning: a temporary copy of the screenshot could not be removed and may still show it after you redact it later."))
	}
	// Die Aufnahme liegt in der Bibliothek; geht nur die Pfadzeile verloren, meldet gibPfadAus das (F-162). Nicht
	// verschlucken: Skripte verlassen sich auf sie. Der Exit-Code bleibt 0 (F-180): Die Aufnahme liegt vor, und der
	// Editor öffnet trotzdem; ein Fehler-Code ließe ein Skript die Aufnahme wiederholen. Ist stdout eine geschlossene
	// Pipe, beendet SIGPIPE das Programm wie jedes Go-Programm schon beim Schreiben (Go-Standard, bewusst so gelassen):
	// Wer die Ausgabe nicht mehr liest, will auch keinen Editor. Mit --json steht dieselbe Angabe als JSON-Zeile da
	// (ADR-0030 Punkt 1).
	gibPfadAus(aus, fehler, t.JSON, pfad, d.Groesse())
	an.raeumeAuf(t.Aufraeumen, time.Now(), fehler)
	return exitOK, &abgelegt{pfad: pfad, d: d, stand: stand}
}

// raeumeAuf räumt nach dem Speichern die Bibliothek auf, höchstens einmal am Tag (ADR-0016 Nachtrag 2). Die Aufnahme
// ist dann schon gespeichert: Was hier scheitert, ist nur eine Warnung und ändert den Exit-Code nicht. Auf stdout
// steht weiter nur der Pfad.
func (an anschluesse) raeumeAuf(e library.Einstellung, jetzt time.Time, fehler io.Writer) {
	if !e.An || an.aufraeumen == nil {
		return
	}
	heute, err := library.BeanspruchenHeute(an.tagesmarke, jetzt)
	switch {
	case errors.Is(err, library.ErrKeinZustandsordner):
		fmt.Fprintln(fehler, i18n.T("Automatic cleanup of the library skipped: no state directory (HOME is not set)."))
		return
	case err != nil:
		fmt.Fprintf(fehler, i18n.T("Automatic cleanup of the library skipped: %v")+"\n", err)
		return
	case !heute:
		return
	}
	erg, err := an.aufraeumen(e, jetzt)
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Automatic cleanup of the library failed: %v")+"\n", err)
	}
	if n := len(erg.Uebersprungen); n > 0 {
		fmt.Fprintf(fehler, i18n.N("Automatic cleanup could not move %d screenshot to the trash.",
			"Automatic cleanup could not move %d screenshots to the trash.", n)+"\n", n)
	}
	for _, u := range erg.Uebersprungen {
		diagnose(fehler, "Aufräumen übersprungen: %s: %v", u.Pfad, u.Grund)
	}
	diagnose(fehler, "Aufräumen: %d in den Papierkorb", len(erg.Verschoben))
}

// merke speichert den Bereich für --repeat. Scheitert das, ist die Aufnahme trotzdem gelungen: nur eine Warnung.
func (an anschluesse) merke(monitor string, b *image.NRGBA, skala float64, bereich image.Rectangle, fehler io.Writer) {
	l := selection.Letzter{Monitor: monitor, Groesse: b.Bounds().Size(), Skala: skala, Bereich: bereich}
	err := selection.SpeichereLetzten(an.letzterPfad, l)
	switch {
	case errors.Is(err, selection.ErrKeinZustandsordner):
		// Fehlertexte des Pakets sind nicht übersetzt; den bekannten Fall übersetzt hier (F-100).
		fmt.Fprintln(fehler, i18n.T("Could not remember the area for --repeat: no state directory (HOME is not set)."))
	case err != nil:
		fmt.Fprintf(fehler, i18n.T("Could not remember the area for --repeat: %v")+"\n", err)
	}
}

// diagnose schreibt nur mit gesetztem SCHNAPP_DEBUG (Fehlersuche bei echten Läufen).
func diagnose(fehler io.Writer, format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		fmt.Fprintf(fehler, "[schnapp] "+format+"\n", args...)
	}
}

// diagnoseUrsache schreibt mit gesetztem SCHNAPP_DEBUG die ursprüngliche Ursache eines übersetzten Speicherfehlers
// (F-129). Sie kann Pfade enthalten und steht deshalb nie in der normalen Ausgabe.
func diagnoseUrsache(fehler io.Writer, err error) {
	if u := ursacheSpeicherfehler(err); u != "" {
		diagnose(fehler, "Ursache des Speicherfehlers: %s", u)
	}
}

// meldeFehler schreibt einen Aufnahmefehler und liefert den Exit-Code. Ein Abbruch durch den Nutzer ist kein Fehler.
func meldeFehler(fehler io.Writer, err error) int {
	if errors.Is(err, capture.ErrAbgebrochen) {
		return exitAbgebrochen
	}
	fmt.Fprintln(fehler, err)
	return exitFehler
}
