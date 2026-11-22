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
	"unicode/utf8"

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
// vier Angaben passen, wird sie angewendet. Im JSON steht sie als kennungJSON.
type Kennung struct {
	Pfad    string // absolut, Bytes wie im Dateisystem (nicht unbedingt gültiges UTF-8)
	Groesse int64
	Zeit    time.Time // Änderungszeit
	SHA256  string
}

// kennungJSON ist die Kennung im JSON. Ein Pfad ohne gültiges UTF-8 (etwa ein alter Latin-1-Dateiname) steht
// verlustfrei als Base64 in pfad_roh; pfad trägt dann eine lesbare Fassung mit U+FFFD. Ohne pfad_roh gilt pfad — so
// lesen sich ältere Kopien unverändert (F-410). encoding/json ersetzte ungültige Bytes sonst still durch U+FFFD, und
// die Kopie passte nie mehr zu ihrem Pfad.
type kennungJSON struct {
	Pfad    string    `json:"pfad"`
	PfadRoh []byte    `json:"pfad_roh,omitempty"`
	Groesse int64     `json:"groesse"`
	Zeit    time.Time `json:"zeit"`
	SHA256  string    `json:"sha256"`
}

// MarshalJSON schreibt die Kennung; einen Pfad ohne gültiges UTF-8 zusätzlich roh (F-410).
func (k Kennung) MarshalJSON() ([]byte, error) {
	j := kennungJSON{Pfad: k.Pfad, Groesse: k.Groesse, Zeit: k.Zeit, SHA256: k.SHA256}
	if !utf8.ValidString(k.Pfad) {
		j.Pfad, j.PfadRoh = strings.ToValidUTF8(k.Pfad, "�"), []byte(k.Pfad)
	}
	return json.Marshal(j)
}

