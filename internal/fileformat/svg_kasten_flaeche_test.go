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
	"github.com/Andiministrator/schnapp/internal/render"
)

// kastenOhneFlaecheDokument ist ein weißes Bild mit einem Kasten ohne Fläche (Breite oder Höhe 0, auch beides).
func kastenOhneFlaecheDokument(t *testing.T, bereich document.Rechteck, strich document.Strich, fuellung color.NRGBA) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 120, 80))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: bereich, Strich: strich, Fuellung: fuellung, Eckradius: 5}); err != nil {
		t.Fatal(err)
	}
	return d
}

// Kritik A-5: Ein Kasten mit Breite oder Höhe 0 (ein Griff zieht eine Kante auf die gegenüberliegende) zeichnet der
// Renderer als Balken. Das SVG schreibt ihn als gefüllten Pfad statt als `<rect>` mit Maß 0 (im SVG unsichtbar); die Daten
// bleiben, und die Arbeitsdatei liest ihn unverändert zurück.
func TestSVGKastenOhneFlaeche(t *testing.T) {
	strich := document.Strich{Farbe: color.NRGBA{200, 30, 40, 255}, Breite: 8}
	for name, bereich := range map[string]document.Rechteck{
		"Höhe 0":   document.R(document.Pt(20, 40), document.Pt(100, 40)),
		"Breite 0": document.R(document.Pt(60, 10), document.Pt(60, 70)),
		"Punkt":    document.R(document.Pt(60, 40), document.Pt(60, 40)),
	} {
		d := kastenOhneFlaecheDokument(t, bereich, strich, color.NRGBA{0, 0, 255, 255})
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		// Der Export trägt nur die Darstellung: ein gefüllter Pfad in der Strichfarbe statt eines <rect> mit Maß 0.
		if strings.Contains(s, `<rect`) {
			t.Errorf("%s: noch ein <rect> mit Maß 0 (unsichtbar im SVG)", name)
		}
		if !strings.Contains(s, `<path d="M`) || !strings.Contains(s, `fill="#c81e28"`) {
			t.Errorf("%s: kein gefüllter Pfad in der Strichfarbe:\n%.600s", name, s)
		}
		// Rundreise über die Arbeitsdatei: dieselben Daten.
		var datei bytes.Buffer
		if err := SchreibeArbeitsdatei(&datei, d, d.Ansichtsbild(), Metadaten{}); err != nil {
			t.Fatal(err)
		}
		g, err := LeseArbeitsdatei(&datei)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, soll := g.Dokument.Objekte(), d.Objekte(); len(got) != 1 || got[0].(document.Kasten).Bereich != soll[0].(document.Kasten).Bereich ||
			got[0].(document.Kasten).Strich != strich {
			t.Errorf("%s: gelesen %+v, Soll %+v", name, got, soll)
		}
	}
	// Ohne sichtbaren Strich bleibt die Gruppe leer (nichts zu zeichnen, auch keine Fläche).
	d := kastenOhneFlaecheDokument(t, document.R(document.Pt(20, 40), document.Pt(100, 40)), document.Strich{Farbe: color.NRGBA{A: 0}, Breite: 8}, color.NRGBA{})
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `<path`) {
		t.Errorf("Kasten ohne sichtbaren Strich zeichnet einen Pfad")
	}
}

// Kritik A-5: Das gerasterte SVG (rsvg-convert, nur Dateien unter dem Testordner) ist dem Bild des Renderers gleich:
// der Balken steht an derselben Stelle (Pixel höchstens leicht verschieden von der Kantenglättung).
func TestSVGKastenOhneFlaecheWieRenderer(t *testing.T) {
	rsvg, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert nicht installiert")
	}
	strich := document.Strich{Farbe: color.NRGBA{0, 0, 0, 255}, Breite: 8}
	for name, bereich := range map[string]document.Rechteck{
		"Höhe 0":   document.R(document.Pt(20, 40), document.Pt(100, 40)),
		"Breite 0": document.R(document.Pt(60, 10), document.Pt(60, 70)),
	} {
		d := kastenOhneFlaecheDokument(t, bereich, strich, color.NRGBA{})
		soll, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		ordner := t.TempDir()
		svg := filepath.Join(ordner, "kasten.svg")
		pngPfad := filepath.Join(ordner, "kasten.png")
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(svg, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(rsvg, "-o", pngPfad, svg).CombinedOutput(); err != nil {
			t.Fatalf("%s: rsvg-convert: %v %s", name, err, out)
		}
		f, err := os.Open(pngPfad)
		if err != nil {
			t.Fatal(err)
		}
		ist, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if ist.Bounds() != soll.Bounds() {
			t.Fatalf("%s: Größe %v, Soll %v", name, ist.Bounds(), soll.Bounds())
		}
		dunkel := 0
		for y := 0; y < soll.Rect.Dy(); y++ {
			for x := 0; x < soll.Rect.Dx(); x++ {
				a := soll.NRGBAAt(x, y)
				r, g, b, _ := ist.At(x, y).RGBA()
				diff := int(a.R) - int(r>>8)
				if diff < -3 || diff > 3 || int(a.G)-int(g>>8) > 3 || int(a.G)-int(g>>8) < -3 || int(a.B)-int(b>>8) > 3 || int(a.B)-int(b>>8) < -3 {
					t.Errorf("%s: Pixel (%d,%d): Renderer %v, SVG %d,%d,%d", name, x, y, a, r>>8, g>>8, b>>8)
					return
				}
				if a.R < 100 {
					dunkel++
				}
			}
		}
		if dunkel < 100 {
			t.Errorf("%s: nur %d dunkle Pixel im Renderer, der Balken fehlt?", name, dunkel)
		}
	}
}
