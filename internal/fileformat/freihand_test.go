// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0049 Punkt 6: Der Freihand-Strich hebt die Version auf 3, kommt im Rundlauf an (auch gemustert und als freier
// Textmarker); in Version 2 ist er ein unbekannter Objekttyp (beschädigt); zu wenige oder zu viele Punkte beschädigen.
func TestFreihandRundlauf(t *testing.T) {
	d := beispielDok(t, 120, 90)
	rot := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}
	gest := rot
	gest.Art = document.StrichGestrichelt
	m := document.NeuerFreihand([]document.Punkt{document.Pt(10, 70), document.Pt(50, 72), document.Pt(100, 68)},
		document.Strich{Farbe: color.NRGBA{R: 255, G: 230, B: 40, A: 255}, Breite: 12})
	m.Ebene.Mischung = document.MischMultiplizieren
	fuege(t, d,
		document.NeuerFreihand([]document.Punkt{document.Pt(5, 5), document.Pt(20.5, 30.25), document.Pt(60, 10)}, rot),
		document.NeuerFreihand([]document.Punkt{document.Pt(5, 40), document.Pt(100, 50)}, gest), m)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if !strings.Contains(svg, `s:format="3"`) || strings.Count(svg, `s:typ="freihand"`) != 3 {
		t.Fatalf("Arbeitsdatei: %s", svg)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	o := g.Dokument.Objekte()
	for i, soll := range d.Objekte() {
		if !document.Gleich(o[i], soll) {
			t.Errorf("Objekt %d nach dem Rundlauf: %+v, Soll %+v", i, o[i], soll)
		}
	}
	for name, fremd := range map[string]string{
		"Version 2":    strings.Replace(svg, `s:format="3"`, `s:format="2"`, 1),
		"ein Punkt":    strings.Replace(svg, `s:punkte="5,40 100,50"`, `s:punkte="5,40"`, 1),
		"Punkt kaputt": strings.Replace(svg, `s:punkte="5,40 100,50"`, `s:punkte="5,40 x,50"`, 1),
		"Doppelraum":   strings.Replace(svg, `s:punkte="5,40 100,50"`, `s:punkte="5,40  100,50"`, 1),
	} {
		if _, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, buf.Bytes(), fremd))); !errors.Is(err, ErrBeschaedigt) {
			t.Errorf("%s: %v, Soll beschädigt", name, err)
		}
	}
	viele := make([]string, document.MaxFreihandPunkte+1)
	for i := range viele {
		viele[i] = "1,1"
	}
	zuviel := strings.Replace(svg, `s:punkte="5,40 100,50"`, `s:punkte="`+strings.Join(viele, " ")+`"`, 1)
	if _, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, buf.Bytes(), zuviel))); !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("zu viele Punkte: %v", err)
	}
	// Ohne Freihand bleibt die Version, wie sie war.
	if v := schreibVersion([]document.Objekt{document.GeraderPfeil(document.Pt(0, 0), document.Pt(9, 9), rot)}); v != 1 {
		t.Errorf("Pfeil: Version %d", v)
	}
	e := exportSVG(t, d)
	if strings.Contains(e, "s:punkte") || !strings.Contains(e, "stroke-linejoin=\"round\"") {
		t.Errorf("Export: %s", e)
	}
}
