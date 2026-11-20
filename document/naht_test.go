// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func leeresDokument(t *testing.T, b, h int) *Dokument {
	t.Helper()
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, b, h)))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// ADR-0023 Punkt 6: Eine Naht je Stelle in der Ansicht; Streifen am Rand der Ansicht ergeben keine, anstoßende
// Streifen eine gemeinsame.
func TestNaehte(t *testing.T) {
	d := leeresDokument(t, 100, 60)
	if n := d.Naehte(); len(n) != 0 {
		t.Fatalf("ohne Streifen %v", n)
	}
	st := []Streifen{
		{Zeilen: false, Von: 0, Bis: 5},   // am linken Rand: keine Naht
		{Zeilen: false, Von: 20, Bis: 30}, // Ansicht 15
		{Zeilen: false, Von: 30, Bis: 35}, // stößt an: dieselbe Naht
		{Zeilen: false, Von: 90, Bis: 100},
		{Zeilen: true, Von: 10, Bis: 20}, // Ansicht 10
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	want := []Naht{{Zeilen: false, Lage: 15}, {Zeilen: true, Lage: 10}}
	if n := d.Naehte(); !reflect.DeepEqual(n, want) {
		t.Errorf("Naehte %v, Soll %v", n, want)
	}
	// Ein Zuschnitt, der vor dem Streifen [90, 100) endet, und einer, an dessen Rand der Zeilenstreifen beginnt.
	if err := d.SetzeZuschnitt(image.Rect(0, 10, 80, 60)); err != nil {
		t.Fatal(err)
	}
	if n := d.Naehte(); !reflect.DeepEqual(n, []Naht{{Zeilen: false, Lage: 15}}) {
		t.Errorf("mit Zuschnitt: %v", n)
	}
}

// Zackenband: Zacken abwechselnd −4 und +4 um die Naht, Zahnbreite 16, quer um halb aufgeweitet; erst die eine Kante
// vorwärts, dann die andere rückwärts.
func TestZackenband(t *testing.T) {
	got := Zackenband(Naht{Zeilen: true, Lage: 10}, 20, 1, 1)
	want := []Punkt{{0, 5}, {8, 13}, {16, 5}, {24, 13}, {24, 15}, {16, 7}, {8, 15}, {0, 7}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("waagrecht:\n got %v\nwant %v", got, want)
	}
	got = Zackenband(Naht{Zeilen: false, Lage: 10}, 20, 1, 1)
	for i := range want {
		want[i] = Punkt{want[i].Y, want[i].X}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("senkrecht:\n got %v\nwant %v", got, want)
	}
	if n := len(Zackenband(Naht{Lage: 3}, 0, 1, 1)); n != 4 {
		t.Errorf("Länge 0: %d Punkte, Soll 4 (ein Zahn)", n)
	}
}

// ADR-0023 Punkt 7: Sichtbar je Achse mit positiver Überdeckung; Rand berühren reicht nicht.
func TestSichtbar(t *testing.T) {
	d := leeresDokument(t, 100, 60)
	if err := d.SetzeZuschnitt(image.Rect(10, 0, 90, 60)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: false, Von: 40, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	for _, fall := range []struct {
		x0, x1 float64
		soll   bool
	}{
		{41, 49, false}, // ganz im Streifen
		{40, 50, false}, // Streifen genau
		{45, 55, true},  // ragt hinaus
		{35, 45, true},
		{0, 9, false}, // links vom Zuschnitt
		{0, 10, false},
		{0, 10.5, true},
		{90, 120, false}, // rechts davon
		{89.5, 120, true},
		{-1e6, 1e6, true},
	} {
		r := Rechteck{Min: Pt(fall.x0, 5), Max: Pt(fall.x1, 6)}
		if got := d.Sichtbar(r); got != fall.soll {
			t.Errorf("x %v..%v: Sichtbar %v, Soll %v", fall.x0, fall.x1, got, fall.soll)
		}
	}
	if d.Sichtbar(Rechteck{Min: Pt(20, 60), Max: Pt(30, 70)}) {
		t.Error("unter dem Bild sichtbar")
	}
}

// Reichweite: Pfeilspitze und Kasten fest; Text und Nummer aus der Schrift (die Übereinstimmung mit den gerasterten
// Pixeln prüft render, TestReichweiteWieGerastert).
func TestReichweite(t *testing.T) {
	rot := Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}
	for name, fall := range map[string]struct {
		o    Objekt
		soll Rechteck
	}{
		// Spitze max(8, 4) um B
		"pfeil": {GeraderPfeil(Pt(0, 50), Pt(50, 50), rot), Rechteck{Pt(-0.5, 42), Pt(58, 58)}},
		"kasten": {Kasten{Bereich: R(Pt(10, 10), Pt(20, 20)), Strich: Strich{Breite: 2}},
			Rechteck{Pt(9, 9), Pt(21, 21)}},
		// nur Leerraum, kein Hintergrund: keine Tinte, nur der Ort
		"leer": {Text{Ort: Pt(5, 7), Inhalt: " \t ", Schriftgrad: 10}, Rechteck{Pt(5, 7), Pt(5, 7)}},
		// Ziffern im Kreis: der Kreis
		"nummer": {Nummer{Mitte: Pt(50, 50), Wert: 7, Radius: 10}, Rechteck{Pt(40, 40), Pt(60, 60)}},
	} {
		got := Reichweite(fall.o)
		if d := got.Min.Minus(fall.soll.Min).Laenge() + got.Max.Minus(fall.soll.Max).Laenge(); d > 1e-9 {
			t.Errorf("%s: Reichweite %v, Soll %v", name, got, fall.soll)
		}
	}
	// Mit Hintergrund gehört dessen Fläche (Grenzen) dazu.
	mit := Text{Ort: Pt(0, 0), Inhalt: "ii", Schriftgrad: 10, Hintergrund: color.NRGBA{A: 255}}
	if r := Reichweite(mit); r.Vereint(mit.Grenzen()) != r {
		t.Errorf("Hintergrund: Reichweite %v ohne Grenzen %v", r, mit.Grenzen())
	}
	// Fünf Ziffern sind breiter als ein kleiner Kreis (F-181).
	n := Nummer{Mitte: Pt(50, 50), Wert: 88888, Radius: 6}
	if r := Reichweite(n); r.Min.X >= 44 || r.Max.X <= 56 {
		t.Errorf("Nummer 88888: Reichweite %v nicht breiter als der Kreis", r)
	}
}

// F-181: 30 Zeichen im Grad 20 ab x = 20 enden vor x = 400 (die alte Schätzung, 1,1 Grad je Zeichen, reichte bis 685).
func TestReichweiteTextBreite(t *testing.T) {
	txt := Text{Ort: Pt(20, 50), Inhalt: "Kennwort hunter2 nicht zeigen!", Schriftgrad: 20}
	r := Reichweite(txt)
	if r.Max.X >= 400 || r.Max.X <= 200 || r.Min.X < 18 || r.Min.X > 22 {
		t.Errorf("Reichweite %v, Soll links ≈ 20, rechts zwischen 200 und 400", r)
	}
}

// Ein Text, dessen geschätzte Grenzen ganz links vom Zuschnitt enden, dessen Glyphen aber hineinreichen, gilt als
// sichtbar; ein Text ganz in einem Zeilenstreifen nicht.
func TestObjektSichtbar(t *testing.T) {
	d := leeresDokument(t, 100, 60)
	if err := d.SetzeZuschnitt(image.Rect(20, 0, 100, 60)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 30, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	// 2 Zeichen à 0,6·10 = 12 geschätzt (bis 17); „WW“ in Go Regular ist breiter und reicht über x = 20.
	knapp := Text{Ort: Pt(5, 5), Inhalt: "WW", Schriftgrad: 10}
	if d.Sichtbar(knapp.Grenzen()) || !d.ObjektSichtbar(knapp) {
		t.Error("Text am Zuschnitt: Grenzen sollen außerhalb, Reichweite innerhalb liegen")
	}
	if d.ObjektSichtbar(Text{Ort: Pt(40, 33), Inhalt: "x", Schriftgrad: 10}) { // y 31 bis 46
		t.Error("Text ganz im Zeilenstreifen sichtbar")
	}
}

// F-173: Der Faktor der Zackenmaße ist max(1, kurze Seite / 1080), ganzzahlig geteilt.
func TestZackenFaktor(t *testing.T) {
	for _, c := range []struct {
		g    image.Point
		want int
	}{
		{image.Pt(1920, 1080), 1}, {image.Pt(3840, 2160), 2}, {image.Pt(1000, 5000), 1}, {image.Pt(7680, 4320), 4},
		{image.Pt(100, 60), 1}, {image.Pt(5000, 1079), 1}, {image.Pt(2159, 5000), 1}, {image.Pt(2160, 5000), 2},
	} {
		if got := ZackenFaktor(c.g); got != c.want {
			t.Errorf("ZackenFaktor(%v) = %d, Soll %d", c.g, got, c.want)
		}
	}
	d := leeresDokument(t, 3840, 2160)
	if f := d.ZackenFaktor(); f != 2 {
		t.Errorf("Dokument 3840×2160: %d, Soll 2", f)
	}
	// Stauchen und Zuschnitt ändern den Faktor nicht: Er gilt für das Original (ADR-0023 Nachtrag 2, Echtlauf #92 —
	// nach der Ansicht fiel eine gestauchte 4K-Aufnahme auf Faktor 1).
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 800, Bis: 1200}}); err != nil {
		t.Fatal(err)
	}
	if f := d.ZackenFaktor(); f != 2 {
		t.Errorf("4K nach Stauchen auf Höhe 1760: %d, Soll 2", f)
	}
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 3840, 1000)); err != nil {
		t.Fatal(err)
	}
	if f := d.ZackenFaktor(); f != 2 {
		t.Errorf("nach Zuschnitt auf Höhe 1000: %d, Soll 2", f)
	}
}

