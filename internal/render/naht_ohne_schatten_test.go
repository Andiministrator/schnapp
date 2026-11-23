// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// verlaufDokument ist ein deckendes Bild b×h mit einem Farbverlauf (Saum und Abdunklung zeigen sich in jedem Kanal) und
// den Streifen st in der Form Abriss.
func verlaufDokument(t testing.TB, b, h int, st []document.Streifen) *document.Dokument {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			i := bild.PixOffset(x, y)
			bild.Pix[i], bild.Pix[i+1], bild.Pix[i+2], bild.Pix[i+3] = uint8(40+x%160), uint8(60+y%150), uint8(100+(x+y)%120), 0xff
		}
	}
	d, err := document.NeuAus(bild)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	return d
}

// imSpalt liefert, welche Pixel ganz im Spalt einer Naht liegen (mit einem Pixel Rand, auch zu den Nachbarspalten): Der
// Spalt ist durchsichtig, nichts darf dort liegen, auch kein Fasersaum einer kreuzenden Naht (F-438, F-499). Abriss:
// ein Stützpunkt je Pixel, die Linie wird also je Spalte (Zeilen-Naht) bzw. Zeile (Spalten-Naht) abgelesen.
func imSpalt(d *document.Dokument) map[image.Point]bool {
	g := d.Groesse()
	res := map[image.Point]bool{}
	von, bis := document.Spaltband()
	for _, n := range d.Naehte() {
		laenge := g.X
		if !n.Zeilen {
			laenge = g.Y
		}
		band := document.Kantenband(d.Kante(), n, laenge, d.KantenFaktor(), von, bis)
		anz := len(band)/2 - 1
		// oben[i] und unten[i]: Lage quer zur Naht bei Stützpunkt i
		quer := func(p document.Punkt) float64 {
			if n.Zeilen {
				return p.Y
			}
			return p.X
		}
		oben := func(i int) float64 { return quer(band[min(max(i, 0), anz)]) }
		unten := func(i int) float64 { return quer(band[2*anz+1-min(max(i, 0), anz)]) }
		for t := 0; t < laenge; t++ {
			o := max(oben(t-1), oben(t), oben(t+1))
			u := min(unten(t-1), unten(t), unten(t+1))
			for q := int(o) + 1; float64(q+1) <= u-1; q++ {
				if n.Zeilen {
					res[image.Pt(t, q)] = true
				} else {
					res[image.Pt(q, t)] = true
				}
			}
		}
	}
	return res
}

// F-407 und F-499: KantenOhneSchatten (das Bild der Leinwand im Editor) mischt wie Bild — wo es deckend ist, sind die
// Pixel dieselben —, lässt den Schatten im Spalt weg, und der Spalt ist durchsichtig, auch dort, wo der Fasersaum einer
// kreuzenden oder eng benachbarten Naht hineinragt (die zweite Leerung, F-438): Sollpixel im Spalt.
func TestKantenOhneSchatten(t *testing.T) {
	st := map[string][]document.Streifen{
		"eine Naht":         {{Zeilen: true, Von: 100, Bis: 101}},
		"kreuzend":          {{Zeilen: true, Von: 100, Bis: 101}, {Zeilen: false, Von: 200, Bis: 201}},
		"eng beieinander":   {{Zeilen: true, Von: 100, Bis: 101}, {Zeilen: true, Von: 110, Bis: 111}},
		"zwei Achsen, drei": {{Zeilen: true, Von: 60, Bis: 61}, {Zeilen: false, Von: 120, Bis: 121}, {Zeilen: false, Von: 130, Bis: 131}},
	}
	for name, s := range st {
		d := verlaufDokument(t, 320, 220, s)
		soll := bild(t, d)
		ist := d.Ansichtsbild()
		KantenOhneSchatten(ist, d)
		if ist.Rect != soll.Rect {
			t.Fatalf("%s: Größe %v, erwartet %v", name, ist.Rect, soll.Rect)
		}
		deckend, gleich := 0, 0
		for y := 0; y < ist.Rect.Dy(); y++ {
			for x := 0; x < ist.Rect.Dx(); x++ {
				p := ist.NRGBAAt(x, y)
				if p.A != 255 {
					continue
				}
				deckend++
				if p == soll.NRGBAAt(x, y) {
					gleich++
				} else if soll.NRGBAAt(x, y).A == 255 {
					t.Errorf("%s: Pixel (%d,%d) im Editor-Bild %v, in Bild %v", name, x, y, p, soll.NRGBAAt(x, y))
				}
			}
		}
		if deckend == 0 || gleich == 0 {
			t.Errorf("%s: %d deckende, %d gleiche Pixel", name, deckend, gleich)
		}
		spalt := imSpalt(d)
		if len(spalt) == 0 {
			t.Fatalf("%s: kein Pixel im Spalt gefunden (Annahme des Tests)", name)
		}
		for p := range spalt {
			if a := ist.NRGBAAt(p.X, p.Y).A; a != 0 {
				t.Errorf("%s: Pixel (%d,%d) im Spalt hat Deckung %d, erwartet 0", name, p.X, p.Y, a)
				break
			}
		}
	}
}
