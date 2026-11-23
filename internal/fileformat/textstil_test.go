// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// textAusSVG schreibt ein Dokument mit einem Text, ersetzt im SVG alt durch neu und liest den Text zurück.
func textAusSVG(t *testing.T, tx document.Text, alt, neu string) (document.Text, error) {
	t.Helper()
	d := beispielDok(t, 64, 48)
	fuege(t, d, tx)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if alt != "" {
		if strings.Count(svg, alt) != 1 {
			t.Fatalf("%q kommt nicht genau einmal vor: %s", alt, svg)
		}
		svg = strings.Replace(svg, alt, neu, 1)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, buf.Bytes(), svg)))
	if err != nil {
		return document.Text{}, err
	}
	o := g.Dokument.Objekte()
	if len(o) != 1 {
		t.Fatalf("%d Objekte", len(o))
	}
	return o[0].(document.Text), nil
}

var stilText = document.Text{Ort: document.Pt(4, 4), Inhalt: "ab", Schriftgrad: 9, Farbe: color.NRGBA{A: 255},
	Schrift: "Example Sans", Fett: true, Unterstrichen: true, Ausrichtung: document.AusrichtungRechts}

// Rundlauf: Schrift, Stil und Ausrichtung kommen unverändert zurück, auch eine fehlende Familie (ADR-0021 Nachtrag 10
// Punkt 4: der Name bleibt).
func TestTextStilRundlauf(t *testing.T) {
	g, err := textAusSVG(t, stilText, "", "")
	if err != nil {
		t.Fatal(err)
	}
	g.ID = 0
	if g != stilText {
		t.Errorf("zurück %+v, erwartet %+v", g, stilText)
	}
	if !g.SchriftFehlt() {
		t.Error("Example Sans gilt als vorhanden")
	}
	// Ohne Stil: keine der Angaben im SVG.
	ohne := document.Text{Ort: document.Pt(4, 4), Inhalt: "ab", Schriftgrad: 9, Farbe: color.NRGBA{A: 255}}
	d := beispielDok(t, 64, 48)
	fuege(t, d, ohne)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); strings.Contains(svg, "s:schrift") || strings.Contains(svg, "s:stil") ||
		strings.Contains(svg, "s:ausrichtung") || strings.Contains(svg, "font-weight") {
		t.Errorf("Voreinstellung schreibt Angaben: %s", svg)
	}
}

// Freiwillige Angaben: Fehlend, ungültig oder unbekannt heißt Go, normal, links — und ist kein Schaden.
func TestTextStilNachsichtig(t *testing.T) {
	for _, f := range []struct{ alt, neu string }{
		{`s:schrift="Example Sans"`, `s:schrift="Go"`},
		{`s:schrift="Example Sans"`, `s:schrift=""`},
		{`s:schrift="Example Sans"`, `s:schrift="a&#x9;b"`},
		{`s:schrift="Example Sans"`, `s:schrift="` + strings.Repeat("x", 129) + `"`},
		{`s:schrift="Example Sans"`, ``},
	} {
		g, err := textAusSVG(t, stilText, f.alt, f.neu)
		if err != nil || g.Schrift != "" || !g.Fett {
			t.Errorf("%s: Schrift %q, Fett %v, Fehler %v", f.neu, g.Schrift, g.Fett, err)
		}
	}
	g, err := textAusSVG(t, stilText, `s:stil="bold underline"`, `s:stil=" italic  wobbly STRIKE strike "`)
	if err != nil || g.Fett || !g.Kursiv || g.Unterstrichen || !g.Durchgestrichen {
		t.Errorf("Stil-Wörter: %+v, %v", g, err)
	}
	for _, neu := range []string{`s:ausrichtung="Right"`, `s:ausrichtung="justify"`, `s:ausrichtung=""`, ``} {
		if g, err := textAusSVG(t, stilText, `s:ausrichtung="right"`, neu); err != nil || g.Ausrichtung != document.AusrichtungLinks {
			t.Errorf("%s: %v, %v", neu, g.Ausrichtung, err)
		}
	}
	if g, err := textAusSVG(t, stilText, `s:ausrichtung="right"`, `s:ausrichtung="center"`); err != nil ||
		g.Ausrichtung != document.AusrichtungMitte {
		t.Errorf("center: %v, %v", g.Ausrichtung, err)
	}
}

// Die SVG-Darstellung nennt Familie, Gewicht und Stil, setzt jede Zeile an ihren Anfang und schreibt die Striche als
// Rechtecke (ADR-0021 Nachtrag 10 Punkte 6 bis 8).
func TestTextStilDarstellung(t *testing.T) {
	tx := stilText
	tx.Inhalt, tx.Schrift, tx.Kursiv, tx.Durchgestrichen = "WWWW\nx", "O'Brien \\ Sans", true, true
	d := beispielDok(t, 64, 48)
	fuege(t, d, tx)
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	for _, teil := range []string{`font-family="&#39;O\&#39;Brien \\ Sans&#39;, Go, Go Regular, sans-serif"`,
		`font-weight="bold"`, `font-style="italic"`} {
		if !strings.Contains(svg, teil) {
			t.Errorf("%s fehlt: %s", teil, svg)
		}
	}
	z := tx.Zeilen()
	if len(z) != 2 || z[0].X != 4 || z[1].X <= z[0].X || z[1].X+z[1].Vorschub != z[0].X+z[0].Vorschub {
		t.Fatalf("Zeilen rechtsbündig: %+v", z)
	}
	if n := strings.Count(svg, `<rect`); n != 4 { // ohne Hintergrund (A = 0): Unter- und Durchstrich je Zeile
		t.Errorf("%d Rechtecke, erwartet 4: %s", n, svg)
	}
}
