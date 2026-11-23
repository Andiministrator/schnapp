// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests zum Rahmen nach ADR-0031 (Karte #187): Menüzeile, Werkzeugleiste, Fußzeile.

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// TestF10UndPfeileWechselnMenues: F10 öffnet „Datei“ mit Fokus auf dem ersten Eintrag, Pfeil rechts/links wechselt das
// Menü (mit Umlauf) und behält den Tastaturfokus in der Liste, F10 schließt wieder.
func TestF10UndPfeileWechselnMenues(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(key.NameF10, 0)
	m := &l.e.menue
	if !m.offen || m.welches != menueDatei || menueFokusNr(l) != 0 {
		t.Fatalf("F10: offen %v, Menü %d, Fokus %s", m.offen, m.welches, menueFokusName(l))
	}
	l.taste(key.NameRightArrow, 0)
	if !m.offen || m.welches != menueBearbeiten || menueFokusNr(l) != 0 {
		t.Errorf("rechts: offen %v, Menü %d, Fokus %s", m.offen, m.welches, menueFokusName(l))
	}
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameLeftArrow, 0)
	if m.welches != menueWerkzeuge {
		t.Errorf("links vom ersten: Menü %d, soll das letzte (Umlauf)", m.welches)
	}
	l.taste(key.NameF10, 0)
	if m.offen || !l.fokus(l.e) {
		t.Errorf("F10 bei offenem Menü: offen %v, Fokus %s", m.offen, menueFokusName(l))
	}
}

// TestMenueWerkzeuge: Das Menü „Werkzeuge“ führt alle neun Werkzeuge mit Buchstaben, markiert das aktive und wählt per
// Eintrag.
func TestMenueWerkzeuge(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	en := l.e.menueEintraegeVon(menueWerkzeuge)
	if len(en) != len(l.e.werkzeugKnoepfe()) {
		t.Fatalf("%d Einträge, %d Werkzeuge", len(en), len(l.e.werkzeugKnoepfe()))
	}
	aktiv := 0
	for _, x := range en {
		if !x.hatWerkzeug || menueKuerzel(x) == "" {
			t.Errorf("%s: kein Werkzeugeintrag mit Kürzel", x.text)
		}
		if x.aktiv {
			aktiv++
		}
	}
	if aktiv != 1 {
		t.Errorf("%d aktive Einträge", aktiv)
	}
	l.e.menue.welches = menueWerkzeuge
	l.titel().Click()
	l.setle()
	for i, x := range l.e.menueEintraege() {
		if x.werkzeug == canvas.WerkzeugStauchen {
			l.e.menue.eintraege[i].Click()
		}
	}
	l.setle()
	if l.e.s.Werkzeug != canvas.WerkzeugStauchen || l.e.menue.offen {
		t.Errorf("Eintrag Stauchen: Werkzeug %v, offen %v", l.e.s.Werkzeug, l.e.menue.offen)
	}
}

// TestUeberfahrenWechseltOffenesMenue: Ist ein Menü per Maus offen, wechselt das Überfahren eines anderen Titels die
// Liste (wie in Qt).
func TestUeberfahrenWechseltOffenesMenue(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.klick(menueKnopfMitte(l))
	if !l.e.menue.offen || l.e.menue.welches != menueDatei {
		t.Fatal("Klick auf „Datei“ öffnet nicht")
	}
	r := l.e.menue.titelRahmen[menueBild]
	p := r.Min.Add(r.Max).Div(2)
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
	l.setle()
	if !l.e.menue.offen || l.e.menue.welches != menueBild {
		t.Errorf("Überfahren von „Bild“: offen %v, Menü %d", l.e.menue.offen, l.e.menue.welches)
	}
}

// TestWerkzeugleisteKlick: Ein Klick auf den Knopf in der Leiste links wählt das Werkzeug (Lage aus werkzeugRahmen).
func TestWerkzeugleisteKlick(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	ws := l.e.werkzeugKnoepfe()
	rahmen, farbe := werkzeugRahmen(l.gtx(), ws)
	for i, w := range ws {
		if w.w != canvas.WerkzeugText {
			continue
		}
		l.klick(rahmen[i].Min.Add(rahmen[i].Max).Div(2))
	}
	if l.e.s.Werkzeug != canvas.WerkzeugText {
		t.Errorf("Klick auf „Text“ links: Werkzeug %v", l.e.s.Werkzeug)
	}
	vorher := l.e.s.Stil.Farbe
	l.e.s.Auswahl = 0
	l.klick(farbe.Min.Add(farbe.Max).Div(2))
	if l.e.s.Stil.Farbe == vorher {
		t.Error("Klick auf das Farbfeld ändert die Farbe nicht")
	}
}

// TestFusszeileFelder: Der Zeigerort folgt der Maus über dem Bild und verschwindet beim Verlassen; ein Klick auf das
// Zoomfeld wechselt zwischen Einpassen und 1:1, einer auf das Farbfeld nimmt die nächste Farbe.
func TestFusszeileFelder(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	f := leinwandFlaeche(l.gtx())
	ort := f.Min.Add(image.Pt(int(l.e.s.Abb.Versatz.X), int(l.e.s.Abb.Versatz.Y))).Add(image.Pt(3, 2))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(ort.X), float32(ort.Y))})
	l.setle()
	if l.e.zeigerBild == nil {
		t.Fatal("kein Zeigerort über dem Bild")
	}
	gefunden := false
	for _, x := range l.e.s.FussFelder(l.e.zeigerBild) {
		if x.Art == canvas.FeldZeiger {
			gefunden = true
		}
	}
	if !gefunden {
		t.Errorf("Feld Zeiger fehlt (Ort %v)", *l.e.zeigerBild)
	}
	l.r.Queue(pointer.Event{Kind: pointer.Leave, Source: pointer.Mouse, Position: f32.Pt(float32(ort.X), float32(ort.Y))})
	l.setle()
	if l.e.zeigerBild != nil {
		t.Error("Zeigerort bleibt nach dem Verlassen stehen")
	}
	l.e.s.Scrollen(document.Pt(10, 10), -3) // hineinzoomen: nicht mehr eingepasst, nicht 1:1
	l.setle()
	if l.e.s.Abb.Zoom == 1 {
		t.Fatal("Vorbedingung: Zoom ist 1")
	}
	l.e.knoepfe.fussZoom.Click()
	l.setle()
	if l.e.s.Abb.Zoom != 1 {
		t.Errorf("Zoomfeld bei Zoom ≠ 1: Zoom %v, soll 1:1", l.e.s.Abb.Zoom)
	}
	vorher := l.e.s.Stil.Farbe
	l.e.s.Auswahl = 0
	l.e.knoepfe.fussFarbe.Click()
	l.setle()
	if l.e.s.Stil.Farbe == vorher {
		t.Error("Farbfeld der Fußzeile ändert die Farbe nicht")
	}
	if text := fussText(canvas.FussFeld{Art: canvas.FeldStand, Ungesichert: true}); text != "unsaved" {
		t.Errorf("Stand: %q", text)
	}
}
