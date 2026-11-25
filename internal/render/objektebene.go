// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"runtime"
	"sync"

	"golang.org/x/image/font"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/raster16"
)

// Objektebene rastert die sichtbaren Objekte von d auf einen durchsichtigen Grund in der Größe der Ansicht: dieselben
// Pixel, die Bild über das Ansichtsbild mischt, aber ohne Bild darunter und ohne die Kanten der Nähte. Der Editor zeigt
// sie statt seiner eigenen Zeichnung, wenn er über vielen Nähten zu viele Bereiche abspielen müsste (ADR-0023
// Nachtrag 20, F-787). Gerastert wird in waagerechten Bändern auf bis zu acht Kernen; jedes Band hat seinen eigenen
// Zeichner und schreibt nur in seine Zeilen (alle Füllungen, Glyphen, Lupen und Ebenen sind auf z.rand bzw. die Fläche
// von z.bild beschnitten). Kosten werden nicht gezählt; über dem Budget rastert sie trotzdem wie Bild.
func Objektebene(d *document.Dokument) (*image.NRGBA, error) {
	return objektebene(d, min(runtime.GOMAXPROCS(0), 8))
}

// objektebene ist Objektebene mit höchstens n Bändern (mindestens 32 Zeilen hoch); ein Test vergleicht n = 1.
func objektebene(d *document.Dokument, n int) (*image.NRGBA, error) {
	g := d.Groesse()
	b := image.NewNRGBA(image.Rectangle{Max: g})
	if g.X <= 0 || g.Y <= 0 {
		return b, nil
	}
	n = max(1, min(n, g.Y/32))
	var wg sync.WaitGroup
	fehler := make([]error, n)
	for i := 0; i < n; i++ {
		band := image.Rect(0, g.Y*i/n, g.X, g.Y*(i+1)/n)
		wg.Add(1)
		go func(i int, band image.Rectangle) {
			defer wg.Done()
			z := zeichner{bild: b.SubImage(band).(*image.NRGBA), rand: band, d: d, faces: map[faceSchluessel]font.Face{},
				auf: &aufwand{}, ras: &raster16.Rasterizer{}, masken: neuerMaskenCache(), ohneZaehlen: true}
			defer z.schliessen()
			fehler[i] = z.objekte()
		}(i, band)
	}
	wg.Wait()
	for _, err := range fehler {
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}
