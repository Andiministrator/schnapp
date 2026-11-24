// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Schnellstile (ADR-0037): Tabelle, Anwenden, Passt, eigene Stile.

// stilSitzung ist eine Sitzung über einem Bild der Größe b mit je einem Pfeil, Kasten, Text, Nummer und einer Lupe
// (alles in Rot und mit den Maßen von Faktor 1 mal dem Faktor des Bilds); ids liefert die Kennungen in dieser Folge.
func stilSitzung(t *testing.T, b image.Point) (s *Sitzung, ids [5]int) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, b.X, b.Y)))
	if err != nil {
		t.Fatal(err)
	}
	fk := float64(d.KantenFaktor())
	strich := document.Strich{Farbe: Beschriftungsfarben[0], Breite: 4 * fk}
	objekte := []document.Objekt{
		document.GeraderPfeil(document.Pt(100, 50), document.Pt(180, 50), strich),
		document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 50)), Strich: strich},
		document.Text{Ort: document.Pt(60, 60), Inhalt: "x", Schriftgrad: 20 * fk, Farbe: Weiss, Hintergrund: Beschriftungsfarben[0]},
		document.Nummer{Mitte: document.Pt(150, 80), Wert: 1, Radius: 14 * fk, Farbe: Beschriftungsfarben[0]},
		document.Lupe{Quelle: document.R(document.Pt(20, 20), document.Pt(40, 40)),
			Ziel: document.R(document.Pt(60, 20), document.Pt(100, 60)), Strich: strich},
	}
	for i, o := range objekte {
		if ids[i], err = d.Hinzufuegen(o); err != nil {
			t.Fatalf("Objekt %d: %v", i, err)
		}
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s = NeueSitzung(d)
	s.SetzeFenster(image.Pt(b.X+48, b.Y+48))
	return s, ids
}

func schnell(t *testing.T, a SchnellArt, kennung string) Schnellstil {
	t.Helper()
	for _, st := range EingebauteSchnellstile(a) {
		if st.Kennung == kennung {
			return st
		}
	}
	t.Fatalf("kein eingebauter Stil %q der Art %d", kennung, a)
	return Schnellstil{}
}

func indexVon(t *testing.T, s *Sitzung, a SchnellArt, kennung string) int {
	t.Helper()
	for i, st := range s.Schnellstile(a) {
		if st.Kennung == kennung {
			return i
		}
	}
	t.Fatalf("Stil %q fehlt", kennung)
	return -1
}

func TestSchnellstileTabelle(t *testing.T) {
	zahl := map[SchnellArt]int{SchnellPfeil: 6, SchnellKasten: 7, SchnellText: 6, SchnellNummer: 4, SchnellLupe: 3}
	for a := SchnellPfeil; a <= SchnellLupe; a++ {
		liste := EingebauteSchnellstile(a)
		if len(liste) != zahl[a] {
			t.Errorf("Art %d: %d eingebaute Stile, Soll %d", a, len(liste), zahl[a])
		}
		gesehen := map[string]bool{}
		for _, st := range liste {
			if st.Kennung == "" || gesehen[st.Kennung] || st.Eigen() {
				t.Errorf("Art %d: Kennung %q leer oder doppelt", a, st.Kennung)
			}
			gesehen[st.Kennung] = true
			if !st.gueltig() || st.Name() == "" {
				t.Errorf("Art %d %q: ungültig oder ohne Namen", a, st.Kennung)
			}
			if st != st.bereinigt().withKennung(st.Kennung) {
				t.Errorf("Art %d %q: Felder, die die Art nicht kennt, sind gesetzt", a, st.Kennung)
			}
		}
	}
	if len(EingebauteSchnellstile(SchnellSprechblase)) != 0 {
		t.Errorf("die Sprechblase hat keine eingebauten Stile, solange es das Werkzeug nicht gibt")
	}
	// Die Kopie schützt die Tabelle.
	k := EingebauteSchnellstile(SchnellPfeil)
	k[0].Breite = 99
	if EingebauteSchnellstile(SchnellPfeil)[0].Breite == 99 {
		t.Errorf("die Tabelle lässt sich über die Kopie ändern")
	}
}

func (st Schnellstil) withKennung(k string) Schnellstil { st.Kennung = k; return st }

