// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// sitzung: Bild 200×100 in einem Fenster 248×148 — eingepasst mit Rand 24 also 1:1, Versatz (24,24).
func sitzung(t *testing.T) (*Sitzung, int, int) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	k, _ := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 50)),
		Fuellung: color.NRGBA{A: 255}})
	p, _ := d.Hinzufuegen(document.GeraderPfeil(document.Pt(100, 50), document.Pt(180, 50), rot))
	if err := d.VergissVerlauf(); err != nil { // wie eine frisch geöffnete Datei
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	if s.Abb.Zoom != 1 || !nah(s.Abb.Versatz, document.Pt(24, 24)) {
		t.Fatalf("Abbildung %+v", s.Abb)
	}
	return s, k, p
}

// f rechnet einen Bildpunkt (Original = Ansicht hier) in Fensterkoordinaten.
func f(x, y float64) document.Punkt { return document.Pt(x+24, y+24) }

func TestVerschiebenEinerAenderung(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	if s.Auswahl != k || !s.Zieht() {
		t.Fatalf("Auswahl %d, zieht %v", s.Auswahl, s.Zieht())
	}
	s.Ziehen(f(40, 35))
	s.Loslassen(f(50, 40))
	o, _ := s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(30, 20)) {
		t.Errorf("verschoben nach %v", b.Min)
	}
	if !s.Geaendert || s.Zieht() {
		t.Errorf("Geaendert %v, zieht %v", s.Geaendert, s.Zieht())
	}
	// Eine Änderung im Verlauf: ein Rückgängig stellt den Anfang her.
	if !s.Rueckgaengig() {
		t.Fatal("Rückgängig")
	}
	o, _ = s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(10, 10)) {
		t.Errorf("nach Rückgängig %v", b.Min)
	}
	if s.D.KannRueckgaengig() {
		t.Error("mehr als eine Änderung im Verlauf")
	}
}

// Ein Klick ohne Bewegung wählt nur aus und hinterlässt nichts im Verlauf.
func TestKlickWaehltNurAus(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(31, 31)) // unter der Schwelle
	s.Loslassen(f(31, 31))
	if s.Auswahl != k || s.Geaendert || s.D.KannRueckgaengig() {
		t.Errorf("Auswahl %d, Geaendert %v, Verlauf %v", s.Auswahl, s.Geaendert, s.D.KannRueckgaengig())
	}
	// Klick ins Leere hebt die Auswahl auf.
	s.Druecken(f(190, 90), TastePrimaer)
	s.Loslassen(f(190, 90))
	if s.Auswahl != 0 {
		t.Errorf("Auswahl %d", s.Auswahl)
	}
}

// Griff des ausgewählten Pfeils biegt ihn; F-18: während des Zugs kein Rückgängig.
func TestGriffZiehenUndKeinRueckgaengigImZug(t *testing.T) {
	s, _, p := sitzung(t)
	s.Druecken(f(120, 50), TastePrimaer) // Pfeil auswählen
	s.Loslassen(f(120, 50))
	if s.Auswahl != p {
		t.Fatalf("Auswahl %d", s.Auswahl)
	}
	s.Druecken(f(140, 50), TastePrimaer) // Bogengriff in der Mitte
	s.Ziehen(f(140, 20))
	if s.Rueckgaengig() || s.Loeschen() {
		t.Error("Rückgängig oder Löschen während des Zugs")
	}
	s.Loslassen(f(140, 10))
	o, _ := s.D.Objekt(p)
	if g := o.(document.Pfeil).Griff(); !nah(g, document.Pt(140, 10)) {
		t.Errorf("Griff bei %v", g)
	}
}

