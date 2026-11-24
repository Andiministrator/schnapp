// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Kritik Karte #193 — Erkundung.

func k193Rauschen(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	rand.New(rand.NewSource(1)).Read(b.Pix)
	for i := 3; i < len(b.Pix); i += 4 {
		b.Pix[i] = 255
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// K193-1: Der SVG-Export bettet je Lupe die ganze Quelle als PNG ein; das Rechenbudget (render.Pruefe) zählt aber nur
// die Zielfläche. n Lupen mit 1-Pixel-Ziel und dem ganzen Bild als Quelle: Export wächst mit n mal Bildgröße.
func TestK193ExportGroesseJeLupe(t *testing.T) {
	groesse := func(n int) (int, time.Duration) {
		d := k193Rauschen(t, 960, 540)
		for i := 0; i < n; i++ {
			fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(960, 540)),
				Ziel: document.R(document.Pt(float64(i), 0), document.Pt(float64(i)+1, 1)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}})
		}
		var buf bytes.Buffer
		t0 := time.Now()
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		return buf.Len(), time.Since(t0)
	}
	g0, _ := groesse(0)
	g20, z20 := groesse(20)
	je := (g20 - g0) / 20
	t.Logf("Export ohne Lupe %d Bytes; mit 20 Lupen %d Bytes in %v; je Lupe %d Bytes (%.2f x Grundbild)", g0, g20, z20, je, float64(je)/float64(g0))
	t.Logf("hochgerechnet auf MaxObjekte=%d: %.1f GiB, %v", document.MaxObjekte, float64(je)*float64(document.MaxObjekte)/(1<<30), z20/20*time.Duration(document.MaxObjekte))
	if je > g0/2 {
		t.Errorf("jede Lupe mit 1 Pixel Ziel kostet im Export %d Bytes (Grundbild %d) — vom Budget nicht erfasst", je, g0)
	}
}

// K193: Ziel ganz außerhalb der Ansicht, Quelle sichtbar: bettet der Export das PNG trotzdem ein?
func TestK193ExportZielVerborgen(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 32, 48)); err != nil {
		t.Fatal(err)
	}
	fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(40, 20), document.Pt(56, 32)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}})
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	t.Logf("Ziel im weggeschnittenen Teil: PNG der Quelle im Export: %v", strings.Contains(buf.String(), "image-rendering:pixelated"))
	// ADR-0033 Nachtrag 2 Punkt 3: Liegt das Ziel in keiner Kachel, entfällt sein Bild.
	if strings.Contains(buf.String(), "image-rendering:pixelated") {
		t.Error("Bild einer Lupe, deren Ziel nicht in der Ansicht liegt, steht im Export")
	}
	// Ziel im Zuschnitt, aber ganz in einem Stauch-Streifen (Mutant L27): auch dann kein Bild.
	d = k193Rauschen(t, 64, 48)
	if err := d.SetzeStreifen([]document.Streifen{{Von: 30, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(36, 20), document.Pt(52, 32)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}})
	if strings.Contains(k193Export(t, d), "image-rendering:pixelated") {
		t.Error("Bild einer Lupe, deren Ziel ganz in einem Streifen liegt, steht im Export")
	}
}

// k193Export liefert den SVG-Export von d.
func k193Export(t *testing.T, d *document.Dokument) string {
	t.Helper()
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// K193-5 (Mutant K14): Im Export liegt das Bild unter dem Rahmen des Ziels (Reihenfolge wie im Renderer).
func TestK193ExportReihenfolge(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(30, 20), document.Pt(46, 32)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 2}})
	g := k193Export(t, d)
	bild, rahmen := strings.Index(g, "image-rendering:pixelated"), strings.LastIndex(g, "<path")
	if bild < 0 || rahmen < 0 || bild > rahmen {
		t.Errorf("Bild bei %d, letzter Pfad (Rahmen des Ziels) bei %d: %s", bild, rahmen, g)
	}
}

// K193-5 (Mutant K15): Eine Quelle mit Bruchteilen bettet die ganzen Pixel darunter ein (abgerundet bzw. aufgerundet).
func TestK193ExportBruchteilQuelle(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(4.6, 4.6), document.Pt(11.4, 9.4)),
		Ziel: document.R(document.Pt(30, 20), document.Pt(43.6, 29.6)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}})
	if g := k193Export(t, d); !strings.Contains(g, `<image x="4" y="4" width="8" height="6"`) {
		t.Errorf("Bild nicht über den ganzen Pixeln 4..12 × 4..10: %s", g)
	}
}