// Der erste Stil jeder Art ist die Voreinstellung des Werkzeugs: ein neuer Editor zeigt ihn als passend.
func TestSchnellstilVoreinstellungPasst(t *testing.T) {
	for _, faktor := range []image.Point{{200, 100}, {2400, 2160}} {
		s, _ := stilSitzung(t, faktor)
		for _, w := range []Werkzeug{WerkzeugPfeil, WerkzeugKasten, WerkzeugText, WerkzeugNummer, WerkzeugLupe} {
			s.SetzeWerkzeug(w)
			a, ok := s.SchnellArt()
			if !ok {
				t.Fatalf("Werkzeug %v ohne Art", w)
			}
			if i := s.PassenderSchnellstil(a); i != 0 {
				t.Errorf("Bild %v, Werkzeug %v: passender Stil %d, Soll 0", faktor, w, i)
			}
		}
	}
}

func TestSchnellArt(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(200, 100))
	for _, w := range []Werkzeug{WerkzeugAuswahl, WerkzeugSchwaerzen, WerkzeugVerpixeln, WerkzeugZuschneiden, WerkzeugStauchen} {
		s.SetzeWerkzeug(w)
		if _, ok := s.SchnellArt(); ok {
			t.Errorf("Werkzeug %v hat keine Stile", w)
		}
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	for i, a := range []SchnellArt{SchnellPfeil, SchnellKasten, SchnellText, SchnellNummer, SchnellLupe} {
		s.Auswahl = ids[i]
		if got, ok := s.SchnellArt(); !ok || got != a {
			t.Errorf("gewähltes Objekt %d: Art %d %v, Soll %d", i, got, ok, a)
		}
	}
	// Das gewählte Objekt geht dem Werkzeug vor.
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.Auswahl = ids[1]
	if got, _ := s.SchnellArt(); got != SchnellKasten {
		t.Errorf("Auswahl vor Werkzeug: Art %d", got)
	}
}

// Ohne Auswahl wird der Stil zum Stand des Werkzeugs, ohne Eintrag im Verlauf.
func TestSchnellstilOhneAuswahl(t *testing.T) {
	s, _ := stilSitzung(t, image.Pt(200, 100))
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.WendeSchnellstilAn(schnell(t, SchnellKasten, "marker")) {
		t.Errorf("ohne Objekt ändert sich das Dokument nicht")
	}
	if s.Geaendert() {
		t.Errorf("Dokument als geändert gemeldet")
	}
	if s.Farbe(FeldVordergrund) != stilGelb || s.Farbe(FeldFuellung).A != 0x66 {
		t.Errorf("Farbpaar %v / %v", s.Farbe(FeldVordergrund), s.Farbe(FeldFuellung))
	}
	if i := s.PassenderSchnellstil(SchnellKasten); i != indexVon(t, s, SchnellKasten, "marker") {
		t.Errorf("passender Stil %d", i)
	}
	// Das Farbpaar gilt nur für den Kasten.
	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.Farbe(FeldVordergrund) != Beschriftungsfarben[0] {
		t.Errorf("Pfeil bekam die Farbe des Kasten-Stils: %v", s.Farbe(FeldVordergrund))
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.PassenderSchnellstil(SchnellKasten) != indexVon(t, s, SchnellKasten, "marker") {
		t.Errorf("Kasten-Stil nach Werkzeugwechsel verloren")
	}
	// Dicker Pfeil: nur die Breite.
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.WendeSchnellstilAn(schnell(t, SchnellPfeil, "red-thick"))
	if s.Stil.Breite != 8 || s.PassenderSchnellstil(SchnellPfeil) != indexVon(t, s, SchnellPfeil, "red-thick") {
		t.Errorf("Breite %v", s.Stil.Breite)
	}
}

// Größen wirken mal Faktor des Bilds; der Stil „passt“ bei jedem Faktor, ein Stil mit anderer Breite nicht.
func TestSchnellstilFaktor(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(2400, 2160))
	if s.faktor() != 2 {
		t.Fatalf("Faktor %v, Soll 2", s.faktor())
	}
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.WendeSchnellstilAn(schnell(t, SchnellPfeil, "red-thick"))
	if s.Stil.Breite != 16 {
		t.Errorf("dicker Pfeil bei Faktor 2: Breite %v, Soll 16", s.Stil.Breite)
	}
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = ids[0]
	if !s.WendeSchnellstilAn(schnell(t, SchnellPfeil, "blue")) {
		t.Fatal("blauer Pfeil ändert den gewählten Pfeil")
	}
	o, _ := s.D.Objekt(ids[0])
	if p := o.(document.Pfeil); p.Strich.Farbe != Beschriftungsfarben[1] || p.Strich.Breite != 8 {
		t.Errorf("Pfeil %+v, Soll Blau und Breite 8", p.Strich)
	}
	if s.PassenderSchnellstil(SchnellPfeil) != indexVon(t, s, SchnellPfeil, "blue") {
		t.Errorf("gewählter Pfeil passt nicht zu Blau")
	}
	st, ok := s.SchnellstilAusStand(SchnellPfeil)
	if !ok || st.Breite != 4 || st.Farbe != Beschriftungsfarben[1] || !st.Eigen() {
		t.Errorf("Stil aus dem Stand: %+v", st)
	}
	// Ein Stil mit Größen über den Grenzen wirkt nicht.
	gross := Schnellstil{Art: SchnellPfeil, Farbe: stilRot, Breite: document.MaxGroesse}
	if s.WendeSchnellstilAn(gross) {
		t.Errorf("Breite %v mal Faktor 2 liegt über der Grenze", gross.Breite)
	}
	if o2, _ := s.D.Objekt(ids[0]); o2 != o {
		t.Errorf("ein abgelehnter Stil hat das Objekt verändert")
	}
}