// Ziehen auf leerer Fläche verschiebt die Ansicht, nicht das Dokument; Esc stellt sie wieder her.
func TestAnsichtVerschiebenUndAbbrechen(t *testing.T) {
	s, _, _ := sitzung(t)
	s.Druecken(f(190, 90), TastePrimaer)
	s.Ziehen(f(170, 80))
	if !nah(s.Abb.Versatz, document.Pt(4, 14)) {
		t.Errorf("Versatz %v", s.Abb.Versatz)
	}
	s.Abbrechen()
	if !nah(s.Abb.Versatz, document.Pt(24, 24)) || s.Geaendert {
		t.Errorf("nach Abbrechen %v, Geaendert %v", s.Abb.Versatz, s.Geaendert)
	}
	// Abbrechen eines Objektzugs stellt das Objekt her.
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(80, 80))
	s.Abbrechen()
	o, _ := s.D.Objekt(s.Auswahl)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(10, 10)) || s.D.ZugLaeuft() {
		t.Errorf("nach Abbrechen %v", b.Min)
	}
}

// Zoomen hält den Punkt unter dem Zeiger; danach folgt die Abbildung nicht mehr der Fenstergröße, bis Einpassen.
func TestZoomUndEinpassen(t *testing.T) {
	s, _, _ := sitzung(t)
	um := f(50, 50)
	s.Scrollen(um, -1)
	if s.Abb.Zoom <= 1 || !nah(s.Abb.ZuAnsicht(um), document.Pt(50, 50)) {
		t.Errorf("Zoom %v", s.Abb)
	}
	z := s.Abb
	s.SetzeFenster(image.Pt(1000, 1000))
	if s.Abb != z {
		t.Error("Fenstergröße überschreibt den Zoom des Nutzers")
	}
	s.Einpassen()
	if s.Abb.Zoom != 1 {
		t.Errorf("Einpassen: %v", s.Abb.Zoom)
	}
	s.EinsZuEins()
	if s.Abb.Zoom != 1 {
		t.Errorf("1:1: %v", s.Abb.Zoom)
	}
	s.Scrollen(um, 0)
	s.Scrollen(um, 1e300)
	if s.Abb.Zoom < MinZoom {
		t.Errorf("Zoom %v", s.Abb.Zoom)
	}
}

func TestLoeschenUndWiederholen(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	s.Loslassen(f(30, 30))
	if !s.Loeschen() || s.Auswahl != 0 {
		t.Fatal("Löschen")
	}
	if _, ok := s.D.Objekt(k); ok {
		t.Fatal("noch da")
	}
	s.Rueckgaengig()
	if _, ok := s.D.Objekt(k); !ok {
		t.Fatal("Rückgängig bringt es nicht zurück")
	}
	s.Wiederholen()
	if _, ok := s.D.Objekt(k); ok {
		t.Fatal("Wiederholen löscht nicht")
	}
	s.Gespeichert()
	if s.Geaendert {
		t.Error("Gespeichert")
	}
}

// Mittlere Taste verschiebt die Ansicht auch über einem Objekt.
func TestMittlereTaste(t *testing.T) {
	s, _, _ := sitzung(t)
	s.Druecken(f(30, 30), TasteMitte)
	s.Ziehen(f(40, 30))
	s.Loslassen(f(40, 30))
	if s.Auswahl != 0 || !nah(s.Abb.Versatz, document.Pt(34, 24)) || s.Geaendert {
		t.Errorf("Auswahl %d, Versatz %v", s.Auswahl, s.Abb.Versatz)
	}
}

// Während eines Zugs ändern Zoom und Einpassen die Abbildung nicht — sonst spränge das gezogene Objekt (Kritik A-6).
func TestKeinZoomImZug(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Druecken(f(30, 30), TastePrimaer)
	s.Ziehen(f(60, 30))
	vorher := s.Abb
	s.Scrollen(f(60, 30), -3)
	s.Einpassen()
	s.EinsZuEins()
	s.SetzeFenster(image.Pt(500, 500))
	if s.Abb != vorher {
		t.Errorf("Abbildung im Zug geändert: %+v → %+v", vorher, s.Abb)
	}
	s.Loslassen(f(60, 30))
	o, _ := s.D.Objekt(k)
	if b := o.(document.Kasten).Bereich; !nah(b.Min, document.Pt(40, 10)) {
		t.Errorf("Kasten bei %v, Soll (40,10)", b.Min)
	}
}
