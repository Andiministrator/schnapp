// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"math"
	"strconv"
	"testing"

	"github.com/Andiministrator/schnapp/zeichnen/svg"
)

func zeichne(t *testing.T, text string) *svg.Element {
	t.Helper()
	w, err := svg.Parse([]byte(text), "test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func bildAus(t *testing.T, text string) []uint8 {
	t.Helper()
	b, err := Zeichnen(zeichne(t, text), 100)
	if err != nil {
		t.Fatal(err)
	}
	return b.Pix
}

const kopf = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">`

// Deckung exakt: ein Rechteck mit halben Pixeln am Rand deckt genau seine Fläche.
func TestDeckungRechteck(t *testing.T) {
	pix := bildAus(t, kopf+`<rect x="10.5" y="20" width="30" height="40" fill="#000"/></svg>`)
	summe := 0.0
	for i := 3; i < len(pix); i += 4 {
		summe += float64(pix[i]) / 255
	}
	if math.Abs(summe-1200) > 1 {
		t.Fatalf("Deckung %.2f statt 1200", summe)
	}
	if pix[(30*100+10)*4+3] != 128 {
		t.Fatalf("Randpixel Alpha %d statt 128", pix[(30*100+10)*4+3])
	}
}

// Waagerechte Kante eines rect bei einem Bruchteil eines Pixels: exakt gedeckt.
func TestDeckungRechteckWaagerecht(t *testing.T) {
	pix := bildAus(t, kopf+`<rect x="10" y="20.1" width="30" height="40" fill="#000"/></svg>`)
	if a := pix[(20*100+20)*4+3]; a != 229 && a != 230 {
		t.Fatalf("Oberkante Alpha %d statt 229/230 (0,9 Deckung)", a)
	}
}

// Strichelung mit runden Enden auf einer Geraden: jeder Strich hat beide Enden.
func TestStrichelungRundGerade(t *testing.T) {
	pix := bildAus(t, kopf+`<path d="M10,50 H90" stroke="#000" stroke-width="8" stroke-linecap="round" stroke-dasharray="20 20" fill="none"/></svg>`)
	s := 0.0
	for i := 3; i < len(pix); i += 4 {
		s += float64(pix[i]) / 255
	}
	// Striche 10-30, 50-70, 90 (Länge 0 am Ende: Punkt); je 20 x 8 plus Kreis r 4, der letzte nur Kreis
	soll := 2*(20*8+math.Pi*16) + math.Pi*16
	if math.Abs(s-soll) > 3 {
		t.Fatalf("Deckung %.1f statt %.1f", s, soll)
	}
}

// Gegenproben des Maßstabs: halbes Pixel verschoben ist gleichwertig, zwei Pixel oder eine andere Farbe nicht.
func TestVergleichGegenproben(t *testing.T) {
	bild := func(x float64, farbe string) []byte {
		return []byte(kopf + `<rect width="100" height="100" fill="#ccc7aa"/><circle cx="` +
			strconv.FormatFloat(x, 'g', -1, 64) + `" cy="50" r="30" fill="` + farbe + `"/></svg>`)
	}
	lade := func(b []byte) *svg.Element {
		w, err := svg.Parse(b, "test")
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	ref, _ := Zeichnen(lade(bild(50, "#4a6f9c")), 100)
	for _, f := range []struct {
		x      float64
		farbe  string
		gleich bool
	}{{50.5, "#4a6f9c", true}, {52, "#4a6f9c", false}, {50, "#496f9c", true},
		{50, "#5a378a", false}} {
		b, _ := Zeichnen(lade(bild(f.x, f.farbe)), 100)
		v, err := Vergleichen(b, ref)
		if err != nil {
			t.Fatal(err)
		}
		if (v.Verstoesse == 0) != f.gleich {
			t.Errorf("x=%v Farbe %s: Verstöße %d, erwartet gleichwertig=%v", f.x, f.farbe, v.Verstoesse, f.gleich)
		}
	}
}

// Gegenproben an dünnen Strichen: sie bestehen nur aus Kantenpixeln; fehlt einer, ist er
// andersfarbig oder zu breit, muss der Vergleich trotzdem anschlagen.
func TestVergleichDuenneStriche(t *testing.T) {
	bild := func(inhalt string) []byte {
		return []byte(kopf + `<rect width="100" height="100" fill="#ccc7aa"/>` + inhalt + `</svg>`)
	}
	zeichneB := func(b []byte) *svg.Element {
		w, err := svg.Parse(b, "test")
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	linie := func(farbe, breite string) string {
		return `<path d="M10,30.5 L90,60.5" stroke="` + farbe + `" stroke-width="` + breite + `" fill="none"/>`
	}
	kreis := `<circle cx="50" cy="50" r="30" fill="none" stroke="#fff" stroke-width="1.5"/>`
	for _, f := range []struct {
		name, a, b string
	}{
		{"1-px-Linie andersfarbig", linie("#000", "1"), linie("#c04a2b", "1")},
		{"Umriss fehlt", kreis, ""},
		{"Strich 3 statt 1", linie("#000", "3"), linie("#000", "1")},
	} {
		a, _ := Zeichnen(zeichneB(bild(f.a)), 100)
		b, _ := Zeichnen(zeichneB(bild(f.b)), 100)
		v, err := Vergleichen(a, b)
		if err != nil {
			t.Fatal(err)
		}
		if v.Verstoesse == 0 {
			t.Errorf("%s: kein Verstoß (Mittel %d)", f.name, v.Mittel)
		}
	}
}

// Strich-Randfälle: nichts darf still verschwinden.
func TestStrichRandfaelle(t *testing.T) {
	deckung := func(inhalt string) float64 {
		pix := bildAus(t, kopf+inhalt+`</svg>`)
		s := 0.0
		for i := 3; i < len(pix); i += 4 {
			s += float64(pix[i]) / 255
		}
		return s
	}
	// Strichgrenze genau auf Stützpunkten: 5 Striche zu 10 x 2
	if d := deckung(`<path d="M0,50 H100" stroke="#000" stroke-width="2" stroke-dasharray="10 10" fill="none"/>`); math.Abs(d-100) > 1 {
		t.Errorf("Strichelung auf Stützpunkten: Deckung %.2f statt 100", d)
	}
	// Punktlinie 0 16 mit runden Enden: 7 Punkte (0, 16, ... 96) mit Radius 2
	if d := deckung(`<path d="M0,50 H100" stroke="#000" stroke-width="4" stroke-dasharray="0 16" stroke-linecap="round" fill="none"/>`); d < 7*math.Pi*4*0.8 {
		t.Errorf("Punktlinie: Deckung %.2f", d)
	}
	// Teilpfad der Länge 0 mit runden Enden: ein Punkt
	if d := deckung(`<path d="M50,50 L50,50" stroke="#000" stroke-width="10" stroke-linecap="round" fill="none"/>`); math.Abs(d-math.Pi*25) > 1.5 {
		t.Errorf("Länge 0: Deckung %.2f statt %.2f", d, math.Pi*25)
	}
	// U-Wende mit runder Ecke: in beiden Laufrichtungen dieselbe Deckung
	a := deckung(`<path d="M20,50 L80,50 L20,50" stroke="#000" stroke-width="10" stroke-linejoin="round" fill="none"/>`)
	b := deckung(`<path d="M80,50 L20,50 L80,50" stroke="#000" stroke-width="10" stroke-linejoin="round" fill="none"/>`)
	if math.Abs(a-b) > 0.5 || math.Abs(a-(600+math.Pi*25/2)) > 1.5 {
		t.Errorf("U-Wende: %.2f und %.2f, erwartet %.2f", a, b, 600+math.Pi*25/2)
	}
	// Muster mit Summe 0 wie none: geschlossenes Quadrat ganz
	if d := deckung(`<rect x="20" y="20" width="60" height="60" stroke="#000" stroke-width="2" stroke-dasharray="0 0" fill="none"/>`); math.Abs(d-(4*60*2)) > 2 {
		t.Errorf("Summe 0: Deckung %.2f statt 480", d)
	}
}

// Strich: waagerechte Linie mit butt-Enden deckt Länge mal Breite, mit round-Enden zusätzlich einen Kreis.
func TestStrichEnden(t *testing.T) {
	deckung := func(ende string) float64 {
		pix := bildAus(t, kopf+`<path d="M20,50 H80" stroke="#000" stroke-width="10" fill="none" stroke-linecap="`+ende+`"/></svg>`)
		s := 0.0
		for i := 3; i < len(pix); i += 4 {
			s += float64(pix[i]) / 255
		}
		return s
	}
	if d := deckung("butt"); math.Abs(d-600) > 1 {
		t.Errorf("butt: %.2f statt 600", d)
	}
	if d := deckung("square"); math.Abs(d-700) > 1 {
		t.Errorf("square: %.2f statt 700", d)
	}
	if d := deckung("round"); math.Abs(d-(600+math.Pi*25)) > 1.5 {
		t.Errorf("round: %.2f statt %.2f", d, 600+math.Pi*25)
	}
}

// Gehrung: ein rechter Winkel mit miter füllt die Ecke ganz, mit bevel fehlt ein halbes Quadrat.
func TestEcken(t *testing.T) {
	deckung := func(ecke string) float64 {
		pix := bildAus(t, kopf+`<path d="M20,80 V20 H80" stroke="#000" stroke-width="10" fill="none" stroke-linejoin="`+ecke+`"/></svg>`)
		s := 0.0
		for i := 3; i < len(pix); i += 4 {
			s += float64(pix[i]) / 255
		}
		return s
	}
	miter, bevel, rund := deckung("miter"), deckung("bevel"), deckung("round")
	if math.Abs(miter-bevel-12.5) > 1 {
		t.Errorf("miter %.2f, bevel %.2f: Unterschied %.2f statt 12.5", miter, bevel, miter-bevel)
	}
	if !(rund > bevel && rund < miter) {
		t.Errorf("round %.2f liegt nicht zwischen bevel %.2f und miter %.2f", rund, bevel, miter)
	}
}

// evenodd: ein Loch aus gleich herum laufenden Quadraten bleibt nur unter evenodd leer.
func TestFuellregel(t *testing.T) {
	d := `M10,10 H90 V90 H10 Z M30,30 H70 V70 H30 Z`
	mitte := func(regel string) uint8 {
		return bildAus(t, kopf+`<path d="`+d+`" fill="#000" fill-rule="`+regel+`"/></svg>`)[(50*100+50)*4+3]
	}
	if mitte("nonzero") != 255 || mitte("evenodd") != 0 {
		t.Errorf("nonzero %d, evenodd %d", mitte("nonzero"), mitte("evenodd"))
	}
}

// Maske nach Luminanz und clipPath mit eigenem transform (wie Illustrator).
func TestMaskeUndClip(t *testing.T) {
	pix := bildAus(t, kopf+`<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100">`+
		`<rect width="50" height="100" fill="#fff"/></mask><g mask="url(#m)"><rect width="100" height="100" fill="#000"/></g></svg>`)
	if pix[(50*100+25)*4+3] != 255 || pix[(50*100+75)*4+3] != 0 {
		t.Errorf("Maske: links %d, rechts %d", pix[(50*100+25)*4+3], pix[(50*100+75)*4+3])
	}
	pix = bildAus(t, kopf+`<defs><style>.c{clip-path:url(#k);}</style><clipPath id="k" transform="translate(50 0)">`+
		`<rect width="50" height="100"/></clipPath></defs><g class="c"><rect width="100" height="100" fill="#000"/></g></svg>`)
	if pix[(50*100+25)*4+3] != 0 || pix[(50*100+75)*4+3] != 255 {
		t.Errorf("clipPath: links %d, rechts %d", pix[(50*100+25)*4+3], pix[(50*100+75)*4+3])
	}
}

// Nicht Unterstütztes ist ein Fehler, kein stilles Übergehen.
func TestFehler(t *testing.T) {
	for _, f := range []string{
		`<text x="1" y="1">a</text>`,
		`<linearGradient id="v"/>`,
		`<rect width="1" height="1" style="fill:red"/>`,
		`<rect width="1" height="1" fill="url(#v)"/>`,
		`<rect width="1" height="1" fill="red"/>`,
		`<rect width="1" height="1" filter="url(#f)"/>`,
		`<image href="a.png"/>`,
		`<use href="#a"/>`,
		`<g transform="perspective(3)"/>`,
		`<style>rect{fill:#000}</style>`,
		`<style>.a{font-size:3px}</style>`,
		`<mask id="m"><rect width="1" height="1"/></mask><g mask="url(#m)"/>`,
		`<g clip-path="url(#fehlt)"><rect width="1" height="1"/></g>`,
		`<rect width="1" height="1" stroke-linejoin="arcs" stroke="#000"/>`,
		`<svg x="10" viewBox="0 0 1 1"><rect width="1" height="1"/></svg>`,
		`<foo:rect xmlns:foo="urn:x" width="1" height="1"/>`,
		`<path d="M1,1 L1,1" stroke="#000" stroke-linecap="square"/>`,
		`<rect width="1" height="1" opacity="NaN"/>`,
		`<rect width="NaN" height="1"/>`,
	} {
		w, err := svg.Parse([]byte(kopf+f+`</svg>`), "test")
		if err != nil {
			continue
		}
		if _, err := Zeichnen(w, 10); err == nil {
			t.Errorf("kein Fehler bei %s", f)
		}
	}
}

// mask erbt vom Ort ihrer Definition.
func TestMaskeErbt(t *testing.T) {
	pix := bildAus(t, kopf+`<g fill="#fff"><mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100">`+
		`<rect width="50" height="100"/></mask></g><g mask="url(#m)"><rect width="100" height="100" fill="#000"/></g></svg>`)
	if pix[(50*100+25)*4+3] != 255 {
		t.Errorf("Maske ohne geerbte weiße Füllung: Alpha %d", pix[(50*100+25)*4+3])
	}
}

// Höhe wie rsvg-convert: aufgerundet, Inhalt oben bündig.
func TestGroesse(t *testing.T) {
	w, _ := svg.Parse([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2000 1350"/>`), "t")
	b, h, _, err := Groesse(w, 1350)
	if err != nil || b != 1350 || h != 912 {
		t.Fatalf("%d x %d, %v", b, h, err)
	}
}
