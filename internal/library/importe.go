// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Übernommene Dateien von außen (ADR-0016 Nachtrag 10): Öffnet der Editor eine Datei außerhalb der Ablage, legt er sie
// als Aufnahme ab und merkt sich hier die SHA-256-Summe ihres Inhalts mit dem Pfad der Aufnahme. Dieselbe unveränderte
// Datei führt beim nächsten Öffnen zur vorhandenen Aufnahme. Die Liste ist nur lokal (neben dem Index, 0600) und eine
// Bequemlichkeit: Fehlt sie oder ist sie kaputt, entsteht höchstens eine weitere Kopie.

// importZeile ist eine Zeile der Liste.
type importZeile struct {
	Summe string `json:"sha256"`
	Pfad  string `json:"pfad"`
}

// importMax begrenzt das Lesen der Liste (eine Zeile ist rund 120 Byte; 8 MiB reichen für zehntausende Übernahmen).
const importMax = 8 << 20

// importPfad ist die Liste neben dem Index.
func (b *Bibliothek) importPfad() string {
	return filepath.Join(filepath.Dir(b.Orte.Index), "imports.jsonl")
}

// Importiert liefert die Aufnahme (Pfad relativ zur Ablage), die aus einer Datei mit dem Inhalt summe (SHA-256)
// übernommen wurde, wenn sie noch als reguläre Datei in der Ablage liegt. Die letzte passende Zeile gilt.
func (b *Bibliothek) Importiert(summe []byte) (string, bool) {
	f, err := os.Open(b.importPfad())
	if err != nil {
		return "", false
	}
	defer f.Close()
	suche, gefunden := hex.EncodeToString(summe), ""
	sc := bufio.NewScanner(io.LimitReader(f, importMax))
	for sc.Scan() {
		var z importZeile
		if json.Unmarshal(sc.Bytes(), &z) == nil && z.Summe == suche && z.Pfad != "" {
			gefunden = z.Pfad
		}
	}
	if gefunden == "" || strings.HasPrefix(gefunden, "/") || strings.Contains("/"+gefunden+"/", "/../") {
		return "", false
	}
	fi, err := os.Lstat(filepath.Join(b.Orte.Ablage, filepath.FromSlash(gefunden)))
	if err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	return gefunden, true
}

// MerkeImport hängt die Übernahme der Datei mit dem Inhalt summe als Aufnahme rel an die Liste.
func (b *Bibliothek) MerkeImport(summe []byte, rel string) (fehler error) {
	defer errprefix.Defer("library", &fehler)
	if err := os.MkdirAll(filepath.Dir(b.importPfad()), 0o700); err != nil {
		return err
	}
	zeile, err := json.Marshal(importZeile{Summe: hex.EncodeToString(summe), Pfad: rel})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(b.importPfad(), os.O_WRONLY|os.O_APPEND|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(zeile, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
