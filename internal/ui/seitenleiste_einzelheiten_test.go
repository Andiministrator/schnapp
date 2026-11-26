// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests aus der Sitzungs-QA vom 23.11.2026 (Kritiker A, A-1 bis A-10) zur rechten Leiste und den Fußzeilen-Listen.

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func mittenPunkt(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }

// vornVon ist das Feld Vordergrund im Farbpaar r der Werkzeugleiste (ADR-0021 Nachtrag 11).
func vornVon(r image.Rectangle) image.Rectangle { v, _, _ := farbpaarRahmen(r); return v }

// A-1: Nach der Farbliste am Farbfeld links hängt die Zoomliste trotzdem am Zoomfeld, per Maus und per Tastatur.
func TestZoomListeNachFarbKnopf(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	// Per Tastatur öffnet das Farbfeld links immer die Liste an sich (die Maus zeigt seit F-817 den Farbwähler).
	l.fokusiere(&l.e.knoepfe.farbe)
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || !l.e.listeAmFarbKnopf {
		t.Fatalf("Vorbedingung: Farbliste am Farbfeld offen %v, Merker %v", l.e.menue.offen, l.e.listeAmFarbKnopf)
	}
	l.taste(key.NameEscape, 0)
	zoom := l.e.fussFeldRahmen[canvas.FeldZoom]
	l.klick(mittenPunkt(zoom))
	n := len(l.e.menueEintraege())
	if !l.e.menue.offen || l.e.menue.welches != listeZoom || l.e.menue.rahmen[n-1].Max.Y > zoom.Min.Y ||
		l.e.menue.rahmen[0].Min.X != zoom.Min.X && l.e.menue.rahmen[0].Max.X < zoom.Max.X {
		t.Errorf("Zoomliste per Maus: offen %v, erste Zeile %v, letzte %v, Feld %v", l.e.menue.offen,
			l.e.menue.rahmen[0], l.e.menue.rahmen[n-1], zoom)
	}
	l.taste(key.NameEscape, 0)
	l.e.listeAmFarbKnopf = true // wie nach einer Farbliste am Farbfeld
	l.fokusiere(&l.e.knoepfe.fussZoom)
	l.taste(key.NameReturn, 0)
	if l.e.listenAnker(listeZoom) != zoom {
		t.Errorf("Zoomliste per Tastatur: Anker %v, Feld %v", l.e.listenAnker(listeZoom), zoom)
	}
}

// A-2, A-3: Strichstärke, Kantenform und Zoomstufe wirken nicht während des Speicherns bzw. eines Zugs, und nicht über
// die Obergrenze; die Klicks in der Leiste werden während des Speicherns verworfen.
func TestStrichSperren(t *testing.T) {
	s := testEditor(t).s
	s.SetzeFenster(image.Pt(400, 300))
	if s.SetzeBreite(document.MaxGroesse+1) || s.Stil.Breite == document.MaxGroesse+1 {
		t.Error("Breite über MaxGroesse angenommen")
	}
	s.Druecken(document.Pt(200, 150), canvas.TastePrimaer)
	s.Ziehen(document.Pt(220, 160))
	if !s.Zieht() {
		t.Fatal("Vorbedingung: kein Zug")
	}
	zoom := s.Abb.Zoom
	if s.SetzeBreite(12) || s.Stil.Breite == 12 {
		t.Error("Breite während eines Zugs geändert")
	}
	if s.SetzeZoom(200); s.Abb.Zoom != zoom {
		t.Error("Zoomstufe während eines Zugs gesetzt")
	}
	s.Loslassen(document.Pt(220, 160))

	l := neuerMenueLauf(t, image.Pt(1200, 800))
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	l.e.s.Auswahl = 0
	l.taste("A", 0)
	plus := l.e.seite.strRahmen[1]
	l.e.speichernJetzt()
	<-v.beginnt
	if !l.e.s.SpeichernLaeuft() {
		t.Fatal("Vorbedingung: Speichern läuft nicht")
	}
	l.klick(mittenPunkt(plus))
	if l.e.s.Stil.Breite != 6 {
		t.Errorf("+ während des Speicherns: %v", l.e.s.Stil.Breite)
	}
	close(v.frei)
}

