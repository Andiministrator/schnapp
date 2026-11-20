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
	got := Zackenband(Naht{Zeilen: true, Lage: 10}, 20, 1)
	want := []Punkt{{0, 5}, {8, 13}, {16, 5}, {24, 13}, {24, 15}, {16, 7}, {8, 15}, {0, 7}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("waagrecht:\n got %v\nwant %v", got, want)
	}
	got = Zackenband(Naht{Zeilen: false, Lage: 10}, 20, 1)
	for i := range want {
		want[i] = Punkt{want[i].Y, want[i].X}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("senkrecht:\n got %v\nwant %v", got, want)
	}
	if n := len(Zackenband(Naht{Lage: 3}, 0, 1)); n != 4 {
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

// Reichweite: obere Schranke dessen, was gezeichnet wird (Text, Nummer, Pfeilspitze); Kasten wie Grenzen.
func TestReichweite(t *testing.T) {
	rot := Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}
	for name, fall := range map[string]struct {
		o    Objekt
		soll Rechteck
	}{
		// 7 Zeichen (Tabulator zählt 4): rechts (1,1·7 + 0,25)·10; oben 0,2·10, unten Grundlinie + 0,3·10.
		"text": {Text{Ort: Pt(0, 0), Inhalt: "ab\tc", Schriftgrad: 10}, Rechteck{Pt(-2.5, -2), Pt(79.5, 13)}},
		// zwei Zeilen: letzte Grundlinie 10·(1 + 1,3)
		"text2": {Text{Ort: Pt(0, 0), Inhalt: "a\nb", Schriftgrad: 10}, Rechteck{Pt(-2.5, -2), Pt(13.5, 26)}},
		// Wert 77, Radius 10: Grad 12, halbe Breite 12·(0,8 + 1) = 21,6
		"nummer": {Nummer{Mitte: Pt(50, 50), Wert: 77, Radius: 10}, Rechteck{Pt(28.4, 38), Pt(71.6, 62)}},
		// Spitze max(8, 4) um B
		"pfeil": {GeraderPfeil(Pt(0, 50), Pt(50, 50), rot), Rechteck{Pt(-0.5, 42), Pt(58, 58)}},
		"kasten": {Kasten{Bereich: R(Pt(10, 10), Pt(20, 20)), Strich: Strich{Breite: 2}},
			Rechteck{Pt(9, 9), Pt(21, 21)}},
	} {
		got := Reichweite(fall.o)
		if d := got.Min.Minus(fall.soll.Min).Laenge() + got.Max.Minus(fall.soll.Max).Laenge(); d > 1e-9 {
			t.Errorf("%s: Reichweite %v, Soll %v", name, got, fall.soll)
		}
		g := fall.o.Grenzen()
		if got.Vereint(g) != got {
			t.Errorf("%s: Reichweite %v enthält Grenzen %v nicht", name, got, g)
		}
	}
}

// Ein Text, dessen geschätzte Grenzen ganz links vom Zuschnitt enden, dessen Glyphen aber hineinreichen können, gilt
// als sichtbar; ein Text ganz in einem Zeilenstreifen nicht.
func TestObjektSichtbar(t *testing.T) {
	d := leeresDokument(t, 100, 60)
	if err := d.SetzeZuschnitt(image.Rect(20, 0, 100, 60)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 30, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	// 2 Zeichen à 0,6·10 = 12 geschätzt (bis 17), Reichweite (1,1·2 + 0,25)·10 = 24,5 (bis 29,5).
	knapp := Text{Ort: Pt(5, 5), Inhalt: "WW", Schriftgrad: 10}
	if d.Sichtbar(knapp.Grenzen()) || !d.ObjektSichtbar(knapp) {
		t.Error("Text am Zuschnitt: Grenzen sollen außerhalb, Reichweite innerhalb liegen")
	}
	if d.ObjektSichtbar(Text{Ort: Pt(40, 33), Inhalt: "x", Schriftgrad: 10}) { // y 31 bis 46
		t.Error("Text ganz im Zeilenstreifen sichtbar")
	}
}
