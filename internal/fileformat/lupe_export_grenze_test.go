// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

// Sitzungs-QA vom 24.11.2026 (Kritiker Fable), QS-2: Der SVG-Export einer präparierten Datei unter dem Rechenbudget
// baute mit tausenden Lupen gleicher Quelle Gigabyte im Arbeitsspeicher (Rest von K193-1, F-609). ADR-0033 Nachtrag 5:
// gleiche Bilder nur einmal, Summe der eingebetteten Lupenpixel höchstens LupenExportFaktor × Ansichtsfläche.

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// lupenRauschen ist ein Dokument w×h mit deckendem Rauschen (kaum komprimierbar, wie im Beleg des Kritikers).
func lupenRauschen(t testing.TB, w, h int) *document.Dokument {
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

// lupenPNGs zählt die eingebetteten Bilder der Lupen (ohne die Kacheln) und die Wiederholungen per <use>.
func lupenPNGs(svg string) (bilder, wiederholt int) {
	return strings.Count(svg, `style="image-rendering:pixelated" href="data:image/png`),
		strings.Count(svg, `<use href="#lupenbild-`)
}

// TestQSExportVieleGleicheLupen: Der Beleg des Kritikers — 4822 Lupen mit Quelle = Ziel = ganzes Bild (960×540), so
// viele, wie render.Pruefe durchlässt. Vorher 9,3 GiB und rund 5 Minuten; jetzt steht das Bild einmal in der Datei.
func TestQSExportVieleGleicheLupen(t *testing.T) {
	const n = 4822
	voll := document.R(document.Pt(0, 0), document.Pt(960, 540))
	var grund bytes.Buffer
	if err := ExportiereSVG(&grund, lupenRauschen(t, 960, 540)); err != nil {
		t.Fatal(err)
	}
	d := lupenRauschen(t, 960, 540)
	// Vorprobe mit 20 Lupen: Ohne Zusammenlegen bräche der volle Beleg Gigabyte im Speicher (auch im Mutantenlauf).
	for i := 0; i < 20; i++ {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: voll, Ziel: voll}); err != nil {
			t.Fatal(err)
		}
	}
	var probe bytes.Buffer
	if err := ExportiereSVG(&probe, d); err != nil {
		t.Fatal(err)
	}
	if b, _ := lupenPNGs(probe.String()); b != 1 {
		t.Fatalf("20 gleiche Lupen: %d eingebettete Bilder statt 1; den vollen Beleg (%d Lupen) nicht erst bauen", b, n)
	}
	for i := 20; i < n; i++ {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: voll, Ziel: voll}); err != nil {
			t.Fatal(err)
		}
	}
	if err := render.Pruefe(d); err != nil {
		t.Fatalf("Vorbedingung: der Beleg liegt unter dem Rechenbudget: %v", err)
	}
	var buf bytes.Buffer
	t0 := time.Now()
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	t.Logf("ohne Lupe %d Bytes; mit %d gleichen Lupen %d Bytes in %v", grund.Len(), n, buf.Len(), time.Since(t0))
	if buf.Len() > 3*grund.Len() {
		t.Errorf("%d gleiche Lupen: Export %d Bytes, mehr als das Dreifache des Exports ohne Lupe (%d)", n, buf.Len(), grund.Len())
	}
	if b, u := lupenPNGs(buf.String()); b != 1 || u != n-1 {
		t.Errorf("%d gleiche Lupen: %d eingebettete Bilder, %d Wiederholungen; erwartet 1 und %d", n, b, u, n-1)
	}
}