// F-173: Bei Faktor 2 sind Zackenhöhe, Zahnbreite und Bandbreite doppelt so groß (vgl. TestZackenband bei Faktor 1).
func TestZackenbandFaktor(t *testing.T) {
	got := Zackenband(Naht{Zeilen: true, Lage: 10}, 40, 2, 1)
	want := []Punkt{{0, 0}, {16, 16}, {32, 0}, {48, 16}, {48, 20}, {32, 4}, {16, 20}, {0, 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Faktor 2:\n got %v\nwant %v", got, want)
	}
}

// F-226 (ADR-0023 Nachtrag 4): Der Faktor nach dem Original wird durch die kurze Seite der Ansicht gedeckelt: Die
// Bandhöhe (2·ZackenBandHalb·Faktor) bleibt höchstens ein Viertel der kurzen Ansichtsseite (bei Faktor 1 ist das die
// Untergrenze).
func TestZackenFaktorGedeckeltDurchAnsicht(t *testing.T) {
	d := leeresDokument(t, 7680, 4320)
	if f := d.ZackenFaktor(); f != 4 {
		t.Fatalf("ganze 8K-Aufnahme: Faktor %d, Soll 4", f)
	}
	// Das Beispiel des Befunds: auf 300×80 zugeschnitten. Vorher ±28 Pixel Band auf 80 Pixeln Höhe.
	if err := d.SetzeZuschnitt(image.Rect(1000, 1000, 1300, 1080)); err != nil {
		t.Fatal(err)
	}
	if f := d.ZackenFaktor(); f != 1 {
		t.Errorf("auf 300×80: Faktor %d, Soll 1", f)
	}
	for _, c := range []struct{ kurz, soll int }{{112, 2}, {111, 1}, {224, 4}, {223, 3}, {1000, 4}} {
		if err := d.SetzeZuschnitt(image.Rect(0, 0, 3000, c.kurz)); err != nil {
			t.Fatal(err)
		}
		f := d.ZackenFaktor()
		if f != c.soll {
			t.Errorf("kurze Seite %d: Faktor %d, Soll %d", c.kurz, f, c.soll)
		}
		if band := 2 * int(ZackenBandHalb) * f; f > 1 && band > c.kurz/4 {
			t.Errorf("kurze Seite %d: Band %d höher als ein Viertel", c.kurz, band)
		}
	}
}

func TestOriginalGroesseOhneKopie(t *testing.T) {
	d := leeresDokument(t, 40, 30)
	if g := d.OriginalGroesse(); g != image.Pt(40, 30) {
		t.Errorf("Größe %v", g)
	}
	if d.OriginalBild() != d.OriginalBild() {
		t.Error("OriginalBild kopiert")
	}
}
