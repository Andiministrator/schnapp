// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/fileformat"
)

// ADR-0022 Punkt 4 (F-126): Entfernen löscht Datei, Vorschau und Index-Eintrag endgültig (kein Papierkorb) und meldet
// es; fehlt die Datei schon, meldet es „nicht gelöscht“ ohne Fehler (F-167).
func TestEntfernen(t *testing.T) {
	b, heim := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Fatalf("Vorschau fehlt vor dem Test: %v", err)
	}
	if weg, err := b.Entfernen(e.Pfad, standIn(t, b, e.Pfad)); err != nil || !weg {
		t.Fatalf("Entfernen: %v, %v", weg, err)
	}
	for _, p := range []string{abs, b.VorschauPfad(e.Pfad)} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s noch da: %v", filepath.Base(p), err)
		}
	}
	idx, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Eintraege) != 0 {
		t.Errorf("Index %+v", idx.Eintraege)
	}
	// Nichts im Papierkorb: Dort läge der ungeschwärzte Stand weiter.
	if n, _ := filepath.Glob(filepath.Join(heim, ".local", "share", "Trash", "files", "*")); len(n) != 0 {
		t.Errorf("Papierkorb: %v", n)
	}
	if weg, err := b.Entfernen(e.Pfad, Stand{}); err != nil || weg {
		t.Errorf("schon fehlende Datei: %v, %v (Soll: false, nil)", weg, err)
	}
}

// Eine inzwischen anders beschriebene Datei bleibt stehen (ErrGeaendert), ebenso ein Symlink am Ziel; ein Pfad
// außerhalb der Ablage ist ErrPfad. In keinem Fall werden Vorschau oder Index angefasst.
func TestEntfernenNichtFremdes(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	alt := standIn(t, b, e.Pfad)
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if err := os.WriteFile(abs, []byte("anders"), 0o600); err != nil {
		t.Fatal(err)
	}
	if weg, err := b.Entfernen(e.Pfad, alt); weg || !errors.Is(err, ErrGeaendert) {
		t.Errorf("geänderte Datei: %v, %v, want ErrGeaendert", weg, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("geänderte Datei entfernt: %v", err)
	}
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Errorf("Vorschau trotz ErrGeaendert entfernt: %v", err)
	}
	ziel := filepath.Join(t.TempDir(), "fremd.png")
	if err := os.WriteFile(ziel, []byte("fremd"), 0o600); err != nil {
		t.Fatal(err)
	}
	e2 := ablegen(t, b, fileformat.Metadaten{})
	abs2 := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e2.Pfad))
	vorher := standIn(t, b, e2.Pfad)
	if err := os.Remove(abs2); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ziel, abs2); err != nil {
		t.Fatal(err)
	}
	if weg, err := b.Entfernen(e2.Pfad, vorher); weg || !errors.Is(err, ErrGeaendert) {
		t.Errorf("Symlink: %v, %v, want ErrGeaendert", weg, err)
	}
	if _, err := os.Stat(ziel); err != nil {
		t.Errorf("Ziel des Symlinks angefasst: %v", err)
	}
	if _, err := b.Entfernen("../ausserhalb.png", alt); !errors.Is(err, ErrPfad) {
		t.Errorf("Pfad außerhalb: %v, want ErrPfad", err)
	}
}

// F-164: Scheitert nur das Löschen der Vorschau, ist die Datei weg (geloescht = true) und der Fehler ErrVorschauAlt.
// Scheitert das Löschen der Datei, bleibt alles stehen und der Fehler ist kein ErrVorschauAlt.
func TestEntfernenTeilfehler(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{})
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	vorschau := b.VorschauPfad(e.Pfad)
	alt := entferneDatei
	t.Cleanup(func() { entferneDatei = alt })
	entferneDatei = func(p string) error {
		if p == vorschau {
			return os.ErrPermission
		}
		return alt(p)
	}
	weg, err := b.Entfernen(e.Pfad, standIn(t, b, e.Pfad))
	if !weg || !errors.Is(err, ErrVorschauAlt) {
		t.Errorf("Vorschau bleibt: %v, %v (Soll: true, ErrVorschauAlt)", weg, err)
	}
	if _, err := os.Stat(abs); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Datei noch da: %v", err)
	}

	e2 := ablegen(t, b, fileformat.Metadaten{})
	abs2 := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e2.Pfad))
	entferneDatei = func(p string) error {
		if p == abs2 {
			return os.ErrPermission
		}
		return alt(p)
	}
	weg, err = b.Entfernen(e2.Pfad, standIn(t, b, e2.Pfad))
	if weg || err == nil || errors.Is(err, ErrVorschauAlt) {
		t.Errorf("Datei bleibt: %v, %v (Soll: false, Fehler ohne ErrVorschauAlt)", weg, err)
	}
	if _, err := os.Stat(b.VorschauPfad(e2.Pfad)); err != nil {
		t.Errorf("Vorschau entfernt, obwohl die Datei blieb: %v", err)
	}
}
