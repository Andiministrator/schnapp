// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Lupe (ADR-0033): Verbindungslinie, Rahmen um die Quelle, Bild im Ziel, Rahmen um das Ziel, in dieser Reihenfolge.
// Das Bild sind die Pixel des Originals im Quellbereich, mit dem nächsten Nachbarn vergrößert, nur soweit die Ansicht
// sie zeigt; Beschriftungen unter der Quelle gehören nicht dazu. Renderer und Editor (LupenInhalt) rechnen dieselben
// Pixel.

// lupe zeichnet v in der Ansicht.
func (z zeichner) lupe(v document.Lupe) {
	strich := v.Strich.Breite > 0 && v.Strich.Farbe.A > 0
	if strich {
		if a, b, ok := v.Linie(); ok {
			z.strich([]document.Punkt{z.ansicht(a), z.ansicht(b)}, false, v.Strich)
		}
		z.eckigerRahmen(v.Quelle, v.Strich)
	}
	z.lupenBild(v)
	if strich {
		z.eckigerRahmen(v.Ziel, v.Strich)
	}
}

// eckigerRahmen zeichnet den Rahmen um r (Original) wie den Strich eines Kastens mit Eckradius 0: mittig auf der Kante,
// spitze Außenecken (document.EckigerRahmen, F-498).
func (z zeichner) eckigerRahmen(r document.Rechteck, s document.Strich) {
	q := document.R(z.ansicht(r.Min), z.ansicht(r.Max))
	var polys [][]document.Punkt
	for _, e := range document.EckigerRahmen(q, s.Breite) {
		polys = append(polys, []document.Punkt{e.Min, document.Pt(e.Max.X, e.Min.Y), e.Max, document.Pt(e.Min.X, e.Max.Y)})
	}
	z.fuelle(polys, nil, s.Farbe)
}

// lupenRahmen ist das Rechteck der Ansichtspixel, die das Ziel von l bedecken kann (nicht beschnitten).
func lupenRahmen(d *document.Dokument, l document.Lupe) image.Rectangle {
	a, b := d.ZuAnsicht(l.Ziel.Min), d.ZuAnsicht(l.Ziel.Max)
	return image.Rect(int(math.Floor(a.X)), int(math.Floor(a.Y)), int(math.Ceil(b.X)), int(math.Ceil(b.Y)))
}

// lupenBild mischt das Bild im Ziel in die Ansicht. Kosten wie eine Füllung der aufs Bild beschnittenen Fläche
// (ADR-0033 Punkt 7), auch im Trockenlauf.
func (z zeichner) lupenBild(v document.Lupe) {
	r := lupenRahmen(z.d, v).Intersect(z.rand)
	if r.Empty() {
		return
	}
	if !z.ohneZaehlen {
		z.auf.flaeche += fuellGewicht * float64(r.Dx()) * float64(r.Dy())
	}
	if z.trocken() {
		return
	}
	r = r.Intersect(z.bild.Rect) // im Zwischenbild der Gruppendeckkraft nur dessen Fläche
	b := z.bild
	lupenPixel(z.d, v, r, ansichtsMitte(z.d, false), ansichtsMitte(z.d, true), func(x, y int, q []byte) {
		if q[3] == 0 {
			return
		}
		e := neuerEinmischer(b, color.NRGBA{R: q[0], G: q[1], B: q[2], A: q[3]})
		i := b.PixOffset(x, y)
		e.misch(b.Pix[i:i+4:i+4], 0xffff) // wie mischeEbene bei voller Deckkraft
	})
}

// ansichtsMitte liefert je Ansichtsspalte (bzw. -zeile) die Lage ihres Mittelpunkts im Original (Dokument.ZuOriginal,
// ADR-0033 Punkt 4).
func ansichtsMitte(d *document.Dokument, zeilen bool) func(int) float64 {
	return func(v int) float64 {
		if zeilen {
			return d.ZuOriginal(document.Pt(0, float64(v)+0.5)).Y
		}
		return d.ZuOriginal(document.Pt(float64(v)+0.5, 0)).X
	}
}

