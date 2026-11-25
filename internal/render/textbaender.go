// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Text über vielen Nähten (F-747, F-750, ADR-0023 Nachtrag 16): Bis dahin lief der Renderer je sichtbarem Bereich
// (document.TextStuecke) über alle Zeilen des Texts und rasterte jede Glyphe in jedem Bereich neu, in dessen Nähe sie
// lag. Über 1024 Zeilenstreifen prüfte er so 20 Texte à 10 000 Zeilen in 5 s statt 22 ms, über 1023 Spaltenstreifen
// rasterte er 76 650 statt 1 740 Glyphen, gezählt war aber jede nur einmal. Die Ansicht ist zwischen zwei Nähten eine
// Verschiebung um ganze Pixel, und die Bereiche sind das Kreuzprodukt waagerechter und senkrechter Abschnitte
// (document.Abschnitte). Deshalb jetzt zeilenweise: Eine Zeile sucht die senkrechten Abschnitte ihrer Höhe per
// Binärsuche, jede Glyphe sucht die waagerechten Abschnitte ihrer Lage ebenso, wird einmal gerastert und je Bereich nur
// verschoben eingemischt. Das Ergebnis ist dasselbe Bild (jedes Pixel gehört zu genau einem Bereich, die Reihenfolge der
// Glyphen darin bleibt), die Kosten zählen wie bisher einmal je sichtbarer Zeile.

// baender und baenderIn sind die Abschnitte der Ansicht auf den Rand beschnitten bzw. per Binärsuche gefunden
// (document.Beschneide, document.AbschnitteIn; der Editor rechnet dasselbe).
type band = document.Abschnitt

func baender(as []document.Abschnitt, lo, hi int) []band { return document.Beschneide(as, lo, hi) }

func baenderIn(bs []band, lo, hi float64) []band { return document.AbschnitteIn(bs, lo, hi) }

// textBaender zeichnet die Zeile s ab (x, grundlinie) im Original in alle Bereiche spalten × zeilen: jede Glyphe, deren
// Ursprung höchstens die seitliche Hülle der Schrift (fonts.Schrift.Huelle) neben einer Spalte liegt, wird einmal
// gerastert und je Bereich um dessen Versatz verschoben, auf ihn beschnitten eingemischt. Vorschub und Unterschneidung
// laufen wie in text auch über ausgelassene Glyphen. Die Kosten zählt der Aufrufer. spalten und zeilen sind nicht leer.
func (z zeichner) textBaender(face font.Face, sch *fonts.Schrift, s string, x, grundlinie float64, farbe color.NRGBA,
	grad float64, spalten, zeilen []band) {
	var puffer sfnt.Buffer
	sf := sch.Font
	_, _, seitlich := sch.Huelle()
	sg := seitlich * grad
	rechts := float64(spalten[len(spalten)-1].B) + sg
	dot := fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round(grundlinie * 64))}
	vorher := rune(-1)
	for _, c := range s {
		saveprogress.Puls() // je Zeichen wie in text (F-147, F-298)
		if vorher >= 0 {
			dot.X += document.KernPaar(sf, &puffer, vorher, c, grad)
		}
		vorher = c
		px := float64(dot.X) / 64
		if px > rechts {
			return // alles Weitere liegt rechts von allen Bereichen
		}
		i := sort.Search(len(spalten), func(k int) bool { return float64(spalten[k].B)+sg >= px })
		if i == len(spalten) || float64(spalten[i].A)-sg > px {
			adv, _ := face.GlyphAdvance(c) // Vorschub auch ohne Glyphe, wie font.Drawer.DrawString
			dot.X += adv
			continue
		}
		z.auf.glyphen++
		gr, maske, mp, adv, ok := face.Glyph(dot, c)
		if !ok {
			adv, _ = face.GlyphAdvance(c) // F-577: Face.Glyph meldet bei unlesbaren Umrissen Vorschub 0
		}
		for k := i; k < len(spalten) && float64(spalten[k].A)-sg <= px; k++ {
			sp := spalten[k]
			for _, ze := range zeilen {
				v := image.Pt(sp.Versatz, ze.Versatz)
				ziel := image.Rect(sp.A, ze.A, sp.B, ze.B).Add(v)
				g := gr.Add(v)
				g2 := g.Intersect(ziel)
				if g2.Empty() {
					continue
				}
				glyphMischen(z.bild, g2, maske, mp.Add(g2.Min.Sub(g.Min)), farbe)
			}
		}
		dot.X += adv
	}
}
