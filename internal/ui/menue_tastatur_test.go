// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// menueLauf fährt den Editor über echte Bilder (frame) mit Gio-Router, damit Fokus, Tasten und Zeiger so laufen wie im
// Fenster. Ohne Fenster (w ist nil): Schließen ist nicht prüfbar.
type menueLauf struct {
	t     *testing.T
	e     *editor
	r     *input.Router
	ops   *op.Ops
	fenst image.Point
	id    int // Text-Objekt „Hallo“, ausgewählt
}

func neuerMenueLauf(t *testing.T, fenster image.Point) *menueLauf {
	t.Helper()
	e := testEditor(t)
	e.thema = material.NewTheme()
	// Ein Speichern, das nichts tut: Löst ein Mutant versehentlich „Speichern“ aus, endet der Lauf rot mit Testnamen
	// statt in einer Panik der Hintergrund-Goroutine ohne Namen (Kritik B-1, Mutant O2).
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	id, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 12,
		Farbe: canvas.Schwaerzfarbe})
	if err != nil {
		t.Fatal(err)
	}
	e.s.Auswahl = id
	e.bildNeu()
	l := &menueLauf{t: t, e: e, r: new(input.Router), ops: new(op.Ops), fenst: fenster, id: id}
	l.bild()
	l.bild()
	return l
}

func (l *menueLauf) gtx() layout.Context {
	return layout.Context{Ops: l.ops, Source: l.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(l.fenst)}
}

// bild zeichnet ein Bild wie die Ereignisschleife und reicht es dem Router.
func (l *menueLauf) bild() {
	l.ops.Reset()
	l.e.frame(l.gtx())
	l.r.Frame(l.ops)
}

// setle lässt Ereignisse und Fokusbefehle sich auswirken (mehrere Bilder, wie das Fenster sie nachzieht).
func (l *menueLauf) setle() {
	for range 3 {
		l.bild()
	}
}

func (l *menueLauf) taste(n key.Name, m key.Modifiers) {
	l.r.Queue(key.Event{Name: n, Modifiers: m, State: key.Press})
	l.setle()
}

func (l *menueLauf) tippe(s string) {
	l.r.Queue(key.EditEvent{Text: s})
	l.setle()
}

func (l *menueLauf) fokus(tag any) bool { return l.r.Source().Focused(tag) }

// fokusiere setzt den Fokus wie die Tab-Taste es täte (Gio macht beides über denselben Befehl).
func (l *menueLauf) fokusiere(tag any) {
	l.gtx().Execute(key.FocusCmd{Tag: tag})
	l.setle()
}

// klick drückt und löst die linke Taste an p (Fensterpixel).
func (l *menueLauf) klick(p image.Point) {
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos})
	l.setle()
}

func (l *menueLauf) inhalt() string {
	o, ok := l.e.s.D.Objekt(l.id)
	if !ok {
		return "<weg>"
	}
	return o.(document.Text).Inhalt
}

// titel ist der Titel des offenen bzw. zuletzt offenen Menüs (ADR-0031: ersetzt den Menütitel).
func (l *menueLauf) titel() *widget.Clickable { return l.e.listenKnopf(l.e.menue.welches) } // auch Fußzeilen-Listen (A-4)

// zuMenue wählt das Menü, das a führt, für das nächste Öffnen (Menü geschlossen).
func (l *menueLauf) zuMenue(a aktion) {
	l.t.Helper()
	if l.e.menue.offen {
		l.t.Fatal("zuMenue bei offenem Menü")
	}
	l.e.menue.welches = l.e.menueMit(a)
}

// index liefert den Index des Eintrags zu einer Handlung im offenen Menü. Steht sie in einem anderen Menü der Zeile,
// wechselt es dorthin, wie der Mensch über den Titel fährt (ADR-0031 Punkt 1), und zeichnet neu.
func (l *menueLauf) index(a aktion) int {
	if m := l.e.menueMit(a); m >= 0 && m != l.e.menue.welches {
		l.e.menue.welches, l.e.menue.cursor = m, -1
		l.setle()
	}
	for i, en := range l.e.menueEintraege() {
		if !en.hatWerkzeug && en.akt == a {
			return i
		}
	}
	l.t.Fatalf("kein Eintrag zu Aktion %d", a)
	return -1
}

