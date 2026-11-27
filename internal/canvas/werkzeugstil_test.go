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