// Mit gewähltem Objekt: ein Rückgängig-Schritt, der Stand des Werkzeugs zieht mit.
func TestSchnellstilMitAuswahl(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(200, 100))
	for i, c := range []struct {
		art     SchnellArt
		kennung string
		werk    Werkzeug
	}{
		{SchnellPfeil, "blue", WerkzeugPfeil},
		{SchnellKasten, "marker", WerkzeugKasten},
		{SchnellText, "black-on-yellow", WerkzeugText},
		{SchnellNummer, "green", WerkzeugNummer},
		{SchnellLupe, "white", WerkzeugLupe},
	} {
		s.SetzeWerkzeug(WerkzeugAuswahl)
		s.Auswahl = ids[i]
		vorher, _ := s.D.Objekt(ids[i])
		stand := s.D.Stand()
		vorN := schritte(t, s)
		if !s.WendeSchnellstilAn(schnell(t, c.art, c.kennung)) {
			t.Errorf("%s: das Objekt hat sich nicht geändert", c.kennung)
			continue
		}
		if n := schritte(t, s) - vorN; n != 1 {
			t.Errorf("%s: %d neue Rückgängig-Schritte, Soll 1", c.kennung, n)
		}
		if !s.Rueckgaengig() {
			t.Fatal("Rückgängig")
		}
		if o, _ := s.D.Objekt(ids[i]); o != vorher || s.D.Stand() != stand {
			t.Errorf("%s: Rückgängig stellt das Objekt nicht her", c.kennung)
		}
		if !s.Wiederholen() {
			t.Fatal("Wiederholen")
		}
		if idx := s.PassenderSchnellstil(c.art); idx != indexVon(t, s, c.art, c.kennung) {
			t.Errorf("%s: passender Stil %d", c.kennung, idx)
		}
		// Stand des Werkzeugs: ohne Auswahl passt derselbe Stil.
		s.Auswahl = 0
		s.SetzeWerkzeug(c.werk)
		if idx := s.PassenderSchnellstil(c.art); idx != indexVon(t, s, c.art, c.kennung) {
			t.Errorf("%s: der Stand des Werkzeugs zog nicht mit (%d)", c.kennung, idx)
		}
	}
	// Dieselbe Anwendung noch einmal: kein neuer Schritt.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = ids[1]
	stand := s.D.Stand()
	if s.WendeSchnellstilAn(schnell(t, SchnellKasten, "marker")) || s.D.Stand() != stand {
		t.Errorf("derselbe Stil zweimal: zweiter Schritt")
	}
}

