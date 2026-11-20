// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
)

func hilfWeiss(b, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	for i := range m.Pix {
		m.Pix[i] = 255
	}
	return m
}

// hilfTinte rastert o allein auf Weiß und liefert je Pixel die Abweichung von Weiß (0 = keine Tinte).
func hilfTinte(t *testing.T, o document.Objekt, b, h int) *image.NRGBA {
	t.Helper()
	d, err := document.Neu(hilfWeiss(b, h))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(o); err != nil {
		t.Fatalf("%#v: %v", o, err)
	}
	bild, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	return bild
}

type verstoss struct {
	o       document.Objekt
	pixel   int     // Zahl der Tintenpixel außerhalb der Reichweite
	weit    float64 // größter Abstand eines solchen Pixels zur Reichweite (Pixelkante innen)
	staerke int     // größte Abweichung von Weiß (0..255) unter diesen Pixeln
}

// hilfPruefe zählt Tintenpixel, die die Reichweite nicht mit positiver Fläche schneiden (genau dann ließe ein
// Zuschnitt auf diese Pixel das Objekt weg, obwohl es dort zu sehen wäre).
func hilfPruefe(t *testing.T, o document.Objekt, b, h int) verstoss {
	bild := hilfTinte(t, o, b, h)
	rw := document.Reichweite(o)
	v := verstoss{o: o}
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			c := bild.NRGBAAt(x, y)
			ab := 255 - int(min(c.R, c.G, c.B))
			if ab == 0 {
				continue
			}
			fx, fy := float64(x), float64(y)
			if fx < rw.Max.X && fx+1 > rw.Min.X && fy < rw.Max.Y && fy+1 > rw.Min.Y {
				continue
			}
			v.pixel++
			dx := math.Max(rw.Min.X-(fx+1), fx-rw.Max.X)
			dy := math.Max(rw.Min.Y-(fy+1), fy-rw.Max.Y)
			v.weit = math.Max(v.weit, math.Max(dx, dy))
			v.staerke = max(v.staerke, ab)
		}
	}
	return v
}

var hilfSchwarz = color.NRGBA{0, 0, 0, 255}

// Zufallssuche: Gibt es Objekte, deren gerasterte Tinte über document.Reichweite hinausragt?
func TestReichweiteIstObermassZufall(t *testing.T) {
	const b, h = 420, 320
	rnd := rand.New(rand.NewSource(111))
	pt := func(lo, hi float64) document.Punkt {
		return document.Pt(lo+rnd.Float64()*(hi-lo), lo*0.7+rnd.Float64()*(hi-lo)*0.7)
	}
	breiten := []float64{0.5, 1, 2, 3, 4, 6, 8, 12, 20, 30}
	arten := map[string][]verstoss{}
	merke := func(art string, o document.Objekt) {
		if v := hilfPruefe(t, o, b, h); v.pixel > 0 {
			arten[art] = append(arten[art], v)
		}
	}
	zahl := map[string]int{}
	for i := 0; i < 600; i++ {
		st := document.Strich{Farbe: hilfSchwarz, Breite: breiten[rnd.Intn(len(breiten))]}
		// Pfeile: lang, kurz, stark gebogen, K nahe B
		a, e := pt(120, 300), pt(120, 300)
		switch i % 4 {
		case 0:
			merke("pfeil-gerade", document.GeraderPfeil(a, e, st))
			zahl["pfeil-gerade"]++
		case 1:
			w := rnd.Float64() * 2 * math.Pi
			l := rnd.Float64() * 6 * math.Max(8, 4*st.Breite) / 4
			e = a.Plus(document.Pt(math.Cos(w)*l, math.Sin(w)*l))
			merke("pfeil-kurz", document.GeraderPfeil(a, e, st))
			zahl["pfeil-kurz"]++
		case 2:
			merke("pfeil-gebogen", document.Pfeil{A: a, B: e, K: pt(120, 300), Strich: st})
			zahl["pfeil-gebogen"]++
		case 3:
			w := rnd.Float64() * 2 * math.Pi
			l := rnd.Float64() * 10
			merke("pfeil-K-nahe-B", document.Pfeil{A: a, B: e, K: e.Plus(document.Pt(math.Cos(w)*l, math.Sin(w)*l)), Strich: st})
			zahl["pfeil-K-nahe-B"]++
		}
	}
	for i := 0; i < 100; i++ {
		st := document.Strich{Farbe: hilfSchwarz, Breite: breiten[rnd.Intn(len(breiten))]}
		p := pt(100, 250)
		k := document.Kasten{Bereich: document.Rechteck{Min: p, Max: p.Plus(document.Pt(rnd.Float64()*120, rnd.Float64()*80))},
			Strich: st, Eckradius: rnd.Float64() * 40}
		if i%2 == 0 {
			k.Fuellung = color.NRGBA{200, 0, 0, 255}
		}
		merke("kasten", k)
		zahl["kasten"]++
	}
	texte := []string{"Hallo Welt", "jgyQ_Åß|", "W\nfj\n\tTab", "AVAVAV To", "  x  ", "ÄÖÜ\n\n.", "₧₧", "MMMMMMMMMMMMMMMMMMMM"}
	for i := 0; i < 96; i++ {
		tx := document.Text{Ort: pt(60, 200), Inhalt: texte[i%len(texte)], Schriftgrad: 4 + rnd.Float64()*36, Farbe: hilfSchwarz}
		if i%3 == 1 {
			tx.Hintergrund = color.NRGBA{255, 220, 0, 255}
		}
		if i%3 == 2 {
			tx.Hintergrund = color.NRGBA{255, 220, 0, 255}
			tx.BreiteGrenze = 5 + rnd.Float64()*40
		}
		merke("text", tx)
		zahl["text"]++
	}
	for i := 0; i < 100; i++ {
		werte := []int{0, 1, 8, 10, 99, 100, 888, 12345, 888888, 1000000}
		merke("nummer", document.Nummer{Mitte: pt(150, 280), Wert: werte[i%len(werte)], Radius: 2 + rnd.Float64()*30, Farbe: hilfSchwarz})
		zahl["nummer"]++
	}
	for _, art := range []string{"pfeil-gerade", "pfeil-kurz", "pfeil-gebogen", "pfeil-K-nahe-B", "kasten", "text", "nummer"} {
		vs := arten[art]
		if len(vs) == 0 {
			t.Logf("%-16s %4d geprüft, kein Tintenpixel außerhalb der Reichweite", art, zahl[art])
			continue
		}
		schlimm := vs[0]
		for _, v := range vs {
			if v.weit > schlimm.weit {
				schlimm = v
			}
		}
		t.Errorf("%-16s %4d geprüft, %d mit Tinte außerhalb der Reichweite; schlimmster: %d Pixel, bis %.2f px außerhalb, Stärke %d/255: %+v",
			art, zahl[art], len(vs), schlimm.pixel, schlimm.weit, schlimm.staerke, schlimm.o)
	}
}

