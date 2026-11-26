// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 20 (F-830): s:ende und s:anfang kommen beim Lesen an und heben die Formatversion auf 2; unbekannte
// Werte und fehlende Attribute gelten als Dreieck bei B und keine Spitze bei A; Voreinstellungen schreibt der Schreiber
// nicht, und eine Datei nur mit Voreinstellungen bleibt Version 1. In Version 1 sind die Attribute unbekannter Inhalt.
func TestPfeilformLesenUndUnbekannt(t *testing.T) {
	d := beispielDok(t, 64, 48)
	strich := document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}
	fuege(t, d,
		document.Pfeil{A: document.Pt(4, 10), B: document.Pt(40, 10), K: document.Pt(22, 10), Strich: strich}.
			MitKoepfen(document.KopfRaute, document.KopfWinkel),
		document.Pfeil{A: document.Pt(4, 30), B: document.Pt(40, 30), K: document.Pt(22, 30), Strich: strich})
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if strings.Count(svg, "s:ende=") != 1 || strings.Count(svg, "s:anfang=") != 1 || !strings.Contains(svg, `s:format="2"`) {
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
	if p := o[0].(document.Pfeil); p.KopfAnfang() != document.KopfRaute || p.Ende != document.KopfWinkel {
		t.Errorf("gelesen %v %v", p.KopfAnfang(), p.Ende)
	}
	if p := o[1].(document.Pfeil); p.KopfAnfang() != document.KopfKeine || p.Ende != document.KopfDreieck {
		t.Errorf("ohne Attribute %v %v", p.KopfAnfang(), p.Ende)
	}
	fremd := strings.Replace(strings.Replace(svg, `s:ende="open"`, `s:ende="spear"`, 1), `s:anfang="diamond"`, `s:anfang="moon"`, 1)
	o = lies(pngMitSVG(t, buf.Bytes(), fremd))
	if p := o[0].(document.Pfeil); p.KopfAnfang() != document.KopfKeine || p.Ende != document.KopfDreieck {
		t.Errorf("unbekannte Werte: %v %v", p.KopfAnfang(), p.Ende)
	}
	// In Version 1 gelten die Attribute nicht (ein Schreiber der Version 1 kannte sie nicht).
	v1 := strings.Replace(svg, `s:format="2"`, `s:format="1"`, 1)
	o = lies(pngMitSVG(t, buf.Bytes(), v1))
	if p := o[0].(document.Pfeil); p.KopfAnfang() != document.KopfKeine || p.Ende != document.KopfDreieck {
		t.Errorf("Version 1: %v %v", p.KopfAnfang(), p.Ende)
	}
	// Nur Voreinstellungen: Version 1.
	d2 := beispielDok(t, 64, 48)
	fuege(t, d2, document.Pfeil{A: document.Pt(4, 30), B: document.Pt(40, 30), K: document.Pt(22, 30), Strich: strich})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, d2, d2.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="1"`) {
		t.Errorf("Pfeil mit Voreinstellung hebt die Version: %s", svg)
	}
}
