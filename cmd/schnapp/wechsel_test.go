// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
)

// schreibeEinfachesPNG legt ein einfaches PNG (keine Arbeitsdatei) an.
func schreibeEinfachesPNG(t *testing.T, ordner, name string) string {
	t.Helper()
	p := filepath.Join(ordner, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

// ADR-0036 Punkt 5: oeffneImEditor gibt dem Editor die Anbindung an die Bibliothek mit; Pfad nennt die Datei, die den
// Stand trägt, auch nach dem Speichern als neue Aufnahme.
func TestEditorBekommtAnbindung(t *testing.T) {
	_, aussen, g := rettungsLauf(t)
	if err := oeffneImEditor(aussen, true, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	if g.bib == nil || g.bib.Bibliothek == nil || g.bib.Oeffnen == nil {
		t.Fatal("keine Anbindung")
	}
	abs, _ := filepath.Abs(aussen)
	if g.geoeffnet.Pfad() != abs || g.geoeffnet.Titel != filepath.Base(abs) || !g.geoeffnet.OhneGrenze {
		t.Errorf("Geöffnet: Pfad %q, Titel %q, OhneGrenze %v", g.geoeffnet.Pfad(), g.geoeffnet.Titel, g.geoeffnet.OhneGrenze)
	}
	if _, err := g.dok.Hinzufuegen(document.Text{Ort: document.Pt(2, 12), Inhalt: "x", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.speichern(g.dok); err != nil {
		t.Fatal(err)
	}
	// Eine Datei von außen kommt als neue Aufnahme in die Bibliothek: Pfad zeigt jetzt dorthin.
	rel, ok := g.bib.Bibliothek.Relativ(g.geoeffnet.Pfad())
	if !ok || rel == "" {
		t.Errorf("Pfad nach dem Speichern: %q", g.geoeffnet.Pfad())
	}
}

// Die Ladefunktion des Wechsels lädt wie `schnapp DATEI` ohne --ignore-limit: ein einfaches Bild bekommt die
// Kantenvorgabe, eine Rettungskopie, Titel und Pfad; Fehler kommen übersetzt.
func TestWechselLadefunktion(t *testing.T) {
	_, aussen, g := rettungsLauf(t)
	if err := oeffneImEditor(aussen, true, document.KanteWelle); err != nil {
		t.Fatal(err)
	}
	einfach := schreibeEinfachesPNG(t, t.TempDir(), "einfach.png")
	o, err := g.bib.Oeffnen(einfach)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if o.Rettung != nil {
			o.Rettung.Ende()
		}
	})
	if o.Dokument.Kante() != document.KanteWelle || o.Dokument.KannRueckgaengig() {
		t.Errorf("Kante %v, Rückgängig %v", o.Dokument.Kante(), o.Dokument.KannRueckgaengig())
	}
	if o.Titel != "einfach.png" || o.Pfad() != einfach || o.OhneGrenze || o.Rettung == nil || o.Speichern == nil {
		t.Errorf("Geöffnet: %+v, Pfad %q", o, o.Pfad())
	}
	// Eine Arbeitsdatei behält ihre Kante.
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 30, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	arbeit := schreibeDatei(t, t.TempDir(), "arbeit.png", d)
	o2, err := g.bib.Oeffnen(arbeit)
	if err != nil {
		t.Fatal(err)
	}
	if o2.Rettung != nil {
		o2.Rettung.Ende()
	}
	if o2.Dokument.Kante() != document.KanteZacken {
		t.Errorf("Arbeitsdatei: Kante %v", o2.Dokument.Kante())
	}
	if _, err := g.bib.Oeffnen(filepath.Join(t.TempDir(), "fehlt.png")); err == nil || err.Error() != "The file does not exist." {
		t.Errorf("fehlende Datei: %v", err)
	}
}

// Der Wechsel prüft das Rechenbudget wie `schnapp DATEI` ohne --ignore-limit (ladeFuerEditor mit Budget).
func TestWechselMitBudget(t *testing.T) {
	_, aussen, g := rettungsLauf(t)
	if err := oeffneImEditor(aussen, true, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	alt := ladeFuer
	var ohneGrenze []bool
	ladeFuer = func(p string, og bool) (string, *fileformat.Gelesen, library.Stand, error) {
		ohneGrenze = append(ohneGrenze, og)
		return "", nil, library.Stand{}, render.ErrZuAufwendig
	}
	t.Cleanup(func() { ladeFuer = alt })
	_, err := g.bib.Oeffnen(aussen)
	if len(ohneGrenze) != 1 || ohneGrenze[0] {
		t.Errorf("geladen mit ohneGrenze %v", ohneGrenze)
	}
	if err == nil || errors.Is(err, render.ErrZuAufwendig) && err.Error() == render.ErrZuAufwendig.Error() {
		t.Errorf("Fehler nicht übersetzt: %v", err)
	}
}

// bearbeiteNeu gibt die Kantenvorgabe aus der Aufnahme an die Ladefunktion weiter.
func TestBearbeiteNeuAnbindung(t *testing.T) {
	_, aussen, g := rettungsLauf(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 30, 20)))
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(aussen)
	if err := bearbeiteNeu(abgelegt{pfad: abs, d: d, kante: document.KanteZacken}); err != nil {
		t.Fatal(err)
	}
	if g.bib == nil || g.geoeffnet.Pfad() != abs || g.geoeffnet.OhneGrenze {
		t.Fatalf("Anbindung %v, Pfad %q", g.bib, g.geoeffnet.Pfad())
	}
	o, err := g.bib.Oeffnen(schreibeEinfachesPNG(t, t.TempDir(), "neu.png"))
	if err != nil {
		t.Fatal(err)
	}
	if o.Rettung != nil {
		o.Rettung.Ende()
	}
	if o.Dokument.Kante() != document.KanteZacken {
		t.Errorf("Kante %v", o.Dokument.Kante())
	}
}
