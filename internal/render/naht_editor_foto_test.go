// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/png"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// TestEditorFotoGegenRenderer misst F-407: Mischt der Editor (Gio) die halbdurchsichtigen Kantenschichten anders als
// der Renderer? Messung, kein Test: Er läuft nur mit SCHNAPP_F407_DATEI (die Arbeitsdatei, die ein Probelauf mit
// scripts/probe.sh --drehbuch abgelegt hat) und SCHNAPP_F407_FOTO (das Foto des Editors bei Zoom 100 %, Skala 1,
// Bildschirm 1600×1000), siehe knowledge/topics/lehren.md. Verglichen werden nur Pixel, die der Renderer deckend
// liefert (Bild mit Fasersaum und Abdunklung, nicht der durchsichtige Spalt und sein Schatten, die der Editor über ein
// Schachbrett zeigt). Das Foto liegt versetzt; der Versatz wird gesucht.
func TestEditorFotoGegenRenderer(t *testing.T) {
	datei, foto := os.Getenv("SCHNAPP_F407_DATEI"), os.Getenv("SCHNAPP_F407_FOTO")
	if datei == "" || foto == "" {
		t.Skip("Messung F-407: SCHNAPP_F407_DATEI und SCHNAPP_F407_FOTO setzen")
	}
	g, err := fileformat.Oeffne(datei)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Bild(g.Dokument)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(foto)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	rgb := func(p image.Point) [3]int {
		r, gg, bl, _ := img.At(p.X, p.Y).RGBA()
		return [3]int{int(r >> 8), int(gg >> 8), int(bl >> 8)}
	}
	vergleich := func(ox, oy int) (summe, n, grosse, maxAbw int) {
		for y := 0; y < b.Rect.Dy(); y++ {
			for x := 0; x < b.Rect.Dx(); x += 1 {
				p := b.NRGBAAt(x, y)
				if p.A != 255 {
					continue
				}
				q := image.Pt(ox+x, oy+y)
				if !q.In(img.Bounds()) {
					continue
				}
				c := rgb(q)
				d := max(abs(c[0]-int(p.R)), abs(c[1]-int(p.G)), abs(c[2]-int(p.B)))
				summe += d
				n++
				maxAbw = max(maxAbw, d)
				if d > 2 {
					grosse++
				}
			}
		}
		return
	}
	besteSumme, bestN, bestOX, bestOY := 1<<62, 0, 0, 0
	for oy := 195; oy <= 205; oy++ {
		for ox := 220; ox <= 232; ox++ {
			s, n, _, _ := vergleich(ox, oy)
			if n > 0 && s*bestN < besteSumme*n || bestN == 0 {
				besteSumme, bestN, bestOX, bestOY = s, n, ox, oy
			}
		}
	}
	_, n, gross, maxAbw := vergleich(bestOX, bestOY)
	t.Logf("Versatz (%d,%d); %d deckende Pixel verglichen, davon %d mit Abweichung über 2, größte Abweichung %d, Summe %d",
		bestOX, bestOY, n, gross, maxAbw, besteSumme)
	// Wo liegen die Abweichungen über 2? Je Zeile gezählt, dazu ein Beispiel.
	zeilen := map[int]int{}
	beispiel := map[int]string{}
	for y := 0; y < b.Rect.Dy(); y++ {
		for x := 0; x < b.Rect.Dx(); x++ {
			p := b.NRGBAAt(x, y)
			q := image.Pt(bestOX+x, bestOY+y)
			if p.A != 255 || !q.In(img.Bounds()) {
				continue
			}
			c := rgb(q)
			if d := max(abs(c[0]-int(p.R)), abs(c[1]-int(p.G)), abs(c[2]-int(p.B))); d > 2 {
				zeilen[y]++
				if _, ok := beispiel[y]; !ok {
					beispiel[y] = "x=" + itoa(x) + " Editor " + itoa(c[0]) + "," + itoa(c[1]) + "," + itoa(c[2]) +
						" Renderer " + itoa(int(p.R)) + "," + itoa(int(p.G)) + "," + itoa(int(p.B))
				}
			}
		}
	}
	// Hypothese: Gio mischt linear. Über der Naht (Ansichtszeile = Originalzeile) ist das Grundpixel bekannt; aus dem
	// Ergebnis des Renderers (sRGB-Mischung mit Weiß) folgt die wirksame Deckung, daraus die lineare Mischung. Verglichen
	// wird sie mit dem Editor (Grünkanal, das Bild ist rot).
	o := g.Dokument.OriginalBild()
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	srgb := func(v float64) float64 {
		if v <= 0.0031308 {
			return 255 * v * 12.92
		}
		return 255 * (1.055*math.Pow(v, 1/2.4) - 0.055)
	}
	var nn int
	var fehlerR, fehlerLin float64
	for y := 266; y < 286; y++ {
		for x := 0; x < b.Rect.Dx(); x++ {
			p := b.NRGBAAt(x, y)
			q := image.Pt(bestOX+x, bestOY+y)
			if p.A != 255 {
				continue
			}
			base := float64(o.NRGBAAt(x, y).G)
			if base < 10 || float64(p.G) <= base+3 { // nur Pixel mit sichtbarem Fasersaum
				continue
			}
			e := float64(rgb(q)[1])
			a := (float64(p.G) - base) / (255 - base)
			erwartet := srgb(lin(base)*(1-a) + a)
			fehlerR += math.Abs(e - float64(p.G))
			fehlerLin += math.Abs(e - erwartet)
			nn++
		}
	}
	if nn > 0 {
		t.Logf("Fasersaum oben, %d Pixel: mittlere Abweichung des Editors vom Renderer %.2f, von der linearen Mischung %.2f",
			nn, fehlerR/float64(nn), fehlerLin/float64(nn))
	}
	for y := 0; y < b.Rect.Dy(); y++ {
		if zeilen[y] > 0 {
			t.Logf("Zeile %d: %d Pixel, %s", y, zeilen[y], beispiel[y])
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
