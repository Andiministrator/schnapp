// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0041 Punkt 1: Jedes Werkzeug merkt seinen Stil; ein Wechsel nimmt nichts mit. Text und Sprechblase teilen sich
// einen.
func TestStilJeWerkzeugGehtNichtUeber(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeBreite(12)
	s.SetzeStrichart(document.StrichGestrichelt)
	s.SchalteSchatten()
	s.SetzeKopf(false, document.KopfPunkt)

	s.SetzeWerkzeug(WerkzeugKasten)
	std := StandardStil()
	if s.Stil.Breite != std.Breite || s.Stil.Strichart != document.StrichVoll || s.Stil.Schatten.An {
		t.Fatalf("Kasten übernimmt den Pfeil: %+v", s.Stil)
	}
	s.SetzeBreite(3)

	s.SetzeWerkzeug(WerkzeugPfeil)
	if s.Stil.Breite != 12 || s.Stil.Strichart != document.StrichGestrichelt || !s.Stil.Schatten.An ||
		s.Stil.KopfEnde != document.KopfPunkt {
		t.Fatalf("Pfeil vergisst seinen Stil: %+v", s.Stil)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.Stil.Breite != 3 {
		t.Fatalf("Kasten vergisst seine Breite: %v", s.Stil.Breite)
	}
	// Auswählen hat keinen Stil: Stil bleibt der des Kastens.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	if s.Stil.Breite != 3 {
		t.Fatalf("Auswählen tauscht den Stil: %v", s.Stil.Breite)
	}
	// Text und Sprechblase teilen sich den Stil.
	s.SetzeWerkzeug(WerkzeugText)
	s.SetzeSchriftgrad(33)
	s.SetzeWerkzeug(WerkzeugSprechblase)
	if s.Stil.Schriftgrad != 33 {
		t.Fatalf("Sprechblase hat nicht den Stil des Texts: %v", s.Stil.Schriftgrad)
	}
}

// ADR-0041 Punkt 2: Eine Änderung an einem gewählten Objekt merkt das Werkzeug dieses Objekts, nicht das aktive.
func TestStilAenderungAmObjektMerktSeinWerkzeug(t *testing.T) {
	s, k, p := sitzung(t)
	s.SetzeWerkzeug(WerkzeugNummer) // aktiver Stil: Nummer
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = p // Pfeil
	s.SetzeBreite(16)
	s.SchalteSchatten()
	if s.Stil.Breite == 16 || s.Stil.Schatten.An {
		t.Errorf("Stil der Nummer übernimmt die Änderung am Pfeil: %+v", s.Stil)
	}
	if st := s.stilVon(WerkzeugPfeil); st.Breite != 16 || !st.Schatten.An {
		t.Errorf("Pfeil merkt die Änderung nicht: %+v", st)
	}
	s.Auswahl = k // Kasten: Farbe
	blau := Beschriftungsfarben[1]
	s.SetzeFarbe(blau)
	if s.stilVon(WerkzeugKasten).Farbe != blau || s.stilVon(WerkzeugPfeil).Farbe == blau {
		t.Errorf("Farbe: Kasten %v, Pfeil %v", s.stilVon(WerkzeugKasten).Farbe, s.stilVon(WerkzeugPfeil).Farbe)
	}
}

