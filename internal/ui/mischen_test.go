// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// mischDokument ist ein 96×64-Bild aus bunten Streifen (kein Grau, damit sRGB- und lineare Mischung weit auseinander
// liegen) mit halbdurchsichtigen Beschriftungen: ein Kasten mit Füllung und Strich, ein Kasten in einer Ebene mit
// Deckkraft 40 % und ein Text mit halbdurchsichtigem Schild.
func mischDokument(t *testing.T) *editor {
	t.Helper()
	i18n.Setze(nil)
	bild := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			bild.SetNRGBA(x, y, color.NRGBA{R: uint8(40 + 20*(x/12)), G: uint8(220 - 25*(y/8)), B: uint8(30 + 25*((x+y)/16)), A: 255})
		}
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	add := func(o document.Objekt) int {
		id, err := d.Hinzufuegen(o)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	add(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(40, 28)),
		Fuellung: color.NRGBA{R: 255, G: 255, A: 128}, Strich: document.Strich{Farbe: color.NRGBA{R: 200, B: 255, A: 160}, Breite: 4}})
	k := add(document.Kasten{Bereich: document.R(document.Pt(50, 6), document.Pt(90, 40)), Fuellung: color.NRGBA{R: 255, A: 255}, Strich: document.Strich{Farbe: color.NRGBA{B: 255, A: 255}, Breite: 6}})
	if !e.s.SetzeDeckkraft(k, 40) {
		t.Fatal("Deckkraft der Ebene nicht gesetzt")
	}
	add(document.Text{Ort: document.Pt(6, 50), Inhalt: "M", Schriftgrad: 14, Farbe: color.NRGBA{A: 255},
		Hintergrund: color.NRGBA{R: 255, G: 255, B: 255, A: 100}})
	return e
}

// F-520, ADR-0020 Nachtrag 14: Der Editor mischt halbdurchsichtige Beschriftungen im sRGB-Raum wie der Renderer
// (vorher linear: bis zu 40 von 255 zu hell). Verglichen wird das Zeichnen des Editors (Gio ohne Fenster, Zoom 100 %)
// über dem Ansichtsbild mit render.Bild; Pixel an Kanten (Glättung) zählen nicht.
func TestEditorMischtWieRenderer(t *testing.T) {
	e := mischDokument(t)
	soll, err := render.Bild(e.s.D)
	if err != nil {
		t.Fatal(err)
	}
	ist := headlessBild(t, image.Pt(96, 64), func(gtx layout.Context) {
		paint.NewImageOp(e.s.Leinwandbild()).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		e.zeichneObjekte(gtx)
	})
	pruefe := func(name string, r image.Rectangle) {
		t.Helper()
		groesst := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				a, b := ist.RGBAAt(x, y), soll.NRGBAAt(x, y)
				groesst = max(groesst, absInt(int(a.R)-int(b.R)), absInt(int(a.G)-int(b.G)), absInt(int(a.B)-int(b.B)))
			}
		}
		t.Logf("%s: größte Abweichung %d", name, groesst)
		if groesst > 2 {
			t.Errorf("%s: größte Abweichung %d je Kanal, Soll höchstens 2", name, groesst)
		}
	}
	pruefe("Füllung 50 %", image.Rect(10, 10, 34, 22))
	pruefe("Strich 63 %", image.Rect(5, 5, 7, 27))
	pruefe("Ebene 40 %", image.Rect(54, 10, 86, 36))
	pruefe("Ebene 40 %, Strich über Füllung (Gruppe)", image.Rect(51, 10, 53, 36))
	pruefe("Schild 39 %", image.Rect(4, 36, 9, 38))
}

