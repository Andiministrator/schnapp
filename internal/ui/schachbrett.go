// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Das Schachbrett zeigt im Editor, was durchsichtig ist: den Spalt jeder Stauch-Naht (ADR-0023 Nachtrag 7), wie in
// Bildprogrammen üblich, eckig und gedämpft (Farben wie im Referenzbild 0023-naht-durchsichtig-muster.png). Ein Feld
// ist schachFeld Ansichtspixel groß und liegt fest im Raster der Ansicht ((x/8 + y/8) gerade: hell), so stoßen die
// Bretter benachbarter Nähte nahtlos aneinander. Es zoomt mit dem Bild (der Zwischenspeicher der Nähte hängt nicht am
// Zoom).
const schachFeld = 8

var (
	schachHell   = color.NRGBA{R: 0x9A, G: 0x9E, B: 0xA3, A: 0xFF}
	schachDunkel = color.NRGBA{R: 0x7A, G: 0x7E, B: 0x83, A: 0xFF}
)

// schachbretter hält die Bilder der Bretter, je Achse und Lage des ersten Felds (gerade oder ungerade), für einen
// Aufbau der Naht-Ops: Alle Nähte einer Achse teilen sich höchstens zwei Bilder.
type schachbretter struct {
	ops [2][2]paint.ImageOp
	da  [2][2]bool
}

// schachBild ist ein Brett mit einem Pixel je Feld: lang Felder entlang der Naht, quer Felder quer dazu; ungerade
// verschiebt das Muster um ein Feld. zeilen: die Naht läuft waagrecht (lang ist die Breite).
func schachBild(lang, quer int, zeilen, ungerade bool) *image.NRGBA {
	b, h := lang, quer
	if !zeilen {
		b, h = quer, lang
	}
	bild := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			c := schachHell
			if (x+y)%2 == 1 != ungerade {
				c = schachDunkel
			}
			bild.SetNRGBA(x, y, c)
		}
	}
	return bild
}

// schachLage liefert für die Naht n das erste Feld quer zur Naht (in Feldern) und die Zahl der Felder quer, so dass
// das Brett das ganze Band ±ZackenBandHalb·faktor um die Naht deckt, sowie die Zahl der Felder entlang der Naht
// (laenge plus ein Stützschritt Überstand, den Kantenlinie.Band haben kann).
func schachLage(n document.Naht, laenge, faktor int) (erstes, quer, lang int) {
	halb := document.ZackenBandHalb * float64(max(faktor, 1))
	erstes = int(math.Floor((float64(n.Lage) - halb) / schachFeld))
	letztes := int(math.Ceil((float64(n.Lage) + halb) / schachFeld))
	lang = (laenge+8*max(faktor, 1))/schachFeld + 2
	return erstes, letztes - erstes, lang
}

// male legt das Brett der Naht n in die aktuelle Clip-Fläche (der Spalt der Naht), in Ansichtskoordinaten.
func (s *schachbretter) male(o *op.Ops, n document.Naht, laenge, faktor int) {
	erstes, quer, lang := schachLage(n, laenge, faktor)
	achse := 0
	if !n.Zeilen {
		achse = 1
	}
	ungerade := erstes%2 != 0
	u := 0
	if ungerade {
		u = 1
	}
	if !s.da[achse][u] {
		// Ein Brett deckt jede Naht dieser Achse; quer reicht für jede Lage gleich viele Felder plus eins.
		img := paint.NewImageOp(schachBild(lang, quer+1, n.Zeilen, ungerade))
		img.Filter = paint.FilterNearest
		s.ops[achse][u], s.da[achse][u] = img, true
	}
	ursprung := f32.Pt(0, float32(erstes*schachFeld))
	if !n.Zeilen {
		ursprung = f32.Pt(float32(erstes*schachFeld), 0)
	}
	t := op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(schachFeld, schachFeld)).Offset(ursprung)).Push(o)
	s.ops[achse][u].Add(o)
	paint.PaintOp{}.Add(o)
	t.Pop()
}
