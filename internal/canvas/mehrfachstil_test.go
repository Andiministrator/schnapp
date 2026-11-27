// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

var (
	mRot  = color.NRGBA{R: 255, A: 255}
	mGelb = color.NRGBA{R: 255, G: 255, A: 255}
	mBlau = color.NRGBA{B: 255, A: 255}
)

// gemischt ist eine Sitzung mit Pfeil (1), Kasten (2) und Text (3) auf einem Bild 300 × 200, alle drei gewählt
// (Hauptauswahl der Kasten).
func gemischt(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 300, 200)))
	if err != nil {
		t.Fatal(err)
	}
	os := []document.Objekt{
		document.GeraderPfeil(document.Pt(20, 20), document.Pt(80, 40), document.Strich{Farbe: mRot, Breite: 4}),
		document.Kasten{Bereich: document.R(document.Pt(100, 20), document.Pt(160, 80)),
			Strich: document.Strich{Farbe: mGelb, Breite: 6}, Fuellung: color.NRGBA{A: 0}},
		document.Text{Ort: document.Pt(200, 100), Inhalt: "Hallo", Schriftgrad: 37, Farbe: mBlau},
	}
	for _, o := range os {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	_ = d.VergissVerlauf()
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(348, 248))
	s.setzeAuswahl(2, []int{1, 3})
	return s
}

func objekt(t *testing.T, s *Sitzung, id int) document.Objekt {
	t.Helper()
	o, ok := s.D.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o
}

// Punkt 1: Die Strichstärke trifft Pfeil und Kasten, nicht den Text, als ein Schritt; beide Werkzeuge merken sie
// (Punkt 2), der Text nicht.
func TestMehrereBreiteEinSchritt(t *testing.T) {
	s := gemischt(t)
	textVor := s.stilVon(WerkzeugText).Breite
	if !s.SetzeBreite(12) {
		t.Fatal("SetzeBreite änderte nichts")
	}
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Breite != 12 {
		t.Errorf("Pfeil %v", p.Strich.Breite)
	}
	if k := objekt(t, s, 2).(document.Kasten); k.Strich.Breite != 12 {
		t.Errorf("Kasten %v", k.Strich.Breite)
	}
	if s.stilVon(WerkzeugPfeil).Breite != 12 || s.stilVon(WerkzeugKasten).Breite != 12 {
		t.Errorf("Werkzeuge merken %v / %v", s.stilVon(WerkzeugPfeil).Breite, s.stilVon(WerkzeugKasten).Breite)
	}
	if s.stilVon(WerkzeugText).Breite != textVor {
		t.Errorf("Text-Stil geändert: %v", s.stilVon(WerkzeugText).Breite)
	}
	s.Rueckgaengig()
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Breite != 4 {
		t.Errorf("Rückgängig: Pfeil %v", p.Strich.Breite)
	}
	if k := objekt(t, s, 2).(document.Kasten); k.Strich.Breite != 6 {
		t.Errorf("Rückgängig: Kasten %v (ein Schritt erwartet)", k.Strich.Breite)
	}
}

// Punkt 4: gemischte Werte und Trägerwahl.
func TestMehrereGemischtUndTraeger(t *testing.T) {
	s := gemischt(t)
	if !s.Gemischt(EigBreite) || !s.Gemischt(EigFarbe) {
		t.Errorf("Breite/Farbe nicht gemischt")
	}
	if s.Breite() != 6 { // Hauptauswahl Kasten
		t.Errorf("Breite zeigt %v, erwartet die des Kastens", s.Breite())
	}
	if s.Gemischt(EigSchrift) || s.Gemischt(EigKopfEnde) { // nur ein Träger
		t.Errorf("ein Träger gilt als gemischt")
	}
	if c, ok := s.ZweiteFarbe(FeldVordergrund); !ok || c != mBlau { // nach der Hauptauswahl der oberste andere: Text
		t.Errorf("ZweiteFarbe %v %v", c, ok)
	}
	if g := s.AktuellerTextStil().Grad; g != 37 { // Hauptauswahl ohne Schrift: der Text
		t.Errorf("Schriftgrad %v", g)
	}
	s.SetzeBreite(9)
	if s.Gemischt(EigBreite) {
		t.Errorf("nach SetzeBreite noch gemischt")
	}
	s.setzeAuswahl(2, nil)
	if s.Gemischt(EigFarbe) {
		t.Errorf("ein gewähltes gilt als gemischt")
	}
}

