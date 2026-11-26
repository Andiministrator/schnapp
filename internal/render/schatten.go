// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/font"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/raster16"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Schatten einer Ebene (ADR-0021 Nachtrag 26 Punkt 3): Die Deckung des Objekts im Zwischenbild wird um ganze Pixel der
// Ansicht versetzt, mit drei Kastenfiltern je Richtung weichgezeichnet (Näherung der Gauß-Unschärfe mit σ = Weich/2) und
// in der Schattenfarbe unter das Objekt gelegt; Schatten und Objekt zusammen mischt ebene mit der Deckkraft der Ebene ein.

// schattenGewicht: Ein Schatten kostet je Pixel seiner Fläche so viel wie so viele Füllungen — Zwischenbild, sechs Läufe
// des Kastenfilters, Färben und Übereinanderlegen (Kostenmodell wie fuellGewicht).
const schattenGewicht = 10.0

// schattenVersatz ist der Versatz des Schattens in ganzen Pixeln der Ansicht.
func schattenVersatz(s document.Schatten) image.Point {
	v := s.Versatz()
	return image.Pt(int(math.Round(v.X)), int(math.Round(v.Y)))
}

// schattenRahmen ist die Fläche, die Objekt (Rahmen r) und Schatten s zusammen decken, aufs Bild begrenzt.
func schattenRahmen(r image.Rectangle, s document.Schatten, rand image.Rectangle) image.Rectangle {
	rad := int(math.Ceil(1.5*s.Weich)) + 1
	sr := r.Add(schattenVersatz(s)).Inset(-rad)
	return r.Union(sr).Intersect(rand)
}

// schattenUnter legt unter das gezeichnete Objekt obj (Zwischenbild, Fläche obj.Rect) seinen Schatten s und liefert beides
// zusammen auf der Fläche ziel; mitObjekt false liefert nur den Schatten (Editor).
func schattenUnter(obj *image.NRGBA, s document.Schatten, ziel image.Rectangle, mitObjekt bool) *image.NRGBA {
	out := image.NewNRGBA(ziel)
	w, h := ziel.Dx(), ziel.Dy()
	if w <= 0 || h <= 0 {
		return out
	}
	maske := make([]float32, w*h)
	v := schattenVersatz(s)
	quelle := obj.Rect.Add(v).Intersect(ziel)
	for y := quelle.Min.Y; y < quelle.Max.Y; y++ {
		qi := obj.PixOffset(quelle.Min.X-v.X, y-v.Y)
		mi := (y-ziel.Min.Y)*w + quelle.Min.X - ziel.Min.X
		for x := 0; x < quelle.Dx(); x++ {
			maske[mi+x] = float32(obj.Pix[qi+4*x+3]) / 255
		}
	}
	weichzeichnen(maske, w, h, s.Weich/2)
	a := float32(s.Farbe.A)
	for y := 0; y < h; y++ {
		saveprogress.Puls()
		oi := out.PixOffset(ziel.Min.X, ziel.Min.Y+y)
		for x := 0; x < w; x++ {
			m := maske[y*w+x] * a
			if m < 0.5 {
				continue
			}
			p := out.Pix[oi+4*x : oi+4*x+4 : oi+4*x+4]
			p[0], p[1], p[2], p[3] = s.Farbe.R, s.Farbe.G, s.Farbe.B, uint8(min(255, m+0.5))
		}
	}
	if mitObjekt {
		draw.Draw(out, obj.Rect, obj, obj.Rect.Min, draw.Over)
	}
	return out
}

// kastenBreiten sind die Breiten dreier Kastenfilter, deren Folge eine Gauß-Unschärfe mit σ annähert (Kovarianz gleich,
// ungerade Breiten).
func kastenBreiten(sigma float64) [3]int {
	const n = 3
	ideal := math.Sqrt(12*sigma*sigma/n + 1)
	wl := int(math.Floor(ideal))
	if wl%2 == 0 {
		wl--
	}
	wl = max(wl, 1)
	wu := wl + 2
	m := int(math.Round((12*sigma*sigma - n*float64(wl*wl) - 4*n*float64(wl) - 3*n) / (-4*float64(wl) - 4)))
	var b [3]int
	for i := range b {
		if i < m {
			b[i] = wl
		} else {
			b[i] = wu
		}
	}
	return b
}

// weichzeichnen zeichnet die Maske m (Breite w, Höhe h) mit σ weich; außerhalb gilt 0.
func weichzeichnen(m []float32, w, h int, sigma float64) {
	if sigma < 0.25 {
		return
	}
	puffer := make([]float32, max(w, h))
	for _, b := range kastenBreiten(sigma) {
		r := b / 2
		if r == 0 {
			continue
		}
		for y := 0; y < h; y++ {
			kasten(m[y*w:(y+1)*w], 1, w, r, puffer)
		}
		saveprogress.Puls()
		for x := 0; x < w; x++ {
			kasten(m[x:], w, h, r, puffer)
		}
		saveprogress.Puls()
	}
}

// kasten mittelt n Werte von z (Abstand schritt) über das Fenster ±r, außerhalb 0, mit laufender Summe.
func kasten(z []float32, schritt, n, r int, puffer []float32) {
	for i := range n {
		puffer[i] = z[i*schritt]
	}
	var summe float32
	for i := 0; i <= min(r, n-1); i++ {
		summe += puffer[i]
	}
	f := 1 / float32(2*r+1)
	for i := range n {
		z[i*schritt] = summe * f
		if j := i + r + 1; j < n {
			summe += puffer[j]
		}
		if j := i - r; j >= 0 {
			summe -= puffer[j]
		}
	}
}

// Schattenbild ist der Schatten von o in der Ansicht von d ohne das Objekt selbst (für den Editor, ADR-0021 Nachtrag 26
// Punkt 4), mit der Fläche des Bilds in Ansichtspixeln; nil ohne sichtbaren Schatten.
func Schattenbild(d *document.Dokument, o document.Objekt) (*image.NRGBA, error) {
	s := o.Ebenenwerte().Schatten
	if !s.Sichtbar() {
		return nil, nil
	}
	rand := image.Rectangle{Max: d.Groesse()}
	z := zeichner{rand: rand, d: d, faces: map[faceSchluessel]font.Face{}, auf: &aufwand{}, ras: &raster16.Rasterizer{},
		masken: neuerMaskenCache(), ohneZaehlen: true}
	defer z.schliessen()
	r := z.ebenenRahmen(o)
	if r.Empty() {
		return nil, nil
	}
	z.bild = image.NewNRGBA(r)
	if err := z.objekt(o); err != nil {
		return nil, err
	}
	return schattenUnter(z.bild, s, schattenRahmen(r, s, rand), false), nil
}

// schattenEbene zeichnet o mit Schatten und mischt beides mit der Deckkraft p ein (ebene).
func (z zeichner) schattenEbene(o document.Objekt, r image.Rectangle, p int) error {
	s := o.Ebenenwerte().Schatten
	ziel := schattenRahmen(r, s, z.rand)
	if !z.ohneZaehlen {
		z.auf.flaeche += fuellGewicht * schattenGewicht * float64(ziel.Dx()) * float64(ziel.Dy())
	}
	if z.trocken() {
		return z.objekt(o)
	}
	zo := z
	zo.bild = image.NewNRGBA(r)
	if err := zo.objekt(o); err != nil {
		return err
	}
	mischeEbene(z.bild, schattenUnter(zo.bild, s, ziel, true), p)
	return nil
}
