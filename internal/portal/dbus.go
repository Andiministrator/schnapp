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

// signalArt sagt, was mit einem Wert aus dem Signalkanal geschieht.
type signalArt int

const (
	signalVerwerfen signalArt = iota // fremdes oder fehlerhaftes Signal
	signalPassend                    // passende Antwort
	signalEnde                       // Kanal geschlossen: Verbindung weg
)

// wertSignal ist der Filter der Abo-Goroutine, ohne Bus prüfbar: Es nimmt nur Request.Response auf pfad an, mit
// zwei Werten (u, a{sv}) und von einem Absender, den absenderOK zulässt (A-7). ok=false heißt: Kanal geschlossen (A-2).
func wertSignal(s *dbus.Signal, ok bool, pfad dbus.ObjectPath, absenderOK func(string) bool) (Antwort, signalArt) {
	if !ok {
		return Antwort{Err: ErrVerbindung}, signalEnde
	}
	if s == nil || s.Path != pfad || s.Name != schnittstelleReq+"."+signalAntwort || len(s.Body) != 2 {
		return Antwort{}, signalVerwerfen
	}
	if absenderOK != nil && !absenderOK(s.Sender) {
		return Antwort{}, signalVerwerfen
	}
	code, ok1 := s.Body[0].(uint32)
	erg, ok2 := s.Body[1].(map[string]dbus.Variant)
	if !ok1 || !ok2 {
		return Antwort{}, signalVerwerfen
	}
	return Antwort{Code: code, Ergebnisse: erg}, signalPassend
}

// absenderPruefung lässt den Namen des Portals und dessen eindeutigen Besitzer zu. Der Bus trägt als Absender eines
// Signals immer den eindeutigen Namen (:1.x); ist der Besitzer beim Anmelden noch unbekannt (das Portal startet erst mit
// SaveFile per D-Bus-Aktivierung), schlägt die Prüfung ihn beim ersten fremden Absender nach (nachschlagen darf nil
// sein). Sonst verwürfe sie jede echte Antwort, und der Export hinge (Verify N-1).
func absenderPruefung(besitzer string, nachschlagen func() string) func(string) bool {
	return func(sender string) bool {
		if sender == dienst || (besitzer != "" && sender == besitzer) {
			return true
		}
		if nachschlagen != nil {
			if neu := nachschlagen(); neu != "" && neu != besitzer {
				besitzer = neu
				return sender == besitzer
			}
		}
		return false
	}
}

// besitzerVon fragt den eindeutigen Namen, dem dienst gerade gehört; leer, wenn niemandem.
func (t *dbusTransport) besitzerVon() string {
	var besitzer string
	if err := t.conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, dienst).Store(&besitzer); err != nil {
		return ""
	}
	return besitzer
}

func (t *dbusTransport) Abonniere(pfad dbus.ObjectPath) (<-chan Antwort, func(), error) {
	pruefe := absenderPruefung(t.besitzerVon(), t.besitzerVon)
	sig := make(chan *dbus.Signal, 8)
	t.conn.Signal(sig)
	opt := []dbus.MatchOption{
		dbus.WithMatchSender(dienst),
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
			case s, ok := <-sig:
				r, art := wertSignal(s, ok, pfad, pruefe)
				switch art {
				case signalEnde: // geschlossener Kanal liefert endlos nil: melden und beenden, nicht drehen
					select {
					case aus <- r:
					default:
					}
					return
				case signalPassend:
					select {
					case aus <- r:
					default:
					}
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

func (t *dbusTransport) Rufe(ctx context.Context, methode, elternfenster, titel string, optionen map[string]dbus.Variant) (dbus.ObjectPath, error) {
	var handle dbus.ObjectPath
	err := t.conn.Object(dienst, objekt).CallWithContext(ctx, fileChooser+"."+methode, 0, elternfenster, titel, optionen).Store(&handle)
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
