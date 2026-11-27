// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// leertasteFilter meldet, ob die Tastenschleife die Leertaste der Leinwand angemeldet hat.
func leertasteFilter(e *editor) bool {
	for _, f := range e.alleTastenFilter() {
		if k, ok := f.(key.Filter); ok && k.Name == key.NameSpace && k.Focus == event.Tag(e) {
			return true
		}
	}
	return false
}

// ADR-0039 Nachtrag 1 über den Gio-Router: Mit Fokus auf der Leinwand hält die Leertaste den Zustand, Ziehen mit der
// linken Taste verschiebt dann die Ansicht (auch mit einem Zeichenwerkzeug, ohne neues Objekt); losgelassen ist alles
// wie vorher.
func TestLeertasteZiehenImEditor(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.fokusiere(l.e)
	if !leertasteFilter(l.e) {
		t.Fatal("Leertaste der Leinwand nicht angemeldet")
	}
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
	l.setle()
	if !l.e.s.Leertaste() {
		t.Fatal("Leertaste gedrückt kommt nicht an")
	}
	o := l.e.leinwandFlaeche(l.gtx()).Min
	punkt := func(x, y float64) f32.Point {
		p := l.e.s.Abb.ZuFenster(document.Pt(x, y)).Plus(document.Pt(float64(o.X), float64(o.Y)))
		return f32.Pt(float32(p.X), float32(p.Y))
	}
	a := punkt(40, 40)
	b := a.Add(f32.Pt(60, 30))
	vorher, zahl := l.e.s.Abb.ZuFenster(document.Pt(0, 0)), len(l.e.s.D.Objekte())
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: a})
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: a})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: b})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: b})
	l.setle()
	if d := l.e.s.Abb.ZuFenster(document.Pt(0, 0)).Minus(vorher); d.X != 60 || d.Y != 30 {
		t.Errorf("Ansicht um %v verschoben, Soll (60, 30)", d)
	}
	if n := len(l.e.s.D.Objekte()); n != zahl {
		t.Errorf("%d Objekte, vorher %d: Leertaste+Ziehen legte einen Kasten an", n, zahl)
	}
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Release})
	l.setle()
	if l.e.s.Leertaste() {
		t.Error("Leertaste losgelassen kommt nicht an")
	}
}

// Während der Texteingabe meldet die Schleife die Leertaste nicht an: Sie kommt als Zeichen (EditEvent) in den Text.
// Fällt der Fokus von der Leinwand oder öffnet ein Menü, gilt eine gehaltene Leertaste als losgelassen.
func TestLeertasteInTexteingabeUndOhneFokus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.fokusiere(l.e)
	l.e.s.Auswahl = l.id
	if !l.e.s.TextBearbeiten() {
		t.Fatal("Texteingabe beginnt nicht")
	}
	l.setle()
	if leertasteFilter(l.e) {
		t.Error("Leertaste während der Texteingabe angemeldet")
	}
	l.tippe(" x")
	if txt, _ := l.e.s.TextObjekt(); !strings.HasSuffix(txt.Inhalt, " x") {
		t.Errorf("Text %q, Soll endet auf Leerzeichen und x", txt.Inhalt)
	}
	if l.e.s.Leertaste() {
		t.Error("Leertaste gilt in der Texteingabe als gehalten")
	}
	l.e.s.TextBestaetigen()
	l.setle()
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
	l.setle()
	if !l.e.s.Leertaste() {
		t.Fatal("Leertaste nach der Texteingabe kommt nicht an")
	}
	l.fokusiere(&l.e.menue.titel[0]) // Fokus wandert auf einen Menütitel, das Loslassen käme nicht mehr an
	if l.e.s.Leertaste() {
		t.Error("Leertaste bleibt gehalten, obwohl die Leinwand den Fokus verlor")
	}
}

// ADR-0039 Punkt 5 und 3: Die Fußzeile nennt die Zahl der gewählten Objekte; schöbe Ausrichten eines aus dem sichtbaren
// Bild, sagt die Fußzeile, warum nichts geschah.
func TestMehrfachauswahlHinweisUndAblehnung(t *testing.T) {
	l, k := zeichenLauf(t)
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	if h := l.e.hinweis(); strings.Contains(h, "selected") {
		t.Errorf("Hinweis mit einer Auswahl %q, Soll ohne Zahl", h)
	}
	if !l.e.s.WaehleAlle() {
		t.Fatal("Strg+A")
	}
	if h := l.e.hinweis(); !strings.HasPrefix(h, "2 objects selected") {
		t.Errorf("Hinweis %q, Soll „2 objects selected …“", h)
	}
	// Zuschnitt bis x 10: Kasten (4…24) und Text sind links zu sehen; rechts ausrichten schöbe den Kasten ganz hinter
	// den Zuschnitt.
	if err := l.e.s.D.SetzeZuschnitt(image.Rect(0, 0, 10, 30)); err != nil {
		t.Fatal(err)
	}
	if l.e.s.AnzahlGewaehlt() != 2 || !l.e.s.KannAusrichten() {
		t.Fatalf("nach Zuschnitt %d gewählt", l.e.s.AnzahlGewaehlt())
	}
	vor := l.e.s.D.Stand()
	l.e.fuehreAus(l.gtx(), aktAusrichtenRechts, false)
	if l.e.s.D.Stand() != vor {
		t.Fatal("Ausrichten geschah trotz Ablehnung")
	}
	if !strings.Contains(l.e.meldung.Text, "visible image") {
		t.Errorf("Meldung %q, Soll die Ablehnung nennen", l.e.meldung.Text)
	}
	_ = k
}
