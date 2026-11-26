// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 19 (Andi, 25.11.2026: „Strich läuft sichtbar in die Spitze“): Schaft und Kopf überlappen nicht.
// Eine halbdurchsichtige Farbe deckt jedes Pixel höchstens einmal — vorher lag der Schaft mit runder Kappe bis B unter
// der Spitze, dort mischte sich die Farbe zweimal (dunkler Fleck). Und der Kopf ist wirklich da: Pixel in seinem Inneren
// tragen die volle einfache Deckung.
func TestPfeilKoepfeDeckenEinmal(t *testing.T) {
	halb := color.NRGBA{A: 128} // Schwarz zu 50 % auf Weiß: einfach gedeckt 127, doppelt 63
	for _, art := range document.Kopfarten {
		for _, enden := range document.PfeilEndenAlle {
			for _, fall := range []struct {
				name    string
				a, k, b document.Punkt
				breite  float64
			}{
				{"gerade", document.Pt(30.3, 60.6), document.Pt(150.4, 60.6), document.Pt(270.5, 60.6), 6},
				{"schräg", document.Pt(40.2, 200.7), document.Pt(160, 140), document.Pt(280.3, 80.1), 9},
				{"gebogen", document.Pt(30, 150), document.Pt(150, 10), document.Pt(270, 150), 12},
			} {
				name := fmt.Sprintf("%s/%s/%s", art.Name(), enden.Name(), fall.name)
				v := document.Pfeil{A: fall.a, B: fall.b, K: fall.k, Strich: document.Strich{Farbe: halb, Breite: fall.breite},
					Kopf: art, Enden: enden}
				d := weissesDokument(t, 320, 240)
				hinzu(t, d, v)
				b := bild(t, d)
				dunkelst := 255
				for i := 0; i < len(b.Pix); i += 4 {
					dunkelst = min(dunkelst, int(b.Pix[i]))
				}
				if dunkelst < 125 {
					t.Errorf("%s: dunkelstes Pixel %d, einfach gedeckt wäre 127: Teile überlappen", name, dunkelst)
				}
				// Im Inneren jedes Kopfs (Mitte zwischen Spitze und Schaftende, etwas zur Spitze hin) ist gedeckt.
				for _, k := range v.Form().Koepfe {
					m := k.Spitze.Mal(0.4).Plus(k.Schaftende.Mal(0.6))
					if art == document.KopfWinkel { // die Spitze ist innen hohl: auf dem Schenkel nahe der Spitze messen
						m = k.Spitze.Minus(k.Richtung.Mal(fall.breite * 0.8))
					}
					x, y := int(m.X), int(m.Y)
					if g := b.NRGBAAt(x, y).R; g > 130 {
						t.Errorf("%s: Kopf bei %v nicht gedeckt (Pixel %d,%d = %d)", name, k.Spitze, x, y, g)
					}
				}
			}
		}
	}
}

// Ein Pfeil kürzer als seine Köpfe zeichnet nur die Köpfe (ZerlegePfeil.Einzeln), ohne Schaft dazwischen.
func TestPfeilKurzNurKoepfe(t *testing.T) {
	v := document.GeraderPfeil(document.Pt(100, 100), document.Pt(110, 100), document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6})
	v.Enden = document.EndenBeide
	pz := ZerlegePfeil(v)
	if len(pz.Teile) != 0 || len(pz.Einzeln) != 2 {
		t.Fatalf("Teile %d, einzeln %d", len(pz.Teile), len(pz.Einzeln))
	}
	d := weissesDokument(t, 200, 200)
	hinzu(t, d, v)
	b := bild(t, d)
	// Beide Köpfe in voller Form: das Dreieck bei B reicht von 87 bis 110, das bei A von 100 bis 123.
	if b.NRGBAAt(90, 100).R > 10 || b.NRGBAAt(120, 100).R > 10 {
		t.Errorf("Köpfe fehlen: %v %v", b.NRGBAAt(90, 100), b.NRGBAAt(120, 100))
	}
}

// Kritik K-5: Mit Ebenen-Deckkraft unter 100 % zeichnet der Renderer über ein Zwischenbild im ebenenRahmen. Der Rahmen
// muss die ganzen Köpfe fassen — vorher schnitt er die Schenkel einer offenen Spitze an kurzen Pfeilen ab.
func TestPfeilDeckkraftSchneidetKopfNicht(t *testing.T) {
	for _, art := range document.Kopfarten {
		v := document.GeraderPfeil(document.Pt(140, 120), document.Pt(150, 120), document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 20})
		v.Kopf, v.Enden = art, document.EndenBeide
		zaehle := func(deckkraft int) int {
			w := v
			w.Ebene = w.Ebene.MitDeckkraft(deckkraft)
			d := weissesDokument(t, 320, 240)
			hinzu(t, d, w)
			b := bild(t, d)
			n := 0
			for i := 0; i < len(b.Pix); i += 4 {
				if b.Pix[i] < 250 {
					n++
				}
			}
			return n
		}
		if voll, fast := zaehle(100), zaehle(99); fast < voll-voll/50 {
			t.Errorf("%s: deckend %d Pixel, mit 99 %% nur %d", art.Name(), voll, fast)
		}
	}
}
