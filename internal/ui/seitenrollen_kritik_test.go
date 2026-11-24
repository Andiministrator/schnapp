// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Regressionstests aus der Sammelkritik Sprint 12 (KS-1, KS-4, KS-6, KS-9; Karte #235, F-611, F-612): Rollen der rechten
// Leiste mit Schriftliste, kleinen Radwegen und Ebenenliste.

import (
	"image"
	"testing"

	"gioui.org/io/key"
)

// niedrigerLauf ist der Lauf im niedrigen Fenster (1200×330), in dem die Leiste rollen muss.
func niedrigerLauf(t *testing.T) *menueLauf {
	t.Helper()
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 330)
	l.setle()
	return l
}

// F-611 (KS-1): Die Schriftliste wird in Fensterkoordinaten gezeichnet; das Feld liegt in Inhaltskoordinaten. Bei gerollter
// Leiste muss sie am sichtbaren Feld hängen und mit der richtigen Unterkante gerechnet werden.
func TestSchriftlisteHaengtBeiGerollterLeisteAmFeld(t *testing.T) {
	l := niedrigerLauf(t)
	se, roll := &l.e.seite, &l.e.seite.roll
	sw := &se.sw
	if !sw.gezeigt || roll.rollMax == 0 {
		t.Fatalf("Kontrolle: Schriftwahl gezeigt %v, rollMax %d", sw.gezeigt, roll.rollMax)
	}
	weit := min(sw.feldRahmen.Min.Y-8-roll.sicht.Min.Y, roll.rollMax)
	if weit < 4 {
		t.Fatalf("Kontrolle: kein Raum zum Rollen bei sichtbarem Feld (Feld %v, Sicht %v)", sw.feldRahmen, roll.sicht)
	}
	l.rolle(float32(weit))
	if roll.rollen != weit {
		t.Fatalf("Kontrolle: gerollt %d, gewollt %d", roll.rollen, weit)
	}
	l.klick(mitte(sw.feldRahmen.Sub(image.Pt(0, roll.rollen))))
	if !sw.offen || sw.rahmen.Empty() {
		t.Fatalf("Liste offen %v, Lage %v (Versatz %d)", sw.offen, sw.rahmen, roll.rollen)
	}
	sichtbar := sw.feldRahmen.Sub(image.Pt(0, roll.rollen))
	if sw.rahmen.Min.Y != sichtbar.Max.Y {
		t.Errorf("Schriftliste beginnt bei y=%d, das sichtbare Feld endet bei y=%d (Versatz %d)",
			sw.rahmen.Min.Y, sichtbar.Max.Y, roll.rollen)
	}
	// Die Höhe rechnet mit der richtigen Unterkante: Die Liste endet nie unter der Fußzeile, und sie ist so hoch wie der
	// Platz bis dahin erlaubt (höchstens 320 dp, höchstens alle Zeilen).
	unten := l.fenst.Y - l.gtx().Dp(fusszeileDp)
	soll := min(len(sw.familien)*l.gtx().Dp(26), l.gtx().Dp(320), unten-sichtbar.Max.Y)
	if sw.rahmen.Dy() != soll || sw.rahmen.Max.Y > unten {
		t.Errorf("Liste %v, Höhe %d, Soll %d, Fußzeile ab %d", sw.rahmen, sw.rahmen.Dy(), soll, unten)
	}
}

// F-611 (KS-1): Rollt das Feld ganz aus der Sicht, schließt die Liste, statt unsichtbar offen zu bleiben und den nächsten
// Klick abzufangen (sw.aussen).
func TestSchriftlisteSchliesstWennFeldAusDerSichtRollt(t *testing.T) {
	l := niedrigerLauf(t)
	roll, sw := &l.e.seite.roll, &l.e.seite.sw
	l.klick(mitte(sw.feldRahmen.Sub(image.Pt(0, roll.rollen))))
	if !sw.offen {
		t.Fatal("Kontrolle: Liste nicht offen")
	}
	// Bei offener Liste liegt eine fensterweite Fläche über der Leiste, das Rad rollt sie dann nicht; der Versatz ändert sich
	// aber auch durch Tab (Rollen zum Fokus) oder ein anderes Fenstermaß. Der Test setzt ihn deshalb unmittelbar.
	roll.rollen = roll.rollMax
	l.setle()
	if roll.rollen != roll.rollMax {
		t.Fatalf("Kontrolle: gerollt %d von %d", roll.rollen, roll.rollMax)
	}
	if sw.feldRahmen.Sub(image.Pt(0, roll.rollen)).Intersect(roll.sicht).Empty() {
		if sw.offen {
			t.Errorf("Feld %v liegt außerhalb der Sicht %v, die Liste steht aber offen (ohne gezeichnet zu sein: %v)",
				sw.feldRahmen.Sub(image.Pt(0, roll.rollen)), roll.sicht, sw.rahmen)
		}
	} else if sw.offen && sw.rahmen.Empty() {
		t.Errorf("sw.offen ohne gezeichnete Liste (Feld %v, Versatz %d)", sw.feldRahmen, roll.rollen)
	}
}

