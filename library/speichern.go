// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/internal/speicherschritt"
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
	g, err := fileformat.LeseArbeitsdatei(speicherschritt.Leser(f)) // Fortschritt je gelesenem Stück (F-298)
	f.Close()
	speicherschritt.Marke("alte-datei-gelesen")
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
	// Liegen gebliebene temporäre Dateien im Ordner vorher räumen (F-274): Ein Rest aus dem Link-Weg ist ein zweiter
	// Name dieser Aufnahme und zeigte nach dem Ersetzen (neuer Inode) weiter den alten, womöglich ungeschwärzten
	// Stand (Kernprinzip 2; ADR-0022 Nachtrag 2). Reste mit demselben Inode wie diese Datei gehen ohne Altersgrenze
	// (Z-1); junge oder zukunftsdatierte andere bleiben und werden gemeldet. Scheitert das Entfernen, wird gespeichert
	// und mit ErrTempRest gewarnt.
	_, restFehler := raeumeTempReste(filepath.Dir(voll), time.Now(), true, st)
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
	speicherschritt.Marke("ersetzt")
	// Neuer Stand für das nächste Speichern. Lässt er sich nicht lesen, passt der Nullwert zu nichts: Das nächste
	// Speichern weicht dann auf eine neue Aufnahme aus, statt blind zu überschreiben.
	var neuStand Stand
	if fi, err := os.Lstat(voll); err == nil {
		neuStand = standAus(fi)
	}
	gr := d.Groesse()
	neu := Eintrag{Pfad: rel, Erstellt: m.Erstellt.UTC(), Breite: gr.X, Hoehe: gr.Y, Schlagworte: m.Schlagworte,
		Quelle: m.Quelle, Bearbeitet: true}
	folge := restFehler
	if err := b.ersetzeImIndex(neu); err != nil {
		folge = errors.Join(folge, fmt.Errorf("%w: %v", ErrIndex, err))
	}
	speicherschritt.Marke("index")
	if err := b.erneuereVorschau(rel, sichtbar); err != nil {
		folge = errors.Join(folge, err)
	}
	speicherschritt.Marke("vorschau")
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

// Entfernen löscht die Aufnahme rel endgültig samt Vorschau und Index-Eintrag — nicht in den Papierkorb, denn dort läge
// der Inhalt weiter. Gedacht für eine Datei, die nach dem Schwärzen nicht mehr aktualisiert werden konnte und noch den
// ungeschwärzten Stand zeigt (Kernprinzip 2; ADR-0022 Nachtrag 2, F-126).
//
// geloescht meldet, ob diese Datei hier entfernt wurde; fehlte sie schon (verschoben, im Papierkorb), ist es false
// ohne Fehler — der Aufrufer darf dann nicht „entfernt“ melden (F-167). Fehler:
//   - ErrGeaendert: Die Datei hat nicht mehr den Stand erwartet (von anderer Seite geändert); sie bleibt, wie alles
//     andere.
//   - ErrVorschauAlt (mit ErrVorschau): Die Datei ist weg, aber das Vorschaubild ließ sich nicht entfernen und zeigt
//     noch den alten Inhalt (F-164).
//   - ErrTempRest: Eine temporäre Kopie derselben Aufnahme (gleicher Inode) im Ordner ließ sich nicht entfernen; sie
//     zeigt noch den alten Inhalt (Z-2). Reste mit gleichem Inode räumt Entfernen ohne Altersgrenze, übrige alte auch.
//   - ErrIndex: Datei und Vorschau sind weg, nur der Index ist nicht nachgeführt (Abgleich holt das nach).
//   - sonst: Die Datei ließ sich nicht entfernen.
func (b *Bibliothek) Entfernen(rel string, erwartet Stand) (geloescht bool, fehler error) {
	b.mu.Lock() // liest und schreibt den Index wie Speichern (F-165)
	defer b.mu.Unlock()
	if !(Eintrag{Pfad: rel, Breite: 1, Hoehe: 1}).gueltig() {
		return false, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
	fi, err := os.Lstat(abs)
	switch {
	case err == nil:
		if !fi.Mode().IsRegular() || standAus(fi) != erwartet {
			return false, ErrGeaendert
		}
		if err := entferneDatei(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
		geloescht = true
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	// Temporäre Kopien derselben Aufnahme räumen (Z-2): Sie wären nach dem Löschen der Datei ihr letzter Name und
	// zeigten weiter den ungeschwärzten Stand. Auch ohne die Datei räumt das alte Reste (nur nach Alter).
	if _, rest := raeumeTempReste(filepath.Dir(abs), time.Now(), true, fi); rest != nil {
		fehler = rest
	}
	if err := b.entferneVorschau(rel); err != nil {
		fehler = errors.Join(fehler, fmt.Errorf("%w: %w: %v", ErrVorschau, ErrVorschauAlt, err))
	}
	idx, err := b.lade()
	if err == nil {
		rest := idx.Eintraege[:0]
		for _, e := range idx.Eintraege {
			if e.Pfad != rel {
				rest = append(rest, e)
			}
		}
		err = b.schreibeIndex(rest)
	}
	if err != nil {
		fehler = errors.Join(fehler, fmt.Errorf("%w: %v", ErrIndex, err))
	}
	return geloescht, fehler
}
