// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0047 Punkt 6: s:form="ellipse" hebt die Version auf 2 und kommt im Rundlauf an, auch gemustert und ohne Fläche;
// in Version 1 und mit unbekanntem Wert gilt das Rechteck; der Export schreibt <ellipse> bzw. gemustert die Stücke.
func TestEllipseRundlauf(t *testing.T) {
	rot := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}
	gestrichelt := rot
	gestrichelt.Art = document.StrichGestrichelt
	d := beispielDok(t, 120, 90)
	fuege(t, d,
		document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(60, 40)), Form: document.FormEllipse, Strich: rot,
			Fuellung: color.NRGBA{G: 200, A: 128}},
		document.Kasten{Bereich: document.R(document.Pt(64, 4), document.Pt(110, 50)), Form: document.FormEllipse,
			Strich: gestrichelt},
		document.Kasten{Bereich: document.R(document.Pt(4, 60), document.Pt(60, 60)), Form: document.FormEllipse, Strich: rot},
		document.Kasten{Bereich: document.R(document.Pt(64, 60), document.Pt(110, 80)), Strich: rot})
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if !strings.Contains(svg, `s:format="2"`) || strings.Count(svg, `s:form="ellipse"`) != 3 || strings.Count(svg, "<ellipse") != 2 {
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
	// Version 1 und unbekannter Wert: Rechteck.
	for _, fremd := range []string{strings.Replace(svg, `s:format="2"`, `s:format="1"`, 1),
		strings.ReplaceAll(svg, `s:form="ellipse"`, `s:form="cloud"`)} {
		g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, buf.Bytes(), fremd)))
		if err != nil {
			t.Fatal(err)
		}
		for i, o := range g.Dokument.Objekte() {
			if o.(document.Kasten).Form != document.FormRechteck {
				t.Errorf("Objekt %d: Form %v", i, o.(document.Kasten).Form)
			}
		}
	}
	// Eine einzelne volle Ellipse hebt die Version allein (ADR-0047 Punkt 6).
	de := beispielDok(t, 64, 48)
	fuege(t, de, document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 30)), Form: document.FormEllipse, Strich: rot})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, de, de.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="2"`) {
		t.Errorf("volle Ellipse allein: %s", svg)
	}
	// Nur ein Rechteck: Version 1, kein s:form.
	d1 := beispielDok(t, 64, 48)
	fuege(t, d1, document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 40)), Strich: rot})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, d1, d1.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="1"`) || strings.Contains(svg, "s:form=") {
		t.Errorf("Rechteck: %s", svg)
	}
}

func TestEllipseExport(t *testing.T) {
	rot := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}
	gepunktet := rot
	gepunktet.Art = document.StrichGepunktet
	d := beispielDok(t, 120, 90)
	fuege(t, d,
		document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(60, 40)), Form: document.FormEllipse, Strich: rot},
		document.Kasten{Bereich: document.R(document.Pt(64, 4), document.Pt(110, 50)), Form: document.FormEllipse,
			Strich: gepunktet, Fuellung: color.NRGBA{B: 200, A: 255}})
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	if strings.Contains(svg, "s:") || strings.Contains(svg, "stroke-dasharray") {
		t.Errorf("Export mit Schnapp-Attributen oder Näherung: %s", svg)
	}
	// Die volle Ellipse als <ellipse> mit Strich, die gepunktete als Füllung plus Stücke.
	if strings.Count(svg, "<ellipse") != 2 || !strings.Contains(svg, `rx="28" ry="18"`) {
		t.Errorf("Export: %s", svg)
	}
}