// oeffneMitTaste: Tab bis zum Titel des gewählten Menüs (Fokus), Enter.
func (l *menueLauf) oeffneMitTaste() {
	l.fokusiere(l.titel())
	l.taste(key.NameReturn, 0)
}

// Tests der Auswahllogik ohne Gio.
func TestMenueNaechster(t *testing.T) {
	an := func(b ...bool) []menueEintrag {
		var r []menueEintrag
		for _, x := range b {
			r = append(r, menueEintrag{an: x})
		}
		return r
	}
	for _, c := range []struct {
		name        string
		en          []menueEintrag
		von, dir    int
		soll        int
		ersteLetzte [2]int
	}{
		{"abwärts", an(true, true, true), 0, 1, 1, [2]int{0, 2}},
		{"graue überspringen", an(true, false, false, true), 0, 1, 3, [2]int{0, 3}},
		{"aufwärts überspringt", an(true, false, false, true), 3, -1, 0, [2]int{0, 3}},
		{"Umlauf abwärts", an(true, true, false), 1, 1, 0, [2]int{0, 1}},
		{"Umlauf aufwärts", an(true, true), 0, -1, 1, [2]int{0, 1}},
		{"ohne Ansteuerung abwärts", an(false, true, true), -1, 1, 1, [2]int{1, 2}},
		{"ohne Ansteuerung aufwärts", an(true, true, false), -1, -1, 1, [2]int{0, 1}},
		{"alle grau", an(false, false), 0, 1, -1, [2]int{-1, -1}},
		{"nur der eine bleibt", an(false, true), 1, 1, 1, [2]int{1, 1}},
		{"leer", nil, -1, 1, -1, [2]int{-1, -1}},
	} {
		if got := menueNaechster(c.en, c.von, c.dir); got != c.soll {
			t.Errorf("%s: %d, soll %d", c.name, got, c.soll)
		}
		if g := [2]int{menueErster(c.en), menueLetzter(c.en)}; g != c.ersteLetzte {
			t.Errorf("%s: erster/letzter %v, soll %v", c.name, g, c.ersteLetzte)
		}
	}
}

// F-255: Menü per Tastatur öffnen setzt den Fokus auf den ersten bedienbaren Eintrag; Esc schließt und gibt den
// Fokus an den Menütitel.
func TestMenueTastaturOeffnenUndEsc(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.oeffneMitTaste()
	m := &l.e.menue
	if !m.offen {
		t.Fatal("Enter auf dem Menütitel öffnet das Menü nicht")
	}
	if !l.fokus(&m.eintraege[0]) {
		t.Errorf("Fokus nicht auf dem ersten Eintrag (Kopieren)")
	}
	l.taste(key.NameEscape, 0)
	if m.offen {
		t.Error("Esc schließt das Menü nicht")
	}
	if !l.fokus(l.titel()) {
		t.Error("nach Esc hat der Menütitel den Fokus nicht")
	}
	// Enter auf dem Knopf öffnet wieder (Gegenprobe: der Fokus liegt wirklich dort).
	l.taste(key.NameReturn, 0)
	if !m.offen {
		t.Error("Enter auf dem Knopf nach Esc öffnet nicht wieder")
	}
}

