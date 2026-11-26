// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 25: Gemusterte Striche bestehen aus überlappungsfreien Teilen. Halbdurchsichtig deckt jedes Pixel
// höchstens einmal, an der Fuge zwischen letztem Strich (Punkt) und Kopf bleibt keine helle Naht, und die Lücken sind
// wirklich leer.

var halbSchwarz = color.NRGBA{A: 128} // auf Weiß: einfach gedeckt 127, doppelt 63

func dunkelstesPixel(b *image.NRGBA) int {
	d := 255
	for i := 0; i < len(b.Pix); i += 4 {
		d = min(d, int(b.Pix[i]))
	}
	return d
}

func TestStrichartPfeilDecktEinmal(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		for _, kopf := range document.Kopfarten {
			for _, fall := range []struct {
				name    string
				a, k, b document.Punkt
				breite  float64
			}{
				{"gerade", document.Pt(30.3, 60.6), document.Pt(150.4, 60.6), document.Pt(270.5, 60.6), 6},
				{"schräg", document.Pt(40.2, 200.7), document.Pt(160, 140), document.Pt(280.3, 80.1), 9},
				{"gebogen", document.Pt(30, 150), document.Pt(150, 10), document.Pt(270, 150), 12},
			} {
				name := fmt.Sprintf("%s/%s/%s", art.Name(), kopf.Name(), fall.name)
				v := document.Pfeil{A: fall.a, B: fall.b, K: fall.k,
					Strich: document.Strich{Farbe: halbSchwarz, Breite: fall.breite, Art: art}}.MitKoepfen(kopf, kopf)
				d := weissesDokument(t, 320, 240)
				hinzu(t, d, v)
				b := bild(t, d)
				if g := dunkelstesPixel(b); g < 125 {
					t.Errorf("%s: dunkelstes Pixel %d, einfach gedeckt wäre 127: Teile überlappen", name, g)
				}
			}
		}
	}
}

// An der Fuge zwischen letztem Strich und Kopf (gerader Pfeil, Fuge mitten in einem Pixel) ist das Pixel auf der
// Mittellinie einfach gedeckt: Gegeneinander laufende Teile höben sich dort auf (helle Naht).
func TestStrichartFugeAmKopf(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		v := document.GeraderPfeil(document.Pt(20.5, 50.5), document.Pt(250.5, 50.5),
			document.Strich{Farbe: halbSchwarz, Breite: 10, Art: art}).MitKoepfen(document.KopfDreieck, document.KopfDreieck)
		d := weissesDokument(t, 300, 100)
		hinzu(t, d, v)
		b := bild(t, d)
		for _, k := range v.Form().Koepfe {
			for _, dx := range []float64{-2, -1, 0, 1, 2} {
				x, y := int(k.Schaftende.X+dx), 50
				if g := b.NRGBAAt(x, y).R; g < 120 || g > 134 {
					t.Errorf("%s: Fuge am Kopf %v, Pixel %d,%d = %d statt rund 127", art.Name(), k.Schaftende, x, y, g)
				}
			}
		}
	}
}

// Die Lücken eines gestrichelten und gepunkteten Strichs sind leer, die Striche gedeckt.
func TestStrichartLueckenLeer(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		v := document.GeraderPfeil(document.Pt(20, 50), document.Pt(280, 50),
			document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6, Art: art}).MitKoepfen(document.KopfKeine, document.KopfKeine)
		d := weissesDokument(t, 300, 100)
		hinzu(t, d, v)
		b := bild(t, d)
		weiss, schwarz, wechsel := 0, 0, 0
		vorher := byte(0)
		for x := 20; x <= 280; x++ {
			g := b.NRGBAAt(x, 50).R
			switch {
			case g == 255:
				weiss++
			case g == 0:
				schwarz++
			}
			if x > 20 && (g > 128) != (vorher > 128) {
				wechsel++
			}
			vorher = g
		}
		if weiss < 50 || schwarz < 30 || wechsel < 10 {
			t.Errorf("%s: weiß %d, schwarz %d, Wechsel %d auf der Mittellinie", art.Name(), weiss, schwarz, wechsel)
		}
		if art == document.StrichGestrichelt {
			// Gerade Enden: knapp hinter dem Ende des ersten Strichs (zwei Drittel der halben Breite) ist frei; eine runde
			// Kappe läge dort.
			m := document.StrichAbschnitte(v.Form().Schaft, art, 6)
			ende := m.Striche[0][len(m.Striche[0])-1]
			if g := b.NRGBAAt(int(ende.X+2.5), 50).R; g < 250 {
				t.Errorf("Strichende bei %v nicht gerade (Pixel dahinter %d)", ende, g)
			}
		}
		if g := b.NRGBAAt(20, 50).R; g > 10 {
			t.Errorf("%s: Anfang nicht gedeckt (%d)", art.Name(), g)
		}
		if g := b.NRGBAAt(279, 50).R; g > 10 {
			t.Errorf("%s: Ende nicht gedeckt (%d)", art.Name(), g)
		}
	}
}

