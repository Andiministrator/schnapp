// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests zum Rahmen nach ADR-0031 (Karte #187): Menüzeile, Werkzeugleiste, Fußzeile.

import (
	"image"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/widget"

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
	if m.welches != menueAnzahl-1 {
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

// TestWerkzeugleisteKlick: Ein Klick auf den Knopf in der Leiste links wählt das Werkzeug (Lage aus werkzeugRahmen). Ein
// Klick auf das Farbfeld klappt die eingeklappte rechte Leiste auf; zeigt sie die Farbe schon, öffnet er die Farbliste
// am Farbfeld, und ein Eintrag setzt die Farbe (ADR-0031 Nachtrag 3).
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
	l.e.seiteZu = true
	l.klick(mittenPunkt(vornVon(farbe)))
	if l.e.seiteZu || l.e.menue.offen {
		t.Fatalf("Klick auf das Farbfeld bei eingeklappter Leiste: zu %v, Liste offen %v", l.e.seiteZu, l.e.menue.offen)
	}
	l.klick(mittenPunkt(vornVon(farbe)))
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe {
		t.Fatalf("zweiter Klick auf das Farbfeld: Liste offen %v (%d)", l.e.menue.offen, l.e.menue.welches)
	}
	if r := l.e.menue.rahmen[0]; r.Min.Y < farbe.Max.Y || r.Min.X != farbe.Min.X {
		t.Errorf("Farbliste hängt nicht unter dem Farbfeld: %v, Farbfeld %v", r, farbe)
	}
	l.e.s.Auswahl = 0
	l.klick(l.e.menue.rahmen[2].Min.Add(l.e.menue.rahmen[2].Max).Div(2))
	if l.e.menue.offen || l.e.s.Stil.Farbe != canvas.Beschriftungsfarben[2] {
		t.Errorf("Eintrag 3 der Farbliste: offen %v, Farbe %v", l.e.menue.offen, l.e.s.Stil.Farbe)
	}
}

// TestFusszeileFelder: Der Zeigerort folgt der Maus über dem Bild und verschwindet beim Verlassen; ein Klick auf das
// Zoomfeld oder das Farbfeld klappt eine Liste über dem Feld auf, ein Eintrag setzt Zoom bzw. Farbe (ADR-0031
// Nachtrag 3).
func TestFusszeileFelder(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	f := l.e.leinwandFlaeche(l.gtx())
	ort := f.Min.Add(image.Pt(int(l.e.s.Abb.Versatz.X), int(l.e.s.Abb.Versatz.Y))).Add(image.Pt(3, 2))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(ort.X), float32(ort.Y))})
	l.setle()
	if l.e.zeigerAnsicht() == nil {
		t.Fatal("kein Zeigerort über dem Bild")
	}
	gefunden := false
	for _, x := range l.e.s.FussFelder(l.e.zeigerAnsicht()) {
		if x.Art == canvas.FeldZeiger {
			gefunden = true
		}
	}
	if !gefunden {
		t.Errorf("Feld Zeiger fehlt (Ort %v)", *l.e.zeigerAnsicht())
	}
	l.r.Queue(pointer.Event{Kind: pointer.Leave, Source: pointer.Mouse, Position: f32.Pt(float32(ort.X), float32(ort.Y))})
	l.setle()
	if l.e.zeigerAnsicht() != nil {
		t.Error("Zeigerort bleibt nach dem Verlassen stehen")
	}
	l.e.s.Scrollen(document.Pt(10, 10), -3) // hineinzoomen: nicht mehr eingepasst, nicht 1:1
	l.setle()
	if l.e.s.Abb.Zoom == 1 {
		t.Fatal("Vorbedingung: Zoom ist 1")
	}
	zoomFeld := l.e.fussFeldRahmen[canvas.FeldZoom]
	l.klick(zoomFeld.Min.Add(zoomFeld.Max).Div(2))
	if !l.e.menue.offen || l.e.menue.welches != listeZoom {
		t.Fatalf("Klick auf das Zoomfeld: Liste offen %v (%d)", l.e.menue.offen, l.e.menue.welches)
	}
	en := l.e.menueEintraege()
	if len(en) != 1+len(canvas.ZoomStufen) || en[0].zoom != 0 || en[0].aktiv {
		t.Fatalf("Zoomliste: %+v", en)
	}
	if r := l.e.menue.rahmen[len(en)-1]; r.Max.Y > zoomFeld.Min.Y {
		t.Errorf("Zoomliste liegt nicht über dem Feld: letzte Zeile %v, Feld %v", r, zoomFeld)
	}
	for i, x := range en {
		if x.zoom == 200 {
			l.klick(l.e.menue.rahmen[i].Min.Add(l.e.menue.rahmen[i].Max).Div(2))
		}
	}
	if l.e.menue.offen || l.e.s.Abb.Zoom != 2 {
		t.Errorf("Eintrag 200 %%: offen %v, Zoom %v", l.e.menue.offen, l.e.s.Abb.Zoom)
	}
	l.klick(zoomFeld.Min.Add(zoomFeld.Max).Div(2))
	l.klick(l.e.menue.rahmen[0].Min.Add(l.e.menue.rahmen[0].Max).Div(2))
	if !l.e.s.Eingepasst() {
		t.Error("Eintrag Einpassen passt nicht ein")
	}
	l.e.s.Auswahl = 0
	farbFeld := l.e.fussFeldRahmen[canvas.FeldFarbe]
	l.klick(farbFeld.Min.Add(farbFeld.Max).Div(2))
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe {
		t.Fatalf("Klick auf das Farbfeld der Fußzeile: Liste offen %v (%d)", l.e.menue.offen, l.e.menue.welches)
	}
	l.klick(l.e.menue.rahmen[1].Min.Add(l.e.menue.rahmen[1].Max).Div(2))
	if l.e.s.Stil.Farbe != canvas.Beschriftungsfarben[1] {
		t.Errorf("Farbliste der Fußzeile, Eintrag 2: Farbe %v", l.e.s.Stil.Farbe)
	}
	if text := fussText(canvas.FussFeld{Art: canvas.FeldStand, Ungesichert: true}); text != "unsaved" {
		t.Errorf("Stand: %q", text)
	}
}

