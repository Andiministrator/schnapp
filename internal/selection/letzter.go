// SPDX-License-Identifier: GPL-3.0-or-later

package selection

import (
	"encoding/json"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
)

// Letzter ist der zuletzt gewählte Bereich für `--repeat` (ADR-0017). Er liegt im Zustandsordner, nicht in den
// Einstellungen (ADR-0018): Schnapp merkt ihn sich selbst.
//
// Wiedererkannt wird der Monitor an Name, Größe und Skala (F-85). Ein baugleicher Monitor am selben Anschluss
// gilt bewusst als derselbe (F-121, akzeptiert): KWin liefert keine Geräte-Kennung, und `ui.Monitor` kennt keine
// Lage. Aufgenommen wird dann nur der früher gewählte Bereich dieses Anschlusses — kein Datenleck.
type Letzter struct {
	Monitor string          `json:"monitor"` // Name der Wayland-Ausgabe
	Groesse image.Point     `json:"size"`    // Bildgröße des Monitors bei der Wahl, in Pixeln
	Skala   float64         `json:"scale"`   // Skalierung des Monitors bei der Wahl (F-85); fehlt in alten Dateien
	Bereich image.Rectangle `json:"area"`    // in Bildpixeln
}

// ErrKeinLetzter: Es gibt noch keinen gemerkten Bereich (oder er ist unbrauchbar).
var ErrKeinLetzter = errors.New("selection: kein letzter Bereich gemerkt")

// ErrKeinZustandsordner: Ohne absolutes XDG_STATE_HOME oder HOME gibt es keinen Ort für den letzten Bereich (F-86).
var ErrKeinZustandsordner = errors.New("selection: kein Zustandsordner (HOME fehlt)")

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

// skalaToleranz ist der erlaubte Unterschied beim Vergleich der Skala (Gleitkomma, kein ==).
const skalaToleranz = 0.001

// SkalaPasst vergleicht die gemerkte Skala mit der jetzigen (F-85).
func (l Letzter) SkalaPasst(skala float64) bool {
	return math.Abs(l.Skala-skala) < skalaToleranz
}

// Gueltig prüft, ob der Bereich zu einem Bild der Größe groesse passt. Eine Datei ohne Skala (Format vor F-85)
// gilt als unbrauchbar: Der Nutzer wählt einmal neu, statt dass ein Monitor mit geänderter Skala unbemerkt passt.
func (l Letzter) Gueltig() bool {
	return l.Monitor != "" && l.Groesse.X > 0 && l.Groesse.Y > 0 && l.Skala > 0 && !l.Bereich.Empty() &&
		l.Bereich.In(image.Rectangle{Max: l.Groesse})
}

// SpeichereLetzten schreibt den Bereich atomar (Ordner 0700, Datei 0600, wie ADR-0016).
func SpeichereLetzten(pfad string, l Letzter) error {
	if pfad == "" {
		return ErrKeinZustandsordner
	}
	if !l.Gueltig() {
		return errors.New("selection: ungültiger Bereich")
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
