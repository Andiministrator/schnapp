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
	// schliessen nimmt einen Hinweis zurück (F-490: sonst steht er in der nächsten Aufnahme).
	schliessen = dienst + ".CloseNotification"
	// Anzeigedauer in Millisekunden; der Server darf sie verkürzen.
	dauer int32 = 8000
	// frist begrenzt den Aufruf: Ein hängender Benachrichtigungsdienst darf die Aufnahme nicht ewig halten.
	frist = 5 * time.Second
)

// ErrKeineKennung: Der Dienst hat geantwortet, aber ohne gültige Kennung — als gescheitert gewertet.
var ErrKeineKennung = errors.New("notify: Antwort ohne Kennung")

// rufe ist die Naht für Tests: führt die Methode des Dienstes aus und liefert den Antwortkörper. nil = echter
// Session-Bus.
var rufe func(ctx context.Context, methode string, args []any) ([]any, error)

// Zeige schickt eine Benachrichtigung mit Titel und Text und liefert ihre Kennung (für Schliesse). Der Absendername
// ist „Schnapp“, das Symbol `schnapp`.
func Zeige(ctx context.Context, titel, text string) (id uint32, fehler error) {
	defer errprefix.Defer("notify", &fehler)
	// critical: auch bei „Nicht stören“ sichtbar (Plasma-Voreinstellung); der Hinweis ist der einzige sichtbare
	// Rest einer Aufnahme ohne Dialog (ADR-0030 Nachtrag 4 Punkt 8, F-466).
	return schicke(ctx, 0, titel, text, 2, dauer)
}

// Ersetze zeigt einen gewöhnlichen Hinweis (Dringlichkeit normal, 2 s) oder ersetzt den mit der Kennung ersetzt
// (0: neu), etwa für den Countdown vor einer verzögerten Aufnahme (ADR-0042). Liefert die Kennung des Hinweises.
func Ersetze(ctx context.Context, ersetzt uint32, titel, text string) (id uint32, fehler error) {
	defer errprefix.Defer("notify", &fehler)
	return schicke(ctx, ersetzt, titel, text, 1, kurz)
}

// kurz ist die Anzeigedauer eines Hinweises, den Ersetze jede Sekunde erneuert.
const kurz int32 = 2000

func schicke(ctx context.Context, ersetzt uint32, titel, text string, dringlich byte, ms int32) (uint32, error) {
	ctx, ende := context.WithTimeout(ctx, frist)
	defer ende()
	args := []any{"Schnapp", ersetzt, "schnapp", titel, text, []string{}, map[string]dbus.Variant{
		"urgency":       dbus.MakeVariant(dringlich),
		"desktop-entry": dbus.MakeVariant("schnapp"),
	}, ms}
	body, err := ruf(ctx, methode, args)
	if err != nil {
		return 0, err
	}
	if len(body) != 1 {
		return 0, ErrKeineKennung
	}
	id, ok := body[0].(uint32)
	if !ok || id == 0 {
		return 0, ErrKeineKennung
	}
	return id, nil
}

// Schliesse nimmt den Hinweis mit der Kennung id zurück (CloseNotification). Der MCP-Server ruft es vor der nächsten
// Aufnahme, damit der vorige Hinweis nicht im Bild steht (F-490); ein schon geschlossener Hinweis ist kein Fehler des
// Dienstes.
func Schliesse(ctx context.Context, id uint32) (fehler error) {
	defer errprefix.Defer("notify", &fehler)
	ctx, ende := context.WithTimeout(ctx, frist)
	defer ende()
	_, err := ruf(ctx, schliessen, []any{id})
	return err
}

func ruf(ctx context.Context, m string, args []any) ([]any, error) {
	if rufe != nil {
		return rufe(ctx, m, args)
	}
	return echt(ctx, m, args)
}

func echt(ctx context.Context, m string, args []any) ([]any, error) {
	// Die gemeinsame Verbindung des Prozesses: CloseNotification kommt so vom selben Absender wie Notify (ein Dienst darf
	// das Schließen fremder Hinweise ablehnen; Sitzungs-QA B-4). Sie bleibt offen, solange der Prozess läuft.
	c, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	call := c.Object(dienst, pfad).CallWithContext(ctx, m, 0, args...)
	return call.Body, call.Err
}
