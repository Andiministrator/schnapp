// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

// dbusTransport ist die echte Umsetzung von transport auf dem Sitzungsbus (eigene, schließbare Verbindung).
type dbusTransport struct{ conn *dbus.Conn }

func neuerDBusTransport() (*dbusTransport, error) {
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return &dbusTransport{conn: c}, nil
}

func (t *dbusTransport) Close() error { return t.conn.Close() }

func (t *dbusTransport) Name() string { return t.conn.Names()[0] }

func (t *dbusTransport) Abonniere(pfad dbus.ObjectPath) (<-chan Antwort, func(), error) {
	sig := make(chan *dbus.Signal, 8)
	t.conn.Signal(sig)
	opt := []dbus.MatchOption{
		dbus.WithMatchObjectPath(pfad),
		dbus.WithMatchInterface(schnittstelleReq),
		dbus.WithMatchMember(signalAntwort),
	}
	if err := t.conn.AddMatchSignal(opt...); err != nil { // kehrt erst zurück, wenn der Bus die Regel kennt
		t.conn.RemoveSignal(sig)
		return nil, nil, err
	}
	aus := make(chan Antwort, 1)
	fertig := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sig:
				if s == nil || s.Path != pfad || s.Name != schnittstelleReq+"."+signalAntwort || len(s.Body) != 2 {
					continue
				}
				code, ok1 := s.Body[0].(uint32)
				erg, ok2 := s.Body[1].(map[string]dbus.Variant)
				if !ok1 || !ok2 {
					continue
				}
				select {
				case aus <- Antwort{code, erg}:
				default:
				}
			case <-fertig:
				return
			}
		}
	}()
	ab := func() {
		close(fertig)
		t.conn.RemoveMatchSignal(opt...)
		t.conn.RemoveSignal(sig)
	}
	return aus, ab, nil
}

func (t *dbusTransport) SaveFile(ctx context.Context, elternfenster, titel string, optionen map[string]dbus.Variant) (dbus.ObjectPath, error) {
	var handle dbus.ObjectPath
	err := t.conn.Object(dienst, objekt).CallWithContext(ctx, fileChooser+".SaveFile", 0, elternfenster, titel, optionen).Store(&handle)
	if err != nil {
		var de dbus.Error
		if e, ok := err.(dbus.Error); ok {
			de = e
		}
		switch {
		case strings.HasSuffix(de.Name, ".ServiceUnknown"), strings.HasSuffix(de.Name, ".NameHasNoOwner"),
			strings.HasSuffix(de.Name, ".UnknownMethod"), strings.HasSuffix(de.Name, ".UnknownObject"),
			strings.HasSuffix(de.Name, ".UnknownInterface"):
			return "", errKeinPortal(err)
		}
		return "", err
	}
	return handle, nil
}

func (t *dbusTransport) Schliesse(pfad dbus.ObjectPath) {
	if pfad == "" {
		return
	}
	t.conn.Object(dienst, pfad).Call(schnittstelleReq+".Close", dbus.FlagNoReplyExpected)
}

func errKeinPortal(err error) error { return fmt.Errorf("%w: %v", ErrKeinPortal, err) }
