// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Sitzungs-QA vom 24.11.2026 (Kritiker Fable), QS-1 (Folge F-603): Hängt stderr (eine volle Pipe, die niemand liest),
// darf die Meldung beim zweiten Signal das Beenden nicht aufhalten — weder in der Wache von --annotate noch in
// mcpSignale noch in der Signalwache von --output/--export. Belegt mit Nähten und mit echten Signalen im Kindprozess
// (Muster wie beschriften_signal_kind_test.go).

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// stderrBlockiert ist ein stderr, das niemand liest: Write kehrt nie zurück. erreicht meldet den ersten Aufruf.
type stderrBlockiert struct{ erreicht chan struct{} }

func (b stderrBlockiert) Write([]byte) (int, error) {
	select {
	case b.erreicht <- struct{}{}:
	default:
	}
	select {}
}

// QS-1a: mcpSignale beendet beim zweiten Signal, auch wenn seine Meldung am blockierten stderr hängt.
func TestMcpZweitesSignalBlockiertesStderr(t *testing.T) {
	kanal := make(chan os.Signal, 4)
	fertig := make(chan struct{})
	defer close(fertig)
	beendet := make(chan int, 1)
	b := stderrBlockiert{erreicht: make(chan struct{}, 1)}
	go mcpSignale(kanal, func() {}, func(c int) { beendet <- c }, b, fertig)
	kanal <- syscall.SIGTERM
	kanal <- syscall.SIGTERM
	select {
	case c := <-beendet:
		if c != exitFehler {
			t.Errorf("Exit %d, erwartet %d", c, exitFehler)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("zweites Signal: mcpSignale hängt am blockierten stderr, beende wird nie gerufen (nur SIGKILL hülfe)")
	}
}

// QS-1a': Auch während ein annotate speichert (die Lage hält), hängt mcpSignale nicht an der Meldung „stopping after the
// running save“: Endet das Speichern, wird beendet.
func TestMcpZweitesSignalBeimSpeichernBlockiertesStderr(t *testing.T) {
	lage := new(beschriftenLage)
	lage.setze(1)
	defer lage.ende()
	kanal := make(chan os.Signal, 4)
	fertig := make(chan struct{})
	defer close(fertig)
	beendet := make(chan int, 1)
	b := stderrBlockiert{erreicht: make(chan struct{}, 1)}
	go mcpSignale(kanal, func() {}, func(c int) { beendet <- c }, b, fertig)
	kanal <- syscall.SIGTERM
	kanal <- syscall.SIGTERM
	select {
	case <-beendet:
		t.Fatal("beendet, während das Speichern noch Fortschritt meldet")
	case <-time.After(500 * time.Millisecond):
	}
	lage.ende()
	select {
	case <-beendet:
	case <-time.After(3 * time.Second):
		t.Fatal("nach dem Speichern: mcpSignale hängt am blockierten stderr, beende wird nie gerufen")
	}
}

// QS-1b: CLI --annotate, Phase 0 oder hängendes Speichern (f kehrt nicht zurück), stderr blockiert: Das zweite Signal
// beendet den Prozess.
func TestAnnotateZweitesSignalBlockiertesStderr(t *testing.T) {
	alteAn, alteAb, altesEnde := meldeAn, meldeAb, beendeProzess
	defer func() { meldeAn, meldeAb, beendeProzess = alteAn, alteAb, altesEnde }()
	var k chan<- os.Signal
	bereit := make(chan struct{})
	meldeAn = func(c chan<- os.Signal, _ ...os.Signal) { k = c; close(bereit) }
	meldeAb = func(chan<- os.Signal) {}
	beendet := make(chan int, 1)
	beendeProzess = func(c int) { beendet <- c }
	b := stderrBlockiert{erreicht: make(chan struct{}, 1)}
	halt, fertig := make(chan struct{}), make(chan struct{})
	// Vor dem Zurücksetzen der Nähte: die Wache meldet noch ab. Hängt sie (Fehlerfall: Meldung am blockierten stderr),
	// nicht ewig warten, sonst hinge der ganze Testlauf.
	defer func() {
		close(halt)
		select {
		case <-fertig:
		case <-time.After(5 * time.Second):
		}
	}()
	go func() {
		defer close(fertig)
		mitBeschriftenWache(new(beschriftenLage), b, func(ctx context.Context) { <-halt }) // f hängt (z. B. Lesen einer FIFO)
	}()
	<-bereit
	k <- syscall.SIGTERM
	k <- syscall.SIGTERM
	select {
	case c := <-beendet:
		if c != exitFehler {
			t.Errorf("Exit %d, erwartet %d", c, exitFehler)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("zweites Signal: die Wache hängt am blockierten stderr, beendeProzess wird nie gerufen (nur SIGKILL hülfe)")
	}
}

// QS-1c: dasselbe für die Signalwache von --output/--export (F-442): Das Schreiben hängt, stderr auch.
func TestSignalwacheZweitesSignalBlockiertesStderr(t *testing.T) {
	alteAn, alteAb, altesEnde, alterFehler := meldeAn, meldeAb, beendeProzess, wacheFehler
	defer func() { meldeAn, meldeAb, beendeProzess, wacheFehler = alteAn, alteAb, altesEnde, alterFehler }()
	var k chan<- os.Signal
	bereit := make(chan struct{})
	meldeAn = func(c chan<- os.Signal, _ ...os.Signal) { k = c; close(bereit) }
	meldeAb = func(chan<- os.Signal) {}
	beendet := make(chan int, 1)
	beendeProzess = func(c int) { beendet <- c }
	wacheFehler = func() io.Writer { return stderrBlockiert{erreicht: make(chan struct{}, 1)} }
	halt, fertig := make(chan struct{}), make(chan struct{})
	// Vor dem Zurücksetzen der Nähte: die Wache meldet noch ab. Hängt sie (Fehlerfall: Meldung am blockierten stderr),
	// nicht ewig warten, sonst hinge der ganze Testlauf.
	defer func() {
		close(halt)
		select {
		case <-fertig:
		case <-time.After(5 * time.Second):
		}
	}()
	go func() {
		defer close(fertig)
		mitSignalwache(func(func() bool, func(string)) error { <-halt; return nil })
	}()
	<-bereit
	k <- syscall.SIGTERM
	k <- syscall.SIGTERM
	select {
	case <-beendet:
	case <-time.After(3 * time.Second):
		t.Fatal("zweites Signal: die Signalwache hängt am blockierten stderr, beendeProzess wird nie gerufen")
	}
}

// --- dasselbe mit echten Signalen im Kindprozess ---

const kindStderrUmgebung = "SCHNAPP_TEST_KIND_STDERR"

// TestKindStderrBlockiert ist nur der Kindprozess: Modus „cli“ fährt die echte Wache von --annotate mit einem f, das
// hängt (Phase 0), Modus „mcp“ die Verdrahtung von laufeMcp (meldeAn, mcpSignale mit beendeProzess) mit einem hängenden
// Werkzeug; stderr der Wache ist in beiden Fällen blockiert. Die Nähte meldeAn/meldeAb/beendeProzess bleiben unverändert.
func TestKindStderrBlockiert(t *testing.T) {
	modus := os.Getenv(kindStderrUmgebung)
	if modus == "" {
		return
	}
	b := stderrBlockiert{erreicht: make(chan struct{}, 1)}
	go func() { <-b.erreicht; os.Stdout.WriteString("KIND-STDERR-BLOCKIERT\n") }()
	switch modus {
	case "cli":
		go func() { time.Sleep(200 * time.Millisecond); os.Stdout.WriteString("KIND-BEREIT\n") }()
		mitBeschriftenWache(new(beschriftenLage), b, func(ctx context.Context) { select {} })
	case "mcp":
		_, abbrechen := context.WithCancel(context.Background())
		kanal := make(chan os.Signal, 4)
		meldeAn(kanal, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
		fertig := make(chan struct{})
		go mcpSignale(kanal, abbrechen, beendeProzess, b, fertig)
		os.Stdout.WriteString("KIND-BEREIT\n")
		select {} // ein Werkzeug hängt, Bediene kehrt nicht zurück
	}
}

// kindStderrLauf startet das Kind, schickt zwei SIGTERM und verlangt, dass es danach mit Exit 1 endet.
func kindStderrLauf(t *testing.T, modus string) {
	if os.Getenv(kindStderrUmgebung) != "" {
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestKindStderrBlockiert$", "-test.count=1")
	cmd.Env = append(os.Environ(), kindStderrUmgebung+"="+modus)
	aus, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() }) // nur der eigene Kindprozess
	zeilen := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(aus)
		for sc.Scan() {
			zeilen <- sc.Text()
		}
		close(zeilen)
	}()
	warte := func(soll string, frist time.Duration) bool {
		ende := time.After(frist)
		for {
			select {
			case z, ok := <-zeilen:
				if !ok {
					return false
				}
				if z == soll {
					return true
				}
			case <-ende:
				return false
			}
		}
	}
	if !warte("KIND-BEREIT", 30*time.Second) {
		t.Fatal("Kind nicht bereit")
	}
	fertig := make(chan error, 1)
	cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(300 * time.Millisecond)
	cmd.Process.Signal(syscall.SIGTERM)
	blockiert := warte("KIND-STDERR-BLOCKIERT", 3*time.Second)
	go func() { fertig <- cmd.Wait() }()
	select {
	case <-fertig:
		if !blockiert {
			t.Errorf("%s: die Meldung erreichte das blockierte stderr nicht (Aufbau)", modus)
		}
		if code := cmd.ProcessState.ExitCode(); code != exitFehler {
			t.Errorf("%s: Ende %v, erwartet Exit %d", modus, cmd.ProcessState, exitFehler)
		}
	case <-time.After(8 * time.Second):
		cmd.Process.Signal(syscall.SIGINT)
		cmd.Process.Signal(syscall.SIGHUP)
		select {
		case <-fertig:
			t.Fatalf("%s: erst das dritte/vierte Signal beendet", modus)
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: zwei SIGTERM, dann SIGINT und SIGHUP beenden den Prozess nicht (Meldung hängt am blockierten stderr: %v) — nur SIGKILL hilft", modus, blockiert)
		}
	}
}

func TestEchteSignaleBlockiertesStderrCLI(t *testing.T) { kindStderrLauf(t, "cli") }
func TestEchteSignaleBlockiertesStderrMCP(t *testing.T) { kindStderrLauf(t, "mcp") }
