// SPDX-License-Identifier: GPL-3.0-or-later

package selection

import (
	"encoding/json"
	"errors"
	"image"
	"os"
	"path/filepath"
)

// Letzter ist der zuletzt gewählte Bereich für `--repeat` (ADR-0017). Er liegt im Zustandsordner, nicht in den
// Einstellungen (ADR-0018): Schnapp merkt ihn sich selbst.
type Letzter struct {
	Monitor string          `json:"monitor"` // Name der Wayland-Ausgabe
	Groesse image.Point     `json:"size"`    // Bildgröße des Monitors bei der Wahl, in Pixeln
	Bereich image.Rectangle `json:"area"`    // in Bildpixeln
}

// ErrKeinLetzter: Es gibt noch keinen gemerkten Bereich (oder er ist unbrauchbar).
var ErrKeinLetzter = errors.New("auswahl: kein letzter Bereich gemerkt")

// ErrKeinZustandsordner: Ohne absolutes XDG_STATE_HOME oder HOME gibt es keinen Ort für den letzten Bereich (F-86).
var ErrKeinZustandsordner = errors.New("auswahl: kein Zustandsordner (HOME fehlt)")

// LetzterPfad ist `$XDG_STATE_HOME/schnapp/last-area.json`, ohne (absolutes) XDG_STATE_HOME
// `$HOME/.local/state/schnapp/last-area.json`. Ohne absolutes HOME ist der Pfad leer.
func LetzterPfad(umgebung func(string) string) string {
	if w := umgebung("XDG_STATE_HOME"); filepath.IsAbs(w) {
		return filepath.Join(filepath.Clean(w), "schnapp", "last-area.json")
	}
	heim := umgebung("HOME")
	if !filepath.IsAbs(heim) {
		return ""
	}
	return filepath.Join(filepath.Clean(heim), ".local", "state", "schnapp", "last-area.json")
}

// Gueltig prüft, ob der Bereich zu einem Bild der Größe groesse passt.
func (l Letzter) Gueltig() bool {
	return l.Monitor != "" && l.Groesse.X > 0 && l.Groesse.Y > 0 && !l.Bereich.Empty() &&
		l.Bereich.In(image.Rectangle{Max: l.Groesse})
}

// SpeichereLetzten schreibt den Bereich atomar (Ordner 0700, Datei 0600, wie ADR-0016).
func SpeichereLetzten(pfad string, l Letzter) error {
	if pfad == "" {
		return ErrKeinZustandsordner
	}
	if !l.Gueltig() {
		return errors.New("auswahl: ungültiger Bereich")
	}
	ordner := filepath.Dir(pfad)
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		return err
	}
	// Den eigenen Ordner auch dann auf 0700 bringen, wenn es ihn schon mit anderen Rechten gab (wie ADR-0016, F-84).
	if err := os.Chmod(ordner, 0o700); err != nil {
		return err
	}
	daten, err := json.Marshal(l)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(ordner, ".last-area-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	err = tmp.Chmod(0o600)
	if err == nil {
		_, err = tmp.Write(append(daten, '\n'))
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

// LadeLetzten liest den gemerkten Bereich. Fehlt die Datei oder ist sie unbrauchbar, kommt ErrKeinLetzter; ohne
// Pfad (kein Zustandsordner) ErrKeinZustandsordner.
func LadeLetzten(pfad string) (Letzter, error) {
	if pfad == "" {
		return Letzter{}, ErrKeinZustandsordner
	}
	daten, err := os.ReadFile(pfad)
	if errors.Is(err, os.ErrNotExist) {
		return Letzter{}, ErrKeinLetzter
	}
	if err != nil {
		return Letzter{}, err
	}
	var l Letzter
	if json.Unmarshal(daten, &l) != nil || !l.Gueltig() {
		return Letzter{}, ErrKeinLetzter
	}
	return l, nil
}
