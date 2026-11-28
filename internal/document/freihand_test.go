// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// ADR-0049 Punkte 1 und 3: Freihand-Strich als Wert, Prüfung der Punkte, Vereinfachen, Glätten, Ausdünnen.

func TestFreihandWertUndGleich(t *testing.T) {
	pts := []Punkt{Pt(0, 0), Pt(10, 5), Pt(20, 0)}
	a := NeuerFreihand(pts, Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 4})
	pts[1] = Pt(99, 99) // eigene Kopie
	if a.Punkte()[1] != Pt(10, 5) {
		t.Fatalf("Punkte geteilt: %v", a.Punkte())
	}
	b := a.MitPunkten(a.Punkte())
	if !Gleich(a, b) || a.Zug == b.Zug {
		t.Error("gleiche Punkte, anderer Zeiger: soll gleich sein")
	}
	c := a.Verschoben(Pt(1, 0))
	if Gleich(a, c) || a.Punkte()[0] != Pt(0, 0) || c.Punkte()[0] != Pt(1, 0) {
		t.Errorf("Verschoben: %v / %v", a.Punkte(), c.Punkte())
	}
	if g := a.Grenzen(); g != R(Pt(-2, -2), Pt(22, 7)) {
		t.Errorf("Grenzen %v", g)
	}
	if Gleich(a, Pfeil{}) || Gleich(Pfeil{}, a) {
		t.Error("Freihand gleich Pfeil")
	}
}

func TestFreihandPruefung(t *testing.T) {
	st := Strich{Farbe: color.NRGBA{A: 255}, Breite: 2}
	if NeuerFreihand([]Punkt{Pt(1, 1)}, st).pruefe() == nil {
		t.Error("ein Punkt angenommen")
	}
	viele := make([]Punkt, MaxFreihandPunkte+1)
	if NeuerFreihand(viele, st).pruefe() == nil {
		t.Error("zu viele Punkte angenommen")
	}
	if NeuerFreihand([]Punkt{Pt(0, 0), Pt(math.NaN(), 1)}, st).pruefe() == nil {
		t.Error("NaN angenommen")
	}
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 50, 50)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(NeuerFreihand([]Punkt{Pt(1, 1), Pt(30, 20)}, st)); err != nil {
		t.Errorf("gültiger Strich abgelehnt: %v", err)
	}
}

func TestVereinfachenGlaettenAusduennen(t *testing.T) {
	// Fast gerade Linie mit Zittern unter der Toleranz: bleibt Anfang und Ende.
	var roh []Punkt
	for i := 0; i <= 100; i++ {
		roh = append(roh, Pt(float64(i), 0.2*math.Sin(float64(i))))
	}
	if v := Vereinfacht(roh, 0.5); len(v) != 2 || v[0] != roh[0] || v[1] != roh[100] {
		t.Errorf("gerade: %v", v)
	}
	// Ein Knick bleibt.
	knick := []Punkt{Pt(0, 0), Pt(5, 0.1), Pt(10, 0), Pt(10, 10)}
	if v := Vereinfacht(knick, 0.5); len(v) != 3 || v[1] != Pt(10, 0) {
		t.Errorf("Knick: %v", v)
	}
	// Glätten: Anfang und Ende bleiben, die Ecke wird abgeschnitten, Punkte verdoppeln sich etwa.
	g := Geglaettet([]Punkt{Pt(0, 0), Pt(10, 0), Pt(10, 10)}, 1)
	if g[0] != Pt(0, 0) || g[len(g)-1] != Pt(10, 10) || len(g) != 4 || g[1] != Pt(7.5, 0) || g[2] != Pt(10, 2.5) {
		t.Errorf("Chaikin: %v", g)
	}
	viele := make([]Punkt, 10000)
	for i := range viele {
		viele[i] = Pt(float64(i), float64(i%7))
	}
	a := Ausgeduennt(viele, 100)
	if len(a) != 100 || a[0] != viele[0] || a[99] != viele[9999] {
		t.Errorf("ausgedünnt: %d, %v %v", len(a), a[0], a[99])
	}
	if f := FreihandAusZug(viele, 0.01); len(f) > MaxFreihandPunkte || len(f) < 2 {
		t.Errorf("FreihandAusZug: %d Punkte", len(f))
	}
}
