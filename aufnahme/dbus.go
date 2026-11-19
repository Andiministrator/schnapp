// SPDX-License-Identifier: GPL-3.0-or-later

package aufnahme

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/godbus/dbus/v5"
)

// dbusTransport ist die echte Umsetzung von transport auf dem Session-Bus. Die Verbindung wird einmal gehalten
// und mit Close geschlossen.
type dbusTransport struct{ conn *dbus.Conn }

func neuerDBusTransport() (*dbusTransport, error) {
	c, err := dbus.ConnectSessionBus() // eigene Verbindung (nicht SessionBus()), deshalb schließbar
	if err != nil {
		return nil, err
	}
	return &dbusTransport{conn: c}, nil
}

func (t *dbusTransport) Close() error { return t.conn.Close() }

func (t *dbusTransport) version(ctx context.Context, s schnittstelle) (uint32, error) {
	var v dbus.Variant
	err := t.conn.Object(kwinDienst, s.pfad).
		CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, s.name, "Version").Store(&v)
	if err != nil {
		return 0, err
	}
	n, ok := v.Value().(uint32)
	if !ok {
		return 0, fmt.Errorf("Version hat Typ %T", v.Value())
	}
	return n, nil
}

func (t *dbusTransport) capture(ctx context.Context, s schnittstelle, methode string, args []any, opt map[string]dbus.Variant) (map[string]dbus.Variant, datenquelle, error) {
	obj := t.conn.Object(kwinDienst, s.pfad)
	aufruf := append(append([]any{}, args...), opt)
	var antwort map[string]dbus.Variant
	if s.memfd {
		fd := dbus.UnixFD(-1)
		if err := obj.CallWithContext(ctx, s.name+"."+methode, 0, aufruf...).Store(&fd, &antwort); err != nil {
			if fd >= 0 { // FD kam an, die übrige Antwort nicht: nicht offen liegen lassen (F-56)
				syscall.Close(int(fd))
			}
			return nil, nil, err
		}
		if fd < 0 {
			return nil, nil, fmt.Errorf("ungültiger Dateideskriptor %d", fd)
		}
		return antwort, &memfdQuelle{f: os.NewFile(uintptr(fd), "memfd")}, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	p := &pipeQuelle{r: r, fertig: make(chan gelesen, 1)}
	go func() { // KWin schreibt erst nach der Antwort; parallel lesen, sonst blockiert die Pipe
		d, err := liesBegrenzt(r, MaxBytes)
		p.fertig <- gelesen{d, err}
	}()
	aufruf = append(aufruf, dbus.UnixFD(w.Fd()))
	err = obj.CallWithContext(ctx, s.name+"."+methode, 0, aufruf...).Store(&antwort)
	w.Close() // unsere Kopie schließen, damit das Lesen mit KWins Ende EOF bekommt
	if err != nil {
		p.Close()
		return nil, nil, err
	}
	p.ctx = ctx
	return antwort, p, nil
}

type gelesen struct {
	daten []byte
	err   error
}

// pipeQuelle sammelt die Pipe-Daten von ScreenShot2.
type pipeQuelle struct {
	r      *os.File
	fertig chan gelesen
	ctx    context.Context
}

func (p *pipeQuelle) lies(n uint64) ([]byte, error) {
	select {
	case g := <-p.fertig:
		if g.err != nil {
			return nil, g.err
		}
		if uint64(len(g.daten)) < n {
			return nil, fmt.Errorf("Pipe lieferte %d Bytes, nötig %d", len(g.daten), n)
		}
		return g.daten[:n], nil
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *pipeQuelle) Close() error { return p.r.Close() } // weckt auch einen noch lesenden Goroutine-Aufruf

// memfdQuelle liest den memfd von ScreenShot3 (ReadAt, ohne mmap: ein Kopiervorgang, keine Signalgefahr bei einer
// nachträglich verkleinerten Datei).
type memfdQuelle struct{ f *os.File }

func (m *memfdQuelle) lies(n uint64) ([]byte, error) {
	st, err := m.f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() < 0 || uint64(st.Size()) < n {
		return nil, fmt.Errorf("memfd hat %d Bytes, nötig %d", st.Size(), n)
	}
	buf := make([]byte, n) // n ≤ MaxBytes, geprüft in groesseRoh
	if _, err := io.ReadFull(io.NewSectionReader(m.f, 0, int64(n)), buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (m *memfdQuelle) Close() error { return m.f.Close() }
