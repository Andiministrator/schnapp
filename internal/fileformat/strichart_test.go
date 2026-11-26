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

// ADR-0021 Nachtrag 25 Punkt 6: s:strichart kommt an Pfeil, Kasten (eckig, rund, ohne Fläche) und Lupe an, hebt die
// Version auf 2, fehlt bei voll; unbekannt und in Version 1 gilt voll.
func TestStrichartLesen(t *testing.T) {
	d := beispielDok(t, 120, 90)
	st := func(a document.Strichart) document.Strich {
		return document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2, Art: a}
	}
	fuege(t, d,
		document.GeraderPfeil(document.Pt(4, 10), document.Pt(60, 10), st(document.StrichGestrichelt)),
		document.Kasten{Bereich: document.R(document.Pt(4, 20), document.Pt(50, 50)), Strich: st(document.StrichGepunktet)},
		document.Kasten{Bereich: document.R(document.Pt(60, 20), document.Pt(110, 50)), Strich: st(document.StrichGestrichelt),
			Eckradius: 8, Fuellung: color.NRGBA{G: 200, A: 255}},
		document.Kasten{Bereich: document.R(document.Pt(60, 60), document.Pt(110, 60)), Strich: st(document.StrichGepunktet)},
		document.Lupe{Quelle: document.R(document.Pt(4, 60), document.Pt(14, 70)), Ziel: document.R(document.Pt(20, 60),
			document.Pt(40, 80)), Strich: st(document.StrichGestrichelt)},
		document.GeraderPfeil(document.Pt(4, 85), document.Pt(60, 85), st(document.StrichVoll)))
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg := string(entpackeSVG(t, buf.Bytes()))
	if strings.Count(svg, "s:strichart=") != 5 || !strings.Contains(svg, `s:format="2"`) {
		t.Fatalf("Attribute oder Version falsch: %s", svg)
	}
	lies := func(png []byte) []document.Objekt {
		t.Helper()
		g, err := LeseArbeitsdatei(bytes.NewReader(png))
		if err != nil {
			t.Fatal(err)
		}
		return g.Dokument.Objekte()
	}
	art := func(o document.Objekt) document.Strichart {
		switch v := o.(type) {
		case document.Pfeil:
			return v.Strich.Art
		case document.Kasten:
			return v.Strich.Art
		case document.Lupe:
			return v.Strich.Art
		}
		return 99
	}
	soll := []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet, document.StrichGestrichelt,
		document.StrichGepunktet, document.StrichGestrichelt, document.StrichVoll}
	o := lies(buf.Bytes())
	for i, a := range soll {
		if art(o[i]) != a {
			t.Errorf("Objekt %d: %v statt %v", i, art(o[i]), a)
		}
		if !document.Gleich(o[i], d.Objekte()[i]) {
			t.Errorf("Objekt %d nach dem Rundlauf anders: %+v", i, o[i])
		}
	}
	fremd := strings.ReplaceAll(svg, `s:strichart="dashed"`, `s:strichart="wavy"`)
	o = lies(pngMitSVG(t, buf.Bytes(), fremd))
	if art(o[0]) != document.StrichVoll || art(o[1]) != document.StrichGepunktet {
		t.Errorf("unbekannter Wert: %v, %v", art(o[0]), art(o[1]))
	}
	// In Version 1 gilt das Attribut nicht (die Lupe gibt es erst ab 2: ohne sie lesen).
	d1 := beispielDok(t, 120, 90)
	fuege(t, d1, document.GeraderPfeil(document.Pt(4, 10), document.Pt(60, 10), st(document.StrichGestrichelt)))
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, d1, d1.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	svg1 := string(entpackeSVG(t, buf.Bytes()))
	if !strings.Contains(svg1, `s:format="2"`) {
		t.Fatalf("gestrichelter Pfeil allein hebt die Version nicht: %s", svg1)
	}
	o = lies(pngMitSVG(t, buf.Bytes(), strings.Replace(svg1, `s:format="2"`, `s:format="1"`, 1)))
	if art(o[0]) != document.StrichVoll {
		t.Errorf("Version 1: %v", art(o[0]))
	}
	// Ein gemusterter Kasten allein hebt die Version.
	dk := beispielDok(t, 64, 48)
	fuege(t, dk, document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 40)), Strich: st(document.StrichGepunktet)})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, dk, dk.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="2"`) {
		t.Errorf("gepunkteter Kasten allein hebt die Version nicht: %s", svg)
	}
	// Nur volle Striche: Version 1, kein Attribut.
	d2 := beispielDok(t, 64, 48)
	fuege(t, d2, document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 40)), Strich: st(document.StrichVoll)})
	buf.Reset()
	if err := SchreibeArbeitsdatei(&buf, d2, d2.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if svg := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(svg, `s:format="1"`) || strings.Contains(svg, "s:strichart") {
		t.Errorf("voller Strich: %s", svg)
	}
}

// Das exportierte SVG zeigt die Stücke an derselben Stelle wie document (rsvg-convert): Mitte jedes Strichs bzw. Punkts
// gedeckt, Mitte jeder Lücke frei — gestrichelter Pfeil mit Kopf, gepunkteter eckiger und gestrichelter runder Kasten.
func TestSVGStrichartLage(t *testing.T) {
	rsvg, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert nicht installiert")
	}
	b := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	rot := color.NRGBA{R: 220, A: 255}
	pfeil := document.GeraderPfeil(document.Pt(20, 20), document.Pt(280, 20),
		document.Strich{Farbe: rot, Breite: 4, Art: document.StrichGestrichelt})
	eckig := document.Kasten{Bereich: document.R(document.Pt(20, 50), document.Pt(140, 180)),
		Strich: document.Strich{Farbe: rot, Breite: 4, Art: document.StrichGepunktet}}
	rund := document.Kasten{Bereich: document.R(document.Pt(160, 50), document.Pt(280, 180)), Eckradius: 15,
		Strich: document.Strich{Farbe: rot, Breite: 4, Art: document.StrichGestrichelt}}
	for _, o := range []document.Objekt{pfeil, eckig, rund} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "dasharray") {
		t.Error("SVG nutzt stroke-dasharray")
	}
	ordner := t.TempDir()
	svg, pngPfad := filepath.Join(ordner, "muster.svg"), filepath.Join(ordner, "muster.png")
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
	gruen := func(p document.Punkt) int {
		_, g, _, _ := bild.At(int(p.X), int(p.Y)).RGBA()
		return int(g >> 8)
	}
	pruefe := func(name string, voll, frei []document.Punkt) {
		for _, p := range voll {
			if g := gruen(p); g > 60 {
				t.Errorf("%s: %v nicht gedeckt (Grün %d)", name, p, g)
			}
		}
		for _, p := range frei {
			if g := gruen(p); g < 200 {
				t.Errorf("%s: Lücke bei %v gedeckt (Grün %d)", name, p, g)
			}
		}
	}
	// Pfeil: Schaft gestrichelt bis unter den Kopf.
	f0 := pfeil.Form()
	m := document.StrichAbschnitte(f0.Schaft, document.StrichGestrichelt, 4)
	var voll, frei []document.Punkt
	for i, z := range m.Striche {
		voll = append(voll, z[0].Plus(z[len(z)-1]).Mal(0.5))
		if i > 0 {
			frei = append(frei, z[0].Plus(m.Striche[i-1][len(m.Striche[i-1])-1]).Mal(0.5))
		}
	}
	pruefe("Pfeil", voll, frei)
	// Eckiger Kasten gepunktet: Punkte gedeckt, dazwischen frei.
	_, punkte := document.EckigerRahmenMuster(eckig.Bereich, 4, document.StrichGepunktet)
	voll, frei = punkte, nil
	for i := 1; i < len(punkte); i++ {
		if punkte[i].X == punkte[i-1].X || punkte[i].Y == punkte[i-1].Y {
			frei = append(frei, punkte[i].Plus(punkte[i-1]).Mal(0.5))
		}
	}
	pruefe("eckig", voll, frei)
	// Runder Kasten gestrichelt.
	mr := document.RundRahmenAbschnitte(rund.Bereich, 15, document.StrichGestrichelt, 4)
	voll, frei = nil, nil
	for i, z := range mr.Striche {
		if len(z) == 2 {
			voll = append(voll, z[0].Plus(z[1]).Mal(0.5))
		} else {
			voll = append(voll, z[len(z)/2]) // ein innerer Punkt des Bogens
		}
		if i > 0 {
			a, e := m0(mr.Striche[i-1]), z[0]
			if a.Minus(e).Laenge() > 4 {
				frei = append(frei, a.Plus(e).Mal(0.5))
			}
		}
	}
	pruefe("rund", voll, frei)
}

func m0(z []document.Punkt) document.Punkt { return z[len(z)-1] }

// F-841: In der Arbeitsdatei steht ein gemustertes Objekt als Näherung (stroke-dasharray), nicht Stück für Stück; ein
// SVG über der Lesegrenze lehnt schon der Schreiber ab, statt eine unlesbare Datei zu schreiben.
func TestStrichartArbeitsdateiKlein(t *testing.T) {
	d := beispielDok(t, 64, 48)
	leer := len(AngabenSVG(d))
	fuege(t, d, document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(8000, 8000)),
		Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1, Art: document.StrichGepunktet}})
	if n := len(AngabenSVG(d)) - leer; n > 1000 {
		t.Errorf("gepunkteter Kasten 8000 × 8000: %d Bytes in den Angaben", n)
	}
	var export bytes.Buffer
	if err := ExportiereSVG(&export, d); err != nil || !strings.Contains(export.String(), " a0.5 0.5 0 1 0 1 0") {
		t.Errorf("Export schreibt die Punkte nicht aus: %v", err)
	}
	alt := schreibGrenzeSVG
	defer func() { schreibGrenzeSVG = alt }()
	schreibGrenzeSVG = leer + 10
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err == nil || !strings.Contains(err.Error(), "MiB") {
		t.Errorf("über der Grenze: %v", err)
	}
}
