// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"
)

// TestFusszeileMeldungVorFeldern (F-507, Kritik B-3, Mutant E28): Eine haltende Meldung mit langem Text behält zwei
// Drittel der Breite; kein Fußfeld liegt links davon. Ohne die Meldung reicht das linke Drittel, und im selben Fenster
// liegen Felder links der Zwei-Drittel-Marke (sonst prüfte der Test nichts).
func TestFusszeileMeldungVorFeldern(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(700, 500))
	zweiDrittel := 700 * 2 / 3
	links := 0
	for _, r := range l.e.fussRahmen {
		if r.Min.X < zweiDrittel {
			links++
		}
		if r.Min.X < 700/3 {
			t.Errorf("Feld %v ohne Meldung im linken Drittel", r)
		}
	}
	if links == 0 {
		t.Fatalf("Vorbedingung: ohne Meldung liegt kein Feld links von %d (%v)", zweiDrittel, l.e.fussRahmen)
	}
	l.e.meldeHaltend(haltHinweis, strings.Repeat("Ein langer Hinweis zum Speichern. ", 8), false)
	l.setle()
	if len(l.e.fussRahmen) >= 1 && len(l.e.fussRahmen) == len(l.e.s.FussFelder(l.e.zeigerAnsicht())) {
		t.Fatalf("Vorbedingung: keine Felder fielen weg (%d)", len(l.e.fussRahmen))
	}
	for _, r := range l.e.fussRahmen {
		if r.Min.X < zweiDrittel {
			t.Errorf("Feld %v links der Marke %d: Die haltende Meldung hätte Vorrang (zwei Drittel der Breite)", r, zweiDrittel)
		}
	}
}

// TestHilfetafelPasstAufDieLeinwand (F-509, Kritik B-3, Mutant E29): Die Tastenübersicht ist nie größer als die Fläche
// der Leinwand; ein kleines Fenster (800×600) schneidet sie ab, statt sie darüber hinausragen zu lassen.
func TestHilfetafelPasstAufDieLeinwand(t *testing.T) {
	for _, fenster := range []image.Point{{800, 600}, {500, 300}, {1200, 800}} {
		l := neuerMenueLauf(t, fenster)
		l.e.hilfeOffen = true
		l.setle()
		flaeche := leinwandFlaeche(l.gtx())
		g := l.e.hilfeGroesse
		if g.X <= 0 || g.Y <= 0 {
			t.Fatalf("%v: Tafel ohne Größe (%v), wurde sie gezeichnet?", fenster, g)
		}
		if g.X > flaeche.Dx() || g.Y > flaeche.Dy() {
			t.Errorf("%v: Tafel %v größer als die Leinwand %v", fenster, g, flaeche.Size())
		}
	}
	// Bei 800×600 ist die Tafel ohne Begrenzung zu hoch (eine Spalte, über 30 Zeilen zu 22 Punkten): Sie füllt die Höhe.
	l := neuerMenueLauf(t, image.Pt(800, 600))
	l.e.hilfeOffen = true
	l.setle()
	if g, f := l.e.hilfeGroesse, leinwandFlaeche(l.gtx()); g.Y != f.Dy() {
		t.Errorf("800×600: Tafelhöhe %d, Soll die Leinwandhöhe %d (abgeschnitten)", g.Y, f.Dy())
	}
}
