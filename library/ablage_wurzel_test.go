// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/fileformat"
)

// Tests zu Karte #114: Ablage als Symlink oder leerer Einhängepunkt (F-270, F-271) und liegen gebliebene
// temporäre Dateien (F-274). Alles in eigenen Testverzeichnissen.

// symlinkAblage legt die Ablage als Symlink auf einen Ordner „auf einer anderen Platte“ an.
func symlinkAblage(t *testing.T, b *Bibliothek, heim string) {
	t.Helper()
	echt := filepath.Join(heim, "platte", "schnapp")
	if err := os.MkdirAll(echt, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(b.Orte.Ablage), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(echt, b.Orte.Ablage); err != nil {
		t.Fatal(err)
	}
}

// F-270: Ein leerer Einhängepunkt (ausgehängtes Laufwerk) bei gefülltem Cache ergibt einen Fehler, keine Liste.
func TestVerwaisteVorschauenLeereAblageIstFehler(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	weg := b.Orte.Ablage + ".eingehaengt"
	if err := os.Rename(b.Orte.Ablage, weg); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(b.Orte.Ablage, 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := b.VerwaisteVorschauen()
	if !errors.Is(err, ErrAblageLeer) || len(l) != 0 {
		t.Fatalf("Liste %v, Fehler %v (Soll: ErrAblageLeer, keine Liste)", l, err)
	}
	if _, err := os.Stat(b.VorschauPfad(e.Pfad)); err != nil {
		t.Errorf("Vorschaubild angefasst: %v", err)
	}
}

// Leere Ablage und leerer Cache: nichts verwaist, kein Fehler.
func TestVerwaisteVorschauenLeerUndLeerIstKeinFehler(t *testing.T) {
	b, _ := neueBibliothek(t)
	if err := os.MkdirAll(b.Orte.Ablage, 0o700); err != nil {
		t.Fatal(err)
	}
	if l, err := b.VerwaisteVorschauen(); err != nil || len(l) != 0 {
		t.Errorf("Liste %v, Fehler %v", l, err)
	}
}

// F-270: Ist die Ablage ein Symlink, gilt das Vorschaubild der vorhandenen Aufnahme nicht als verwaist; ein echtes
// Waisenbild wird weiter gefunden.
func TestVerwaisteVorschauenSymlinkAblage(t *testing.T) {
	b, heim := neueBibliothek(t)
	symlinkAblage(t, b, heim)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	l, err := b.VerwaisteVorschauen()
	if err != nil || len(l) != 0 {
		t.Fatalf("Liste %v, Fehler %v (Soll: leer)", l, err)
	}
	waise := b.VorschauPfad("2026-11/weg.png")
	schreibe(t, waise, "x")
	l, err = b.VerwaisteVorschauen()
	if err != nil || len(l) != 1 || l[0] != waise {
		t.Errorf("Liste %v, Fehler %v (Soll: nur %s, nicht die Vorschau von %s)", l, err, waise, e.Pfad)
	}
}

// F-271: Neuaufbau bei Symlink-Ablage nimmt die Aufnahmen auf (vorher: leerer Index ohne Fehler).
func TestNeuaufbauSymlinkAblage(t *testing.T) {
	b, heim := neueBibliothek(t)
	symlinkAblage(t, b, heim)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	if err := os.Remove(b.Orte.Index); err != nil {
		t.Fatal(err)
	}
	erg, err := b.Neuaufbau()
	if err != nil || erg.Aufgenommen != 1 {
		t.Fatalf("Neuaufbau %+v %v (Soll: 1 Aufnahme)", erg, err)
	}
	g, err := b.Laden()
	if err != nil || len(g.Eintraege) != 1 || g.Eintraege[0].Pfad != e.Pfad {
		t.Errorf("Index %+v %v", g.Eintraege, err)
	}
}