// Punkt 4: Die Leiste zeigt die Abschnitte aller gewählten Arten und nennt die Zahl.
func TestMehrereSeitenleiste(t *testing.T) {
	s := gemischt(t)
	i := s.Seitenleiste()
	if i.Anzahl != 3 || !i.Strich || !i.Text || !i.Pfeilform || i.Spitze || i.Werkzeug != WerkzeugKasten {
		t.Errorf("Seitenleiste %+v", i)
	}
	s.setzeAuswahl(2, nil)
	if i := s.Seitenleiste(); i.Anzahl != 0 || i.Text || i.Pfeilform {
		t.Errorf("ein gewähltes: %+v", i)
	}
}

// Punkt 1: Farbe für alle, Füllung nur für Kasten und Text; jedes Werkzeug merkt seine Farbe.
func TestMehrereFarbeUndFuellung(t *testing.T) {
	s := gemischt(t)
	gruen := color.NRGBA{G: 200, A: 255}
	s.SetzeFarbe(gruen)
	for id := 1; id <= 3; id++ {
		if c, _ := objektFarbe(objekt(t, s, id), FeldVordergrund); c != gruen {
			t.Errorf("Objekt %d Farbe %v", id, c)
		}
	}
	if s.stilVon(WerkzeugText).Farbe != gruen || s.stilVon(WerkzeugPfeil).Farbe != gruen {
		t.Errorf("Werkzeuge merken die Farbe nicht")
	}
	if !s.Fuellbar() || !s.SetzeFeld(FeldFuellung) {
		t.Fatal("Füllung nicht wählbar")
	}
	s.SetzeFarbe(mRot)
	if k := objekt(t, s, 2).(document.Kasten); k.Fuellung != mRot {
		t.Errorf("Kasten-Füllung %v", k.Fuellung)
	}
	if x := objekt(t, s, 3).(document.Text); x.Hintergrund != mRot {
		t.Errorf("Text-Hintergrund %v", x.Hintergrund)
	}
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Farbe != gruen {
		t.Errorf("Pfeil geändert: %v", p.Strich.Farbe)
	}
}

// Ein Farbzug über mehrere ist ein Schritt; der Abbruch stellt Objekte und alle beteiligten Werkzeug-Stile her.
func TestMehrereFarbZug(t *testing.T) {
	s := gemischt(t)
	pfeilVor := s.stilVon(WerkzeugPfeil).Farbe
	if !s.BeginneFarbZug() {
		t.Fatal("kein Farbzug")
	}
	s.FarbZugHSV(HSVAus(mGelb), 255)
	s.FarbZugHSV(HSVAus(mRot), 255)
	if c, _ := objektFarbe(objekt(t, s, 3), FeldVordergrund); c != mRot {
		t.Errorf("im Zug: Text %v", c)
	}
	s.BrecheFarbZugAb()
	if c, _ := objektFarbe(objekt(t, s, 3), FeldVordergrund); c != mBlau {
		t.Errorf("nach Abbruch: Text %v", c)
	}
	if s.stilVon(WerkzeugPfeil).Farbe != pfeilVor {
		t.Errorf("Abbruch ließ den Pfeil-Stil geändert")
	}
	s.BeginneFarbZug()
	s.FarbZugHSV(HSVAus(mRot), 255)
	s.BeendeFarbZug()
	s.Rueckgaengig()
	for id, c := range map[int]color.NRGBA{1: mRot, 2: mGelb, 3: mBlau} {
		if got, _ := objektFarbe(objekt(t, s, id), FeldVordergrund); got != c {
			t.Errorf("nach Rückgängig Objekt %d %v (ein Schritt erwartet)", id, got)
		}
	}
}

