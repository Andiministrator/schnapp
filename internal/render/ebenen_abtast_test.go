// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Vorlage aus dem Verify-Lauf 23.11.2026 (V23-1, V23-5 M12): Jede Art wird über die Stauch-Nähte geschoben; bei 50 %
// Deckkraft darf kein Pixel fehlen, das bei 100 % Tinte trägt — mit und ohne Zuschnitt, Streifen in beiden Achsen.

func abtastBild(t *testing.T, o document.Objekt, deck int, zuschnitt bool) *image.NRGBA {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 400, 300))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if zuschnitt {
		if err := d.SetzeZuschnitt(image.Rect(10, 10, 390, 290)); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 120, Bis: 180}, {Zeilen: true, Von: 80, Bis: 110}}); err != nil {
		t.Fatal(err)
	}
	if o != nil {
		if _, err := d.Hinzufuegen(document.MitEbene(o, document.Ebene{}.MitDeckkraft(deck))); err != nil {
			t.Fatal(err)
		}
	}
	bild, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	return bild
}

// abtastFehlt zählt Pixel mit deutlicher Tinte bei 100 % (gegen das Bild ohne Objekt), die bei 50 % gleich dem Bild
// ohne Objekt sind.
func abtastFehlt(leer, voll, halb *image.NRGBA) (tinte, fehlt int) {
	ab := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	for y := 0; y < voll.Rect.Dy(); y++ {
		for x := 0; x < voll.Rect.Dx(); x++ {
			l, v, h := leer.NRGBAAt(x, y), voll.NRGBAAt(x, y), halb.NRGBAAt(x, y)
			if max(ab(l.R, v.R), ab(l.G, v.G), ab(l.B, v.B), ab(l.A, v.A)) < 64 {
				continue
			}
			tinte++
			if h == l {
				fehlt++
			}
		}
	}
	return
}

func TestEbenenRahmenAbtasten(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	gelb := color.NRGBA{R: 255, G: 220, A: 255}
	arten := map[string]func(dx float64) document.Objekt{
		"Kasten hinter der Naht, Strich 32": func(dx float64) document.Objekt {
			return document.Kasten{Bereich: document.R(document.Pt(180+dx, 110+dx), document.Pt(330, 250)),
				Strich: document.Strich{Farbe: schwarz, Breite: 32}}
		},
		"Kasten vor der Naht, Strich 32": func(dx float64) document.Objekt {
			return document.Kasten{Bereich: document.R(document.Pt(30, 30), document.Pt(120-dx, 80-dx)),
				Strich: document.Strich{Farbe: schwarz, Breite: 32}}
		},
		"Kasten rund hinter der Naht, Strich 24": func(dx float64) document.Objekt {
			return document.Kasten{Bereich: document.R(document.Pt(180+dx, 110+dx), document.Pt(330, 250)), Eckradius: 12,
				Strich: document.Strich{Farbe: schwarz, Breite: 24}}
		},
		"Pfeil zur Naht, Strich 32": func(dx float64) document.Objekt {
			return document.GeraderPfeil(document.Pt(330, 250), document.Pt(182+dx, 112+dx), document.Strich{Farbe: schwarz, Breite: 32})
		},
		"Pfeil gebogen, Strich 12": func(dx float64) document.Objekt {
			return document.Pfeil{A: document.Pt(40, 40), B: document.Pt(300, 200+dx), K: document.Pt(320, 20+dx),
				Strich: document.Strich{Farbe: schwarz, Breite: 12}}
		},
		"Text mit Schild, Unter- und Durchstrich": func(dx float64) document.Objekt {
			return document.Text{Ort: document.Pt(40+6*dx, 40+4*dx), Inhalt: "Hallo Welt gjpq\nzweite Zeile\nÄÖ", Schriftgrad: 36,
				Farbe: schwarz, Hintergrund: gelb, Unterstrichen: true, Durchgestrichen: true, Fett: true, Kursiv: true,
				Ausrichtung: document.AusrichtungRechts}
		},
		"Nummer fünfstellig": func(dx float64) document.Objekt {
			return document.Nummer{Mitte: document.Pt(170+2*dx, 100+2*dx), Wert: 99999, Radius: 14, Farbe: schwarz}
		},
	}
	for _, zuschnitt := range []bool{false, true} {
		leer := abtastBild(t, nil, 100, zuschnitt)
		for name, bau := range arten {
			schlimmst, bei, tinteBei := 0, 0.0, 0
			for dx := 0.0; dx <= 24; dx += 3 {
				o := bau(dx)
				tinte, fehlt := abtastFehlt(leer, abtastBild(t, o, 100, zuschnitt), abtastBild(t, o, 50, zuschnitt))
				if fehlt > schlimmst {
					schlimmst, bei, tinteBei = fehlt, dx, tinte
				}
			}
			if schlimmst > 0 {
				t.Errorf("Zuschnitt %v, %s: bei 50 %% fehlen bis zu %d von %d Tintenpixeln (dx = %v)", zuschnitt, name, schlimmst, tinteBei, bei)
			}
		}
	}
}