// ADR-0041 Punkte 3 und 4: WerkzeugStile liefert die Stile auf Faktor 1, SetzeWerkzeugStile nimmt sie mal dem Faktor
// der Sitzung; nicht gesetzte Schlüssel behalten die Voreinstellung, zu große Werte ebenso.
func TestWerkzeugStileFaktor(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2400, 2160))) // Faktor 2
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeBreite(20)
	s.SetzeWerkzeug(WerkzeugKasten)
	var pfeil settings.WerkzeugStil
	for _, e := range s.WerkzeugStile() {
		if e.Art == settings.ArtPfeil {
			pfeil = e
		}
	}
	if pfeil.Breite != 10 {
		t.Fatalf("Pfeil auf Faktor 1: %v, Soll 10", pfeil.Breite)
	}
	// In eine Sitzung mit Faktor 2 geladen: wieder 20 px.
	d2, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2400, 2160)))
	s2 := NeueSitzung(d2)
	s2.SetzeWerkzeugStile([]settings.WerkzeugStil{pfeil})
	if st := s2.stilVon(WerkzeugPfeil); st.Breite != 20 {
		t.Fatalf("Pfeil bei Faktor 2 geladen: %v, Soll 20", st.Breite)
	}

	d1, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300))) // Faktor 1
	s1 := NeueSitzung(d1)
	rot := color.NRGBA{R: 200, A: 255}
	s1.SetzeWerkzeugStile([]settings.WerkzeugStil{
		pfeil,
		{Stil: settings.Stil{Art: settings.ArtKasten, Farbe: rot, Breite: 5e9}, Gesetzt: map[string]bool{"color": true, "width": true}},
		{Stil: settings.Stil{Art: settings.ArtNummer, Radius: 30}, Gesetzt: map[string]bool{"radius": true}},
		{Stil: settings.Stil{Art: "unbekannt", Breite: 1}},
	})
	if st := s1.stilVon(WerkzeugPfeil); st.Breite != 10 {
		t.Errorf("Pfeil bei Faktor 1: %v", st.Breite)
	}
	if st := s1.stilVon(WerkzeugKasten); st.Farbe != rot || st.Breite != StandardStil().Breite {
		t.Errorf("Kasten: %+v (zu große Breite bleibt Voreinstellung)", st)
	}
	if st := s1.stilVon(WerkzeugNummer); st.Radius != 30 || st.Farbe != Beschriftungsfarben[0] {
		t.Errorf("Nummer: %+v (Farbe nicht gesetzt: Voreinstellung)", st)
	}
	s1.SetzeWerkzeug(WerkzeugPfeil)
	if s1.Stil.Breite != 10 {
		t.Errorf("Wechsel zum Pfeil: %v", s1.Stil.Breite)
	}
	// Hin und zurück ohne Änderung ändert nichts.
	vor := s1.WerkzeugStile()
	s3 := NeueSitzung(d1)
	s3.SetzeWerkzeugStile(vor)
	nach := s3.WerkzeugStile()
	if len(nach) != len(vor) {
		t.Fatalf("%d Stile nach dem Übertragen, vorher %d", len(nach), len(vor))
	}
	for i := range vor {
		if vor[i].Art != nach[i].Art || vor[i].Breite != nach[i].Breite || vor[i].Farbe != nach[i].Farbe ||
			vor[i].Radius != nach[i].Radius {
			t.Errorf("[%s] %+v → %+v", vor[i].Art, vor[i], nach[i])
		}
	}
}

func werkzeugStand(t *testing.T, s *Sitzung) string {
	t.Helper()
	inhalt, err := settings.FormatWerkzeugStile(s.WerkzeugStile())
	if err != nil {
		t.Fatal(err)
	}
	return inhalt
}

// F-927 (Kritik #383 K-1): Zuschneiden ändert den Faktor des Dokuments, nicht die gemerkten Stile.
func TestZuschnittAendertGemerktenStilNicht(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160))) // Faktor 2
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeWerkzeugStile(nil)
	vor := werkzeugStand(t, s)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 500, 200)); err != nil {
		t.Fatal(err)
	}
	if d.KantenFaktor() != 1 {
		t.Fatalf("Kontrolle: Faktor nach dem Zuschnitt %d", d.KantenFaktor())
	}
	if nach := werkzeugStand(t, s); vor != nach {
		t.Errorf("Zuschneiden allein ändert tools.ini:\nvorher:\n%s\nnachher:\n%s", vor, nach)
	}
}

// F-928 (Kritik #383 K-2): Schatten einschalten an einem Objekt nimmt die Werte seines Werkzeugs, nicht des aktiven.
func TestSchattenAnFremdemObjekt(t *testing.T) {
	s, k, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	sch := s.Stil.Schatten
	sch.An, sch.Laenge = true, 3
	s.SetzeSchatten(sch)
	sch.An = false
	s.SetzeSchatten(sch)
	s.SetzeWerkzeug(WerkzeugPfeil)
	sp := s.Stil.Schatten
	sp.An, sp.Laenge = true, 33
	s.SetzeSchatten(sp)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	s.SchalteSchatten()
	o, _ := s.D.Objekt(k)
	if l := o.Ebenenwerte().Schatten.Laenge; l != 3 {
		t.Errorf("Kasten bekommt Länge %v, Soll 3 (die des Kasten-Werkzeugs)", l)
	}
	if l := s.stilVon(WerkzeugKasten).Schatten.Laenge; l != 3 {
		t.Errorf("Kasten-Werkzeug merkt Länge %v, Soll 3", l)
	}
	if l := s.stilVon(WerkzeugPfeil).Schatten.Laenge; l != 33 {
		t.Errorf("Pfeil-Werkzeug verliert seine Länge: %v", l)
	}
}

// F-930 (Kritik #383 K-4): Hin und zurück über Faktor 2, 3, 5, 7 gibt dieselben Werte (auf Tausendstel).
func TestWerkzeugStileFaktorRundung(t *testing.T) {
	for _, gr := range []image.Point{{2400, 2160}, {4000, 3240}, {6000, 5400}, {8000, 7560}} {
		d, _ := document.Neu(image.NewNRGBA(image.Rectangle{Max: gr}))
		s := NeueSitzung(d)
		for _, b := range []float64{0.1, 0.7, 1.1, 2.3, 4.35, 7.7} {
			s.SetzeWerkzeugStile([]settings.WerkzeugStil{{Stil: settings.Stil{Art: settings.ArtPfeil, Breite: b},
				Gesetzt: map[string]bool{"width": true}}})
			if aus := s.WerkzeugStile(); aus[0].Breite != b {
				t.Errorf("Faktor %v: Breite %v wird %v", s.faktor(), b, aus[0].Breite)
			}
		}
	}
}