// Regler der Strichstärke über mehrere.
func TestMehrereBreiteZug(t *testing.T) {
	s := gemischt(t)
	s.BeginneBreiteZug()
	s.BreiteZugSetze(20)
	s.BeendeBreiteZug()
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Breite != 20 {
		t.Errorf("Pfeil %v", p.Strich.Breite)
	}
	if s.stilVon(WerkzeugPfeil).Breite != 20 || s.stilVon(WerkzeugKasten).Breite != 20 {
		t.Errorf("Zug nicht gemerkt: %v / %v", s.stilVon(WerkzeugPfeil).Breite, s.stilVon(WerkzeugKasten).Breite)
	}
	s.Rueckgaengig()
	if k := objekt(t, s, 2).(document.Kasten); k.Strich.Breite != 6 {
		t.Errorf("Kasten %v", k.Strich.Breite)
	}
	s.SetzeBreite(5) // Stil der Werkzeuge vor dem abgebrochenen Zug: 5
	s.Rueckgaengig()
	s.BeginneBreiteZug()
	s.BreiteZugSetze(20)
	s.BrecheBreiteZugAb()
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Breite != 4 || s.stilVon(WerkzeugPfeil).Breite != 5 ||
		s.stilVon(WerkzeugKasten).Breite != 5 {
		t.Errorf("Abbruch: Pfeil %v, Stil %v", p.Strich.Breite, s.stilVon(WerkzeugPfeil).Breite)
	}
}

// Strichart und Spitzen: nur die Träger.
func TestMehrereStrichartUndSpitzen(t *testing.T) {
	s := gemischt(t)
	if !s.SetzeStrichart(document.StrichGestrichelt) {
		t.Fatal("Strichart änderte nichts")
	}
	if objekt(t, s, 1).(document.Pfeil).Strich.Art != document.StrichGestrichelt ||
		objekt(t, s, 2).(document.Kasten).Strich.Art != document.StrichGestrichelt {
		t.Errorf("Strichart nicht an beiden")
	}
	if !s.SetzeKopf(false, document.KopfPunkt) {
		t.Fatal("Spitze änderte nichts")
	}
	if objekt(t, s, 1).(document.Pfeil).Ende != document.KopfPunkt || s.stilVon(WerkzeugPfeil).KopfEnde != document.KopfPunkt {
		t.Errorf("Spitze nicht gesetzt oder nicht gemerkt")
	}
}

// Punkt 3: Schatten — einschalten, wenn einer aus ist; geändert wird nur der verschobene Wert.
func TestMehrereSchatten(t *testing.T) {
	s := gemischt(t)
	if !s.SchalteSchatten() {
		t.Fatal("Schatten nicht eingeschaltet")
	}
	for id := 1; id <= 3; id++ {
		if !objekt(t, s, id).Ebenenwerte().Schatten.An {
			t.Errorf("Objekt %d ohne Schatten", id)
		}
	}
	// Länge des Pfeils eigens ändern, dann die Richtung an der Leiste: die Länge des Pfeils bleibt.
	p := objekt(t, s, 1)
	sp := p.Ebenenwerte().Schatten
	sp.Laenge = 30
	if err := s.D.Ersetzen(mitSchatten(p, sp)); err != nil {
		t.Fatal(err)
	}
	sch := s.Schatten()
	sch.Winkel = 45
	s.SetzeSchatten(sch)
	for id := 1; id <= 3; id++ {
		if w := objekt(t, s, id).Ebenenwerte().Schatten.Winkel; w != 45 {
			t.Errorf("Objekt %d Winkel %v", id, w)
		}
	}
	if l := objekt(t, s, 1).Ebenenwerte().Schatten.Laenge; l != 30 {
		t.Errorf("Länge des Pfeils %v, erwartet 30", l)
	}
	if s.stilVon(WerkzeugText).Schatten.Winkel != 45 {
		t.Errorf("Text-Werkzeug merkt den Winkel nicht")
	}
	// Einer aus: Schalter schaltet alle ein; alle an: alle aus (Nullwert).
	if !s.SchalteSchatten() {
		t.Fatal("ausschalten")
	}
	for id := 1; id <= 3; id++ {
		if sch := objekt(t, s, id).Ebenenwerte().Schatten; sch != (document.Schatten{}) {
			t.Errorf("Objekt %d Schatten %+v", id, sch)
		}
	}
	if s.stilVon(WerkzeugText).Schatten.An || s.stilVon(WerkzeugText).Schatten.Winkel != 45 {
		t.Errorf("Werkzeug behält die Werte nicht: %+v", s.stilVon(WerkzeugText).Schatten)
	}
}

