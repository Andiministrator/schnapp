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

// Speichern schreibt ein bearbeitetes Dokument an den Platz einer Aufnahme der Bibliothek zurück (ADR-0020):
// Erstellzeit, Schlagworte, Quelle und Autor bleiben aus der Datei, Bearbeitet wird wahr. sichtbar ist das fertige
// Bild (render.Bild). Danach folgen Index und Vorschau; scheitert nur einer davon, kommt der Eintrag mit ErrIndex
// bzw. ErrVorschau (und ErrVorschauAlt, wenn das alte Vorschaubild bleibt). Eine fremde oder beschädigte Datei an diesem Platz wird nicht überschrieben (ErrFremd) — der
// Aufrufer legt dann mit Ablegen eine neue Aufnahme an.
func (b *Bibliothek) Speichern(rel string, d *document.Dokument, sichtbar image.Image) (Eintrag, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !(Eintrag{Pfad: rel, Breite: 1, Hoehe: 1}).gueltig() {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	voll := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
	st, err := os.Lstat(voll)
	if err != nil {
		return Eintrag{}, err
	}
	if !st.Mode().IsRegular() {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	// Auch die Ordner auf dem Weg dürfen nicht aus der Ablage hinausführen (Monatsordner als Symlink, Kritik A-11).
	if echt, ok := b.Relativ(voll); !ok || echt != rel {
		return Eintrag{}, fmt.Errorf("%w: %q", ErrPfad, rel)
	}
	f, err := os.Open(voll)
	if err != nil {
		return Eintrag{}, err
	}
	g, err := fileformat.LeseArbeitsdatei(f)
	f.Close()
	if err != nil {
		if fileformat.IstBeschaedigt(err) {
			return Eintrag{}, fmt.Errorf("%w: %v", ErrFremd, err)
		}
		return Eintrag{}, err
	}
	if g.Fremd {
		return Eintrag{}, ErrFremd
	}
	m := g.Meta
	m.Bearbeitet = true
	if m.Erstellt.IsZero() {
		m.Erstellt = st.ModTime().UTC().Truncate(time.Second)
	}
	err = schreibeAtomar(voll, func(w io.Writer) error {
		return fileformat.SchreibeArbeitsdatei(w, d, sichtbar, m)
	})
	if err != nil {
		return Eintrag{}, err
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
	return neu, folge
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
