// SPDX-License-Identifier: GPL-3.0-or-later

// Package rescue führt die Rettungskopie des Editors (ADR-0029): eine kleine JSON-Datei mit den Angaben einer
// Bearbeitung (Objekte, Zuschnitt, Streifen, Schwärzungen seit dem letzten Speichern), ohne Pixel. Sie liegt im
// privaten Zustandsordner, wird nach Änderungen gebündelt geschrieben, nach jedem gelungenen Speichern gelöscht und
// beim nächsten Öffnen derselben, unveränderten Datei angewendet.
package rescue

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Format ist die Fassung des JSON. Eine neuere Fassung wird nicht angewendet.
const Format = 1

// Grenzen beim Lesen: Eine Rettungskopie ist klein (wenige KiB); größere oder längere werden nicht gelesen.
const (
	maxDatei         = 64 << 20
	maxSchwaerzungen = 10000
	// Haltezeit ist, wie lange eine nicht anwendbare Rettungskopie liegen bleibt (ADR-0029 Punkt 6).
	Haltezeit = 30 * 24 * time.Hour
)

// Fehler.
var (
	ErrKeine      = errors.New("rescue: keine Rettungskopie")
	ErrAnders     = errors.New("rescue: die Datei hat sich seit der Rettungskopie geändert")
	ErrFormat     = errors.New("rescue: Rettungskopie unlesbar")
	ErrNeuer      = errors.New("rescue: Rettungskopie einer neueren Fassung")
	ErrKeinOrt    = errors.New("rescue: kein Zustandsordner (weder XDG_STATE_HOME noch HOME)")
	ErrGroesse    = errors.New("rescue: Rettungskopie passt nicht zur Bildgröße")
	ErrZuGross    = errors.New("rescue: Rettungskopie zu groß")
	ErrUnfertig   = errors.New("rescue: Verlauf unvollständig, Rettungskopie nicht geschrieben")
	errKeineDatei = errors.New("rescue: keine gewöhnliche Datei")
)

// Kennung beschreibt die Ausgangsdatei, auf die sich eine Rettungskopie bezieht (ADR-0029 Punkt 2): Nur wenn alle
// vier Angaben passen, wird sie angewendet.
type Kennung struct {
	Pfad    string    `json:"pfad"` // absolut
	Groesse int64     `json:"groesse"`
	Zeit    time.Time `json:"zeit"` // Änderungszeit
	SHA256  string    `json:"sha256"`
}

// KennungVon liest die Datei pfad (absolut) und bildet ihre Kennung.
func KennungVon(pfad string) (_ Kennung, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	f, err := os.Open(pfad)
	if err != nil {
		return Kennung{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Kennung{}, err
	}
	if !st.Mode().IsRegular() {
		return Kennung{}, fmt.Errorf("%w: %s", errKeineDatei, pfad)
	}
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return Kennung{}, err
	}
	return Kennung{Pfad: pfad, Groesse: n, Zeit: st.ModTime(), SHA256: hex.EncodeToString(h.Sum(nil))}, nil
}

// Gleich meldet, ob zwei Kennungen dieselbe Datei im selben Zustand beschreiben.
func (k Kennung) Gleich(o Kennung) bool {
	return k.Pfad == o.Pfad && k.Groesse == o.Groesse && k.Zeit.Equal(o.Zeit) && k.SHA256 == o.SHA256
}

// Schwaerzung ist eine Schwärzung im JSON (document.SchwaerzAngabe).
type Schwaerzung struct {
	Bereich [4]int `json:"bereich"` // x0 y0 x1 y1 im Original
	Block   int    `json:"block,omitempty"`
	Farbe   string `json:"farbe"` // #rrggbbaa
	Mosaik  int    `json:"mosaik,omitempty"`
}

// Kopie ist eine Rettungskopie.
type Kopie struct {
	Format        int           `json:"format"`
	Datei         Kennung       `json:"datei"`
	Gesichert     time.Time     `json:"gesichert"`
	Schwaerzungen []Schwaerzung `json:"schwaerzungen"`
	Angaben       string        `json:"angaben"` // SVG wie Block snAP (fileformat.AngabenSVG)
}