// TestQSExportVerschiedeneLupenGrenze: Lupen mit verschiedenen, fast bildgroßen Quellen (je ein Pixel versetzt) lassen
// sich nicht zusammenlegen. Bis LupenExportFaktor solcher Lupen gehen durch; eine mehr lehnt der Export vor dem Kodieren
// ab (nichts geschrieben), ExportiereSVGOhneGrenze schreibt sie (--ignore-limit).
func TestQSExportVerschiedeneLupenGrenze(t *testing.T) {
	bau := func(n int) *document.Dokument {
		d := lupenRauschen(t, 960, 540)
		for i := 0; i < n; i++ {
			q := document.R(document.Pt(float64(i), 0), document.Pt(960, 540))
			if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: q}); err != nil {
				t.Fatal(err)
			}
		}
		return d
	}
	var buf bytes.Buffer
	d := bau(LupenExportFaktor)
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatalf("%d Lupen: %v", LupenExportFaktor, err)
	}
	if b, u := lupenPNGs(buf.String()); b != LupenExportFaktor || u != 0 {
		t.Errorf("%d verschiedene Lupen: %d Bilder, %d Wiederholungen", LupenExportFaktor, b, u)
	}
	d = bau(LupenExportFaktor + 1)
	if err := render.Pruefe(d); err != nil {
		t.Fatalf("Vorbedingung: unter dem Rechenbudget: %v", err)
	}
	buf.Reset()
	t0 := time.Now()
	err := ExportiereSVG(&buf, d)
	dauer := time.Since(t0)
	if !errors.Is(err, ErrLupenZuGross) || !strings.HasPrefix(err.Error(), "fileformat: ") {
		t.Fatalf("%d Lupen: %v, erwartet ErrLupenZuGross mit Präfix", LupenExportFaktor+1, err)
	}
	if buf.Len() != 0 {
		t.Errorf("abgelehnt, aber %d Bytes geschrieben", buf.Len())
	}
	if err := PruefeSVGExport(d); !errors.Is(err, ErrLupenZuGross) {
		t.Errorf("PruefeSVGExport: %v", err)
	}
	t.Logf("Ablehnung nach %v (ohne Kodieren)", dauer)
	buf.Reset()
	if err := ExportiereSVGOhneGrenze(&buf, d); err != nil {
		t.Fatal(err)
	}
	if b, _ := lupenPNGs(buf.String()); b != LupenExportFaktor+1 {
		t.Errorf("ohne Grenze: %d Bilder", b)
	}
}

// TestQSExportAlltagMitLupen: Ein Alltagsdokument (1920×1080, sechs Lupen mit Quellen um 300×200, dazu eine
// verkleinernde) bleibt weit unter der Grenze und wird geschrieben.
func TestQSExportAlltagMitLupen(t *testing.T) {
	d := lupenRauschen(t, 1920, 1080)
	for i := 0; i < 6; i++ {
		x := float64(50 + 300*i)
		q := document.R(document.Pt(x, 40), document.Pt(x+280, 240))
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: document.R(document.Pt(x, 400), document.Pt(x+560, 800)),
			Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 3}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Hinzufuegen(document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(1920, 1080)),
		Ziel: document.R(document.Pt(1500, 850), document.Pt(1692, 958))}); err != nil {
		t.Fatal(err)
	}
	g := d.Groesse()
	n := LupenExportPixel(d)
	t.Logf("eingebettete Lupenpixel %d, Grenze %d", n, LupenExportFaktor*g.X*g.Y)
	if n > int64(g.X*g.Y) {
		t.Errorf("Alltagsdokument bettet %d Pixel ein, mehr als die Ansicht (%d)", n, g.X*g.Y)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if b, _ := lupenPNGs(buf.String()); b != 7 {
		t.Errorf("Alltagsdokument: %d Bilder statt 7", b)
	}
}

