// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 26 Punkt 6: s:schatten an jeder Objektart, Version 2, Rundlauf; in Version 1 und bei ungültigem
// Wert kein Schatten; ohne Schatten kein Attribut.
func TestSchattenLesen(t *testing.T) {
	d := beispielDok(t, 120, 90)
	sch := document.Schatten{An: true, Winkel: 120, Laenge: 7.5, Weich: 4, Farbe: color.NRGBA{R: 10, G: 20, B: 30, A: 90}}
	mit := func(o document.Objekt) document.Objekt { return document.MitEbene(o, document.Ebene{Schatten: sch}) }
	rot := color.NRGBA{R: 200, A: 255}
	fuege(t, d,
		mit(document.GeraderPfeil(document.Pt(4, 10), document.Pt(60, 10), document.Strich{Farbe: rot, Breite: 2})),
		mit(document.Kasten{Bereich: document.R(document.Pt(4, 20), document.Pt(50, 50)), Strich: document.Strich{Farbe: rot, Breite: 2}}),
		mit(document.Text{Ort: document.Pt(60, 20), Inhalt: "Hi", Schriftgrad: 10, Farbe: rot}),
		mit(document.Nummer{Mitte: document.Pt(90, 70), Wert: 1, Radius: 6, Farbe: rot}),
		document.Kasten{Bereich: document.R(document.Pt(60, 60), document.Pt(80, 80)), Strich: document.Strich{Farbe: rot, Breite: 2}})
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if strings.Count(svg, `s:schatten="120 7.5 4 #0a141e5a"`) != 4 || !strings.Contains(svg, `s:format="2"`) ||
		strings.Count(svg, "<filter") != 4 {
		t.Fatalf("Attribute, Filter oder Version falsch: %s", svg)
	}
	lies := func(p []byte) []document.Objekt {
		t.Helper()
		g, err := LeseArbeitsdatei(bytes.NewReader(p))
		if err != nil {
			t.Fatal(err)
		}
		return g.Dokument.Objekte()
	}
	o := lies(buf.Bytes())
	for i := range 4 {
		if got := o[i].Ebenenwerte().Schatten; got != sch {
			t.Errorf("Objekt %d: %+v", i, got)
		}
	}
	if o[4].Ebenenwerte().Schatten.An {
		t.Error("Kasten ohne Schatten hat einen")
	}
	for _, kaputt := range []string{"120 7.5 4", "x 7.5 4 #0a141e5a", "120 -1 4 #0a141e5a", "120 7.5 999 #0a141e5a", "120 7.5 4 red"} {
		o = lies(pngMitSVG(t, buf.Bytes(), strings.Replace(svg, `120 7.5 4 #0a141e5a`, kaputt, 1)))
		if o[0].Ebenenwerte().Schatten.An {
			t.Errorf("ungültiger Wert %q gilt als Schatten", kaputt)
		}
	}
	o = lies(pngMitSVG(t, buf.Bytes(), strings.Replace(svg, `s:format="2"`, `s:format="1"`, 1)))
	if o[0].Ebenenwerte().Schatten.An {
		t.Error("Version 1 liest den Schatten")
	}
	d2 := beispielDok(t, 64, 48)
	fuege(t, d2, document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 40)), Strich: document.Strich{Farbe: rot, Breite: 2}})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, d2, d2.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="1"`) || strings.Contains(svg, "schatten") {
		t.Errorf("ohne Schatten: %s", svg)
	}
}

// Das exportierte SVG zeigt den Schatten (rsvg-convert): versetzt nach rechts unten, gegen das Licht nichts.
func TestSVGSchattenSichtbar(t *testing.T) {
	rsvg, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert nicht installiert")
	}
	b := image.NewNRGBA(image.Rect(0, 0, 200, 160))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Bereich: document.R(document.Pt(40, 40), document.Pt(100, 80)), Fuellung: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}
	k.Ebene.Schatten = document.Schatten{An: true, Winkel: 135, Laenge: 14.142135623730951, Weich: 2, Farbe: color.NRGBA{A: 255}}
	if _, err := d.Hinzufuegen(k); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	ordner := t.TempDir()
	svg, pngPfad := filepath.Join(ordner, "s.svg"), filepath.Join(ordner, "s.png")
	if err := os.WriteFile(svg, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(rsvg, "-o", pngPfad, svg).CombinedOutput(); err != nil {
		t.Fatalf("rsvg-convert: %v %s", err, out)
	}
	f, err := os.Open(pngPfad)
	if err != nil {
		t.Fatal(err)
	}
	bild, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	grau := func(x, y int) int { r, _, _, _ := bild.At(x, y).RGBA(); return int(r >> 8) }
	if g := grau(105, 85); g > 60 {
		t.Errorf("Schatten unten rechts %d", g)
	}
	if g := grau(70, 60); g < 250 {
		t.Errorf("Kasten über dem Schatten %d", g)
	}
	if g := grau(35, 35); g < 250 {
		t.Errorf("oben links %d (kein Schatten gegen das Licht)", g)
	}
}