// Ort ist der Ordner der Rettungskopien: $XDG_STATE_HOME/schnapp/rescue, ohne XDG_STATE_HOME ~/.local/state/…
// (ADR-0029 Punkt 3). getenv ist os.Getenv oder eine Attrappe.
func Ort(getenv func(string) string) (string, error) {
	if s := getenv("XDG_STATE_HOME"); filepath.IsAbs(s) {
		return filepath.Join(s, "schnapp", "rescue"), nil
	}
	h := getenv("HOME")
	if !filepath.IsAbs(h) {
		return "", ErrKeinOrt
	}
	return filepath.Join(h, ".local", "state", "schnapp", "rescue"), nil
}

// Name ist der Dateiname der Rettungskopie zu einer Ausgangsdatei: SHA-256 des absoluten Pfads, damit kein Pfad im
// Dateinamen steht.
func Name(pfad string) string {
	h := sha256.Sum256([]byte(pfad))
	return hex.EncodeToString(h[:]) + ".json"
}

// Erzeuge bildet die Rettungskopie von d zur Ausgangsdatei k (auf der Ereignisschleife: liest nur Angaben, keine
// Pixel). Ist die Liste der Schwärzungen unvollständig (Grenze des Verlaufs), kommt ErrUnfertig.
func Erzeuge(d *document.Dokument, k Kennung, jetzt time.Time) (Kopie, error) {
	liste, ok := d.SchwaerzungenImVerlauf()
	if !ok {
		return Kopie{}, ErrUnfertig
	}
	kp := Kopie{Format: Format, Datei: k, Gesichert: jetzt.UTC(), Angaben: string(fileformat.AngabenSVG(d)),
		Schwaerzungen: make([]Schwaerzung, 0, len(liste))}
	for _, s := range liste {
		r, c := s.Bereich, s.Art.Farbe
		kp.Schwaerzungen = append(kp.Schwaerzungen, Schwaerzung{
			Bereich: [4]int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y}, Block: s.Art.Block, Mosaik: s.Art.Mosaik,
			Farbe: fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A),
		})
	}
	return kp, nil
}