// Pfeiltasten, Pos1, Ende: graue Einträge übersprungen, an den Enden Umlauf.
func TestMenuePfeiltasten(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	// Menü „Bearbeiten“ (ADR-0031): Rückgängig, Wiederholen (grau), Text bearbeiten, Löschen.
	l.zuMenue(aktWiederholen)
	en := l.e.menueEintraegeVon(menueBearbeiten)
	if en[1].akt != aktWiederholen || en[1].an {
		t.Fatalf("Vorbedingung: Wiederholen nicht grau an zweiter Stelle: %+v", en)
	}
	stand := l.e.s.D.Stand()
	l.oeffneMitTaste()
	m := &l.e.menue
	zu := func(a aktion, wo string) {
		t.Helper()
		if i := l.index(a); !l.fokus(&m.eintraege[i]) {
			t.Errorf("%s: Fokus nicht auf Eintrag %d (%s)", wo, i, en[i].text)
		}
	}
	zu(aktRueckgaengig, "nach dem Öffnen")
	l.taste(key.NameDownArrow, 0)
	zu(aktTextBearbeiten, "runter über den grauen Eintrag Wiederholen")
	l.taste(key.NameUpArrow, 0)
	zu(aktRueckgaengig, "hoch über den grauen Eintrag Wiederholen")
	l.taste(key.NameEnd, 0)
	zu(aktLoeschen, "Ende")
	l.taste(key.NameDownArrow, 0)
	zu(aktRueckgaengig, "Umlauf abwärts vom letzten zum ersten")
	l.taste(key.NameUpArrow, 0)
	zu(aktLoeschen, "Umlauf aufwärts vom ersten zum letzten")
	l.taste(key.NameHome, 0)
	zu(aktRueckgaengig, "Pos1")
	if !m.offen {
		t.Error("Pfeiltasten schließen das Menü")
	}
	// Kein Eintrag führt dabei etwas aus.
	if l.e.s.D.Stand() != stand || l.inhalt() != "Hallo" {
		t.Errorf("Pfeiltasten änderten das Dokument: Inhalt %q", l.inhalt())
	}
}

// Ohne offenes Menü greifen Pfeiltasten nicht ab (kein Filter, kein Eingriff in die Leinwand).
func TestPfeiltastenOhneMenueTunNichts(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	vorher, stand := l.e.s.Abb, l.e.s.D.Stand()
	for _, n := range []key.Name{key.NameDownArrow, key.NameUpArrow, key.NameHome, key.NameEnd} {
		l.taste(n, 0)
	}
	if l.e.menue.offen || l.e.s.Abb != vorher || l.e.s.D.Stand() != stand {
		t.Error("Pfeiltasten wirken bei geschlossenem Menü")
	}
}

// F-255 Punkt 2 (Prüfbefehl von F-255): Text setzen, Menü per Tastatur, zu „Text bearbeiten“, Enter, tippen — der Text
// kommt an, weil der Fokus nach dem Ausführen auf der Leinwand liegt.
func TestMenueTextBearbeitenPerTastaturDanachTippen(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.oeffneMitTaste()
	for !l.fokus(&l.e.menue.eintraege[l.index(aktTextBearbeiten)]) {
		l.taste(key.NameDownArrow, 0)
		if l.e.menue.cursor > 20 {
			t.Fatal("Eintrag „Text bearbeiten“ nicht erreichbar")
		}
	}
	l.taste(key.NameReturn, 0)
	if l.e.menue.offen {
		t.Error("Enter auf dem Eintrag lässt das Menü offen")
	}
	if !l.e.s.TextLaeuft() {
		t.Fatal("Enter auf „Text bearbeiten“ startet keine Eingabe")
	}
	if !l.fokus(l.e) {
		t.Error("nach dem Ausführen hat die Leinwand den Fokus nicht")
	}
	l.tippe("x")
	if got := l.inhalt(); got != "Hallox" {
		t.Errorf("getippter Text kam nicht an: %q, soll %q", got, "Hallox")
	}
}

