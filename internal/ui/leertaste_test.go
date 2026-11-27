// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/layout"

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

// F-905: Gio prüft die Umschalter auch beim Loslassen. Wird die Leertaste mit Strg, Alt oder Super losgelassen, muss das
// Loslassen trotzdem ankommen; ein Druck mit Strg, Alt oder Super hält sie nicht.
func TestLeertasteLoslassenMitUmschaltern(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.fokusiere(l.e)
	for _, m := range []key.Modifiers{key.ModCtrl, key.ModAlt, key.ModSuper, key.ModShift} {
		l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
		l.setle()
		if !l.e.s.Leertaste() {
			t.Fatalf("%v: Druck kommt nicht an", m)
		}
		l.r.Queue(key.Event{Name: key.NameSpace, State: key.Release, Modifiers: m})
		l.setle()
		if l.e.s.Leertaste() {
			t.Errorf("Loslassen mit %v kommt nicht an, Leertaste hängt", m)
		}
	}
	for _, m := range []key.Modifiers{key.ModCtrl, key.ModAlt, key.ModSuper} {
		l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press, Modifiers: m})
		l.setle()
		if l.e.s.Leertaste() {
			t.Errorf("%v+Leertaste hält die Leertaste", m)
		}
	}
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press, Modifiers: key.ModShift})
	l.setle()
	if !l.e.s.Leertaste() {
		t.Error("Umschalt+Leertaste hält die Leertaste nicht")
	}
}

// F-906: Mit gehaltener Leertaste ist ein zweiter Druck kein Doppelklick: Er beginnt keine Texteingabe und hebt keinen
// Zuschnitt auf, sondern verschiebt die Ansicht.
func TestLeertasteKeinDoppelklick(t *testing.T) {
	doppel := func(l *menueLauf, p image.Point) {
		l.maus(pointer.Move, p, 0)
		l.setle()
		for range 2 {
			l.maus(pointer.Press, p, pointer.ButtonPrimary)
			l.setle()
			l.maus(pointer.Release, p, 0)
			l.setle()
		}
	}
	halte := func(l *menueLauf) {
		l.fokusiere(l.e)
		l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
		l.setle()
		if !l.e.s.Leertaste() {
			t.Fatal("Leertaste kommt nicht an")
		}
	}
	// Text: Doppelklick begänne die Texteingabe.
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	halte(l)
	// Schon ein einzelner Druck zählt nicht als erster Klick eines Doppelklicks.
	l.maus(pointer.Move, image.Pt(700, 700), 0)
	l.maus(pointer.Press, image.Pt(700, 700), pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, image.Pt(700, 700), 0)
	l.setle()
	if l.e.hatDruck {
		t.Error("ein einzelner Verschiebe-Druck zählt für einen Doppelklick")
	}
	o, _ := l.e.s.D.Objekt(l.id)
	g := o.Grenzen()
	doppel(l, l.bildPt(image.Pt(int((g.Min.X+g.Max.X)/2), int((g.Min.Y+g.Max.Y)/2))))
	if l.e.s.TextLaeuft() {
		t.Error("Doppelklick mit gehaltener Leertaste beginnt die Texteingabe")
	}
	// Zuschneiden: Doppelklick höbe den Zuschnitt auf.
	l = neuerMenueLauf(t, image.Pt(1200, 900))
	gr := l.e.s.D.Groesse()
	if err := l.e.s.D.SetzeZuschnitt(image.Rect(0, 0, gr.X/2, gr.Y/2)); err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugZuschneiden)
	l.setle()
	halte(l)
	vor := l.e.s.D.Stand()
	doppel(l, l.bildPt(image.Pt(3, 3)))
	if l.e.s.D.Stand() != vor {
		t.Errorf("Doppelklick mit gehaltener Leertaste änderte das Dokument, Größe %v", l.e.s.D.Groesse())
	}
	// Losgelassen: Ein Klick gleich danach an derselben Stelle zählt nicht als zweiter Klick eines Doppelklicks.
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Release})
	l.setle()
	if l.e.hatDruck {
		t.Error("ein Verschiebe-Druck zählt für den nächsten Doppelklick")
	}
}