// lupenPixel ruft f für jedes Pixel (x, y) von r, das ein sichtbares Pixel q (NRGBA, vier Bytes) des Originals zeigt.
// spalte und zeile liefern den Mittelpunkt einer Spalte bzw. Zeile von r im Original; liegt er nicht im Ziel, oder ist
// das Quellpixel außerhalb des Bilds, des Zuschnitts oder in einem Stauch-Streifen, bleibt das Pixel leer.
func lupenPixel(d *document.Dokument, l document.Lupe, r image.Rectangle, spalte, zeile func(int) float64,
	f func(x, y int, q []byte)) {
	orig := d.OriginalBild()
	quelle := func(o, von, bis float64, nimm func(float64) int, sichtbar func(int) bool) int {
		if !(o >= von && o < bis) {
			return -1
		}
		if c := nimm(o); sichtbar(c) {
			return c
		}
		return -1
	}
	sx := make([]int, r.Dx())
	for i := range sx {
		sx[i] = quelle(spalte(r.Min.X+i), l.Ziel.Min.X, l.Ziel.Max.X, l.QuellSpalte, d.SpalteSichtbar)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if (y-r.Min.Y)%64 == 0 {
			saveprogress.Puls()
		}
		sy := quelle(zeile(y), l.Ziel.Min.Y, l.Ziel.Max.Y, l.QuellZeile, d.ZeileSichtbar)
		if sy < 0 {
			continue
		}
		for i, c := range sx {
			if c < 0 {
				continue
			}
			o := orig.PixOffset(c, sy)
			f(r.Min.X+i, y, orig.Pix[o:o+4:o+4])
		}
	}
}

// LupenInhalt liefert das Bild im Ziel der Lupe l, wie Bild es zeichnet: in Ansichtskoordinaten (die Grenzen des
// Ergebnisses), auf die Ansicht begrenzt, unsichtbare Quellpixel durchsichtig. Der Editor zeichnet es mit dem Filter
// „nächster Nachbar“, so zeigen Leinwand und Datei dieselben Pixel (ADR-0033 Punkt 3). nil, wenn das Ziel nicht in die
// Ansicht reicht.
func LupenInhalt(d *document.Dokument, l document.Lupe) *image.NRGBA {
	r := lupenRahmen(d, l).Intersect(image.Rectangle{Max: d.Groesse()})
	if r.Empty() {
		return nil
	}
	aus := image.NewNRGBA(r)
	lupenPixel(d, l, r, ansichtsMitte(d, false), ansichtsMitte(d, true), func(x, y int, q []byte) {
		copy(aus.Pix[aus.PixOffset(x, y):], q)
	})
	return aus
}

// LupenVorschau liefert das Bild im Ziel von l ohne Zuschnitt und Stauchen der Ansicht abgebildet, für die Vorschau der
// Ebenenliste: höchstens seite Pixel je Seite, das Ziel gleichmäßig abgetastet; Grenzen ab (0, 0). Unsichtbare
// Quellpixel bleiben auch hier durchsichtig (ADR-0033 Punkt 2).
func LupenVorschau(d *document.Dokument, l document.Lupe, seite int) *image.NRGBA {
	n := max(1, min(seite, int(math.Ceil(l.Ziel.Breite()))))
	m := max(1, min(seite, int(math.Ceil(l.Ziel.Hoehe()))))
	aus := image.NewNRGBA(image.Rect(0, 0, n, m))
	lupenPixel(d, l, aus.Rect,
		func(x int) float64 { return l.Ziel.Min.X + (float64(x)+0.5)*l.Ziel.Breite()/float64(n) },
		func(y int) float64 { return l.Ziel.Min.Y + (float64(y)+0.5)*l.Ziel.Hoehe()/float64(m) },
		func(x, y int, q []byte) { copy(aus.Pix[aus.PixOffset(x, y):], q) })
	return aus
}
