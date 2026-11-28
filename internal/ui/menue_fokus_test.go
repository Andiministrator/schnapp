// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests zur Nacharbeit F-255: Fokus nach Esc, Leertaste, schnelle Tastenfolgen.

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// menueKnopfMitte: Mitte des Titels des gewählten Menüs (ADR-0031).
func menueKnopfMitte(l *menueLauf) image.Point {
	r := l.e.menue.titelRahmen[l.e.menue.welches]
	return r.Min.Add(r.Max).Div(2)
}

func drueckeUndLoese(l *menueLauf, n key.Name) {
	l.r.Queue(key.Event{Name: n, State: key.Press})
	l.setle()
	l.r.Queue(key.Event{Name: n, State: key.Release})
	l.setle()
}

func menueFokusNr(l *menueLauf) int {
	for i := range l.e.menueEintraege() {
		if l.fokus(&l.e.menue.eintraege[i]) {
			return i
		}
	}
	return -1
}

func menueFokusName(l *menueLauf) string {
	switch {
	case l.fokus(l.e):
		return "Leinwand"
	case l.fokus(l.titel()):
		return "Menütitel"
	case menueFokusNr(l) >= 0:
		return "Eintrag " + l.e.menueEintraege()[menueFokusNr(l)].text
	case l.e.fokussierterKnopf(l.gtx()) != nil:
		return "anderer Knopf"
	}
	return "nil/unbekannt"
}

// V1: Texteingabe läuft, Menü per Maus geöffnet, Esc: Der Text muss weiter tippbar sein.
func TestMenueTextLaeuftMausMenueEscDannTippen(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.e.s.TextBearbeiten()
	l.setle()
	l.tippe("a")
	if l.inhalt() != "Halloa" {
		t.Fatalf("Vorbedingung: %q", l.inhalt())
	}
	l.klick(menueKnopfMitte(l))
	if !l.e.menue.offen {
		t.Fatal("Menütitel nicht getroffen")
	}
	if !l.e.s.TextLaeuft() {
		t.Fatal("Öffnen per Maus beendet die Texteingabe")
	}
	t.Logf("Fokus nach Maus-Öffnen: %s", menueFokusName(l))
	l.tippe("Q") // Nachtrag 1: wirkt nicht
	if l.inhalt() != "Halloa" {
		t.Errorf("getippter Text bei offenem Menü kam an: %q", l.inhalt())
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen {
		t.Fatal("Esc schließt nicht")
	}
	if !l.e.s.TextLaeuft() {
		t.Fatal("Esc bei offenem Menü beendet die Texteingabe")
	}
	t.Logf("Fokus nach Esc: %s", menueFokusName(l))
	l.tippe("b")
	if got := l.inhalt(); got != "Halloab" {
		t.Errorf("Texteingabe läuft, Menü per Maus auf, Esc: getippter Text geht verloren: %q, soll %q (Fokus: %s)",
			got, "Halloab", menueFokusName(l))
	}
	// Leertaste: soll ein Leerzeichen sein, kein Menü öffnen.
	drueckeUndLoese(l, key.NameSpace)
	if l.e.menue.offen {
		t.Errorf("Leertaste während der Texteingabe öffnet das Menü (Fokus am Menütitel)")
	}
}

// V2: Ohne Texteingabe, Menü per Maus auf, Esc, dann Enter: soll „Text bearbeiten“ (Auswahl ist ein Text), nicht Menü.
func TestMenueMausMenueEscDannEnter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.klick(menueKnopfMitte(l))
	if !l.e.menue.offen {
		t.Fatal("Menütitel nicht getroffen")
	}
	if i := menueFokusNr(l); i >= 0 {
		t.Errorf("per Maus geöffnet, Fokus auf Eintrag %d", i)
	}
	l.taste(key.NameEscape, 0)
	t.Logf("Fokus nach Maus-Öffnen und Esc: %s", menueFokusName(l))
	l.taste(key.NameReturn, 0)
	if l.e.menue.offen || !l.e.s.TextLaeuft() {
		t.Errorf("Maus-Öffnen, Esc, Enter: Menü offen %v, Texteingabe %v (erwartet: Enter bearbeitet den gewählten Text)",
			l.e.menue.offen, l.e.s.TextLaeuft())
	}
}