// A-3: Schwelle der rechten Leiste genau bei 752 dp, Haken im Menü, grüne Wahl und Kürzel in den Listen, Stufen nach
// ADR, Hinweistexte, angezeigte Breite des gewählten Objekts.
func TestLeisteUndListenEinzelheiten(t *testing.T) {
	for _, b := range []int{werkzeugeDp + seiteDp + minLeinwandDp - 1, werkzeugeDp + seiteDp + minLeinwandDp} {
		l := neuerMenueLauf(t, image.Pt(b, 600))
		if sichtbar := l.e.seite.kopf != ""; sichtbar != (b >= 752) {
			t.Errorf("Breite %d: Leiste sichtbar %v", b, sichtbar)
		}
	}
	if fmt.Sprint(canvas.StrichStufen) != "[1 2 3 4 6 8 12 16 24 32]" || fmt.Sprint(canvas.ZoomStufen) != "[25 50 100 200 400]" {
		t.Errorf("Stufen %v / %v weichen von ADR-0021 Nachtrag 7 / ADR-0031 Nachtrag 3 ab", canvas.StrichStufen, canvas.ZoomStufen)
	}
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	haken := func() bool {
		for _, en := range l.e.menueEintraegeVon(menueAnsicht) {
			if en.akt == aktSeitenleiste {
				return en.aktiv
			}
		}
		t.Fatal("Eintrag Seitenleiste fehlt")
		return false
	}
	if !haken() {
		t.Error("Haken fehlt bei offener Leiste")
	}
	l.e.seiteZu = true
	if haken() {
		t.Error("Haken bei eingeklappter Leiste")
	}
	l.e.seiteZu = false
	l.e.s.SetzeFarbe(canvas.Beschriftungsfarben[2])
	for i, en := range l.e.menueEintraegeVon(listeFarbe) {
		if en.aktiv != (i == 2) || menueKuerzel(en) != fmt.Sprintf("#%02X%02X%02X", en.farbe.R, en.farbe.G, en.farbe.B) {
			t.Errorf("Farbliste Eintrag %d: aktiv %v, Kürzel %q", i, en.aktiv, menueKuerzel(en))
		}
	}
	l.e.s.SetzeZoom(100) // auf einer Stufe, dann eingepasst: grün ist nur „Einpassen“, auch wenn der Zoom 100 % ist
	l.e.s.Einpassen()
	if !l.e.s.Eingepasst() {
		t.Fatal("Vorbedingung: nicht eingepasst")
	}
	for _, en := range l.e.menueEintraegeVon(listeZoom) {
		if en.aktiv != (en.zoom == 0) {
			t.Errorf("eingepasst bei Zoom %.2f: Eintrag %d aktiv %v", l.e.s.Abb.Zoom, en.zoom, en.aktiv)
		}
	}
	l.e.s.Abb.Zoom = 1 // genau eine Stufe, die Ansicht bleibt eingepasst gemeldet
	for _, en := range l.e.menueEintraegeVon(listeZoom) {
		if en.zoom == 100 && en.aktiv {
			t.Error("eingepasst und 100 %: Eintrag 100 % ist grün")
		}
	}
	l.e.s.SetzeZoom(200)
	for _, en := range l.e.menueEintraegeVon(listeZoom) {
		if en.aktiv != (en.zoom == 200) {
			t.Errorf("Zoomliste %d: aktiv %v", en.zoom, en.aktiv)
		}
		soll := map[int]string{0: kuerzelText(aktEinpassen), 100: kuerzelText(aktEins)}[en.zoom]
		if menueKuerzel(en) != soll {
			t.Errorf("Zoomliste %d: Kürzel %q, Soll %q", en.zoom, menueKuerzel(en), soll)
		}
	}
	gesehen := map[string]bool{}
	for _, h := range []canvas.SeitenHinweis{canvas.SeitenHinweisNichts, canvas.SeitenHinweisEndgueltig,
		canvas.SeitenHinweisOriginal, canvas.SeitenHinweisKeineNaht} {
		if txt := seitenHinweisText(h); txt == "" || gesehen[txt] {
			t.Errorf("Hinweis %d: %q leer oder doppelt", h, txt)
		} else {
			gesehen[txt] = true
		}
	}
	if seitenHinweisText(canvas.SeitenHinweisEndgueltig) != "The pixels are overwritten for good once saved." {
		t.Error("Schwärzen-Hinweis vertauscht")
	}
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(5, 5), document.Pt(40, 30),
		document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 7}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	if l.e.seite.strichText != "7 px" {
		t.Errorf("angezeigte Breite %q, soll die des gewählten Pfeils (7 px)", l.e.seite.strichText)
	}
	if len(canvas.Beschriftungsfarben) > len(l.e.seite.farben) { // A-10
		t.Errorf("%d Beschriftungsfarben, aber nur %d Knöpfe", len(canvas.Beschriftungsfarben), len(l.e.seite.farben))
	}
}

