// SPDX-License-Identifier: GPL-3.0-or-later

package bibliothek

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// ErrKeinZustandsordner: Ohne absolutes HOME (und ohne absolutes XDG_STATE_HOME) gibt es keinen Ort für die
// Tagesmarke; das automatische Aufräumen entfällt dann.
var ErrKeinZustandsordner = errors.New("bibliothek: kein Zustandsordner (HOME fehlt)")

// TagesmarkePfad ist `$XDG_STATE_HOME/schnapp/last-cleanup`, ohne (absolutes) XDG_STATE_HOME
// `$HOME/.local/state/schnapp/last-cleanup` (ADR-0016 Nachtrag 2). Ohne absolutes HOME ist der Pfad leer.
func TagesmarkePfad(umgebung func(string) string) string {
	if w := umgebung("XDG_STATE_HOME"); filepath.IsAbs(w) {
		return filepath.Join(filepath.Clean(w), "schnapp", "last-cleanup")
	}
	heim := umgebung("HOME")
	if !filepath.IsAbs(heim) {
		return ""
	}
	return filepath.Join(filepath.Clean(heim), ".local", "state", "schnapp", "last-cleanup")
}

// BeanspruchenHeute entscheidet, ob das automatische Aufräumen heute laufen darf, und merkt es sich: Steht in der
// Marke schon das heutige Datum (Ortszeit von jetzt), kommt false. Sonst schreibt die Funktion das heutige Datum und
// liefert true. Die Marke wird vor dem Aufräumen geschrieben, damit es höchstens einmal am Tag läuft, auch wenn es
// scheitert; lässt sie sich nicht schreiben, kommt false mit Fehler — dann wird nicht aufgeräumt (im Zweifel nichts
// tun). Eine unlesbare oder fremde Marke gilt als „heute noch nicht“.
func BeanspruchenHeute(pfad string, jetzt time.Time) (bool, error) {
	if pfad == "" {
		return false, ErrKeinZustandsordner
	}
	heute := []byte(jetzt.Format(time.DateOnly) + "\n")
	// Fehlt die Marke oder ist sie unlesbar, wird sie neu geschrieben; scheitert das, entfällt das Aufräumen.
	if alt, err := os.ReadFile(pfad); err == nil && bytes.Equal(alt, heute) {
		return false, nil
	}
	if err := schreibeMarke(pfad, heute); err != nil {
		return false, err
	}
	return true, nil
}

// schreibeMarke schreibt atomar (Ordner 0700, Datei 0600, wie ADR-0016).
func schreibeMarke(pfad string, daten []byte) error {
	ordner := filepath.Dir(pfad)
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(ordner, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(ordner, ".last-cleanup-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	err = tmp.Chmod(0o600)
	if err == nil {
		_, err = tmp.Write(daten)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(name, pfad)
	}
	if err != nil {
		_ = os.Remove(name) // nur die eigene temporäre Datei
	}
	return err
}