// Dasselbe mit der Maus: Klick auf „Text bearbeiten“, danach kommt getippter Text an.
func TestMenueMausEintragDannTippen(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.titel().Click()
	l.setle()
	if !l.e.menue.offen {
		t.Fatal("Menü nicht offen")
	}
	r := l.e.menue.rahmen[l.index(aktTextBearbeiten)]
	l.klick(r.Min.Add(r.Size().Div(2)))
	if !l.e.s.TextLaeuft() || l.e.menue.offen {
		t.Fatalf("Klick auf den Eintrag: Eingabe %v, Menü offen %v", l.e.s.TextLaeuft(), l.e.menue.offen)
	}
	l.tippe("y")
	if got := l.inhalt(); got != "Halloy" {
		t.Errorf("getippter Text kam nicht an: %q", got)
	}
}

// Per Maus geöffnet: kein Fokus in der Liste (sonst erschiene ein Ring ohne Tastatur).
func TestMenueMausOeffnenLaesstFokus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	// Zeiger über den Menütitel: Hovered. Aus dem Test ohne Zeigerereignis ist Hovered falsch, also wie ein
	// Tastaturklick; die Mausart wird deshalb über einen echten Klick auf den Knopf geprüft.
	l.klick(menueKnopfMitte(l)) // Titel „Datei“ in der Menüzeile (ADR-0031)
	if !l.e.menue.offen {
		t.Fatal("Menütitel nicht getroffen: Klick öffnet das Menü nicht (ADR-0026 Nachtrag 2)")
	}
	for i := range l.e.menueEintraege() {
		if l.fokus(&l.e.menue.eintraege[i]) {
			t.Errorf("per Maus geöffnet, aber Eintrag %d hat den Fokus", i)
		}
	}
}

// menueAuswerten: Klick daneben schließt, grauer Eintrag tut nichts, zweiter Klick auf „Menü“ schließt (F-253).
func TestMenueAuswertenKlicks(t *testing.T) {
	t.Run("Klick daneben schließt", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.titel().Click()
		l.setle()
		if !l.e.menue.offen {
			t.Fatal("Menü nicht offen")
		}
		l.klick(image.Pt(30, 700))
		if l.e.menue.offen {
			t.Error("Klick daneben schließt das Menü nicht")
		}
		if !l.fokus(l.e) {
			t.Error("nach Klick daneben hat die Leinwand den Fokus nicht")
		}
	})
	t.Run("grauer Eintrag tut nichts", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.titel().Click()
		l.setle()
		i := l.index(aktWiederholen)
		if l.e.menueEintraege()[i].an {
			t.Fatal("Vorbedingung: Wiederholen ist grau")
		}
		r := l.e.menue.rahmen[i]
		stand := l.e.s.D.Stand()
		l.klick(r.Min.Add(r.Size().Div(2)))
		if !l.e.menue.offen {
			t.Error("Klick auf einen grauen Eintrag schließt das Menü")
		}
		if l.e.s.D.Stand() != stand || l.e.s.TextLaeuft() {
			t.Error("Klick auf einen grauen Eintrag bewirkt etwas")
		}
	})
	t.Run("zweiter Klick auf Menü schließt", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.titel().Click()
		l.setle()
		l.titel().Click()
		l.setle()
		if l.e.menue.offen {
			t.Error("zweiter Klick auf „Menü“ lässt die Liste offen")
		}
	})
	t.Run("Klick auf die Fläche zwischen Einträgen schließt nicht", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.titel().Click()
		l.setle()
		// Trennlinie zwischen „Kopieren“ und „Schließen“ im Menü Datei (Gruppenabstand).
		a, b := l.e.menue.rahmen[l.index(aktKopieren)], l.e.menue.rahmen[l.index(aktSchliessen)]
		l.klick(image.Pt(a.Min.X+5, (a.Max.Y+b.Min.Y)/2))
		if !l.e.menue.offen {
			t.Error("Klick zwischen Einträgen schließt das Menü")
		}
	})
}

