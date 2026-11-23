// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0032 Punkt 9: Name, Ausblenden, Sperre und Deckkraft überstehen Schreiben und Lesen an jeder Art; fehlend
// heißt Voreinstellung.
func TestEbenenRundlauf(t *testing.T) {
	d := beispielDok(t, 64, 48)
	eb := document.Ebene{Name: "Schild „oben“ & <x>", Ausgeblendet: true, Gesperrt: true}.MitDeckkraft(35)
	rot := color.NRGBA{200, 0, 0, 255}
	objekte := []document.Objekt{
		document.Pfeil{Ebene: eb, A: document.Pt(1, 1), B: document.Pt(9, 9), K: document.Pt(5, 5), Strich: document.Strich{Farbe: rot, Breite: 2}},
		document.Kasten{Ebene: eb, Bereich: document.R(document.Pt(2, 2), document.Pt(8, 8)), Strich: document.Strich{Farbe: rot, Breite: 1}},
		document.Text{Ebene: eb, Ort: document.Pt(3, 3), Inhalt: "a", Schriftgrad: 9, Farbe: rot},
		document.Nummer{Ebene: eb, Mitte: document.Pt(20, 20), Wert: 2, Radius: 5, Farbe: rot},
		document.Nummer{Mitte: document.Pt(30, 20), Wert: 3, Radius: 5, Farbe: rot}, // ohne Angaben
	}
	fuege(t, d, objekte...)
	in, err := leseSVG(bytes.NewReader(svgArbeitsdatei(d, Metadaten{})))
	if err != nil {
		t.Fatal(err)
	}
	if len(in.objekte) != len(objekte) {
		t.Fatalf("%d Objekte gelesen", len(in.objekte))
	}
	for i, o := range in.objekte[:4] {
		if o.Ebenenwerte() != eb {
			t.Errorf("Objekt %d (%T): %+v, Soll %+v", i, o, o.Ebenenwerte(), eb)
		}
	}
	if e := in.objekte[4].Ebenenwerte(); e != (document.Ebene{}) {
		t.Errorf("Objekt ohne Angaben gelesen als %+v", e)
	}
	svg := string(svgArbeitsdatei(d, Metadaten{}))
	if n := strings.Count(svg, `display="none"`); n != 4 {
		t.Errorf("%d× display=none, Soll 4 (nur die ausgeblendeten)", n)
	}
	if n := strings.Count(svg, `opacity="0.35"`); n != 4 {
		t.Errorf("%d× opacity=0.35, Soll 4", n)
	}
}

// Ungültige oder unbekannte Werte der Ebenen-Attribute sind kein Schaden: Voreinstellung (8.5).
func TestEbenenUngueltigKeinSchaden(t *testing.T) {
	d := beispielDok(t, 64, 48)
	fuege(t, d, document.Nummer{Mitte: document.Pt(20, 20), Wert: 1, Radius: 5, Farbe: color.NRGBA{A: 255}})
	svg := string(svgArbeitsdatei(d, Metadaten{}))
	for _, attr := range []string{`s:deckkraft="101"`, `s:deckkraft="-1"`, `s:deckkraft="+5"`, `s:deckkraft="50.5"`,
		`s:ausgeblendet="yes"`, `s:gesperrt="1"`, `s:name="a&#xA;b"`, `s:name="` + strings.Repeat("x", 101) + `"`} {
		kaputt := strings.Replace(svg, `s:typ="nummer"`, `s:typ="nummer" `+attr, 1)
		in, err := leseSVG(strings.NewReader(kaputt))
		if err != nil {
			t.Errorf("%s: %v", attr, err)
			continue
		}
		if e := in.objekte[0].Ebenenwerte(); e != (document.Ebene{}) {
			t.Errorf("%s: gelesen als %+v", attr, e)
		}
	}
	in, err := leseSVG(strings.NewReader(strings.Replace(svg, `s:typ="nummer"`, `s:typ="nummer" s:deckkraft="0"`, 1)))
	if err != nil || in.objekte[0].Ebenenwerte().Deckkraft() != 0 {
		t.Errorf("Deckkraft 0: %v", err)
	}
}

// Der SVG-Export lässt ausgeblendete Ebenen weg (auch ihren Text) und schreibt die Deckkraft ohne s:-Attribute.
func TestEbenenExportSVG(t *testing.T) {
	d := beispielDok(t, 64, 48)
	schwarz := color.NRGBA{A: 255}
	fuege(t, d,
		document.Text{Ebene: document.Ebene{Ausgeblendet: true}, Ort: document.Pt(3, 3), Inhalt: "GEHEIM", Schriftgrad: 9, Farbe: schwarz},
		document.Text{Ebene: document.Ebene{Name: "Sichtbar"}.MitDeckkraft(60), Ort: document.Pt(3, 20), Inhalt: "offen", Schriftgrad: 9, Farbe: schwarz})
	var b bytes.Buffer
	if err := ExportiereSVG(&b, d); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if strings.Contains(s, "GEHEIM") {
		t.Error("ausgeblendeter Text steht im Export")
	}
	if !strings.Contains(s, "offen") || !strings.Contains(s, `opacity="0.6"`) {
		t.Error("sichtbarer Text oder seine Deckkraft fehlt im Export")
	}
	if strings.Contains(s, "s:") || strings.Contains(s, "Sichtbar") {
		t.Error("Export trägt Schnapp-Attribute oder den Namen der Ebene")
	}
}
