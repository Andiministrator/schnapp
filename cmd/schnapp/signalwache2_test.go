// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-441, F-442, F-443: Signal nach dem Schreiben, zweites Signal bei hängendem Schreiben, geerbtes „ignoriert“.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// standardSignal setzt s auf die Voreinstellung zurück und löscht das Ignoriert-Merkmal: signal.Reset allein lässt
// ein mit signal.Ignore gesetztes Merkmal stehen, Notify und Stop löschen es.
func standardSignal(s os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, s)
	signal.Stop(c)
}

// fangbar stellt sicher, dass sigs nicht ignoriert sind (läuft der Test unter nohup oder in einem Hintergrundauftrag,
// erbt er ignoriertes SIGHUP; F-443), und stellt den Zustand nach dem Test wieder her.
func fangbar(t *testing.T, sigs ...syscall.Signal) {
	t.Helper()
	for _, s := range sigs {
		if signal.Ignored(s) {
			standardSignal(s)
			t.Cleanup(func() { signal.Ignore(s) })
		}
	}
}

// setzeNaht ersetzt die Schreibnaht für die Dauer des Tests.
func setzeNaht(t *testing.T, f func(w io.Writer, inhalt []byte, abgebrochen func() bool) error) {
	t.Helper()
	alt := schreibNaht
	schreibNaht = f
	t.Cleanup(func() { schreibNaht = alt })
}

// wartePunkt wartet höchstens fünf Sekunden auf bedingung.
func wartePunkt(bedingung func() bool) bool {
	for stop := time.Now().Add(5 * time.Second); time.Now().Before(stop); time.Sleep(time.Millisecond) {
		if bedingung() {
			return true
		}
	}
	return bedingung()
}

// F-441: Kommt das Signal nach dem Write, ist das Ziel vollständig; der Fehler sagt das (errNachSchreiben, nicht
// errAbgebrochen), und es bleibt keine temporäre Datei.
func TestSignalwacheSignalNachSchreiben(t *testing.T) {
	fangbar(t, syscall.SIGHUP)
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		dir := t.TempDir()
		ziel := filepath.Join(dir, "aus.png")
		setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
			if _, err := w.Write(inhalt); err != nil {
				return err
			}
			if err := syscall.Kill(os.Getpid(), sig); err != nil {
				t.Error(err)
			}
			if !wartePunkt(abgebrochen) {
				t.Error("Signal kam nicht an")
			}
			return nil
		})
		err := schreibeInhalt(ziel, []byte("inhalt"), true, nil)
		if !errors.Is(err, errNachSchreiben) || errors.Is(err, errAbgebrochen) {
			t.Fatalf("%v: Fehler %v, Soll errNachSchreiben", sig, err)
		}
		if b, err := os.ReadFile(ziel); err != nil || string(b) != "inhalt" {
			t.Fatalf("%v: Ziel %q, %v", sig, b, err)
		}
		if namen := ordnerInhalt(t, dir); len(namen) != 1 {
			t.Fatalf("%v: Zielordner: %v", sig, namen)
		}
	}
}

// F-441 am Aufruf: Bei --export mit Signal nach dem Schreiben meldet Schnapp „vollständig geschrieben“, gibt die
// Ergebniszeile aus (der Agent sieht den wahren Stand) und endet mit Exit 1.
func TestExportSignalNachSchreiben(t *testing.T) {
	fangbar(t, syscall.SIGHUP)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "q.png")
	d, err := document.Neu(testbild(20, 10, 3))
	if err != nil {
		t.Fatal(err)
	}
	var a bytes.Buffer
	if err := fileformat.SchreibeArbeitsdatei(&a, d, testbild(20, 10, 3), fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, a.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		if _, err := w.Write(inhalt); err != nil {
			return err
		}
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Error(err)
		}
		wartePunkt(abgebrochen)
		return nil
	})
	ziel := filepath.Join(dir, "aus.png")
	code, aus, fehler := fuehre(t, keineMonitore, "--export="+ziel, "--json", quelle)
	if code != exitFehler || !strings.Contains(fehler, "written completely") || strings.Contains(fehler, "Could not write") ||
		!strings.Contains(aus, `"path"`) {
		t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Fatalf("Ziel %v, %v", st, err)
	}
}

// F-441: Die Meldung sagt „vollständig geschrieben“, nicht „could not be written“.
func TestMeldeSchreibFehlerNachSchreiben(t *testing.T) {
	var b bytes.Buffer
	meldeSchreibFehler(&b, "x.png", errNachSchreiben)
	if s := b.String(); !strings.Contains(s, "written completely") || strings.Contains(s, "could not") {
		t.Errorf("Meldung %q", s)
	}
	b.Reset()
	meldeSchreibFehler(&b, "x.png", errAbgebrochen)
	if s := b.String(); !strings.Contains(s, "Could not write x.png") {
		t.Errorf("Meldung %q", s)
	}
}

