// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 26 Punkt 3: Der Schatten liegt versetzt entgegen dem Licht, scharf in der Schattenfarbe, weich
// verlaufend, unter dem Objekt; mit der Deckkraft der Ebene zusammen eingemischt; er kostet im Trockenlauf.

func schattenKasten(s document.Schatten) document.Kasten {
	k := document.Kasten{Bereich: document.R(document.Pt(40, 40), document.Pt(100, 80)),
		Fuellung: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}
	k.Ebene.Schatten = s
	return k
}

func TestSchattenScharfVersetzt(t *testing.T) {
	s := document.Schatten{An: true, Winkel: 135, Laenge: 10 * math.Sqrt2, Farbe: color.NRGBA{A: 255}}
	d := weissesDokument(t, 200, 160)
	hinzu(t, d, schattenKasten(s))
	b := bild(t, d)
	// Versatz (+10, +10): unten rechts schwarz, oben links weiß, der Kasten selbst weiß.
	if g := b.NRGBAAt(105, 85).R; g != 0 {
		t.Errorf("Schatten unten rechts %d, Soll 0", g)
	}
	if g := b.NRGBAAt(111, 60).R; g != 255 {
		t.Errorf("rechts neben dem Schatten %d, Soll 255", g)
	}
	if g := b.NRGBAAt(70, 60).R; g != 255 {
		t.Errorf("Kasten über dem Schatten %d, Soll 255", g)
	}
	if g := b.NRGBAAt(45, 35).R; g != 255 {
		t.Errorf("oben links %d, Soll 255 (kein Schatten gegen das Licht)", g)
	}
}

func TestSchattenWeichUndFarbe(t *testing.T) {
	s := document.Schatten{An: true, Winkel: 90, Laenge: 20, Weich: 12, Farbe: color.NRGBA{A: 102}}
	d := weissesDokument(t, 200, 180)
	hinzu(t, d, schattenKasten(s))
	b := bild(t, d)
	// Licht von oben: Schatten 20 px nach unten. In der Mitte des Schattens 40 % Schwarz (255 · 0,6 = 153), zum Rand
	// hin heller, weit außen weiß.
	mitte := int(b.NRGBAAt(70, 85).R)
	if mitte < 150 || mitte > 158 {
		t.Errorf("Mitte des Schattens %d, Soll rund 153", mitte)
	}
	rand := int(b.NRGBAAt(70, 100).R)
	if rand <= mitte || rand >= 255 {
		t.Errorf("Rand des Schattens %d, Soll zwischen %d und 255 (weich)", rand, mitte)
	}
	if g := b.NRGBAAt(70, 125).R; g != 255 {
		t.Errorf("weit unter dem Schatten %d, Soll 255", g)
	}
}

func TestSchattenMitDeckkraftDerEbene(t *testing.T) {
	s := document.Schatten{An: true, Winkel: 135, Laenge: 10 * math.Sqrt2, Farbe: color.NRGBA{A: 255}}
	k := schattenKasten(s)
	k.Fuellung = color.NRGBA{A: 255}
	k.Ebene = k.Ebene.MitDeckkraft(50)
	d := weissesDokument(t, 200, 160)
	hinzu(t, d, k)
	b := bild(t, d)
	// Kasten und Schatten schwarz, zusammen zu 50 %: die Überlappung einmal gedeckt, nicht doppelt.
	for _, p := range [][2]int{{70, 60}, {105, 85}, {95, 75}} {
		if g := b.NRGBAAt(p[0], p[1]).R; g < 125 || g > 130 {
			t.Errorf("Pixel %v = %d, Soll rund 127 (Gruppendeckkraft)", p, g)
		}
	}
}

func TestSchattenKostet(t *testing.T) {
	d := weissesDokument(t, 400, 300)
	hinzu(t, d, schattenKasten(document.Schatten{}))
	ohne := Aufwand(d)
	d2 := weissesDokument(t, 400, 300)
	hinzu(t, d2, schattenKasten(document.StandardSchatten(1)))
	if mit := Aufwand(d2); mit <= ohne*2 {
		t.Errorf("Aufwand mit Schatten %v, ohne %v", mit, ohne)
	}
}

// Die drei Kastenfilter haben zusammen die Varianz σ² (Gauß-Näherung), und Weichzeichnen erhält die Masse im Inneren.
func TestKastenBreiten(t *testing.T) {
	for _, sigma := range []float64{0.5, 1, 2.5, 4, 10, 37} {
		v := 0.0
		for _, b := range kastenBreiten(sigma) {
			if b%2 == 0 || b < 1 {
				t.Fatalf("σ %v: Breite %d", sigma, b)
			}
			v += float64(b*b-1) / 12
		}
		if math.Abs(v-sigma*sigma) > 0.6*sigma+0.4 {
			t.Errorf("σ %v: Varianz %v, Soll %v", sigma, v, sigma*sigma)
		}
	}
	w, h := 64, 64
	m := make([]float32, w*h)
	m[32*w+32] = 1
	weichzeichnen(m, w, h, 3)
	summe := float32(0)
	for _, x := range m {
		summe += x
	}
	if math.Abs(float64(summe)-1) > 1e-4 || m[32*w+32] >= 1 || m[32*w+33] <= 0 {
		t.Errorf("Summe %v, Mitte %v", summe, m[32*w+32])
	}
}

// Der Editor bekommt denselben Schatten ohne das Objekt.
func TestSchattenbildOhneObjekt(t *testing.T) {
	s := document.Schatten{An: true, Winkel: 135, Laenge: 10 * math.Sqrt2, Farbe: color.NRGBA{A: 255}}
	d := weissesDokument(t, 200, 160)
	k := schattenKasten(s)
	hinzu(t, d, k)
	sb, err := Schattenbild(d, k)
	if err != nil || sb == nil {
		t.Fatalf("%v, %v", sb, err)
	}
	if a := sb.NRGBAAt(105, 85).A; a != 255 {
		t.Errorf("Schatten unten rechts Deckung %d", a)
	}
	if a := sb.NRGBAAt(45, 45).A; a != 0 {
		t.Errorf("im Kasten außerhalb des Schattens Deckung %d: das Schattenbild trägt das Objekt", a)
	}
	if sb, _ := Schattenbild(d, schattenKasten(document.Schatten{})); sb != nil {
		t.Error("Schattenbild ohne Schatten")
	}
}
