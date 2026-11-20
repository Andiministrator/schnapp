// SPDX-License-Identifier: GPL-3.0-or-later

// Package portal spricht das Desktop-Portal über D-Bus an, ohne Gio: den Dateidialog zum Speichern
// (org.freedesktop.portal.FileChooser.SaveFile, ADR-0025). Der Busaufruf liegt hinter der Schnittstelle transport,
// damit Aufbau der Anfrage und Auswertung der Antwort ohne echten Bus prüfbar sind.
package portal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dienst             = "org.freedesktop.portal.Desktop"
	objekt             = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	fileChooser        = "org.freedesktop.portal.FileChooser"
	schnittstelleReq   = "org.freedesktop.portal.Request"
	signalAntwort      = "Response"
	musterGlob         = 0 // Art 0: Glob-Muster, Art 1: MIME-Typ (Portal-Spezifikation)
	musterMIME         = 1
	antwortOK          = 0
	antwortAbgebrochen = 1
)

var (
	// ErrAbgebrochen: Der Nutzer hat den Dialog abgebrochen (Antwort 1). Kein Fehler im eigentlichen Sinn.
	ErrAbgebrochen = errors.New("Dialog abgebrochen")
	// ErrKeinPortal: Es gibt keinen Sitzungsbus oder keinen Portal-Dienst mit FileChooser.
	ErrKeinPortal = errors.New("kein Desktop-Portal verfügbar")
)

// Filter ist ein Dateifilter des Dialogs (Name, dazu Globs und MIME-Typen).
type Filter struct {
	Name  string
	Globs []string
	MIMEs []string
}

// Anfrage beschreibt einen Speichern-Dialog.
type Anfrage struct {
	Titel     string
	Dateiname string // Vorschlag (current_name)
	Filter    []Filter
	Aktuell   int // Index des voreingestellten Filters
}

// Ergebnis ist die Wahl des Nutzers.
type Ergebnis struct {
	URI    string
	Filter int // Index in Anfrage.Filter, -1 wenn nicht bestimmbar
}

// Antwort ist das Signal Request.Response, schon aus dem Bus-Paket gelöst.
type Antwort struct {
	Code       uint32
	Ergebnisse map[string]dbus.Variant
}

// transport ist der Busaufruf. Die echte Umsetzung steht in dbus.go, die Tests nehmen eine Attrappe.
type transport interface {
	// Name ist der eindeutige Verbindungsname (":1.42").
	Name() string
	// Abonniere meldet sich für Response auf pfad an; erst wenn die Methode zurückkehrt, ist die Anmeldung wirksam.
	Abonniere(pfad dbus.ObjectPath) (<-chan Antwort, func(), error)
	SaveFile(ctx context.Context, elternfenster, titel string, optionen map[string]dbus.Variant) (dbus.ObjectPath, error)
	// Schliesse bricht eine offene Anfrage ab (Request.Close), soweit möglich.
	Schliesse(pfad dbus.ObjectPath)
	Close() error
}

// Dialog öffnet Dateidialoge über das Portal.
type Dialog struct {
	t     transport
	token func() string
}

// Neu verbindet mit dem Sitzungsbus. Ohne Bus: ErrKeinPortal.
func Neu() (*Dialog, error) {
	t, err := neuerDBusTransport()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeinPortal, err)
	}
	return &Dialog{t: t, token: zufallsToken}, nil
}

// Close gibt die Verbindung frei.
func (d *Dialog) Close() error { return d.t.Close() }

func zufallsToken() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "schnapp_0"
	}
	return "schnapp_" + hex.EncodeToString(b[:])
}

// RequestPfad ist der Objektpfad, unter dem das Portal die Antwort sendet:
// /org/freedesktop/portal/desktop/request/<Absender ohne ':' und mit '_' statt '.'>/<Token>.
func RequestPfad(absender, token string) dbus.ObjectPath {
	s := strings.ReplaceAll(strings.TrimPrefix(absender, ":"), ".", "_")
	return dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + s + "/" + token)
}

