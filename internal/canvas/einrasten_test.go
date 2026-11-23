// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Einrasten und Hilfslinien (ADR-0021 Nachtrag 12, Karte #196). Bild 200×100; Kasten A (10,10)–(30,40) wird gezogen,
// Ziel Z (130,60)–(170,90). Ziele in x: 0, 100, 200 (Bild) und 130, 150, 170 (Z); in y: 0, 50, 100 und 60, 75, 90.
// A hat die Kanten und Mitte x 10/20/30 und y 10/25/40: Ohne Verschiebung liegt keine in Fangweite.
func einrastLauf(t *testing.T) (s *Sitzung, a, z int) {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	s = NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	a = kastenAm(t, s, document.R(document.Pt(10, 10), document.Pt(30, 40)))
	z = kastenAm(t, s, document.R(document.Pt(130, 60), document.Pt(170, 90)))
	return s, a, z
}

// ziehA zieht A um (dx, 0) im Original, hält vor dem Loslassen an und liefert die Linien dabei.
func ziehA(t *testing.T, s *Sitzung, a int, dx, dy float64) (document.Rechteck, []Hilfslinie) {
	t.Helper()
	von := s.Abb.ZuFenster(document.Pt(20, 25))
	s.Druecken(von, TastePrimaer)
	if s.Auswahl != a {
		t.Fatalf("Auswahl %d, Soll %d", s.Auswahl, a)
	}
	s.Ziehen(von.Plus(document.Pt(dx, dy).Mal(s.Abb.Zoom / 2)))
	s.Ziehen(von.Plus(document.Pt(dx, dy).Mal(s.Abb.Zoom)))
	linien := s.Hilfslinien()
	s.Loslassen(von.Plus(document.Pt(dx, dy).Mal(s.Abb.Zoom)))
	o, _ := s.D.Objekt(a)
	return o.(document.Kasten).Bereich, linien
}

func hat(linien []Hilfslinie, l Hilfslinie) bool { return slices.Contains(linien, l) }

func TestEinrastenAnKante(t *testing.T) {
	s, a, _ := einrastLauf(t)
	// Rechte Kante bei 129, Ziel 130: rastet ein, Versatz 100.
	b, linien := ziehA(t, s, a, 99, 0)
	if !gleich(b.Min, document.Pt(110, 10)) || !gleich(b.Max, document.Pt(130, 40)) {
		t.Errorf("Bereich %v–%v, Soll (110,10)–(130,40)", b.Min, b.Max)
	}
	if !hat(linien, Hilfslinie{Lage: 130}) || len(linien) != 1 {
		t.Errorf("Linien %v, Soll genau x=130", linien)
	}
}

func TestEinrastenAnMitte(t *testing.T) {
	s, a, _ := einrastLauf(t)
	// Mitte bei 149, Ziel (Mitte von Z) 150.
	b, linien := ziehA(t, s, a, 129, 0)
	if !gleich(b.Min, document.Pt(140, 10)) {
		t.Errorf("Bereich %v, Soll Ecke (140,10)", b)
	}
	if !hat(linien, Hilfslinie{Lage: 150}) {
		t.Errorf("Linien %v, Soll x=150", linien)
	}
}

func TestEinrastenAmBildrand(t *testing.T) {
	s, a, _ := einrastLauf(t)
	b, linien := ziehA(t, s, a, -8, 0) // linke Kante bei 2, Ziel 0
	if !gleich(b.Min, document.Pt(0, 10)) || !hat(linien, Hilfslinie{Lage: 0}) {
		t.Errorf("links: Bereich %v, Linien %v", b, linien)
	}
	s, a, _ = einrastLauf(t)
	b, linien = ziehA(t, s, a, 168, 0) // rechte Kante bei 198, Ziel 200
	if !gleich(b.Max, document.Pt(200, 40)) || !hat(linien, Hilfslinie{Lage: 200}) {
		t.Errorf("rechts: Bereich %v, Linien %v", b, linien)
	}
	s, a, _ = einrastLauf(t)
	b, linien = ziehA(t, s, a, 0, 12) // untere Kante bei 52, Ziel (Mitte des Bilds) 50
	if !gleich(b.Max, document.Pt(30, 50)) || !hat(linien, Hilfslinie{Waagrecht: true, Lage: 50}) {
		t.Errorf("Mitte des Bilds: Bereich %v, Linien %v", b, linien)
	}
}

