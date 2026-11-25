// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/raster16"
	"golang.org/x/image/font"
)

// abstandZurLinie ist der Abstand von p zum offenen Linienzug pts (ein Punkt: Abstand zu ihm).
func abstandZurLinie(p document.Punkt, pts []document.Punkt) float64 {
	if len(pts) == 1 {
		return p.Minus(pts[0]).Laenge()
	}
	best := math.Inf(1)
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		d := b.Minus(a)
		l := d.X*d.X + d.Y*d.Y
		t := 0.0
		if l > 0 {
			t = min(max(((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l, 0), 1)
		}
		best = min(best, p.Minus(a.Plus(d.Mal(t))).Laenge())
	}
	return best
}

// linienAbweichung misst, wie weit die Deckung von b (schwarz auf weiß) von der analytischen Deckung des Strichs der
// halben Breite hw um den offenen Linienzug pts abweicht: an Pixeln nahe der Kante geschätzt mit 32×32 Stichproben je
// Pixel, weiter innen soll das Pixel ganz, weiter außen gar nicht gedeckt sein (kein Loch, nichts daneben); ohne die
// Pixel, für die aus wahr ist (andere Teile des Objekts). Liefert die größte Abweichung an der Kante und abseits davon,
// je mit ihrem Pixel, und die Zahl der geprüften Kantenpixel.
func linienAbweichung(b *image.NRGBA, pts []document.Punkt, hw float64, aus func(document.Punkt) bool) abweichung {
	g := huelleVon(pts, hw+2)
	r := image.Rect(int(math.Floor(g.Min.X)), int(math.Floor(g.Min.Y)), int(math.Ceil(g.Max.X)), int(math.Ceil(g.Max.Y))).
		Intersect(b.Rect)
	const n = 32
	var a abweichung
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			mitte := document.Pt(float64(x)+0.5, float64(y)+0.5)
			if aus != nil && aus(mitte) {
				continue
			}
			ist := 1 - float64(b.NRGBAAt(x, y).R)/255
			m := abstandZurLinie(mitte, pts)
			if math.Abs(m-hw) > 1 {
				soll := 0.0
				if m < hw {
					soll = 1
				}
				if f := math.Abs(ist - soll); f > a.innen {
					a.innen, a.woInnen = f, image.Pt(x, y)
				}
				continue
			}
			treffer := 0
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					p := document.Pt(float64(x)+(float64(sx)+0.5)/n, float64(y)+(float64(sy)+0.5)/n)
					if abstandZurLinie(p, pts) <= hw {
						treffer++
					}
				}
			}
			a.geprueft++
			if f := math.Abs(ist - float64(treffer)/(n*n)); f > a.kante {
				a.kante, a.woKante = f, image.Pt(x, y)
			}
		}
	}
	return a
}

// abweichung ist das Ergebnis von linienAbweichung.
type abweichung struct {
	kante, innen     float64 // größte Abweichung an der Kante bzw. abseits (ganz drinnen oder draußen)
	woKante, woInnen image.Point
	geprueft         int // Kantenpixel
}

// pruefeAbweichung meldet Abweichungen über grenze an der Kante und jede abseits der Kante (über 0,01: ein Loch oder
// Deckung daneben).
func pruefeAbweichung(t *testing.T, name string, a abweichung, grenze float64, mindestens int) {
	t.Helper()
	if a.geprueft < mindestens {
		t.Fatalf("%s: nur %d Kantenpixel", name, a.geprueft)
	}
	t.Logf("%s: höchstens %.4f Abweichung über %d Kantenpixel (Pixel %v), abseits %.4f", name, a.kante, a.geprueft,
		a.woKante, a.innen)
	if a.kante > grenze {
		t.Errorf("%s: Deckung weicht bis %.3f von der analytischen ab, zulässig %.3f (Pixel %v)", name, a.kante, grenze,
			a.woKante)
	}
	if a.innen > 0.01 {
		t.Errorf("%s: abseits der Kante weicht die Deckung bis %.3f ab (Pixel %v): Loch oder Deckung daneben", name,
			a.innen, a.woInnen)
	}
}

