// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// stelleDerArt ist die Zeile der Spitzenart k in der Liste (Reihenfolge document.Kopfarten).
func stelleDerArt(k document.Kopfart) int { return slices.Index(document.Kopfarten, k) }

// ADR-0021 Nachtrag 20: Der Abschnitt „Arrow style“ zeigt beim Werkzeug Pfeil und am gewählten Pfeil je Ende ein
// Klappfeld. Ein Klick öffnet die Liste aller Spitzenarten unter dem Feld; die Wahl gilt fürs Werkzeug bzw. ändert den
// gewählten Pfeil als einen Rückgängig-Schritt. Der Name steht im Feld und in der Statuszeile.
func TestPfeilformKlappfelder(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.setle()
	if !l.e.pfeilform.rahm[0].Empty() {
		t.Fatal("Pfeilform an einem gewählten Text")
	}
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	anfang, ende := l.e.pfeilform.rahm[0], l.e.pfeilform.rahm[1]
	if anfang.Empty() || ende.Empty() || ende.Min.Y <= anfang.Min.Y {
		t.Fatalf("Felder nicht gezeichnet: %v %v", anfang, ende)
	}
	l.maus(pointer.Move, mitte(ende), 0)
	l.setle()
	if h := l.e.hinweis(); !strings.Contains(h, "at the end") || !strings.Contains(h, kopfName(document.KopfDreieck)) {
		t.Errorf("Hinweis %q", h)
	}
	// Klick aufs Feld „End“: Liste aller Arten unter dem Feld, die gewählte (Dreieck) hervorgehoben.
	l.klick(mitte(ende))
	l.setle()
	m := &l.e.menue
	if !m.offen || m.welches != listeKopfEnde || len(l.e.menueEintraege()) != len(document.Kopfarten) {
		t.Fatalf("Liste nicht offen: offen %v, welche %d, %d Einträge", m.offen, m.welches, len(l.e.menueEintraege()))
	}
	if en := l.e.menueEintraege()[stelleDerArt(document.KopfDreieck)]; !en.aktiv || en.kopfAnfang {
		t.Errorf("Dreieck nicht als gewählt markiert: %+v", en)
	}
	if m.rahmen[0].Min.Y < ende.Max.Y-1 {
		t.Errorf("Liste %v nicht unter dem Feld %v", m.rahmen[0], ende)
	}
	vor := l.e.s.D.Stand()
	l.klick(mitte(m.rahmen[stelleDerArt(document.KopfPunkt)]))
	l.setle()
	if _, e := l.e.s.Pfeilform(); e != document.KopfPunkt || l.e.s.D.Stand() != vor || m.offen {
		t.Errorf("Werkzeug: Ende %v, Schritt %v, Liste offen %v", e, l.e.s.D.Stand() != vor, m.offen)
	}
	// Am gewählten Pfeil, per Tastatur: Enter am Feld „Start“ öffnet die Liste mit dem Fokus auf „None“.
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(5, 5), document.Pt(40, 30),
		document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 4}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.e.s.Auswahl = id
	l.setle()
	l.e.knopfPerTaste(&l.e.pfeilform.feld[0])
	l.setle()
	if !m.offen || m.welches != listeKopfAnfang || m.cursor != stelleDerArt(document.KopfKeine) {
		t.Fatalf("Liste per Taste: offen %v, welche %d, Cursor %d", m.offen, m.welches, m.cursor)
	}
	vor = l.e.s.D.Stand()
	l.klick(mitte(m.rahmen[stelleDerArt(document.KopfRaute)]))
	l.setle()
	o, _ := l.e.s.D.Objekt(id)
	if p := o.(document.Pfeil); p.KopfAnfang() != document.KopfRaute || p.Ende != document.KopfDreieck || l.e.s.D.Stand() == vor {
		t.Errorf("gewählter Pfeil: Anfang %v, Ende %v, Schritt %v", p.KopfAnfang(), p.Ende, l.e.s.D.Stand() != vor)
	}
	if a, _ := l.e.s.Pfeilform(); a != document.KopfRaute {
		t.Errorf("Leiste zeigt %v", a)
	}
}

// Jede Vorschau der Liste ist sichtbar: Die Vorschau eines Endes zeichnet einen Pfeil, auch bei Punkt, Quadrat und
// Balken, die über das Ende hinausragen (eingerückt um document.KopfUeberstand); ohne Spitze bleibt der Schaft.
func TestKopfMusterPasstInsFeld(t *testing.T) {
	for _, art := range document.Kopfarten {
		for _, amAnfang := range []bool{false, true} {
			anfang, ende := document.KopfKeine, art
			if amAnfang {
				anfang, ende = art, document.KopfKeine
			}
			h := musterPfeil(anfang, ende, image.Rect(0, 0, 44, 16), 2, color.NRGBA{A: 255}).Form().Huelle(2)
			// Ein Ende ohne Spitze hat die runde Kappe des Strichs (halbe Breite 1).
			if h.Min.X < -1-1e-9 || h.Max.X > 45+1e-9 {
				t.Errorf("%s (Anfang %v): Hülle %v ragt aus 0…44", art.Name(), amAnfang, h)
			}
		}
	}
}
