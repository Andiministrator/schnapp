// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0049 Punkt 2: Der Freihand-Strich deckt seine Pixel, halbdurchsichtig jedes höchstens einmal, auch wo er sich
// kreuzt; Punkte außerhalb bleiben weiß.
func TestFreihandZeichnetUeberlappungsfrei(t *testing.T) {
	d := weissesDokument(t, 120, 100)
	halb := color.NRGBA{R: 200, A: 128}
	// Eine Schleife: kreuzt sich bei (60, 50).
	pts := []document.Punkt{document.Pt(20, 20), document.Pt(100, 80), document.Pt(100, 20), document.Pt(20, 80)}
	hinzu(t, d, document.NeuerFreihand(pts, document.Strich{Farbe: halb, Breite: 10}))
	b := bild(t, d)
	einmal := b.NRGBAAt(40, 35) // auf der ersten Strecke
	kreuz := b.NRGBAAt(60, 50)  // Kreuzung
	if einmal.G == 255 {
		t.Fatalf("Strich nicht gezeichnet: %v", einmal)
	}
	if kreuz != einmal {
		t.Errorf("Kreuzung %v, einfach %v: doppelt gedeckt", kreuz, einmal)
	}
	if c := b.NRGBAAt(60, 10); c != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("außerhalb %v", c)
	}
	// Gemustert gibt es Lücken.
	d2 := weissesDokument(t, 120, 40)
	hinzu(t, d2, document.NeuerFreihand([]document.Punkt{document.Pt(10, 20), document.Pt(110, 20)},
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 4, Art: document.StrichGestrichelt}))
	b2 := bild(t, d2)
	luecke := false
	for x := 10; x < 110; x++ {
		if b2.NRGBAAt(x, 20).R == 255 {
			luecke = true
		}
	}
	if !luecke {
		t.Error("gestrichelt ohne Lücke")
	}
}

// F-1043 (Kritik #333 K-1, Mutant FH28): Der Trockenlauf zählt Punkte und Teile eines Freihand-Strichs im Aufwand der
// Kanten. Ein gepunkteter Zickzack zu 4096 Punkten (8189 Teile) kostet gemessen 2,9e7 Kanten-Aufwand, davon 1,2e7 aus
// dieser Zeile. Seit F-1051 trägt ein Dokument höchstens 256 solcher Striche (MaxFreihandGesamt); der Test setzt die
// Kantengrenze deshalb auf 4,6e9: 200 Striche liegen mit der Zeile darüber (5,8e9), ohne sie (3,4e9) darunter.
func TestFreihandTrockenlaufZaehlt(t *testing.T) {
	alt := grenzen.kanten
	grenzen.kanten = 4.6e9
	t.Cleanup(func() { grenzen.kanten = alt })
	d := weissesDokument(t, 300, 200)
	pts := make([]document.Punkt, document.MaxFreihandPunkte)
	for i := range pts {
		pts[i] = document.Pt(float64(i%2)*290+5, float64(i%190)+5)
	}
	for range 200 {
		hinzu(t, d, document.NeuerFreihand(pts, document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 4, Art: document.StrichGepunktet}))
	}
	if err := Pruefe(d); err == nil {
		t.Errorf("200 Freihand-Striche zu 4096 Punkten ohne Budgetfehler (Kanten %.3g)", trockenlauf(d, false).kanten)
	}
}

// F-1043 (K-2, Mutant FH29): Mit Ebenen-Deckkraft unter 100 % reicht das Zwischenbild bis hinter den starren Rand
// eines dicken Strichs an einer Naht; dieselben Pixel wie bei 100 %, über mehrere Lagen des Strichs zum Streifen.
func TestFreihandDeckkraftRandAnNaht(t *testing.T) {
	spur := func(p int, von, bis, y float64, breite float64) map[image.Point]bool {
		d := weissesDokument(t, 300, 160)
		if err := d.SetzeStreifen([]document.Streifen{{Von: 161, Bis: 200}}); err != nil {
			t.Fatal(err)
		}
		grund, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		f := document.NeuerFreihand([]document.Punkt{document.Pt(von, y), document.Pt(bis, y+3), document.Pt(bis+10, y+30)},
			document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: breite})
		f.Ebene = f.Ebene.MitDeckkraft(p)
		hinzu(t, d, f)
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		m := map[image.Point]bool{}
		for yy := 0; yy < b.Rect.Dy(); yy++ {
			for x := 0; x < b.Rect.Dx(); x++ {
				if b.NRGBAAt(x, yy) != grund.NRGBAAt(x, yy) {
					m[image.Pt(x, yy)] = true
				}
			}
		}
		return m
	}
	gesamt := 0
	for _, breite := range []float64{20, 40, 60} {
		for _, bis := range []float64{150, 160, 170, 190, 200, 210, 230} {
			voll, halb := spur(100, 60, bis, 50, breite), spur(99, 60, bis, 50, breite)
			fehlt := 0
			for p := range voll {
				if !halb[p] {
					fehlt++
				}
			}
			gesamt += len(voll)
			if fehlt > len(voll)/20 { // Ränder mit winziger Deckung fallen bei 99 % auf den Grund (Rundung)
				t.Errorf("Breite %v bis %v: %d von %d Pixeln fehlen mit Gruppendeckkraft", breite, bis, fehlt, len(voll))
			}
		}
	}
	if gesamt == 0 {
		t.Fatal("nichts gezeichnet")
	}
}