// K193-1: Ist das Ziel kleiner als die Quelle, bettet der Export das fertige Bild in Zielauflösung ein, nicht die
// größere Quelle.
func TestK193ExportVerkleinert(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	fuege(t, d, document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(40, 30)),
		Ziel: document.R(document.Pt(50, 0), document.Pt(60, 7.5)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}})
	if g := k193Export(t, d); !strings.Contains(g, `<image x="50" y="0" width="10" height="8"`) {
		t.Errorf("Bild nicht in Zielauflösung (50..60 × 0..8): %s", g)
	}
}

// K193: Deckkraft 0 — kein Bild im Export.
func TestK193ExportDeckkraftNull(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	l := document.Lupe{Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(30, 20), document.Pt(46, 32)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}
	l.Ebene = l.Ebene.MitDeckkraft(0)
	fuege(t, d, l)
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "image-rendering:pixelated") {
		t.Error("Lupe mit Deckkraft 0 im Export")
	}
}

// K193: Version 2 ohne Lupe ist lesbar; eine Lupe im laufenden Zug zählt nicht (Version 1, keine Lupe).
func TestK193VersionZug(t *testing.T) {
	d := k193Rauschen(t, 64, 48)
	if _, err := d.BeginneNeuenZug(document.Lupe{Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(30, 20), document.Pt(46, 32)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	svg := string(svgArbeitsdatei(d, Metadaten{}))
	if !strings.Contains(svg, `s:format="1"`) || strings.Contains(svg, "lupe") {
		t.Errorf("Zug: %.300s", svg)
	}
	if _, err := leseSVG(strings.NewReader(strings.Replace(svg, `s:format="1"`, `s:format="2"`, 1))); err != nil {
		t.Errorf("Version 2 ohne Lupe: %v", err)
	}
}

// K193: Grenzwerte präparierter Lupen: Koordinaten am Rand von MaxKoordinate, Größe MaxGroesse, winzige Flächen.
func TestK193Grenzwerte(t *testing.T) {
	for name, l := range map[string]document.Lupe{
		"Rand":          {Quelle: document.R(document.Pt(-1e6, -1e6), document.Pt(-1e6+1e5, -1e6+1e5)), Ziel: document.R(document.Pt(1e6-1e5, 1e6-1e5), document.Pt(1e6, 1e6))},
		"winzigeQuelle": {Quelle: document.R(document.Pt(5, 5), document.Pt(5+1e-9, 5+1e-9)), Ziel: document.R(document.Pt(-5e4, -5e4), document.Pt(5e4, 5e4))},
		"winzigesZiel":  {Quelle: document.R(document.Pt(-5e4, -5e4), document.Pt(5e4, 5e4)), Ziel: document.R(document.Pt(5, 5), document.Pt(5+1e-9, 5+1e-9))},
		"subnormal":     {Quelle: document.R(document.Pt(0, 0), document.Pt(5e-324, 5e-324)), Ziel: document.R(document.Pt(0, 0), document.Pt(64, 48))},
	} {
		d := k193Rauschen(t, 64, 48)
		l.Strich = document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 2}
		if _, err := d.Hinzufuegen(l); err != nil {
			t.Logf("%s: abgelehnt: %v", name, err)
			continue
		}
		g, _ := schreibeLies(t, d)
		if len(g.Dokument.Objekte()) != 1 {
			t.Errorf("%s: Rundlauf verliert die Lupe", name)
		} else if got := g.Dokument.Objekte()[0].(document.Lupe); got.Quelle != l.Quelle || got.Ziel != l.Ziel {
			t.Errorf("%s: Rundlauf ändert die Lupe: %v / %v", name, got.Quelle, got.Ziel)
		}
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Errorf("%s: Export: %v", name, err)
		}
		if j := strings.LastIndex(buf.String(), `viewBox="`); j >= 0 {
			t.Logf("%s: letzte viewBox im Export: %.70s", name, buf.String()[j:])
		}
	}
}

