// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func testLupe() Lupe {
	return Lupe{Quelle: R(Pt(10, 10), Pt(20, 16)), Ziel: R(Pt(40, 10), Pt(60, 22)),
		Strich: Strich{Farbe: color.NRGBA{255, 0, 0, 255}, Breite: 2}}
}

// TestLupePruefe: Breite und Höhe beider Rechtecke > 0 und ≤ MaxGroesse, Koordinaten im Bereich, Strich gültig.
func TestLupePruefe(t *testing.T) {
	if err := testLupe().pruefe(); err != nil {
		t.Fatal(err)
	}
	kaputt := map[string]func(*Lupe){
		"Quelle ohne Breite": func(l *Lupe) { l.Quelle.Max.X = l.Quelle.Min.X },
		"Quelle verkehrt":    func(l *Lupe) { l.Quelle.Max.Y = l.Quelle.Min.Y - 1 },
		"Ziel ohne Höhe":     func(l *Lupe) { l.Ziel.Max.Y = l.Ziel.Min.Y },
		"Ziel zu breit":      func(l *Lupe) { l.Ziel.Max.X = l.Ziel.Min.X + MaxGroesse + 1 },
		"Quelle zu hoch":     func(l *Lupe) { l.Quelle.Max.Y = l.Quelle.Min.Y + MaxGroesse + 0.5 },
		"Koordinate NaN":     func(l *Lupe) { l.Ziel.Min.X = math.NaN() },
		"Koordinate zu groß": func(l *Lupe) { l.Quelle.Min.X, l.Quelle.Max.X = MaxKoordinate+1, MaxKoordinate+2 },
		"Strich negativ":     func(l *Lupe) { l.Strich.Breite = -1 },
		"Ebene kaputt":       func(l *Lupe) { l.Transparenz = 101 },
	}
	for name, f := range kaputt {
		l := testLupe()
		f(&l)
		if l.pruefe() == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
	// Genau an der Grenze gültig.
	l := testLupe()
	l.Ziel.Max.X = l.Ziel.Min.X + MaxGroesse
	if err := l.pruefe(); err != nil {
		t.Errorf("Ziel genau MaxGroesse: %v", err)
	}
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if _, err := d.Hinzufuegen(Lupe{Quelle: R(Pt(1, 1), Pt(1, 2)), Ziel: R(Pt(3, 3), Pt(4, 4))}); err == nil {
		t.Error("Hinzufuegen nimmt eine Lupe ohne Quellbreite")
	}
}

// TestLupeGrenzen: Grenzen umfassen beide Rechtecke mit halbem Strich; die Vergrößerung je Achse.
func TestLupeGrenzen(t *testing.T) {
	l := testLupe()
	if g, want := l.Grenzen(), R(Pt(9, 9), Pt(61, 23)); g != want {
		t.Errorf("Grenzen %v, erwartet %v", g, want)
	}
	if x, y := l.Vergroesserung(); x != 2 || y != 2 {
		t.Errorf("Vergrößerung %v, %v", x, y)
	}
	if l.Kennung() != 0 || l.mitKennung(7).Kennung() != 7 {
		t.Error("Kennung")
	}
	if MitEbene(l, Ebene{Name: "Lupe"}).Ebenenwerte().Name != "Lupe" {
		t.Error("MitEbene")
	}
}

// TestLupeQuellpixel: nächster Nachbar nach ADR-0033 Punkt 3.
func TestLupeQuellpixel(t *testing.T) {
	l := testLupe() // Quelle x 10..20, Ziel x 40..60: Zielspalten 40 und 41 zeigen 10, 58 und 59 zeigen 19
	for _, f := range []struct {
		z    float64
		want int
	}{{40.5, 10}, {41.5, 10}, {42.5, 11}, {58.5, 19}, {59.5, 19}} {
		if got := l.QuellSpalte(f.z); got != f.want {
			t.Errorf("QuellSpalte(%v) = %d, erwartet %d", f.z, got, f.want)
		}
	}
	l.Ziel = R(Pt(40, 10), Pt(70, 28)) // 3×
	if got := l.QuellZeile(10.5 + 3*4); got != 14 {
		t.Errorf("QuellZeile = %d, erwartet 14", got)
	}
	l.Quelle = R(Pt(10.5, 10), Pt(20.5, 16)) // halbe Pixel: Ziel 40.5 zeigt floor(10.5 + 0.25) = 10
	if got := l.QuellSpalte(40.5); got != 10 {
		t.Errorf("halbe Quelle: %d", got)
	}
}

// TestLupeLinie: von Kante zu Kante zwischen den Mittelpunkten, keine Linie bei Überlappung oder Berührung.
func TestLupeLinie(t *testing.T) {
	l := testLupe() // Mitten (15,13) und (50,16)
	a, b, ok := l.Linie()
	if !ok {
		t.Fatal("keine Linie")
	}
	nah := func(p, q Punkt) bool { return math.Abs(p.X-q.X) < 1e-9 && math.Abs(p.Y-q.Y) < 1e-9 }
	// Austritt aus der Quelle bei x = 20 (t = 5/35), Eintritt ins Ziel bei x = 40 (t = 25/35).
	if !nah(a, Pt(20, 13+3*5.0/35)) || !nah(b, Pt(40, 13+3*25.0/35)) {
		t.Errorf("Linie %v – %v", a, b)
	}
	// Senkrecht übereinander: Austritt unten, Eintritt oben.
	l.Ziel = R(Pt(5, 30), Pt(25, 42))
	a, b, ok = l.Linie()
	if !ok || !nah(a, Pt(15, 16)) || !nah(b, Pt(15, 30)) {
		t.Errorf("senkrecht: %v – %v %v", a, b, ok)
	}
	// Überlappung: keine Linie.
	l.Ziel = R(Pt(18, 14), Pt(38, 26))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Überlappung")
	}
	// Überlappung an einer Ecke, die Gerade zwischen den Mitten läuft außerhalb der Schnittfläche: trotzdem keine Linie.
	l.Ziel = R(Pt(19, 15), Pt(110, 17))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Überlappung an der Ecke")
	}
	// Berühren an der Kante: nichts bleibt übrig.
	l.Ziel = R(Pt(20, 10), Pt(40, 16))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Berührung")
	}
	// Ziel enthält die Quelle: keine Linie.
	l.Ziel = R(Pt(0, 0), Pt(100, 100))
	if _, _, ok := l.Linie(); ok {
		t.Error("Linie trotz Enthaltensein")
	}
}