// F-265: Im kleinsten Fenster (480 × 320) liegt jeder Eintrag im Fenster, und der unterste ist per Klick erreichbar.
func TestMenueKleinstesFensterLetzterEintragErreichbar(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(480, 320))
	l.titel().Click()
	l.setle()
	fenster := image.Rectangle{Max: l.fenst}
	for i, en := range l.e.menueEintraege() {
		r := l.e.menue.rahmen[i]
		if !r.In(fenster) || r.Dy() < 20 {
			t.Errorf("Eintrag %d (%s): %v nicht im Fenster %v oder zu niedrig", i, en.text, r, fenster)
		}
	}
	// Der letzte Eintrag („Schließen“) braucht ein Fenster; geprüft wird der vorletzte („1:1“) per Klick.
	i := l.index(aktEins)
	r := l.e.menue.rahmen[i]
	l.klick(r.Min.Add(r.Size().Div(2)))
	if l.e.menue.offen {
		t.Error("Klick auf den vorletzten Eintrag führt nicht aus")
	}
	if l.e.s.Abb.Zoom != 1 {
		t.Errorf("Zoom nach „1:1“: %v", l.e.s.Abb.Zoom)
	}
}

// Fokus auf einem Werkzeugknopf: Der Hinweis erscheint auch ohne Zeiger (F-255 Punkt 4), nur für diesen Knopf.
func TestWerkzeugHinweisBeiTastaturfokus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if n := len(l.e.hinweisKnoepfe(l.gtx())); n != 0 {
		t.Fatalf("ohne Fokus und Zeiger %d Hinweise", n)
	}
	l.fokusiere(&l.e.knoepfe.pfeil)
	h := l.e.hinweisKnoepfe(l.gtx())
	if len(h) != 1 || h[0].w != canvas.WerkzeugPfeil {
		t.Errorf("mit Fokus auf „Pfeil“: %v", h)
	}
	l.fokusiere(&l.e.knoepfe.kasten)
	h = l.e.hinweisKnoepfe(l.gtx())
	if len(h) != 1 || h[0].w != canvas.WerkzeugKasten {
		t.Errorf("mit Fokus auf „Kasten“: %v", h)
	}
	// Bei offenem Menü zeichnet zeichneWerkzeugHinweis nichts (Test über die Zeichnung: kein Absturz, Menü bleibt offen).
	l.titel().Click()
	l.setle()
	if !l.e.menue.offen {
		t.Error("Menü nicht offen")
	}
}