type musterD struct {
	Art    uint32
	Muster string
}

// filterD ist die Drahtform (sa(us)).
type filterD struct {
	Name   string
	Muster []musterD
}

func drahtFilter(f Filter) filterD {
	w := filterD{Name: f.Name, Muster: []musterD{}}
	for _, g := range f.Globs {
		w.Muster = append(w.Muster, musterD{musterGlob, g})
	}
	for _, m := range f.MIMEs {
		w.Muster = append(w.Muster, musterD{musterMIME, m})
	}
	return w
}

// baueOptionen setzt die Optionen von SaveFile zusammen.
func baueOptionen(token string, a Anfrage) map[string]dbus.Variant {
	opt := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"modal":        dbus.MakeVariant(true),
	}
	if a.Dateiname != "" {
		opt["current_name"] = dbus.MakeVariant(a.Dateiname)
	}
	if len(a.Filter) > 0 {
		fl := make([]filterD, len(a.Filter))
		for i, f := range a.Filter {
			fl[i] = drahtFilter(f)
		}
		opt["filters"] = dbus.MakeVariant(fl)
		if a.Aktuell >= 0 && a.Aktuell < len(fl) {
			opt["current_filter"] = dbus.MakeVariant(fl[a.Aktuell])
		}
	}
	return opt
}

// Speichern zeigt den Dialog und wartet auf die Antwort (oder auf ctx). Die Anmeldung für Request.Response steht vor
// dem Aufruf von SaveFile: Das Portal kann antworten, bevor der Aufruf zurückkehrt (bekannte Wettlauf-Falle).
func (d *Dialog) Speichern(ctx context.Context, a Anfrage) (Ergebnis, error) {
	token := d.token()
	pfad := RequestPfad(d.t.Name(), token)
	ch, ab, err := d.t.Abonniere(pfad)
	if err != nil {
		return Ergebnis{}, err
	}
	defer ab()
	handle, err := d.t.SaveFile(ctx, "", a.Titel, baueOptionen(token, a))
	if err != nil {
		return Ergebnis{}, err
	}
	var ch2 <-chan Antwort
	if handle != pfad && handle != "" { // ältere Portale: anderer Pfad als erwartet, dann dort zusätzlich lauschen
		c, ab2, err := d.t.Abonniere(handle)
		if err == nil {
			defer ab2()
			ch2 = c
		}
	}
	select {
	case r := <-ch:
		return auswerten(r, a)
	case r := <-ch2:
		return auswerten(r, a)
	case <-ctx.Done():
		d.t.Schliesse(handle)
		return Ergebnis{}, ctx.Err()
	}
}

// auswerten übersetzt Request.Response: 0 = Wahl, 1 = abgebrochen, sonst Fehler.
func auswerten(r Antwort, a Anfrage) (Ergebnis, error) {
	switch r.Code {
	case antwortAbgebrochen:
		return Ergebnis{}, ErrAbgebrochen
	case antwortOK:
	default:
		return Ergebnis{}, fmt.Errorf("Dialog endete ohne Ergebnis (Antwort %d)", r.Code)
	}
	var uris []string
	if v, ok := r.Ergebnisse["uris"]; ok {
		if err := v.Store(&uris); err != nil {
			return Ergebnis{}, fmt.Errorf("Antwort ohne lesbare Adressen: %w", err)
		}
	}
	if len(uris) == 0 {
		return Ergebnis{}, errors.New("Antwort ohne Datei")
	}
	erg := Ergebnis{URI: uris[0], Filter: -1}
	if v, ok := r.Ergebnisse["current_filter"]; ok {
		var f filterD
		if err := v.Store(&f); err == nil {
			for i, g := range a.Filter {
				if g.Name == f.Name {
					erg.Filter = i
					break
				}
			}
		}
	}
	return erg, nil
}