// V3: Der fokussierte Eintrag wird bei offenem Menü grau (Klickfläche verschwindet, Gio setzt den Fokus auf nil).
func TestMenueFokussierterEintragWirdGrau(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.zuMenue(aktTextBearbeiten) // Menü „Bearbeiten“
	l.oeffneMitTaste()
	ziel := l.index(aktTextBearbeiten)
	for n := 0; menueFokusNr(l) != ziel; n++ {
		if n > 20 {
			t.Fatal("nicht erreichbar")
		}
		l.taste(key.NameDownArrow, 0)
	}
	l.e.s.Auswahl = 0
	l.setle()
	if l.e.menueEintraege()[ziel].an {
		t.Fatal("Vorbedingung: Eintrag ist jetzt grau")
	}
	t.Logf("Fokus nach Grauwerden: %s", menueFokusName(l))
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || l.e.s.TextLaeuft() {
		t.Errorf("Enter auf grau gewordenem Eintrag wirkt: offen %v, Text %v", l.e.menue.offen, l.e.s.TextLaeuft())
	}
	l.taste(key.NameDownArrow, 0)
	// Nächster erreichbarer Eintrag im Menü „Bearbeiten“: Löschen ist ohne Auswahl grau und wird übersprungen; „Alles
	// auswählen“ ist an, solange es Objekte gibt (ADR-0039 Punkt 2).
	if i := menueFokusNr(l); i != l.index(aktAllesWaehlen) {
		t.Errorf("Runter nach Grauwerden: Fokus %d (%s), soll Alles auswählen", i, menueFokusName(l))
	}
	// Danach: Farben tauschen ist ohne Auswahl grau; „Text aus dem Bild kopieren“ ist ohne Suchergebnis grau, aber mit
	// Erklärung erreichbar (ADR-0026 Nachtrag 8).
	l.taste(key.NameDownArrow, 0)
	if i := menueFokusNr(l); i != l.index(aktTextKopieren) {
		t.Errorf("Runter: Fokus %d (%s), soll Text aus dem Bild kopieren", i, menueFokusName(l))
	}
	// Hoch: Alles auswählen, Einfügen, Kopieren, dann über die grauen Ausschneiden und Wiederholen auf Rückgängig
	// (ADR-0045 Punkt 6).
	for range 4 {
		l.taste(key.NameUpArrow, 0)
	}
	if i := menueFokusNr(l); i != l.index(aktRueckgaengig) {
		t.Errorf("Hoch: Fokus %d (%s)", i, menueFokusName(l))
	}
	l.taste(key.NameEscape, 0)
	if l.e.menue.offen || !l.fokus(l.titel()) {
		t.Errorf("Esc: offen %v, Fokus %s", l.e.menue.offen, menueFokusName(l))
	}
}

// V4: Nachtrag 1 bleibt dicht: andere Tasten und Text wirken bei offenem Menü nicht.
func TestMenueOffenesMenueSperrtTasten(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.TextBearbeiten()
	l.e.s.Tippe("!")
	l.e.s.TextBestaetigen() // es gibt etwas zum Rückgängigmachen
	l.setle()
	l.oeffneMitTaste()
	stand, wz, abb := l.e.s.D.Stand(), l.e.s.Werkzeug, l.e.s.Abb
	for _, k := range []struct {
		n key.Name
		m key.Modifiers
	}{{"A", 0}, {"T", 0}, {"X", 0}, {key.NameDeleteForward, 0}, {key.NameDeleteBackward, 0}, {"Z", key.ModShortcut},
		{"C", key.ModShortcut}, {"S", key.ModShortcut}, {"0", key.ModShortcut}, {"1", key.ModShortcut},
		{key.NameLeftArrow, 0}, {key.NameRightArrow, 0}, {key.NamePageDown, 0}, {key.NamePageUp, 0}} {
		l.taste(k.n, k.m)
		if !l.e.menue.offen || l.e.s.D.Stand() != stand || l.e.s.Werkzeug != wz || l.e.s.Abb != abb || l.e.s.TextLaeuft() ||
			l.inhalt() != "Hallo!" {
			t.Fatalf("Taste %q/%v wirkt bei offenem Menü", k.n, k.m)
		}
	}
	l.tippe("zzz")
	if l.inhalt() != "Hallo!" || l.e.s.D.Stand() != stand {
		t.Errorf("getippter Text wirkt bei offenem Menü: %q", l.inhalt())
	}
	if l.e.meldung.Text != "" {
		t.Errorf("Meldung nach gesperrten Tasten: %q", l.e.meldung.Text)
	}
}

