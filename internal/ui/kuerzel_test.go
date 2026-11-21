// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

func testEditor(t *testing.T) *editor {
	t.Helper()
	i18n.Setze(nil)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	return &editor{s: canvas.NeueSitzung(d)}
}

// E-21, ADR-0026 Punkt 6: Jede Taste des Editors hat einen Knopf, einen Menüeintrag oder eine begründete Ausnahme.
func TestJedeTasteAuchPerMaus(t *testing.T) {
	e := testEditor(t)
	imMenue := map[aktion]bool{}
	for _, en := range e.menueEintraege() {
		imMenue[en.akt] = true
	}
	werkzeuge := map[canvas.Werkzeug]bool{}
	for _, w := range e.werkzeugKnoepfe() {
		werkzeuge[w.w] = true
	}
	for _, ta := range tastenTabelle() {
		name := tastenName(ta.filter)
		switch {
		case ta.ausnahme != "":
			if ta.hatWerkzeug {
				t.Errorf("%s: Werkzeug und Ausnahme zugleich", name)
			}
		case ta.hatWerkzeug:
			if !werkzeuge[ta.werkzeug] {
				t.Errorf("%s: Werkzeug %v ohne Knopf in der Leiste", name, ta.werkzeug)
			}
		case ta.akt == aktKeine:
			t.Errorf("%s: weder Handlung noch Werkzeug noch Ausnahme", name)
		case !imMenue[ta.akt]:
			t.Errorf("%s: Handlung %d ohne Menüeintrag", name, ta.akt)
		}
	}
	if len(e.menueEintraege()) > len(e.menue.eintraege) {
		t.Errorf("%d Menüeinträge, aber nur %d Clickables", len(e.menueEintraege()), len(e.menue.eintraege))
	}
}

// Jeder Menüeintrag mit Kürzel zeigt es an; die Anzeige folgt der Tabelle.
func TestMenueZeigtKuerzel(t *testing.T) {
	e := testEditor(t)
	soll := map[aktion]string{aktKopieren: "Ctrl+C", aktExport: "Ctrl+Shift+E", aktSpeichern: "Ctrl+S",
		aktRueckgaengig: "Ctrl+Z", aktWiederholen: "Ctrl+Shift+Z", aktTextBearbeiten: "Enter", aktLoeschen: "Del",
		aktEinpassen: "Ctrl+0", aktEins: "Ctrl+1", aktSchliessen: "Ctrl+W", aktGanzesBild: "", aktZurueckholen: ""}
	for _, en := range e.menueEintraege() {
		if got := menueKuerzel(en.akt); got != soll[en.akt] {
			t.Errorf("%s: Kürzel %q, Soll %q", en.text, got, soll[en.akt])
		}
	}
	if got := werkzeugKuerzel(canvas.WerkzeugPfeil); got != "A" {
		t.Errorf("Pfeil: %q", got)
	}
}

// Die Tabelle trennt Strg+C (Kopieren) von C (Zuschneiden) und Strg+Z von Strg+Umschalt+Z, wie die Filter von Gio.
func TestFindeTaste(t *testing.T) {
	for _, fall := range []struct {
		name   key.Name
		mod    key.Modifiers
		text   bool
		akt    aktion
		wz     canvas.Werkzeug
		hatWz  bool
		gefund bool
	}{
		{"C", key.ModShortcut, false, aktKopieren, 0, false, true},
		{"C", 0, false, aktKeine, canvas.WerkzeugZuschneiden, true, true},
		{"Z", key.ModShortcut, false, aktRueckgaengig, 0, false, true},
		{"Z", key.ModShortcut | key.ModShift, false, aktWiederholen, 0, false, true},
		{"E", key.ModShortcut, false, aktKeine, 0, false, false}, // Export nur mit Umschalt
		{key.NameReturn, key.ModShift, false, aktEnter, 0, false, true},
		{key.NameTab, 0, false, aktKeine, 0, false, false},
		{key.NameTab, 0, true, aktKeine, 0, false, true},
	} {
		ta, ok := findeTaste(fall.name, fall.mod, fall.text)
		if ok != fall.gefund || ta.akt != fall.akt || ta.hatWerkzeug != fall.hatWz || (fall.hatWz && ta.werkzeug != fall.wz) {
			t.Errorf("%s/%v: %+v %v", fall.name, fall.mod, ta, ok)
		}
	}
}