func TestEinrastenAusserhalbDerFangweite(t *testing.T) {
	s, a, _ := einrastLauf(t)
	b, linien := ziehA(t, s, a, 40, 0)
	if !gleich(b.Min, document.Pt(50, 10)) || len(linien) != 0 {
		t.Errorf("Bereich %v, Linien %v: nichts darf einrasten", b, linien)
	}
	// Obere Kante bei 7 (Ziel 0): 7 Pixel sind außerhalb der Fangweite 6, die Mitte (22) und die untere (37) auch.
	s, a, _ = einrastLauf(t)
	b, linien = ziehA(t, s, a, 0, -3)
	if !gleich(b.Min, document.Pt(10, 7)) || len(linien) != 0 {
		t.Errorf("7 Pixel: Bereich %v, Linien %v, Soll (10,7) ohne Linien", b, linien)
	}
	// Bei 6 Pixeln rastet es ein (die Fangweite selbst zählt mit).
	s, a, _ = einrastLauf(t)
	b, linien = ziehA(t, s, a, 0, -4)
	if !gleich(b.Min, document.Pt(10, 0)) || !hat(linien, Hilfslinie{Waagrecht: true, Lage: 0}) {
		t.Errorf("6 Pixel: Bereich %v, Linien %v, Soll Min.Y 0 und y=0", b, linien)
	}
}

func TestEinrastenAltSchaltetAb(t *testing.T) {
	s, a, _ := einrastLauf(t)
	s.SetzeAlt(true)
	b, linien := ziehA(t, s, a, 99, 0)
	if !gleich(b.Min, document.Pt(109, 10)) || len(linien) != 0 {
		t.Errorf("mit Alt: Bereich %v, Linien %v, Soll (109,10) ohne Linien", b, linien)
	}
	s, a, _ = einrastLauf(t)
	s.SetzeAlt(true)
	s.SetzeAlt(false) // wieder losgelassen: es rastet ein
	b, linien = ziehA(t, s, a, 99, 0)
	if !gleich(b.Min, document.Pt(110, 10)) || len(linien) == 0 {
		t.Errorf("ohne Alt: Bereich %v, Linien %v", b, linien)
	}
}

func TestEinrastenSchalterAus(t *testing.T) {
	s, a, _ := einrastLauf(t)
	if !s.EinrastenAn() {
		t.Fatal("Voreinstellung muss an sein")
	}
	s.SchalteEinrasten()
	if s.EinrastenAn() {
		t.Fatal("Schalter bleibt an")
	}
	b, linien := ziehA(t, s, a, 99, 0)
	if !gleich(b.Min, document.Pt(109, 10)) || len(linien) != 0 {
		t.Errorf("Einrasten aus: Bereich %v, Linien %v", b, linien)
	}
}

func TestEinrastenBeiZoomZwei(t *testing.T) {
	// Die Fangweite gilt in Fensterpixeln: Bei Zoom 2 sind 6 Fensterpixel nur 3 Pixel des Originals.
	s, a, _ := einrastLauf(t)
	b, _ := ziehA(t, s, a, 96, 0) // Kanten 106/116/126: 126 liegt 4 von 130 (zoom 1: rastet ein)
	if !gleich(b.Max, document.Pt(130, 40)) {
		t.Fatalf("Zoom 1: Bereich %v, Soll Max (130,40)", b)
	}
	s, a, _ = einrastLauf(t)
	s.SetzeZoom(200)
	b, linien := ziehA(t, s, a, 96, 0) // 4 Pixel Original = 8 Fensterpixel: rastet nicht ein
	if !gleich(b.Min, document.Pt(106, 10)) || len(linien) != 0 {
		t.Errorf("Zoom 2: Bereich %v, Linien %v, Soll (106,10) ohne Linien", b, linien)
	}
	s, a, _ = einrastLauf(t)
	s.SetzeZoom(200)
	b, linien = ziehA(t, s, a, 99, 0) // 1 Pixel Original = 2 Fensterpixel: rastet ein
	if !gleich(b.Min, document.Pt(110, 10)) || !hat(linien, Hilfslinie{Lage: 130}) {
		t.Errorf("Zoom 2 nah: Bereich %v, Linien %v", b, linien)
	}
}

func TestEinrastenIgnoriertAusgeblendetes(t *testing.T) {
	s, a, z := einrastLauf(t)
	if !s.SchalteAusgeblendet(z) {
		t.Fatal("Ausblenden")
	}
	b, linien := ziehA(t, s, a, 99, 0)
	if !gleich(b.Min, document.Pt(109, 10)) || len(linien) != 0 {
		t.Errorf("Ziel ausgeblendet: Bereich %v, Linien %v, nichts darf einrasten", b, linien)
	}
}