// F-611 (KS-1): Reicht der Platz unter dem Feld bis zur Fußzeile nicht für eine Zeile, wird keine Liste gezeichnet, und dann
// steht sie auch nicht offen (kein ungezeichnetes sw.offen, das den nächsten Klick abfängt).
func TestSchriftlisteSchliesstOhnePlatzUnterDemFeld(t *testing.T) {
	l := niedrigerLauf(t)
	sw := &l.e.seite.sw
	l.klick(mitte(sw.feldRahmen.Sub(image.Pt(0, l.e.seite.roll.rollen))))
	if !sw.offen || sw.rahmen.Empty() {
		t.Fatalf("Kontrolle: Liste offen %v, Lage %v", sw.offen, sw.rahmen)
	}
	l.fenst = image.Pt(1200, 140)
	l.setle()
	feld := sw.feldRahmen.Sub(image.Pt(0, l.e.seite.roll.rollen))
	unten := l.fenst.Y - l.gtx().Dp(fusszeileDp)
	if feld.Intersect(l.e.seite.roll.sicht).Empty() || unten-feld.Max.Y >= l.gtx().Dp(26) {
		t.Fatalf("Kontrolle: Feld %v, Sicht %v, Fußzeile ab %d: kein Fall ohne Platz unter dem Feld", feld, l.e.seite.roll.sicht, unten)
	}
	if sw.offen && sw.rahmen.Empty() {
		t.Errorf("Liste offen, aber nicht gezeichnet (Feld %v, Fußzeile ab %d)", feld, unten)
	}
}

// F-611 (KS-1): Die Höhe der Schriftliste rechnet mit der sichtbaren Unterkante des Felds. Beim ungerollten Feld bliebe im
// engen Fenster kein Platz für eine Zeile, beim gerollten bleiben 40 px bis zur Fußzeile.
func TestSchriftlisteHoeheMitSichtbarerUnterkante(t *testing.T) {
	l := niedrigerLauf(t)
	sw, roll := &l.e.seite.sw, &l.e.seite.roll
	versatz := 37
	l.fenst = image.Pt(1200, sw.feldRahmen.Max.Y-versatz+40+l.gtx().Dp(fusszeileDp))
	l.setle()
	if roll.rollMax < versatz {
		t.Fatalf("Kontrolle: rollMax %d, gebraucht %d", roll.rollMax, versatz)
	}
	roll.rollen = versatz
	l.setle()
	feld := sw.feldRahmen.Sub(image.Pt(0, roll.rollen))
	if !l.imSichtfeld(sw.feldRahmen) {
		t.Fatalf("Kontrolle: Feld %v nicht ganz in der Sicht %v (Versatz %d)", feld, roll.sicht, roll.rollen)
	}
	l.klick(mitte(feld))
	if !sw.offen || sw.rahmen.Dy() != 40 {
		t.Errorf("Liste offen %v, Lage %v, Höhe %d, Soll 40 (Feld im Fenster %v)", sw.offen, sw.rahmen, sw.rahmen.Dy(), feld)
	}
}

