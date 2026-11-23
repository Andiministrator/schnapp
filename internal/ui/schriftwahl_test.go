// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

func (l *menueLauf) text() document.Text {
	o, _ := l.e.s.D.Objekt(l.id)
	return o.(document.Text)
}

// TestSchriftListeMaus: Klick aufs Feld öffnet die Liste, Klick auf einen Eintrag setzt die Familie als einen Schritt
// und schließt sie; Klick daneben schließt ohne Wahl.
func TestSchriftListeMaus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	if !sw.gezeigt || sw.feldText != fonts.Go {
		t.Fatalf("Abschnitt %v, Feld %q", sw.gezeigt, sw.feldText)
	}
	l.klick(mittenPunkt(sw.feldRahmen))
	if !sw.offen || sw.rahmen.Empty() || len(sw.familien) < 2 || sw.familien[1] != fonts.GoMono {
		t.Fatalf("Liste offen %v, Lage %v, Familien %v", sw.offen, sw.rahmen, sw.familien)
	}
	vor := l.e.s.D.Stand()
	l.klick(mittenPunkt(sw.zeilen[1]))
	if sw.offen || l.text().Schrift != fonts.GoMono || l.e.s.D.Stand() == vor || !l.fokus(&sw.feld) {
		t.Fatalf("nach Klick auf Go Mono: offen %v, Schrift %q, Fokus %s", sw.offen, l.text().Schrift, menueFokusName(l))
	}
	l.setle()
	if sw.feldText != fonts.GoMono {
		t.Errorf("Feld zeigt %q", sw.feldText)
	}
	l.klick(mittenPunkt(sw.feldRahmen))
	vor = l.e.s.D.Stand()
	l.klick(image.Pt(600, 400)) // daneben, auf die Leinwand
	if sw.offen || l.e.s.D.Stand() != vor {
		t.Errorf("Klick daneben: offen %v, Dokument geändert %v", sw.offen, l.e.s.D.Stand() != vor)
	}
}

// TestSchriftListeTastatur: Enter am Feld öffnet die Liste am gezeigten Eintrag, Pfeile wählen, Enter setzt, Esc
// verwirft; getippte Buchstaben springen und wechseln kein Werkzeug.
func TestSchriftListeTastatur(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	l.fokusiere(&sw.feld)
	l.taste(key.NameReturn, 0)
	if !sw.offen || sw.cursor != 0 {
		t.Fatalf("Enter: offen %v, Cursor %d", sw.offen, sw.cursor)
	}
	l.taste(key.NameDownArrow, 0)
	l.taste(key.NameEscape, 0)
	if sw.offen || l.text().Schrift != "" || !l.fokus(&sw.feld) {
		t.Fatalf("Esc: offen %v, Schrift %q, Fokus %s", sw.offen, l.text().Schrift, menueFokusName(l))
	}
	l.taste(key.NameReturn, 0)
	w := l.e.s.Werkzeug
	zeit := time.Unix(100, 0)
	sw.jetzt = func() time.Time { return zeit }
	for _, z := range []string{"G", "O", " ", "M"} { // Gio liefert Taste und Text
		l.r.Queue(key.Event{Name: key.Name(z), State: key.Press})
		l.r.Queue(key.EditEvent{Text: z})
		l.setle()
	}
	if l.e.s.Werkzeug != w || sw.cursor != 1 {
		t.Fatalf("Tippen „go m“: Werkzeug %v (vorher %v), Cursor %d", l.e.s.Werkzeug, w, sw.cursor)
	}
	l.taste(key.NameUpArrow, 0)
	l.taste(key.NameDownArrow, 0)
	l.taste(key.NameEnd, 0)
	l.taste(key.NameHome, 0)
	l.taste(key.NameDownArrow, 0)
	l.taste(key.NameReturn, 0)
	if sw.offen || l.text().Schrift != fonts.GoMono {
		t.Errorf("Enter auf Go Mono: offen %v, Schrift %q", sw.offen, l.text().Schrift)
	}
	// Nach der Pause beginnt ein neuer Anfang.
	l.taste(key.NameReturn, 0)
	l.tippe("go m")
	zeit = zeit.Add(2 * tippPause)
	l.tippe("g")
	if sw.cursor != 0 {
		t.Errorf("nach der Pause: Cursor %d, erwartet 0 (Go)", sw.cursor)
	}
	// Werkzeugwechsel ohne Text schließt die Liste.
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	if sw.offen || sw.gezeigt {
		t.Errorf("Pfeil: Liste offen %v, Abschnitt %v", sw.offen, sw.gezeigt)
	}
}

