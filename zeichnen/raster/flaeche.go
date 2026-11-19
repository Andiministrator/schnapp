// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"math"
	"sort"

	"github.com/Andiministrator/schnapp/zeichnen/geo"
)

// Unterzeilen je Bildzeile. Waagerecht wird die Deckung exakt gerechnet, senkrecht an Unterzeilen abgetastet; so
// arbeitet auch der Scan-Konverter von Cairo ("tor", 15 Unterzeilen), den rsvg-convert nutzt. Mit derselben Zahl
// liegen die Mischpixel an Kanten am nächsten an rsvg-convert.
const Unterzeilen = 15

// Raster der Koordinaten (Cairo rechnet in 24.8-Festkomma, also auf 1/256 Pixel).
const festkomma = 256.0

// Maske: Deckung 0..1 je Pixel in einem Ausschnitt des Bildes (X0, Y0 linke obere Ecke, B x H Pixel).
type Maske struct {
	X0, Y0, B, H int
	A            []float32
}

type kante struct {
	x0, y0, y1, dxdy float64
	richtung         int
	sAnfang, sEnde   int
}

type kreuzung struct {
	x        float64
	richtung int
}

func fest(v float64) float64 { return math.Round(v*festkomma) / festkomma }

// Fuellen rastert geschlossene Polygone (Gerätekoordinaten) mit Kantenglättung; evenodd wählt die Füllregel,
// sonst nonzero. bildB, bildH begrenzen den Ausschnitt. nil, wenn nichts im Bild liegt.
func Fuellen(polys [][]geo.P, evenodd bool, bildB, bildH int) *Maske {
	if m := kasten(polys, bildB, bildH); m != nil {
		return m
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	var kanten []kante
	for _, poly := range polys {
		n := len(poly)
		if n < 2 {
			continue
		}
		for i := 0; i < n; i++ {
			p, q := poly[i], poly[(i+1)%n]
			px, py, qx, qy := fest(p.X), fest(p.Y), fest(q.X), fest(q.Y)
			minX, maxX = math.Min(minX, px), math.Max(maxX, px)
			minY, maxY = math.Min(minY, py), math.Max(maxY, py)
			if py == qy {
				continue
			}
			r := 1
			if py > qy {
				px, py, qx, qy = qx, qy, px, py
				r = -1
			}
			k := kante{x0: px, y0: py, y1: qy, dxdy: (qx - px) / (qy - py), richtung: r}
			k.sAnfang = int(math.Ceil(py*Unterzeilen - 0.5))
			k.sEnde = int(math.Ceil(qy*Unterzeilen - 0.5))
			if k.sAnfang >= k.sEnde {
				continue
			}
			kanten = append(kanten, k)
		}
	}
	if len(kanten) == 0 {
		return nil
	}
	x0 := max(0, int(math.Floor(minX)))
	x1 := min(bildB, int(math.Ceil(maxX)))
	y0 := max(0, int(math.Floor(minY)))
	y1 := min(bildH, int(math.Ceil(maxY)))
	if x0 >= x1 || y0 >= y1 {
		return nil
	}
	m := &Maske{X0: x0, Y0: y0, B: x1 - x0, H: y1 - y0}
	m.A = make([]float32, m.B*m.H)
	sort.Slice(kanten, func(i, j int) bool { return kanten[i].sAnfang < kanten[j].sAnfang })

	direkt := make([]float64, m.B+1)
	diff := make([]float64, m.B+2)
	var aktiv []*kante
	var kr []kreuzung
	naechste := 0
	const anteil = 1.0 / Unterzeilen
	links, rechts := float64(x0), float64(x1)
	spanne := func(a, b float64) {
		a, b = math.Max(a, links), math.Min(b, rechts)
		if b <= a {
			return
		}
		a -= links
		b -= links
		ia, ib := int(a), int(b)
		if ia == ib {
			direkt[ia] += (b - a) * anteil
			return
		}
		direkt[ia] += (float64(ia+1) - a) * anteil
		diff[ia+1] += anteil
		diff[ib] -= anteil
		if ib < m.B {
			direkt[ib] += (b - float64(ib)) * anteil
		}
	}
	for zeile := y0; zeile < y1; zeile++ {
		for k := 0; k < Unterzeilen; k++ {
			s := zeile*Unterzeilen + k
			for naechste < len(kanten) && kanten[naechste].sAnfang <= s {
				aktiv = append(aktiv, &kanten[naechste])
				naechste++
			}
			behalten := aktiv[:0]
			for _, e := range aktiv {
				if e.sEnde > s {
					behalten = append(behalten, e)
				}
			}
			aktiv = behalten
			if len(aktiv) == 0 {
				continue
			}
			ys := (float64(s) + 0.5) / Unterzeilen
			kr = kr[:0]
			for _, e := range aktiv {
				if e.sAnfang > s {
					continue
				}
				kr = append(kr, kreuzung{x: e.x0 + (ys-e.y0)*e.dxdy, richtung: e.richtung})
			}
			sort.Slice(kr, func(i, j int) bool { return kr[i].x < kr[j].x })
			w := 0
			for i, c := range kr {
				w += c.richtung
				innen := w != 0
				if evenodd {
					innen = w%2 != 0
				}
				if innen && i+1 < len(kr) {
					spanne(c.x, kr[i+1].x)
				}
			}
		}
		zeilenAnfang := (zeile - y0) * m.B
		summe := 0.0
		for x := 0; x < m.B; x++ {
			summe += diff[x]
			v := summe + direkt[x]
			if v > 1 {
				v = 1
			} else if v < 0 {
				v = 0
			}
			m.A[zeilenAnfang+x] = float32(v)
			direkt[x], diff[x] = 0, 0
		}
		direkt[m.B], diff[m.B], diff[m.B+1] = 0, 0, 0
	}
	return m
}

// kasten: ein achsparalleles Rechteck exakt decken, auch senkrecht (Cairo nimmt dafür ebenfalls einen eigenen,
// exakten Weg; sonst weicht eine waagerechte Kante zwischen zwei Unterzeilen um bis zu 1/30 ab). nil, wenn die
// Polygone kein solches Rechteck sind oder nichts im Bild liegt.
func kasten(polys [][]geo.P, bildB, bildH int) *Maske {
	if len(polys) != 1 {
		return nil
	}
	p := polys[0]
	if len(p) < 4 {
		return nil
	}
	// nur achsparallele Stücke; Zwischenpunkte zerlegter Kanten liegen auf dem Rand des Rahmens (ein
	// zerlegtes rect hat viele x- und y-Werte); Fläche = Rechteck, ein Hin und Zurück ohne Fläche fällt durch
	ax, bx, ay, by := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for i, q := range p {
		r := p[(i+1)%len(p)]
		if fest(q.X) != fest(r.X) && fest(q.Y) != fest(r.Y) {
			return nil
		}
		ax, bx = math.Min(ax, fest(q.X)), math.Max(bx, fest(q.X))
		ay, by = math.Min(ay, fest(q.Y)), math.Max(by, fest(q.Y))
	}
	for _, q := range p {
		x, y := fest(q.X), fest(q.Y)
		if x != ax && x != bx && y != ay && y != by {
			return nil
		}
	}
	gerastert := make([]geo.P, len(p))
	for i, q := range p {
		gerastert[i] = geo.P{X: fest(q.X), Y: fest(q.Y)}
	}
	if math.Abs(math.Abs(flaeche(gerastert))-(bx-ax)*(by-ay)) > 1e-9*(1+(bx-ax)*(by-ay)) {
		return nil
	}
	x0, x1 := max(0, int(math.Floor(ax))), min(bildB, int(math.Ceil(bx)))
	y0, y1 := max(0, int(math.Floor(ay))), min(bildH, int(math.Ceil(by)))
	if x0 >= x1 || y0 >= y1 {
		return &Maske{}
	}
	m := &Maske{X0: x0, Y0: y0, B: x1 - x0, H: y1 - y0}
	m.A = make([]float32, m.B*m.H)
	ueber := func(a, b, lo float64) float64 { return math.Max(0, math.Min(b, lo+1)-math.Max(a, lo)) }
	for y := 0; y < m.H; y++ {
		fy := ueber(ay, by, float64(y0+y))
		for x := 0; x < m.B; x++ {
			m.A[y*m.B+x] = float32(fy * ueber(ax, bx, float64(x0+x)))
		}
	}
	return m
}
