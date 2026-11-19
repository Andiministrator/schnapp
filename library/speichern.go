// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
)

// Relativ liefert den Pfad einer Datei relativ zur Ablage, wenn sie dort liegt (nach Auflösen von Verweisen, damit
// ein Symlink von außen nicht als Bibliotheksaufnahme gilt). Sonst ok = false.
func (b *Bibliothek) Relativ(abs string) (rel string, ok bool) {
	ablage, err := filepath.EvalSymlinks(b.Orte.Ablage)
	if err != nil {
		return "", false
	}
	echt, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	r, err := filepath.Rel(ablage, echt)
	if err != nil || r == "." || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) || filepath.IsAbs(r) {
		return "", false
	}
	rel = filepath.ToSlash(r)
	if !(Eintrag{Pfad: rel, Breite: 1, Hoehe: 1}).gueltig() {
		return "", false
	}
	return rel, true
}

// ErrGeaendert: Die Datei wurde seit dem Öffnen (oder letzten Speichern) von anderer Seite geändert oder ersetzt,
// etwa von einem zweiten Editor (F-110). Speichern überschreibt sie dann nicht; der Aufrufer legt eine neue Aufnahme
// an.
var ErrGeaendert = errors.New("bibliothek: Datei seit dem Öffnen geändert")

// Stand kennzeichnet den Zustand einer Datei auf dem Datenträger: Gerät, Inode, Größe und Änderungszeit (ns). Jedes
// Speichern der Bibliothek ersetzt die Datei per Umbenennen und damit den Inode; ein Werkzeug, das an Ort und Stelle
// schreibt, ändert Zeit oder Größe. Der Nullwert passt zu keiner Datei.
type Stand struct {
	geraet, inode uint64
	groesse       int64
	zeit          int64
}

// standAus liest den Stand aus einer Dateiangabe.
func standAus(fi os.FileInfo) Stand {
	s := Stand{groesse: fi.Size(), zeit: fi.ModTime().UnixNano()}
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		s.geraet, s.inode = uint64(sys.Dev), sys.Ino
	}
	return s
}

// StandVon liefert den Stand der Datei pfad (Verweise werden aufgelöst). Der Editor holt ihn vor dem Lesen, damit
// eine Änderung zwischen Lesen und Merken nicht unbemerkt bleibt.
func StandVon(pfad string) (Stand, error) {
	fi, err := os.Stat(pfad)
	if err != nil {
		return Stand{}, err
	}
	return standAus(fi), nil
}

// Speichern schreibt ein bearbeitetes Dokument an den Platz einer Aufnahme der Bibliothek zurück (ADR-0020):
// Erstellzeit, Schlagworte, Quelle und Autor bleiben aus der Datei, Bearbeitet wird wahr. sichtbar ist das fertige
// Bild (render.Bild). erwartet ist der Stand der Datei beim Öffnen oder letzten Speichern; weicht er ab, wird nicht
// überschrieben (ErrGeaendert, F-110). Zurück kommt der neue Stand für das nächste Speichern. Danach folgen Index und
// Vorschau; scheitert nur einer davon, kommt der Eintrag mit ErrIndex bzw. ErrVorschau (und ErrVorschauAlt, wenn das
// alte Vorschaubild bleibt). Eine fremde oder beschädigte Datei an diesem Platz wird nicht überschrieben (ErrFremd) —
// der Aufrufer legt dann mit Ablegen eine neue Aufnahme an.
//
// Der Vergleich ist race-arm, nicht race-frei: Zwischen Prüfen und Umbenennen bleiben Millisekunden, in denen ein
// anderer Prozess schreiben könnte (keine Sperrdateien; der Mutex gilt nur in diesem Prozess).
func (b *Bibliothek) Speichern(rel string, erwartet Stand, d *document.Dokument, sichtbar image.Image) (Eintrag, Stand, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !(Eintrag{Pfad: rel, Breite: 1, Hoehe: 1}).gueltig() {
		return Eintrag{}, Stand{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
	st, err := os.Lstat(voll)
	if err != nil {
		return Eintrag{}, Stand{}, err
	}
	if !st.Mode().IsRegular() {
		return Eintrag{}, Stand{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	// Auch die Ordner auf dem Weg dürfen nicht aus der Ablage hinausführen (Monatsordner als Symlink, Kritik A-11).
	if echt, ok := b.Relativ(voll); !ok || echt != rel {
		return Eintrag{}, Stand{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	if standAus(st) != erwartet {
		return Eintrag{}, Stand{}, fmt.Errorf("%w: %q", ErrGeaendert, rel)
	}
	f, err := os.Open(voll)
	if err != nil {
		return Eintrag{}, Stand{}, err
	}
	g, err := fileformat.LeseArbeitsdatei(f)
	f.Close()
	if err != nil {
		if fileformat.IstBeschaedigt(err) {
			return Eintrag{}, Stand{}, fmt.Errorf("%w: %v", ErrFremd, err)
		}
		return Eintrag{}, Stand{}, err
	}
	if g.Fremd {
		return Eintrag{}, Stand{}, ErrFremd
	}
	m := g.Meta
	m.Bearbeitet = true
	if m.Erstellt.IsZero() {
		m.Erstellt = st.ModTime().UTC().Truncate(time.Second)
	}
	err = schreibeAtomar(voll, func(w io.Writer) error {
		if err := fileformat.SchreibeArbeitsdatei(w, d, sichtbar, m); err != nil {
			return err
		}
		// Das Kodieren dauert: kurz vor dem Ersetzen noch einmal vergleichen, um das Fenster klein zu halten.
		if fi, err := os.Lstat(voll); err != nil || standAus(fi) != erwartet {
			return fmt.Errorf("%w: %q", ErrGeaendert, rel)
		}
		return nil
	})
	if err != nil {
		return Eintrag{}, Stand{}, err
	}
	// Neuer Stand für das nächste Speichern. Lässt er sich nicht lesen, passt der Nullwert zu nichts: Das nächste
	// Speichern weicht dann auf eine neue Aufnahme aus, statt blind zu überschreiben.
	var neuStand Stand
	if fi, err := os.Lstat(voll); err == nil {
		neuStand = standAus(fi)
	}
	gr := d.Groesse()
	neu := Eintrag{Pfad: rel, Erstellt: m.Erstellt.UTC(), Breite: gr.X, Hoehe: gr.Y, Schlagworte: m.Schlagworte,
		Quelle: m.Quelle, Bearbeitet: true}
	var folge error
	if err := b.ersetzeImIndex(neu); err != nil {
		folge = fmt.Errorf("%w: %v", ErrIndex, err)
	}
	if err := b.erneuereVorschau(rel, sichtbar); err != nil {
		folge = errors.Join(folge, err)
	}
	return neu, neuStand, folge
}

// ersetzeImIndex setzt den Eintrag mit demselben Pfad (oder hängt ihn an) und schreibt den Index.
func (b *Bibliothek) ersetzeImIndex(neu Eintrag) error {
	idx, err := b.lade()
	if err != nil {
		return err
	}
	ersetzt := false
	for i := range idx.Eintraege {
		if idx.Eintraege[i].Pfad == neu.Pfad {
			idx.Eintraege[i] = neu
			ersetzt = true
		}
	}
	if !ersetzt {
		idx.Eintraege = append(idx.Eintraege, neu)
		sortiere(idx.Eintraege)
	}
	return b.schreibeIndex(idx.Eintraege)
}
