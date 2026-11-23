// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-603 (Karte #228): Hängt die Ausgabe von --annotate nach dem Speichern an einem blockierten stdout, beendet ein
// Signal den Prozess trotzdem. Die Tests laufen mit echten Signalen in einem Kindprozess (dieselbe Testdatei, über
// eine Umgebungsvariable gewählt), weil nur so die echte Abmeldung der Wache (signal.Stop) mitspielt; die Nähte
// meldeAn/meldeAb/beendeProzess bleiben im Kind unverändert.

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// kindAnnotateUmgebung wählt den Kindmodus: „still“ (kein Signal vor der Ausgabe) oder „signal“ (ein Signal an sich
// selbst während des Speichern, wie im Fall aus F-603). kindAnnotateHeim ist das Heim, das der Elternprozess anlegt und
// aufräumt (ein durch das Signal beendetes Kind räumt nichts mehr auf).
const (
	kindAnnotateUmgebung = "SCHNAPP_TEST_KIND_ANNOTATE"
	kindAnnotateHeim     = "SCHNAPP_TEST_KIND_ANNOTATE_HEIM"
	kindBlockiert        = "KIND-STDOUT-BLOCKIERT"
)

// blockierenderSchreiber meldet seinen ersten Aufruf auf stderr und kehrt nie zurück (eine volle Pipe, die niemand liest).
type blockierenderSchreiber struct{}

func (blockierenderSchreiber) Write([]byte) (int, error) {
	os.Stderr.WriteString(kindBlockiert + "\n")
	select {}
}

// TestKindAnnotateBlockiertesStdout ist nur der Kindprozess der Tests unten; ohne die Umgebungsvariable tut er nichts.
func TestKindAnnotateBlockiertesStdout(t *testing.T) {
	modus := os.Getenv(kindAnnotateUmgebung)
	if modus == "" {
		return
	}
	i18n.Setze(nil)
	heim := os.Getenv(kindAnnotateHeim)
	os.Setenv("HOME", heim)
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		os.Setenv(name, "")
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	pfad := helleAufnahme(t, library.Neu(orte))
	liste := filepath.Join(heim, "liste.json")
	if err := os.WriteFile(liste, []byte(`{"annotations":[{"type":"redact","from":[0,0],"to":[100,100]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if modus == "signal" {
		saveprogress.SetzeHaken(func(schritt string) {
			if schritt == "ersetzt" {
				syscall.Kill(os.Getpid(), syscall.SIGTERM)
				time.Sleep(100 * time.Millisecond) // die Wache nimmt es an, während das Speichern noch läuft
			}
		})
	}
	beschrifte(Aufruf{Datei: pfad, Spez: liste}, blockierenderSchreiber{}, os.Stderr, document.KanteAbriss)
	t.Fatal("beschrifte ist trotz blockiertem stdout zurückgekehrt")
}

// starteAnnotateKind startet den Kindprozess und wartet, bis er an stdout hängt. Zurück kommen der Prozess, ein Kanal
// mit seinem Ende und der bis dahin gelesene stderr-Text.
func starteAnnotateKind(t *testing.T, modus string) (*exec.Cmd, <-chan error, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestKindAnnotateBlockiertesStdout$", "-test.count=1")
	cmd.Env = append(os.Environ(), kindAnnotateUmgebung+"="+modus, kindAnnotateHeim+"="+t.TempDir())
	fehler, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() }) // nur der eigene Kindprozess, falls ein Test vorher scheitert
	gelesen := make(chan string, 1)
	rest := make(chan struct{})
	go func() {
		var text strings.Builder
		zeilen := bufio.NewScanner(fehler)
		gemeldet := false
		for zeilen.Scan() {
			text.WriteString(zeilen.Text() + "\n")
			if !gemeldet && zeilen.Text() == kindBlockiert {
				gemeldet = true
				gelesen <- text.String()
			}
		}
		if !gemeldet {
			gelesen <- text.String()
		}
		close(rest)
	}()
	ende := make(chan error, 1)
	var text string
	select {
	case text = <-gelesen:
	case <-time.After(2 * time.Minute): // Speichern unter -race ist langsam
		t.Fatal("das Kind erreicht die Ausgabe nicht")
	}
	if !strings.Contains(text, kindBlockiert) {
		t.Fatalf("das Kind endete vor der Ausgabe:\n%s", text)
	}
	go func() { <-rest; ende <- cmd.Wait() }()
	return cmd, ende, text
}

// erwarteSignalEnde prüft, dass das Kind nach sig binnen 10 s endet, und zwar durch genau dieses Signal.
func erwarteSignalEnde(t *testing.T, cmd *exec.Cmd, ende <-chan error, sig syscall.Signal) {
	t.Helper()
	if err := cmd.Process.Signal(sig); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ende:
	case <-time.After(10 * time.Second):
		t.Fatalf("%v beendet --annotate mit blockiertem stdout nicht (nur SIGKILL hülfe)", sig)
	}
	st, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !st.Signaled() || st.Signal() != sig {
		t.Errorf("Ende %v, erwartet durch %v", cmd.ProcessState, sig)
	}
}

// TestAnnotateBlockiertesStdoutEndetBeiSignal (F-603): Nach dem Speichern ist die Wache abgemeldet; ein Signal
// während der hängenden Ausgabe beendet den Prozess wie ohne Schnapp (SIGTERM, SIGINT, SIGHUP).
func TestAnnotateBlockiertesStdoutEndetBeiSignal(t *testing.T) {
	if os.Getenv(kindAnnotateUmgebung) != "" {
		return
	}
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd, ende, _ := starteAnnotateKind(t, "still")
			erwarteSignalEnde(t, cmd, ende, sig)
		})
	}
}

// TestAnnotateSignalBeimSpeichernDannBlockiertesStdout (F-603, der Fall aus dem Befund): Ein erstes Signal kommt
// während des Speicherns (das Speichern wird fertig, die Meldung steht auf stderr), danach hängt stdout. Das zweite
// Signal beendet den Prozess sofort, nicht erst SIGKILL.
func TestAnnotateSignalBeimSpeichernDannBlockiertesStdout(t *testing.T) {
	if os.Getenv(kindAnnotateUmgebung) != "" {
		return
	}
	cmd, ende, text := starteAnnotateKind(t, "signal")
	if !strings.Contains(text, "written completely") {
		t.Errorf("Meldung nach dem Speichern fehlt:\n%s", text)
	}
	erwarteSignalEnde(t, cmd, ende, syscall.SIGTERM)
}
