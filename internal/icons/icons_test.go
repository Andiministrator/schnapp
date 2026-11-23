// SPDX-License-Identifier: GPL-3.0-or-later

package icons

import (
	"math"
	"os"
	"strings"
	"testing"
)

// TestSymboleImRaster: Jedes Symbol hat Teile, jedes Teil mindestens drei Punkte, alle Punkte liegen im Raster, und
// jeder Name kommt einmal vor.
func TestSymboleImRaster(t *testing.T) {
	namen := map[string]bool{}
	n := 0
	for _, g := range Gruppen() {
		for _, s := range g.Symbole {
			n++
			if s.Name == "" || namen[s.Name] {
				t.Errorf("Name leer oder doppelt: %q", s.Name)
			}
			namen[s.Name] = true
			if len(s.Teile) == 0 {
				t.Errorf("%s: keine Teile", s.Name)
			}
			for _, tl := range s.Teile {
				if len(tl.Punkte) < 3 {
					t.Errorf("%s: Teil mit %d Punkten", s.Name, len(tl.Punkte))
				}
				for _, p := range tl.Punkte {
					if p.X < 0 || p.Y < 0 || p.X > Raster || p.Y > Raster {
						t.Errorf("%s: Punkt %v außerhalb des Rasters", s.Name, p)
					}
				}
			}
		}
	}
	if n < 40 {
		t.Errorf("nur %d Symbole", n)
	}
}

// TestLinieEckig: Eine waagrechte Linie wird ein Rechteck, das um die halbe Strichbreite über die Enden hinausreicht.
func TestLinieEckig(t *testing.T) {
	l := linie(4, 10, 20, 10)
	want := []Punkt{{3, 11}, {21, 11}, {21, 9}, {3, 9}}
	for i, p := range l.Punkte {
		if p != want[i] {
			t.Fatalf("Punkte %v, erwartet %v", l.Punkte, want)
		}
	}
}

// TestSVG: gültiger Rahmen, Akzentteile in der Akzentfarbe.
func TestSVG(t *testing.T) {
	s := Farbe.SVG("#ECEAE5", "#F5A65B")
	if !strings.HasPrefix(s, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">`) || !strings.HasSuffix(s, "</svg>") {
		t.Fatalf("Rahmen: %s", s)
	}
	if strings.Count(s, `fill="#F5A65B"`) != 1 || strings.Count(s, `fill="#ECEAE5"`) != 4 {
		t.Errorf("Farben: %s", s)
	}
}

// TestLogoSWieMarke: Pfad, Strich, viewBox und Auge des „s“ stimmen mit der Datei aus brand/werkzeug überein; die
// Punktfolge beginnt und endet an den Enden des Pfads und läuft durch die äußersten Punkte der Bögen.
func TestLogoSWieMarke(t *testing.T) {
	b, err := os.ReadFile("../../brand/logo/svg/schnapp-s-dunkel.svg")
	if err != nil {
		t.Fatal(err)
	}
	svg := string(b)
	for _, teil := range []string{`viewBox="-16 84 108 132"`, `d="` + LogoSPfad + `"`, `stroke-width="22"`,
		`<circle cx="30" cy="125" r="11"`, `<circle cx="34.0" cy="125" r="5.5"`} {
		if !strings.Contains(svg, teil) {
			t.Errorf("Logo-Datei enthält %q nicht", teil)
		}
	}
	l := LogoSLinie(16)
	if l[0] != (Punkt{76, 100}) || l[len(l)-1] != (Punkt{0, 200}) {
		t.Errorf("Enden: %v … %v", l[0], l[len(l)-1])
	}
	minX, maxX := 1e9, -1e9
	for _, p := range l {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
	}
	if math.Abs(minX) > 1e-9 || math.Abs(maxX-76) > 1e-9 {
		t.Errorf("Bögen: x von %v bis %v, erwartet 0 bis 76", minX, maxX)
	}
}