// Eigenschaften des Texts und des Kastens: alle Felder des Stils kommen am Objekt an, Ort und Inhalt bleiben.
func TestSchnellstilAlleFelder(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(200, 100))
	s.Auswahl = ids[2]
	st := Schnellstil{Art: SchnellText, Farbe: stilBlau, Fuellung: stilGelb, Schriftgrad: 32, Schrift: "Test Sans",
		Fett: true, Kursiv: true, Unterstrichen: true, Durchgestrichen: true, Ausrichtung: document.AusrichtungMitte}
	if !s.WendeSchnellstilAn(st) {
		t.Fatal("Text nicht geändert")
	}
	o, _ := s.D.Objekt(ids[2])
	tx := o.(document.Text)
	if tx.Farbe != stilBlau || tx.Hintergrund != stilGelb || tx.Schriftgrad != 32 || tx.Schrift != "Test Sans" || !tx.Fett ||
		!tx.Kursiv || !tx.Unterstrichen || !tx.Durchgestrichen || tx.Ausrichtung != document.AusrichtungMitte {
		t.Errorf("Text %+v", tx)
	}
	if tx.Inhalt != "x" || tx.Ort != document.Pt(60, 60) {
		t.Errorf("Ort oder Inhalt verändert: %+v", tx)
	}
	if got := s.AktuellerTextStil(); got.Schrift != "Test Sans" || got.Grad != 32 || !got.Fett {
		t.Errorf("Leiste zeigt %+v", got)
	}
	s.Auswahl = 0
	s.SetzeWerkzeug(WerkzeugText)
	if got := s.AktuellerTextStil(); got.Schrift != "Test Sans" || got.Grad != 32 || !got.Kursiv || !got.Unterstrichen ||
		!got.Durchgestrichen || got.Ausrichtung != document.AusrichtungMitte {
		t.Errorf("Stand des Text-Werkzeugs %+v", got)
	}
	if s.PassenderSchnellstil(SchnellText) != -1 || !s.SchnellstilPasst(st) {
		t.Errorf("der Stil ist nicht eingebaut, aber er passt")
	}
}

// Ein Stil der anderen Art ändert das gewählte Objekt nicht, aber den Stand seines Werkzeugs.
func TestSchnellstilAndereArt(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(200, 100))
	s.Auswahl = ids[0]
	vorher, _ := s.D.Objekt(ids[0])
	if s.WendeSchnellstilAn(schnell(t, SchnellNummer, "blue")) {
		t.Errorf("Dokument geändert")
	}
	if o, _ := s.D.Objekt(ids[0]); o != vorher || s.Geaendert() {
		t.Errorf("der Pfeil hat sich verändert")
	}
	s.Auswahl = 0
	s.SetzeWerkzeug(WerkzeugNummer)
	if s.PassenderSchnellstil(SchnellNummer) != indexVon(t, s, SchnellNummer, "blue") {
		t.Errorf("Stand des Nummern-Werkzeugs zog nicht mit")
	}
}

// Texteingabe: der Stil gehört zur Eingabe, ein Schritt für das ganze Anlegen.
func TestSchnellstilInTexteingabe(t *testing.T) {
	s, _ := stilSitzung(t, image.Pt(200, 100))
	n := len(s.D.Objekte())
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 20), TastePrimaer)
	s.Loslassen(f(120, 20))
	s.Tippe("hallo")
	if s.WendeSchnellstilAn(schnell(t, SchnellText, "white-on-blue")) {
		t.Errorf("in der Eingabe meldet der Stil keine Dokumentänderung")
	}
	s.Enter(false)
	tx := letztes(t, s).(document.Text)
	if tx.Inhalt != "hallo" || tx.Farbe != Weiss || tx.Hintergrund != Beschriftungsfarben[1] {
		t.Errorf("Text %+v", tx)
	}
	if !s.Rueckgaengig() || len(s.D.Objekte()) != n {
		t.Errorf("Anlegen mit Stil sind mehr als ein Schritt")
	}
}

func TestSchnellstilSperren(t *testing.T) {
	s, ids := stilSitzung(t, image.Pt(200, 100))
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.Druecken(f(20, 70), TastePrimaer)
	s.Ziehen(f(80, 70))
	if !s.Zieht() {
		t.Fatal("zieht nicht")
	}
	if s.WendeSchnellstilAn(schnell(t, SchnellPfeil, "blue")) || s.Stil.Farbe == Beschriftungsfarben[1] {
		t.Errorf("während des Ziehens wirkt kein Stil")
	}
	s.Loslassen(f(80, 70))
	// Sprechblase: kein Werkzeug, keine Wirkung.
	s.Auswahl = ids[0]
	bl := Schnellstil{Art: SchnellSprechblase, Farbe: stilRot, Schriftgrad: 20}
	if s.WendeSchnellstilAn(bl) {
		t.Errorf("Sprechblase wirkt")
	}
	// Ungültig: ohne Breite.
	if s.WendeSchnellstilAn(Schnellstil{Art: SchnellPfeil, Farbe: stilBlau}) || s.Stil.Breite == 0 {
		t.Errorf("Stil ohne Breite wirkt")
	}
}

