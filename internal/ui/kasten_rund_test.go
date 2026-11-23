// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/render"
)

// F-519: Die Außenkante eines gerundeten Kastens (Eckradius 6, Strich 8) ist in Renderer, SVG-Export und Editor
// dieselbe. Der Strich liegt mittig auf dem Pfad, die Außenecke hat also den Radius 6 + 4 = 10, die Innenecke 6 − 4 = 2.
// Der Editor zeichnet über Gio (nicht prüfbar ohne Fenster): Dort gilt die berechnete Größe (kastenRundung, Rechteck);
// das SVG sagt rx und stroke-width; der Renderer liefert die Pixel, die gegen den Radius aus Editor und SVG gehalten
// werden — Pixel, die ganz außerhalb des Außenkreises liegen, bleiben weiß, Pixel ganz im Strich sind dunkel,
// Pixel ganz im Innenkreis wieder weiß.
func TestKastenRundeKanteInRendererSvgUndEditor(t *testing.T) {
	const radius, breite = 6.0, 8.0
	bereich := document.R(document.Pt(30, 30), document.Pt(80, 70))
	k := document.Kasten{Bereich: bereich, Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: breite}, Eckradius: radius}

	b := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for i := range b.Pix {
		b.Pix[i] = 255
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}

	// Editor: ganzzahliges Rechteck bei Zoom 1 und der Radius, den Gio bekommt.
	r := image.Rect(30, 30, 80, 70)
	edR := kastenRundung(k, r)
	if edR != int(radius) {
		t.Fatalf("Editor-Radius %d, Soll %v", edR, radius)
	}
	if kastenStrich(k, r) != nil {
		t.Fatal("gerundeter Kasten zeichnet den eckigen Strich")
	}

	// SVG: rx, stroke-width, Rechteck und runde Strichecken.
	var svg bytes.Buffer
	if err := fileformat.ExportiereSVG(&svg, d); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`<rect[^>]*>`).Find(svg.Bytes())
	if m == nil {
		t.Fatalf("kein Kasten im SVG:\n%s", svg.String())
	}
	attr := func(name string) float64 {
		g := regexp.MustCompile(`\s` + regexp.QuoteMeta(name) + `="([^"]*)"`).FindSubmatch(m)
		if g == nil {
			t.Fatalf("Attribut %s fehlt in %s", name, m)
		}
		v, err := strconv.ParseFloat(string(g[1]), 64)
		if err != nil {
			t.Fatalf("Attribut %s: %v", name, err)
		}
		return v
	}
	if rx, sw := attr("rx"), attr("stroke-width"); rx != float64(edR) || sw != breite {
		t.Errorf("SVG rx %v, stroke-width %v; Editor-Radius %d, Strich %v", rx, sw, edR, breite)
	}
	if x, y, w, h := attr("x"), attr("y"), attr("width"), attr("height"); x != float64(r.Min.X) || y != float64(r.Min.Y) ||
		w != float64(r.Dx()) || h != float64(r.Dy()) {
		t.Errorf("SVG-Rechteck %v,%v %vx%v, Editor-Rechteck %v", x, y, w, h, r)
	}
	if !bytes.Contains(m, []byte(`stroke-linejoin="round"`)) {
		t.Errorf("SVG ohne runde Strichecken: %s", m)
	}

	// Renderer: Pixel der Ecke oben links gegen Außen- (R + 4) und Innenkreis (R − 4) um (36, 36).
	px, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	aussen, innen := float64(edR)+breite/2, float64(edR)-breite/2
	mx, my := float64(r.Min.X+edR), float64(r.Min.Y+edR)
	var dunkel, weiss int
	for y := r.Min.Y - 5; y < r.Min.Y+edR; y++ {
		for x := r.Min.X - 5; x < r.Min.X+edR; x++ {
			// Nächster und fernster Punkt des Pixels zum Mittelpunkt des Eckkreises (beide Achsen liegen vor ihm).
			nah := math.Hypot(math.Min(float64(x+1), mx)-mx, math.Min(float64(y+1), my)-my)
			fern := math.Hypot(float64(x)-mx, float64(y)-my)
			c := px.NRGBAAt(x, y)
			switch {
			case nah > aussen+0.2: // ganz außerhalb des Außenkreises
				if c.R < 250 {
					t.Errorf("Pixel (%d,%d) außerhalb der runden Außenkante: %v, Soll weiß", x, y, c)
				}
				weiss++
			case fern < aussen-0.2 && nah > innen+0.2:
				// ganz im Strich (Pixel, die ganz im Außenkreis und ganz außerhalb des Innenkreises liegen)
				if c.R > 20 {
					t.Errorf("Pixel (%d,%d) im Strich der Ecke: %v, Soll dunkel", x, y, c)
				}
				dunkel++
			}
		}
	}
	// Ohne Treffer prüfte die Schleife nichts (zum Beispiel nach einer Verschiebung der Ecke).
	if weiss < 10 || dunkel < 10 {
		t.Errorf("Eckprobe zu dünn: %d weiße, %d dunkle Pixel", weiss, dunkel)
	}
	// Innenecke: (35,35) liegt ganz im Innenkreis (Radius 2 um (36,36)) und ist weiß, (33,33) liegt im Strich.
	if c := px.NRGBAAt(35, 35); c.R < 250 {
		t.Errorf("Innenecke (35,35) %v, Soll weiß", c)
	}
	if c := px.NRGBAAt(33, 33); c.R > 20 {
		t.Errorf("Strich (33,33) %v, Soll dunkel", c)
	}
}