// TestSichtbarerAusschnitt: Pixel außerhalb des Zuschnitts, in Streifen und außerhalb des Originals sind durchsichtig.
func TestSichtbarerAusschnitt(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 10, 8))
	for i := range b.Pix {
		b.Pix[i] = uint8(i%251 + 1)
	}
	d, err := Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(2, 0, 10, 8)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 3, Bis: 5}, {Von: 6, Bis: 7}}); err != nil {
		t.Fatal(err)
	}
	a := d.SichtbarerAusschnitt(image.Rect(-3, -3, 20, 20))
	if a.Rect != b.Rect {
		t.Fatalf("Grenzen %v", a.Rect)
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 10; x++ {
			sichtbar := x >= 2 && x != 6 && (y < 3 || y >= 5)
			if d.SpalteSichtbar(x) != (x >= 2 && x != 6) || d.ZeileSichtbar(y) != (y < 3 || y >= 5) {
				t.Fatalf("Sichtbar (%d,%d)", x, y)
			}
			want := color.NRGBA{}
			if sichtbar {
				want = b.NRGBAAt(x, y)
			}
			if got := a.NRGBAAt(x, y); got != want {
				t.Errorf("(%d,%d): %v, erwartet %v", x, y, got, want)
			}
		}
	}
	if !d.SichtbarerAusschnitt(image.Rect(20, 20, 30, 30)).Rect.Empty() {
		t.Error("Ausschnitt neben dem Bild nicht leer")
	}
	if d.SpalteSichtbar(-1) || d.SpalteSichtbar(10) || d.ZeileSichtbar(8) {
		t.Error("außerhalb sichtbar")
	}
}