// linienGrenze ist die zulässige Abweichung: Stichprobenfehler (1/32), die Sehne der Bögen an Gelenken und Enden
// (höchstens bogenSehne unter dem Kreis) und Rundung der 8-Bit-Farbe.
const linienGrenze = 1.0/32 + bogenSehne + 0.01

// abstandZumDreieck ist 0 im Dreieck abc, sonst der Abstand zu seinem Rand.
func abstandZumDreieck(p, a, b, c document.Punkt) float64 {
	kreuz := func(u, v, w document.Punkt) float64 { return (v.X-u.X)*(w.Y-u.Y) - (v.Y-u.Y)*(w.X-u.X) }
	s1, s2, s3 := kreuz(a, b, p), kreuz(b, c, p), kreuz(c, a, p)
	if (s1 >= 0 && s2 >= 0 && s3 >= 0) || (s1 <= 0 && s2 <= 0 && s3 <= 0) {
		return 0
	}
	return abstandZumUmriss(p, []document.Punkt{a, b, c})
}

// pfeilLinie ist der Linienzug, den Renderer und Editor für den Schaft von v ziehen (48 Strecken), und eine Prüfung, ob
// ein Punkt höchstens 2 Pixel von der Spitze entfernt liegt (dort deckt die Spitze in einer eigenen Füllung).
func pfeilLinie(v document.Pfeil) ([]document.Punkt, func(document.Punkt) bool) {
	const n = 48
	pts := make([]document.Punkt, 0, n+1)
	for i := 0; i <= n; i++ {
		pts = append(pts, v.Punkt(float64(i)/n))
	}
	r := v.RichtungAmEnde()
	g := math.Max(8, 4*v.Strich.Breite)
	nrm := document.Pt(-r.Y, r.X)
	fuss := v.B.Minus(r.Mal(g))
	a, b, c := v.B, fuss.Plus(nrm.Mal(g/2)), fuss.Minus(nrm.Mal(g/2))
	return pts, func(p document.Punkt) bool { return abstandZumDreieck(p, a, b, c) <= 2 }
}

// F-796 (ADR-0023 Nachtrag 22): Der Schaft eines Pfeils, gerade und gebogen, deckt jedes Pixel so, wie die Fläche aller
// Punkte höchstens eine halbe Strichbreite vom Linienzug es analytisch deckt. Vorher füllte der Renderer je Strecke ein
// Rechteck und je Punkt einen Kreis in einer Füllung; an jedem Gelenk addierte sich deren Kantenglättung (Abweichung bis
// 0,3 bis 0,5, Ist-Messung im Nachtrag). Die Spitze ist eine eigene Füllung; Pixel nahe der Spitze zählen nicht.
func TestPfeilStrichDecktWieAnalytisch(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for _, fall := range []struct {
		name    string
		a, k, b document.Punkt
		breite  float64
	}{
		{"gerade schräg Strich 24", document.Pt(30.3, 40.7), document.Pt(140.45, 95.45), document.Pt(250.6, 150.2), 24},
		{"gebogen Strich 24", document.Pt(30, 150), document.Pt(140, 20), document.Pt(260, 140), 24},
		{"gebogen Strich 3", document.Pt(30.4, 150.6), document.Pt(140, 20), document.Pt(260.2, 140.3), 3},
		{"gebogen Strich 60", document.Pt(40, 180), document.Pt(150, 30), document.Pt(270, 170), 60},
		{"S-nah gebogen Strich 10", document.Pt(20, 30), document.Pt(280, 40), document.Pt(40, 180), 10},
	} {
		v := document.Pfeil{A: fall.a, B: fall.b, K: fall.k, Strich: document.Strich{Farbe: schwarz, Breite: fall.breite}}
		d := weissesDokument(t, 320, 240)
		hinzu(t, d, v)
		b := bild(t, d)
		pts, nahSpitze := pfeilLinie(v)
		pruefeAbweichung(t, fall.name, linienAbweichung(b, pts, fall.breite/2, nahSpitze), linienGrenze, 100)
	}
}