// Deckendes bleibt, wie es war: Deckende Flächen schreiben ihre sRGB-Werte unverändert (Menü, Leisten, Schachbrett,
// Auswahlrahmen, ADR-0020 Nachtrag 14).
func TestEditorDeckendBleibtUnveraendert(t *testing.T) {
	farbe := color.NRGBA{R: 37, G: 142, B: 201, A: 255}
	b := headlessBild(t, image.Pt(8, 8), func(gtx layout.Context) { paint.Fill(gtx.Ops, farbe) })
	if p := b.RGBAAt(3, 3); p.R != 37 || p.G != 142 || p.B != 201 || p.A != 255 {
		t.Errorf("deckende Fläche: Pixel %v, Soll (37,142,201,255)", p)
	}
}

// F-621, ADR-0020 Nachtrag 14: Die Zwischenebene einer Deckkraftgruppe (`paint.PushOpacity`) ist eine RGBA8-Textur, kein
// sRGB-Puffer. Das zeigt nur eine Gruppe, in der sich Halbdurchsichtiges selbst überlappt: Im sRGB-Puffer mischt die
// Hardware dort linear (die Ebenen-Fälle von TestEditorMischtWieRenderer mit deckender Füllung und deckendem Strich
// blieben gleich, der Mutant überlebte). Die Gruppe wird hier von Hand im sRGB-Raum gerechnet: Rot 50 %, darüber Blau
// 50 %, dann als Ganzes mit 50 % über Grün.
func TestZwischenebeneMischtImSRGBRaum(t *testing.T) {
	ist := headlessBild(t, image.Pt(40, 20), func(gtx layout.Context) {
		paint.Fill(gtx.Ops, color.NRGBA{G: 200, A: 255})
		defer paint.PushOpacity(gtx.Ops, 0.5).Pop()
		flaeche := func(r image.Rectangle, c color.NRGBA) {
			defer clip.Rect(r).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, c)
		}
		flaeche(image.Rect(2, 2, 30, 18), color.NRGBA{R: 255, A: 128})
		flaeche(image.Rect(10, 2, 38, 18), color.NRGBA{B: 255, A: 128})
	})
	// Gruppe in der Überlappung (x 10..30), vormultipliziert, 0..1: Blau (a1) über Rot (a1).
	a1 := 128.0 / 255
	gR, gB, gA := a1*(1-a1), a1, a1+a1*(1-a1)
	const d = 0.5 // Gruppendeckkraft, hinten Grün (0, 200/255, 0)
	soll := func(g, hinten float64) int { return int(255*(g*d+hinten*(1-gA*d)) + 0.5) }
	sR, sG, sB := soll(gR, 0), soll(0, 200.0/255), soll(gB, 0)
	p := ist.RGBAAt(20, 10)
	if absInt(int(p.R)-sR) > 3 || absInt(int(p.G)-sG) > 3 || absInt(int(p.B)-sB) > 3 {
		t.Errorf("Pixel in der Überlappung %v, Soll (%d,%d,%d) im sRGB-Raum", p, sR, sG, sB)
	}
}

