// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Datei → „New capture“ (Strg+N, ADR-0046 Punkt 7): startet dasselbe Programm mit NeueAufnahmeArgument als abgelösten
// Prozess.
// Er nimmt auf wie der globale Kurzbefehl (Bereich als Voreinstellung, [capture] delay und cursor liest er selbst),
// legt in der Bibliothek ab und öffnet ein eigenes Editorfenster; dieses Fenster bleibt, wie es ist.

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// NeueAufnahmeArgument ist das Argument, mit dem Strg+N das eigene Programm startet: cmd/schnapp schließt damit erst die
// geerbten Deskriptoren ohne CLOEXEC und nimmt dann auf wie ohne Argumente (Kritik K-7).
const NeueAufnahmeArgument = "--internal-new-capture"

// neueAufnahmeSperre: So lange nach einem Start startet Strg+N keinen weiteren Prozess (Kritik K-2: gehaltene Taste
// wiederholt den Tastendruck, ein Doppelklick im Menü ebenso).
var neueAufnahmeSperre = 3 * time.Second

// neueAufnahmeProgramm liefert das eigene Programm; Naht für Tests.
var neueAufnahmeProgramm = os.Executable

// neueAufnahmeStufen sind die Verzögerungen der Einträge „New capture in N s“ im Menü Datei, wie die Stufen der
// Einstellungstafel (ADR-0042 Nachtrag 2).
var neueAufnahmeStufen = [...]int{3, 5, 10}

// neueAufnahmeArgumente liefert die Argumente des neuen Prozesses: sek > 0 gibt --delay=SEK mit (vor [capture] delay),
// sonst gilt die Einstellung (ADR-0042 Nachtrag 2 Punkt 2).
func neueAufnahmeArgumente(sek int) []string {
	if sek > 0 {
		return []string{NeueAufnahmeArgument, "--delay=" + strconv.Itoa(sek)}
	}
	return []string{NeueAufnahmeArgument}
}

// neueAufnahmeStart startet programm abgelöst mit args und holt es nach dem Ende ab (kein Zombie); Naht für Tests.
var neueAufnahmeStart = func(programm string, args []string) error {
	c := exec.Command(programm, args...)
	c.Dir = "/"
	// Eigene Sitzung: Ein Strg+C im Terminal dieses Fensters trifft die neue Aufnahme nicht, und sie überlebt das
	// Schließen dieses Fensters. Ein- und Ausgabe leer (der neue Prozess schreibt den Pfad der Aufnahme auf stdout);
	// die Fehlerausgabe nur zur Diagnose, sonst schriebe er nach dem Schließen des Terminals ins Leere, und nie in eine
	// Pipe: `schnapp 2>&1 | tee` endete sonst erst mit dem zweiten Fenster (wie beim Halter, F-325; Kritik K-6).
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if stderrWeitergeben(os.Getenv("SCHNAPP_DEBUG") != "", wlclip.IstPipe(2)) {
		c.Stderr = os.Stderr
	}
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }() // Ende abholen; Abbruch im Overlay (Exit 3) ist kein Fehler dieses Fensters
	return nil
}

// stderrWeitergeben entscheidet, ob die neue Aufnahme die Fehlerausgabe erbt: nur zur Diagnose und nie als Pipe (F-1012).
func stderrWeitergeben(debug, pipe bool) bool { return debug && !pipe }

// neueAufnahmeGrau ist der Grund, warum „New capture“ nicht geht (leer: geht): Mit geerbtem WAYLAND_SOCKET fände der
// neue Prozess den Compositor nicht sicher (wie der Halter der Zwischenablage, ADR-0024 Nachtrag 5).
func neueAufnahmeGrau() string {
	if wlclip.HalterMoeglich(os.Getenv) != nil {
		return i18n.T("Not available: this window was started on an inherited Wayland connection.")
	}
	return ""
}

// neueAufnahme startet die neue Aufnahme, mit sek > 0 nach so vielen Sekunden (sonst nach [capture] delay); scheitert
// der Start, sagt es die Fußzeile.
func (e *editor) neueAufnahme(sek int) {
	if neueAufnahmeGrau() != "" || e.erzwungen || e.s.SignalVorgemerkt() {
		return
	}
	if !e.neuGestartet.IsZero() && time.Since(e.neuGestartet) < neueAufnahmeSperre {
		return // eben gestartet; die Fußzeile sagt es schon
	}
	programm, err := neueAufnahmeProgramm()
	if err == nil {
		err = neueAufnahmeStart(programm, neueAufnahmeArgumente(sek))
	}
	if err != nil {
		diagnoseEditor("Neue Aufnahme: %v", err)
		e.meldeNebenbei(i18n.T("Could not start a new capture."), true)
		return
	}
	e.neuGestartet = time.Now()
	e.meldeNebenbei(i18n.T("Starting a new capture…"), false)
}
