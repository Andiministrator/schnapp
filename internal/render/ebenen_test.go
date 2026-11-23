// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

func weissesDok(t *testing.T) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 60, 40))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func nah(a, b color.NRGBA, tol int) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) <= tol && d(a.G, b.G) <= tol && d(a.B, b.B) <= tol && d(a.A, b.A) <= tol
}

// ADR-0032 Punkte 2, 4 und 10: Eine ausgeblendete Ebene fehlt im Bild; die Deckkraft wirkt auf das Objekt als Ganzes —
// wo Schaft und Spitze eines Pfeils sich überlagern, ist er nicht dunkler als am Schaft allein.
func TestEbenenImRenderer(t *testing.T) {
	rot := color.NRGBA{255, 0, 0, 255}
	weiss := color.NRGBA{255, 255, 255, 255}
	pfeil := document.GeraderPfeil(document.Pt(4, 20), document.Pt(56, 20), document.Strich{Farbe: rot, Breite: 6})

	d := weissesDok(t)
	if _, err := d.Hinzufuegen(document.Kasten{Ebene: document.Ebene{Ausgeblendet: true},
		Bereich: document.R(document.Pt(0, 0), document.Pt(60, 40)), Fuellung: rot, Strich: document.Strich{Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	b, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	if c := b.NRGBAAt(30, 10); c != weiss {
		t.Errorf("ausgeblendeter Kasten gezeichnet: %v", c)
	}

	for _, fall := range []struct {
		deck int
		soll color.NRGBA
	}{{100, rot}, {50, color.NRGBA{255, 127, 127, 255}}, {0, weiss}} {
		d := weissesDok(t)
		p := pfeil
		p.Ebene = p.Ebene.MitDeckkraft(fall.deck)
		if _, err := d.Hinzufuegen(p); err != nil {
			t.Fatal(err)
		}
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		schaft, beides := b.NRGBAAt(12, 20), b.NRGBAAt(48, 20) // Spitze ist max(8, 24) lang: ab x = 32
		if !nah(schaft, fall.soll, 1) || !nah(beides, fall.soll, 1) {
			t.Errorf("Deckkraft %d: Schaft %v, Schaft+Spitze %v, Soll %v", fall.deck, schaft, beides, fall.soll)
		}
		if a := Aufwand(d); fall.deck < 100 && fall.deck > 0 && a <= 0 {
			t.Errorf("Deckkraft %d: Aufwand %v", fall.deck, a)
		}
	}
}

// Die Gruppendeckkraft mischt in sRGB wie jede Füllung: Eine halbdurchsichtige Ebene mit deckender Farbe ergibt
// dasselbe wie dieselbe Farbe mit halbem Alpha, solange sich nichts überlagert (Kasten ohne Strich).
func TestEbenenDeckkraftWieFarbalpha(t *testing.T) {
	blau := color.NRGBA{0, 0, 255, 255}
	k := document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 30)), Fuellung: blau}
	d1 := weissesDok(t)
	k1 := k
	k1.Ebene = k1.Ebene.MitDeckkraft(60)
	if _, err := d1.Hinzufuegen(k1); err != nil {
		t.Fatal(err)
	}
	d2 := weissesDok(t)
	k2 := k
	k2.Fuellung.A = 153 // 60 %
	if _, err := d2.Hinzufuegen(k2); err != nil {
		t.Fatal(err)
	}
	b1, _ := Bild(d1)
	b2, _ := Bild(d2)
	if c1, c2 := b1.NRGBAAt(30, 20), b2.NRGBAAt(30, 20); !nah(c1, c2, 1) {
		t.Errorf("Ebene 60 %%: %v, Farbe mit Alpha 60 %%: %v", c1, c2)
	}
}
