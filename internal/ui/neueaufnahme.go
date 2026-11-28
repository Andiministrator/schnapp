// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Datei → „New capture“ (Strg+N, ADR-0046 Punkt 7): startet dasselbe Programm ohne Argumente als abgelösten Prozess.
// Er nimmt auf wie der globale Kurzbefehl (Bereich als Voreinstellung, [capture] delay und cursor liest er selbst),
// legt in der Bibliothek ab und öffnet ein eigenes Editorfenster; dieses Fenster bleibt, wie es ist.

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// neueAufnahmeProgramm liefert das eigene Programm; Naht für Tests.
var neueAufnahmeProgramm = os.Executable

// neueAufnahmeStart startet programm abgelöst und holt es nach dem Ende ab (kein Zombie); Naht für Tests.
var neueAufnahmeStart = func(programm string) error {
	c := exec.Command(programm)
	c.Dir = "/"
	// Eigene Sitzung: Ein Strg+C im Terminal dieses Fensters trifft die neue Aufnahme nicht, und sie überlebt das
	// Schließen dieses Fensters. Ein- und Ausgabe leer (der neue Prozess schreibt den Pfad der Aufnahme auf stdout);
	// die Fehlerausgabe nur zur Diagnose, sonst schriebe er nach dem Schließen des Terminals ins Leere.
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		c.Stderr = os.Stderr
	}
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }() // Ende abholen; Abbruch im Overlay (Exit 3) ist kein Fehler dieses Fensters
	return nil
}

// neueAufnahmeGrau ist der Grund, warum „New capture“ nicht geht (leer: geht): Mit geerbtem WAYLAND_SOCKET fände der
// neue Prozess den Compositor nicht sicher (wie der Halter der Zwischenablage, ADR-0024 Nachtrag 5).
func neueAufnahmeGrau() string {
	if wlclip.HalterMoeglich(os.Getenv) != nil {
		return i18n.T("Not available: this window was started on an inherited Wayland connection.")
	}
	return ""
}

// neueAufnahme startet die neue Aufnahme; scheitert der Start, sagt es die Fußzeile.
func (e *editor) neueAufnahme() {
	if neueAufnahmeGrau() != "" || e.erzwungen || e.s.SignalVorgemerkt() {
		return
	}
	programm, err := neueAufnahmeProgramm()
	if err == nil {
		err = neueAufnahmeStart(programm)
	}
	if err != nil {
		diagnoseEditor("Neue Aufnahme: %v", err)
		e.meldeNebenbei(i18n.T("Could not start a new capture."), true)
		return
	}
	e.meldeNebenbei(i18n.T("Starting a new capture…"), false)
}
