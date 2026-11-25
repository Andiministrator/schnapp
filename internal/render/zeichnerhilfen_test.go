// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image/color"

	"golang.org/x/image/font"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// Testhilfen für einzeilige Texte ohne Naht. Der Produktcode zeichnet Text nur noch über textBaender (zeilenweise über
// die Abschnitte der Ansicht, F-747, ADR-0023 Nachtrag 16); die Tests prüfen damit weiter die seitliche Hülle der
// Schrift, den Vorschub bei unlesbaren Glyphen und die Sichtbarkeitsregel mit genau einem Bereich (F-768).

// zeileSichtbar meldet, ob eine Textzeile ab x auf dieser Grundlinie das Bild schneiden kann: senkrecht, und rechts
// beginnt die erste Glyphe nicht hinter dem Rand. Die Grenzen folgen der Schrift sch (fonts.Schrift.Huelle, F-576).
func (z zeichner) zeileSichtbar(sch *fonts.Schrift, x, grundlinie, grad float64) bool {
	oben, unten, seitlich := sch.Huelle()
	return grundlinie-grad*oben <= float64(z.rand.Max.Y) && grundlinie+grad*unten >= float64(z.rand.Min.Y) &&
		x <= float64(z.rand.Max.X)+seitlich*grad
}

// text zeichnet eine Zeile in den Rand des Zeichners: ein Bereich für textBaender. Was das Bild nicht schneidet, wird
// nicht gerastert. Die Kosten zählt der Aufrufer.
func (z zeichner) text(face font.Face, sch *fonts.Schrift, s string, x, grundlinie float64, farbe color.NRGBA, grad float64) {
	oben, unten, _ := sch.Huelle()
	rand := z.rand
	if rand.Empty() || grundlinie-grad*oben > float64(rand.Max.Y) || grundlinie+grad*unten < float64(rand.Min.Y) {
		return
	}
	z.textBaender(face, sch, s, x, grundlinie, farbe, grad, []band{{A: rand.Min.X, B: rand.Max.X}},
		[]band{{A: rand.Min.Y, B: rand.Max.Y}})
}
