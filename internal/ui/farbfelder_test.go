// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Eigene Farbfelder (ADR-0021 Nachtrag 24, Karte #364).

import (
	"image"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// farbfelderDatei leitet das Speichern der Farbfelder in eine eigene Datei um und liefert ihren Pfad.
func farbfelderDatei(t *testing.T) string {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "colors.ini")
	alt := aendereFarben
	aendereFarben = func(aend func([]color.NRGBA, bool) []color.NRGBA) ([]color.NRGBA, error) {
		return settings.AendereFarbenDatei(pfad, aend)
	}
	t.Cleanup(func() { aendereFarben = alt })
	return pfad
}

func TestFarbfelderKlickPlusEntfernen(t *testing.T) {
	pfad := farbfelderDatei(t)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	f := &l.e.seite.ff
	if !slices.Equal(f.farben, canvas.StandardFarbfelder) {
		t.Fatalf("ohne Datei: %v, Soll die eingebauten", f.farben)
	}
	// Klick setzt die Farbe des gewählten Texts.
	l.klick(mittenPunkt(f.rahmen[4])) // Weiß
	if got := l.e.s.AktuelleFarbe(); got != canvas.StandardFarbfelder[4] {
		t.Fatalf("Klick auf Weiß: %v", got)
	}
	// „+“ nimmt eine neue Farbe auf und schreibt die Datei.
	neu := color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xFF}
	l.e.s.SetzeHex(canvas.HexText(neu))
	l.setle()
	if !l.e.kannFarbfeldHinzu() {
		t.Fatal("„+“ grau, obwohl die Farbe neu ist")
	}
	l.klick(mittenPunkt(f.plusRahmen))
	if n := len(f.farben); n != len(canvas.StandardFarbfelder)+1 || f.farben[n-1] != neu {
		t.Fatalf("nach „+“: %v", f.farben)
	}
	if gelesen, da, _ := settings.LadeFarbenDatei(pfad); !da || !slices.Equal(gelesen, f.farben) {
		t.Errorf("Datei nach „+“: %v (da %v)", gelesen, da)
	}
	if l.e.kannFarbfeldHinzu() {
		t.Error("„+“ an, obwohl die Farbe schon ein Feld ist")
	}
	// Rechtsklick entfernt.
	l.rechtsklick(mittenPunkt(f.rahmen[0]))
	if slices.Contains(f.farben, canvas.StandardFarbfelder[0]) {
		t.Errorf("nach Rechtsklick noch da: %v", f.farben)
	}
	// Entf am Feld mit Fokus entfernt, ohne das gewählte Objekt zu löschen.
	n := len(l.e.s.D.Objekte())
	weg := f.farben[0]
	l.fokusiere(&f.knoepfe[0])
	l.taste(key.NameDeleteForward, 0)
	if slices.Contains(f.farben, weg) || len(l.e.s.D.Objekte()) != n {
		t.Errorf("nach Entf: %v, Objekte %d (vorher %d)", f.farben, len(l.e.s.D.Objekte()), n)
	}
	if gelesen, _, _ := settings.LadeFarbenDatei(pfad); !slices.Equal(gelesen, f.farben) {
		t.Errorf("Datei nach dem Entfernen: %v, Leiste %v", gelesen, f.farben)
	}
}

func TestFarbfeldHinweise(t *testing.T) {
	farbfelderDatei(t)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	f := &l.e.seite.ff
	l.fokusiere(&f.knoepfe[1])
	if !strings.Contains(l.e.hinweis(), "Right-click or Del") {
		t.Errorf("Hinweis am Farbfeld: %q", l.e.hinweis())
	}
	l.e.s.SetzeFarbe(canvas.StandardFarbfelder[1])
	l.fokusiere(&f.plus)
	if h := l.e.hinweis(); !strings.Contains(h, "already") {
		t.Errorf("Hinweis an „+“ bei vorhandener Farbe: %q", h)
	}
}

// Ein Fehler beim Schreiben lässt die Farbfelder, wie sie sind, und sagt es in der Statuszeile.
func TestFarbfelderFehlerBleibt(t *testing.T) {
	alt := aendereFarben
	aendereFarben = func(func([]color.NRGBA, bool) []color.NRGBA) ([]color.NRGBA, error) {
		return nil, settings.ErrFarbenGesperrt
	}
	t.Cleanup(func() { aendereFarben = alt })
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	f := &l.e.seite.ff
	vorher := slices.Clone(f.liste())
	l.rechtsklick(mittenPunkt(f.rahmen[0]))
	if !slices.Equal(f.farben, vorher) || !l.e.meldung.Fehler {
		t.Errorf("nach Fehler: %v, Meldung %+v", f.farben, l.e.meldung)
	}
}

// Ohne colors.ini gelten die eingebauten Farbfelder; eine Datei ohne Zeile heißt: keine.
func TestFarbfelderLaden(t *testing.T) {
	alt := ladeFarben
	t.Cleanup(func() { ladeFarben = alt })
	e := testEditor(t)
	ladeFarben = func() ([]color.NRGBA, bool, []string) { return nil, false, nil }
	e.farbfelderLaden()
	if !slices.Equal(e.seite.ff.farben, canvas.StandardFarbfelder) || !e.seite.ff.geladen {
		t.Errorf("ohne Datei: %v", e.seite.ff.farben)
	}
	ladeFarben = func() ([]color.NRGBA, bool, []string) { return nil, true, nil }
	e.farbfelderLaden()
	if len(e.seite.ff.liste()) != 0 {
		t.Errorf("leere Datei: %v, Soll keine Farbfelder", e.seite.ff.farben)
	}
}
