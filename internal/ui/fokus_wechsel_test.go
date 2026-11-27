// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"
)

// F-912 (Verify zu F-907): Enter an einer Vorschau der Leiste öffnet sie; der Tastaturfokus bleibt in der Leiste, damit
// Pfeil und Enter die nächste öffnen (ADR-0036 Punkt 1). Ohne Fokus in der Leiste (Wechsel aus dem Code) holt ihn die
// Leinwand.
func TestFokusNachWechselAusDerLeiste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.e.fokusHolen = false // wie nach dem ersten Bild eines echten Editors
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[2])
	l.taste(key.NameReturn, 0)
	l.ruhig()
	l.setle()
	if l.fokus(l.e) || !l.fokus(&k.knoepfe[2]) {
		t.Fatalf("nach Enter: Leinwand %v, Vorschau %v; Soll Fokus an der Vorschau", l.fokus(l.e), l.fokus(&k.knoepfe[2]))
	}
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameReturn, 0)
	l.ruhig()
	if n := len(l.aufrufe()); n != 2 {
		t.Errorf("%d Öffnen, Soll 2: Pfeil und Enter blieben nicht in der Leiste", n)
	}
	// Wechsel ohne Fokus in der Leiste: die Leinwand bekommt ihn (uebernimm setzt fokusHolen).
	l.fokusiere(nil)
	l.e.wechselAnfrage(l.pfade[0])
	l.ruhig()
	l.setle()
	if !l.fokus(l.e) {
		t.Error("nach einem Wechsel ohne Fokus in der Leiste hat die Leinwand keinen Fokus")
	}
}

// F-912 Fall B: Entf an der offenen Aufnahme der Leiste wechselt zur Nachbarin und legt sie in den Papierkorb; der
// Fokus bleibt in der Leiste.
func TestFokusNachEntfInDerLeiste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.fokusHolen = false
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[1]) // die offene
	l.taste(key.NameDeleteForward, 0)
	l.ruhig()
	l.warte(func() bool { return len(k.l.Eintraege()) == 2 })
	l.setle()
	inLeiste := false
	for i := range k.knoepfe {
		if l.fokus(&k.knoepfe[i]) {
			inLeiste = true
		}
	}
	if !inLeiste || l.fokus(l.e) {
		t.Errorf("nach Entf: Fokus in der Leiste %v, Leinwand %v", inLeiste, l.fokus(l.e))
	}
}

// F-913: Hinter Menü, Einstellungstafel und voller Ansicht der Bibliothek nimmt sich die Leinwand den Fokus nicht; der
// Merker bleibt, bis nichts mehr davorliegt.
func TestFokusHolenNichtHinterTafeln(t *testing.T) {
	for _, fall := range []struct {
		name         string
		setze, loese func(e *editor)
	}{
		{"Menü", func(e *editor) { e.menue.offen = true }, func(e *editor) { e.menue.offen = false }},
		{"Tafel", func(e *editor) { e.einst.offen = true }, func(e *editor) { e.einst.offen = false }},
		{"Bibliothek", func(e *editor) { e.bibl.offen = true }, func(e *editor) { e.bibl.offen = false }},
	} {
		l := neuerMenueLauf(t, image.Pt(1200, 900))
		l.fokusiere(&l.e.menue.titel[0])
		fall.setze(l.e)
		l.e.fokusHolen = true
		l.e.tasten(l.gtx()) // ohne Zeichnen: eine künstlich offene Bibliothek ließe sich nicht zeichnen
		if l.fokus(l.e) || !l.e.fokusHolen {
			t.Errorf("%s: Leinwand nimmt den Fokus dahinter (Fokus %v, Merker %v)", fall.name, l.fokus(l.e), l.e.fokusHolen)
		}
		fall.loese(l.e)
		l.setle() // wie im Fenster: tasten läuft im Bild, der Fokusbefehl wirkt dort
		if l.e.fokusHolen || !l.fokus(l.e) {
			t.Errorf("%s: nach dem Schließen kein Fokus auf der Leinwand (Merker %v)", fall.name, l.e.fokusHolen)
		}
	}
}