// TestHilfeTastenkuerzel: F1 und der Eintrag im Menü Hilfe öffnen die Übersicht; sie nennt jedes Kürzel der Tabelle
// mit Handlung oder Werkzeug; Esc und ein Klick schließen sie (ADR-0031 Punkt 1).
func TestHilfeTastenkuerzel(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(key.NameF1, 0)
	if !l.e.hilfeOffen {
		t.Fatal("F1 öffnet die Übersicht nicht")
	}
	tasten := map[string]bool{}
	for _, z := range l.e.hilfeZeilen() {
		for _, k := range strings.Split(z.taste, ", ") {
			tasten[k] = true
		}
	}
	for _, ta := range tastenTabelle() {
		if ta.ausnahme != "" {
			continue
		}
		if n := tastenName(ta.filter); !tasten[n] {
			t.Errorf("Kürzel %q fehlt in der Übersicht", n)
		}
	}
	l.taste(key.NameEscape, 0)
	if l.e.hilfeOffen {
		t.Error("Esc schließt die Übersicht nicht")
	}
	l.e.menue.welches = menueHilfe
	l.titel().Click()
	l.setle()
	l.e.menue.eintraege[l.index(aktTastenkuerzel)].Click()
	l.setle()
	if !l.e.hilfeOffen {
		t.Fatal("Menüeintrag öffnet die Übersicht nicht")
	}
	l.klick(l.e.leinwandFlaeche(l.gtx()).Min.Add(image.Pt(5, 5)))
	if l.e.hilfeOffen {
		t.Error("Klick schließt die Übersicht nicht")
	}
	Version = "9.9.9"
	l.e.fuehreAus(l.gtx(), aktUeber, true)
	if !strings.Contains(l.e.meldung.Text, "9.9.9") {
		t.Errorf("Über Schnapp: %q", l.e.meldung.Text)
	}
}

// TestHilfeSperrtTasten: Bei offener Übersicht wirken Entf und Werkzeugbuchstaben nicht; Esc schließt sie auch während
// des Speicherns (Sitzungs-QA B-2).
func TestHilfeSperrtTasten(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(key.NameF1, 0)
	wz := l.e.s.Werkzeug
	l.taste(key.NameDeleteForward, 0)
	l.taste("X", 0)
	if _, ok := l.e.s.D.Objekt(l.id); !ok || l.e.s.Werkzeug != wz {
		t.Errorf("Tasten wirken hinter der Übersicht: Objekt da %v, Werkzeug %v", ok, l.e.s.Werkzeug)
	}
	if !l.e.hilfeOffen {
		t.Fatal("Übersicht durch Tasten geschlossen")
	}
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	defer close(v.frei)
	l.e.speichernJetzt() // Strg+S wirkt hinter der Übersicht nicht; das Speichern kommt hier von außen
	l.setle()
	if !l.e.s.SpeichernLaeuft() {
		t.Fatal("Vorbedingung: Speichern läuft nicht")
	}
	l.taste(key.NameEscape, 0)
	if l.e.hilfeOffen {
		t.Error("Esc während des Speicherns schließt die Übersicht nicht")
	}
}

