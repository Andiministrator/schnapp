// SPDX-License-Identifier: GPL-3.0-or-later

// Zusatztests des Verify-Laufs zur Nacharbeit #383 (Fable, 27.11.2026, F-934 bis F-936), übernommen; V7 war ein Fehler
// im Test (mit gewähltem Objekt liefert stand() zu Recht das Objekt) und fehlt.

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

func verifyBreite(t *testing.T, s *Sitzung, art string) float64 {
	t.Helper()
	for _, e := range s.WerkzeugStile() {
		if e.Art == art {
			return e.Breite
		}
	}
	t.Fatalf("kein Stil %s", art)
	return 0
}

func verify4K(t *testing.T) (*Sitzung, *document.Dokument) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160))) // Faktor 2
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeWerkzeugStile(nil)
	return s, d
}

// V1: Schnellstil nach einem Zuschnitt (Faktor 2 -> 1).
func TestVerifySchnellstilNachZuschnitt(t *testing.T) {
	s, d := verify4K(t)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 500, 200)); err != nil {
		t.Fatal(err)
	}
	st := EingebauteSchnellstile(SchnellPfeil)[0]
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.WendeSchnellstilAn(st)
	t.Logf("Schnellstil Breite (Faktor 1) %v, Werkzeug %v, passt=%v", st.Breite, s.Stil.Breite, s.SchnellstilPasst(st))
	if b := verifyBreite(t, s, settings.ArtPfeil); b != st.Breite {
		t.Errorf("tools.ini bekommt width = %v, der gewählte Schnellstil ist %v", b, st.Breite)
	}
}

// V1b: eigener Schnellstil aus dem Stand nach einem Zuschnitt.
func TestVerifySchnellstilAusStandNachZuschnitt(t *testing.T) {
	s, d := verify4K(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	vor, _ := s.SchnellstilAusStand(SchnellPfeil)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 500, 200)); err != nil {
		t.Fatal(err)
	}
	nach, _ := s.SchnellstilAusStand(SchnellPfeil)
	if vor.Breite != nach.Breite {
		t.Errorf("eigener Stil aus unverändertem Stand: vor dem Zuschnitt %v, danach %v", vor.Breite, nach.Breite)
	}
	t.Logf("eingebauter Stil passt nach dem Zuschnitt: %d", s.PassenderSchnellstil(SchnellPfeil))
}

// V2: Regler/Stufe nach einem Zuschnitt.
func TestVerifyBreiteNachZuschnitt(t *testing.T) {
	s, d := verify4K(t)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 500, 200)); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeBreite(8)
	if b := verifyBreite(t, s, settings.ArtPfeil); b != 8 {
		t.Errorf("8 px bei Faktor 1 des Dokuments gewählt, tools.ini bekommt %v", b)
	}
	l, w := s.SchattenReglerMax()
	t.Logf("SchattenReglerMax nach Zuschnitt %v/%v, Schatten des Werkzeugs %+v", l, w, s.Stil.Schatten)
}

// V3: Rundung bei Faktor 3: gewählter ganzer Wert kommt nicht ganz zurück.
func TestVerifyRundungFaktor3(t *testing.T) {
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 4000, 3240)))
	s := NeueSitzung(d)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeBreite(7)
	s.SetzeWerkzeug(WerkzeugText)
	s.SetzeSchriftgrad(20)
	aus := s.WerkzeugStile()
	s2 := NeueSitzung(d)
	s2.SetzeWerkzeugStile(aus)
	if b := s2.stilVon(WerkzeugPfeil).Breite; b != 7 {
		t.Errorf("Faktor %v: 7 px gewählt, nächste Sitzung %v", s.faktor(), b)
	}
	if g := s2.stilVon(WerkzeugText).Schriftgrad; g != 20 {
		t.Errorf("Faktor %v: Grad 20 gewählt, nächste Sitzung %v", s.faktor(), g)
	}
}

// V4: Abbruch, wenn das Werkzeug des Objekts nicht das aktive ist (Zweig stile-Karte).
func TestVerifyAbbruchFremdesWerkzeug(t *testing.T) {
	s, k, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	vorK, vorP := s.stilVon(WerkzeugKasten), s.stilVon(WerkzeugPfeil)
	if !s.BeginneBreiteZug() {
		t.Fatal("kein Zug")
	}
	s.BreiteZugSetze(31)
	if s.stilVon(WerkzeugKasten).Breite != 31 {
		t.Fatalf("Kontrolle: Kasten merkt %v", s.stilVon(WerkzeugKasten).Breite)
	}
	s.Esc()
	if s.stilVon(WerkzeugKasten) != vorK || s.stilVon(WerkzeugPfeil) != vorP {
		t.Errorf("Abbruch: Kasten %+v, Pfeil %+v", s.stilVon(WerkzeugKasten), s.stilVon(WerkzeugPfeil))
	}
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	s.Abbrechen()
	if s.stilVon(WerkzeugKasten) != vorK || s.stilVon(WerkzeugPfeil) != vorP {
		t.Errorf("Farbabbruch: Kasten %+v, Pfeil %+v", s.stilVon(WerkzeugKasten), s.stilVon(WerkzeugPfeil))
	}
	// Abgeschlossener Zug, danach ein Abbruch ohne laufenden Zug: nichts wird zurückgespielt.
	s.BeginneBreiteZug()
	s.BreiteZugSetze(9)
	s.BeendeBreiteZug()
	s.SetzeBreite(11)
	s.BrecheBreiteZugAb()
	s.BrecheFarbZugAb()
	s.Abbrechen()
	if b := s.stilVon(WerkzeugKasten).Breite; b != 11 {
		t.Errorf("alter Merker zurückgespielt: %v", b)
	}
}

// V5: Texteingabe mit Esc: Stil des Werkzeugs bleibt geändert (Festhalten des Verhaltens).
func TestVerifyTextEscBehaeltStil(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	vor := s.Stil.Schriftgrad
	s.Druecken(f(20, 20), TastePrimaer)
	s.Loslassen(f(20, 20))
	if s.zug != zugText {
		t.Skip("keine Texteingabe")
	}
	s.SetzeSchriftgrad(vor + 17)
	s.Esc()
	t.Logf("Schriftgrad vor %v, nach Esc der Texteingabe %v", vor, s.Stil.Schriftgrad)
}

// V6: Wechsel des Werkzeugs nach Zuschnitt: Voreinstellung des noch nie benutzten Werkzeugs.
func TestVerifyVoreinstellungNachZuschnitt(t *testing.T) {
	s, d := verify4K(t)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 500, 200)); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if b := verifyBreite(t, s, settings.ArtKasten); b != StandardStil().Breite {
		t.Errorf("Kasten nie angefasst, tools.ini bekommt %v statt %v", b, StandardStil().Breite)
	}
}
func TestVerifyWinzig(t *testing.T) {
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
	s := NeueSitzung(d)
	s.SetzeWerkzeugStile([]settings.WerkzeugStil{{Stil: settings.Stil{Art: settings.ArtPfeil, Breite: 0.0004},
		Gesetzt: map[string]bool{"width": true}}})
	inhalt, err := settings.FormatWerkzeugStile(s.WerkzeugStile())
	t.Logf("Breite im Stil %v, Datei (%v):\n%s", s.stilVon(WerkzeugPfeil).Breite, err, inhalt)
}