// F-612 (KS-4): Radwege unter einem Pixel je Ereignis (Touchpad) gehen nicht verloren, der Rest wird aufgehoben.
func TestKleineRadschritteRollenDieLeiste(t *testing.T) {
	l := niedrigerLauf(t)
	roll := &l.e.seite.roll
	if roll.rollMax < 40 {
		t.Fatalf("Kontrolle: rollMax %d", roll.rollMax)
	}
	for range 50 {
		l.rolle(0.6) // zusammen 30 px
	}
	if roll.rollen < 29 || roll.rollen > 30 {
		t.Errorf("50 Radereignisse zu 0,6 px (30 px): Versatz %d, Soll 29 bis 30", roll.rollen)
	}
	for range 50 {
		l.rolle(-0.6)
	}
	if roll.rollen != 0 {
		t.Errorf("50 Radereignisse zu −0,6 px zurück: Versatz %d, Soll 0", roll.rollen)
	}
}

// F-612 (KS-4): Der Rest wird am Anschlag nicht aufgespart: Wer am Ende noch nach unten wischt und dann zurück, rollt
// sofort zurück.
func TestRadRestBleibtAmAnschlagNichtHaengen(t *testing.T) {
	l := niedrigerLauf(t)
	roll := &l.e.seite.roll
	l.rolle(float32(roll.rollMax))
	for range 100 {
		l.rolle(0.6) // am Anschlag
	}
	l.rolle(-3)
	if roll.rollen != roll.rollMax-3 {
		t.Errorf("nach 60 px über das Ende hinaus und 3 px zurück: Versatz %d, Soll %d", roll.rollen, roll.rollMax-3)
	}
}

// F-612 (KS-6): Steht die Ebenenliste am Anschlag, gibt sie das Mausrad an die Leiste weiter.
func TestRadUeberEbenenlisteRolltDieLeisteWeiter(t *testing.T) {
	l := niedrigerLauf(t)
	eb, roll := &l.e.ebenen, &l.e.seite.roll
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	if !eb.gezeigt || roll.rollMax == 0 {
		t.Fatalf("Kontrolle: Ebenen gezeigt %v, rollMax %d", eb.gezeigt, roll.rollMax)
	}
	l.rolle(1 << 20) // ans Ende: dort steht die Ebenenliste im Bild
	if roll.rollen != roll.rollMax {
		t.Fatalf("Kontrolle: gerollt %d von %d", roll.rollen, roll.rollMax)
	}
	li := eb.liste.Sub(image.Pt(0, roll.rollen)).Intersect(roll.sicht)
	if li.Empty() {
		t.Fatalf("Kontrolle: Ebenenliste %v nicht in der Sicht %v (Versatz %d)", eb.liste, roll.sicht, roll.rollen)
	}
	l.radSchritte(mitte(li), 40, -15) // 600 px nach oben über der Liste
	if roll.rollen == roll.rollMax {
		t.Errorf("600 px Rad nach oben über der Ebenenliste (Inhalt %d, Höhe %d, Listenversatz %d): Leiste rollt nicht zurück (%d von %d)",
			eb.inhaltH, eb.liste.Dy(), eb.versatz, roll.rollen, roll.rollMax)
	}
}

// F-612 (KS-6): Hat die Liste noch Weg, behält sie das Rad; die Leiste bleibt stehen, bis die Liste am Anschlag ist.
func TestRadUeberEbenenlisteBlaettertErstDieListe(t *testing.T) {
	l := niedrigerLauf(t)
	eb, roll := &l.e.ebenen, &l.e.seite.roll
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	l.rolle(1 << 20)
	li := eb.liste.Sub(image.Pt(0, roll.rollen)).Intersect(roll.sicht)
	if li.Empty() || eb.inhaltH <= eb.liste.Dy() {
		t.Skipf("Liste %v hat nichts zu blättern (Inhalt %d)", eb.liste, eb.inhaltH)
	}
	vor := roll.rollen
	l.radSchritte(mitte(li), 1, 3)
	if eb.versatz != 3 || roll.rollen != vor {
		t.Errorf("3 px über der blätterbaren Liste: Listenversatz %d (Soll 3), Leiste %d (Soll %d)", eb.versatz, roll.rollen, vor)
	}
}

