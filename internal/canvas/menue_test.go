// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
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

// Während eines Zugs ist „Gestauchtes zurückholen“ aus (F-253): Es änderte sonst die Ansicht unter dem Zug.
func TestKannZurueckholenNichtImZug(t *testing.T) {
	s := randSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 50, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Druecken(fv(s, 20, 20), TastePrimaer)
	s.Ziehen(fv(s, 80, 80))
	if s.KannZurueckholen() || s.HoleGestauchtesZurueck() {
		t.Error("im Zug an")
	}
	s.Loslassen(fv(s, 80, 80))
	if !s.KannZurueckholen() {
		t.Error("nach dem Zug aus")
	}
}

// ADR-0023 Nachtrag 6: „Seam style“ wechselt die Kantenform reihum, jeder Wechsel ein Rückgängig-Schritt, auch ohne
// Naht; im Zug und während des Speicherns aus.
func TestWechsleKante(t *testing.T) {
	s := randSitzung(t)
	if s.D.Kante() != document.KanteAbriss || !s.KannKanteWechseln() {
		t.Fatalf("Ausgang %v, an %v", s.D.Kante(), s.KannKanteWechseln())
	}
	vor := s.D.Stand()
	for _, soll := range []document.Kante{document.KanteWelle, document.KanteZacken, document.KanteAbriss} {
		if k, ok := s.WechsleKante(); !ok || k != soll || s.D.Kante() != soll {
			t.Fatalf("WechsleKante %v %v, Soll %v", k, ok, soll)
		}
	}
	if !s.Geaendert() {
		t.Error("Wechsel gilt nicht als Änderung")
	}
	for _, soll := range []document.Kante{document.KanteZacken, document.KanteWelle, document.KanteAbriss} {
		if !s.Rueckgaengig() || s.D.Kante() != soll {
			t.Fatalf("Rückgängig: %v, Soll %v", s.D.Kante(), soll)
		}
	}
	if s.D.Stand() != vor {
		t.Error("drei Rückgängig führen nicht auf den Stand davor")
	}
	s.Einpassen()
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Druecken(fv(s, 20, 20), TastePrimaer)
	s.Ziehen(fv(s, 80, 80))
	if s.KannKanteWechseln() {
		t.Error("im Zug an")
	}
	if _, ok := s.WechsleKante(); ok || s.D.Kante() != document.KanteAbriss {
		t.Error("im Zug gewechselt")
	}
	s.Loslassen(fv(s, 80, 80))
	s.hg.laeuft = true // Speichern im Hintergrund (ADR-0020 Nachtrag 6)
	if _, ok := s.WechsleKante(); ok || s.KannKanteWechseln() {
		t.Error("während des Speicherns gewechselt")
	}
}
