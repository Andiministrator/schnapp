// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 25 Punkt 7: Das Klappfeld „Line style“ steht bei Pfeil, Kasten und Lupe unter dem Regler der
// Strichstärke. Ein Klick öffnet die Liste aller Arten unter dem Feld; die Wahl gilt fürs Werkzeug bzw. ändert das
// gewählte Objekt als einen Rückgängig-Schritt. Der Name steht im Feld und in der Statuszeile.
func TestStrichartKlappfeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.setle()
	if !l.e.strichart.rahm.Empty() {
		t.Fatal("Strichart an einem gewählten Text")
	}
	l.e.s.Auswahl = 0
	for _, w := range []canvas.Werkzeug{canvas.WerkzeugPfeil, canvas.WerkzeugKasten, canvas.WerkzeugLupe} {
		l.e.s.SetzeWerkzeug(w)
		l.setle()
		if l.e.strichart.rahm.Empty() {
			t.Errorf("Werkzeug %v: Feld nicht gezeichnet", w)
		}
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugNummer)
	l.setle()
	if !l.e.strichart.rahm.Empty() {
		t.Error("Feld bei der Nummer")
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.setle()
	feld := l.e.strichart.rahm
	l.maus(pointer.Move, mitte(feld), 0)
	l.setle()
	if h := l.e.hinweis(); !strings.Contains(h, "Line style") || !strings.Contains(h, strichartName(document.StrichVoll)) {
		t.Errorf("Hinweis %q", h)
	}
	l.klick(mitte(feld))
	l.setle()
	m := &l.e.menue
	if !m.offen || m.welches != listeStrichart || len(l.e.menueEintraege()) != len(document.Stricharten) {
		t.Fatalf("Liste nicht offen: offen %v, welche %d, %d Einträge", m.offen, m.welches, len(l.e.menueEintraege()))
	}
	if en := l.e.menueEintraege()[0]; !en.aktiv || !en.hatStrichart {
		t.Errorf("voll nicht als gewählt markiert: %+v", en)
	}
	if m.rahmen[0].Min.Y < feld.Max.Y-1 {
		t.Errorf("Liste %v nicht unter dem Feld %v", m.rahmen[0], feld)
	}
	stelle := func(a document.Strichart) int { return slices.Index(document.Stricharten, a) }
	vor := l.e.s.D.Stand()
	l.klick(mitte(m.rahmen[stelle(document.StrichGepunktet)]))
	l.setle()
	if a := l.e.s.Strichart(); a != document.StrichGepunktet || l.e.s.D.Stand() != vor || m.offen {
		t.Errorf("Werkzeug: %v, Schritt %v, Liste offen %v", a, l.e.s.D.Stand() != vor, m.offen)
	}
	// Am gewählten Pfeil, per Tastatur: Enter am Feld öffnet die Liste mit dem Fokus auf seiner Art.
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(5, 5), document.Pt(40, 30),
		document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 4}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.e.s.Auswahl = id
	l.setle()
	if l.e.strichart.rahm.Empty() {
		t.Fatal("Feld am gewählten Pfeil nicht gezeichnet")
	}
	l.e.knopfPerTaste(&l.e.strichart.feld)
	l.setle()
	if !m.offen || m.welches != listeStrichart || m.cursor != stelle(document.StrichVoll) {
		t.Fatalf("Liste per Taste: offen %v, welche %d, Cursor %d", m.offen, m.welches, m.cursor)
	}
	vor = l.e.s.D.Stand()
	l.klick(mitte(m.rahmen[stelle(document.StrichGestrichelt)]))
	l.setle()
	o, _ := l.e.s.D.Objekt(id)
	if p := o.(document.Pfeil); p.Strich.Art != document.StrichGestrichelt || l.e.s.D.Stand() == vor {
		t.Errorf("gewählter Pfeil: %v, Schritt %v", p.Strich.Art, l.e.s.D.Stand() != vor)
	}
}