// F-631 (Kritik C-11), ADR-0020 Nachtrag 14: Wie TestZwischenebeneMischtImSRGBRaum, aber über einem Verlauf und mit
// vielen Pixeln: Jedes Pixel der Überlappung weicht höchstens um 1 je Kanal von der Handrechnung im sRGB-Raum ab. Mit
// dem Mutanten M215-4 (Zwischenebene als sRGB-Textur) gemessen: dieselbe Grenze, also hebt sich Kodieren und Dekodieren
// der Zwischenebene auf (GLES kennt kein abschaltbares FRAMEBUFFER_SRGB; nur die 8-Bit-Rundung liegt anders).
func TestZwischenebeneVieleStellen(t *testing.T) {
	const w, h = 64, 48
	hinten := func(x, y int) (float64, float64, float64) {
		return float64(x*4) / 255, float64(255-y*5) / 255, float64(x*2+y*3) / 255
	}
	ist := headlessBild(t, image.Pt(w, h), func(gtx layout.Context) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b := hinten(x, y)
				func() {
					defer clip.Rect(image.Rect(x, y, x+1, y+1)).Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: uint8(r*255 + 0.5), G: uint8(g*255 + 0.5), B: uint8(b*255 + 0.5), A: 255})
				}()
			}
		}
		defer paint.PushOpacity(gtx.Ops, 0.5).Pop()
		flaeche := func(r image.Rectangle, c color.NRGBA) {
			defer clip.Rect(r).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, c)
		}
		flaeche(image.Rect(4, 4, 44, 44), color.NRGBA{R: 255, G: 40, A: 128})
		flaeche(image.Rect(20, 4, 60, 44), color.NRGBA{B: 255, G: 90, A: 128})
	})
	a1 := 128.0 / 255
	gA := a1 + a1*(1-a1)
	const d = 0.5
	grösst, n := 0, 0
	for y := 4; y < 44; y++ {
		for x := 20; x < 44; x++ { // Überlappung
			hr, hg, hb := hinten(x, y)
			gR := a1 * (1 - a1) * 1
			gG := a1*(1-a1)*40/255 + a1*90/255
			gB := a1
			soll := func(g, hi float64) int { return int(255*(g*d+hi*(1-gA*d)) + 0.5) }
			p := ist.RGBAAt(x, y)
			grösst = max(grösst, absInt(int(p.R)-soll(gR, hr)), absInt(int(p.G)-soll(gG, hg)), absInt(int(p.B)-soll(gB, hb)))
			n++
		}
	}
	t.Logf("%d Pixel, größte Abweichung %d", n, grösst)
	if grösst > 1 {
		t.Errorf("größte Abweichung %d je Kanal über %d Pixel, Soll höchstens 1", grösst, n)
	}
}

// F-632, ADR-0020 Nachtrag 14: Die Abdunklung von Auswahl-Overlay und Zuschneiden wirkt wie vor der Umstellung auf das
// Mischen im sRGB-Raum, als Gio linear mischte und 0x99 Schwarzanteil über Weiß 170 ergab. Soll: die alte Linearmischung,
// gerechnet; Ist: das Zeichnen mit den heutigen Konstanten in Gio ohne Fenster. Über Weiß, Grau 128 und Dunkelgrau 40.
func TestAbdunkelnWieVorDerUmstellung(t *testing.T) {
	lin := func(v float64) float64 {
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	kodiere := func(l float64) float64 {
		if l <= 0.0031308 {
			return l * 12.92
		}
		return 1.055*math.Pow(l, 1/2.4) - 0.055
	}
	alt := func(grund, farbe uint8) int { // 0x99 linear gemischt, wie Gio vor 26fd556
		a := 0x99 / 255.0
		return int(255*kodiere(lin(float64(grund)/255)*(1-a)+lin(float64(farbe)/255)*a) + 0.5)
	}
	if got := alt(255, 0x16); got < 169 || got > 171 {
		t.Fatalf("Rechnung der alten Mischung über Weiß: %d, Soll 170", got)
	}
	for _, c := range []struct {
		name  string
		farbe color.NRGBA
	}{{"Overlay grau-950", farbeAbdunkeln}, {"Zuschneiden Schwarz", farbeZuschnittDunkel}} {
		for _, grund := range []uint8{255, 128, 40} {
			b := headlessBild(t, image.Pt(4, 4), func(gtx layout.Context) {
				paint.Fill(gtx.Ops, color.NRGBA{R: grund, G: grund, B: grund, A: 255})
				paint.FillShape(gtx.Ops, c.farbe, clip.Rect(image.Rect(0, 0, 4, 4)).Op())
			})
			ist := int(b.RGBAAt(2, 2).G)
			soll := alt(grund, c.farbe.G)
			if absInt(ist-soll) > 5 {
				t.Errorf("%s über %d: Pixel %d, Soll (alte Linearmischung) %d ±5", c.name, grund, ist, soll)
			}
			if grund == 255 && absInt(ist-170) > 3 {
				t.Errorf("%s über Weiß: %d, Soll 170", c.name, ist)
			}
		}
	}
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