// TestQSExportZaehltNurExportiertes: Die Grenze zählt nur, was der Export schreibt — ausgeblendete Lupen, Lupen mit
// Deckkraft 0 % und solche, deren Ziel in keiner Kachel liegt, betten nichts ein.
func TestQSExportZaehltNurExportiertes(t *testing.T) {
	d := lupenRauschen(t, 400, 300)
	for i := 0; i < 3*LupenExportFaktor; i++ {
		q := document.R(document.Pt(float64(i), 0), document.Pt(400, 300))
		l := document.Lupe{Quelle: q, Ziel: q}
		switch i % 3 {
		case 0:
			l.Ebene.Ausgeblendet = true
		case 1:
			l.Ebene = l.Ebene.MitDeckkraft(0)
		default:
			l.Ziel = document.R(document.Pt(500, 400), document.Pt(600, 500)) // außerhalb der Ansicht
		}
		if _, err := d.Hinzufuegen(l); err != nil {
			t.Fatal(err)
		}
	}
	if n := LupenExportPixel(d); n != 0 {
		t.Errorf("unsichtbare Lupen zählen %d Pixel", n)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
}

// TestQSExportWiederholungGleichesBild: Die Wiederholung per <use> zeigt dasselbe wie ein eigenes Bild — sie liegt in
// einem <svg> mit eigener Lage und viewBox (zwei gleiche Quellen, verschiedene Ziele), das <use> verweist auf das erste
// Bild.
func TestQSExportWiederholungGleichesBild(t *testing.T) {
	d := lupenRauschen(t, 200, 100)
	q := document.R(document.Pt(10, 10), document.Pt(30, 20))
	for _, z := range []document.Rechteck{document.R(document.Pt(50, 10), document.Pt(90, 30)), document.R(document.Pt(100, 50), document.Pt(140, 70))} {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: z}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	for _, teil := range []string{
		`<svg x="50" y="10" width="40" height="20" viewBox="10 10 20 10" preserveAspectRatio="none"><image x="10" y="10" width="20" height="10"`,
		`id="lupenbild-1"/></svg>`,
		`<svg x="100" y="50" width="40" height="20" viewBox="10 10 20 10" preserveAspectRatio="none"><use href="#lupenbild-1"/></svg>`,
	} {
		if !strings.Contains(svg, teil) {
			t.Errorf("Export ohne %q", teil)
		}
	}
}

// TestQSExportZielbildSchluessel: Zwei verkleinernde Lupen (Zielbild) mit gleichem Ziel, aber verschiedenen Quellen
// tragen verschiedene Pixel und dürfen nicht zusammengelegt werden.
func TestQSExportZielbildSchluessel(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			b.SetNRGBA(x, y, qaFarbe(x, y))
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	z := document.R(document.Pt(150, 80), document.Pt(160, 90))
	for _, q := range []document.Rechteck{document.R(document.Pt(0, 0), document.Pt(100, 100)), document.R(document.Pt(100, 0), document.Pt(200, 100))} {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: z}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if b, u := lupenPNGs(buf.String()); b != 2 || u != 0 {
		t.Errorf("zwei Zielbilder verschiedener Quellen: %d Bilder, %d Wiederholungen; erwartet 2 und 0", b, u)
	}
}

func qaFarbe(x, y int) color.NRGBA {
	return color.NRGBA{uint8(x), uint8(y), uint8(0x80 | (x>>8)&3 | ((y>>8)&3)<<2), 255}
}
func qaOrt(c color.NRGBA) (int, int, bool) {
	if c.A != 255 || c.B&0x80 == 0 || c.B&0x70 != 0 {
		return 0, 0, false
	}
	return int(c.R) | int(c.B&3)<<8, int(c.G) | int(c.B>>2&3)<<8, true
}

// TestQSExportPNGNurAnsicht (Sitzungs-QA QS-V3): Die im SVG-Export eingebetteten PNGs der Lupen tragen nur Pixel, die
// die Ansicht zeigt und die unter der Quelle liegen (Zuschnitt, Streifen je Achse, zufällige Lagen mit Bruchteilen,
// Vergrößerung größer und kleiner 1). Die Kacheln des Exports (ganzes Bild der Ansicht) werden übersprungen: geprüft
// werden nur Bilder mit image-rendering:pixelated davor.
func TestQSExportPNGNurAnsicht(t *testing.T) {
	lupenPNG := regexp.MustCompile(`style="image-rendering:pixelated" href="data:image/png;base64,([A-Za-z0-9+/=]+)"`)
	rnd := rand.New(rand.NewSource(424242))
	faelle, bilder, pixel := 0, 0, 0
	runden := 300
	if testing.Short() {
		runden = 60
	}
	for it := 0; it < runden; it++ {
		w, h := 260+rnd.Intn(300), 260+rnd.Intn(200)
		b := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				b.SetNRGBA(x, y, qaFarbe(x, y))
			}
		}
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		crop := image.Rect(rnd.Intn(30), rnd.Intn(30), w-rnd.Intn(30), h-rnd.Intn(30))
		if d.SetzeZuschnitt(crop) != nil {
			continue
		}
		var st []document.Streifen
		for _, zeilen := range []bool{false, true} {
			lo, hi := crop.Min.X, crop.Max.X
			if zeilen {
				lo, hi = crop.Min.Y, crop.Max.Y
			}
			pos := lo + 40 + rnd.Intn(40)
			for n := rnd.Intn(3); n > 0 && pos+60 < hi; n-- {
				br := 1 + rnd.Intn(40)
				st = append(st, document.Streifen{Zeilen: zeilen, Von: pos, Bis: pos + br})
				pos += br + rnd.Intn(70)
			}
		}
		if len(st) > 0 && d.SetzeStreifen(st) != nil {
			continue
		}
		rect := func(maxw, maxh float64) document.Rechteck {
			x, y := -20+rnd.Float64()*float64(w), -20+rnd.Float64()*float64(h)
			return document.R(document.Pt(x, y), document.Pt(x+0.3+rnd.Float64()*maxw, y+0.3+rnd.Float64()*maxh))
		}
		l := document.Lupe{Quelle: rect(200, 150), Ziel: rect(200, 150)}
		if _, err := d.Hinzufuegen(l); err != nil {
			continue
		}
		faelle++
		var buf bytes.Buffer
		if err := ExportiereSVG(&buf, d); err != nil {
			t.Fatal(err)
		}
		sichtbar := func(ox, oy int) bool {
			if !image.Pt(ox, oy).In(crop) {
				return false
			}
			for _, s := range st {
				v := ox
				if s.Zeilen {
					v = oy
				}
				if v >= s.Von && v < s.Bis {
					return false
				}
			}
			return true
		}
		for _, m := range lupenPNG.FindAllSubmatch(buf.Bytes(), -1) {
			roh, err := base64.StdEncoding.DecodeString(string(m[1]))
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(roh))
			if err != nil {
				t.Fatal(err)
			}
			bilder++
			r := img.Bounds()
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
					if c.A == 0 {
						continue
					}
					pixel++
					ox, oy, ok := qaOrt(c)
					if !ok {
						t.Fatalf("Fall %d: PNG-Pixel ohne Originalfarbe %v", it, c)
					}
					if !sichtbar(ox, oy) {
						t.Fatalf("Fall %d: LECK — das PNG der Lupe trägt Original %d,%d, das die Ansicht nicht zeigt (Zuschnitt %v, Streifen %v, Lupe %v)", it, ox, oy, crop, st, l)
					}
					if float64(ox+1) <= l.Quelle.Min.X || float64(ox) >= l.Quelle.Max.X || float64(oy+1) <= l.Quelle.Min.Y || float64(oy) >= l.Quelle.Max.Y {
						t.Fatalf("Fall %d: das PNG der Lupe trägt Original %d,%d außerhalb der Quelle %v", it, ox, oy, l.Quelle)
					}
				}
			}
		}
	}
	t.Logf("%d Fälle, %d eingebettete Lupenbilder, %d Pixel geprüft", faelle, bilder, pixel)
}

// BenchmarkLupenExportGrenze misst den SVG-Export einer Rausch-Ansicht ohne Lupe und mit LupenExportFaktor Lupen
// verschiedener, fast bildgroßer Quellen (das Teuerste unter der Grenze; ADR-0033 Nachtrag 5): 960×540 und 3840×2160.
// Läuft nur mit -bench.
func BenchmarkLupenExportGrenze(b *testing.B) {
	for _, g := range []image.Point{{960, 540}, {3840, 2160}} {
		for _, n := range []int{0, LupenExportFaktor} {
			d := lupenRauschen(b, g.X, g.Y)
			for i := 0; i < n; i++ {
				q := document.R(document.Pt(float64(i), 0), document.Pt(float64(g.X), float64(g.Y)))
				if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: q}); err != nil {
					b.Fatal(err)
				}
			}
			b.Run(fmt.Sprintf("%dx%d/%d", g.X, g.Y, n), func(b *testing.B) {
				for range b.N {
					var buf bytes.Buffer
					if err := ExportiereSVG(&buf, d); err != nil {
						b.Fatal(err)
					}
					b.ReportMetric(float64(buf.Len())/(1<<20), "MiB")
				}
			})
		}
	}
}
