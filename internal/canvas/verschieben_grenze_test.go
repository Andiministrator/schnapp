// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ---- F-160 (1): Verschieben und Griffe lassen ein Objekt nie ganz aus dem Bild ----

func kastenAm(t *testing.T, s *Sitzung, r document.Rechteck) int {
	t.Helper()
	id, err := s.D.Hinzufuegen(document.Kasten{Bereich: r, Fuellung: rot.Farbe})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.D.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	s.Gespeichert()
	return id
}

func TestVerschiebenBleibtImBild(t *testing.T) {
	for _, fall := range []struct {
		name   string
		dx, dy float64
		soll   document.Punkt // linke obere Ecke des Kastens (Start 60,20; 40×30) nach dem Ziehen
	}{
		{"weit nach rechts", 1000, 0, document.Pt(192, 20)},      // 200 − 8
		{"weit nach links", -1000, 0, document.Pt(8-40, 20)},     // rechte Kante bei 8
		{"weit nach unten", 0, 1000, document.Pt(60, 92)},        // 100 − 8
		{"weit nach oben", 0, -1000, document.Pt(60, 8-30)},      // untere Kante bei 8
		{"diagonal: je Achse", 1000, 1000, document.Pt(192, 92)}, // gleitet am Rand entlang
		{"im Bild: unbegrenzt", 30, 20, document.Pt(90, 40)},
	} {
		s, _, _ := sitzung(t)
		s.SchalteEinrasten() // diese Probe misst die Geometrie ohne Einrasten (eigene Tests: einrasten_test.go)
		id := kastenAm(t, s, document.R(document.Pt(60, 20), document.Pt(100, 50)))
		s.Auswahl = id
		s.Druecken(f(80, 35), TastePrimaer)
		s.Ziehen(f(80+fall.dx/2, 35+fall.dy/2))
		s.Loslassen(f(80+fall.dx, 35+fall.dy))
		o, _ := s.D.Objekt(id)
		if k := o.(document.Kasten); !gleich(k.Bereich.Min, fall.soll) || !gleich(k.Bereich.Max, fall.soll.Plus(document.Pt(40, 30))) {
			t.Errorf("%s: Bereich %v, Soll Ecke %v", fall.name, k.Bereich, fall.soll)
		}
	}
}

// Die Grenze zählt in Fensterpixeln: Bei Zoom 2 genügen 4 Pixel der Ansicht.
func TestVerschiebenGrenzeInFensterpixeln(t *testing.T) {
	s, _, _ := sitzung(t)
	id := kastenAm(t, s, document.R(document.Pt(60, 20), document.Pt(100, 50)))
	s.Abb = Abbildung{Zoom: 2, Versatz: document.Pt(0, 0)}
	s.Auswahl = id
	p := s.Abb.ZuFenster(document.Pt(80, 35))
	s.Druecken(p, TastePrimaer)
	s.Ziehen(p.Plus(document.Pt(1000, 0)))
	s.Loslassen(p.Plus(document.Pt(2000, 0)))
	o, _ := s.D.Objekt(id)
	if k := o.(document.Kasten); !gleich(k.Bereich.Min, document.Pt(196, 20)) { // 200 − 8/2
		t.Errorf("Bereich %v, Soll linke Kante 196", k.Bereich)
	}
}

func TestGriffBleibtImBild(t *testing.T) {
	s, _, _ := sitzung(t)
	id := kastenAm(t, s, document.R(document.Pt(190, 10), document.Pt(230, 50))) // ragt 10 px ins Bild
	s.Auswahl = id
	s.Druecken(f(190, 10), TastePrimaer) // Griff links oben
	s.Ziehen(f(600, 10))
	s.Loslassen(f(1000, 10)) // über die rechte Kante hinaus: sonst klappte der Kasten ganz aus dem Bild
	o, _ := s.D.Objekt(id)
	k := o.(document.Kasten)
	if !gleich(k.Bereich.Min, document.Pt(192, 10)) || !gleich(k.Bereich.Max, document.Pt(230, 50)) {
		t.Errorf("Bereich %v, Soll (192,10)–(230,50): 8 Pixel bleiben im Bild", k.Bereich)
	}
}
