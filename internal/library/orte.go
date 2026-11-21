// SPDX-License-Identifier: GPL-3.0-or-later

// Package library verwaltet die Ablage der Aufnahmen (ADR-0016): Dateien im Bilderordner, einen Index als
// Cache (JSON-Zeilen) mit Vorschaubildern, Suche und das Aufräumen alter, unbeachteter Aufnahmen über den
// freedesktop-Papierkorb. Nur Standardbibliothek, kein cgo.
package library

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Rechte nach ADR-0016: Aufnahmen zeigen oft Vertrauliches.
const (
	rechteOrdner os.FileMode = 0o700
	rechteDatei  os.FileMode = 0o600
)

// Orte sind die Verzeichnisse und Dateien der Bibliothek. Alle Pfade sind absolut.
type Orte struct {
	Ablage     string // <Bilderordner>/schnapp
	Index      string // <Datenordner>/schnapp/bibliothek.jsonl
	Vorschau   string // <Cacheordner>/schnapp/vorschau
	Papierkorb string // <Datenordner>/Trash (freedesktop-Papierkorb des Nutzers)
}

// ErrKeinHeim meldet, dass die Umgebung kein absolutes Heimverzeichnis liefert.
var ErrKeinHeim = errors.New("bibliothek: kein absolutes Heimverzeichnis (HOME)")

// OrteAus bestimmt die Orte aus einer Umgebung (Umgebungsvariable → Wert, leer wenn nicht gesetzt). Das Programm
// übergibt os.Getenv, Tests ihre eigene Umgebung.
func OrteAus(umgebung func(string) string) (Orte, error) {
	heim := umgebung("HOME")
	if !filepath.IsAbs(heim) {
		return Orte{}, ErrKeinHeim
	}
	heim = filepath.Clean(heim)
	ordner := func(name, vorgabe string) string {
		if w := umgebung(name); filepath.IsAbs(w) { // laut XDG gelten nur absolute Werte
			return filepath.Clean(w)
		}
		return filepath.Join(heim, vorgabe)
	}
	konfig := ordner("XDG_CONFIG_HOME", ".config")
	daten := ordner("XDG_DATA_HOME", filepath.Join(".local", "share"))
	cache := ordner("XDG_CACHE_HOME", ".cache")

	bilder := bilderordner(filepath.Join(konfig, "user-dirs.dirs"), heim)
	return Orte{
		Ablage:     filepath.Join(bilder, "schnapp"),
		Index:      filepath.Join(daten, "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(cache, "schnapp", "vorschau"),
		Papierkorb: filepath.Join(daten, "Trash"),
	}, nil
}

// bilderordner liest XDG_PICTURES_DIR aus user-dirs.dirs; fehlt die Datei oder die Zeile, gilt ~/Pictures.
func bilderordner(datei, heim string) string {
	vorgabe := filepath.Join(heim, "Pictures")
	f, err := os.Open(datei)
	if err != nil {
		return vorgabe
	}
	defer f.Close()
	gefunden := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() { // die letzte gültige Zeile gewinnt, wie bei einer Shell-Datei
		z := strings.TrimSpace(sc.Text())
		wert, ok := strings.CutPrefix(z, "XDG_PICTURES_DIR=")
		if !ok {
			continue
		}
		wert = strings.TrimSpace(wert)
		if len(wert) >= 2 && wert[0] == '"' && wert[len(wert)-1] == '"' {
			wert = wert[1 : len(wert)-1]
		}
		wert = strings.ReplaceAll(wert, `\"`, `"`)
		if r, ok := strings.CutPrefix(wert, "$HOME"); ok && (r == "" || r[0] == '/') {
			wert = heim + r
		}
		if filepath.IsAbs(wert) {
			gefunden = filepath.Clean(wert)
		}
	}
	if gefunden == "" || gefunden == heim { // das Heimverzeichnis selbst ist nach XDG „kein Bilderordner“
		return vorgabe
	}
	return gefunden
}