// TestSchriftGroesseStilAusrichtung: Größe − / +, die Umschalter und die Ausrichtung ändern den gewählten Text, je ein
// Rückgängig-Schritt; eine fehlende Familie steht mit „(missing)“ im Feld.
func TestSchriftGroesseStilAusrichtung(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	vor := l.e.s.D.Stand()
	l.klick(mittenPunkt(sw.grRahmen[1])) // 12 → 14
	l.klick(mittenPunkt(sw.stilRahmen[0]))
	l.klick(mittenPunkt(sw.stilRahmen[3]))
	l.klick(mittenPunkt(sw.ausrRahmen[2]))
	tx := l.text()
	if tx.Schriftgrad != 14 || !tx.Fett || !tx.Durchgestrichen || tx.Kursiv || tx.Ausrichtung != document.AusrichtungRechts {
		t.Fatalf("Text %+v", tx)
	}
	if n := schritteSeit(l, vor); n != 4 {
		t.Errorf("%d Schritte, erwartet 4", n)
	}
	if sw.grText != "14 px" {
		t.Errorf("Größe zeigt %q", sw.grText)
	}
	l.klick(mittenPunkt(sw.grRahmen[0]))
	l.klick(mittenPunkt(sw.grRahmen[0]))
	if g := l.text().Schriftgrad; g != 10 {
		t.Errorf("zweimal −: %v", g)
	}
	l.e.s.SetzeSchrift("Example Missing")
	l.setle()
	if sw.feldText != "Example Missing (missing)" {
		t.Errorf("Feld %q", sw.feldText)
	}
	// Unter der Hilfetafel wirken die Knöpfe nicht.
	l.e.hilfeOffen = true
	vor = l.e.s.D.Stand()
	l.klick(mittenPunkt(sw.stilRahmen[1]))
	if l.e.s.D.Stand() != vor {
		t.Error("Stil unter der Hilfetafel geändert")
	}
}

// TestSchriftListeInDerTexteingabe (Sitzungs-QA B-1): Nach der Wahl in der Texteingabe liegt der Fokus an der
// Leinwand, weitere Zeichen kommen im Text an.
func TestSchriftListeInDerTexteingabe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugText)
	l.setle()
	f := l.e.leinwandFlaeche(l.gtx())
	p := l.e.s.Abb.ZuFenster(document.Pt(5, 20)) // trifft den Testtext „Hallo“: die Eingabe gilt ihm
	l.klick(f.Min.Add(image.Pt(int(p.X), int(p.Y))))
	l.fokusiere(l.e)
	l.tippe("a")
	if !l.e.s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	l.klick(mittenPunkt(sw.feldRahmen))
	l.klick(mittenPunkt(sw.zeilen[1]))
	l.tippe("b")
	tx, ok := l.e.s.TextObjekt()
	if !ok || tx.Inhalt != "Halloab" || tx.Schrift != fonts.GoMono || !l.fokus(l.e) {
		t.Errorf("nach der Wahl: Eingabe %v, Inhalt %q, Schrift %q, Fokus %s", ok, tx.Inhalt, tx.Schrift, menueFokusName(l))
	}
}

// TestSchriftListeOhneLeiste (Sitzungs-QA B-6): Wird die Leiste bei offener Liste ausgeblendet, schließt die Liste und
// die Kürzel wirken wieder.
func TestSchriftListeOhneLeiste(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	l.klick(mittenPunkt(sw.feldRahmen))
	if !sw.offen {
		t.Fatal("Liste nicht offen")
	}
	l.fenst = image.Pt(700, 800)
	l.setle()
	l.taste("P", 0)
	if sw.offen || l.e.s.Werkzeug != canvas.WerkzeugVerpixeln {
		t.Errorf("schmales Fenster: Liste offen %v, Werkzeug %v", sw.offen, l.e.s.Werkzeug)
	}
}