// F-929 (Kritik #383 K-3): Ein Wert, der mal dem Faktor die Grenze überschritte, wird begrenzt, nicht verworfen.
func TestWerkzeugStileGrenzeBegrenzt(t *testing.T) {
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2400, 2160))) // Faktor 2
	s := NeueSitzung(d)
	s.SetzeWerkzeugStile([]settings.WerkzeugStil{
		{Stil: settings.Stil{Art: settings.ArtText, Schriftgrad: 3000}, Gesetzt: map[string]bool{"size": true}},
		{Stil: settings.Stil{Art: settings.ArtNummer, Radius: 3000}, Gesetzt: map[string]bool{"radius": true}},
		{Stil: settings.Stil{Art: settings.ArtPfeil, Breite: document.MaxGroesse}, Gesetzt: map[string]bool{"width": true}},
		{Stil: settings.Stil{Art: settings.ArtKasten}, Eckradius: document.MaxGroesse, Gesetzt: map[string]bool{"corner_radius": true}},
	})
	if g := s.stilVon(WerkzeugText).Schriftgrad; g != document.MaxSchrift {
		t.Errorf("Text: %v, Soll %v", g, document.MaxSchrift)
	}
	if r := s.stilVon(WerkzeugNummer).Radius; r != document.MaxSchrift {
		t.Errorf("Nummer: %v, Soll %v", r, document.MaxSchrift)
	}
	if b := s.stilVon(WerkzeugPfeil).Breite; b != document.MaxGroesse {
		t.Errorf("Pfeil: %v, Soll %v", b, document.MaxGroesse)
	}
	if e := s.stilVon(WerkzeugKasten).Eckradius; e != document.MaxGroesse {
		t.Errorf("Kasten: %v, Soll %v", e, document.MaxGroesse)
	}
	// Schatten mal Faktor, Füllung nur, wo es eine gibt, Strichart nur gültig.
	s.SetzeWerkzeugStile([]settings.WerkzeugStil{
		{Stil: settings.Stil{Art: settings.ArtLupe, Fuellung: Beschriftungsfarben[1],
			Schatten: document.Schatten{An: true, Winkel: 90, Laenge: 5, Weich: 4}}, Strichart: document.Strichart(9),
			Gesetzt: map[string]bool{"fill": true, "shadow": true, "shadow_on": true, "line_style": true}},
	})
	if st := s.stilVon(WerkzeugLupe); st.Fuellung == Beschriftungsfarben[1] || st.Schatten.Laenge != 10 ||
		st.Schatten.Weich != 8 || !st.Schatten.An || st.Strichart != document.StrichVoll {
		t.Errorf("Lupe: %+v", st)
	}
}

// F-931 (Kritik #383 K-5): Ein abgebrochener Regler-, Schatten- oder Farbzug lässt auch den Stil des Werkzeugs, wie er
// war; ein abgeschlossener merkt ihn.
func TestAbgebrochenerZugLaesstStil(t *testing.T) {
	s, _, p := sitzung(t)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = p
	vor := s.stilVon(WerkzeugPfeil)
	s.BeginneBreiteZug()
	s.BreiteZugSetze(31)
	s.BrecheBreiteZugAb()
	if b := s.stilVon(WerkzeugPfeil).Breite; b != vor.Breite {
		t.Errorf("Breitenzug abgebrochen: merkt %v statt %v", b, vor.Breite)
	}
	s.BeginneSchattenZug()
	sch := s.Schatten()
	sch.An, sch.Laenge = true, 25
	s.SchattenZugSetze(sch)
	s.BrecheBreiteZugAb()
	if l := s.stilVon(WerkzeugPfeil).Schatten; l != vor.Schatten {
		t.Errorf("Schattenzug abgebrochen: %+v statt %+v", l, vor.Schatten)
	}
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	s.BrecheFarbZugAb()
	if f := s.stilVon(WerkzeugPfeil).Farbe; f != vor.Farbe {
		t.Errorf("Farbzug abgebrochen: %v statt %v", f, vor.Farbe)
	}
	// Abgeschlossen: gemerkt.
	s.BeginneBreiteZug()
	s.BreiteZugSetze(31)
	s.BeendeBreiteZug()
	if b := s.stilVon(WerkzeugPfeil).Breite; b != 31 {
		t.Errorf("Breitenzug abgeschlossen: %v", b)
	}
}
