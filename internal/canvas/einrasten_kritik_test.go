// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Kritik zu Karte #196 (ADR-0021 Nachtrag 12): Einrasten bei Zuschnitt und Stauchen, Griff über die Gegenseite,
// Abbruch, Alt ohne Zeigerbewegung.

// einrastAnsicht liefert eine Sitzung auf einem 200×100-Bild mit Zuschnitt z und Streifen st, Zoom 1, und den Kasten A
// (10,10)–(30,40) bzw. den übergebenen Bereich.
func einrastAnsicht(t *testing.T, z image.Rectangle, st []document.Streifen, a document.Rechteck) (*Sitzung, int) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	if !z.Empty() {
		if err := d.SetzeZuschnitt(z); err != nil {
			t.Fatal(err)
		}
	}
	if st != nil {
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	g := d.Groesse()
	s.SetzeFenster(image.Pt(g.X+48, g.Y+48))
	s.SetzeZoom(100)
	if s.Abb.Zoom != 1 {
		t.Fatalf("Zoom %v, Soll 1", s.Abb.Zoom)
	}
	return s, kastenAm(t, s, a)
}

// fo rechnet einen Punkt des Originals in Fensterkoordinaten (über Zuschnitt und Streifen).
func fo(s *Sitzung, x, y float64) document.Punkt {
	return s.Abb.ZuFenster(s.D.ZuAnsicht(document.Pt(x, y)))
}

// ziehVon zieht in zwei Schritten von a nach b (Original), hält vor dem Loslassen an und liefert die Linien dabei.
func ziehVon(s *Sitzung, a, b document.Punkt) []Hilfslinie {
	s.Druecken(fo(s, a.X, a.Y), TastePrimaer)
	s.Ziehen(fo(s, (a.X+b.X)/2, (a.Y+b.Y)/2))
	s.Ziehen(fo(s, b.X, b.Y))
	return s.Hilfslinien()
}

func bereich(t *testing.T, s *Sitzung, id int) document.Rechteck {
	t.Helper()
	o, ok := s.D.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o.(document.Kasten).Bereich
}

var kastenA = document.R(document.Pt(10, 10), document.Pt(30, 40))

// Die Mitte des Bilds ist die Mitte der Ansicht: Mit einem Streifen 100–160 ist das Bild 140 breit, die Mitte liegt
// bei 70 — nicht bei 100 (Mitte zwischen den Rändern des Originals), dort ist die Naht.
func TestEinrastenBildmitteBeiStreifen(t *testing.T) {
	st := []document.Streifen{{Zeilen: false, Von: 100, Bis: 160}}
	s, a := einrastAnsicht(t, image.Rectangle{}, st, kastenA)
	linien := ziehVon(s, document.Pt(20, 25), document.Pt(69, 25)) // Mitte von A bei 69, Ziel 70
	s.Loslassen(fo(s, 69, 25))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(60, 10)) || !hat(linien, Hilfslinie{Lage: 70}) {
		t.Errorf("Mitte: Bereich %v, Linien %v, Soll Min.X 60 und x=70", b, linien)
	}
	s, a = einrastAnsicht(t, image.Rectangle{}, st, kastenA)
	linien = ziehVon(s, document.Pt(20, 25), document.Pt(88, 25)) // rechte Kante bei 98: die Naht bei 100 ist kein Ziel
	s.Loslassen(fo(s, 88, 25))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(78, 10)) || len(linien) != 0 {
		t.Errorf("Naht: Bereich %v, Linien %v, Soll Min.X 78 ohne Linien", b, linien)
	}
	// Senkrecht dasselbe mit einem Zeilenstreifen 50–80: Höhe 70, Mitte bei 35 (nicht 50).
	s, a = einrastAnsicht(t, image.Rectangle{}, []document.Streifen{{Zeilen: true, Von: 50, Bis: 80}},
		document.R(document.Pt(10, 4), document.Pt(30, 14)))
	linien = ziehVon(s, document.Pt(20, 9), document.Pt(20, 34)) // Mitte von A (y 9) bei 34, Ziel 35
	s.Loslassen(fo(s, 20, 34))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(10, 30)) || !hat(linien, Hilfslinie{Waagrecht: true, Lage: 35}) {
		t.Errorf("Mitte senkrecht: Bereich %v, Linien %v, Soll Min.Y 30 und y=35", b, linien)
	}
}

