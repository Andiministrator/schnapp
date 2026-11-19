// SPDX-License-Identifier: GPL-3.0-or-later

// Package auswahl ist die Logik des Vollbild-Overlays (Karte #34, ADR-0017): Abbildung zwischen eingefrorenem Bild
// und Fenster, Bereich aufziehen, Monitor per Klick, Lupe, Maßanzeige, letzter Bereich. Das Paket kennt Gio nicht;
// die Oberfläche (Paket oberflaeche) übersetzt Ereignisse und zeichnet nur.
//
// Koordinaten: Fensterpunkte in Pixeln des Fensterpuffers (float32, wie Gio sie liefert), Bildpunkte in Pixeln des
// aufgenommenen Bildes. Mit dem Gio-Patch (ADR-0015) ist der Puffer so groß wie der Monitor in echten Pixeln, und
// KWin liefert das Bild in derselben Auflösung (native-resolution) — dann ist die Abbildung 1:1.
package auswahl

import (
	"image"
	"math"
)

// Punkt ist ein Punkt in Fenster- oder Bildkoordinaten.
type Punkt struct{ X, Y float32 }

// Abbildung bildet Bildpunkte auf Fensterpunkte ab: fenster = bild·Zoom + Versatz.
type Abbildung struct {
	Zoom    float32
	Versatz Punkt
}

// NeueAbbildung zeigt ein Bild der Größe bild in einem Fenster der Größe fenster: 1:1 ab der Ecke, wenn beide
// gleich groß sind (pixelgenau), sonst eingepasst und mittig. Ein leeres Fenster oder Bild ergibt Zoom 1.
func NeueAbbildung(bild, fenster image.Point) Abbildung {
	if bild == fenster || bild.X <= 0 || bild.Y <= 0 || fenster.X <= 0 || fenster.Y <= 0 {
		return Abbildung{Zoom: 1}
	}
	bx, by := float64(bild.X), float64(bild.Y)
	fx, fy := float64(fenster.X), float64(fenster.Y)
	z := math.Min(fx/bx, fy/by)
	return Abbildung{
		Zoom:    float32(z),
		Versatz: Punkt{float32((fx - bx*z) / 2), float32((fy - by*z) / 2)},
	}
}

// ZuBild rechnet einen Fensterpunkt in Bildkoordinaten um.
func (a Abbildung) ZuBild(p Punkt) Punkt {
	return Punkt{(p.X - a.Versatz.X) / a.Zoom, (p.Y - a.Versatz.Y) / a.Zoom}
}

// ZuFenster rechnet einen Bildpunkt in Fensterkoordinaten um.
func (a Abbildung) ZuFenster(p Punkt) Punkt {
	return Punkt{p.X*a.Zoom + a.Versatz.X, p.Y*a.Zoom + a.Versatz.Y}
}

// Pixel ist das Bildpixel unter einem Fensterpunkt (abgerundet), auf das Bild der Größe groesse beschnitten.
func (a Abbildung) Pixel(p Punkt, groesse image.Point) image.Point {
	b := a.ZuBild(p)
	return image.Pt(
		klemme(int(math.Floor(float64(b.X))), 0, groesse.X-1),
		klemme(int(math.Floor(float64(b.Y))), 0, groesse.Y-1),
	)
}

// RechteckImFenster rechnet ein Bildrechteck in ein Fensterrechteck um (für das Zeichnen).
func (a Abbildung) RechteckImFenster(r image.Rectangle) image.Rectangle {
	p := a.ZuFenster(Punkt{float32(r.Min.X), float32(r.Min.Y)})
	q := a.ZuFenster(Punkt{float32(r.Max.X), float32(r.Max.Y)})
	return image.Rect(int(math.Round(float64(p.X))), int(math.Round(float64(p.Y))),
		int(math.Round(float64(q.X))), int(math.Round(float64(q.Y))))
}

func klemme(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}