// UnmarshalJSON liest die Kennung, unbekannte Felder abgelehnt wie in der ganzen Kopie; pfad_roh geht vor pfad.
func (k *Kennung) UnmarshalJSON(daten []byte) error {
	var j kennungJSON
	dec := json.NewDecoder(bytes.NewReader(daten))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&j); err != nil {
		return err
	}
	*k = Kennung{Pfad: j.Pfad, Groesse: j.Groesse, Zeit: j.Zeit, SHA256: j.SHA256}
	if j.PfadRoh != nil {
		k.Pfad = string(j.PfadRoh)
	}
	return nil
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
// Pixel). Ist die Liste der Schwärzungen unvollständig (Grenze des Verlaufs), kommt ErrUnfertig. Belegen die
// Schwärzungen mehr, als Anwenden beim Öffnen annimmt (Flächenbudget eines frisch geöffneten Dokuments,
// document.VerlaufGrenze), kommt ErrZuGross (F-431): Eine solche Kopie würde beim Öffnen abgelehnt und verdrängte die
// zuletzt geschriebene, anwendbare. Die Sicherung lässt in beiden Fällen die vorige Kopie stehen.
func Erzeuge(d *document.Dokument, k Kennung, jetzt time.Time) (_ Kopie, fehler error) {
	defer errprefix.Defer("rescue", &fehler) // F-451: auch Fehler aus document.PruefeSchwaerzung tragen das Präfix
	liste, ok := d.SchwaerzungenImVerlauf()
	if !ok {
		return Kopie{}, ErrUnfertig
	}
	if err := budgetPruefer(d, liste, document.VerlaufGrenze()); err != nil {
		return Kopie{}, err
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
		if strings.HasSuffix(n, sperrEndung) {
			// Sperre einer abgestürzten Sitzung: nur entfernen, wenn niemand sie hält (F-413).
			if ok, err := raeumeSperre(filepath.Join(ort, n)); err != nil {
				fehlerListe = append(fehlerListe, err)
			} else if ok {
				entfernt++
			}
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
// einen Abzug mit den angewendeten Angaben; lehnt es ab, bleibt d unverändert. Passt die Kopie nicht (Größe, Grenzen),
// bleibt d ebenso unverändert.
//
// Flächenbudget (F-412): Alle Schwärzungen werden vorab ohne Pixel geprüft (document.PruefeSchwaerzung) und ihre
// Flächen summiert. Belegten sie im Rückgängig-Verlauf mehr Pixelkopien, als er noch aufnimmt (document.VerlaufPlatz,
// 4 Bytes je Pixel, Grenze 512 MiB), kommt ErrZuGross, bevor ein Pixel angefasst wird. So bleibt die Arbeit beim
// Öffnen auf rund 128 Mi Pixel begrenzt (rund eine Sekunde), und keine Schwärzung fällt beim Wiederherstellen aus dem
// Verlauf (Rückgängig trägt bis zum gespeicherten Stand, Erzeuge liefert danach kein ErrUnfertig). Eine Sitzung kann
// dennoch Schwärzungen über dem Budget halten, ohne dass der Verlauf eine verworfen hätte: begrenzeVerlauf lässt den
// jüngsten Schritt immer stehen, eine einzelne Schwärzung über 512 MiB setzt den Merker nicht (F-431). Deshalb prüft
// Erzeuge dasselbe Budget (pruefeBudget mit document.VerlaufGrenze) und schreibt eine solche Kopie nicht; die zuletzt
// geschriebene bleibt. Abgelehnt heißt: nichts angewendet, auch keine der Schwärzungen (Prinzip 2: kein halb
// geschwärztes Ergebnis gilt als wiederhergestellt).
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
	// Schwärzungen vorab ohne Pixel prüfen und gegen den Platz im Verlauf rechnen (F-412).
	if err := pruefeBudget(d, liste, d.VerlaufPlatz()); err != nil {
		return err
	}
	// Die Angaben erst an einem Abzug: Scheitert ein Schritt oder die Prüfung, bleibt d, wie es war. Die Schwärzungen
	// ändern nur Pixel, die weder SetzeAngaben noch pruefe lesen; sie sind oben geprüft und laufen nur einmal, an d.
	probe, err := d.Abzug()
	if err != nil {
		return err
	}
	if err := probe.SetzeAngaben(a.Objekte, a.Zuschnitt, a.Streifen, a.Kante); err != nil {
		return err
	}
	if pruefe != nil {
		if err := pruefe(probe); err != nil {
			return err
		}
	}
	for _, s := range liste {
		if err := d.Schwaerzen(s.Bereich, s.Art); err != nil {
			return err // nicht erreichbar: geprüft wie oben, ohne Zug (sonst schon Abzug gescheitert)
		}
	}
	return d.SetzeAngaben(a.Objekte, a.Zuschnitt, a.Streifen, a.Kante)
}

// budgetPruefer ist die Naht für Tests (F-451): ein Fehler aus document.PruefeSchwaerzung ist sonst nicht herzustellen.
var budgetPruefer = pruefeBudget

// pruefeBudget prüft die Schwärzungen ohne Pixel (document.PruefeSchwaerzung) und rechnet ihre Flächen im Bild gegen
// platz (Bytes im Rückgängig-Verlauf, 4 je Pixel). Anwenden rechnet gegen den freien Platz des Dokuments, Erzeuge gegen
// den eines frisch geöffneten (F-412, F-431).
func pruefeBudget(d *document.Dokument, liste []document.SchwaerzAngabe, platz int64) error {
	bedarf := int64(0)
	for _, s := range liste {
		r, err := d.PruefeSchwaerzung(s.Bereich, s.Art)
		if err != nil {
			return err
		}
		if bedarf += 4 * int64(r.Dx()) * int64(r.Dy()); bedarf > platz { // r liegt im Bild: kein Überlauf
			return fmt.Errorf("%w: Schwärzungen belegen mehr als %d MiB im Rückgängig-Verlauf", ErrZuGross, platz>>20)
		}
	}
	return nil
}

// eigenerName erkennt die Dateinamen dieses Pakets: Kopie (<64 hex>.json), beiseitegelegte Kopie
// (<64 hex>.<Zeit>.alt.json), Sperre einer Sitzung (<64 hex>.lock) und temporäre Datei beim Schreiben (.schreibt-…).
func eigenerName(n string) bool {
	if strings.HasPrefix(n, ".schreibt-") {
		return true
	}
	if len(n) < 64 || !istHex(n[:64]) {
		return false
	}
	rest := n[64:]
	if rest == ".json" || rest == sperrEndung {
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
