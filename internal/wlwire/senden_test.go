// SPDX-License-Identifier: GPL-3.0-or-later

package wlwire

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSendeTeilschreiben: schreibt WriteMsgUnix nur einen Teil (F-207), schickt Sende den Rest nach; Nachricht und
// Dateideskriptor kommen vollständig an, die Folgenachricht bleibt an der richtigen Stelle.
func TestSendeTeilschreiben(t *testing.T) {
	c, s := paar(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	aufrufe := 0
	c.sendeMsg = func(b, oob []byte) (int, error) {
		aufrufe++
		n, _, err := c.c.WriteMsgUnix(b[:5], oob, nil) // nur 5 Byte, mit dem Deskriptor
		return n, err
	}
	if err := c.Sende(7, 3, new(Nachricht).Str("hallo welt").FD(int(w.Fd()))); err != nil {
		t.Fatalf("Sende: %v", err)
	}
	c.sendeMsg = nil
	if err := c.Sende(8, 1, nil); err != nil {
		t.Fatal(err)
	}
	s.SetReadDeadline(time.Now().Add(5 * time.Second))
	e, err := s.Lies()
	if err != nil || e.Obj != 7 || e.Op != 3 {
		t.Fatalf("Lies: %+v, %v", e, err)
	}
	if l := (Leser{B: e.Args}); l.Str() != "hallo welt" {
		t.Errorf("Argumente %q", e.Args)
	}
	if fd, err := s.NimmFD(); err != nil {
		t.Errorf("NimmFD: %v", err)
	} else {
		syscall.Close(fd)
	}
	if e, err := s.Lies(); err != nil || e.Obj != 8 || e.Op != 1 {
		t.Errorf("Folgenachricht: %+v, %v", e, err)
	}
	if aufrufe != 1 {
		t.Errorf("%d Sendeaufrufe", aufrufe)
	}
}

// TestSendeTeilschreibenFehler: scheitert der Rest (Verbindung zu), meldet Sende den Fehler samt Byte-Zahlen.
func TestSendeTeilschreibenFehler(t *testing.T) {
	c, _ := paar(t)
	c.sendeMsg = func(b, oob []byte) (int, error) {
		c.c.Close() // der Rest kann nicht mehr gehen
		return 3, nil
	}
	err := c.Sende(7, 3, new(Nachricht).U32(1))
	if err == nil || !strings.Contains(err.Error(), "nur teilweise gesendet (3 von 12 Byte)") {
		t.Errorf("Fehler %v", err)
	}
}
