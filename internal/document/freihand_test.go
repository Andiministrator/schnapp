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
	if NeuerFreihand([]Punkt{Pt(5, 5), Pt(5, 5), Pt(5, 5)}, st).pruefe() == nil { // F-1040
		t.Error("Strich ohne Länge angenommen")
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

// F-1043 (Kritik #333 K-3, Mutant FH30): Die Reichweite trägt einen Pixel Reserve für die Kantenglättung über die Grenzen.
func TestFreihandReichweiteMitReserve(t *testing.T) {
	a := NeuerFreihand([]Punkt{Pt(0, 0), Pt(10, 5), Pt(20, 0)}, Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 4})
	if r, w := Reichweite(a), R(Pt(-3, -3), Pt(23, 8)); r != w {
		t.Errorf("Reichweite %v, erwartet %v", r, w)
	}
}

// F-1036 (Kritik #333 K-2, ADR-0049 Nachtrag 1): Nach dem Glätten wird noch einmal vereinfacht und auf 1/100 px
// gerundet. Gespeichert werden nie mehr Punkte als roh; wo Glätten mehr ergäbe, bleibt der ungeglättete Zug.
func TestFreihandAusZugHoechstensRohUndGerundet(t *testing.T) {
	// Zickzack mit spitzen Ecken: geglättet und wieder vereinfacht wären es 10 Punkte, roh sind es 4.
	zick := []Punkt{Pt(0, 0), Pt(20.123, 30), Pt(40, 0.004), Pt(60, 30)}
	if f := FreihandAusZug(zick, 0.5); len(f) > len(zick) || f[1] != Pt(20.12, 30) || f[2] != Pt(40, 0) {
		t.Errorf("Zickzack: %v", f)
	}
	// Ein Kreisbogen aus vielen Rohpunkten: geglättet, wieder vereinfacht, jede Zahl ein Vielfaches von 1/100.
	var bogen []Punkt
	for i := 0; i <= 300; i++ {
		w := math.Pi * float64(i) / 300
		bogen = append(bogen, Pt(100.3+80*math.Cos(w), 90.7-80*math.Sin(w)))
	}
	f := FreihandAusZug(bogen, 0.5)
	if len(f) < 10 || len(f) > 40 {
		t.Errorf("Bogen: %d Punkte, erwartet rund 20 (vereinfacht, geglättet, wieder vereinfacht)", len(f))
	}
	for _, p := range f {
		if p != aufHundertstel(p) || math.Abs(p.X*100-math.Round(p.X*100)) > 1e-6 {
			t.Fatalf("Punkt %v nicht auf 1/100 gerundet", p)
		}
		if r := p.Minus(Pt(100.3, 90.7)).Laenge(); r > 80.01 || r < 79 {
			t.Errorf("Punkt %v mit Radius %.2f", p, r)
			break
		}
	}
	// Das zweite Vereinfachen nimmt die halbe Toleranz: Der geglättete Zug liegt höchstens 0,25 px (plus Rundung)
	// neben dem gespeicherten.
	for _, p := range Geglaettet(Vereinfacht(bogen, 0.5), 2) {
		best := math.Inf(1)
		for i := 1; i < len(f); i++ {
			best = math.Min(best, abstandStrecke(p, f[i-1], f[i]))
		}
		if best > 0.26 {
			t.Errorf("geglätteter Punkt %v liegt %.3f px neben dem gespeicherten Zug", p, best)
			break
		}
	}
	if f[0] != aufHundertstel(bogen[0]) || f[len(f)-1] != aufHundertstel(bogen[300]) {
		t.Errorf("Anfang/Ende %v %v", f[0], f[len(f)-1])
	}
	// Runden lässt gleich gewordene Nachbarn weg, zwei Punkte bleiben.
	if g := Gerundet([]Punkt{Pt(1, 1), Pt(1.001, 1), Pt(2, 2)}); len(g) != 2 {
		t.Errorf("Gerundet: %v", g)
	}
	if g := Gerundet([]Punkt{Pt(1, 1), Pt(1.001, 1.002)}); len(g) != 2 || g[0] != g[1] {
		t.Errorf("Gerundet zwei gleiche: %v", g)
	}
	// Verschieben rundet wie das Anlegen.
	v := NeuerFreihand([]Punkt{Pt(1, 2), Pt(3.5, 4)}, Strich{Breite: 2}).Verschoben(Pt(1/0.73, 0.1234))
	if p := v.Punkte(); p[0] != Pt(2.37, 2.12) || p[1] != Pt(4.87, 4.12) {
		t.Errorf("Verschoben: %v", p)
	}
}
