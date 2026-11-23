// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// TestEnterNachMenuewechselImSelbenBild (F-512): Pfeil rechts und Enter im selben Bild lösen keinen Eintrag aus, weder
// den des alten noch den des Nachbarmenüs an derselben Stelle; das Nachbarmenü bleibt offen. Ohne die Sperre löste
// Enter „Text bearbeiten“ aus (Bearbeiten, dritter Eintrag), weil der Fokus noch auf der Klickfläche der Datei-Liste lag.
func TestEnterNachMenuewechselImSelbenBild(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800)) // Text „Hallo“ ist ausgewählt: „Text bearbeiten“ ist bedienbar
	l.fokusiere(l.e)
	l.taste(key.NameF10, 0)
	l.taste(key.NameDownArrow, 0)
	l.taste(key.NameDownArrow, 0)
	if l.e.menue.welches != menueDatei || menueFokusNr(l) != 2 {
		t.Fatalf("Vorbedingung: Menü %d, Fokus %d", l.e.menue.welches, menueFokusNr(l))
	}
	if en := l.e.menueEintraegeVon(menueBearbeiten); !en[2].an || en[2].akt != aktTextBearbeiten {
		t.Fatalf("Vorbedingung: dritter Eintrag von Bearbeiten ist %+v", en[2])
	}
	l.r.Queue(key.Event{Name: key.NameRightArrow, State: key.Press})
	l.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	l.setle()
	if !l.e.menue.offen || l.e.menue.welches != menueBearbeiten {
		t.Errorf("nach Rechts und Enter: Menü offen %v (%d), soll Bearbeiten offen", l.e.menue.offen, l.e.menue.welches)
	}
	if l.e.s.TextLaeuft() {
		t.Error("Enter im selben Bild wie Pfeil rechts hat den Eintrag des Nachbarmenüs ausgelöst")
	}
	// Im Folgebild gilt Enter wieder: Es löst den Eintrag mit Fokus (Rückgängig) aus und schließt die Liste.
	l.taste(key.NameReturn, 0)
	if l.e.menue.offen {
		t.Error("Enter auf dem Eintrag des Nachbarmenüs wirkt im nächsten Bild nicht")
	}
}

// TestZeigerFeldFolgtZoomPerTasteUndZuschnitt (F-513): Das Fußfeld „Zeigerort“ zeigt Pixel des Originals, rechnet bei
// jedem Bild aus der Fensterposition (ein Zoom per Taste lässt es nicht stehen) und bleibt beim Zuschneiden über der
// Vollansicht vorhanden.
func TestZeigerFeldFolgtZoomPerTasteUndZuschnitt(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	ort := func() (image.Point, bool) {
		for _, x := range l.e.s.FussFelder(l.e.zeigerAnsicht()) {
			if x.Art == canvas.FeldZeiger {
				return x.Ort, true
			}
		}
		return image.Point{}, false
	}
	// Hineingezoomt den Zeiger auf die Leinwandmitte setzen, dann den Zoom per Taste ändern, ohne ihn zu bewegen.
	fl := l.e.leinwandFlaeche(l.gtx())
	mitte := fl.Min.Add(fl.Max).Div(2)
	l.e.s.Scrollen(document.Pt(float64(mitte.X-fl.Min.X), float64(mitte.Y-fl.Min.Y)), -3)
	l.setle()
	z1 := l.e.s.Abb.Zoom
	if z1 <= 1 {
		t.Fatalf("Vorbedingung: Zoom %v nach dem Hineinzoomen", z1)
	}
	l.maus(pointer.Move, mitte, 0)
	l.setle()
	vorher, ok := ort()
	if !ok {
		t.Fatal("Vorbedingung: Feld Zeigerort fehlt über der Mitte")
	}
	l.fokusiere(l.e)
	l.taste("1", key.ModShortcut)
	if l.e.s.Abb.Zoom != 1 {
		t.Fatalf("Strg+1: Zoom %v", l.e.s.Abb.Zoom)
	}
	a := l.e.s.Abb.ZuAnsicht(document.Pt(float64(mitte.X-fl.Min.X), float64(mitte.Y-fl.Min.Y)))
	soll := image.Pt(int(a.X), int(a.Y))
	o, ok := ort()
	if !ok || o != soll {
		t.Errorf("nach Strg+1 bei ruhendem Zeiger: Ort %v (da %v), erwartet %v", o, ok, soll)
	}
	if o == vorher {
		t.Errorf("Ort nach dem Zoom unverändert %v: veraltet", o)
	}

	// Zuschnitt ab (10,10): Ansichtspunkt (2,2) ist Original (12,12).
	l.e.s.Auswahl = 0
	if err := l.e.s.D.SetzeZuschnitt(image.Rect(10, 10, l.e.s.D.Groesse().X, l.e.s.D.Groesse().Y)); err != nil {
		t.Fatal(err)
	}
	l.e.bildNeu()
	l.setle()
	fenster := func(ansicht document.Punkt) image.Point {
		q := l.e.s.Abb.ZuFenster(ansicht)
		return image.Pt(int(q.X)+fl.Min.X, int(q.Y)+fl.Min.Y)
	}
	l.maus(pointer.Move, fenster(document.Pt(2.5, 2.5)), 0)
	l.setle()
	if o, ok := ort(); !ok || o != image.Pt(12, 12) {
		t.Errorf("mit Zuschnitt: Ort %v (da %v), erwartet (12,12)", o, ok)
	}
	// Zuschneiden zeigt die Vollansicht; über dem abgedunkelten Teil ist das Feld da, Original (4,4).
	l.e.s.SetzeWerkzeug(canvas.WerkzeugZuschneiden)
	l.e.bildNeu()
	l.setle()
	l.maus(pointer.Move, fenster(document.Pt(-5.5, -5.5)), 0)
	l.setle()
	if o, ok := ort(); !ok || o != image.Pt(4, 4) {
		t.Errorf("Zuschneiden, abgedunkelter Teil: Ort %v (da %v), erwartet (4,4)", o, ok)
	}
}
