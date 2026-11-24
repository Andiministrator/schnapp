// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// ADR-0036 Punkt 6: Fehlt das Vorschaubild einer Aufnahme, schreibt ErzeugeVorschau es aus der Datei neu; ein Pfad aus
// der Ablage hinaus und ein Ordner statt einer Datei bleiben ohne Wirkung.
func TestErzeugeVorschau(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 400, 200)
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	if err != nil {
		t.Fatal(err)
	}
	vorschau := b.VorschauPfad(e.Pfad)
	if err := os.Remove(vorschau); err != nil {
		t.Fatal(err)
	}
	b.ErzeugeVorschau(e.Pfad)
	f, err := os.Open(vorschau)
	if err != nil {
		t.Fatalf("Vorschau nicht erzeugt: %v", err)
	}
	f.Close()

	// Aus der Ablage hinaus: nichts schreiben, auch wenn dort eine Arbeitsdatei läge.
	aussen := filepath.Join(filepath.Dir(b.Orte.Ablage), "aussen.png")
	if err := os.Link(filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad)), aussen); err != nil {
		t.Fatal(err)
	}
	b.ErzeugeVorschau("../aussen.png")
	if _, err := os.Stat(b.VorschauPfad("../aussen.png")); err == nil {
		t.Error("Vorschau für einen Pfad außerhalb der Ablage geschrieben")
	}
	// Ein Ordner mit Endung .png: nichts.
	ordner := filepath.Join(b.Orte.Ablage, "2026-11", "ordner.png")
	if err := os.Mkdir(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	b.ErzeugeVorschau("2026-11/ordner.png")
	if _, err := os.Stat(b.VorschauPfad("2026-11/ordner.png")); err == nil {
		t.Error("Vorschau für einen Ordner geschrieben")
	}
}
