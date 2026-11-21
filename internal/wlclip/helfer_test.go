// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"bytes"
	"errors"
	"fmt"
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
		// Ablösung: eigene Sitzung, stdout und stderr auf /dev/null, fd 3 eine Pipe (Kritik Fable, Befund 4)
		var abgeloest []string
		if sid, _, _ := syscall.RawSyscall(syscall.SYS_GETSID, 0, 0, 0); int(sid) == os.Getpid() {
			abgeloest = append(abgeloest, "sitzung")
		}
		var null syscall.Stat_t
		syscall.Stat("/dev/null", &null)
		for _, fd := range []int{1, 2} {
			var st syscall.Stat_t
			if syscall.Fstat(fd, &st) == nil && st.Rdev == null.Rdev && st.Mode&syscall.S_IFMT == syscall.S_IFCHR {
				abgeloest = append(abgeloest, "null"+strconv.Itoa(fd))
			}
		}
		if istPipe(3) {
			abgeloest = append(abgeloest, "pipe3")
		}
		os.WriteFile(filepath.Join(ordner, "abgeloest"), []byte(strings.Join(abgeloest, " ")), 0o600)
		io.WriteString(meldung, "ok "+ManagerExt+"\n")
		meldung.Close()
		return 0
	case "geerbt":
		// F-189: wie HelferLauf schließt die Attrappe geerbte Deskriptoren ab 4 ohne CLOEXEC; gezählt wird je
		// vorher/nachher, dazu die Zahl der Deskriptoren mit CLOEXEC (Laufzeit), die bleiben müssen.
		zaehle := func() (ohne, mit int) {
			for _, fd := range offeneDeskriptoren() {
				if fd < 4 {
					continue
				}
				fl, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
				switch {
				case errno != 0:
				case fl&syscall.FD_CLOEXEC != 0:
					mit++
				default:
					ohne++
				}
			}
			return
		}
		o1, m1 := zaehle()
		zu := schliesseGeerbte()
		o2, m2 := zaehle()
		os.WriteFile(filepath.Join(ordner, "fds"), []byte(fmt.Sprintf("%d %d %d %d %d", o1, o2, m1, m2, len(zu))), 0o600)
		io.WriteString(meldung, "ok "+ManagerExt+"\n")
		meldung.Close()
		return 0
	case "fehler":
		io.WriteString(meldung, "fehler: kaputt\n")
		return 1
	case "keindc":
		io.WriteString(meldung, "fehler: "+ErrKeinDataControl.Error()+"\n")
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
	if b, _ := os.ReadFile(filepath.Join(ordner, "abgeloest")); string(b) != "sitzung null1 null2 pipe3" {
		t.Errorf("Ablösung %q, erwartet „sitzung null1 null2 pipe3“", b)
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
	ordner, _, _, err := uebergibMit(t, "stumm", 2*time.Second)
	if err == nil {
		t.Fatal("kein Fehler bei stummem Halter")
	}
	if d := time.Since(start); d > 8*time.Second {
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

// TestIstPipe: nur eine offene Pipe gilt als Meldeweg; ein Socket oder eine Datei nicht (Befund 1).
func TestIstPipe(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	if !istPipe(int(w.Fd())) {
		t.Error("Pipe nicht erkannt")
	}
	fds, _ := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	defer syscall.Close(fds[0])
	defer syscall.Close(fds[1])
	if istPipe(fds[0]) {
		t.Error("Socket als Pipe erkannt")
	}
	f, _ := os.Create(filepath.Join(t.TempDir(), "datei"))
	defer f.Close()
	if istPipe(int(f.Fd())) {
		t.Error("Datei als Pipe erkannt")
	}
}

// TestUebergibKeinDataControl: meldet der Halter, dass der Compositor kein data-control bietet, liefert Uebergib
// einen Fehler, für den errors.Is(err, ErrKeinDataControl) gilt (der Editor merkt sich das und startet keinen Halter mehr).
func TestUebergibKeinDataControl(t *testing.T) {
	_, _, _, err := uebergibMit(t, "keindc", 10*time.Second)
	if !errors.Is(err, ErrKeinDataControl) {
		t.Errorf("Fehler %v, erwartet ErrKeinDataControl", err)
	}
	if _, _, _, err := uebergibMit(t, "fehler", 10*time.Second); errors.Is(err, ErrKeinDataControl) {
		t.Errorf("anderer Fehler %v gilt als ErrKeinDataControl", err)
	}
}

// TestHalterMoeglich: mit geerbtem WAYLAND_SOCKET ist kein Halter-Prozess möglich.
func TestHalterMoeglich(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if err := HalterMoeglich(env(map[string]string{"WAYLAND_SOCKET": "7"})); !errors.Is(err, ErrWaylandSocket) {
		t.Errorf("Fehler %v, erwartet ErrWaylandSocket", err)
	}
	if err := HalterMoeglich(env(map[string]string{"WAYLAND_DISPLAY": "wayland-0"})); err != nil {
		t.Errorf("Fehler %v, erwartet nil", err)
	}
}

// TestUebergibSchliesstGeerbte (F-189): Ein im Editor ohne CLOEXEC offener Deskriptor (hier fd >= 4 per
// syscall.Open ohne O_CLOEXEC, wie `schnapp 5>datei`) kommt im Halter an und wird dort geschlossen; Deskriptoren mit
// CLOEXEC (Laufzeit) bleiben.
func TestUebergibSchliesstGeerbte(t *testing.T) {
	var fds [3]int
	for i := range fds {
		fd, err := syscall.Open(filepath.Join(t.TempDir(), "geerbt"), syscall.O_CREAT|syscall.O_RDWR, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		fds[i] = fd
		defer syscall.Close(fd)
	}
	ordner, _, _, err := uebergibMit(t, "geerbt", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(ordner, "fds"))
	var o1, o2, m1, m2, zu int
	if _, err := fmt.Sscanf(string(b), "%d %d %d %d %d", &o1, &o2, &m1, &m2, &zu); err != nil {
		t.Fatalf("Zählung %q: %v", b, err)
	}
	if o1 < len(fds) {
		t.Errorf("im Kind vorher %d Deskriptoren ohne CLOEXEC, erwartet mindestens %d (Test erreicht die Vererbung nicht)", o1, len(fds))
	}
	if o2 != 0 || zu != o1 {
		t.Errorf("nachher %d offen ohne CLOEXEC, %d geschlossen (vorher %d), erwartet 0 und %d", o2, zu, o1, o1)
	}
	if m1 != m2 {
		t.Errorf("Deskriptoren mit CLOEXEC: vorher %d, nachher %d, sie müssen bleiben", m1, m2)
	}
}

// TestSchliesseOhneCloexecZaehlt prüft die Zählung ohne Halter-Kind: Ein Deskriptor ohne CLOEXEC wird geschlossen und
// gemeldet, einer mit CLOEXEC bleibt offen und fehlt in der Rückgabe.
func TestSchliesseOhneCloexecZaehlt(t *testing.T) {
	// F_DUPFD setzt CLOEXEC nicht; ab 10 liegt die Nummer sicher über der Grenze 4.
	var p [2]int
	if err := syscall.Pipe2(p[:], 0); err != nil {
		t.Fatal(err)
	}
	ohne, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(p[0]), syscall.F_DUPFD, 10)
	syscall.Close(p[0])
	defer syscall.Close(p[1])
	if errno != 0 {
		t.Fatal(errno)
	}
	fd := int(ohne)

	var q [2]int
	if err := syscall.Pipe2(q[:], syscall.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(q[0])
	defer syscall.Close(q[1])

	zu := schliesseOhneCloexec([]int{fd, q[0], q[1]})
	if len(zu) != 1 || zu[0] != fd {
		t.Fatalf("geschlossen %v, erwartet [%d]", zu, fd)
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno != syscall.EBADF {
		t.Errorf("Deskriptor %d nach dem Schließen: Fehler %v, erwartet EBADF", fd, errno)
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(q[0]), syscall.F_GETFD, 0); errno != 0 {
		t.Errorf("Deskriptor mit CLOEXEC wurde geschlossen: %v", errno)
	}
}
