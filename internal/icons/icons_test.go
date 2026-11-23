// SPDX-License-Identifier: GPL-3.0-or-later

package icons

import (
	"math"
	"os"
	"regexp"
	"strconv"
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

// zahlen liest die Zahlen aus s (durch Kommas, Leerzeichen und Buchstaben getrennt).
func zahlen(t *testing.T, s string) []float64 {
	t.Helper()
	var r []float64
	for _, m := range regexp.MustCompile(`-?[0-9]+(?:\.[0-9]+)?`).FindAllString(s, -1) {
		f, err := strconv.ParseFloat(m, 64)
		if err != nil {
			t.Fatal(err)
		}
		r = append(r, f)
	}
	return r
}

// TestLogoSWieMarke (F-511): Die Maße kommen aus der Datei brand/logo/svg/schnapp-s-dunkel.svg und werden mit den
// Konstanten verglichen, nicht mit Literalen im Test: Pfad, Strichbreite, viewBox, Augapfel und Pupille; die
// Punktfolge LogoSLinie liegt auf dem Pfad der Datei (Geraden und die beiden Halbkreise mit ihrer Seite).
func TestLogoSWieMarke(t *testing.T) {
	b, err := os.ReadFile("../../brand/logo/svg/schnapp-s-dunkel.svg")
	if err != nil {
		t.Fatal(err)
	}
	svg := string(b)
	attr := func(name, in string) string {
		m := regexp.MustCompile(name + `="([^"]*)"`).FindStringSubmatch(in)
		if m == nil {
			t.Fatalf("Attribut %s fehlt in %s", name, in)
		}
		return m[1]
	}
	if z := zahlen(t, attr("viewBox", svg)); len(z) != 4 || [4]float64{z[0], z[1], z[2], z[3]} != LogoSBox {
		t.Errorf("viewBox der Datei %v, LogoSBox %v", z, LogoSBox)
	}
	pfad := regexp.MustCompile(`<path [^>]*>`).FindString(svg)
	if d := attr("d", pfad); d != LogoSPfad {
		t.Errorf("Pfad der Datei %q, LogoSPfad %q", d, LogoSPfad)
	}
	if z := zahlen(t, attr("stroke-width", pfad)); len(z) != 1 || z[0] != LogoSStrich {
		t.Errorf("Strichbreite der Datei %v, LogoSStrich %v", z, LogoSStrich)
	}
	kreise := regexp.MustCompile(`<circle [^>]*>`).FindAllString(svg, -1)
	if len(kreise) != 2 {
		t.Fatalf("%d Kreise in der Datei, erwartet Augapfel und Pupille", len(kreise))
	}
	for i, soll := range [][3]float64{LogoSAugapfel, LogoSPupille} {
		k := kreise[i]
		got := [3]float64{zahlen(t, attr("cx", k))[0], zahlen(t, attr("cy", k))[0], zahlen(t, attr("r", k))[0]}
		if got != soll {
			t.Errorf("Kreis %d der Datei %v, Konstante %v", i, got, soll)
		}
	}

	// Pfad lesen: M x,y H x A r,r 0 0 sweep x,y H x A … H x; der Linienzug muss darauf liegen.
	type stueck struct {
		von, nach Punkt
		bogen     bool
		sweep     float64
		mitte     Punkt
		radius    float64
	}
	var st []stueck
	cur := Punkt{}
	for _, m := range regexp.MustCompile(`[MHA][^MHA]*`).FindAllString(LogoSPfad, -1) {
		z := zahlen(t, m[1:])
		switch m[0] {
		case 'M':
			cur = Punkt{z[0], z[1]}
		case 'H':
			st = append(st, stueck{von: cur, nach: Punkt{z[0], cur.Y}})
			cur = Punkt{z[0], cur.Y}
		case 'A':
			nach := Punkt{z[5], z[6]}
			mitte := Punkt{(cur.X + nach.X) / 2, (cur.Y + nach.Y) / 2}
			if r := math.Hypot(nach.X-cur.X, nach.Y-cur.Y) / 2; math.Abs(r-z[0]) > 1e-9 {
				t.Fatalf("Bogen ist kein Halbkreis: Radius %v, Abstand/2 %v", z[0], r)
			}
			st = append(st, stueck{von: cur, nach: nach, bogen: true, sweep: z[4], mitte: mitte, radius: z[0]})
			cur = nach
		}
	}
	l := LogoSLinie(16)
	if l[0] != st[0].von || l[len(l)-1] != st[len(st)-1].nach {
		t.Errorf("Enden: %v … %v, Pfad %v … %v", l[0], l[len(l)-1], st[0].von, st[len(st)-1].nach)
	}
	auf := func(p Punkt) bool {
		for _, s := range st {
			if !s.bogen {
				if math.Abs(p.Y-s.von.Y) < 1e-9 && p.X >= math.Min(s.von.X, s.nach.X)-1e-9 && p.X <= math.Max(s.von.X, s.nach.X)+1e-9 {
					return true
				}
				continue
			}
			// Sweep 0 läuft hier links herum (Bogen nach links, x <= Mitte), Sweep 1 rechts herum (x >= Mitte).
			seite := p.X <= s.mitte.X+1e-9
			if s.sweep == 1 {
				seite = p.X >= s.mitte.X-1e-9
			}
			if seite && math.Abs(math.Hypot(p.X-s.mitte.X, p.Y-s.mitte.Y)-s.radius) < 1e-9 {
				return true
			}
		}
		return false
	}
	minX, maxX := 1e9, -1e9
	for i, p := range l {
		if !auf(p) {
			t.Errorf("Punkt %d %v liegt nicht auf dem Pfad der Datei", i, p)
		}
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
	}
	if math.Abs(minX) > 1e-9 || math.Abs(maxX-76) > 1e-9 {
		t.Errorf("Bögen: x von %v bis %v, erwartet 0 bis 76", minX, maxX)
	}
}
