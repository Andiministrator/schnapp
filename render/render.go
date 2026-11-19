// SPDX-License-Identifier: GPL-3.0-or-later

// Package render setzt Ansichtsbild und Objekte eines Dokuments zu fertigen Pixeln zusammen (ADR-0020 Punkt 3):
// für das sichtbare Bild der Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Ohne GPU, ohne Gio. Maße wie im
// SVG-Export (fileformat/svg.go), damit Datei, Export und Bildschirm übereinstimmen: Pfeilspitze max(8, 4·Breite),
// Textzeilen im Abstand 1,3 × Schriftgrad, Nummer mit Schriftgrad 1,2 × Radius. Text wird wie im SVG-Export nicht
// umbrochen (F-70).
package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/Andiministrator/schnapp/document"
)

// Bild liefert die Ansicht des Dokuments mit allen Objekten. Objekte liegen im Original und werden über
// Dokument.ZuAnsicht abgebildet; was außerhalb der Ansicht liegt, fällt weg.
func Bild(d *document.Dokument) (*image.NRGBA, error) {
	b := d.Ansichtsbild()
	z := zeichner{bild: b, d: d, faces: map[faceSchluessel]font.Face{}}
	defer z.schliessen()
	for _, o := range d.Objekte() {
		if err := z.objekt(o); err != nil {
			return nil, err
		}
	}
	return b, nil
}

type zeichner struct {
	bild  *image.NRGBA
	d     *document.Dokument
	faces map[faceSchluessel]font.Face
}

func (z zeichner) schliessen() {
	for _, f := range z.faces {
		_ = f.Close()
	}
}

func (z zeichner) ansicht(p document.Punkt) document.Punkt { return z.d.ZuAnsicht(p) }

func (z zeichner) objekt(o document.Objekt) error {
	switch v := o.(type) {
	case document.Pfeil:
		const n = 48
		pts := make([]document.Punkt, 0, n+1)
		for i := 0; i <= n; i++ {
			pts = append(pts, z.ansicht(v.Punkt(float64(i)/n)))
		}
		z.strich(pts, false, v.Strich)
		r := v.RichtungAmEnde()
		g := math.Max(8, 4*v.Strich.Breite)
		nrm := document.Pt(-r.Y, r.X)
		fuss := v.B.Minus(r.Mal(g))
		z.fuelle([][]document.Punkt{{z.ansicht(v.B), z.ansicht(fuss.Plus(nrm.Mal(g / 2))),
			z.ansicht(fuss.Minus(nrm.Mal(g / 2)))}}, v.Strich.Farbe)
	case document.Kasten:
		r := document.R(z.ansicht(v.Bereich.Min), z.ansicht(v.Bereich.Max))
		umriss := rundesRechteck(r, v.Eckradius)
		if v.Fuellung.A > 0 {
			z.fuelle([][]document.Punkt{umriss}, v.Fuellung)
		}
		if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
			z.strich(umriss, true, v.Strich)
		}
	case document.Text:
		if v.Hintergrund.A > 0 {
			g := v.Grenzen()
			r := document.R(z.ansicht(g.Min), z.ansicht(g.Max))
			z.fuelle([][]document.Punkt{rundesRechteck(r, 0)}, v.Hintergrund)
		}
		face, err := z.schrift(false, v.Schriftgrad)
		if err != nil {
			return err
		}
		ort := z.ansicht(v.Ort)
		for i, zeile := range strings.Split(v.Inhalt, "\n") {
			// Tabulator als vier Leerzeichen, Wagenrücklauf weg: die Go-Schrift hat für beide keine Glyphe (Kritik A-3).
			zeile = document.Zeichenbar(zeile)
			z.text(face, zeile, ort.X, ort.Y+v.Schriftgrad*(1+1.3*float64(i)), v.Farbe, v.Schriftgrad)
		}
	case document.Nummer:
		m := z.ansicht(v.Mitte)
		z.fuelle([][]document.Punkt{kreis(m, v.Radius)}, v.Farbe)
		face, err := z.schrift(true, v.Radius*1.2)
		if err != nil {
			return err
		}
		s := strconv.Itoa(v.Wert)
		breite := float64(font.MeasureString(face, s)) / 64
		// Senkrecht mittig wie dominant-baseline="central": Grundlinie um die halbe Ziffernhöhe unter der Mitte.
		hoehe := float64(face.Metrics().CapHeight) / 64
		z.text(face, s, m.X-breite/2, m.Y+hoehe/2, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, v.Radius*1.2)
	}
	return nil
}

