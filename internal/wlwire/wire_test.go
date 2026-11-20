// SPDX-License-Identifier: GPL-3.0-or-later

package wlwire

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

// paar liefert zwei verbundene Enden: Client und Server (der nachgebaute Compositor).
func paar(t *testing.T) (client, server *Verbindung) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	ende := func(fd int) *net.UnixConn {
		f := os.NewFile(uintptr(fd), "paar")
		defer f.Close()
		c, err := net.FileConn(f)
		if err != nil {
			t.Fatal(err)
		}
		return c.(*net.UnixConn)
	}
	client, server = Neue(ende(fds[0])), Neue(ende(fds[1]))
	t.Cleanup(func() { client.Close(); server.Close() })
	return client, server
}

// TestSocketPfad: WAYLAND_DISPLAY absolut, relativ zu XDG_RUNTIME_DIR, Voreinstellung wayland-0, Fehler ohne Ordner.
func TestSocketPfad(t *testing.T) {
	for _, fall := range []struct {
		display, run, want string
		fehler             bool
	}{
		{"/x/sock", "", "/x/sock", false},
		{"wayland-1", "/run/u", "/run/u/wayland-1", false},
		{"", "/run/u", "/run/u/wayland-0", false},
		{"wayland-1", "", "", true},
	} {
		env := map[string]string{"WAYLAND_DISPLAY": fall.display, "XDG_RUNTIME_DIR": fall.run}
		got, err := SocketPfad(func(k string) string { return env[k] })
		if (err != nil) != fall.fehler || got != fall.want {
			t.Errorf("%q/%q: %q, %v; erwartet %q, Fehler %v", fall.display, fall.run, got, err, fall.want, fall.fehler)
		}
	}
}

// TestLeserKurz: abgeschnittene Argumente sind Fehler, keine Panik.
func TestLeserKurz(t *testing.T) {
	l := Leser{B: []byte{9, 0, 0, 0, 'a', 'b'}}
	if l.Str(); l.Err == nil {
		t.Error("abgeschnittene Zeichenkette ohne Fehler")
	}
	l = Leser{B: []byte{1, 2}}
	if l.U32(); l.Err == nil {
		t.Error("abgeschnittenes Wort ohne Fehler")
	}
}

// TestRundlaufUndFixed: Nachrichten (mit Zeichenkette, Festkomma und Dateideskriptor) kommen unverändert an, der
// Rundlauf endet mit der Callback-Antwort, ein wl_display.error wird zum Fehler, Schließen ist ErrVerbindungZu.
func TestRundlaufUndFixed(t *testing.T) {
	c, s := paar(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if err := c.Sende(7, 3, new(Nachricht).Str("hallo").Fixed(-1.5).Fixed(100.25).FD(int(w.Fd()))); err != nil {
		t.Fatal(err)
	}
	s.SetReadDeadline(time.Now().Add(5 * time.Second))
	e, err := s.Lies()
	if err != nil || e.Obj != 7 || e.Op != 3 {
		t.Fatalf("Lies: %+v, %v", e, err)
	}
	l := Leser{B: e.Args}
	if str, a, b := l.Str(), int32(l.U32()), int32(l.U32()); str != "hallo" || a != -384 || b != 25664 || l.Err != nil {
		t.Errorf("Argumente: %q %d %d %v", str, a, b, l.Err)
	}
	if fd, err := s.NimmFD(); err != nil {
		t.Errorf("NimmFD: %v", err)
	} else {
		syscall.Close(fd)
	}

	go func() {
		s.SetReadDeadline(time.Now().Add(5 * time.Second))
		e, err := s.Lies()
		if err != nil || e.Obj != IDDisplay || e.Op != OpDisplaySync {
			return
		}
		cb := Leser{B: e.Args}
		id := cb.U32()
		s.Sende(99, 0, nil) // fremdes Ereignis vor der Antwort
		s.Sende(id, EvCallbackDone, new(Nachricht).U32(1))
	}()
	gesehen := 0
	if err := c.Rundlauf(5*time.Second, func(e Ereignis) error { gesehen++; return nil }); err != nil || gesehen != 1 {
		t.Errorf("Rundlauf: %v, %d Ereignisse", err, gesehen)
	}

	go func() {
		s.SetReadDeadline(time.Now().Add(5 * time.Second))
		s.Lies()
		s.Sende(IDDisplay, EvDisplayError, new(Nachricht).U32(5).U32(1).Str("kaputt"))
	}()
	if err := c.Rundlauf(5*time.Second, func(Ereignis) error { return nil }); err == nil {
		t.Error("wl_display.error ohne Fehler")
	}

	s.Close()
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Lies(); !errors.Is(err, ErrVerbindungZu) {
		t.Errorf("nach Schließen: %v", err)
	}
}
