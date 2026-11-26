// SPDX-License-Identifier: GPL-3.0-or-later

package portal

// „Im Dateimanager zeigen“ (ADR-0038 Punkt 3): org.freedesktop.FileManager1.ShowItems auf dem Sitzungsbus. Dolphin und
// Nautilus bieten die Schnittstelle an; sie öffnet den Ordner der Datei und markiert sie. Kein Portal im engeren Sinn,
// aber derselbe Bus und dieselbe Fehlerbehandlung wie die Dateidialoge.

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dateimanagerDienst  = "org.freedesktop.FileManager1"
	dateimanagerObjekt  = dbus.ObjectPath("/org/freedesktop/FileManager1")
	dateimanagerMethode = dateimanagerDienst + ".ShowItems"
)

// ErrKeinDateimanager: Es gibt keinen Sitzungsbus oder keinen Dienst org.freedesktop.FileManager1.
var ErrKeinDateimanager = errors.New("portal: kein Dateimanager (org.freedesktop.FileManager1) verfügbar")

// PfadZuURI wandelt einen absoluten Pfad in eine file://-Adresse (Prozentkodierung), die Umkehrung von URIZuPfad.
// Relative und leere Pfade lehnt sie ab.
func PfadZuURI(pfad string) (string, error) {
	if pfad == "" || !filepath.IsAbs(pfad) || strings.ContainsRune(pfad, 0) {
		return "", ErrKeineDatei
	}
	return (&url.URL{Scheme: "file", Path: filepath.Clean(pfad)}).String(), nil
}

// dateimanagerRuf ruft ShowItems mit den Adressen auf; Naht für Tests.
var dateimanagerRuf = func(ctx context.Context, uris []string) error {
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeinDateimanager, err)
	}
	defer c.Close()
	return c.Object(dateimanagerDienst, dateimanagerObjekt).CallWithContext(ctx, dateimanagerMethode, 0, uris, "").Err
}

// ZeigeImDateimanager öffnet den Ordner der Datei pfad im Dateimanager und markiert sie. Fehlt der Dienst, liefert sie
// ErrKeinDateimanager.
func ZeigeImDateimanager(ctx context.Context, pfad string) error {
	uri, err := PfadZuURI(pfad)
	if err != nil {
		return err
	}
	err = dateimanagerRuf(ctx, []string{uri})
	if err != nil && keinDienst(err) {
		return fmt.Errorf("%w: %v", ErrKeinDateimanager, err)
	}
	return err
}

// keinDienst meldet, ob err sagt, dass Dienst, Objekt oder Methode auf dem Bus fehlen.
func keinDienst(err error) bool {
	var de dbus.Error
	switch e := err.(type) {
	case dbus.Error:
		de = e
	case *dbus.Error:
		de = *e
	default:
		return false
	}
	for _, s := range []string{".ServiceUnknown", ".NameHasNoOwner", ".UnknownMethod", ".UnknownObject",
		".UnknownInterface"} {
		if strings.HasSuffix(de.Name, s) {
			return true
		}
	}
	return false
}
