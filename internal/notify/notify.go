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

// ErsetzeMitAktion ist Ersetze mit einem Knopf am Hinweis: schluessel ist die Kennung der Aktion, die ActionInvoked
// meldet (Aktionen), beschriftung der sichtbare Text (ADR-0042 Nachtrag 2: „Cancel“ am Countdown).
func ErsetzeMitAktion(ctx context.Context, ersetzt uint32, titel, text, schluessel, beschriftung string) (id uint32, fehler error) {
	defer errprefix.Defer("notify", &fehler)
	return schicke(ctx, ersetzt, titel, text, 1, kurz, schluessel, beschriftung)
}

// kurz ist die Anzeigedauer eines Hinweises, den Ersetze jede Sekunde erneuert.
const kurz int32 = 2000

// schicke ruft Notify; aktionen sind Paare aus Kennung und Beschriftung (Spezifikation: actions).
func schicke(ctx context.Context, ersetzt uint32, titel, text string, dringlich byte, ms int32, aktionen ...string) (uint32, error) {
	ctx, ende := context.WithTimeout(ctx, frist)
	defer ende()
	if aktionen == nil {
		aktionen = []string{}
	}
	args := []any{"Schnapp", ersetzt, "schnapp", titel, text, aktionen, map[string]dbus.Variant{
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

// Aktion ist ein gemeldeter Klick auf einen Knopf eines Hinweises (Signal ActionInvoked).
type Aktion struct {
	ID         uint32
	Schluessel string
}

// abonniere ist die Naht für Tests: meldet ActionInvoked auf aus, bis ctx endet; der Aufbau hat höchstens aufbau.
// nil = echter Session-Bus.
var abonniere func(ctx context.Context, aufbau time.Duration, aus chan<- Aktion) error

// Aktionen abonniert ActionInvoked des Benachrichtigungsdienstes (ADR-0042 Nachtrag 2) und liefert die Klicks auf
// Knöpfe aller Hinweise; wer nur seine eigenen will, vergleicht die Kennung. Das Abonnement steht, wenn Aktionen ohne
// Fehler zurückkehrt, und lebt, bis ctx endet (danach kommt nichts mehr, der Kanal bleibt offen). aufbau begrenzt nur
// das Anmelden beim Bus, nicht die Lebensdauer (Kritik F-1052: mit der Frist als Lebensdauer kam kein Klick an).
func Aktionen(ctx context.Context, aufbau time.Duration) (aus <-chan Aktion, fehler error) {
	defer errprefix.Defer("notify", &fehler)
	k := make(chan Aktion, 4)
	f := abonniere
	if f == nil {
		f = echtAbonniere
	}
	if err := f(ctx, aufbau, k); err != nil {
		return nil, err
	}
	return k, nil
}

func echtAbonniere(ctx context.Context, aufbau time.Duration, aus chan<- Aktion) error {
	c, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	// Nur Signale des Benachrichtigungsdienstes: Der Bus vergleicht den Absender mit dem Inhaber des Namens; ein
	// anderes Programm kann den Countdown so nicht abbrechen (Kritik F-1054).
	regel := []dbus.MatchOption{dbus.WithMatchSender(dienst), dbus.WithMatchObjectPath(pfad),
		dbus.WithMatchInterface(dienst), dbus.WithMatchMember("ActionInvoked")}
	fctx, ende := context.WithTimeout(ctx, aufbau)
	defer ende()
	if err := c.AddMatchSignalContext(fctx, regel...); err != nil { // kehrt erst zurück, wenn der Bus die Regel kennt
		return err
	}
	roh := make(chan *dbus.Signal, 16)
	c.Signal(roh)
	go func() {
		defer func() {
			c.RemoveSignal(roh)
			_ = c.RemoveMatchSignal(regel...)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-roh:
				a, ok := aktionAus(sig)
				if !ok {
					continue
				}
				select {
				case aus <- a:
				default: // niemand liest: verwerfen statt den Bus zu blockieren
				}
			}
		}
	}()
	return nil
}

// aktionAus liest ein Signal ActionInvoked (Kennung uint32, Schlüssel string); alles andere zählt nicht.
func aktionAus(sig *dbus.Signal) (Aktion, bool) {
	if sig == nil || sig.Name != dienst+".ActionInvoked" || len(sig.Body) != 2 {
		return Aktion{}, false
	}
	id, ok1 := sig.Body[0].(uint32)
	sch, ok2 := sig.Body[1].(string)
	if !ok1 || !ok2 {
		return Aktion{}, false
	}
	return Aktion{ID: id, Schluessel: sch}, true
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
