// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 19 Punkt 5: Die Zeilen „Head“ und „Ends“ erscheinen beim Werkzeug Pfeil und am gewählten Pfeil;
// ein Klick setzt die Wahl fürs Werkzeug bzw. ändert den Pfeil, der Name steht in der Statuszeile.
func TestPfeilformKnoepfe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.setle()
	if !l.e.pfeilform.kopfRahm[0].Empty() {
		t.Fatal("Pfeilform an einem gewählten Text")
	}
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	r := l.e.pfeilform.kopfRahm[2] // Punkt
	if r.Empty() || l.e.pfeilform.endenRahm[3].Empty() {
		t.Fatalf("Knöpfe nicht gezeichnet: %v %v", l.e.pfeilform.kopfRahm, l.e.pfeilform.endenRahm)
	}
	l.maus(pointer.Move, mitte(r), 0)
	l.setle()
	if l.e.hinweis() != kopfName(document.KopfPunkt) {
		t.Errorf("Hinweis %q", l.e.hinweis())
	}
	vor := l.e.s.D.Stand()
	l.klick(mitte(r))
	if k, _ := l.e.s.Pfeilform(); k != document.KopfPunkt || l.e.s.D.Stand() != vor {
		t.Errorf("Werkzeug: Kopf %v, Schritt %v", k, l.e.s.D.Stand() != vor)
	}
	// Am gewählten Pfeil: „keine“ macht eine Linie, als ein Rückgängig-Schritt.
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(5, 5), document.Pt(40, 30),
		document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 4}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.e.s.Auswahl = id
	l.setle()
	vor = l.e.s.D.Stand()
	l.klick(mitte(l.e.pfeilform.endenRahm[0]))
	o, _ := l.e.s.D.Objekt(id)
	if p := o.(document.Pfeil); p.Enden != document.EndenKeine || l.e.s.D.Stand() == vor {
		t.Errorf("gewählter Pfeil: Enden %v, Schritt %v", p.Enden, l.e.s.D.Stand() != vor)
	}
	// Die gewählte Wahl ist hervorgehoben: dieselben Knöpfe, der Zustand kommt aus der Sitzung.
	if _, e := l.e.s.Pfeilform(); e != document.EndenKeine {
		t.Errorf("Leiste zeigt %v", e)
	}
}