// Kasten eckig und rund, Lupe: halbdurchsichtig einfach gedeckt, Ecken gedeckt.
func TestStrichartKastenDecktEinmal(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		for _, radius := range []float64{0, 6, 30} {
			for _, breite := range []float64{1, 6, 13} {
				name := fmt.Sprintf("%s/Radius %v/Strich %v", art.Name(), radius, breite)
				k := document.Kasten{Bereich: document.R(document.Pt(40.5, 30.25), document.Pt(260.25, 190.5)),
					Strich: document.Strich{Farbe: halbSchwarz, Breite: breite, Art: art}, Eckradius: radius}
				d := weissesDokument(t, 300, 220)
				hinzu(t, d, k)
				b := bild(t, d)
				if g := dunkelstesPixel(b); g < 125 {
					t.Errorf("%s: dunkelstes Pixel %d: Teile überlappen", name, g)
				}
				// Auf der Mittellinie der oberen Kante gibt es Lücken (weiß) und Striche bzw. Punkte (gedeckt).
				weiss, gedeckt := 0, 0
				for x := 90; x <= 210; x++ {
					switch g := b.NRGBAAt(x, 30).R; {
					case g == 255:
						weiss++
					case g < 140:
						gedeckt++
					}
				}
				if breite >= 6 && (weiss < 10 || gedeckt < 10) {
					t.Errorf("%s: obere Kante %d weiß, %d gedeckt", name, weiss, gedeckt)
				}
				if radius == 0 && breite >= 6 {
					for _, e := range []image.Point{{40, 30}, {260, 30}, {260, 190}, {40, 190}} {
						if g := b.NRGBAAt(e.X, e.Y).R; g > 135 {
							t.Errorf("%s: Ecke %v nicht gedeckt (%d)", name, e, g)
						}
					}
				}
			}
		}
	}
}

func TestStrichartLupeZeichnet(t *testing.T) {
	for _, art := range []document.Strichart{document.StrichGestrichelt, document.StrichGepunktet} {
		l := document.Lupe{Quelle: document.R(document.Pt(20, 20), document.Pt(60, 60)),
			Ziel:   document.R(document.Pt(150, 80), document.Pt(230, 160)),
			Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 4, Art: art}}
		d := weissesDokument(t, 300, 200)
		hinzu(t, d, l)
		b := bild(t, d)
		rot := 0
		for i := 0; i < len(b.Pix); i += 4 {
			if b.Pix[i] == 255 && b.Pix[i+1] == 0 {
				rot++
			}
		}
		if rot < 100 {
			t.Errorf("%s: nur %d rote Pixel", art.Name(), rot)
		}
	}
}

// F-840: Ein präparierter gemusterter Strich über die ganze Koordinatenspanne wird voll (MaxMusterStuecke), statt
// Millionen Stücke zu zerlegen; viele Muster knapp unter der Grenze zählen im Aufwand, Pruefe lehnt sie ab.
func TestStrichartMusterBegrenzt(t *testing.T) {
	riesig := document.GeraderPfeil(document.Pt(-1e6, -1e6), document.Pt(1e6, 1e6),
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 0.01, Art: document.StrichGepunktet})
	t0 := time.Now()
	if pz := ZerlegePfeil(riesig); len(pz.Teile) > 60 {
		t.Errorf("riesiger gepunkteter Pfeil: %d Teile, Soll voll (wenige)", len(pz.Teile))
	}
	if d := time.Since(t0); d > 100*time.Millisecond {
		t.Errorf("Zerlegen dauerte %v", d)
	}
	knapp := document.GeraderPfeil(document.Pt(-30000, 50), document.Pt(34000, 50),
		document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 1, Art: document.StrichGepunktet})
	if pz := ZerlegePfeil(knapp); len(pz.Teile) < 10000 || len(pz.Teile) > document.MaxMusterStuecke {
		t.Fatalf("knapp unter der Grenze: %d Teile", len(pz.Teile))
	}
	d := weissesDokument(t, 200, 100)
	for range 1000 {
		hinzu(t, d, knapp)
	}
	t0 = time.Now()
	if err := Pruefe(d); err == nil {
		t.Error("1000 Muster knapp unter der Grenze: Pruefe nimmt sie an")
	}
	if dauer := time.Since(t0); dauer > 15*time.Second {
		t.Errorf("Pruefe brauchte %v", dauer)
	}
}

// V-1 der Sitzungs-Kritik 26.11. (5) (Rest von F-840): Gepunktete eckige Rahmen, die fast ganz außerhalb des Bilds
// liegen, zählen ihre Stücke; viele solcher Lupen lehnt Pruefe ab, statt das Öffnen lange aufzuhalten. Ein Rahmen ganz
// außerhalb wird gar nicht zerlegt.
func TestStrichartRahmenAusserhalbGezaehlt(t *testing.T) {
	l := document.Lupe{Quelle: document.R(document.Pt(201, 0), document.Pt(8401, 8200)),
		Ziel:   document.R(document.Pt(190, 0), document.Pt(200, 10)),
		Strich: document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 0.01, Art: document.StrichGepunktet}}
	d := weissesDokument(t, 200, 100)
	for range 2000 {
		hinzu(t, d, l)
	}
	t0 := time.Now()
	if err := Pruefe(d); err == nil {
		t.Error("2000 Lupen mit riesigem gepunktetem Rahmen: Pruefe nimmt sie an")
	}
	if dauer := time.Since(t0); dauer > 15*time.Second {
		t.Errorf("Pruefe brauchte %v", dauer)
	}
	ganz := l
	ganz.Quelle = document.R(document.Pt(1000, 1000), document.Pt(9000, 9000))
	d2 := weissesDokument(t, 200, 100)
	hinzu(t, d2, ganz)
	if a := trockenlauf(d2, false); a.kanten > 5e6 { // Ziel und Linie zählen, der Quellrahmen allein läge bei 1,6e7
		t.Errorf("Rahmen ganz außerhalb kostet %v", a.kanten)
	}
}
