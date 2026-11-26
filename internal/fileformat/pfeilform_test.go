// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 19 Punkt 4: s:kopf und s:enden kommen beim Lesen an; unbekannte Werte und fehlende Attribute
// gelten als Dreieck am Ende (kein Schaden, FormatVersion 1); Voreinstellungen schreibt der Schreiber nicht.
func TestPfeilformLesenUndUnbekannt(t *testing.T) {
	d := beispielDok(t, 64, 48)
	strich := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}
	fuege(t, d,
		document.Pfeil{A: document.Pt(4, 10), B: document.Pt(40, 10), K: document.Pt(22, 10), Strich: strich,
			Kopf: document.KopfWinkel, Enden: document.EndenBeide},
		document.Pfeil{A: document.Pt(4, 30), B: document.Pt(40, 30), K: document.Pt(22, 30), Strich: strich})
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if strings.Count(svg, "s:kopf=") != 1 || strings.Count(svg, "s:enden=") != 1 || !strings.Contains(svg, `s:format="1"`) {
		t.Fatalf("Voreinstellung geschrieben oder Version falsch: %s", svg)
	}
	lies := func(png []byte) []document.Objekt {
		t.Helper()
		g, err := LeseArbeitsdatei(bytes.NewReader(png))
		if err != nil {
			t.Fatal(err)
		}
		return g.Dokument.Objekte()
	}
	o := lies(buf.Bytes())
	if p := o[0].(document.Pfeil); p.Kopf != document.KopfWinkel || p.Enden != document.EndenBeide {
		t.Errorf("gelesen %v %v", p.Kopf, p.Enden)
	}
	if p := o[1].(document.Pfeil); p.Kopf != document.KopfDreieck || p.Enden != document.EndenEnde {
		t.Errorf("ohne Attribute %v %v", p.Kopf, p.Enden)
	}
	fremd := strings.Replace(strings.Replace(svg, `s:kopf="open"`, `s:kopf="spear"`, 1), `s:enden="both"`, `s:enden="around"`, 1)
	o = lies(pngMitSVG(t, buf.Bytes(), fremd))
	if p := o[0].(document.Pfeil); p.Kopf != document.KopfDreieck || p.Enden != document.EndenEnde {
		t.Errorf("unbekannte Werte: %v %v", p.Kopf, p.Enden)
	}
}
