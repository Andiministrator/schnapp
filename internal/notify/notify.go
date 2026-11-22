// SPDX-License-Identifier: GPL-3.0-or-later

// Package notify zeigt eine Desktop-Benachrichtigung über org.freedesktop.Notifications (Session-Bus). Schnapp braucht
// sie als sichtbaren Hinweis bei jeder Aufnahme über den MCP-Server (ADR-0030 Punkt 8, Nachtrag 2 Punkt 5): Scheitert
// sie, scheitert die Aufnahme. Darum meldet Zeige jeden Fehler, auch eine Antwort ohne Kennung.
package notify

import (
	"context"
	"errors"
	"time"

	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/godbus/dbus/v5"
)

const (
	dienst  = "org.freedesktop.Notifications"
	pfad    = "/org/freedesktop/Notifications"
	methode = dienst + ".Notify"
	// Anzeigedauer in Millisekunden; der Server darf sie verkürzen.
	dauer int32 = 8000
	// frist begrenzt den Aufruf: Ein hängender Benachrichtigungsdienst darf die Aufnahme nicht ewig halten.
	frist = 5 * time.Second
)

// ErrKeineKennung: Der Dienst hat geantwortet, aber ohne gültige Kennung — als gescheitert gewertet.
var ErrKeineKennung = errors.New("notify: Antwort ohne Kennung")

// rufe ist die Naht für Tests: führt Notify aus und liefert den Antwortkörper. nil = echter Session-Bus.
var rufe func(ctx context.Context, args []any) ([]any, error)

// Zeige schickt eine Benachrichtigung mit Titel und Text. Der Absendername ist „Schnapp“, das Symbol `schnapp`.
func Zeige(ctx context.Context, titel, text string) (fehler error) {
	defer errprefix.Defer("notify", &fehler)
	ctx, ende := context.WithTimeout(ctx, frist)
	defer ende()
	args := []any{"Schnapp", uint32(0), "schnapp", titel, text, []string{}, map[string]dbus.Variant{
		"urgency":       dbus.MakeVariant(byte(1)), // normal
		"desktop-entry": dbus.MakeVariant("schnapp"),
	}, dauer}
	var body []any
	var err error
	if rufe != nil {
		body, err = rufe(ctx, args)
	} else {
		body, err = echt(ctx, args)
	}
	if err != nil {
		return err
	}
	if len(body) != 1 {
		return ErrKeineKennung
	}
	if id, ok := body[0].(uint32); !ok || id == 0 {
		return ErrKeineKennung
	}
	return nil
}

func echt(ctx context.Context, args []any) ([]any, error) {
	c, err := dbus.ConnectSessionBus() // eigene Verbindung, danach geschlossen
	if err != nil {
		return nil, err
	}
	defer c.Close()
	call := c.Object(dienst, pfad).CallWithContext(ctx, methode, 0, args...)
	return call.Body, call.Err
}
