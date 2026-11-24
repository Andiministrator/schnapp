// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// flaeche ist die Fläche des Vielecks (positiv im Uhrzeigersinn bei y nach unten).
func flaeche(v []Punkt) float64 {
	var s float64
	for i, a := range v {
		b := v[(i+1)%len(v)]
		s += a.X*b.Y - b.X*a.Y
	}
	return s / 2
}

// Das Schild schildG = (100,100)-(200,140), Schriftgrad 20: Eckabstand 5, Basis höchstens 20.
var schildG = Rechteck{Min: Pt(100, 100), Max: Pt(200, 140)}

func TestSpitzendreieckSeiten(t *testing.T) {
	const grad = 20.0
	for _, tc := range []struct {
		name  string
		tip   Punkt
		seite int
		a, b  Punkt
	}{
		{"oben", Pt(150, 60), 0, Pt(140, 100), Pt(160, 100)},
		{"rechts", Pt(260, 120), 1, Pt(200, 112.5), Pt(200, 127.5)},
		{"unten", Pt(150, 190), 2, Pt(160, 140), Pt(140, 140)},
		{"links", Pt(40, 120), 3, Pt(100, 127.5), Pt(100, 112.5)},
	} {
		a, sp, b, seite, ok := spitzendreieck(schildG, tc.tip, grad)
		if !ok || seite != tc.seite || a != tc.a || b != tc.b || sp != tc.tip {
			t.Errorf("%s: %v %v %v seite %d ok %v", tc.name, a, sp, b, seite, ok)
		}
	}
}

// Die Basis bleibt von jeder Ecke mindestens 0,25 Schriftgrad entfernt, auch wenn der Strahl dicht an einer Ecke liegt,
// und höchstens einen Schriftgrad breit; die Spitze in jede Richtung rund um das Schild liefert ein Dreieck.
func TestSpitzendreieckAlleRichtungen(t *testing.T) {
	const grad = 20.0
	e := SpitzeAbstand * grad
	m := Pt(150, 120)
	for w := 0; w < 360; w++ {
		for _, r := range []float64{70, 200} {
			sp := m.Plus(Pt(r*math.Cos(float64(w)*math.Pi/180), r*math.Sin(float64(w)*math.Pi/180)))
			a, s, b, seite, ok := spitzendreieck(schildG, sp, grad)
			if !ok {
				t.Fatalf("Winkel %d Radius %v: kein Dreieck", w, r)
			}
			if s != sp {
				t.Errorf("Spitze verändert")
			}
			for _, p := range []Punkt{a, b} {
				var entlang, lang float64
				switch seite {
				case 0, 2:
					lang, entlang = schildG.Breite(), p.X-schildG.Min.X
					if p.Y != schildG.Min.Y && p.Y != schildG.Max.Y {
						t.Errorf("Winkel %d: Basispunkt %v nicht auf Ober-/Unterkante", w, p)
					}
				default:
					lang, entlang = schildG.Hoehe(), p.Y-schildG.Min.Y
					if p.X != schildG.Min.X && p.X != schildG.Max.X {
						t.Errorf("Winkel %d: Basispunkt %v nicht auf Seitenkante", w, p)
					}
				}
				if entlang < e-1e-9 || entlang > lang-e+1e-9 {
					t.Errorf("Winkel %d: Basispunkt %v zu nah an der Ecke (%.3f von %.1f)", w, p, entlang, lang)
				}
			}
			if d := math.Hypot(a.X-b.X, a.Y-b.Y); d > grad+1e-9 {
				t.Errorf("Winkel %d: Basis %v breiter als der Schriftgrad", w, d)
			}
		}
	}
}

func TestSprechblaseUmriss(t *testing.T) {
	tx := Text{Ort: Pt(10, 10), Inhalt: "Hallo", Schriftgrad: 20}
	g := tx.Grenzen()
	// Ohne Spitze: nur das Schild, im Uhrzeigersinn.
	v := Sprechblase(tx)
	if len(v) != 4 || v[0] != g.Min || v[2] != g.Max || math.Abs(flaeche(v)-g.Breite()*g.Hoehe()) > 1e-9 {
		t.Errorf("ohne Spitze: %v", v)
	}
	for _, sp := range []Punkt{Pt(g.Min.X-80, g.Min.Y+3), Pt(g.Max.X+80, g.Max.Y+60), Pt(g.Min.X+10, g.Min.Y-90),
		Pt(g.Min.X+10, g.Max.Y+90), Pt(g.Max.X+50, g.Min.Y-50), Pt(g.Min.X-50, g.Max.Y+50)} {
		tx.Spitze = &sp
		v = Sprechblase(tx)
		if len(v) != 7 {
			t.Fatalf("Spitze %v: %d Ecken", sp, len(v))
		}
		if f := flaeche(v); f <= g.Breite()*g.Hoehe() {
			t.Errorf("Spitze %v: Fläche %.1f nicht größer als das Schild", sp, f)
		}
		// Das Dreieck sitzt auf der Schildseite: Fläche = Schild + Dreieck, der Umriss kreuzt sich nicht selbst.
		da, dsp, db, _ := tx.Spitzendreieck()
		dreieck := math.Abs((db.X-da.X)*(dsp.Y-da.Y)-(db.Y-da.Y)*(dsp.X-da.X)) / 2
		if f := flaeche(v); math.Abs(f-g.Breite()*g.Hoehe()-dreieck) > 1e-6 {
			t.Errorf("Spitze %v: Fläche %.3f, Soll Schild %.3f + Dreieck %.3f", sp, f, g.Breite()*g.Hoehe(), dreieck)
		}
		found := false
		for _, p := range v {
			found = found || p == sp
		}
		if !found {
			t.Errorf("Spitze %v fehlt im Umriss", sp)
		}
		if h := tx.Huelle(); !h.Enthaelt(sp) || h.Vereint(g) != h {
			t.Errorf("Hülle %v enthält Spitze %v nicht", h, sp)
		}
		if !tx.Enthaelt(sp) || !tx.Enthaelt(g.Min.Plus(g.Max).Mal(0.5)) {
			t.Errorf("Spitze %v: Treffer auf Spitze oder Mitte fehlt", sp)
		}
	}
}

