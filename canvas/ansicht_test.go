// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// F-111: Sitzungstests mit Zuschnitt und Streifen. Sie rechnen vom Fenster über die Ansicht ins Original: Einpassen
// richtet sich nach der Größe der Ansicht, Treffer und Verschieben nach den Originalkoordinaten.

// sitzungMitAnsicht: Original 200×100, Zuschnitt (20,10)–(200,100), Spalten 60 bis 80 herausgeschnitten → Ansicht
// 160×90. Ein Kasten im Original (90,20)–(150,40) liegt in der Ansicht bei (50,10)–(110,30). Fenster 208×138: Mit
// Rand 24 passt die Ansicht genau 1:1 (das Original nicht: es ergäbe Zoom 0,8).
func sitzungMitAnsicht(t *testing.T) (*Sitzung, int) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	k, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(90, 20), document.Pt(150, 40)),
		Fuellung: color.NRGBA{A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(20, 10, 200, 100)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 60, Bis: 80}}); err != nil {
		t.Fatal(err)
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	if g := d.Groesse(); g != image.Pt(160, 90) {
		t.Fatalf("Ansicht %v, Soll 160×90", g)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(208, 138))
	return s, k
}

// fa rechnet einen Ansichtspunkt in Fensterkoordinaten (Zoom 1, Versatz 24).
func fa(x, y float64) document.Punkt { return document.Pt(x+24, y+24) }

func TestSitzungEinpassenFolgtDerAnsicht(t *testing.T) {
	s, _ := sitzungMitAnsicht(t)
	if s.Abb.Zoom != 1 || !nah(s.Abb.Versatz, document.Pt(24, 24)) {
		t.Fatalf("Abbildung %+v, Soll Zoom 1 mit Versatz (24,24) für die Ansicht 160×90", s.Abb)
	}
	// Ein neuer Zuschnitt ändert die Größe der Ansicht; Einpassen folgt ihr: 100×50 → Fenster 208×138 → Zoom 1 (nicht
	// größer), mittig.
	if err := s.D.SetzeZuschnitt(image.Rect(100, 50, 200, 100)); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	if g := s.D.Groesse(); g != image.Pt(100, 50) {
		t.Fatalf("Ansicht %v, Soll 100×50", g)
	}
	if s.Abb.Zoom != 1 || !nah(s.Abb.Versatz, document.Pt(54, 44)) {
		t.Errorf("Abbildung %+v, Soll Zoom 1, Versatz (54,44) = (Fenster − Ansicht)/2", s.Abb)
	}
	// Eine große Ansicht wird verkleinert eingepasst: Fenster 108×58 → Rand 24 lässt 60×10 → Zoom 0,2 für 100×50.
	s.SetzeFenster(image.Pt(108, 58))
	s.Einpassen()
	if !nah(document.Pt(s.Abb.Zoom, 0), document.Pt(0.2, 0)) || !nah(s.Abb.Versatz, document.Pt(44, 24)) {
		t.Errorf("Abbildung %+v, Soll Zoom 0,2 mit Versatz (44,24)", s.Abb)
	}
}

// Treffer rechnen über Zuschnitt und Streifen ins Original: Ein Klick auf den Kasten in der Ansicht wählt ihn; ein
// Klick an der Stelle, an der er läge, wenn man Zuschnitt und Streifen vergäße, nicht.
func TestSitzungTrefferMitZuschnittUndStreifen(t *testing.T) {
	s, k := sitzungMitAnsicht(t)
	s.Druecken(fa(52, 20), TastePrimaer) // Ansicht (52,20) = Original (92,30), im Kasten
	s.Loslassen(fa(52, 20))
	if s.Auswahl != k {
		t.Fatalf("Auswahl %d, Soll der Kasten %d", s.Auswahl, k)
	}
	s.Abwaehlen()
	for _, p := range []document.Punkt{fa(120, 20), fa(30, 20), fa(52, 40), fa(105, 0)} {
		// (120,20) wäre ohne Abbildung im Kasten (Original 120), ist aber Original 160; (30,20) ist Original 50.
		s.Druecken(p, TastePrimaer)
		s.Loslassen(p)
		if s.Auswahl != 0 {
			t.Errorf("Klick bei %v wählte %d, Soll keinen Treffer", p, s.Auswahl)
		}
	}
	// Die Ränder in der Ansicht: 50 bis 110 und 10 bis 30 (mit Toleranz 6).
	for _, p := range []document.Punkt{fa(108, 20), fa(52, 12)} {
		s.Druecken(p, TastePrimaer)
		s.Loslassen(p)
		if s.Auswahl != k {
			t.Errorf("Klick bei %v: Auswahl %d, Soll der Kasten", p, s.Auswahl)
		}
		s.Abwaehlen()
	}
}

// Verschieben über den Schnitt: Ein Zug in der Ansicht um −22 (von 52 nach 30) überspringt den herausgeschnittenen
// Streifen (20 Pixel) und verschiebt im Original um −42. Ein Rückgängig stellt den Anfang her.
func TestSitzungVerschiebenUeberDenSchnitt(t *testing.T) {
	s, k := sitzungMitAnsicht(t)
	s.Druecken(fa(52, 20), TastePrimaer)
	s.Ziehen(fa(40, 20))
	s.Loslassen(fa(30, 20))
	o, _ := s.D.Objekt(k)
	b := o.(document.Kasten).Bereich
	if !nah(b.Min, document.Pt(48, 20)) || !nah(b.Max, document.Pt(108, 40)) {
		t.Errorf("Kasten nach dem Zug %v bis %v, Soll (48,20) bis (108,40)", b.Min, b.Max)
	}
	if !s.Geaendert() || !s.Rueckgaengig() {
		t.Fatal("Geändert oder Rückgängig")
	}
	o, _ = s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(90, 20)) {
		t.Errorf("nach Rückgängig %v", b.Min)
	}
	// Ohne Überqueren des Schnitts folgt der Kasten 1:1: Ansicht +10 = Original +10.
	s.Druecken(fa(60, 20), TastePrimaer)
	s.Loslassen(fa(70, 25))
	o, _ = s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(100, 25)) {
		t.Errorf("Zug ohne Schnitt: %v, Soll (100,25)", b.Min)
	}
}

// Zeilenstreifen ebenso: Original 100×100, Zeilen 30 bis 50 herausgeschnitten → Ansicht 100×80; Kasten im Original
// (10,60)–(40,90), in der Ansicht (10,40)–(40,70).
func TestSitzungVerschiebenUeberZeilenstreifen(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 100, 100)))
	if err != nil {
		t.Fatal(err)
	}
	k, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 60), document.Pt(40, 90)), Fuellung: color.NRGBA{A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 30, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(148, 128))
	if s.Abb.Zoom != 1 || !nah(s.Abb.Versatz, document.Pt(24, 24)) {
		t.Fatalf("Abbildung %+v, Soll Zoom 1, Versatz (24,24)", s.Abb)
	}
	s.Druecken(fa(20, 45), TastePrimaer) // Original (20,65)
	s.Loslassen(fa(20, 20))              // Original (20,20): 45 nach oben
	o, _ := s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(10, 15)) || !nah(b.Max, document.Pt(40, 45)) {
		t.Errorf("Kasten %v bis %v, Soll (10,15) bis (40,45)", b.Min, b.Max)
	}
}