// F-796: Lupen-Linie und Schenkel der Sprechblase ziehen ihren Strich über zeichner.strich (eine Strecke bzw. zwei mit
// einem spitzen Gelenk). Geprüft direkt am Strich, weil die Rahmen der Lupe die Enden der Linie überdecken und die
// Schenkel am Schild enden.
func TestStrichDecktWieAnalytisch(t *testing.T) {
	for _, fall := range []struct {
		name   string
		pts    []document.Punkt
		breite float64
	}{
		{"Lupen-Linie schräg Strich 8", []document.Punkt{document.Pt(30.2, 40.6), document.Pt(250.7, 170.1)}, 8},
		{"Lupen-Linie Strich 2", []document.Punkt{document.Pt(30.5, 40.25), document.Pt(250.5, 120.75)}, 2},
		{"Schenkel spitz Strich 6", []document.Punkt{document.Pt(40, 40), document.Pt(260, 120.4), document.Pt(60.3, 70)}, 6},
		{"Schenkel stumpf Strich 12", []document.Punkt{document.Pt(40, 60), document.Pt(160.5, 190.5), document.Pt(270, 50)}, 12},
		{"ein Punkt Strich 20", []document.Punkt{document.Pt(150.3, 120.6)}, 20},
	} {
		pruefeAbweichung(t, fall.name, linienAbweichung(strichBild(t, fall.pts, fall.breite), fall.pts, fall.breite/2, nil),
			linienGrenze, 50)
	}
}

// strichBild zieht den Strich pts der Breite breite schwarz auf ein weißes Bild 320×240 (zeichner.strich, Ansicht).
func strichBild(t *testing.T, pts []document.Punkt, breite float64) *image.NRGBA {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 320, 240))
	for i := range b.Pix {
		b.Pix[i] = 255
	}
	z := zeichner{bild: b, rand: b.Bounds(), d: weissesDokument(t, 320, 240), faces: map[faceSchluessel]font.Face{},
		auf: &aufwand{}, ras: &raster16.Rasterizer{}, masken: neuerMaskenCache()}
	z.strich(pts, false, document.Strich{Farbe: color.NRGBA{A: 255}, Breite: breite})
	return b
}

// F-796, Faltung (ADR-0023 Nachtrag 22): Wo der Krümmungsradius unter der halben Breite liegt oder ein spitzes Gelenk
// längere Gehrungen bräuchte als seine Strecken lang sind, enden die Teile innen senkrecht und überlappen. Dort bleibt
// kein Loch und nichts daneben; an der Kante addiert sich die Glättung nur an diesen Gelenken (gemessen, nicht
// begrenzt; vorher galt das an jedem Gelenk). Abseits der gefalteten Gelenke bleibt die Kante wie analytisch: Der
// zweite Teil misst nur Pixel mindestens hw+2 von ihnen entfernt. Beim spitzen, kurzen V entfällt er: Ein V-Gelenk
// faltet nur, wenn seine Enden näher als 2·hw beieinander liegen (hw·cot(α/2) > Länge heißt 2·Länge·sin(α/2) < 2·hw·
// cos(α/2)); dann überlappen auch die Kappen, das ganze V ist Faltung.
func TestStrichGefaltetOhneLoch(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	haarnadel := document.Pfeil{A: document.Pt(100, 200), K: document.Pt(120, -100), B: document.Pt(140, 200),
		Strich: document.Strich{Farbe: schwarz, Breite: 24}}
	pts, _ := pfeilLinie(haarnadel)
	for _, fall := range []struct {
		name   string
		pts    []document.Punkt
		breite float64
		fern   bool // Kante abseits der Faltung prüfen
	}{
		{"Haarnadel Strich 24", pts, 24, true},
		{"Schenkel sehr spitz kurz Strich 12", []document.Punkt{document.Pt(100, 100), document.Pt(160, 110), document.Pt(102, 106)}, 12, false},
		// Gehrung 1,3-mal so lang wie die Schenkel (Spitze 13,2°, Schenkel 40, Strich 12): knapp gefaltet.
		{"Schenkel spitz knapp gefaltet Strich 12", []document.Punkt{document.Pt(160.27, 115.4), document.Pt(200, 120), document.Pt(160.27, 124.6)}, 12, false},
		{"Kehrtwende Strich 10", []document.Punkt{document.Pt(60, 100), document.Pt(200, 100), document.Pt(120, 100)}, 10, true},
		// F-803: An einer achsparallelen Kehrtwende ist der Sinus ±0; in zwei der vier Richtungen drehte der Fächer
		// rückwärts, die runde Kappe am Umkehrpunkt fehlte (abseits der Kante 1,0).
		{"Kehrtwende nach links Strich 10", []document.Punkt{document.Pt(200, 100), document.Pt(60, 100), document.Pt(120, 100)}, 10, true},
		{"Kehrtwende nach unten Strich 10", []document.Punkt{document.Pt(100, 40), document.Pt(100, 180), document.Pt(100, 120)}, 10, true},
		{"Kehrtwende nach oben Strich 10", []document.Punkt{document.Pt(100, 180), document.Pt(100, 40), document.Pt(100, 120)}, 10, true},
	} {
		ls := neuerLinienStrich(fall.pts, false, fall.breite/2)
		var falten []document.Punkt
		for k, p := range ls.pts {
			if ls.gefaltet[k] {
				falten = append(falten, p)
			}
		}
		if len(falten) == 0 {
			t.Errorf("%s: kein gefaltetes Gelenk, der Fall prüft die Faltung nicht", fall.name)
		}
		b := strichBild(t, fall.pts, fall.breite)
		a := linienAbweichung(b, fall.pts, fall.breite/2, nil)
		t.Logf("%s: %d gefaltete Gelenke, an der Kante höchstens %.4f (Pixel %v), abseits %.4f (Pixel %v)", fall.name,
			len(falten), a.kante, a.woKante, a.innen, a.woInnen)
		if a.innen > 0.01 {
			t.Errorf("%s: abseits der Kante weicht die Deckung bis %.3f ab (Pixel %v): Loch oder Deckung daneben",
				fall.name, a.innen, a.woInnen)
		}
		if !fall.fern {
			continue
		}
		fern := func(p document.Punkt) bool {
			for _, f := range falten {
				if p.Minus(f).Laenge() < fall.breite/2+2 {
					return true
				}
			}
			return false
		}
		pruefeAbweichung(t, fall.name+" fern der Faltung", linienAbweichung(b, fall.pts, fall.breite/2, fern), linienGrenze, 20)
	}
}

