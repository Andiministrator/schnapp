// SPDX-License-Identifier: GPL-3.0-or-later

package auswahl

import (
	"fmt"
	"image"
)

// Lupe beschreibt die Vergrößerung um den Zeiger: Quelle sind Bildpixel (kann über den Bildrand ragen, dort zeichnet
// die Oberfläche Grund), Ziel ist der Platz im Fenster. Mitte ist das Pixel unter dem Zeiger.
type Lupe struct {
	Quelle image.Rectangle
	Ziel   image.Rectangle
	Mitte  image.Point
	Faktor int // Fensterpixel je Bildpixel
}

const (
	// LupePixel ist die Kantenlänge der Lupe in Bildpixeln (ungerade, damit es ein Mittelpixel gibt).
	LupePixel = 17
	// LupeFaktor ist die Vergrößerung.
	LupeFaktor = 8
	// lupeAbstand: Abstand der Lupe vom Zeiger in Fensterpixeln.
	lupeAbstand = 24
)

// LupeAn berechnet die Lupe für einen Zeiger in einem Fenster der Größe fenster. Die Lupe steht rechts unter dem
// Zeiger und weicht an den Rändern nach links bzw. oben aus, damit sie ganz sichtbar bleibt.
func (z *Zustand) LupeAn(zeiger Punkt, fenster image.Point) Lupe {
	m := z.Abb.Pixel(zeiger, z.Groesse)
	h := LupePixel / 2
	quelle := image.Rect(m.X-h, m.Y-h, m.X+h+1, m.Y+h+1)
	kante := LupePixel * LupeFaktor
	zx, zy := int(zeiger.X), int(zeiger.Y)
	x, y := zx+lupeAbstand, zy+lupeAbstand
	if x+kante > fenster.X {
		x = zx - lupeAbstand - kante
	}
	if y+kante > fenster.Y {
		y = zy - lupeAbstand - kante
	}
	x, y = klemme(x, 0, fenster.X-kante), klemme(y, 0, fenster.Y-kante)
	return Lupe{Quelle: quelle, Ziel: image.Rect(x, y, x+kante, y+kante), Mitte: m, Faktor: LupeFaktor}
}

// MassText ist die Maßanzeige eines Bereichs in Bildpixeln, z. B. „640 × 480“.
func MassText(r image.Rectangle) string { return fmt.Sprintf("%d × %d", r.Dx(), r.Dy()) }

// PositionText ist die Anzeige der Pixelposition, z. B. „120, 45“.
func PositionText(p image.Point) string { return fmt.Sprintf("%d, %d", p.X, p.Y) }

// MassPlatz liefert die linke obere Ecke für ein Maß-Schild der Größe schild unter dem Fensterrechteck r; reicht der
// Platz darunter nicht, steht es darüber, sonst innen oben. Die Ecke bleibt im Fenster.
func MassPlatz(r image.Rectangle, schild, fenster image.Point) image.Point {
	const luft = 8
	x := r.Min.X
	y := r.Max.Y + luft
	if y+schild.Y > fenster.Y {
		y = r.Min.Y - luft - schild.Y
		if y < 0 {
			y = r.Min.Y + luft
		}
	}
	return image.Pt(klemme(x, 0, fenster.X-schild.X), klemme(y, 0, fenster.Y-schild.Y))
}