func TestEinrastenIstEinRueckgaengigSchritt(t *testing.T) {
	s, a, _ := einrastLauf(t)
	vor := s.D.Stand()
	ziehA(t, s, a, 99, 0)
	if !s.Rueckgaengig() {
		t.Fatal("kein Schritt")
	}
	o, _ := s.D.Objekt(a)
	if b := o.(document.Kasten).Bereich; !gleich(b.Min, document.Pt(10, 10)) {
		t.Errorf("nach Rückgängig %v, Soll Ausgang", b)
	}
	if s.D.Stand() != vor {
		t.Errorf("ein Rückgängig muss zum Stand vor dem Zug führen")
	}
	if s.Hilfslinien() != nil {
		t.Errorf("Linien nach dem Loslassen: %v", s.Hilfslinien())
	}
}

func TestEinrastenAmGriff(t *testing.T) {
	s, a, _ := einrastLauf(t)
	s.Auswahl = a
	s.Druecken(s.Abb.ZuFenster(document.Pt(30, 25)), TastePrimaer) // Griff O
	s.Ziehen(s.Abb.ZuFenster(document.Pt(80, 49)))
	s.Ziehen(s.Abb.ZuFenster(document.Pt(127, 49))) // x: Ziel 130; y: Mitte des Bilds 50, doch O bewegt nur x
	linien := s.Hilfslinien()
	s.Loslassen(s.Abb.ZuFenster(document.Pt(127, 49)))
	o, _ := s.D.Objekt(a)
	if b := o.(document.Kasten).Bereich; !gleich(b.Max, document.Pt(130, 40)) {
		t.Errorf("Bereich %v, Soll Max.X 130", b)
	}
	if len(linien) != 1 || linien[0] != (Hilfslinie{Lage: 130}) {
		t.Errorf("Linien %v, Soll nur x=130 (die y-Linie gehört nicht zum Kantengriff)", linien)
	}
}

func TestEinrastenBeimNeuenKasten(t *testing.T) {
	s, _, _ := einrastLauf(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Druecken(s.Abb.ZuFenster(document.Pt(60, 10)), TastePrimaer)
	s.Ziehen(s.Abb.ZuFenster(document.Pt(100, 30)))
	s.Ziehen(s.Abb.ZuFenster(document.Pt(127, 33))) // Ziel x=130
	linien := s.Hilfslinien()
	s.Loslassen(s.Abb.ZuFenster(document.Pt(127, 33)))
	k, ok := letztes(t, s).(document.Kasten)
	if !ok || !gleich(k.Bereich.Max, document.Pt(130, 33)) {
		t.Errorf("neuer Kasten %v", letztes(t, s))
	}
	if !hat(linien, Hilfslinie{Lage: 130}) {
		t.Errorf("Linien %v, Soll x=130", linien)
	}
	// Mit Umschalt bleibt das Quadrat genau.
	s.Umschalt = true
	s.Druecken(s.Abb.ZuFenster(document.Pt(86, 30)), TastePrimaer)
	s.Ziehen(s.Abb.ZuFenster(document.Pt(100, 45)))
	s.Ziehen(s.Abb.ZuFenster(document.Pt(127, 71)))
	if l := s.Hilfslinien(); len(l) != 0 {
		t.Errorf("mit Umschalt rastet nichts ein: %v", l)
	}
	s.Abbrechen()
}

// Wird die Verschiebung durch die Sichtbarkeitsgrenze gekürzt, rastet nichts mehr ein und es bleibt keine Linie stehen.
func TestEinrastenKuerztBegrenzung(t *testing.T) {
	s, a, _ := einrastLauf(t)
	b, linien := ziehA(t, s, a, 188, 0) // linke Kante bei 198: Ziel 200 läge außerhalb von „mindestens 8 sichtbar“
	if !gleich(b.Min, document.Pt(192, 10)) {
		t.Errorf("Bereich %v, Soll Min.X 192 (200 − MinSichtbar)", b)
	}
	if hat(linien, Hilfslinie{Lage: 200}) {
		t.Errorf("Linien %v: x=200 liegt nicht an", linien)
	}
}

func TestEinrastenBeiNeuerNummer(t *testing.T) {
	s, _, _ := einrastLauf(t)
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Druecken(s.Abb.ZuFenster(document.Pt(60, 30)), TastePrimaer)
	s.Ziehen(s.Abb.ZuFenster(document.Pt(100, 30)))
	s.Ziehen(s.Abb.ZuFenster(document.Pt(148, 30))) // Mitte x: Ziel 150 (Mitte von Z)
	linien := s.Hilfslinien()
	s.Loslassen(s.Abb.ZuFenster(document.Pt(148, 30)))
	n, ok := letztes(t, s).(document.Nummer)
	if !ok || !gleich(document.Pt(n.Mitte.X, 0), document.Pt(150, 0)) {
		t.Errorf("neue Nummer %v, Soll Mitte x=150", letztes(t, s))
	}
	if !hat(linien, Hilfslinie{Lage: 150}) {
		t.Errorf("Linien %v, Soll x=150", linien)
	}
}