// F-619 (Karte #239): Radwege unter einem Pixel je Ereignis gehen in der Ebenenliste nicht verloren, der Rest wird wie
// in der Leiste aufgehoben (rollRad).
func TestKleineRadschritteBlaetternDieEbenenliste(t *testing.T) {
	l := niedrigerLauf(t)
	eb, roll := &l.e.ebenen, &l.e.seite.roll
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	l.rolle(1 << 20)
	li := eb.liste.Sub(image.Pt(0, roll.rollen)).Intersect(roll.sicht)
	if li.Empty() || eb.inhaltH-eb.liste.Dy() < 16 {
		t.Fatalf("Kontrolle: Liste %v, Inhalt %d, zu wenig zum Blättern", eb.liste, eb.inhaltH)
	}
	vor := roll.rollen
	l.radSchritte(mitte(li), 40, 0.4) // zusammen 16 px
	if eb.versatz < 15 || eb.versatz > 16 || roll.rollen != vor {
		t.Errorf("40 Radereignisse zu 0,4 px über der Liste: Listenversatz %d (Soll 15 bis 16), Leiste %d (Soll %d)", eb.versatz, roll.rollen, vor)
	}
}

// F-612 (KS-9, KM10): Die Fokuswache rechnet die Zeigerlage ins Bild der gerollten Leiste um. Ein Klick in das fokussierte,
// gerollte Feld ist ein Klick ins Feld: Fokus und halbe Eingabe bleiben.
func TestKlickInsFokussierteGerollteFeld(t *testing.T) {
	l := niedrigerLauf(t)
	fw, roll := &l.e.seite.fw, &l.e.seite.roll
	l.fokusiere(&fw.hex)
	l.tippe("12")
	if roll.rollen == 0 || fw.hex.neu || fw.hex.text != "12" {
		t.Fatalf("Kontrolle: Versatz %d, Eingabe neu=%v text=%q", roll.rollen, fw.hex.neu, fw.hex.text)
	}
	l.klick(mitte(fw.hex.rahmen.Sub(image.Pt(0, roll.rollen))))
	if !l.fokus(&fw.hex) || fw.hex.text != "12" {
		t.Errorf("Klick ins gerollte, fokussierte Feld: Fokus %s, Eingabe %q (Soll Hex-Feld, \"12\")", menueFokusName(l), fw.hex.text)
	}
}

// F-612 (KS-9, KM11): Nur ein Fokuswechsel rollt zum Fokus; das Mausrad bekämpft ein Feld mit Fokus nicht.
func TestRadRolltWegVomFokussiertenFeld(t *testing.T) {
	l := niedrigerLauf(t)
	fw, roll := &l.e.seite.fw, &l.e.seite.roll
	l.fokusiere(&fw.hex)
	if roll.rollen == 0 {
		t.Fatalf("Kontrolle: Versatz 0 trotz Fokus auf Hex (%v, Sicht %v)", fw.hex.rahmen, roll.sicht)
	}
	l.rolle(-float32(roll.rollen))
	if roll.rollen != 0 {
		t.Errorf("Rad nach oben bei Fokus im Feld: Versatz %d, Soll 0 (das Bild rollt dem Rad nicht hinterher)", roll.rollen)
	}
}

// F-612 (KS-9, KM12): Fokus weg und wieder auf dasselbe Element rollt erneut ins Bild: Die Fokuslage gilt nur für ein Bild.
func TestFokusZurueckAufDasselbeElementRolltErneut(t *testing.T) {
	l := niedrigerLauf(t)
	fw, roll := &l.e.seite.fw, &l.e.seite.roll
	l.fokusiere(&fw.hex)
	l.fokusiere(l.e)
	l.rolle(-float32(roll.rollMax))
	if roll.rollen != 0 {
		t.Fatalf("Kontrolle: Versatz %d nach dem Zurückrollen", roll.rollen)
	}
	l.fokusiere(&fw.hex)
	if !l.imSichtfeld(fw.hex.rahmen) {
		t.Errorf("Hex %v nach erneutem Fokus nicht im Bild %v (Versatz %d)", fw.hex.rahmen, roll.sicht, roll.rollen)
	}
}