func TestEigeneSchnellstile(t *testing.T) {
	s, _ := stilSitzung(t, image.Pt(200, 100))
	s.SetzeWerkzeug(WerkzeugKasten)
	if _, r := s.FuegeSchnellstilHinzu(SchnellKasten); r != HinzuDoppelt {
		t.Errorf("der Stand ist der eingebaute Rote Kasten: %v", r)
	}
	if s.KannSchnellstilHinzufuegen(SchnellKasten) {
		t.Errorf("„+“ wirkt bei Doppeltem")
	}
	if _, r := s.FuegeSchnellstilHinzu(SchnellSprechblase); r != HinzuNichts {
		t.Errorf("Sprechblase: %v", r)
	}
	var eigene []Schnellstil
	for i := 0; i < MaxEigeneSchnellstile; i++ {
		s.SetzeBreite(float64(10 + i))
		if !s.KannSchnellstilHinzufuegen(SchnellKasten) {
			t.Fatalf("Stil %d: „+“ wirkt nicht", i)
		}
		st, r := s.FuegeSchnellstilHinzu(SchnellKasten)
		if r != HinzuOK || !st.Eigen() || st.Breite != float64(10+i) {
			t.Fatalf("Stil %d: %+v %v", i, st, r)
		}
		eigene = append(eigene, st)
	}
	liste := s.Schnellstile(SchnellKasten)
	if len(liste) != 7+16 || !liste[7].Eigen() || liste[7].Breite != 10 || liste[22].Breite != 25 {
		t.Fatalf("Kacheln: %d, Reihenfolge falsch", len(liste))
	}
	if s.PassenderSchnellstil(SchnellKasten) != 22 {
		t.Errorf("der zuletzt angelegte Stil passt nicht")
	}
	s.SetzeBreite(99)
	if _, r := s.FuegeSchnellstilHinzu(SchnellKasten); r != HinzuVoll {
		t.Errorf("17. eigener Stil: %v", r)
	}
	// Andere Arten zählen getrennt.
	s.SetzeWerkzeug(WerkzeugNummer)
	s.SetzeFarbe(stilGelb)
	if _, r := s.FuegeSchnellstilHinzu(SchnellNummer); r != HinzuOK {
		t.Errorf("Nummer: %v", r)
	}
	// Entfernen: eingebaute nicht, eigene ja, Rest bleibt in der Reihenfolge.
	if s.EntferneSchnellstil(SchnellKasten, 0) || s.EntferneSchnellstil(SchnellKasten, -1) || s.EntferneSchnellstil(SchnellKasten, 99) {
		t.Errorf("Entfernen eines eingebauten oder unbekannten Stils")
	}
	if !s.EntferneSchnellstil(SchnellKasten, 7+3) {
		t.Fatal("eigener Stil nicht entfernt")
	}
	liste = s.Schnellstile(SchnellKasten)
	if len(liste) != 7+15 || liste[10].Breite != 14 || liste[7].Breite != 10 {
		t.Errorf("Reihenfolge nach dem Entfernen: %v", liste[10].Breite)
	}
	if got := s.Schnellstile(SchnellNummer); len(got) != 5 || !got[4].Eigen() {
		t.Errorf("Nummern-Stile verändert: %d", len(got))
	}
	if s.EigeneSchnellstile()[0].Art != SchnellKasten || len(s.EigeneSchnellstile()) != 16 {
		t.Errorf("eigene Stile für die Datei: %d", len(s.EigeneSchnellstile()))
	}
	// Eine Kopie, keine Sicht auf die Liste der Sitzung.
	k := s.EigeneSchnellstile()
	k[0].Breite = 77
	if s.EigeneSchnellstile()[0].Breite == 77 {
		t.Errorf("EigeneSchnellstile gibt die innere Liste heraus")
	}
	_ = eigene
}

