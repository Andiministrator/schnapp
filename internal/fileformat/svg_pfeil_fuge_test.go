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

// F-826: Im SVG sind Schaft, Kopf und halbe Scheibe eines Pfeils getrennte Formen. Gerastert (rsvg-convert, 12-fach)
// darf entlang der Mittellinie keine helle Fuge stehen, und eine durchscheinende Farbe deckt die Überlappung nicht
// doppelt: Jedes Pixel der Mittellinie zwischen den Enden hat die Farbe der Schaftmitte.
func TestSVGPfeilOhneFuge(t *testing.T) {
	rsvg, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert nicht installiert")
	}
	const zoom = 12
	for name, fall := range map[string]struct {
		farbe        color.NRGBA
		anfang, ende document.Kopfart
	}{
		"deckend, Kopf bei B":             {color.NRGBA{220, 30, 30, 255}, document.KopfKeine, document.KopfDreieck},
		"durchscheinend, Kopf bei B":      {color.NRGBA{220, 30, 30, 128}, document.KopfKeine, document.KopfDreieck},
		"durchscheinend, Kopf bei A":      {color.NRGBA{20, 90, 200, 128}, document.KopfDreieck, document.KopfKeine},
		"durchscheinend, beide, Balken":   {color.NRGBA{20, 90, 200, 160}, document.KopfQuer, document.KopfQuer},
		"deckend, beide, Raute und Kerbe": {color.NRGBA{20, 90, 200, 255}, document.KopfRaute, document.KopfKerbe},
	} {
		b := image.NewNRGBA(image.Rect(0, 0, 80, 40))
		draw.Draw(b, b.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		// Die Enden liegen nicht auf ganzen Pixeln, damit die Kanten mitten in Pixeln stehen.
		p := document.Pfeil{A: document.Pt(10.3, 20), B: document.Pt(69.6, 20), K: document.Pt(40, 20),
			Strich: document.Strich{Farbe: fall.farbe, Breite: 4}}.MitKoepfen(fall.anfang, fall.ende)
		if _, err := d.Hinzufuegen(p); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		if fall.farbe.A < 255 && !strings.Contains(buf.String(), `<g opacity="`) {
			t.Errorf("%s: durchscheinender Pfeil ohne Gruppendeckkraft", name)
		}
		ordner := t.TempDir()
		svg, pngPfad := filepath.Join(ordner, "pfeil.svg"), filepath.Join(ordner, "pfeil.png")
		if err := os.WriteFile(svg, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(rsvg, "-z", "12", "-o", pngPfad, svg).CombinedOutput(); err != nil {
			t.Fatalf("%s: rsvg-convert: %v %s", name, err, out)
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
		y := 20 * zoom // Mittellinie des Pfeils
		farbe := func(x int) [3]int {
			r, g, b, _ := bild.At(x, y).RGBA()
			return [3]int{int(r >> 8), int(g >> 8), int(b >> 8)}
		}
		mitte := farbe(40 * zoom)
		// Von knapp hinter A bis knapp vor B (Spitzen sind dort noch breit genug): überall dieselbe Farbe.
		for x := 13 * zoom; x <= 67*zoom; x++ {
			c := farbe(x)
			for i := range c {
				if diff := c[i] - mitte[i]; diff < -3 || diff > 3 {
					t.Errorf("%s: Mittellinie bei x = %.2f (Original): %v, Schaftmitte %v", name, float64(x)/zoom, c, mitte)
					goto weiter
				}
			}
		}
	weiter:
	}
}
