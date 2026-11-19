// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"errors"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
)

// Speichern schreibt zurück: Objekte in der Datei, Bearbeitet in Datei und Index, Schlagworte und Erstellzeit
// bleiben, kein zweiter Eintrag.
func TestSpeichernSchreibtZurueck(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{Schlagworte: []string{"Rechnung"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(9, 9)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	neu, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, d.Ansichtsbild())
	if err != nil {
		t.Fatal(err)
	}
	if !neu.Bearbeitet || !neu.Erstellt.Equal(e.Erstellt) || len(neu.Schlagworte) != 1 || neu.Pfad != e.Pfad {
		t.Errorf("Eintrag %+v, vorher %+v", neu, e)
	}
	g, err := fileformat.Oeffne(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)))
	if err != nil {
		t.Fatal(err)
	}
	if g.Fremd || len(g.Dokument.Objekte()) != 1 || !g.Meta.Bearbeitet || len(g.Meta.Schlagworte) != 1 {
		t.Errorf("Datei: Fremd %v, Objekte %d, Meta %+v", g.Fremd, len(g.Dokument.Objekte()), g.Meta)
	}
	idx, err := b.Laden()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Eintraege) != 1 || !idx.Eintraege[0].Bearbeitet {
		t.Errorf("Index %+v", idx.Eintraege)
	}
}

// Ein fremdes PNG in der Ablage wird nicht überschrieben.
func TestSpeichernUeberschreibtKeinFremdesBild(t *testing.T) {
	b, _ := neueBibliothek(t)
	ordner := filepath.Join(b.Orte.Ablage, "2026-11")
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(ordner, "fremd.png")
	f, err := os.Create(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, muster(8, 8)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	vorher, _ := os.ReadFile(pfad)
	d := neuesDokument(t, 8, 8)
	if _, _, err := b.Speichern("2026-11/fremd.png", standIn(t, b, "2026-11/fremd.png"), d, d.Ansichtsbild()); !errors.Is(err, ErrFremd) {
		t.Fatalf("Fehler %v, Soll ErrFremd", err)
	}
	if nachher, _ := os.ReadFile(pfad); string(nachher) != string(vorher) {
		t.Error("fremde Datei verändert")
	}
	if _, _, err := b.Speichern("../aussen.png", Stand{}, d, d.Ansichtsbild()); !errors.Is(err, ErrPfad) {
		t.Errorf("Pfad außerhalb: %v", err)
	}
}

func TestRelativ(t *testing.T) {
	b, heim := neueBibliothek(t)
	d := neuesDokument(t, 4, 4)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	if rel, ok := b.Relativ(abs); !ok || rel != e.Pfad {
		t.Errorf("in der Ablage: %q %v", rel, ok)
	}
	aussen := filepath.Join(heim, "aussen.png")
	if err := os.WriteFile(aussen, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Relativ(aussen); ok {
		t.Error("Datei außerhalb gilt als Bibliothek")
	}
	// Ein Verweis in der Ablage, der nach außen zeigt, gilt nicht.
	verweis := filepath.Join(b.Orte.Ablage, "2026-11", "verweis.png")
	if err := os.Symlink(aussen, verweis); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Relativ(verweis); ok {
		t.Error("Verweis nach außen gilt als Bibliothek")
	}
	if _, ok := b.Relativ(b.Orte.Ablage); ok {
		t.Error("Ablage selbst gilt als Aufnahme")
	}
}

// Ein Monatsordner, der als Verweis aus der Ablage hinausführt, wird nicht beschrieben (Kritik A-11).
func TestSpeichernNichtDurchOrdnerVerweis(t *testing.T) {
	b, heim := neueBibliothek(t)
	aussen := filepath.Join(heim, "aussen")
	if err := os.MkdirAll(aussen, 0o700); err != nil {
		t.Fatal(err)
	}
	d := neuesDokument(t, 4, 4)
	var buf bytes.Buffer
	if err := fileformat.SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aussen, "x.png"), buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b.Orte.Ablage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(aussen, filepath.Join(b.Orte.Ablage, "2026-11")); err != nil {
		t.Fatal(err)
	}
	vorher, _ := os.ReadFile(filepath.Join(aussen, "x.png"))
	if _, _, err := b.Speichern("2026-11/x.png", Stand{}, d, d.Ansichtsbild()); !errors.Is(err, ErrPfad) {
		t.Fatalf("Fehler %v, Soll ErrPfad", err)
	}
	if nachher, _ := os.ReadFile(filepath.Join(aussen, "x.png")); !bytes.Equal(nachher, vorher) {
		t.Error("Datei außerhalb der Ablage verändert")
	}
}
