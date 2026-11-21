// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-108: Das SVG hat dieselben Maße wie der Renderer: runde Strichecken, der Eckradius wie WirksamerEckradius (nie
// elliptisch), der Text-Hintergrund aus Grenzen(), die Schrift „Go“ (nicht Open Sans) mit serifenloser Rückfallfamilie,
// die Ziffern auf der Grundlinie der Nummer.
func TestSVGMasseWieRenderer(t *testing.T) {
	d, err := document.Neu(testbild(300, 200))
	if err != nil {
		t.Fatal(err)
	}
	schwarz := color.NRGBA{0, 0, 0, 255}
	kasten := document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(50, 30)), Strich: document.Strich{Farbe: schwarz, Breite: 4}, Eckradius: 100}
	text := document.Text{Ort: document.Pt(60, 60), Inhalt: "Hg", Schriftgrad: 40, Farbe: schwarz, Hintergrund: color.NRGBA{255, 255, 0, 255}}
	nummer := document.Nummer{Mitte: document.Pt(100, 150), Wert: 7, Radius: 20, Farbe: schwarz}
	for _, o := range []document.Objekt{kasten, text, nummer} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, soll := range []string{
		`rx="10"`, // min(100, min(40, 20) / 2), nicht 20 und nicht elliptisch
		`stroke-linejoin="round"`,
		fmt.Sprintf(`font-family="%s"`, svgSchrift),
		fmt.Sprintf(`y="%s"`, zahl(nummer.Grundlinie())),
		fmt.Sprintf(`y="%s"`, zahl(text.Grundlinie(0))),
	} {
		if !strings.Contains(s, soll) {
			t.Errorf("SVG enthält %s nicht", soll)
		}
	}
	g := text.Grenzen()
	for _, soll := range []string{fmt.Sprintf(`width="%s"`, zahl(g.Breite())), fmt.Sprintf(`height="%s"`, zahl(g.Hoehe()))} {
		if !strings.Contains(s, soll) {
			t.Errorf("Hintergrund des Textes: SVG enthält %s nicht (Grenzen %v)", soll, g)
		}
	}
	for _, nein := range []string{"Open Sans", "dominant-baseline"} {
		if strings.Contains(s, nein) {
			t.Errorf("SVG enthält noch %q", nein)
		}
	}
	if !strings.Contains(svgSchrift, "sans-serif") {
		t.Errorf("Schrift %q ohne Rückfallfamilie", svgSchrift)
	}
}
