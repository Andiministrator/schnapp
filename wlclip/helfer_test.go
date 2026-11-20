// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMain: Mit HelferArgument startet das Testprogramm als Attrappe des Halters (Verhalten aus WLCLIP_ATTRAPPE), damit
// Uebergib ohne Compositor geprüft werden kann.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == HelferArgument {
		os.Exit(attrappe(os.Getenv("WLCLIP_ATTRAPPE"), os.Getenv("WLCLIP_ORDNER")))
	}
	os.Exit(m.Run())
}

func attrappe(art, ordner string) int {
	meldung := os.NewFile(3, "meldung")
	os.WriteFile(filepath.Join(ordner, "pid"), []byte(strconv.Itoa(os.Getpid())), 0o600)
	switch art {
	case "ok":
		daten, _ := io.ReadAll(os.Stdin)
		os.WriteFile(filepath.Join(ordner, "daten"), daten, 0o600)
		wd, _ := os.Getwd()
		os.WriteFile(filepath.Join(ordner, "ordner"), []byte(wd), 0o600)
		io.WriteString(meldung, "ok "+ManagerExt+"\n")
		meldung.Close()
		return 0
	case "fehler":
		io.WriteString(meldung, "fehler: kaputt\n")
		return 1
	case "stumm":
		time.Sleep(30 * time.Second)
		return 0
	}
	return 0 // ohne Meldung beendet
}

func uebergibMit(t *testing.T, art string, frist time.Duration) (string, int, string, error) {
	t.Helper()
	ordner := t.TempDir()
	t.Setenv("WLCLIP_ATTRAPPE", art)
	t.Setenv("WLCLIP_ORDNER", ordner)
	programm, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pid, manager, err := Uebergib(programm, []byte("png-bytes"), frist)
	return ordner, pid, manager, err
}

// TestUebergibOK: Daten kommen über die Standardeingabe an, Arbeitsordner ist /, PID und Manager kommen zurück.
func TestUebergibOK(t *testing.T) {
	ordner, pid, manager, err := uebergibMit(t, "ok", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if manager != ManagerExt {
		t.Errorf("Manager %q", manager)
	}
	if b, _ := os.ReadFile(filepath.Join(ordner, "pid")); string(b) != strconv.Itoa(pid) {
		t.Errorf("PID %d, Attrappe meldet %s", pid, b)
	}
	if b, _ := os.ReadFile(filepath.Join(ordner, "daten")); string(b) != "png-bytes" {
		t.Errorf("Daten %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(ordner, "ordner")); string(b) != "/" {
		t.Errorf("Arbeitsordner %q, erwartet /", b)
	}
}

// TestUebergibFehler: Fehlertext des Halters und fehlende Meldung werden zum Fehler.
func TestUebergibFehler(t *testing.T) {
	if _, _, _, err := uebergibMit(t, "fehler", 10*time.Second); err == nil || err.Error() != "kaputt" {
		t.Errorf("Fehler %v, erwartet „kaputt“", err)
	}
	if _, _, _, err := uebergibMit(t, "leer", 10*time.Second); err == nil || !strings.Contains(err.Error(), "ohne Meldung") {
		t.Errorf("Fehler %v, erwartet „ohne Meldung“", err)
	}
}

// TestUebergibStumm: meldet sich der Halter nicht, beendet Uebergib nach der Frist das eigene Kind.
func TestUebergibStumm(t *testing.T) {
	start := time.Now()
	ordner, _, _, err := uebergibMit(t, "stumm", 300*time.Millisecond)
	if err == nil {
		t.Fatal("kein Fehler bei stummem Halter")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Uebergib kehrte nach %v zurück", d)
	}
	b, _ := os.ReadFile(filepath.Join(ordner, "pid"))
	pid, perr := strconv.Atoi(string(b))
	if perr != nil {
		t.Fatalf("PID der Attrappe fehlt: %q", b)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("Kind %d läuft noch oder ist nicht abgeholt (%v)", pid, err)
	}
}

// meldePuffer sammelt die Meldung des Helfers.
type meldePuffer struct {
	bytes.Buffer
	zu int
}

func (m *meldePuffer) Close() error { m.zu++; return nil }

// TestHelferOhneCompositor: ohne erreichbaren Compositor meldet der Helfer einen Fehler (genau einmal) und endet mit 1.
func TestHelferOhneCompositor(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "nicht-da")
	var m meldePuffer
	if code := Helfer(strings.NewReader("bild"), &m, time.Second); code != 1 {
		t.Errorf("Exit %d, erwartet 1", code)
	}
	if !strings.HasPrefix(m.String(), "fehler: ") || m.zu != 1 || strings.Count(m.String(), "\n") != 1 {
		t.Errorf("Meldung %q, %d-mal geschlossen", m.String(), m.zu)
	}
}

// TestHelferLeereEingabe: ohne Daten setzt der Helfer nichts.
func TestHelferLeereEingabe(t *testing.T) {
	var m meldePuffer
	if code := Helfer(strings.NewReader(""), &m, time.Second); code != 1 || !strings.HasPrefix(m.String(), "fehler: ") {
		t.Errorf("Exit %d, Meldung %q", code, m.String())
	}
}