func TestSetzeEigeneSchnellstile(t *testing.T) {
	s, _ := stilSitzung(t, image.Pt(200, 100))
	var ein []Schnellstil
	for i := 0; i < MaxEigeneSchnellstile+3; i++ {
		ein = append(ein, Schnellstil{Art: SchnellPfeil, Farbe: stilRot, Breite: float64(1 + i)})
	}
	ein = append(ein,
		Schnellstil{Art: SchnellSprechblase, Farbe: stilRot, Schriftgrad: 20}, // bleibt für die Datei
		Schnellstil{Art: SchnellKasten, Farbe: stilRot},                       // ohne Breite: ungültig
		Schnellstil{Art: SchnellArt(42), Farbe: stilRot, Breite: 1},           // unbekannte Art
		Schnellstil{Art: SchnellText, Farbe: stilRot, Schriftgrad: 20, Ausrichtung: 7},
		Schnellstil{Art: SchnellNummer, Kennung: "red", Farbe: stilRot, Radius: 14, Breite: 5}, // Kennung und Breite fallen weg
	)
	s.SetzeEigeneSchnellstile(ein)
	got := s.EigeneSchnellstile()
	if len(got) != MaxEigeneSchnellstile+2 {
		t.Fatalf("%d Stile übernommen, Soll %d", len(got), MaxEigeneSchnellstile+2)
	}
	if got[MaxEigeneSchnellstile].Art != SchnellSprechblase || got[len(got)-1] != (Schnellstil{Art: SchnellNummer, Farbe: stilRot, Radius: 14}) {
		t.Errorf("Sprechblase oder Nummer falsch: %+v", got[MaxEigeneSchnellstile:])
	}
	if l := s.Schnellstile(SchnellSprechblase); len(l) != 1 || !l[0].Eigen() {
		t.Errorf("Sprechblasen-Stile: %v", l)
	}
	s.SetzeEigeneSchnellstile(nil)
	if len(s.EigeneSchnellstile()) != 0 {
		t.Errorf("Liste nicht geleert")
	}
}

// Die Umwandlung zu settings.Stil ist umkehrbar und bewahrt die Art.
func TestSchnellstilEinstellungRundlauf(t *testing.T) {
	for a := SchnellPfeil; a <= SchnellLupe; a++ {
		for _, st := range append(EingebauteSchnellstile(a), Schnellstil{Art: a, Farbe: stilBlau, Fuellung: stilGelb, Breite: 3,
			Schriftgrad: 12, Radius: 9, Schrift: "Test Sans", Fett: true, Ausrichtung: document.AusrichtungRechts}) {
			e := st.Einstellung()
			text, err := settings.FormatStile([]settings.Stil{e})
			if err != nil {
				t.Fatalf("Art %d: %v", a, err)
			}
			gelesen, warn := settings.ParseStile("x", text)
			if len(warn) != 0 || len(gelesen) != 1 {
				t.Fatalf("Art %d: %v %q\n%s", a, gelesen, warn, text)
			}
			zurueck, ok := SchnellstilAus(gelesen[0])
			if !ok || zurueck != st.bereinigt() {
				t.Errorf("Art %d %q: %+v, Soll %+v", a, st.Kennung, zurueck, st.bereinigt())
			}
		}
	}
	if _, ok := SchnellstilAus(settings.Stil{Art: "x"}); ok {
		t.Errorf("unbekannte Art angenommen")
	}
}

func TestSchnellstilGleichWie(t *testing.T) {
	a := Schnellstil{Art: SchnellPfeil, Farbe: stilRot, Breite: 4, Schriftgrad: 99}
	b := Schnellstil{Art: SchnellPfeil, Kennung: "x", Farbe: stilRot, Breite: 4}
	if !a.gleichWie(b) {
		t.Errorf("Felder, die die Art nicht kennt, und die Kennung zählen nicht")
	}
	for _, c := range []Schnellstil{
		{Art: SchnellPfeil, Farbe: stilBlau, Breite: 4}, {Art: SchnellPfeil, Farbe: stilRot, Breite: 4.001},
		{Art: SchnellLupe, Farbe: stilRot, Breite: 4},
	} {
		if a.gleichWie(c) {
			t.Errorf("%+v gilt als gleich", c)
		}
	}
	// Deckkraft der Füllung zählt.
	k1 := Schnellstil{Art: SchnellKasten, Farbe: stilRot, Fuellung: color.NRGBA{A: 0x66}, Breite: 4}
	k2 := k1
	k2.Fuellung.A = 0x67
	if k1.gleichWie(k2) {
		t.Errorf("Füllungen mit anderer Deckkraft")
	}
}