// Gezieltes Gegenbeispiel: kurzer schräger Pfeil. Die Ecke der Spitze liegt bei B − r·g ± n·g/2, also bis g·√5/2
// (1,118·g) von B entfernt entlang einer Achse; Reichweite nimmt nur ±g.
func TestPfeilspitzeEckeNichtAbgeschnitten(t *testing.T) {
	for _, breite := range []float64{3, 4, 10} {
		t.Run(fmt.Sprintf("Breite%v", breite), func(t *testing.T) {
			g := math.Max(8, 4*breite)
			w := math.Atan(0.5)
			r := document.Pt(math.Cos(w), math.Sin(w))
			bp := document.Pt(150, 100)
			a := bp.Minus(r.Mal(5)) // 5 px lang
			pf := document.GeraderPfeil(a, bp, document.Strich{Farbe: hilfSchwarz, Breite: breite})
			rw := document.Reichweite(pf)
			ecke := bp.X - g*math.Sqrt(5)/2
			t.Logf("g=%v: Reichweite Min.X=%.2f, Ecke der Spitze bei x=%.2f (%.2f px außerhalb)", g, rw.Min.X, ecke, rw.Min.X-ecke)

			// Ganzes Bild: Tinte links von der Reichweite?
			voll := hilfTinte(t, pf, 300, 200)
			kante := int(math.Floor(rw.Min.X)) // Zuschnitt [0, kante): Reichweite schneidet ihn nicht
			tinte := 0
			for y := 0; y < 200; y++ {
				for x := 0; x < kante; x++ {
					if voll.NRGBAAt(x, y).R != 255 {
						tinte++
					}
				}
			}
			t.Logf("ohne Zuschnitt: %d Tintenpixel in x < %d", tinte, kante)

			// Zuschnitt auf x < kante: Objekt fällt aus PNG und SVG, obwohl dort Tinte war.
			d, err := document.Neu(hilfWeiss(300, 200))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.Hinzufuegen(pf); err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeZuschnitt(image.Rect(0, 0, kante, 200)); err != nil {
				t.Fatal(err)
			}
			bild, err := Bild(d)
			if err != nil {
				t.Fatal(err)
			}
			nach := 0
			for i := 0; i < len(bild.Pix); i += 4 {
				if bild.Pix[i] != 255 {
					nach++
				}
			}
			var buf bytes.Buffer
			if err := fileformat.ExportiereSVG(&buf, d); err != nil {
				t.Fatal(err)
			}
			imSVG := strings.Contains(buf.String(), `stroke-linecap`)
			t.Logf("mit Zuschnitt x<%d: ObjektSichtbar=%v, Tintenpixel im PNG=%d, Pfeil im SVG=%v", kante, d.ObjektSichtbar(pf), nach, imSVG)
			if tinte > 0 && (nach == 0 || !imSVG) {
				t.Errorf("sichtbares Stück fehlt: %d Tintenpixel im Ausschnitt, nach dem Zuschnitt PNG %d Pixel, im SVG %v", tinte, nach, imSVG)
			}
		})
	}
}
