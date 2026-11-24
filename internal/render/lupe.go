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
// Das Bild ist die Ansicht unter der Quelle (document.LupenQuelle), mit dem nächsten Nachbarn vergrößert, nur soweit die
// Ansicht sie zeigt; Beschriftungen unter der Quelle gehören nicht dazu. Renderer und Editor (LupenInhalt) rechnen dieselben
// Pixel.

// lupe zeichnet v in der Ansicht.
func (z zeichner) lupe(v document.Lupe) {
	strich := v.Strich.Breite > 0 && v.Strich.Farbe.A > 0
	if strich {
		if a, b, ok := v.Linie(); ok {
			// Je sichtbarem Bereich und auf ihn beschnitten, wie SVG-Export und Treffer (F-695).
			z.jeStueck(huelleVon([]document.Punkt{a, b}, v.Strich.Breite/2+1), func(zs zeichner, abb func(document.Punkt) document.Punkt) {
				zs.strich([]document.Punkt{abb(a), abb(b)}, false, v.Strich)
			})
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

// lupenBild mischt das Bild im Ziel in die Ansicht. Kosten wie eine Füllung der aufs Bild beschnittenen Zielfläche
// (ADR-0033 Punkt 7) und dazu wie eine Füllung der aufs Original begrenzten Quellfläche, weil der SVG-Export bis zu so
// viele Pixel einbettet (Nachtrag 2 Punkt 3); beides auch im Trockenlauf.
func (z zeichner) lupenBild(v document.Lupe) {
	r := lupenRahmen(z.d, v).Intersect(z.rand)
	if r.Empty() {
		return // ohne Ziel in der Ansicht bettet auch der SVG-Export kein Bild ein
	}
	if !z.ohneZaehlen {
		z.auf.flaeche += fuellGewicht * float64(r.Dx()) * float64(r.Dy())
		z.auf.flaeche += fuellGewicht * quellFlaeche(z.d, v)
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

// quellFlaeche ist die Fläche der Quelle von l, aufs Original begrenzt (Pixel).
func quellFlaeche(d *document.Dokument, l document.Lupe) float64 {
	g := d.OriginalGroesse()
	w := math.Min(l.Quelle.Max.X, float64(g.X)) - math.Max(l.Quelle.Min.X, 0)
	h := math.Min(l.Quelle.Max.Y, float64(g.Y)) - math.Max(l.Quelle.Min.Y, 0)
	return math.Max(0, w) * math.Max(0, h)
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

// lupenPixel ruft f für jedes Pixel (x, y) von r, das ein Pixel q (NRGBA, vier Bytes) des Originals zeigt. spalte und
// zeile liefern den Mittelpunkt einer Spalte bzw. Zeile von r im Original; welches Pixel er zeigt, entscheidet
// document.LupenQuelle (eine Rechnung für Renderer, Editor und SVG-Export, ADR-0033 Nachtrag 2 Punkt 1).
func lupenPixel(d *document.Dokument, l document.Lupe, r image.Rectangle, spalte, zeile func(int) float64,
	f func(x, y int, q []byte)) {
	d.LupenQuelle(l).Abtasten(r.Dx(), r.Dy(),
		func(i int) float64 { return spalte(r.Min.X + i) },
		func(j int) float64 { return zeile(r.Min.Y + j) },
		func(i, j int, q []byte) { f(r.Min.X+i, r.Min.Y+j, q) },
		func(j int) {
			if j%64 == 0 {
				saveprogress.Puls()
			}
		})
}

// LupenInhalt liefert das Bild im Ziel der Lupe l, wie Bild es zeichnet: in Ansichtskoordinaten (die Grenzen des
// Ergebnisses), auf die Ansicht begrenzt, Unsichtbares (vor dem Zuschnitt, im Spalt einer Naht) durchsichtig. Der
// Editor zeichnet es mit dem Filter „nächster Nachbar“, so zeigen Leinwand und Datei dieselben Pixel (ADR-0033 Punkt
// 3). nil, wenn das Ziel nicht in die Ansicht reicht.
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

// LupenVorschau liefert das Bild im Ziel von l, ohne die Lage des Ziels in der Ansicht abzubilden, für die Vorschau der
// Ebenenliste: höchstens seite Pixel je Seite, das Ziel gleichmäßig abgetastet; Grenzen ab (0, 0). Unsichtbares bleibt
// auch hier durchsichtig (ADR-0033 Punkt 2).
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
