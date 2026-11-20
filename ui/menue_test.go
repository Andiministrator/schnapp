// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/canvas"
	"github.com/Andiministrator/schnapp/document"
)

// F-253 (ADR-0026): Das Menü führt seine Handlungen über fuehreAus aus. Per Maus wird ein laufender Text vorher
// bestätigt, damit die Handlung wirkt (Löschen braucht eine abgeschlossene Eingabe); per Taste nicht.
func TestMenueLoeschenBestaetigtTextVorher(t *testing.T) {
	for _, vonMaus := range []bool{true, false} {
		e := testEditor(t)
		id, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 12,
			Farbe: canvas.Schwaerzfarbe})
		if err != nil {
			t.Fatal(err)
		}
		e.s.Auswahl = id
		if !e.s.TextBearbeiten() {
			t.Fatal("Vorbedingung: Texteingabe")
		}
		e.fuehreAus(layout.Context{}, aktLoeschen, vonMaus)
		_, noch := e.s.D.Objekt(id)
		if vonMaus && (noch || e.s.TextLaeuft()) {
			t.Errorf("per Maus: Text noch da %v, Eingabe läuft %v", noch, e.s.TextLaeuft())
		}
		if !vonMaus && (!noch || !e.s.TextLaeuft()) {
			t.Errorf("per Taste: Text weg %v oder Eingabe beendet %v", !noch, !e.s.TextLaeuft())
		}
	}
}

// „Text bearbeiten“ und „Gestauchtes zurückholen“ wirken über fuehreAus wie Enter bzw. der Doppelklick mit X.
func TestMenueTextBearbeitenUndZurueckholen(t *testing.T) {
	e := testEditor(t)
	id, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 12,
		Farbe: canvas.Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	e.s.Auswahl = id
	e.fuehreAus(layout.Context{}, aktTextBearbeiten, true)
	if !e.s.TextLaeuft() {
		t.Error("Text bearbeiten: keine Eingabe")
	}
	e.s.TextBestaetigen()
	if err := e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 10, Bis: 14}}); err != nil {
		t.Fatal(err)
	}
	e.fuehreAus(layout.Context{}, aktZurueckholen, true)
	if len(e.s.D.Streifen()) != 0 || !e.s.Geaendert() {
		t.Errorf("Zurückholen: Streifen %v, geändert %v", e.s.D.Streifen(), e.s.Geaendert())
	}
}

// Jeder Menüeintrag hat eine Wirkung in fuehreAus: Ein Eintrag, den fuehreAus nicht kennt, wäre ein toter Knopf.
func TestMenueEintraegeHabenWirkung(t *testing.T) {
	e := testEditor(t)
	for _, en := range e.menueEintraege() {
		if !aktionAusfuehrbar(en.akt) {
			t.Errorf("%s: keine Wirkung in fuehreAus", en.text)
		}
	}
	if aktionAusfuehrbar(aktKeine) || aktionAusfuehrbar(aktEsc) || aktionAusfuehrbar(aktEnter) {
		t.Error("aktKeine, aktEsc oder aktEnter als Menü-Handlung ausführbar")
	}
}
