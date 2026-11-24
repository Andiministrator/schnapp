// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

// F-618 (Karte #239): SIGTERM, SIGINT und SIGHUP lösen Gehaltenes und beenden die Fernbedienung mit Exit 1. Ein Test
// mit der Attrappe und ein Test mit echtem Signal an einen eigenen Kindprozess (dieselbe Testdatei, über eine
// Umgebungsvariable gewählt).

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

const kindFBUmgebung = "SCHNAPP_TEST_KIND_FERNBEDIENUNG"

// TestKindFernbedienungHaelt ist nur der Kindprozess: er hält Taste und Maustaste und wartet lange. Ohne die
// Umgebungsvariable tut er nichts.
func TestKindFernbedienungHaelt(t *testing.T) {
	if os.Getenv(kindFBUmgebung) == "" {
		return
	}
	befehle, err := ParseDrehbuch("taste-halten A\ndruecke 5 6\nwarte 60000\n")
	if err != nil {
		t.Fatal(err)
	}
	a := &abspieler{e: &schreibEingabe{}, abbruch: make(chan struct{})}
	a.signalwache(os.Exit, os.Stderr)
	os.Stderr.WriteString("KIND-BEREIT\n")
	os.Exit(abspielen(a, befehle, 0, nil, io.Discard, os.Stderr))
}

// schreibEingabe meldet jedes Ereignis auf stderr.
type schreibEingabe struct{}

func (schreibEingabe) Zeiger(x, y int) error { return nil }
func (schreibEingabe) Maus(knopf, zustand uint32) error {
	os.Stderr.WriteString("EREIGNIS maus " + map[uint32]string{gedrueckt: "an", los: "los"}[zustand] + "\n")
	return nil
}
func (schreibEingabe) Rad(w float64) error { return nil }
func (schreibEingabe) Tastatur(code, zustand uint32) error {
	os.Stderr.WriteString("EREIGNIS taste " + map[uint32]string{gedrueckt: "an", los: "los"}[zustand] + "\n")
	return nil
}
func (schreibEingabe) Keysym(sym, zustand uint32) error { return nil }
func (schreibEingabe) Fassung() uint32                  { return 6 }

// TestSignalLoestGehaltenes: ein echtes Signal an den Kindprozess löst Maus und Tasten und endet mit Exit 1.
func TestSignalLoestGehaltenes(t *testing.T) {
	if os.Getenv(kindFBUmgebung) != "" {
		return
	}
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestKindFernbedienungHaelt$", "-test.count=1")
			cmd.Env = append(os.Environ(), kindFBUmgebung+"=1")
			fehler, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { cmd.Process.Kill() }) // nur der eigene Kindprozess
			zeilen := make(chan string, 64)
			go func() {
				s := bufio.NewScanner(fehler)
				for s.Scan() {
					zeilen <- s.Text()
				}
				close(zeilen)
			}()
			// Warten, bis beide Tasten gehalten werden (zwei „an“-Ereignisse nach „bereit“).
			an := 0
			frist := time.After(30 * time.Second)
			for an < 2 {
				select {
				case z, ok := <-zeilen:
					if !ok {
						t.Fatal("Kind endete vor dem Halten")
					}
					if strings.HasSuffix(z, " an") {
						an++
					}
				case <-frist:
					t.Fatal("Kind hält nicht")
				}
			}
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			var los int
			for z := range zeilen {
				if strings.HasSuffix(z, " los") {
					los++
				}
			}
			err = cmd.Wait()
			ee, ok := err.(*exec.ExitError)
			if !ok || ee.ExitCode() != 1 {
				t.Errorf("Ende %v, erwartet Exit 1", err)
			}
			if los != 2 {
				t.Errorf("%d Loslassen-Ereignisse, erwartet 2 (Maus und Taste)", los)
			}
		})
	}
}

// TestWacheMitAttrappe: ein Signal im Kanal löst Gehaltenes, ruft beende(1) und unterbricht „warte“.
func TestWacheMitAttrappe(t *testing.T) {
	r := &aufzeichner{}
	a := &abspieler{e: r, abbruch: make(chan struct{})}
	if err := a.tasteHalten(Kombi{Taste: 30}); err != nil {
		t.Fatal(err)
	}
	kanal := make(chan os.Signal, 1)
	ende := make(chan int, 1)
	var sb strings.Builder
	halt := a.wacheAuf(kanal, func() {}, func(c int) { ende <- c }, &sb)
	defer halt()
	kanal <- syscall.SIGTERM
	select {
	case c := <-ende:
		if c != 1 {
			t.Errorf("Exit %d, erwartet 1", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wache beendet nicht")
	}
	a.mu.Lock() // beende läuft unter der Sperre; sie wird danach frei
	a.mu.Unlock()
	if len(r.ereignisse) != 2 || r.ereignisse[1] != "taste 30 0" {
		t.Errorf("Ereignisse %v, erwartet Drücken und Loslassen", r.ereignisse)
	}
}