// TestUeberfahrenWechseltPerTasteOffenesMenueNicht (F-514, Mutant U3): Nur ein per Maus geöffnetes Menü wechselt beim
// Überfahren eines anderen Titels; ein per Taste (F10) geöffnetes bleibt, der Zeiger ruht meist irgendwo darüber.
func TestUeberfahrenWechseltPerTasteOffenesMenueNicht(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.taste(key.NameF10, 0)
	if !l.e.menue.offen || l.e.menue.welches != menueDatei || !l.e.menue.perTasteOffen {
		t.Fatalf("Vorbedingung: offen %v, Menü %d", l.e.menue.offen, l.e.menue.welches)
	}
	r := l.e.menue.titelRahmen[menueBild]
	p := r.Min.Add(r.Max).Div(2)
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
	l.setle()
	if !l.e.menue.offen || l.e.menue.welches != menueDatei {
		t.Errorf("Überfahren von „Bild“ bei per Taste offenem Menü: offen %v, Menü %d, soll Datei bleiben",
			l.e.menue.offen, l.e.menue.welches)
	}
}

// TestHinweisAmWerkzeugknopfNenntKuerzel (F-514, Mutant U9): Der Hinweis führt Name und Kürzel jedes Werkzeugs.
func TestHinweisAmWerkzeugknopfNenntKuerzel(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	for _, w := range l.e.werkzeugKnoepfe() {
		k := werkzeugKuerzel(w.w)
		if k == "" {
			t.Errorf("%s: kein Kürzel", w.text)
		}
		if h := werkzeugHinweis(w); h != w.text+" ("+k+")" {
			t.Errorf("Hinweis %q, erwartet %q", h, w.text+" ("+k+")")
		}
	}
}

// TestFusszeileLaesstFelderWeg (F-514, Mutant U2): Ein schmales Fenster zeigt weniger Felder; keines liegt im linken
// Drittel, das der Meldung gehört. Ein breites Fenster zeigt alle.
func TestFusszeileLaesstFelderWeg(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	alle := len(l.e.s.FussFelder(l.e.zeigerAnsicht()))
	if got := len(l.e.fussRahmen); got != alle {
		t.Fatalf("breites Fenster: %d von %d Feldern", got, alle)
	}
	l.fenst = image.Pt(360, 400)
	l.setle()
	if got := len(l.e.fussRahmen); got == 0 || got >= alle {
		t.Errorf("schmales Fenster: %d von %d Feldern, erwartet mindestens eins weniger", got, alle)
	}
	for _, r := range l.e.fussRahmen {
		if r.Min.X < 360/3 {
			t.Errorf("Feld %v ragt ins linke Drittel (Meldung)", r)
		}
	}
}

// TestKlickInDieMenuezeileSchliesstListe (F-515): Ein Klick in die Menüzeile abseits der Titel (hier rechts neben dem
// letzten Titel) schließt eine offene Liste wie ein Klick daneben; ein Klick auf einen Titel tut weiter, was er tat.
func TestKlickInDieMenuezeileSchliesstListe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.klick(menueKnopfMitte(l))
	if !l.e.menue.offen {
		t.Fatal("Vorbedingung: Menü offen")
	}
	letzter := l.e.menue.titelRahmen[menueAnzahl-1]
	l.klick(image.Pt(letzter.Max.X+40, letzter.Min.Y+5))
	if l.e.menue.offen {
		t.Error("Klick in die leere Menüzeile lässt die Liste offen")
	}
	l.klick(menueKnopfMitte(l)) // ein Titel öffnet weiter
	if !l.e.menue.offen {
		t.Error("Klick auf den Titel öffnet nicht mehr")
	}
}

// TestFarbHinweisNichtUeberOffenerListe (F-515): Wie die Hinweise der Werkzeuge verschwindet der Hinweis am Farbfeld,
// solange eine Liste offen ist.
func TestFarbHinweisNichtUeberOffenerListe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(&l.e.knoepfe.farbe)
	if !l.e.farbHinweisZeigen(l.gtx()) {
		t.Fatal("Vorbedingung: Hinweis bei Tastaturfokus")
	}
	l.e.menue.offen = true
	if l.e.farbHinweisZeigen(l.gtx()) {
		t.Error("Hinweis am Farbfeld steht über der offenen Liste")
	}
}