// F-612 (KS-9, KM13): Auch ein Knopf mit Fokus rollt ins Bild (nicht nur die Eingabefelder).
func TestFokusAufVerdecktenKnopfRolltInsBild(t *testing.T) {
	l := niedrigerLauf(t)
	eb, roll := &l.e.ebenen, &l.e.seite.roll
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	l.rolle(-float32(roll.rollMax))
	if roll.rollen != 0 || !eb.gezeigt {
		t.Fatalf("Kontrolle: Versatz %d, Ebenen gezeigt %v", roll.rollen, eb.gezeigt)
	}
	r := eb.knopfRahmen[ebKnopfHoch]
	if l.imSichtfeld(r) {
		t.Fatalf("Kontrolle: Ebenenknopf %v liegt schon im Bild %v", r, roll.sicht)
	}
	l.fokusiere(&eb.knoepfe[ebKnopfHoch])
	if !l.imSichtfeld(r) {
		t.Errorf("Knopf %v mit Fokus nicht im Bild %v (Versatz %d)", r, roll.sicht, roll.rollen)
	}
}

// F-612 (KS-9, KM15): Schrumpft der Inhalt (die Auswahl fällt weg), wird der Versatz noch im selben Bild gedeckelt.
func TestVersatzNachSchrumpfendemInhalt(t *testing.T) {
	l := niedrigerLauf(t)
	roll := &l.e.seite.roll
	l.rolle(1 << 20)
	voll := roll.rollMax
	if voll == 0 || roll.rollen != voll {
		t.Fatalf("Kontrolle: gerollt %d von %d", roll.rollen, voll)
	}
	l.e.s.Auswahl = 0 // ohne gewählten Text entfällt die Schriftwahl: Der Inhalt wird kürzer, der Fokus bleibt, wo er war
	l.setle()
	if roll.rollMax >= voll {
		t.Fatalf("Kontrolle: rollMax %d nicht kleiner als vorher %d", roll.rollMax, voll)
	}
	if roll.rollen > roll.rollMax {
		t.Errorf("Versatz %d über dem größten Versatz %d nach dem Schrumpfen des Inhalts", roll.rollen, roll.rollMax)
	}
}

// F-612 (KS-9, KM16): Der Daumen des Rollbalkens hat eine Mindestlänge, auch über sehr langem Inhalt.
func TestRollDaumenMindestlaenge(t *testing.T) {
	if _, n := rollDaumen(200, 200, 20000, 0); n != 20 {
		t.Errorf("Daumenlänge über 100-fachem Inhalt: %d, Soll die Mindestlänge 20 (ein Zehntel der Bahn)", n)
	}
}

// F-615 (KS-10, K220-4): Fährt Tab während einer Texteingabe von einem Knopf der Leiste weiter, bleibt es die Fokusfahrt:
// Der Editor fängt Tab nur ab, solange die Leinwand den Fokus hat. Der Text bleibt unverändert.
func TestTabVonKnopfWaehrendTexteingabeFaehrtWeiter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	sw := &l.e.seite.sw
	l.textEingabeStarten()
	vor, _ := l.e.s.TextObjekt()
	l.fokusiere(&sw.stil[0])
	if !l.fokus(&sw.stil[0]) {
		t.Fatalf("Kontrolle: Fokus nicht auf dem Fett-Knopf (%s)", menueFokusName(l))
	}
	l.tabTaste(0)
	if l.fokus(&sw.stil[0]) || l.fokus(l.e) {
		t.Errorf("Tab vom Knopf: Fokus %s, Soll das nächste Element der Leiste", menueFokusName(l))
	}
	if nach, ok := l.e.s.TextObjekt(); !ok || nach.Inhalt != vor.Inhalt {
		t.Errorf("Tab änderte den Text: %q, vorher %q", nach.Inhalt, vor.Inhalt)
	}
}

// F-615 (KS-10, KM19): Verschwindet ein Feld mit halber Eingabe (Leiste klappt ein), bleibt die Eingabe nicht liegen.
func TestVerschwundenesFeldVerwirftSeineEingabe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("12")
	if !l.fokus(&fw.hex) || fw.hex.neu || fw.hex.text != "12" {
		t.Fatalf("Kontrolle: Fokus %s, neu=%v, text=%q", menueFokusName(l), fw.hex.neu, fw.hex.text)
	}
	l.fenst = image.Pt(700, 800) // unter 752 dp: die Leiste ist eingeklappt
	l.setle()
	if !fw.hex.neu || fw.hex.text != "" {
		t.Errorf("verschwundenes Feld: neu=%v, text=%q, Soll neu=true, leer", fw.hex.neu, fw.hex.text)
	}
}