// F-907: Die Leinwand holt sich beim Öffnen (und nach einem Wechsel) den Tastaturfokus, damit Leertaste+Ziehen ohne
// vorherigen Klick ins Bild wirkt; offene Tafel oder volle Ansicht behalten ihn.
func TestLeinwandHoltFokus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	if l.fokus(l.e) {
		t.Fatal("Vorbedingung: testEditor startet ohne Fokus")
	}
	l.e.fokusHolen = true // wie neuerEditor
	l.setle()
	if !l.fokus(l.e) || l.e.fokusHolen {
		t.Fatalf("Fokus %v, fokusHolen %v; Soll Fokus auf der Leinwand, Merker verbraucht", l.fokus(l.e), l.e.fokusHolen)
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
	l.setle()
	zahl := len(l.e.s.D.Objekte())
	l.zug(image.Pt(5, 5), image.Pt(30, 25))
	if len(l.e.s.D.Objekte()) != zahl {
		t.Error("Leertaste+Ziehen nach dem Öffnen legte einen Kasten an")
	}
	// Bei offener Übersicht F1 bleibt der Merker stehen und greift erst danach.
	l.fokusiere(&l.e.menue.titel[0])
	l.e.hilfeOffen, l.e.fokusHolen = true, true
	l.setle()
	if l.fokus(l.e) || !l.e.fokusHolen {
		t.Error("Leinwand nimmt sich den Fokus hinter der offenen Übersicht")
	}
	l.e.hilfeOffen = false
	l.setle()
	if !l.fokus(l.e) {
		t.Error("nach dem Schließen der Übersicht kein Fokus auf der Leinwand")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	if !neuerEditor("datei.png", d, nil, nil).fokusHolen {
		t.Error("neuerEditor setzt fokusHolen nicht")
	}
}

// Loslass-Bedingungen (Kritik #334, F-910): Menü, Übersicht, Einstellungstafel, volle Ansicht, Texteingabe und der
// Fokusverlust des Fensters lassen eine gehaltene Leertaste los.
func TestLeertasteLaesstLos(t *testing.T) {
	for _, fall := range []struct {
		name  string
		setze func(e *editor)
	}{
		{"Menü", func(e *editor) { e.menue.offen = true }},
		{"Übersicht", func(e *editor) { e.hilfeOffen = true }},
		{"Tafel", func(e *editor) { e.einst.offen = true }},
		{"Bibliothek", func(e *editor) { e.bibl.offen = true }},
		{"Texteingabe", func(e *editor) { e.s.TextBearbeiten() }},
		{"Fensterfokus", func(e *editor) { e.fensterFokus(false) }},
	} {
		l := neuerMenueLauf(t, image.Pt(1200, 900))
		l.fokusiere(l.e)
		l.e.s.SetzeLeertaste(true)
		if !l.e.leertasteGilt(l.gtx()) {
			t.Fatalf("%s: Vorbedingung", fall.name)
		}
		fall.setze(l.e)
		var gtx layout.Context = l.gtx()
		if fall.name != "Fensterfokus" && l.e.leertasteGilt(gtx) {
			t.Errorf("%s: leertasteGilt bleibt wahr", fall.name)
		}
		l.e.tasten(gtx)
		if l.e.s.Leertaste() {
			t.Errorf("%s: Leertaste bleibt gehalten", fall.name)
		}
		l.e.fensterFokus(true)
	}
	// Ein Druck bei offenem Menü hält sie nicht.
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.fokusiere(l.e)
	l.e.menue.offen = true
	l.r.Queue(key.Event{Name: key.NameSpace, State: key.Press})
	l.bild()
	if l.e.s.Leertaste() {
		t.Error("Druck bei offenem Menü hält die Leertaste")
	}
}

// Werkzeug Auswählen erklärt Umschalt+Klick, Rahmen und Leertaste; die Übersicht F1 nennt Leertaste+Ziehen (F-911).
func TestAuswahlHinweisUndHilfezeile(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 900))
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	if h := l.e.hinweis(); !strings.Contains(h, "Shift+click") || !strings.Contains(h, "Space+drag") {
		t.Errorf("Hinweis Auswählen %q", h)
	}
	gefunden := false
	for _, z := range l.e.hilfeZeilen() {
		if z.text == "Move the view" && strings.HasPrefix(z.taste, "Space+") {
			gefunden = true
		}
	}
	if !gefunden {
		t.Error("Übersicht F1 ohne Zeile „Move the view“ mit Leertaste")
	}
}
