// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Karte #220, F-574: Lücken der Farbwähler-Tests aus der Sitzungs-QA vom 23.11.2026 abends — Pipette nach dem
// Schwärzen, bei Zoom ≠ 1 und mit Stauchen; die Farbzug-Tests prüfen die Farbe im Abzug, nicht nur ein Protokoll.

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// verlaufBild ist ein Bild, dessen Pixel (x, y) die Farbe (x, y, 7) tragen: Jede Verwechslung des Orts ist sichtbar.
func verlaufBild(b, h int) *image.NRGBA {
	o := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			o.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 7, A: 255})
		}
	}
	return o
}

// pipettiere schaltet die Pipette ein und klickt an Ansichtspunkt a; liefert die aufgenommene Farbe.
func pipettiere(t *testing.T, s *Sitzung, a document.Punkt) color.NRGBA {
	t.Helper()
	if !s.SchaltePipette() {
		t.Fatal("Pipette geht nicht an")
	}
	p := s.Abb.ZuFenster(a)
	s.Druecken(p, TastePrimaer)
	s.Loslassen(p)
	if s.PipetteLaeuft() {
		t.Fatal("Pipette läuft nach dem Klick weiter")
	}
	return s.AktuelleFarbe()
}

// Pipette nach dem Schwärzen: Sie nimmt das Schwarz, nicht die Farbe, die darunter stand (Schwärzen ist endgültig).
func TestPipetteNachSchwaerzen(t *testing.T) {
	d, err := document.Neu(verlaufBild(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(20, 20, 60, 60), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	if got, want := pipettiere(t, s, document.Pt(30.5, 30.5)), (color.NRGBA{A: 255}); got != want {
		t.Errorf("im Geschwärzten: %v, erwartet %v", got, want)
	}
	if got, want := pipettiere(t, s, document.Pt(70.5, 30.5)), (color.NRGBA{R: 70, G: 30, B: 7, A: 255}); got != want {
		t.Errorf("neben dem Geschwärzten: %v, erwartet %v", got, want)
	}
}

// Pipette bei Zoom ≠ 1: der Bildpunkt unter dem Zeiger, nicht der unter dem Fensterpunkt bei Zoom 1.
func TestPipetteBeiZoom(t *testing.T) {
	for _, zoom := range []int{50, 200, 400} {
		d, err := document.Neu(verlaufBild(200, 100))
		if err != nil {
			t.Fatal(err)
		}
		s := NeueSitzung(d)
		s.SetzeFenster(image.Pt(248, 148))
		s.SetzeZoom(zoom)
		if s.Abb.Zoom == 1 {
			t.Fatalf("Zoom %d: Abbildung bleibt bei 1", zoom)
		}
		// Mitte des Fensters: Der Ansichtspunkt dort bestimmt den Soll-Bildpunkt über die Abbildung.
		mitte := s.Abb.ZuAnsicht(document.Pt(124, 74))
		got := pipettiere(t, s, mitte)
		ort := image.Pt(int(mitte.X), int(mitte.Y))
		want := color.NRGBA{R: uint8(ort.X), G: uint8(ort.Y), B: 7, A: 255}
		if got != want {
			t.Errorf("Zoom %d: %v, erwartet %v (Bildpunkt %v)", zoom, got, want, ort)
		}
	}
}

// Pipette mit Stauchen und Zuschnitt: Die Farbe kommt vom Originalpunkt hinter dem Streifen, nicht vom Ansichtspunkt.
func TestPipetteMitStauchen(t *testing.T) {
	d, err := document.Neu(verlaufBild(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(20, 10, 200, 100)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 60, Bis: 80}}); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(208, 138))
	// Ansicht: 160 breit; der Streifen (Original-Spalten 60 bis 80) fehlt, ab Ansichtsspalte 40 liegt Original 80.
	for _, c := range []struct {
		ansicht document.Punkt
		ort     image.Point
	}{
		{document.Pt(10.5, 5.5), image.Pt(30, 15)}, // vor dem Streifen: Zuschnitt-Versatz
		{document.Pt(40.5, 5.5), image.Pt(80, 15)}, // gleich hinter der Naht: springt über 20 Spalten
		{document.Pt(100.5, 50.5), image.Pt(140, 60)},
	} {
		got := pipettiere(t, s, c.ansicht)
		want := color.NRGBA{R: uint8(c.ort.X), G: uint8(c.ort.Y), B: 7, A: 255}
		if got != want {
			t.Errorf("Ansicht %v: %v, erwartet Original %v = %v", c.ansicht, got, c.ort, want)
		}
	}
}

