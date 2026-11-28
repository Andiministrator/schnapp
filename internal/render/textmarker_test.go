// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0048: Der Textmarker mischt multiplizierend — heller Grund nimmt seine Farbe an, dunkle Schrift bleibt dunkel.

// multiplizierSoll ist die Formel von ADR-0048 Punkt 1 in Gleitkomma (Werte 0 bis 1, nicht vormultipliziert).
func multiplizierSoll(b, s color.NRGBA, deckung float64) color.NRGBA {
	as := float64(s.A) / 255 * deckung
	ab := float64(b.A) / 255
	ao := as + ab*(1-as)
	if ao == 0 {
		return color.NRGBA{}
	}
	k := func(cs, cb uint8) uint8 {
		s, b := float64(cs)/255, float64(cb)/255
		co := as*(1-ab)*s + as*ab*s*b + (1-as)*ab*b
		return uint8(math.Round(co / ao * 255))
	}
	return color.NRGBA{R: k(s.R, b.R), G: k(s.G, b.G), B: k(s.B, b.B), A: uint8(math.Round(ao * 255))}
}

func TestMischeMultipliziertWieFormel(t *testing.T) {
	r := rand.New(rand.NewPCG(48, 1))
	zufall := func() color.NRGBA {
		c := color.NRGBA{R: uint8(r.IntN(256)), G: uint8(r.IntN(256)), B: uint8(r.IntN(256)), A: uint8(r.IntN(256))}
		if r.IntN(3) == 0 {
			c.A = 255 // deckender Grund ist der Normalfall
		}
		return c
	}
	for i := 0; i < 20000; i++ {
		b, s := zufall(), zufall()
		ma := uint32(r.IntN(0x10000))
		if i%4 == 0 {
			ma = 0xffff
		}
		p := []byte{b.R, b.G, b.B, b.A}
		mischeMultipliziert(p, s, ma)
		soll := multiplizierSoll(b, s, float64(ma)/0xffff)
		ist := color.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}
		// Bei kleinem Alpha schwankt der entmultiplizierte Wert stärker; dort zählt nur der Beitrag.
		tol := 2
		if soll.A < 16 {
			tol = 255
		}
		if abst(int(ist.A), int(soll.A)) > 1 || abst(int(ist.R), int(soll.R)) > tol || abst(int(ist.G), int(soll.G)) > tol ||
			abst(int(ist.B), int(soll.B)) > tol {
			t.Fatalf("Grund %v, Quelle %v, Deckung %#x: %v, Soll %v", b, s, ma, ist, soll)
		}
	}
}

func abst(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}

// textmarkerDok ist ein weißes Bild mit einem schwarzen Balken („Schrift“) und einem gelben Textmarker darüber.
func textmarkerDok(t *testing.T, deckkraft int, alpha uint8, mischung document.Mischart) *document.Dokument {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, 120, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 120; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: alpha}
			if y >= 26 && y < 34 && x >= 40 && x < 80 {
				c = color.NRGBA{A: 255}
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	m := document.GeraderPfeil(document.Pt(10, 30), document.Pt(110, 30),
		document.Strich{Farbe: color.NRGBA{R: 0xFF, G: 0xE6, B: 0x28, A: 0xFF}, Breite: 20})
	m.Ende = document.KopfKeine
	m.Ebene.Mischung = mischung
	m.Ebene = m.Ebene.MitDeckkraft(deckkraft)
	hinzu(t, d, m)
	return d
}

func TestTextmarkerFaerbtHellesUndLaesstDunkles(t *testing.T) {
	b := bild(t, textmarkerDok(t, 100, 255, document.MischMultiplizieren))
	if c := b.NRGBAAt(20, 30); c != (color.NRGBA{R: 0xFF, G: 0xE6, B: 0x28, A: 0xFF}) {
		t.Errorf("weißer Grund unter dem Marker: %v, Soll die Markerfarbe", c)
	}
	if c := b.NRGBAAt(60, 30); c != (color.NRGBA{A: 255}) {
		t.Errorf("schwarze Schrift unter dem Marker: %v, Soll schwarz", c)
	}
	if c := b.NRGBAAt(20, 5); c != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("neben dem Marker: %v", c)
	}
	// Deckkraft der Ebene 50 %: halb so stark, Schrift bleibt schwarz.
	h := bild(t, textmarkerDok(t, 50, 255, document.MischMultiplizieren))
	if c := h.NRGBAAt(20, 30); abst(int(c.B), 255-(255-0x28)/2) > 2 || c.R != 255 {
		t.Errorf("Deckkraft 50 %%: %v", c)
	}
	if c := h.NRGBAAt(60, 30); c != (color.NRGBA{A: 255}) {
		t.Errorf("Deckkraft 50 %%, Schrift: %v", c)
	}
	// Mit Schatten mischt die ganze Ebene multiplizierend (Schatten und Strich im Zwischenbild): Die Schrift bleibt
	// schwarz, der weiße Grund nimmt die Markerfarbe an.
	ds := textmarkerDok(t, 100, 255, document.MischMultiplizieren)
	m := ds.Objekte()[0].(document.Pfeil)
	m.Ebene.Schatten = document.Schatten{An: true, Winkel: 90, Laenge: 4, Farbe: color.NRGBA{B: 200, A: 255}}
	ds2, err := document.Neu(ds.Original())
	if err != nil {
		t.Fatal(err)
	}
	hinzu(t, ds2, m)
	bs := bild(t, ds2)
	if c := bs.NRGBAAt(60, 28); c != (color.NRGBA{A: 255}) {
		t.Errorf("mit Schatten, Schrift: %v, Soll schwarz", c)
	}
	if c := bs.NRGBAAt(20, 28); c != (color.NRGBA{R: 0xFF, G: 0xE6, B: 0x28, A: 0xFF}) {
		t.Errorf("mit Schatten, weißer Grund: %v", c)
	}
	// Auf durchsichtigem Grund wirkt er wie „über“ (W3C, ADR-0048 Punkt 1).
	d := bild(t, textmarkerDok(t, 100, 0, document.MischMultiplizieren))
	if c := d.NRGBAAt(20, 30); c != (color.NRGBA{R: 0xFF, G: 0xE6, B: 0x28, A: 0xFF}) {
		t.Errorf("durchsichtiger Grund: %v", c)
	}
}

// Ohne Mischung deckt derselbe Strich die Schrift: Der Unterschied kommt von der Mischung, nicht vom Strich.
func TestTextmarkerOhneMischungDeckt(t *testing.T) {
	d := textmarkerDok(t, 100, 255, document.MischNormal)
	if c := bild(t, d).NRGBAAt(60, 30); c.R != 0xFF {
		t.Errorf("ohne Mischung: %v, Soll die Markerfarbe", c)
	}
}
