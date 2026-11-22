// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
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

// ADR-0023 Nachtrag 6: „Seam style“ steht im Menü (ohne eigenes Kürzel), wechselt die Form reihum als ein
// Rückgängig-Schritt und nennt die neue Form in der Statusleiste. Nachtrag 7 (F-426): Ohne Naht ist der Eintrag grau
// und tut nichts, das Bild bleibt ungeändert.
func TestMenueKantenform(t *testing.T) {
	e := testEditor(t)
	eintrag := func() (gefunden, an bool) {
		for _, en := range e.menueEintraege() {
			if en.akt == aktKantenform {
				return en.text == "Seam style", en.an
			}
		}
		return false, false
	}
	if gefunden, an := eintrag(); !gefunden || an {
		t.Fatalf("ohne Naht: Eintrag gefunden %v, an %v; Soll vorhanden und grau", gefunden, an)
	}
	e.fuehreAus(layout.Context{}, aktKantenform, true)
	if e.s.D.Kante() != document.KanteAbriss || e.s.Geaendert() {
		t.Fatalf("ohne Naht gewechselt: %v, geändert %v", e.s.D.Kante(), e.s.Geaendert())
	}
	if err := e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 10, Bis: 15}}); err != nil {
		t.Fatal(err)
	}
	if gefunden, an := eintrag(); !gefunden || !an {
		t.Fatal("mit Naht: Eintrag „Seam style“ fehlt oder ist grau")
	}
	if e.s.D.Kante() != document.KanteAbriss {
		t.Fatalf("Ausgang %v", e.s.D.Kante())
	}
	for _, c := range []struct {
		k    document.Kante
		text string
	}{{document.KanteWelle, "Seam style: wave"}, {document.KanteZacken, "Seam style: zigzag"},
		{document.KanteAbriss, "Seam style: torn edge"}} {
		e.fuehreAus(layout.Context{}, aktKantenform, true)
		if e.s.D.Kante() != c.k || e.meldung.Text != c.text {
			t.Errorf("Form %v, Meldung %q; Soll %v, %q", e.s.D.Kante(), e.meldung.Text, c.k, c.text)
		}
	}
	e.rueckgaengig()
	if e.s.D.Kante() != document.KanteZacken {
		t.Errorf("Rückgängig: %v, Soll zigzag", e.s.D.Kante())
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

// TestMenueZeilenhoeheGedeckelt: Zeilenhöhe von 20 bis 32 dp für jede Fensterhöhe, auch knapp unter der Listenhöhe (F-262).
func TestMenueZeilenhoeheGedeckelt(t *testing.T) {
	for h := 0; h <= 1200; h++ {
		z := menueZeilenhoehe(h, 6, 20, 32, 12)
		if z < 20 || z > 32 {
			t.Fatalf("Fensterhöhe %d: Zeilenhöhe %d außerhalb 20..32", h, z)
		}
	}
	if z := menueZeilenhoehe(408, 6, 20, 32, 12); z != 32 {
		t.Errorf("Fensterhöhe 408: %d, will 32", z)
	}
	if z := menueZeilenhoehe(320, 6, 20, 32, 12); z != 25 {
		t.Errorf("Fensterhöhe 320: %d, will 25", z)
	}
}

// F-261, ADR-0026 Nachtrag 1: Bei offenem Menü kommt auch getippter Text nicht in eine laufende Texteingabe.
func TestMenueOffenSperrtGetippterText(t *testing.T) {
	e := testEditor(t)
	id, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 12,
		Farbe: canvas.Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	e.s.Auswahl = id
	r, ops := new(input.Router), new(op.Ops)
	bild := func() {
		ops.Reset()
		e.tasten(layout.Context{Ops: ops, Source: r.Source()})
		event.Op(ops, e)
		r.Frame(ops)
	}
	inhalt := func() string {
		o, _ := e.s.D.Objekt(id)
		return o.(document.Text).Inhalt
	}
	bild()
	layout.Context{Ops: ops, Source: r.Source()}.Execute(key.FocusCmd{Tag: e})
	bild()
	if !e.s.TextBearbeiten() {
		t.Fatal("Vorbedingung: Texteingabe läuft nicht")
	}
	bild()
	r.Queue(key.EditEvent{Text: "a"})
	bild()
	if inhalt() != "Halloa" {
		t.Fatalf("Kontrolle: bei geschlossenem Menü kommt Text an, ist %q", inhalt())
	}
	e.menue.offen = true
	r.Queue(key.EditEvent{Text: "xyz"})
	bild()
	if inhalt() != "Halloa" {
		t.Errorf("bei offenem Menü kam Text an: %q", inhalt())
	}
}