// Der Bildrand ist der Rand des Zuschnitts, nicht der des Originals.
func TestEinrastenAmRandDesZuschnitts(t *testing.T) {
	s, a := einrastAnsicht(t, image.Rect(20, 10, 200, 100), nil, document.R(document.Pt(40, 20), document.Pt(60, 40)))
	linien := ziehVon(s, document.Pt(50, 30), document.Pt(32, 30)) // linke Kante bei 22, Rand des Zuschnitts 20
	s.Loslassen(fo(s, 32, 30))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(20, 20)) || len(linien) != 1 || !hat(linien, Hilfslinie{Lage: 20}) {
		t.Errorf("Bereich %v, Linien %v, Soll Min (20,20) und nur x=20", b, linien)
	}
}

// Eine Kante eines Ziels, die außerhalb des Zuschnitts oder in einem Streifen liegt, ist kein Ziel: Sie ist nicht zu
// sehen, und das Ende eines neuen Objekts läge sonst im Verborgenen (F-150, F-157).
func TestEinrastenNichtAnVerborgenerLinie(t *testing.T) {
	// Zuschnitt bis x = 160; Z reicht mit der rechten Kante 167 hinaus. A (20 breit) mit der rechten Kante bei 166:
	// Ziel ist der Bildrand 160 für die Mitte (156), nicht 167.
	s, a := einrastAnsicht(t, image.Rect(0, 0, 160, 100), nil, kastenA)
	kastenAm(t, s, document.R(document.Pt(100, 60), document.Pt(167, 90)))
	linien := ziehVon(s, document.Pt(20, 25), document.Pt(156, 25))
	s.Loslassen(fo(s, 156, 25))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(150, 10)) || !hat(linien, Hilfslinie{Lage: 160}) ||
		hat(linien, Hilfslinie{Lage: 167}) {
		t.Errorf("Zuschnitt: Bereich %v, Linien %v, Soll Min.X 150 und nur x=160", b, linien)
	}
	// Streifen 100–160; Z reicht mit der rechten Kante 103 hinein. Ein neuer Kasten mit dem Ende bei 98 bleibt dort.
	s, _ = einrastAnsicht(t, image.Rectangle{}, []document.Streifen{{Zeilen: false, Von: 100, Bis: 160}},
		document.R(document.Pt(60, 60), document.Pt(103, 90)))
	s.SetzeWerkzeug(WerkzeugKasten)
	linien = ziehVon(s, document.Pt(20, 10), document.Pt(98, 33))
	s.Loslassen(fo(s, 98, 33))
	k, ok := letztes(t, s).(document.Kasten)
	if !ok || !gleich(k.Bereich.Max, document.Pt(98, 33)) || len(linien) != 0 {
		t.Errorf("Streifen: neuer Kasten %v, Linien %v, Soll Ende (98,33) ohne Linien", letztes(t, s), linien)
	}
	// Dasselbe senkrecht: Zeilenstreifen 50–80, Z reicht mit der unteren Kante 53 hinein; das Ende bei y = 48 bleibt.
	s, _ = einrastAnsicht(t, image.Rectangle{}, []document.Streifen{{Zeilen: true, Von: 50, Bis: 80}},
		document.R(document.Pt(60, 20), document.Pt(100, 53)))
	s.SetzeWerkzeug(WerkzeugKasten)
	linien = ziehVon(s, document.Pt(20, 10), document.Pt(33, 48))
	s.Loslassen(fo(s, 33, 48))
	if k, ok := letztes(t, s).(document.Kasten); !ok || !gleich(k.Bereich.Max, document.Pt(33, 48)) || len(linien) != 0 {
		t.Errorf("Zeilenstreifen: neuer Kasten %v, Linien %v, Soll Ende (33,48) ohne Linien", letztes(t, s), linien)
	}
	// Die Kanten des Streifens selbst (100 und 160) sind zu sehen und bleiben Ziele: Z endet jetzt bei 100.
	s, _ = einrastAnsicht(t, image.Rectangle{}, []document.Streifen{{Zeilen: false, Von: 100, Bis: 160}},
		document.R(document.Pt(60, 60), document.Pt(100, 90)))
	s.SetzeWerkzeug(WerkzeugKasten)
	linien = ziehVon(s, document.Pt(20, 10), document.Pt(98, 33))
	s.Loslassen(fo(s, 98, 33))
	if k, ok := letztes(t, s).(document.Kasten); !ok || !gleich(k.Bereich.Max, document.Pt(100, 33)) ||
		!hat(linien, Hilfslinie{Lage: 100}) {
		t.Errorf("Kante am Streifen: neuer Kasten %v, Linien %v, Soll Ende x=100 mit Linie", letztes(t, s), linien)
	}
}