// Verhalten je Menüeintrag: Jeder Eintrag bewirkt, was sein Name sagt (nicht nur „Aktion nicht nil“).
func TestMenueEintraegeVerhalten(t *testing.T) {
	i18n.Setze(nil)
	t.Setenv("WAYLAND_SOCKET", "1") // der Halter startet keinen Prozess; das Fenster böte selbst an
	gtx := layout.Context{}
	neu := func(t *testing.T) (*editor, int) {
		e := testEditor(t)
		id, err := e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "Hallo", Schriftgrad: 12,
			Farbe: canvas.Schwaerzfarbe})
		if err != nil {
			t.Fatal(err)
		}
		e.s.Auswahl = id
		e.s.SetzeFenster(image.Pt(800, 600))
		return e, id
	}
	t.Run("Kopieren", func(t *testing.T) {
		e, _ := neu(t)
		e.fuehreAus(gtx, aktKopieren, true)
		warteKopieren(t, e, gtx)
		if e.meldung.Text != i18n.T("Copied to clipboard") {
			t.Errorf("Meldung %q", e.meldung.Text)
		}
	})
	t.Run("Speichern", func(t *testing.T) {
		e, _ := neu(t)
		aufrufe := 0
		e.speichern = func(*document.Dokument) (Gespeichert, error) { aufrufe++; return Gespeichert{}, nil }
		e.fuehreAus(gtx, aktSpeichern, true)
		warteSpeichern(t, e)
		if aufrufe != 1 || e.s.Geaendert() {
			t.Errorf("Speichern: %d Aufrufe, geändert %v", aufrufe, e.s.Geaendert())
		}
		e.fuehreAus(gtx, aktSpeichern, true)
		if aufrufe != 1 || e.meldung.Text != i18n.T("No changes to save.") {
			t.Errorf("zweites Speichern ohne Änderung: %d Aufrufe, Meldung %q", aufrufe, e.meldung.Text)
		}
	})
	t.Run("Rückgängig und Wiederholen", func(t *testing.T) {
		e, id := neu(t)
		e.s.TextBearbeiten()
		e.s.Tippe("!")
		e.s.TextBestaetigen()
		inhalt := func() string { o, _ := e.s.D.Objekt(id); return o.(document.Text).Inhalt }
		if inhalt() != "Hallo!" {
			t.Fatalf("Vorbedingung: %q", inhalt())
		}
		e.fuehreAus(gtx, aktRueckgaengig, true)
		if inhalt() != "Hallo" {
			t.Errorf("Rückgängig: %q", inhalt())
		}
		e.fuehreAus(gtx, aktWiederholen, true)
		if inhalt() != "Hallo!" {
			t.Errorf("Wiederholen: %q", inhalt())
		}
	})
	t.Run("Text bearbeiten", func(t *testing.T) {
		e, _ := neu(t)
		e.fuehreAus(gtx, aktTextBearbeiten, true)
		if !e.s.TextLaeuft() {
			t.Error("keine Texteingabe")
		}
	})
	t.Run("Löschen", func(t *testing.T) {
		e, id := neu(t)
		e.fuehreAus(gtx, aktLoeschen, true)
		if _, da := e.s.D.Objekt(id); da {
			t.Error("Objekt noch da")
		}
	})
	t.Run("Ganzes Bild", func(t *testing.T) {
		e, _ := neu(t)
		if err := e.s.D.SetzeZuschnitt(image.Rect(5, 5, 30, 20)); err != nil {
			t.Fatal(err)
		}
		if !e.s.KannGanzesBild() {
			t.Fatal("Vorbedingung: Eintrag bedienbar")
		}
		e.fuehreAus(gtx, aktGanzesBild, true)
		if got := e.s.D.Zuschnitt(); got != (image.Rectangle{Max: e.s.D.OriginalGroesse()}) {
			t.Errorf("Zuschnitt nach „Ganzes Bild“: %v", got)
		}
	})
	t.Run("Gestauchtes zurückholen", func(t *testing.T) {
		e, _ := neu(t)
		if err := e.s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 10, Bis: 14}}); err != nil {
			t.Fatal(err)
		}
		e.fuehreAus(gtx, aktZurueckholen, true)
		if len(e.s.D.Streifen()) != 0 {
			t.Errorf("Streifen: %v", e.s.D.Streifen())
		}
	})
	t.Run("Einpassen und 1:1", func(t *testing.T) {
		e, _ := neu(t)
		e.fuehreAus(gtx, aktEins, true)
		if e.s.Abb.Zoom != 1 {
			t.Errorf("1:1: Zoom %v", e.s.Abb.Zoom)
		}
		e.fuehreAus(gtx, aktEinpassen, true)
		if soll := canvas.Einpassen(e.s.D.Groesse(), image.Pt(800, 600), canvas.Rand); e.s.Abb != soll {
			t.Errorf("Einpassen: %v, soll %v", e.s.Abb, soll)
		}
	})
	t.Run("Export blockt bei offenem Dialog", func(t *testing.T) {
		// Der Dialog selbst ist ohne Portal nicht prüfbar (nie gegen den echten Desktop starten): Läuft schon einer,
		// ist der Eintrag grau und die Handlung startet keinen zweiten.
		e, _ := neu(t)
		e.exportOffen = true
		for _, en := range e.menueEintraege() {
			if en.akt == aktExport && en.an {
				t.Error("Export bei offenem Dialog nicht grau")
			}
		}
		e.fuehreAus(gtx, aktExport, true)
		if e.exportAntwort != nil {
			t.Error("zweiter Dialog gestartet")
		}
	})
}
