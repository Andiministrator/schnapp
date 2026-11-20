// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// ADR-0026 Punkt 4: „Gestauchtes zurückholen“ hebt alle Streifen im Zuschnitt auf (mit und ohne Naht), lässt
// Streifen außerhalb und den Zuschnitt stehen und ist ein Rückgängig-Schritt.
func TestHoleGestauchtesZurueck(t *testing.T) {
	s := randSitzung(t)
	if s.KannZurueckholen() || s.HoleGestauchtesZurueck() {
		t.Fatal("ohne Streifen an")
	}
	aussen := document.Streifen{Von: 5, Bis: 15} // Spalten links vom Zuschnitt
	st := []document.Streifen{{Zeilen: true, Von: 50, Bis: 60}, {Zeilen: true, Von: 180, Bis: 200}, aussen,
		{Von: 100, Bis: 120}}
	if err := s.D.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	if err := s.D.SetzeZuschnitt(image.Rect(20, 0, 200, 200)); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	vor := s.D.Stand()
	if !s.KannZurueckholen() || !s.HoleGestauchtesZurueck() {
		t.Fatal("nicht zurückgeholt")
	}
	if got := s.D.Streifen(); len(got) != 1 || got[0] != aussen {
		t.Errorf("Streifen %v, Soll nur %v", got, aussen)
	}
	if z := s.D.Zuschnitt(); z != image.Rect(20, 0, 200, 200) {
		t.Errorf("Zuschnitt %v geändert", z)
	}
	if s.KannZurueckholen() {
		t.Error("bleibt an, obwohl nichts mehr verborgen ist")
	}
	if !s.Rueckgaengig() || s.D.Stand() != vor || len(s.D.Streifen()) != 4 {
		t.Errorf("Rückgängig: Streifen %v", s.D.Streifen())
	}
}

// Löschen und Text bearbeiten sind nur mit passender Auswahl an (Menüeinträge, ADR-0026 Punkt 2).
func TestKannLoeschenUndTextBearbeiten(t *testing.T) {
	s := randSitzung(t)
	if s.KannLoeschen() || s.KannTextBearbeiten() {
		t.Fatal("ohne Auswahl an")
	}
	kasten, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 50)),
		Strich: document.Strich{Farbe: Schwaerzfarbe, Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	text, err := s.D.Hinzufuegen(document.Text{Ort: document.Pt(80, 80), Inhalt: "Hallo", Schriftgrad: 16,
		Farbe: Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = kasten
	if !s.KannLoeschen() || s.KannTextBearbeiten() {
		t.Error("Kasten: Löschen aus oder Text bearbeiten an")
	}
	s.Auswahl = text
	if !s.KannLoeschen() || !s.KannTextBearbeiten() {
		t.Error("Text: Löschen oder Text bearbeiten aus")
	}
	if !s.TextBearbeiten() || s.KannLoeschen() || s.KannTextBearbeiten() {
		t.Error("während der Texteingabe an")
	}
}