// Ein gesperrtes Objekt ist zu sehen und bleibt Ziel (Nachtrag 12 Punkt 1).
func TestEinrastenAnGesperrtemZiel(t *testing.T) {
	s, a, z := einrastLauf(t)
	if !s.SchalteGesperrt(z) {
		t.Fatal("Sperren")
	}
	b, linien := ziehA(t, s, a, 99, 0)
	if !gleich(b.Max, document.Pt(130, 40)) || !hat(linien, Hilfslinie{Lage: 130}) {
		t.Errorf("gesperrtes Ziel: Bereich %v, Linien %v, Soll Max.X 130 mit Linie", b, linien)
	}
}

// Wird eine Ecke über die Gegenseite gezogen, ordnet sich der Kasten neu; die Linie bleibt, denn die Kante liegt an.
func TestEinrastenGriffUeberDieGegenseite(t *testing.T) {
	s, a, _ := einrastLauf(t)
	s.Auswahl = a
	linien := ziehVon(s, document.Pt(10, 10), document.Pt(129, 12)) // Griff NW: x 129, Ziel 130; y 12 ohne Ziel
	s.Loslassen(fo(s, 129, 12))
	b := bereich(t, s, a)
	if !gleich(b.Min, document.Pt(30, 12)) || !gleich(b.Max, document.Pt(130, 40)) {
		t.Fatalf("Bereich %v–%v, Soll (30,12)–(130,40)", b.Min, b.Max)
	}
	if len(linien) != 1 || linien[0] != (Hilfslinie{Lage: 130}) {
		t.Errorf("Linien %v, Soll genau x=130", linien)
	}
	// Kantengriff S über die obere Kante hinaus: nur die waagrechte Linie, auch wenn der Zeiger in x an einem Ziel liegt.
	s, a, _ = einrastLauf(t)
	s.Auswahl = a
	linien = ziehVon(s, document.Pt(20, 40), document.Pt(129, 1)) // y 1, Ziel Bildrand 0; x 129 gehört nicht zum Griff
	s.Loslassen(fo(s, 129, 1))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(10, 0)) || !gleich(b.Max, document.Pt(30, 10)) {
		t.Fatalf("Bereich %v–%v, Soll (10,0)–(30,10)", b.Min, b.Max)
	}
	if len(linien) != 1 || linien[0] != (Hilfslinie{Waagrecht: true, Lage: 0}) {
		t.Errorf("Linien %v, Soll genau y=0", linien)
	}
	// Kantengriff O, der Zeiger liegt in y an einer Linie (40, Oberkante von Y), auf der zufällig die Unterkante von A
	// steht: Der Griff bewegt y nicht, also gehört die Linie nicht zum Zug.
	s, a, _ = einrastLauf(t)
	kastenAm(t, s, document.R(document.Pt(150, 40), document.Pt(190, 55)))
	s.Auswahl = a
	linien = ziehVon(s, document.Pt(30, 25), document.Pt(127, 38))
	s.Loslassen(fo(s, 127, 38))
	if b := bereich(t, s, a); !gleich(b.Max, document.Pt(130, 40)) {
		t.Fatalf("Bereich %v, Soll Max (130,40)", b)
	}
	if len(linien) != 1 || linien[0] != (Hilfslinie{Lage: 130}) {
		t.Errorf("Linien %v, Soll genau x=130 (y=40 gehört nicht zum Kantengriff O)", linien)
	}
	// Dasselbe für den Kantengriff S: Der Zeiger liegt in x an der linken Kante von X (30), auf der die rechte Kante
	// von A steht.
	s, a, _ = einrastLauf(t)
	kastenAm(t, s, document.R(document.Pt(30, 60), document.Pt(50, 80)))
	s.Auswahl = a
	linien = ziehVon(s, document.Pt(20, 40), document.Pt(28, 58))
	s.Loslassen(fo(s, 28, 58))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(10, 10)) || !gleich(b.Max, document.Pt(30, 60)) {
		t.Fatalf("Bereich %v–%v, Soll (10,10)–(30,60)", b.Min, b.Max)
	}
	if len(linien) != 1 || linien[0] != (Hilfslinie{Waagrecht: true, Lage: 60}) {
		t.Errorf("Linien %v, Soll genau y=60 (x=30 gehört nicht zum Kantengriff S)", linien)
	}
}

