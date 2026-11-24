// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// TestSprechblaseFormat: s:spitze gehört zu Formatversion 2 (ADR-0021 Nachtrag 13 Punkt 7) — das Beispiel liest sich, in
// Version 1 ist das Attribut unbekannter Inhalt (ignoriert, kein Schaden), ein unlesbarer Punkt macht Version 2
// beschädigt, und LeseMetadaten urteilt jedes Mal wie das volle Lesen.
func TestSprechblaseFormat(t *testing.T) {
	png, svg := beispielSVG(t, "callout")
	if !strings.Contains(svg, `s:format="2"`) || strings.Count(svg, ` s:spitze="`) != 2 {
		t.Fatalf("Beispiel: %s", svg)
	}
	lies := func(s string) (*Gelesen, string) {
		daten := pngMitSVG(t, png, s)
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		_, kopf := LeseMetadaten(bytes.NewReader(daten))
		if urteil(kopf) != urteil(err) {
			t.Errorf("LeseMetadaten %q, voll %q", urteil(kopf), urteil(err))
		}
		return g, urteil(err)
	}
	spitzen := func(g *Gelesen) (n int) {
		for _, o := range g.Dokument.Objekte() {
			if tx, ok := o.(document.Text); ok && tx.Spitze != nil {
				n++
			}
		}
		return n
	}

	g, u := lies(svg)
	if u != "ok" || spitzen(g) != 2 {
		t.Fatalf("Version 2: %s, %d Spitzen", u, spitzen(g))
	}
	if sp := g.Dokument.Objekte()[1].(document.Text).Spitze; sp == nil || *sp != (document.Punkt{X: 12, Y: 22.5}) {
		t.Errorf("Spitze des zweiten Textes: %v", sp)
	}

	v1 := strings.Replace(svg, `s:format="2"`, `s:format="1"`, 1)
	if g, u := lies(v1); u != "ok" || spitzen(g) != 0 || len(g.Dokument.Objekte()) != 2 {
		t.Errorf("Version 1 mit s:spitze: %s, %d Spitzen (Soll ok, keine Spitze, beide Texte)", u, spitzen(g))
	}
	// Auch ein unlesbarer Wert ist in Version 1 unbekannter Inhalt.
	v1kaputt := strings.Replace(v1, `s:spitze="40,30"`, `s:spitze="x"`, 1)
	if _, u := lies(v1kaputt); u != "ok" {
		t.Errorf("Version 1 mit kaputtem s:spitze: %s", u)
	}

	for name, f := range map[string]string{
		"ein Wert":     strings.Replace(svg, `s:spitze="40,30"`, `s:spitze="40"`, 1),
		"Text":         strings.Replace(svg, `s:spitze="40,30"`, `s:spitze="a,b"`, 1),
		"leer":         strings.Replace(svg, `s:spitze="40,30"`, `s:spitze=""`, 1),
		"zu groß":      strings.Replace(svg, `s:spitze="40,30"`, `s:spitze="2e6,30"`, 1),
		"mit Leerraum": strings.Replace(svg, `s:spitze="40,30"`, `s:spitze="40, 30"`, 1),
	} {
		if _, u := lies(f); u != "beschädigt" {
			t.Errorf("%s: %s, Soll beschädigt", name, u)
		}
	}
}

// TestSprechblaseVersionWahl: Die kleinste Version, die den Inhalt trägt: 2 mit einer Spitze (auch ohne Lupe), 1 ohne.
func TestSprechblaseVersionWahl(t *testing.T) {
	sp := document.Pt(10, 10)
	tx := document.Text{Ort: document.Pt(30, 20), Inhalt: "x", Schriftgrad: 9}
	if v := schreibVersion([]document.Objekt{tx}); v != 1 {
		t.Errorf("Text ohne Spitze: Version %d", v)
	}
	tx.Spitze = &sp
	if v := schreibVersion([]document.Objekt{tx}); v != 2 {
		t.Errorf("Text mit Spitze: Version %d", v)
	}
	if v := schreibVersion([]document.Objekt{document.Nummer{}, tx}); v != 2 {
		t.Errorf("Spitze nach anderem Objekt: Version %d", v)
	}
}

// TestSprechblaseExport: Der SVG-Export zeigt die Blase als Pfad, ohne Daten (`s:`) und mit spitzen Ecken; ohne Füllung
// stehen die Schenkel als Linie in der Schriftfarbe.
func TestSprechblaseExport(t *testing.T) {
	_, svg := beispielSVG(t, "callout")
	if strings.Count(svg, "<path") != 2 {
		t.Fatalf("Pfade im Beispiel: %s", svg)
	}
	d := beispielDok(t, 64, 48)
	sp1, sp2 := document.Pt(40, 30), document.Pt(12, 22.5)
	fuege(t, d,
		document.Text{Ort: document.Pt(4, 4), Inhalt: "Look", Schriftgrad: 9, Hintergrund: beispielBlau, Spitze: &sp1},
		document.Text{Ort: document.Pt(24, 34), Inhalt: "Here", Schriftgrad: 9, Farbe: beispielBlau, Spitze: &sp2})
	var puffer bytes.Buffer
	if err := ExportiereSVG(&puffer, d); err != nil {
		t.Fatal(err)
	}
	out := puffer.String()
	if strings.Contains(out, "s:") || strings.Contains(out, "urn:x-schnapp") {
		t.Errorf("Export enthält Daten des Formats: %s", out)
	}
	if strings.Count(out, "<path") != 2 || !strings.Contains(out, `stroke-linejoin="miter"`) ||
		!strings.Contains(out, `fill="none"`) || !strings.Contains(out, `stroke-width="1"`) {
		t.Errorf("Export ohne Blasenpfade: %s", out)
	}
	// Die Spitze steht im Pfad.
	if !strings.Contains(out, "L40,30") || !strings.Contains(out, "L12,22.5") {
		t.Errorf("Spitzen fehlen im Pfad: %s", out)
	}
}

// TestSprechblaseRundlauf: Schreiben und Lesen einer Arbeitsdatei erhält die Spitze; eine Spitze innerhalb des Schilds
// (kein Dreieck) bleibt erhalten und wird als Schild gezeichnet.
func TestSprechblaseRundlauf(t *testing.T) {
	d := beispielDok(t, 64, 48)
	innen := document.Pt(10, 8)
	aussen := document.Pt(-20, 100.5) // außerhalb des Bilds
	fuege(t, d,
		document.Text{Ort: document.Pt(4, 4), Inhalt: "Hi", Schriftgrad: 12, Hintergrund: beispielBlau, Spitze: &innen},
		document.Text{Ort: document.Pt(30, 20), Inhalt: "Yo", Schriftgrad: 12, Hintergrund: beispielBlau, Spitze: &aussen})
	var puffer bytes.Buffer
	if err := SchreibeArbeitsdatei(&puffer, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(puffer.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	o := g.Dokument.Objekte()
	if len(o) != 2 || *o[0].(document.Text).Spitze != innen || *o[1].(document.Text).Spitze != aussen {
		t.Errorf("Spitzen nach dem Rundlauf: %#v", o)
	}
}
