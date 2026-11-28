// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0048 Punkt 7: s:mischung="multiply" an der Ebene hebt die Version auf 2 und kommt im Rundlauf an; in Version 1
// und mit unbekanntem Wert gilt „normal“. Arbeitsdatei und Export schreiben mix-blend-mode.
func TestMischungRundlauf(t *testing.T) {
	d := beispielDok(t, 120, 90)
	m := document.GeraderPfeil(document.Pt(4, 20), document.Pt(100, 20), document.Strich{Farbe: color.NRGBA{R: 255, G: 230, B: 40, A: 255}, Breite: 12})
	m.Ende = document.KopfKeine
	m.Ebene.Mischung = document.MischMultiplizieren
	k := document.Kasten{Bereich: document.R(document.Pt(10, 40), document.Pt(60, 70)), Strich: document.Strich{Farbe: color.NRGBA{B: 200, A: 255}, Breite: 2}}
	fuege(t, d, m, k)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if !strings.Contains(svg, `s:format="2"`) || strings.Count(svg, `s:mischung="multiply"`) != 1 ||
		strings.Count(svg, `mix-blend-mode:multiply`) != 1 {
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
	for _, fremd := range []string{strings.Replace(svg, `s:format="2"`, `s:format="1"`, 1),
		strings.ReplaceAll(svg, `s:mischung="multiply"`, `s:mischung="screen"`)} {
		g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, buf.Bytes(), fremd)))
		if err != nil {
			t.Fatal(err)
		}
		for i, o := range g.Dokument.Objekte() {
			if o.Ebenenwerte().Mischung != document.MischNormal {
				t.Errorf("Objekt %d: Mischung %v", i, o.Ebenenwerte().Mischung)
			}
		}
	}
	// Die Mischung allein hebt die Version (ein Pfeil mit Dreieck am Ende und voller Strich bliebe sonst bei 1).
	d1 := beispielDok(t, 64, 48)
	p := document.GeraderPfeil(document.Pt(4, 4), document.Pt(40, 30), document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3})
	fuege(t, d1, p)
	if v := schreibVersion(d1.Objekte()); v != 1 {
		t.Fatalf("Pfeil ohne Mischung: Version %d", v)
	}
	p.Ebene.Mischung = document.MischMultiplizieren
	if v := schreibVersion([]document.Objekt{p}); v != versionMischung {
		t.Errorf("Pfeil mit Mischung: Version %d", v)
	}
	if e := exportSVG(t, d); !strings.Contains(e, `mix-blend-mode:multiply`) || strings.Contains(e, `s:mischung`) {
		t.Errorf("Export ohne mix-blend-mode oder mit s:-Attribut: %s", e)
	}
}
