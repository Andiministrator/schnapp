// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #220, F-574: Reste des Farbwählers aus der Sitzungs-QA vom 23.11.2026 abends.

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// textEingabeStarten startet im Testlauf eine Texteingabe am Testtext („Hallo“) und tippt ein Zeichen.
func (l *menueLauf) textEingabeStarten() {
	l.t.Helper()
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugText)
	l.setle()
	f := l.e.leinwandFlaeche(l.gtx())
	q := l.e.s.Abb.ZuFenster(document.Pt(5, 20)) // trifft den Testtext „Hallo“: die Eingabe gilt ihm
	l.klick(f.Min.Add(image.Pt(int(q.X), int(q.Y))))
	l.fokusiere(l.e)
	l.tippe("a")
	if !l.e.s.TextLaeuft() {
		l.t.Fatal("keine Texteingabe")
	}
}

// tabTaste liefert Tab so, wie das Fenster es tut (third_party/gio/app/window.go): als Systemereignis; nimmt es kein
// Filter an, fährt der Fokus weiter bzw. zurück.
func (l *menueLauf) tabTaste(m key.Modifiers) {
	l.t.Helper()
	l.r.WakeupTime() // frühere Weckrufe verwerfen, wie das Fenster nach jedem Bild
	l.r.Queue(input.SystemEvent{Event: key.Event{Name: key.NameTab, Modifiers: m, State: key.Press}})
	_, behandelt := l.r.WakeupTime()
	if !behandelt {
		if m == key.ModShift {
			l.r.MoveFocus(key.FocusBackward)
		} else {
			l.r.MoveFocus(key.FocusForward)
		}
	}
	l.setle()
}

// F-574 (Tab): Läuft eine Texteingabe und steht der Fokus im Hex-Feld, verlässt Tab das Feld, übernimmt die Eingabe
// und löst die Texteingabe nicht aus ihrer Bahn. Heute fängt der Editor Tab während der Texteingabe ab (F-153), auch
// wenn der Fokus in einem Eingabefeld der Leiste liegt.
func TestTabImHexFeldWaehrendTexteingabe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.textEingabeStarten()
	l.klick(mittenPunkt(fw.hex.rahmen))
	if !l.fokus(&fw.hex) {
		t.Fatalf("Klick fokussiert das Hex-Feld nicht (%s)", menueFokusName(l))
	}
	l.tabTaste(0)
	if l.fokus(&fw.hex) {
		t.Errorf("Tab verließ das Hex-Feld nicht (Fokus %s)", menueFokusName(l))
	}
}

// F-574 (Tab, Deckkraft): dasselbe im Deckkraft-Feld, mit Umschalt rückwärts.
func TestUmschaltTabImDeckkraftFeldWaehrendTexteingabe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.textEingabeStarten()
	l.klick(mittenPunkt(fw.deckkraft.rahmen))
	if !l.fokus(&fw.deckkraft) {
		t.Fatalf("Klick fokussiert das Deckkraft-Feld nicht (%s)", menueFokusName(l))
	}
	l.tabTaste(key.ModShift)
	if l.fokus(&fw.deckkraft) {
		t.Errorf("Umschalt+Tab verließ das Deckkraft-Feld nicht (Fokus %s)", menueFokusName(l))
	}
}

// F-574 (Rücktaste): Im unberührten Deckkraft-Feld löscht die Rücktaste die Ziffer, nicht das Prozentzeichen mit
// dem Leerzeichen; das gewählte Objekt bleibt.
func TestRuecktasteImUnberuehrtenDeckkraftFeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.deckkraft.rahmen))
	if !l.fokus(&fw.deckkraft) {
		t.Fatalf("Klick fokussiert das Deckkraft-Feld nicht (%s)", menueFokusName(l))
	}
	l.e.s.SetzeDeckkraftText("50")
	l.setle()
	l.klick(mittenPunkt(fw.deckkraft.rahmen))
	l.taste(key.NameDeleteBackward, 0)
	if fw.deckkraft.gezeigt != "5" {
		t.Errorf("nach Rücktaste im unberührten Feld: %q, erwartet %q", fw.deckkraft.gezeigt, "5")
	}
	if _, ok := l.e.s.D.Objekt(l.id); !ok {
		t.Error("Rücktaste im Feld löschte das gewählte Objekt")
	}
}

// F-574 (Fokus nil): Verschwindet der Farbabschnitt bei Fokus im Feld (Fenster wird zu schmal, die Leiste klappt
// selbst ein), geht der Fokus an die Leinwand statt ins Leere.
func TestFokusWennFarbabschnittVerschwindet(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	if !l.fokus(&fw.hex) {
		t.Fatalf("Klick fokussiert das Hex-Feld nicht (%s)", menueFokusName(l))
	}
	l.fenst = image.Pt(700, 800) // unter 752 dp: die Leiste ist eingeklappt
	l.setle()
	if l.e.seiteSichtbar(l.gtx()) {
		t.Fatal("Leiste bei 700 dp noch sichtbar")
	}
	if !l.fokus(l.e) {
		t.Errorf("Fokus nach dem Verschwinden des Felds: %s, erwartet die Leinwand", menueFokusName(l))
	}
}

// F-574 (Zeiger-Cancel): Bricht das System ein Ziehen in der Farbfläche ab (pointer.Cancel), stellt die Oberfläche die
// Farbe wie vor dem Ziehen her, beendet den Zug und lässt keinen Rückgängig-Schritt zurück.
func TestZeigerCancelImFarbzug(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(l.e.seite.farbRahmen[0])) // Rot, deckend: Ausgang
	vor, stand := l.textHintergrund(), l.e.s.D.Stand()
	a, b := lageIn(fw.ton, 0.3, 0.5), lageIn(fw.ton, 0.6, 0.5)
	l.maus(pointer.Move, a, 0)
	l.setle()
	l.maus(pointer.Press, a, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Move, b, pointer.ButtonPrimary)
	l.setle()
	if !l.e.s.FarbZugLaeuft() || l.textHintergrund() == vor {
		t.Fatalf("Vorbedingung: Zug läuft %v, Schild %v (vorher %v)", l.e.s.FarbZugLaeuft(), l.textHintergrund(), vor)
	}
	l.r.Queue(pointer.Event{Kind: pointer.Cancel, Source: pointer.Mouse})
	l.setle()
	if l.e.s.FarbZugLaeuft() || fw.zieht != reglerKeiner || l.textHintergrund() != vor || l.e.s.D.Stand() != stand {
		t.Errorf("nach Cancel: Zug %v, Regler %v, Schild %v (vorher %v), Stand geändert %v", l.e.s.FarbZugLaeuft(),
			fw.zieht, l.textHintergrund(), vor, l.e.s.D.Stand() != stand)
	}
}
