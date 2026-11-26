// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #355: Die offene Aufnahme trägt in der Leiste einen orangen Punkt, solange sie ungesicherte Änderungen hat;
// dieselbe Quelle wie das Feld „ungesichert“ der Fußzeile (canvas.Sitzung.Ungesichert).

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/library"
)

func TestLeistePunktNurBeiUngesichertAnDerOffenen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.e.s.Gespeichert() // der Testeditor beginnt mit einem neuen, ungespeicherten Dokument
	l.ruhig()
	if k.punkt != (image.Rectangle{}) {
		t.Fatalf("Punkt %v ohne Änderung", k.punkt)
	}
	if _, err := l.e.s.D.Hinzufuegen(wzKasten()); err != nil {
		t.Fatal(err)
	}
	l.ruhig()
	if !l.e.s.Ungesichert() {
		t.Fatal("Vorbedingung: Sitzung nicht ungesichert")
	}
	offen := -1
	for i := k.von; i < k.bis; i++ {
		if k.l.IstAktuell(i) {
			offen = i
		}
	}
	if offen < 0 {
		t.Fatal("keine offene Vorschau gezeichnet")
	}
	kr := k.kacheln[offen-k.von]
	if k.punkt.Empty() || !k.punkt.In(kr) {
		t.Errorf("Punkt %v nicht in der offenen Vorschau %v", k.punkt, kr)
	}
	// Ecke oben links, nicht dort, wo das ✕ (oben rechts) erscheint.
	if k.punkt.Max.X > kr.Min.X+kr.Dx()/2 || k.punkt.Max.Y > kr.Min.Y+kr.Dy()/2 {
		t.Errorf("Punkt %v nicht oben links in %v", k.punkt, kr)
	}
	// Der Hinweis der offenen nennt die ungesicherten Änderungen.
	l.maus(pointer.Move, kr.Min.Add(image.Pt(kr.Dx()/2, kr.Dy()/2)), 0)
	l.ruhig()
	if !strings.Contains(k.hinweis, "unsaved") {
		t.Errorf("Hinweis %q ohne ungesicherte Änderungen", k.hinweis)
	}
	// Nach dem Speichern ist der Punkt weg.
	l.e.s.Gespeichert()
	l.ruhig()
	if k.punkt != (image.Rectangle{}) {
		t.Errorf("Punkt %v nach dem Speichern", k.punkt)
	}
}

func TestLeisteHinweisUngesichert(t *testing.T) {
	e := library.Eintrag{Pfad: "2026-11/a.png", Erstellt: time.Date(2026, 11, 26, 9, 0, 0, 0, time.Local), Breite: 800, Hoehe: 600}
	if h := leisteHinweis(e, true, true); !strings.Contains(h, "unsaved") {
		t.Errorf("offen und ungesichert: %q", h)
	}
	if h := leisteHinweis(e, true, false); strings.Contains(h, "unsaved") || !strings.Contains(h, "open now") {
		t.Errorf("offen und gesichert: %q", h)
	}
	if h := leisteHinweis(e, false, false); strings.Contains(h, "unsaved") || strings.Contains(h, "open now") {
		t.Errorf("nicht offen: %q", h)
	}
}
