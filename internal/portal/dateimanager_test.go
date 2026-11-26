// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/godbus/dbus/v5"
)

// PfadZuURI kodiert, was in einer Adresse eine Bedeutung hat, und URIZuPfad holt den Pfad unverändert zurück.
func TestPfadZuURI(t *testing.T) {
	for _, p := range []string{"/a/b.png", "/Bilder/schnapp/2026-11/a b#c?d%e.png", "/x/ä ö.png"} {
		u, err := PfadZuURI(p)
		if err != nil {
			t.Fatalf("%q: %v", p, err)
		}
		if p == "/a/b.png" && u != "file:///a/b.png" {
			t.Errorf("%q → %q", p, u)
		}
		zurueck, err := URIZuPfad(u)
		if err != nil || zurueck != p {
			t.Errorf("%q → %q → %q (%v)", p, u, zurueck, err)
		}
	}
	for _, p := range []string{"", "a/b.png", "/a\x00b"} {
		if _, err := PfadZuURI(p); !errors.Is(err, ErrKeineDatei) {
			t.Errorf("%q: %v, Soll ErrKeineDatei", p, err)
		}
	}
}

// ZeigeImDateimanager ruft ShowItems mit genau der Adresse der Datei; fehlt der Dienst, heißt der Fehler
// ErrKeinDateimanager, andere Fehler bleiben, was sie sind.
func TestZeigeImDateimanager(t *testing.T) {
	alt := dateimanagerRuf
	t.Cleanup(func() { dateimanagerRuf = alt })
	var gerufen []string
	var antwort error
	dateimanagerRuf = func(_ context.Context, uris []string) error {
		gerufen = append(gerufen, uris...)
		return antwort
	}
	if err := ZeigeImDateimanager(context.Background(), "/a/b c.png"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gerufen, []string{"file:///a/b%20c.png"}) {
		t.Errorf("gerufen %q", gerufen)
	}
	for _, name := range []string{"org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner",
		"org.freedesktop.DBus.Error.UnknownMethod", "org.freedesktop.DBus.Error.UnknownObject",
		"org.freedesktop.DBus.Error.UnknownInterface"} {
		antwort = dbus.Error{Name: name}
		if err := ZeigeImDateimanager(context.Background(), "/a.png"); !errors.Is(err, ErrKeinDateimanager) {
			t.Errorf("%s: %v, Soll ErrKeinDateimanager", name, err)
		}
		antwort = &dbus.Error{Name: name}
		if err := ZeigeImDateimanager(context.Background(), "/a.png"); !errors.Is(err, ErrKeinDateimanager) {
			t.Errorf("*%s: %v, Soll ErrKeinDateimanager", name, err)
		}
	}
	antwort = dbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"}
	if err := ZeigeImDateimanager(context.Background(), "/a.png"); err == nil || errors.Is(err, ErrKeinDateimanager) {
		t.Errorf("AccessDenied: %v", err)
	}
	gerufen = nil
	if err := ZeigeImDateimanager(context.Background(), "relativ.png"); !errors.Is(err, ErrKeineDatei) || gerufen != nil {
		t.Errorf("relativer Pfad: %v, gerufen %q", err, gerufen)
	}
}
