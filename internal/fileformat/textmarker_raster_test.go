// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-1030 (Kritik #329 K-2): Das Export-SVG mit mix-blend-mode, gerastert mit rsvg-convert, zeigt den Textmarker wie der
// Renderer: Weiß wird Gelb, Schwarz bleibt Schwarz, Deckkraft 50 % halb so stark, auch mit Schatten und an einer Naht.
func TestExportTextmarkerGerastert(t *testing.T) {
	rsvg, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert nicht installiert")
	}
	for _, naht := range []bool{false, true} {
		b := image.NewNRGBA(image.Rect(0, 0, 200, 160))
		for y := 0; y < 160; y++ {
			for x := 0; x < 200; x++ {
				c := color.NRGBA{255, 255, 255, 255}
				if x >= 60 && x < 100 {
					c = color.NRGBA{0, 0, 0, 255}
				}
				b.SetNRGBA(x, y, c)
			}
		}
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		gelb := color.NRGBA{R: 0xFF, G: 0xE6, B: 0x28, A: 0xFF}
		mk := func(y float64) document.Pfeil {
			m := document.GeraderPfeil(document.Pt(10, y), document.Pt(150, y), document.Strich{Farbe: gelb, Breite: 20})
			m.Ende = document.KopfKeine
			m.Ebene.Mischung = document.MischMultiplizieren
			return m
		}
		m1 := mk(20)
		m2 := mk(60)
		m2.Ebene = m2.Ebene.MitDeckkraft(50)
		m3 := mk(100)
		m3.Ebene.Schatten = document.Schatten{An: true, Winkel: 90, Laenge: 4, Farbe: color.NRGBA{B: 200, A: 255}}
		for _, o := range []document.Objekt{m1, m2, m3} {
			if _, err := d.Hinzufuegen(o); err != nil {
				t.Fatal(err)
			}
		}
		if naht {
			if err := d.SetzeStreifen([]document.Streifen{{Von: 160, Bis: 180}}); err != nil {
				t.Fatal(err)
			}
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		svg, pngPfad := filepath.Join(dir, "e.svg"), filepath.Join(dir, "e.png")
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
		at := func(x, y int) [3]int {
			r, g, bl, _ := bild.At(x, y).RGBA()
			return [3]int{int(r >> 8), int(g >> 8), int(bl >> 8)}
		}
		pruefe := func(name string, x, y int, soll [3]int) {
			t.Helper()
			ist := at(x, y)
			for i := range ist {
				if ist[i]-soll[i] > 3 || soll[i]-ist[i] > 3 {
					t.Errorf("Naht %v, %s: %v, Soll %v", naht, name, ist, soll)
					return
				}
			}
		}
		pruefe("voll über Weiß", 30, 20, [3]int{255, 230, 40})
		pruefe("voll über Schwarz", 80, 20, [3]int{0, 0, 0})
		pruefe("50 % über Weiß", 30, 60, [3]int{255, 242, 147})
		pruefe("50 % über Schwarz", 80, 60, [3]int{0, 0, 0})
		pruefe("mit Schatten über Weiß", 30, 100, [3]int{255, 230, 40})
		pruefe("mit Schatten über Schwarz", 80, 100, [3]int{0, 0, 0})
	}
}