// V5: Echte Tastenfolge (Drücken und Loslassen): Enter auf dem Menütitel öffnet genau einmal, Enter auf dem Eintrag
// führt genau einmal aus; Leertaste ebenso.
func TestMenueEnterUndLeertasteMitLoslassen(t *testing.T) {
	for _, n := range []key.Name{key.NameReturn, key.NameSpace} {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.zuMenue(aktEins) // Menü „Ansicht“: Einpassen, 1:1
		l.fokusiere(l.titel())
		drueckeUndLoese(l, n)
		if !l.e.menue.offen {
			t.Fatalf("%q auf dem Menütitel öffnet nicht (oder öffnet und schließt)", n)
		}
		if menueFokusNr(l) != 0 {
			t.Errorf("%q: Fokus %s, soll erster Eintrag", n, menueFokusName(l))
		}
		l.taste(key.NameDownArrow, 0)
		if menueFokusNr(l) != l.index(aktEins) {
			t.Fatalf("%q: Fokus %s, soll 1:1", n, menueFokusName(l))
		}
		drueckeUndLoese(l, n)
		if l.e.menue.offen || l.e.s.Abb.Zoom != 1 {
			t.Errorf("%q auf Eintrag „1:1“: offen %v, Zoom %v", n, l.e.menue.offen, l.e.s.Abb.Zoom)
		}
		if !l.fokus(l.e) {
			t.Errorf("%q: nach Ausführen Fokus %s, soll Leinwand", n, menueFokusName(l))
		}
	}
}

// V6: Schnelle Folge ohne Bild dazwischen: Enter (öffnet), sofort Runter.
func TestMenueSchnelleFolgeEnterRunter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.titel())
	l.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	l.r.Queue(key.Event{Name: key.NameReturn, State: key.Release})
	l.r.Queue(key.Event{Name: key.NameDownArrow, State: key.Press})
	l.r.Queue(key.Event{Name: key.NameDownArrow, State: key.Release})
	l.setle()
	l.setle()
	t.Logf("offen %v, Fokus %s (ideal: Eintrag Export…)", l.e.menue.offen, menueFokusName(l))
	if !l.e.menue.offen {
		t.Errorf("Menü nicht offen")
	}
	if menueFokusNr(l) != 1 {
		t.Errorf("Enter und sofort Runter: Fokus %s, soll zweiter Eintrag", menueFokusName(l))
	}
}

// V7: Heuristik „Zeiger nicht über dem Knopf = Tastatur“.
func TestMenueArtDesKlicksAusEreignis(t *testing.T) {
	t.Run("Zeiger ruht auf dem Knopf, Leertaste", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		p := menueKnopfMitte(l)
		l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
		l.setle()
		if !l.titel().Hovered() {
			t.Fatal("Vorbedingung: Zeiger über dem Knopf")
		}
		l.fokusiere(l.titel())
		drueckeUndLoese(l, key.NameSpace)
		if !l.e.menue.offen {
			t.Fatal("Leertaste öffnet nicht")
		}
		if menueFokusNr(l) != 0 {
			t.Errorf("Leertaste bei ruhendem Zeiger: Fokus %s, soll erster Eintrag (ADR-0026 Nachtrag 2)", menueFokusName(l))
		}
		l.taste(key.NameDownArrow, 0)
		t.Logf("danach Runter: Fokus %s", menueFokusName(l))
	})
	t.Run("Zeiger ruht auf dem Knopf, Enter", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		p := menueKnopfMitte(l)
		l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
		l.setle()
		l.fokusiere(l.titel())
		drueckeUndLoese(l, key.NameReturn)
		if !l.e.menue.offen || menueFokusNr(l) != 0 {
			t.Errorf("Enter bei ruhendem Zeiger: offen %v, Fokus %s", l.e.menue.offen, menueFokusName(l))
		}
	})
	t.Run("Tippen mit dem Finger", func(t *testing.T) {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		l.fokusiere(l.e)
		p := menueKnopfMitte(l)
		pos := f32.Pt(float32(p.X), float32(p.Y))
		l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: pos})
		l.setle()
		l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: pos})
		l.setle()
		if !l.e.menue.offen {
			t.Fatal("Fingertipp öffnet das Menü nicht (ADR-0026 Nachtrag 2)")
		}
		if i := menueFokusNr(l); i >= 0 {
			t.Errorf("Fingertipp gilt als Tastatur: Fokusring auf Eintrag %d", i)
		}
	})
}