// Schattenregler über mehrere: Objekte ohne Schatten bleiben ohne.
func TestMehrereSchattenZug(t *testing.T) {
	s := gemischt(t)
	p := objekt(t, s, 1)
	if err := s.D.Ersetzen(mitSchatten(p, document.StandardSchatten(1))); err != nil {
		t.Fatal(err)
	}
	s.BeginneSchattenZug()
	sch := s.Schatten()
	sch.Laenge = 17
	s.SchattenZugSetze(sch)
	s.BeendeBreiteZug()
	if l := objekt(t, s, 1).Ebenenwerte().Schatten.Laenge; l != 17 {
		t.Errorf("Pfeil Länge %v", l)
	}
	if objekt(t, s, 2).Ebenenwerte().Schatten.An {
		t.Errorf("Kasten bekam einen Schatten")
	}
}

// Schrift für alle Texte; Schnellstil nur für die gewählten seiner Art.
func TestMehrereTextUndSchnellstil(t *testing.T) {
	s := gemischt(t)
	if !s.SetzeSchriftgrad(40) {
		t.Fatal("Schriftgrad änderte nichts")
	}
	if objekt(t, s, 3).(document.Text).Schriftgrad != 40 {
		t.Errorf("Text nicht geändert")
	}
	st := EingebauteSchnellstile(SchnellPfeil)[1]
	s.WendeSchnellstilAn(st)
	if objekt(t, s, 1).(document.Pfeil).Strich.Farbe != st.Farbe {
		t.Errorf("Pfeil nicht im Schnellstil")
	}
	if k := objekt(t, s, 2).(document.Kasten); k.Strich.Farbe != mGelb {
		t.Errorf("Kasten vom Pfeil-Stil geändert: %v", k.Strich.Farbe)
	}
}

// Tauschen: nur Objekte mit Füllung.
func TestMehrereTauschen(t *testing.T) {
	s := gemischt(t)
	s.TauscheFarben()
	if k := objekt(t, s, 2).(document.Kasten); k.Strich.Farbe != (color.NRGBA{}) || k.Fuellung != mGelb {
		t.Errorf("Kasten nicht getauscht: %+v", k)
	}
	if p := objekt(t, s, 1).(document.Pfeil); p.Strich.Farbe != mRot {
		t.Errorf("Pfeil geändert")
	}
}

// In der Texteingabe gilt nur der Text der Eingabe.
func TestMehrereNichtInTexteingabe(t *testing.T) {
	s := gemischt(t)
	s.zug = zugText
	s.textID = 3
	if s.stilMehrere() {
		t.Error("stilMehrere in der Texteingabe")
	}
}

// Punkt 5: Ebenenliste mit Strg und Umschalt.
func TestEbenenlisteMehrere(t *testing.T) {
	s := gemischt(t)
	s.WaehleEbene(1)
	if !s.WaehleEbeneMehr(3, false) || !slices.Equal(s.Gewaehlt(), []int{1, 2, 3}) || s.Auswahl != 3 {
		t.Fatalf("Umschalt: %v Haupt %d", s.Gewaehlt(), s.Auswahl)
	}
	if !s.WaehleEbeneMehr(2, true) || !slices.Equal(s.Gewaehlt(), []int{1, 3}) {
		t.Fatalf("Strg heraus: %v", s.Gewaehlt())
	}
	z := s.Ebenen()
	n := 0
	for _, e := range z {
		if e.Gewaehlt {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d Zeilen hervorgehoben", n)
	}
	// Gesperrtes lässt sich nicht dazunehmen: allein gewählt.
	s.SchalteGesperrt(2)
	s.WaehleEbene(1)
	s.WaehleEbeneMehr(2, true)
	if !slices.Equal(s.Gewaehlt(), []int{2}) {
		t.Errorf("gesperrtes: %v", s.Gewaehlt())
	}
	// Umschalt über ein gesperrtes hinweg lässt es aus.
	s.WaehleEbene(1)
	s.WaehleEbeneMehr(3, false)
	if !slices.Equal(s.Gewaehlt(), []int{1, 3}) {
		t.Errorf("Umschalt über gesperrtes: %v", s.Gewaehlt())
	}
	// Entsperrt kommt es nicht dazu: Es war nie gewählt.
	s.SchalteGesperrt(2)
	if !slices.Equal(s.Gewaehlt(), []int{1, 3}) {
		t.Errorf("nach Entsperren: %v", s.Gewaehlt())
	}
}