// Esc oder ein Zeiger-Cancel während eines eingerasteten Zugs: keine Linie bleibt stehen, das Objekt steht wie vorher,
// der Verlauf ist unberührt; der nächste Druck zeigt vor der ersten Bewegung nichts vom alten Zug.
func TestEinrastenAbbruchHinterlaesstNichts(t *testing.T) {
	for _, abbruch := range []struct {
		name string
		tu   func(s *Sitzung)
	}{{"Esc", (*Sitzung).Esc}, {"Cancel", (*Sitzung).Abbrechen}} {
		s, a, _ := einrastLauf(t)
		vor := s.D.Stand()
		if linien := ziehVon(s, document.Pt(20, 25), document.Pt(119, 25)); !hat(linien, Hilfslinie{Lage: 130}) {
			t.Fatalf("%s: Linien %v, Soll x=130 vor dem Abbruch", abbruch.name, linien)
		}
		abbruch.tu(s)
		if l := s.Hilfslinien(); l != nil {
			t.Errorf("%s: Linien nach dem Abbruch: %v", abbruch.name, l)
		}
		if b := bereich(t, s, a); !gleich(b.Min, document.Pt(10, 10)) || s.D.Stand() != vor || s.Zieht() {
			t.Errorf("%s: Bereich %v, Stand %d (vorher %d), zieht %v", abbruch.name, b, s.D.Stand(), vor, s.Zieht())
		}
		if s.Rueckgaengig() {
			t.Errorf("%s: der abgebrochene Zug hat einen Schritt hinterlassen", abbruch.name)
		}
		s.Druecken(fo(s, 20, 25), TastePrimaer)
		if l := s.Hilfslinien(); l != nil {
			t.Errorf("%s: Linien des alten Zugs nach neuem Druck: %v", abbruch.name, l)
		}
		s.Abbrechen()
	}
}

// Alt ohne Zeigerbewegung (Tastenereignis): Der laufende Zug wird sofort neu gerechnet, und es bleibt ein Schritt.
func TestAltTasteWirktOhneZeigerbewegung(t *testing.T) {
	s, a, _ := einrastLauf(t)
	s.AltTaste(true) // ohne Zug: nur der Zustand
	if !s.AltGedrueckt() || !gleich(bereich(t, s, a).Min, document.Pt(10, 10)) || s.Zieht() {
		t.Fatalf("Alt ohne Zug: Alt %v, Bereich %v, zieht %v", s.AltGedrueckt(), bereich(t, s, a), s.Zieht())
	}
	s.AltTaste(false)
	vor := s.D.Stand()
	if linien := ziehVon(s, document.Pt(20, 25), document.Pt(119, 25)); !hat(linien, Hilfslinie{Lage: 130}) ||
		!gleich(bereich(t, s, a).Min, document.Pt(110, 10)) {
		t.Fatalf("vor Alt: Bereich %v, Linien %v, Soll eingerastet bei 110", bereich(t, s, a), linien)
	}
	s.AltTaste(true)
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(109, 10)) || len(s.Hilfslinien()) != 0 {
		t.Errorf("Alt gedrückt: Bereich %v, Linien %v, Soll 109 ohne Linien", b, s.Hilfslinien())
	}
	s.AltTaste(false)
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(110, 10)) || !hat(s.Hilfslinien(), Hilfslinie{Lage: 130}) {
		t.Errorf("Alt losgelassen: Bereich %v, Linien %v, Soll 110 mit Linie", b, s.Hilfslinien())
	}
	s.AltTaste(true)
	s.Loslassen(fo(s, 119, 25))
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(109, 10)) {
		t.Errorf("mit Alt losgelassen: Bereich %v, Soll 109", b)
	}
	if !s.Rueckgaengig() || s.D.Stand() != vor || !gleich(bereich(t, s, a).Min, document.Pt(10, 10)) || s.Rueckgaengig() {
		t.Errorf("der Zug mit Alt-Wechseln muss genau ein Schritt sein (Bereich %v)", bereich(t, s, a))
	}
	// Vor der ersten Bewegung (unter der Ziehschwelle) bewegt die Taste nichts.
	s.AltTaste(false)
	s.Druecken(fo(s, 20, 25), TastePrimaer)
	s.AltTaste(true)
	if b := bereich(t, s, a); !gleich(b.Min, document.Pt(10, 10)) {
		t.Errorf("Alt vor der ersten Bewegung: Bereich %v, Soll unverändert", b)
	}
	s.Abbrechen()
}