func TestSprechblaseOhneDreieck(t *testing.T) {
	tx := Text{Ort: Pt(10, 10), Inhalt: "Hallo", Schriftgrad: 20}
	g := tx.Grenzen()
	for name, sp := range map[string]Punkt{
		"im Schild":    g.Min.Plus(g.Max).Mal(0.5),
		"auf dem Rand": Pt(g.Max.X, g.Min.Y+4),
		"sehr nah":     Pt(g.Max.X+4.9, g.Min.Y+4), // unter 0,25 × 20 = 5
		"Ecke nah":     Pt(g.Max.X+3, g.Max.Y+3),   // Abstand 4,24
	} {
		tx.Spitze = &sp
		if _, _, _, ok := tx.Spitzendreieck(); ok {
			t.Errorf("%s: Dreieck trotz Nähe", name)
		}
		if v := Sprechblase(tx); len(v) != 4 {
			t.Errorf("%s: %d Ecken", name, len(v))
		}
		if h := tx.Huelle(); !h.Enthaelt(sp) {
			t.Errorf("%s: Hülle ohne Spitze", name)
		}
	}
	// Knapp außerhalb der Schwelle gibt es das Dreieck.
	sp := Pt(g.Max.X+5.1, (g.Min.Y+g.Max.Y)/2)
	tx.Spitze = &sp
	if _, _, _, ok := tx.Spitzendreieck(); !ok {
		t.Error("Spitze 5,1 vom Schild: kein Dreieck")
	}
}

func TestSpitzendreieckKurzeSeite(t *testing.T) {
	// Seite höchstens zweimal Eckabstand (10 bei Grad 20): kein Dreieck.
	g := Rechteck{Min: Pt(0, 0), Max: Pt(100, 10)}
	if _, _, _, _, ok := spitzendreieck(g, Pt(150, 5), 20); ok {
		t.Error("Seite von 10 trägt kein Dreieck")
	}
	// Knapp mehr: Basis (11−10)/2 = 0,5 breit, ganz zwischen den Eckabständen (5 bis 6).
	g = Rechteck{Min: Pt(0, 0), Max: Pt(100, 11)}
	a, _, b, _, ok := spitzendreieck(g, Pt(150, 5), 20)
	lo, hi := math.Min(a.Y, b.Y), math.Max(a.Y, b.Y)
	if !ok || lo < 5-1e-9 || hi > 6+1e-9 || math.Abs(hi-lo-0.5) > 1e-9 {
		t.Errorf("kurze Seite: %v %v ok %v", a, b, ok)
	}
	// Entartetes Schild.
	if _, _, _, _, ok := spitzendreieck(Rechteck{}, Pt(50, 50), 20); ok {
		t.Error("Schild ohne Fläche")
	}
}

func TestSprechblaseTreffer(t *testing.T) {
	tx := Text{Ort: Pt(10, 10), Inhalt: "Hallo", Schriftgrad: 20}
	g := tx.Grenzen()
	sp := Pt(g.Max.X+60, (g.Min.Y+g.Max.Y)/2)
	tx.Spitze = &sp
	if !tx.Enthaelt(Pt(g.Max.X+10, (g.Min.Y+g.Max.Y)/2)) {
		t.Error("Dreieck nicht getroffen")
	}
	if tx.Enthaelt(Pt(g.Max.X+30, g.Min.Y)) {
		t.Error("Punkt neben dem Dreieck getroffen")
	}
	if tx.Enthaelt(Pt(g.Min.X-1, g.Min.Y)) {
		t.Error("Punkt links vom Schild getroffen")
	}
	if !tx.Enthaelt(Pt(g.Min.X, g.Min.Y)) {
		t.Error("Ecke nicht getroffen (Rand eingeschlossen)")
	}
	tx.Spitze = nil
	if tx.Enthaelt(Pt(g.Max.X+10, (g.Min.Y+g.Max.Y)/2)) || !tx.Enthaelt(g.Max) {
		t.Error("ohne Spitze: nur das Schild")
	}
}

func TestSpitzePruefungUndReichweite(t *testing.T) {
	tx := Text{Ort: Pt(10, 10), Inhalt: "Hi", Schriftgrad: 20}
	for _, sp := range []Punkt{{math.NaN(), 0}, {0, math.Inf(1)}, {2e6, 0}} {
		tx.Spitze = &sp
		if err := tx.pruefe(); err == nil {
			t.Errorf("Spitze %v gilt", sp)
		}
	}
	aussen := Pt(-50, 500) // außerhalb des Bilds ist erlaubt
	tx.Spitze = &aussen
	if err := tx.pruefe(); err != nil {
		t.Errorf("Spitze %v: %v", aussen, err)
	}
	if r := Reichweite(tx); !r.Enthaelt(aussen) {
		t.Errorf("Reichweite %v umfasst die Spitze nicht", r)
	}
	if b := tx.SchenkelBreite(); math.Abs(b-1.2) > 1e-9 {
		t.Errorf("Schenkelbreite %v", b)
	}
	if kl := (Text{Schriftgrad: 5}); kl.SchenkelBreite() != 1 {
		t.Errorf("Schenkelbreite klein %v", kl.SchenkelBreite())
	}
}