// pfeilFarbe ist die Strichfarbe des Pfeils p im Dokument d.
func pfeilFarbe(t *testing.T, d *document.Dokument, p int) color.NRGBA {
	t.Helper()
	o, ok := d.Objekt(p)
	if !ok {
		t.Fatal("Pfeil fehlt")
	}
	return o.(document.Pfeil).Strich.Farbe
}

// A-1 (stark): Schließen im Farbzug speichert den Stand vor dem Ziehen — im Abzug steht die alte Farbe.
func TestFarbZugSchliessenAbzugHatAlteFarbe(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	vor := pfeilFarbe(t, s.D, p)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	if pfeilFarbe(t, s.D, p) == vor {
		t.Fatal("Vorbedingung: der Zug färbt den Pfeil nicht")
	}
	z, abzug := s.SchliessenAnfrage()
	if z != SchlussWarten || abzug == nil {
		t.Fatalf("Schluss %v, Abzug da %v", z, abzug != nil)
	}
	if got := pfeilFarbe(t, abzug, p); got != vor {
		t.Errorf("Farbe im Abzug %v, erwartet die Farbe vor dem Zug %v", got, vor)
	}
	if s.FarbZugLaeuft() || s.D.ZugLaeuft() {
		t.Error("Farbzug läuft nach der Schließen-Anfrage weiter")
	}
}

// A-1 (stark): dasselbe für das Signal.
func TestFarbZugSignalAbzugHatAlteFarbe(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	vor := pfeilFarbe(t, s.D, p)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	zu, abzug := s.SignalAnfrage()
	if zu || abzug == nil {
		t.Fatalf("zu %v, Abzug da %v: geänderter Stand muss gespeichert werden", zu, abzug != nil)
	}
	if got := pfeilFarbe(t, abzug, p); got != vor {
		t.Errorf("Farbe im Abzug %v, erwartet die Farbe vor dem Zug %v", got, vor)
	}
}

// A-4 (stark): Esc im Farbzug stellt die Farbe wieder her, ohne Rückgängig-Schritt, und behält die Auswahl.
func TestFarbZugEscStelltFarbeWieder(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	vor, stand := pfeilFarbe(t, s.D, p), s.D.Stand()
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	s.Esc()
	if s.FarbZugLaeuft() || s.Auswahl != p || pfeilFarbe(t, s.D, p) != vor || s.D.Stand() != stand {
		t.Errorf("nach Esc: Zug %v, Auswahl %d, Farbe %v (vorher %v), Stand geändert %v", s.FarbZugLaeuft(), s.Auswahl,
			pfeilFarbe(t, s.D, p), vor, s.D.Stand() != stand)
	}
}

// Zeiger-Cancel im Farbzug (BrecheFarbZugAb): Farbe des Objekts wie vor dem Ziehen, kein Schritt im Verlauf.
func TestFarbZugAbbruchStelltFarbeWieder(t *testing.T) {
	s, _, p := sitzung(t)
	s.Auswahl = p
	s.SetzeFarbe(Beschriftungsfarben[2])
	vor, stand := pfeilFarbe(t, s.D, p), s.D.Stand()
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 200, S: 1, V: 1}, 255)
	s.BrecheFarbZugAb()
	if s.FarbZugLaeuft() || s.D.ZugLaeuft() || pfeilFarbe(t, s.D, p) != vor || s.D.Stand() != stand {
		t.Errorf("nach Abbruch: Zug %v, Farbe %v (vorher %v), Stand geändert %v", s.FarbZugLaeuft(),
			pfeilFarbe(t, s.D, p), vor, s.D.Stand() != stand)
	}
}
