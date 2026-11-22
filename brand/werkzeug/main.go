// SPDX-License-Identifier: GPL-3.0-or-later

// Werkzeug für die Marke (Karte #23, ersetzt paket.py und cd.py): baut das Logo-Paket und die Bilder und das PDF des
// Corporate Designs. Alle Dateien unter brand/logo/ und brand/cd/ werden hier erzeugt — nie von Hand ändern.
//
// Aufruf (im Repo, beliebiger Unterordner):
//
//	cd brand/werkzeug && go run . paket   → brand/logo/
//	cd brand/werkzeug && go run . cd      → brand/cd/, brand/corporate-design.pdf
//	cd brand/werkzeug && go run . alles
//
// Braucht: Open Sans SemiBold (OFL; Pfad über SCHNAPP_SCHRIFT, Vorgabe der Arch-Pfad), rsvg-convert, magick
// (ImageMagick, nur favicon.ico), für cd zusätzlich chromium und qpdf.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const schriftVorgabe = "/usr/share/fonts/TTF/OpenSans-SemiBold.ttf"

// findeMarke sucht vom Arbeitsordner aufwärts den Ordner brand/ mit corporate-design.md.
func findeMarke() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(d, "brand")
		if _, err := os.Stat(filepath.Join(p, "corporate-design.md")); err == nil {
			return p, nil
		}
		oben := filepath.Dir(d)
		if oben == d {
			return "", errors.New("brand/corporate-design.md nicht gefunden (im Repo aufrufen)")
		}
		d = oben
	}
}

func neuerBauer() (*bauer, error) {
	marke, err := findeMarke()
	if err != nil {
		return nil, err
	}
	pfad := os.Getenv("SCHNAPP_SCHRIFT")
	if pfad == "" {
		pfad = schriftVorgabe
	}
	s, err := ladeSchrift(pfad)
	if err != nil {
		return nil, fmt.Errorf("Schrift (SCHNAPP_SCHRIFT): %w", err)
	}
	return &bauer{schrift: s, marke: marke}, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Aufruf: go run . paket|cd|alles")
		os.Exit(2)
	}
	b, err := neuerBauer()
	if err == nil {
		switch os.Args[1] {
		case "paket":
			err = b.paket()
		case "cd":
			err = b.cd()
		case "alles":
			if err = b.paket(); err == nil {
				err = b.cd()
			}
		default:
			err = fmt.Errorf("unbekannt: %q (paket, cd oder alles)", os.Args[1])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}