// V8: Nichts bedienbar: kein Hängen, -1.
func TestMenueAllesGrau(t *testing.T) {
	en := make([]menueEintrag, 12)
	for _, von := range []int{-1, 0, 5, 11, 99, -7} {
		for _, dir := range []int{1, -1} {
			if got := menueNaechster(en, von, dir); got != -1 {
				t.Errorf("von %d dir %d: %d", von, dir, got)
			}
		}
	}
	en[4].an = true
	for _, von := range []int{-1, 0, 4, 11} {
		for _, dir := range []int{1, -1} {
			if got := menueNaechster(en, von, dir); got != 4 {
				t.Errorf("einer an, von %d dir %d: %d", von, dir, got)
			}
		}
	}
}

// V9: Tab-Fahrt (Gio MoveFocus) bei offenem Menü: wohin wandert der Fokus, bleibt das Menü bedienbar?
func TestMenueTabFahrtBeiOffenemMenue(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.oeffneMitTaste()
	weg := []string{menueFokusName(l)}
	for range 14 {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		weg = append(weg, menueFokusName(l))
		// Ein grauer Eintrag mit Erklärung ist per Tastatur erreichbar (ADR-0026 Nachtrag 8; im Testfenster ohne
		// Bibliothek: Datei → Library), einer ohne nicht.
		if i := menueFokusNr(l); i >= 0 && !l.e.menueEintraege()[i].an && l.e.menueEintraege()[i].erklaerung == "" {
			t.Errorf("Tab erreicht grauen Eintrag %d", i)
		}
	}
	t.Logf("Tab-Fahrt ab erstem Eintrag: %v", weg)
	// Nach der Tab-Fahrt auf einen Leistenknopf: Pfeil runter muss wieder in die Liste führen.
	l.fokusiere(&l.e.knoepfe.farbe)
	l.taste(key.NameDownArrow, 0)
	if menueFokusNr(l) < 0 {
		t.Errorf("Pfeil runter von einem Leistenknopf aus: Fokus %s", menueFokusName(l))
	}
	// Tab-Reihenfolge bei geschlossenem Menü: Menütitel erreichbar per Umschalt+Tab von der Leinwand?
	l.taste(key.NameEscape, 0)
	l.fokusiere(l.e)
	rueck := []string{}
	for range 4 {
		l.r.MoveFocus(key.FocusBackward)
		l.setle()
		rueck = append(rueck, menueFokusName(l))
	}
	t.Logf("Umschalt+Tab ab Leinwand: %v", rueck)
}

// V10: Per Tastatur geöffnet (Fokus in der Liste), dann Mausklick auf den Menütitel und Klick daneben: Fokus bleibt
// nirgends hängen, getippter Text käme an.
func TestMenueTastaturOffenDannMaus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.oeffneMitTaste()
	l.klick(menueKnopfMitte(l))
	if l.e.menue.offen {
		t.Errorf("Klick auf den Menütitel bei offenem Menü schließt nicht")
	}
	if !l.fokus(l.e) {
		t.Errorf("danach Fokus %s, soll Leinwand", menueFokusName(l))
	}
	l.taste(key.NameReturn, 0) // Text bearbeiten
	l.tippe("k")
	if l.inhalt() != "Hallok" {
		t.Errorf("Text nach Schließen per Mausklick: %q (Fokus %s, offen %v)", l.inhalt(), menueFokusName(l), l.e.menue.offen)
	}
}

// V11: Werkzeug-Hinweis: Fokus auf einem Knopf, Zeiger über einem anderen.
func TestMenueHinweisFokusUndZeiger(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(&l.e.knoepfe.fussZoom) // kein Werkzeugknopf
	if n := len(l.e.hinweisKnoepfe(l.gtx())); n != 0 {
		t.Errorf("Fokus auf Speichern: %d Hinweise", n)
	}
	l.fokusiere(&l.e.knoepfe.pfeil)
	if l.e.s.Werkzeug == canvas.WerkzeugPfeil {
		t.Errorf("Fokus allein wählt das Werkzeug")
	}
}
