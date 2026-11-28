// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image/color"
	"math"
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

// F-1043 (Kritik #333 K-13, Mutant FH36): Der Export schreibt einen gemusterten Freihand-Strich als ausdrückliche
// Stücke (mehrere Teilpfade, stumpfe Enden, kein stroke-dasharray), die Arbeitsdatei als ein Pfad mit Näherung.
func TestFreihandExportGemustertAlsStuecke(t *testing.T) {
	d := beispielDok(t, 200, 60)
	fuege(t, d, document.NeuerFreihand([]document.Punkt{document.Pt(10, 30), document.Pt(100, 20), document.Pt(190, 30)},
		document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 4, Art: document.StrichGestrichelt}))
	var ex bytes.Buffer
	if err := ExportiereSVG(&ex, d); err != nil {
		t.Fatal(err)
	}
	svg := ex.String()
	if strings.Contains(svg, "dasharray") || !strings.Contains(svg, `stroke-linecap="butt"`) || strings.Count(svg, " M") < 3 {
		t.Errorf("Export ohne ausdrückliche Stücke: %s", svg)
	}
	if strings.Contains(svg, `stroke-linecap="round"`) {
		t.Error("Export trägt den vollen Pfad mit runden Enden")
	}
	if a := string(AngabenSVG(d)); !strings.Contains(a, "stroke-dasharray") {
		t.Errorf("Arbeitsdatei ohne Näherung: %s", a)
	}
}

// F-1036 (Kritik #333 K-2): Ein langer, wirklichkeitsnaher Zug (ganze Fensterpixel im Abstand von rund 2 px, Zoom 1 und
// 0,73, mit und ohne Zittern) wird gespeichert mit höchstens so vielen Punkten wie roh und kurzen Zahlen. Vorher: 4096
// Punkte (Chaikin vervierfachte), 32 bzw. 75 Byte je Punkt, 130 bzw. 306 KB je Strich. Gemessen 28.11.2026 nach
// ADR-0049 Nachtrag 1: 1142 bis 2189 Punkte, 25 bis 27 Byte je Punkt (s:punkte absolut und der Pfad relativ, je auf
// 1/100 px), höchstens 60 KB je Strich.
func TestFreihandGroesseJeStrich(t *testing.T) {
	for _, zoom := range []float64{1, 0.73} {
		for _, zittern := range []float64{0, 1} {
			var roh []document.Punkt
			var letzter document.Punkt
			for i := 0; len(roh) < document.MaxFreihandPunkte; i++ {
				a := float64(i) * 0.002
				r := 50 + 20*a
				z := zittern * math.Sin(float64(i)*0.37)
				p := document.Pt(math.Round(1000+(r+z)*math.Cos(a)), math.Round(1000+(r+z)*math.Sin(a)))
				if len(roh) == 0 || p.Minus(letzter).Laenge() >= 2 {
					roh = append(roh, document.Pt(p.X/zoom, p.Y/zoom))
					letzter = p
				}
			}
			pts := document.FreihandAusZug(roh, 0.5/zoom)
			d := beispielDok(t, 3000, 3000)
			fuege(t, d, document.NeuerFreihand(pts, document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}))
			svg := AngabenSVG(d)
			jePunkt := float64(len(svg)) / float64(len(pts))
			t.Logf("Zoom %.2f Zittern %.0f: roh %d, gespeichert %d Punkte, SVG %d Bytes (%.1f je Punkt)",
				zoom, zittern, len(roh), len(pts), len(svg), jePunkt)
			if len(pts) > len(roh) || jePunkt > 28 || len(svg) > 64<<10 {
				t.Errorf("Zoom %.2f Zittern %.0f: %d Punkte (roh %d), %.1f Byte je Punkt, %d Bytes", zoom, zittern,
					len(pts), len(roh), jePunkt, len(svg))
			}
		}
	}
}

// F-1036: Die Darstellung schreibt den ersten Punkt absolut, die übrigen relativ in Hundertsteln, gerechnet aus den
// gerundeten Lagen (keine Rundungsfehler, die sich ansammeln).
func TestFreihandPfadRelativ(t *testing.T) {
	pts := []document.Punkt{document.Pt(10.5, 20.0004), document.Pt(13.71, 18.9), document.Pt(13.71, 18.9),
		document.Pt(0.004, 120.006)}
	if p, soll := freihandPfad(pts), "M10.5,20l3.21,-1.1 0,0 -13.71,101.11"; p != soll {
		t.Errorf("Pfad %q, Soll %q", p, soll)
	}
	if p := freihandPfad(pts[:1]); p != "M10.5,20" {
		t.Errorf("ein Punkt: %q", p)
	}
}
