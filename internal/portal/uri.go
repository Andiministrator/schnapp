// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// ErrKeineDatei: Die Adresse ist keine lokale Datei (kein file://), ADR-0025 Punkt 3.
var ErrKeineDatei = errors.New("portal: keine lokale Datei (file://)")

// URIZuPfad wandelt eine file://-Adresse des Portals in einen Pfad (Prozentkodierung aufgelöst). Andere Schemata,
// fremde Rechnernamen, relative und leere Pfade lehnt sie ab.
func URIZuPfad(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", errprefix.Wrap("portal", err) // F-379
	}
	if !strings.EqualFold(u.Scheme, "file") {
		return "", ErrKeineDatei
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", ErrKeineDatei
	}
	p := u.Path
	if p == "" || p[0] != '/' || strings.ContainsRune(p, 0) || strings.HasSuffix(p, "/") {
		return "", ErrKeineDatei
	}
	return filepath.Clean(p), nil
}