// TestTabFolgtDerReihenfolgeDerFussfelder (F-515): Vom Farbfeld links springt Tab durch die Farben der rechten Leiste
// (ADR-0031 Nachtrag 3), Pipette, Hex und Deckkraft (Nachtrag 5) zu den klickbaren Feldern der Fußzeile von links nach
// rechts: erst Farbe, dann Zoom.
func TestTabFolgtDerReihenfolgeDerFussfelder(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 960)) // hoch genug für Schnellstile (ADR-0037), Farbe und Ebenen zugleich
	l.fokusiere(&l.e.knoepfe.farbe)
	// Farbpaar der Werkzeugleiste: Vordergrund, Füllung, Tauschen (ADR-0021 Nachtrag 11).
	for i, k := range []*widget.Clickable{&l.e.knoepfe.fuellung, &l.e.knoepfe.tausch} {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(k) {
			t.Fatalf("Tab %d im Farbpaar: Fokus auf %s", i+1, menueFokusName(l))
		}
	}
	// Zuerst der Abschnitt Schnellstile (ADR-0037): Überschrift, die Kacheln, „+“.
	schnell := []*widget.Clickable{&l.e.schnell.kopf}
	for j := range l.e.schnell.liste {
		schnell = append(schnell, &l.e.schnell.kacheln[j])
	}
	schnell = append(schnell, &l.e.schnell.plus)
	if len(l.e.schnell.liste) < 6 {
		t.Fatalf("Kontrolle: %d Kacheln der Schnellstile für den Text", len(l.e.schnell.liste))
	}
	for i, k := range schnell {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(k) {
			t.Fatalf("Tab %d im Abschnitt Schnellstile: Fokus woanders (%s)", i+1, menueFokusName(l))
		}
	}
	// Gewählt ist ein Text: erst Schrift, Größe, Stil und Ausrichtung (ADR-0031 Nachtrag 6).
	for i, k := range l.e.seite.sw.knoepfe() {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(k) {
			t.Fatalf("Tab %d nach dem Farbfeld: Fokus nicht auf Knopf %d der Schrift (%s)", i+1, i, menueFokusName(l))
		}
	}
	// Der Schalter „Spitze“ steht unter der Schrift (ADR-0021 Nachtrag 13 Punkt 6).
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&l.e.spitze.knopf) {
		t.Fatalf("Tab nach der Schrift: Fokus nicht auf dem Schalter Spitze (%s)", menueFokusName(l))
	}
	// Die Überschrift „Farbe“ klappt den Abschnitt ein und aus (ADR-0032 Punkt 6).
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&l.e.seite.farbeKopf) {
		t.Fatalf("Tab nach der Schrift: Fokus nicht auf der Überschrift Farbe (%s)", menueFokusName(l))
	}
	// Die Zeile des Farbpaars: Vordergrund, Füllung, Tauschen, Keine (ADR-0021 Nachtrag 11).
	for i := range l.e.seite.paar {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(&l.e.seite.paar[i]) {
			t.Fatalf("Tab %d im Farbpaar der Leiste: Fokus woanders (%s)", i+1, menueFokusName(l))
		}
	}
	for i := range canvas.Beschriftungsfarben {
		l.r.MoveFocus(key.FocusForward) // wie die Tab-Taste (Gio)
		l.setle()
		if !l.fokus(&l.e.seite.farben[i]) {
			t.Fatalf("Tab %d nach dem Farbfeld: Fokus nicht auf Farbe %d der rechten Leiste (%s)", i+1, i, menueFokusName(l))
		}
	}
	// Danach Pipette, Hex und Deckkraft (ADR-0031 Nachtrag 5).
	for _, z := range []struct {
		tag  any
		name string
	}{{&l.e.seite.fw.pipette, "Pipette"}, {&l.e.seite.fw.hex, "Hex"}, {&l.e.seite.fw.deckkraft, "Deckkraft"}} {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(z.tag) {
			t.Fatalf("Tab nach den Farben: Fokus nicht auf %s (%s)", z.name, menueFokusName(l))
		}
	}
	// Dann der Abschnitt Ebenen: Überschrift, Deckkraft, Knöpfe ↑ ↓ Doppeln Löschen (ADR-0032 Punkt 6).
	ebenen := []any{&l.e.ebenen.kopf, &l.e.ebenen.deck}
	for j := range l.e.ebenen.knoepfe {
		ebenen = append(ebenen, &l.e.ebenen.knoepfe[j])
	}
	for i, tag := range ebenen {
		l.r.MoveFocus(key.FocusForward)
		l.setle()
		if !l.fokus(tag) {
			t.Fatalf("Tab %d im Abschnitt Ebenen: Fokus woanders (%s)", i+1, menueFokusName(l))
		}
	}
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&l.e.knoepfe.fussFarbe) {
		t.Fatalf("erstes Tab nach dem Farbwähler: Fokus nicht auf dem Fußfeld Farbe (%s)", menueFokusName(l))
	}
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&l.e.knoepfe.fussZoom) {
		t.Errorf("zweites Tab: Fokus nicht auf dem Fußfeld Zoom (%s)", menueFokusName(l))
	}
}