// A-3 (U4a, U4b): Das Farbfeld links klappt die Leiste nur auf, wenn sie die Farbe zeigen kann; sonst öffnet es die
// Farbliste.
func TestFarbKnopfOhneFarbeOderSchmal(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.taste("B", 0) // Schwärzen: keine Farbe in der Leiste
	l.e.seiteZu = true
	l.setle()
	_, farbe := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	l.klick(mittenPunkt(vornVon(farbe)))
	if !l.e.seiteZu || !l.e.menue.offen || l.e.menue.welches != listeFarbe {
		t.Errorf("Werkzeug ohne Farbe: zu %v, Liste offen %v", l.e.seiteZu, l.e.menue.offen)
	}
	l = neuerMenueLauf(t, image.Pt(700, 600))
	l.e.seiteZu = true
	_, farbe = werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	l.klick(mittenPunkt(vornVon(farbe)))
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe {
		t.Errorf("schmales Fenster: Liste offen %v", l.e.menue.offen)
	}
}

// A-5, A-6: Tastaturfokus bleibt am grauen Stufenknopf und geht beim Einklappen an die Leinwand; die Leertaste an einem
// Feld mit Liste öffnet sie wie Enter (Fokus im ersten Eintrag, Esc zurück ans Feld).
func TestLeisteTastaturfokus(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.s.Auswahl = 0
	l.taste("A", 0)
	l.e.s.SetzeBreite(24)
	l.fokusiere(&l.e.seite.strich[1])
	l.taste(key.NameReturn, 0)
	if l.e.s.Stil.Breite != 32 || !l.fok(&l.e.seite.strich[1]) {
		t.Errorf("Plus auf die letzte Stufe: Breite %v, Fokus am Knopf %v", l.e.s.Stil.Breite, l.fok(&l.e.seite.strich[1]))
	}
	l.taste("B", key.ModShortcut|key.ModShift)
	if !l.e.seiteZu || !l.fok(l.e) {
		t.Errorf("Einklappen mit Fokus in der Leiste: zu %v, Fokus Leinwand %v", l.e.seiteZu, l.fok(l.e))
	}
	l.fokusiere(&l.e.knoepfe.fussZoom)
	l.taste(key.NameSpace, 0)
	if !l.e.menue.offen || l.e.menue.welches != listeZoom || !l.e.menue.perTasteOffen || menueFokusNr(l) != 0 {
		t.Errorf("Leertaste am Zoomfeld: offen %v, per Taste %v, Fokus %d", l.e.menue.offen, l.e.menue.perTasteOffen, menueFokusNr(l))
	}
	l.taste(key.NameEscape, 0)
	if !l.fok(&l.e.knoepfe.fussZoom) {
		t.Errorf("Esc nach Leertaste: Fokus %s", menueFokusName(l))
	}
	// Ebenso am Fußfeld Farbe und am Farbfeld links (Sitzungs-QA N-8); dort hängt die Liste am Farbfeld.
	for _, f := range []struct {
		c      any
		amLink bool
	}{{&l.e.knoepfe.fussFarbe, false}, {&l.e.knoepfe.farbe, true}} {
		l.fokusiere(f.c)
		l.taste(key.NameSpace, 0)
		if !l.e.menue.offen || l.e.menue.welches != listeFarbe || !l.e.menue.perTasteOffen || menueFokusNr(l) != 0 ||
			l.e.listeAmFarbKnopf != f.amLink {
			t.Errorf("Leertaste an %T (Farbfeld links %v): offen %v, per Taste %v, Fokus %d, am Farbfeld %v", f.c, f.amLink,
				l.e.menue.offen, l.e.menue.perTasteOffen, menueFokusNr(l), l.e.listeAmFarbKnopf)
		}
		l.taste(key.NameEscape, 0)
		if !l.fok(f.c) {
			t.Errorf("Esc: Fokus %s", menueFokusName(l))
		}
	}
}

// A-8: Bei offener Übersicht der Tastenkürzel ändert ein Klick in die rechte Leiste nichts.
func TestHilfeOffenSperrtLeiste(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	vor := l.e.s.D.Stand()
	l.e.hilfeOffen = true
	l.setle()
	l.klick(mittenPunkt(l.e.seite.ff.rahmen[1]))
	if l.e.s.D.Stand() != vor || l.e.s.Stil.Farbe == canvas.Beschriftungsfarben[1] {
		t.Error("Klick in die Leiste hinter der Hilfetafel hat etwas geändert")
	}
}

// fok meldet, ob tag den Tastaturfokus hat.
func (l *menueLauf) fok(tag any) bool { return l.fokus(tag) }