// F-803: Ein Pfeil, dessen Kontrollpunkt auf der Linie hinter dem Ziel liegt, kehrt an einem Stützpunkt genau um. In
// allen vier Achsrichtungen trägt der Umkehrpunkt die runde Kappe: abseits der Kante kein Pixel daneben und kein Loch.
// Vorher fehlte sie nach links und nach oben (Ende flach, abseits der Kante 1,0).
func TestPfeilKehrtwendeAlleRichtungen(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for _, fall := range []struct {
		name    string
		a, k, b document.Punkt
	}{
		{"nach links, kehrt um", document.Pt(300, 100), document.Pt(-150, 100), document.Pt(250, 100)},
		{"nach rechts, kehrt um", document.Pt(20, 100), document.Pt(470, 100), document.Pt(70, 100)},
		{"nach unten, kehrt um", document.Pt(100, 20), document.Pt(100, 370), document.Pt(100, 60)},
		{"nach oben, kehrt um", document.Pt(100, 220), document.Pt(100, -130), document.Pt(100, 180)},
	} {
		v := document.Pfeil{A: fall.a, B: fall.b, K: fall.k, Strich: document.Strich{Farbe: schwarz, Breite: 12}}
		d := weissesDokument(t, 320, 240)
		hinzu(t, d, v)
		b := bild(t, d)
		pts, nahSpitze := pfeilLinie(v)
		a := linienAbweichung(b, pts, 6, nahSpitze)
		t.Logf("%s: an der Kante höchstens %.4f (Pixel %v), abseits %.4f (Pixel %v)", fall.name, a.kante, a.woKante,
			a.innen, a.woInnen)
		if a.geprueft < 50 {
			t.Fatalf("%s: nur %d Kantenpixel", fall.name, a.geprueft)
		}
		if a.innen > 0.01 {
			t.Errorf("%s: abseits der Kante weicht die Deckung bis %.3f ab (Pixel %v): Kappe am Umkehrpunkt fehlt",
				fall.name, a.innen, a.woInnen)
		}
	}
}