// fuelle füllt Polygone (Ansichtskoordinaten) mit Kantenglättung. Alle Polygone werden gleich ausgerichtet, damit
// Überlappungen sich addieren statt aufzuheben.
func (z zeichner) fuelle(polys [][]document.Punkt, farbe color.NRGBA) {
	if farbe.A == 0 {
		return
	}
	// Nur das umschließende Rechteck der Polygone rastern (sonst kostet jedes Objekt das ganze Bild).
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		for _, q := range p {
			minX, minY = math.Min(minX, q.X), math.Min(minY, q.Y)
			maxX, maxY = math.Max(maxX, q.X), math.Max(maxY, q.Y)
		}
	}
	if minX > maxX {
		return
	}
	ausschnitt := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1).
		Intersect(z.bild.Bounds())
	if ausschnitt.Empty() {
		return
	}
	ox, oy := float64(ausschnitt.Min.X), float64(ausschnitt.Min.Y)
	r := vector.NewRasterizer(ausschnitt.Dx(), ausschnitt.Dy())
	r.DrawOp = draw.Over
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		if flaeche(p) < 0 {
			p = umgekehrt(p)
		}
		r.MoveTo(float32(p[0].X-ox), float32(p[0].Y-oy))
		for _, q := range p[1:] {
			r.LineTo(float32(q.X-ox), float32(q.Y-oy))
		}
		r.ClosePath()
	}
	r.Draw(z.bild, ausschnitt, image.NewUniform(farbe), image.Point{})
}

// strich zieht einen Linienzug der Breite s.Breite mit runden Enden und Ecken: je Strecke ein Viereck, je Punkt ein
// Kreis, alles gleich ausgerichtet und in einem Durchgang gefüllt (keine doppelte Deckung an den Stößen).
func (z zeichner) strich(pts []document.Punkt, geschlossen bool, s document.Strich) {
	hw := s.Breite / 2
	if hw <= 0 || len(pts) == 0 {
		return
	}
	var polys [][]document.Punkt
	n := len(pts)
	strecken := n - 1
	if geschlossen {
		strecken = n
	}
	for i := 0; i < strecken; i++ {
		a, b := pts[i], pts[(i+1)%n]
		d := b.Minus(a)
		l := d.Laenge()
		if l < 1e-9 {
			continue
		}
		nrm := document.Pt(-d.Y/l*hw, d.X/l*hw)
		polys = append(polys, []document.Punkt{a.Plus(nrm), b.Plus(nrm), b.Minus(nrm), a.Minus(nrm)})
	}
	for _, p := range pts {
		polys = append(polys, kreis(p, hw))
	}
	z.fuelle(polys, s.Farbe)
}

// text zeichnet eine Zeile. Was das Bild nicht schneidet, wird nicht gerastert (F-118): eine Zeile, deren Höhe ganz
// über oder unter dem Bild liegt, entfällt, ebenso jede Glyphe, die um mehr als einen Schriftgrad links oder rechts
// vom Bild liegt (Vorschub und Unterschneidung laufen weiter, damit sichtbare Glyphen am selben Ort stehen).
func (z zeichner) text(face font.Face, s string, x, grundlinie float64, farbe color.NRGBA, grad float64) {
	rand := z.bild.Bounds()
	if grundlinie-grad*1.5 > float64(rand.Max.Y) || grundlinie+grad < float64(rand.Min.Y) {
		return
	}
	dr := font.Drawer{Dst: z.bild, Src: image.NewUniform(farbe), Face: face}
	dot := fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round(grundlinie * 64))}
	links, rechts := float64(rand.Min.X)-grad, float64(rand.Max.X)+grad
	vorher := rune(-1)
	for _, c := range s {
		if vorher >= 0 {
			dot.X += face.Kern(vorher, c)
		}
		vorher = c
		if px := float64(dot.X) / 64; px > rechts {
			return // alles Weitere liegt rechts vom Bild
		} else if px < links {
			if adv, ok := face.GlyphAdvance(c); ok {
				dot.X += adv
			}
			continue
		}
		dr.Dot = dot
		dr.DrawString(string(c))
		dot = dr.Dot
	}
}