// F-442: Hängt das Schreiben, beendet ein zweites Signal den Aufruf: temporäre Datei weg, Exit 1. Das erste Signal
// allein beendet nichts.
func TestSignalwacheZweitesSignalBeiHaengendemSchreiben(t *testing.T) {
	fangbar(t, syscall.SIGTERM)
	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	drin := make(chan struct{})
	frei := make(chan struct{})
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		close(drin)
		<-frei
		return errors.New("test: Schreiben hing")
	})
	var mu sync.Mutex
	var code = -1
	gemeldet := make(chan struct{})
	alt := beendeProzess
	beendeProzess = func(c int) {
		mu.Lock()
		code = c
		mu.Unlock()
		close(gemeldet)
	}
	t.Cleanup(func() { beendeProzess = alt })

	fertig := make(chan error, 1)
	go func() { fertig <- schreibeInhalt(ziel, []byte("inhalt"), true, nil) }()
	<-drin
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-gemeldet:
		t.Fatal("ein einziges Signal beendete den Prozess")
	case <-time.After(100 * time.Millisecond):
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-gemeldet:
	case <-time.After(5 * time.Second):
		t.Fatal("das zweite Signal beendete den Prozess nicht")
	}
	mu.Lock()
	if code != exitFehler {
		t.Errorf("Exit-Code %d, Soll %d", code, exitFehler)
	}
	mu.Unlock()
	for _, n := range ordnerInhalt(t, dir) {
		if strings.HasPrefix(n, tempVorsatz) {
			t.Errorf("temporäre Datei %s blieb liegen", n)
		}
	}
	close(frei)
	if err := <-fertig; err == nil {
		t.Error("hängendes Schreiben ohne Fehler beendet")
	}
	if _, err := os.Stat(ziel); err == nil {
		t.Error("Ziel entstand")
	}
}

// F-443: Ein beim Start ignoriertes Signal (nohup) meldet die Wache nicht an; der Aufruf gelingt trotz SIGHUP.
func TestSignalwacheLaesstIgnoriertesInRuhe(t *testing.T) {
	war := signal.Ignored(syscall.SIGHUP)
	signal.Ignore(syscall.SIGHUP)
	t.Cleanup(func() {
		if war {
			signal.Ignore(syscall.SIGHUP)
		} else {
			standardSignal(syscall.SIGHUP)
		}
	})
	var angemeldet []os.Signal
	altAn := meldeAn
	meldeAn = func(c chan<- os.Signal, sigs ...os.Signal) {
		angemeldet = append(angemeldet, sigs...)
		altAn(c, sigs...)
	}
	t.Cleanup(func() { meldeAn = altAn })

	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	err := schreibeInhalt(ziel, []byte("inhalt"), true, func(abgebrochen func() bool) {
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Error(err)
		}
		time.Sleep(50 * time.Millisecond)
	})
	if err != nil {
		t.Fatalf("Fehler %v, Soll nil (SIGHUP war ignoriert)", err)
	}
	for _, s := range angemeldet {
		if s == syscall.SIGHUP {
			t.Error("SIGHUP trotz Ignorieren angemeldet")
		}
	}
	if len(angemeldet) != 2 {
		t.Errorf("angemeldet: %v, Soll SIGTERM und SIGINT", angemeldet)
	}
	if !signal.Ignored(syscall.SIGHUP) {
		t.Error("SIGHUP nicht mehr ignoriert")
	}
}

// Die Wache meldet ihren Kanal nach dem Schreiben genau einmal ab (sonst finge sie Signale des späteren Programms).
func TestSignalwacheMeldetSichAb(t *testing.T) {
	altAb := meldeAb
	var abgemeldet int
	meldeAb = func(c chan<- os.Signal) {
		abgemeldet++
		altAb(c)
	}
	t.Cleanup(func() { meldeAb = altAb })
	if err := schreibeInhalt(filepath.Join(t.TempDir(), "aus.png"), []byte("x"), true, nil); err != nil {
		t.Fatal(err)
	}
	if abgemeldet != 1 {
		t.Errorf("Abmeldungen: %d, Soll 1", abgemeldet)
	}
}

// Ein Signal, das erst beim Abmelden im Kanal liegt, zählt noch (das Ziel ist dann vollständig, F-441).
func TestSignalwacheSpätesSignalZaehlt(t *testing.T) {
	altAb := meldeAb
	meldeAb = func(c chan<- os.Signal) {
		altAb(c)
		c <- syscall.SIGTERM
	}
	t.Cleanup(func() { meldeAb = altAb })
	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	if err := schreibeInhalt(ziel, []byte("x"), true, nil); !errors.Is(err, errNachSchreiben) {
		t.Fatalf("Fehler %v, Soll errNachSchreiben", err)
	}
	if b, err := os.ReadFile(ziel); err != nil || string(b) != "x" {
		t.Fatalf("Ziel %q, %v", b, err)
	}
}
