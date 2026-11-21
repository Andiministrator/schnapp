// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/godbus/dbus/v5"
)

// F-56: Die Tests dieser Datei fahren dbusTransport.capture auf Deskriptor-Ebene. Eine Attrappe ersetzt nur den
// D-Bus-Aufruf (Feld rufe) und liefert statt KWin echte Deskriptoren (Pipe, Datei als memfd-Ersatz); kein Bus,
// keine Aufnahme.

// fdOffen fragt per fstat, ob die Nummer noch einen Deskriptor bezeichnet (nur eigene Mittel, nichts aus /proc).
func fdOffen(fd int) bool {
	var st syscall.Stat_t
	return syscall.Fstat(fd, &st) == nil
}

// memfdErsatz legt eine Datei mit dem Inhalt an und gibt einen eigenen Deskriptor darauf zurück (der Aufrufer
// des Transports ist für ihn verantwortlich, wie bei einem von KWin gelieferten memfd).
func memfdErsatz(t *testing.T, inhalt []byte) int {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "memfd")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(inhalt); err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Dup(int(f.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	return fd
}

func rohAntwort() map[string]dbus.Variant {
	return map[string]dbus.Variant{"type": dbus.MakeVariant("raw")}
}

func TestCapturePipeFD(t *testing.T) {
	ctx := context.Background()
	s := schnittstelle2

	t.Run("Daten kommen an, Schreibende wird geschlossen", func(t *testing.T) {
		var wfd int
		tr := &dbusTransport{rufe: func(_ context.Context, _ schnittstelle, _ string, aufruf []any) ([]any, error) {
			u, ok := aufruf[len(aufruf)-1].(dbus.UnixFD)
			if !ok {
				t.Fatalf("letzter Parameter ist %T, nicht UnixFD", aufruf[len(aufruf)-1])
			}
			wfd = int(u)
			if _, err := syscall.Write(wfd, []byte("0123456789")); err != nil {
				t.Fatal(err)
			}
			return []any{rohAntwort()}, nil
		}}
		antwort, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		if antwort["type"].Value() != "raw" {
			t.Errorf("Antwort: %v", antwort)
		}
		if d, err := q.lies(4); err != nil || string(d) != "0123" {
			t.Errorf("lies(4) = %q, %v", d, err)
		}
		if fdOffen(wfd) {
			t.Error("Schreibende nach dem Aufruf noch offen")
		}
	})

	t.Run("zu kurz", func(t *testing.T) {
		tr := &dbusTransport{rufe: func(_ context.Context, _ schnittstelle, _ string, aufruf []any) ([]any, error) {
			syscall.Write(int(aufruf[len(aufruf)-1].(dbus.UnixFD)), []byte("abc"))
			return []any{rohAntwort()}, nil
		}}
		_, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		if _, err := q.lies(10); err == nil || !strings.Contains(err.Error(), "Pipe lieferte 3 Bytes") {
			t.Errorf("zu kurz: %v", err)
		}
	})

	t.Run("zu lang", func(t *testing.T) {
		alt := pipeGrenze
		pipeGrenze = 8
		defer func() { pipeGrenze = alt }()
		tr := &dbusTransport{rufe: func(_ context.Context, _ schnittstelle, _ string, aufruf []any) ([]any, error) {
			syscall.Write(int(aufruf[len(aufruf)-1].(dbus.UnixFD)), bytes.Repeat([]byte("x"), 20))
			return []any{rohAntwort()}, nil
		}}
		_, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		if _, err := q.lies(4); err == nil || !strings.Contains(err.Error(), "mehr als 8 Bytes") {
			t.Errorf("zu lang: %v", err)
		}
	})

	t.Run("Aufruffehler schließt das Schreibende", func(t *testing.T) {
		var wfd int
		tr := &dbusTransport{rufe: func(_ context.Context, _ schnittstelle, _ string, aufruf []any) ([]any, error) {
			wfd = int(aufruf[len(aufruf)-1].(dbus.UnixFD))
			return nil, errors.New("KWin lehnt ab")
		}}
		_, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err == nil || q != nil {
			t.Fatalf("Fehler erwartet: %v, %v", q, err)
		}
		if fdOffen(wfd) {
			t.Error("Schreibende nach Fehler noch offen")
		}
	})
}

func TestCaptureMemfdFD(t *testing.T) {
	ctx := context.Background()
	s := schnittstelle3

	t.Run("Daten kommen an, Close schließt", func(t *testing.T) {
		fd := memfdErsatz(t, []byte("0123456789"))
		tr := &dbusTransport{rufe: func(context.Context, schnittstelle, string, []any) ([]any, error) {
			return []any{dbus.UnixFD(fd), rohAntwort()}, nil
		}}
		antwort, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if antwort["type"].Value() != "raw" {
			t.Errorf("Antwort: %v", antwort)
		}
		if d, err := q.lies(10); err != nil || string(d) != "0123456789" {
			t.Errorf("lies(10) = %q, %v", d, err)
		}
		if _, err := q.lies(11); err == nil || !strings.Contains(err.Error(), "memfd hat 10 Bytes") {
			t.Errorf("zu kurz: %v", err)
		}
		if !fdOffen(fd) {
			t.Fatal("Deskriptor vor Close schon zu")
		}
		q.Close()
		if fdOffen(fd) {
			t.Error("Deskriptor nach Close noch offen")
		}
	})

	t.Run("Antwort unlesbar: Deskriptor wird geschlossen", func(t *testing.T) {
		fd := memfdErsatz(t, []byte("x"))
		tr := &dbusTransport{rufe: func(context.Context, schnittstelle, string, []any) ([]any, error) {
			return []any{dbus.UnixFD(fd), "keine Map"}, nil // zweiter Wert hat den falschen Typ
		}}
		_, q, err := tr.capture(ctx, s, "CaptureScreen", nil, nil)
		if err == nil || q != nil {
			t.Fatalf("Fehler erwartet: %v, %v", q, err)
		}
		if fdOffen(fd) {
			t.Error("Deskriptor nach Antwortfehler noch offen (Leck)")
		}
	})

	t.Run("ungültiger Deskriptor", func(t *testing.T) {
		tr := &dbusTransport{rufe: func(context.Context, schnittstelle, string, []any) ([]any, error) {
			return []any{dbus.UnixFD(-1), rohAntwort()}, nil
		}}
		if _, _, err := tr.capture(ctx, s, "CaptureScreen", nil, nil); err == nil {
			t.Error("fd -1 ohne Fehler")
		}
	})

	t.Run("Aufruffehler", func(t *testing.T) {
		tr := &dbusTransport{rufe: func(context.Context, schnittstelle, string, []any) ([]any, error) {
			return nil, errors.New("KWin lehnt ab")
		}}
		if _, _, err := tr.capture(ctx, s, "CaptureScreen", nil, nil); err == nil {
			t.Error("Aufruffehler ohne Fehler")
		}
	})
}