// kreis nähert einen Kreis mit genug Ecken für glatte Ränder an.
func kreis(m document.Punkt, r float64) []document.Punkt {
	n := int(math.Min(256, math.Max(16, r*2)))
	pts := make([]document.Punkt, n)
	for i := range pts {
		w := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = document.Pt(m.X+r*math.Cos(w), m.Y+r*math.Sin(w))
	}
	return pts
}

// rundesRechteck liefert den Umriss eines Rechtecks mit Eckradius (auf die halbe kürzere Seite begrenzt).
func rundesRechteck(r document.Rechteck, radius float64) []document.Punkt {
	radius = math.Max(0, math.Min(radius, math.Min(r.Breite(), r.Hoehe())/2))
	if radius < 0.5 {
		return []document.Punkt{r.Min, document.Pt(r.Max.X, r.Min.Y), r.Max, document.Pt(r.Min.X, r.Max.Y)}
	}
	ecken := []struct {
		m document.Punkt
		w float64
	}{
		{document.Pt(r.Max.X-radius, r.Min.Y+radius), -math.Pi / 2},
		{document.Pt(r.Max.X-radius, r.Max.Y-radius), 0},
		{document.Pt(r.Min.X+radius, r.Max.Y-radius), math.Pi / 2},
		{document.Pt(r.Min.X+radius, r.Min.Y+radius), math.Pi},
	}
	const schritte = 12
	var pts []document.Punkt
	for _, e := range ecken {
		for i := 0; i <= schritte; i++ {
			w := e.w + math.Pi/2*float64(i)/schritte
			pts = append(pts, document.Pt(e.m.X+radius*math.Cos(w), e.m.Y+radius*math.Sin(w)))
		}
	}
	return pts
}

func flaeche(p []document.Punkt) float64 {
	var s float64
	for i := range p {
		q := p[(i+1)%len(p)]
		s += p[i].X*q.Y - q.X*p[i].Y
	}
	return s / 2
}

func umgekehrt(p []document.Punkt) []document.Punkt {
	r := make([]document.Punkt, len(p))
	for i := range p {
		r[i] = p[len(p)-1-i]
	}
	return r
}

// Schriften: Go Regular und Go Bold (BSD-Lizenz, in golang.org/x/image), einmal geparst; sfnt-Fonts sind
// nebenläufig nutzbar. Faces sind es nicht — jeder Aufruf von Bild hält deshalb seine eigenen (zeichner.faces).
var (
	schriftEinmal sync.Once
	schriften     [2]*opentype.Font // 0 normal, 1 fett
	schriftFehler error
)

type faceSchluessel struct {
	fett bool
	grad float64
}

func (z zeichner) schrift(fett bool, grad float64) (font.Face, error) {
	schriftEinmal.Do(func() {
		for i, roh := range [][]byte{goregular.TTF, gobold.TTF} {
			if schriften[i], schriftFehler = opentype.Parse(roh); schriftFehler != nil {
				return
			}
		}
	})
	if schriftFehler != nil {
		return nil, schriftFehler
	}
	k := faceSchluessel{fett, grad}
	if f, ok := z.faces[k]; ok {
		return f, nil
	}
	i := 0
	if fett {
		i = 1
	}
	face, err := opentype.NewFace(schriften[i], &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	z.faces[k] = face
	return face, nil
}