// K193-2/K193-3 im SVG-Export (ADR-0033 Nachtrag 2 Punkte 1 und 3): Das eingebettete Bild zeigt in jedem Zielpixel
// dasselbe wie der Renderer (render.LupenInhalt), auch wenn ein Streifen und der Spalt seiner Naht in der Quelle liegen;
// einmal mit dem Ansichtsausschnitt (Vergrößerung 2), einmal mit dem Bild in Zielauflösung (Vergrößerung 0,5).
func TestK193ExportGleichRenderer(t *testing.T) {
	zahlen := regexp.MustCompile(`<svg x="([-0-9.e]+)" y="([-0-9.e]+)" width="([-0-9.e]+)" height="([-0-9.e]+)" viewBox="([-0-9.e]+) ([-0-9.e]+) ([-0-9.e]+) ([-0-9.e]+)" preserveAspectRatio="none"><image x="([-0-9]+)" y="([-0-9]+)"`)
	for _, f := range []struct {
		name           string
		quelle, ziel   document.Rechteck
		zielaufloesung bool
	}{
		{"2x über der Naht", document.R(document.Pt(120, 100), document.Pt(240, 140)), document.R(document.Pt(210, 180), document.Pt(350, 260)), false},
		{"0,5x über der Naht", document.R(document.Pt(100, 0), document.Pt(300, 100)), document.R(document.Pt(250, 150), document.Pt(325, 200)), true},
	} {
		d := k193Rauschen(t, 400, 300)
		if err := d.SetzeStreifen([]document.Streifen{{Von: 150, Bis: 200}}); err != nil {
			t.Fatal(err)
		}
		l := document.Lupe{Quelle: f.quelle, Ziel: f.ziel, Strich: document.Strich{Farbe: color.NRGBA{A: 255}}}
		fuege(t, d, l)
		svg := k193Export(t, d)
		m := zahlen.FindStringSubmatch(svg)
		if m == nil {
			t.Fatalf("%s: kein Bild der Lupe", f.name)
		}
		var v [10]float64
		for i := range v {
			v[i], _ = strconv.ParseFloat(m[i+1], 64)
		}
		box := document.R(document.Pt(v[4], v[5]), document.Pt(v[4]+v[6], v[5]+v[7]))
		if ziel := box == f.ziel; ziel != f.zielaufloesung {
			t.Errorf("%s: viewBox %v, Zielauflösung erwartet: %v", f.name, box, f.zielaufloesung)
		}
		px, ok := exportBild(t, svg)
		if !ok {
			t.Fatalf("%s: kein PNG", f.name)
		}
		soll := render.LupenInhalt(d, l)
		verglichen, leer := 0, 0
		for vy := soll.Rect.Min.Y; vy < soll.Rect.Max.Y; vy++ {
			for vx := soll.Rect.Min.X; vx < soll.Rect.Max.X; vx++ {
				z := d.ZuOriginal(document.Pt(float64(vx)+0.5, float64(vy)+0.5))
				if !(z.X >= f.ziel.Min.X && z.X < f.ziel.Max.X && z.Y >= f.ziel.Min.Y && z.Y < f.ziel.Max.Y) {
					continue
				}
				// So tastet ein SVG-Betrachter das verschachtelte <svg> ab: Ziel → viewBox → Pixel des PNG.
				bx := int(math.Floor(box.Min.X+(z.X-f.ziel.Min.X)*box.Breite()/f.ziel.Breite())) - int(v[8])
				by := int(math.Floor(box.Min.Y+(z.Y-f.ziel.Min.Y)*box.Hoehe()/f.ziel.Hoehe())) - int(v[9])
				got := color.NRGBA{}
				if image.Pt(bx, by).In(px.Bounds()) {
					got = px.NRGBAAt(bx, by)
				}
				want := soll.NRGBAAt(vx, vy)
				if got != want {
					t.Fatalf("%s: Zielpixel Ansicht (%d,%d): SVG %v, Renderer %v", f.name, vx, vy, got, want)
				}
				verglichen++
				if want.A == 0 {
					leer++
				}
			}
		}
		t.Logf("%s: %d Zielpixel verglichen, davon %d leer (Spalt); PNG %v", f.name, verglichen, leer, px.Bounds())
		if verglichen == 0 || leer == 0 {
			t.Errorf("%s: Prüfung ohne Wirkung (%d verglichen, %d leer)", f.name, verglichen, leer)
		}
	}
}