// Schreibe legt kp atomar im Ordner ort ab (Ordner 0700, Datei 0600): temporäre Datei, Sync, Umbenennen.
func Schreibe(ort string, kp Kopie) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	daten, err := json.Marshal(kp)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ort, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(ort, ".schreibt-*")
	if err != nil {
		return err
	}
	weg := tmp.Name()
	defer func() {
		if fehler != nil {
			_ = os.Remove(weg)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(daten); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(weg, filepath.Join(ort, Name(kp.Datei.Pfad)))
}

// Lies liest die Rettungskopie zur Ausgangsdatei pfad aus dem Ordner ort; ohne Kopie kommt ErrKeine.
func Lies(ort, pfad string) (_ *Kopie, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	f, err := os.Open(filepath.Join(ort, Name(pfad)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrKeine
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	daten, err := io.ReadAll(io.LimitReader(f, maxDatei+1))
	if err != nil {
		return nil, err
	}
	if len(daten) > maxDatei {
		return nil, ErrZuGross
	}
	var kp Kopie
	dec := json.NewDecoder(bytes.NewReader(daten))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&kp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	switch {
	case kp.Format > Format:
		return nil, fmt.Errorf("%w: %d > %d", ErrNeuer, kp.Format, Format)
	case kp.Format < 1:
		return nil, fmt.Errorf("%w: Fassung %d", ErrFormat, kp.Format)
	case kp.Datei.Pfad != pfad:
		return nil, fmt.Errorf("%w: Pfad passt nicht", ErrFormat)
	case len(kp.Schwaerzungen) > maxSchwaerzungen:
		return nil, ErrZuGross
	}
	return &kp, nil
}

// Loesche entfernt die Rettungskopie zur Ausgangsdatei pfad; fehlt sie, ist das kein Fehler.
func Loesche(ort, pfad string) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	err := os.Remove(filepath.Join(ort, Name(pfad)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Aufraeumen entfernt Rettungskopien und liegen gebliebene temporäre Dateien, die älter als Haltezeit sind
// (ADR-0029 Punkt 6). Nur Dateien dieses Ordners mit eigenen Namen; ein fehlender Ordner ist kein Fehler.
func Aufraeumen(ort string, jetzt time.Time) (entfernt int, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	eintraege, err := os.ReadDir(ort)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var fehlerListe []error
	for _, e := range eintraege {
		n := e.Name()
		if !e.Type().IsRegular() || !eigenerName(n) {
			continue
		}
		info, err := e.Info()
		if err != nil || jetzt.Sub(info.ModTime()) < Haltezeit {
			continue
		}
		if err := os.Remove(filepath.Join(ort, n)); err != nil && !errors.Is(err, os.ErrNotExist) {
			fehlerListe = append(fehlerListe, err)
			continue
		}
		entfernt++
	}
	return entfernt, errors.Join(fehlerListe...)
}

// Anwenden wendet kp auf d an (ADR-0029 Punkt 6): erst die Schwärzungen in ihrer Reihenfolge, dann die Angaben in
// einem Schritt. Rückgängig führt zurück auf den Stand vor dem Anwenden. pruefe (etwa render.Pruefe) bekommt vorher
// einen Abzug mit angewendeter Kopie; lehnt es ab, bleibt d unverändert. Passt die Kopie nicht (Größe, Grenzen), bleibt
// d ebenso unverändert.
func Anwenden(d *document.Dokument, kp *Kopie, pruefe func(*document.Dokument) error) (fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	a, err := fileformat.LeseAngaben(strings.NewReader(kp.Angaben))
	if err != nil {
		return err
	}
	if a.Groesse != d.OriginalGroesse() {
		return fmt.Errorf("%w: %v gegen %v", ErrGroesse, a.Groesse, d.OriginalGroesse())
	}
	liste := make([]document.SchwaerzAngabe, 0, len(kp.Schwaerzungen))
	for _, s := range kp.Schwaerzungen {
		var c color.NRGBA
		if n, err := fmt.Sscanf(s.Farbe, "#%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A); err != nil || n != 4 || len(s.Farbe) != 9 {
			return fmt.Errorf("%w: Farbe %q", ErrFormat, s.Farbe)
		}
		r := image.Rect(s.Bereich[0], s.Bereich[1], s.Bereich[2], s.Bereich[3])
		liste = append(liste, document.SchwaerzAngabe{Bereich: r, Art: document.Schwaerzung{Block: s.Block, Farbe: c, Mosaik: s.Mosaik}})
	}
	// Erst an einem Abzug: Scheitert ein Schritt oder die Prüfung, bleibt d, wie es war.
	probe, err := d.Abzug()
	if err != nil {
		return err
	}
	if err := wende(probe, liste, a); err != nil {
		return err
	}
	if pruefe != nil {
		if err := pruefe(probe); err != nil {
			return err
		}
	}
	return wende(d, liste, a)
}

func wende(d *document.Dokument, liste []document.SchwaerzAngabe, a *fileformat.Angaben) error {
	for _, s := range liste {
		if err := d.Schwaerzen(s.Bereich, s.Art); err != nil {
			return err
		}
	}
	return d.SetzeAngaben(a.Objekte, a.Zuschnitt, a.Streifen, a.Kante)
}

// eigenerName erkennt die Dateinamen dieses Pakets: Kopie (<64 hex>.json), beiseitegelegte Kopie
// (<64 hex>.<Zeit>.alt.json) und temporäre Datei beim Schreiben (.schreibt-…).
func eigenerName(n string) bool {
	if strings.HasPrefix(n, ".schreibt-") {
		return true
	}
	if len(n) < 64 || !istHex(n[:64]) {
		return false
	}
	rest := n[64:]
	if rest == ".json" {
		return true
	}
	mitte, ok := strings.CutSuffix(rest, ".alt.json")
	if !ok || len(mitte) < 2 || mitte[0] != '.' {
		return false
	}
	for _, c := range mitte[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func istHex(s string) bool {
	for _, c := range s {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// legeBeiseite benennt die Kopie zur Ausgangsdatei pfad um in <Name ohne .json>.<Zeit>.alt.json und liefert den neuen
// Dateinamen. Die Sitzung schreibt und löscht danach nur unter dem gewöhnlichen Namen (Verify 7, V7-1).
func legeBeiseite(ort, pfad string, jetzt time.Time) (_ string, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	alt := Name(pfad)
	neu := strings.TrimSuffix(alt, ".json") + fmt.Sprintf(".%d.alt.json", jetzt.UnixNano())
	if err := os.Rename(filepath.Join(ort, alt), filepath.Join(ort, neu)); err != nil {
		return "", err
	}
	return neu, nil
}
