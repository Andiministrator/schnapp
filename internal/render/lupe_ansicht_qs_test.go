// SPDX-License-Identifier: GPL-3.0-or-later

package render

// Sitzungs-QA vom 24.11.2026 (Kritiker Fable), Verify der Lupen-Behebung (ADR-0033 Nachträge 2 und 3): Zufallsproben,
// dass eine Lupe nur zeigt, was die Ansicht zeigt (QS-V1, QS-V1b), und dass Deckkraft die Kosten der Quelle nicht umgeht
// (QS-V2). Als Regressionstests übernommen.

import (
	"image"
	"image/color"
	"math/rand"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// qaFarbe gibt jedem Pixel (x, y) bis 1024×1024 eine eigene deckende Farbe; qaOrt rechnet zurück.
func qaFarbe(x, y int) color.NRGBA {
	return color.NRGBA{uint8(x), uint8(y), uint8(0x80 | (x>>8)&3 | ((y>>8)&3)<<2), 255}
}
func qaOrt(c color.NRGBA) (int, int, bool) {
	if c.A != 255 || c.B&0x80 == 0 || c.B&0x70 != 0 {
		return 0, 0, false
	}
	return int(c.R) | int(c.B&3)<<8, int(c.G) | int(c.B>>2&3)<<8, true
}

type qaFall struct {
	d        *document.Dokument
	crop     image.Rectangle
	streifen []document.Streifen
	l        document.Lupe
}

// qaZufall baut ein Dokument mit Zuschnitt, bis zu zwei Streifen je Achse, zufälliger Kantenform und einer Lupe mit
// Bruchteil-Koordinaten (Quelle und Ziel dürfen über Zuschnitt und Original hinausragen).
func qaZufall(rnd *rand.Rand) (qaFall, bool) {
	w, h := 260+rnd.Intn(300), 260+rnd.Intn(200)
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, qaFarbe(x, y))
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		return qaFall{}, false
	}
	f := qaFall{d: d, crop: image.Rect(0, 0, w, h)}
	if rnd.Intn(3) > 0 {
		f.crop = image.Rect(rnd.Intn(30), rnd.Intn(30), w-rnd.Intn(30), h-rnd.Intn(30))
		if d.SetzeZuschnitt(f.crop) != nil {
			return qaFall{}, false
		}
	}
	for _, zeilen := range []bool{false, true} {
		lo, hi := f.crop.Min.X, f.crop.Max.X
		if zeilen {
			lo, hi = f.crop.Min.Y, f.crop.Max.Y
		}
		pos := lo + 40 + rnd.Intn(40)
		for n := rnd.Intn(3); n > 0 && pos+60 < hi; n-- {
			br := 1 + rnd.Intn(40)
			f.streifen = append(f.streifen, document.Streifen{Zeilen: zeilen, Von: pos, Bis: pos + br})
			pos += br + rnd.Intn(70) // auch 0: berührende Streifen / Nähte
		}
	}
	if len(f.streifen) > 0 && d.SetzeStreifen(f.streifen) != nil {
		return qaFall{}, false
	}
	if d.SetzeKante(document.Kante(rnd.Intn(3))) != nil {
		return qaFall{}, false
	}
	rect := func(maxw, maxh float64) document.Rechteck {
		x, y := -20+rnd.Float64()*float64(w), -20+rnd.Float64()*float64(h)
		return document.R(document.Pt(x, y), document.Pt(x+0.3+rnd.Float64()*maxw, y+0.3+rnd.Float64()*maxh))
	}
	f.l = document.Lupe{Quelle: rect(120, 90), Ziel: rect(300, 200)}
	id, err := d.Hinzufuegen(f.l)
	if err != nil {
		return qaFall{}, false
	}
	f.l.ID = id
	return f, true
}

// sichtbarImOriginal: Pixel (ox, oy) liegt im Zuschnitt und in keinem Streifen; dazu sein Ort in der Ansicht.
func (f qaFall) ansichtsOrt(ox, oy int) (image.Point, bool) {
	if !image.Pt(ox, oy).In(f.crop) {
		return image.Point{}, false
	}
	vx, vy := ox-f.crop.Min.X, oy-f.crop.Min.Y
	for _, s := range f.streifen {
		v, p := oy, &vy
		if !s.Zeilen {
			v, p = ox, &vx
		}
		if v >= s.Von && v < s.Bis {
			return image.Point{}, false
		}
		if v >= s.Bis {
			*p -= s.Bis - s.Von
		}
	}
	return image.Pt(vx, vy), true
}

// QS-V1: Kein Pixel einer Lupe (LupenInhalt, LupenVorschau) stammt aus dem Weggeschnittenen, aus einem Streifen, von
// außerhalb der Quelle oder von einem Pixel, das die Ansicht ohne Lupe ganz radiert (Spalt).
func TestQSLupeZeigtNurAnsicht(t *testing.T) {
	rnd := rand.New(rand.NewSource(20261124))
	faelle, pixel, teilweise := 0, 0, 0
	for it := 0; it < 400; it++ {
		f, ok := qaZufall(rnd)
		if !ok {
			continue
		}
		faelle++
		l := f.l
		// Ansicht ohne die Lupe: Rückgängig nimmt nur das Hinzufügen der Lupe zurück
		if err := f.d.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
		ohne, err := Bild(f.d)
		if err == nil {
			err = f.d.Wiederholen()
		}
		if err != nil {
			t.Fatal(err)
		}
		pruefe := func(name string, b *image.NRGBA) {
			if b == nil {
				return
			}
			for y := b.Rect.Min.Y; y < b.Rect.Max.Y; y++ {
				for x := b.Rect.Min.X; x < b.Rect.Max.X; x++ {
					c := b.NRGBAAt(x, y)
					if c.A == 0 {
						continue
					}
					pixel++
					ox, oy, ok := qaOrt(c)
					if !ok {
						t.Fatalf("Fall %d %s: Pixel %d,%d trägt keine Originalfarbe: %v", it, name, x, y, c)
					}
					v, sichtbar := f.ansichtsOrt(ox, oy)
					if !sichtbar {
						t.Fatalf("Fall %d %s: LECK — Pixel %d,%d zeigt Original %d,%d, das die Ansicht nicht zeigt (Zuschnitt %v, Streifen %v, Lupe %v)",
							it, name, x, y, ox, oy, f.crop, f.streifen, l)
					}
					if float64(ox+1) <= l.Quelle.Min.X || float64(ox) >= l.Quelle.Max.X || float64(oy+1) <= l.Quelle.Min.Y || float64(oy) >= l.Quelle.Max.Y {
						t.Fatalf("Fall %d %s: Pixel %d,%d zeigt Original %d,%d außerhalb der Quelle %v", it, name, x, y, ox, oy, l.Quelle)
					}
					switch a := ohne.NRGBAAt(v.X, v.Y).A; {
					case a == 0:
						t.Fatalf("Fall %d %s: Pixel %d,%d zeigt Original %d,%d (Ansicht %v), das die Ansicht ganz radiert (Spalt); Streifen %v",
							it, name, x, y, ox, oy, v, f.streifen)
					case a != 255:
						teilweise++
					}
				}
			}
		}
		pruefe("LupenInhalt", LupenInhalt(f.d, l))
		pruefe("LupenVorschau", LupenVorschau(f.d, l, 96))
	}
	t.Logf("%d Fälle, %d Lupenpixel geprüft, davon %d von teilweise radierten Ansichtspixeln (Kantenglättung, Nachtrag 3 Punkt 2)", faelle, pixel, teilweise)
}

// QS-V2: Gruppendeckkraft — eine Lupe mit 50 % kostet im Trockenlauf nicht weniger als mit 100 % (sonst umginge
// Deckkraft die Kosten der Quelle).
func TestQSBudgetDeckkraft(t *testing.T) {
	kosten := func(p int) float64 {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
		if err != nil {
			t.Fatal(err)
		}
		l := document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(400, 300)), Ziel: document.R(document.Pt(10, 10), document.Pt(11, 11))}
		l.Ebene = l.Ebene.MitDeckkraft(p)
		if _, err := d.Hinzufuegen(l); err != nil {
			t.Fatal(err)
		}
		return Aufwand(d)
	}
	a, h, n := kosten(100), kosten(50), kosten(0)
	t.Logf("Aufwand 100 %%: %.0f, 50 %%: %.0f, 0 %%: %.0f", a, h, n)
	if h < a {
		t.Errorf("Lupe mit 50 %% kostet %.0f, mit 100 %% %.0f", h, a)
	}
}

// QS-V1b: dasselbe bei Kantenfaktor 2 (Bild 2400×2300): Spalt der Naht, Streifen, Zuschnitt.
func TestQSLupeZeigtNurAnsichtFaktor2(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	const w, h = 2400, 2300
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x>>8 | (y>>8)<<4), 255})
		}
	}
	for it := 0; it < 6; it++ {
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		crop := image.Rect(rnd.Intn(40), rnd.Intn(40), w-rnd.Intn(40), h-rnd.Intn(40))
		sx, sy := 900+rnd.Intn(300), 900+rnd.Intn(300)
		st := []document.Streifen{{Von: sx, Bis: sx + 1 + rnd.Intn(60)}, {Zeilen: true, Von: sy, Bis: sy + 1 + rnd.Intn(60)}}
		if err := d.SetzeZuschnitt(crop); err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeKante(document.Kante(it % 3)); err != nil {
			t.Fatal(err)
		}
		if d.KantenFaktor() != 2 {
			t.Fatalf("Kantenfaktor %d", d.KantenFaktor())
		}
		ohne, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		// Quelle quer über beide Nähte, mit Bruchteilen; Ziel groß daneben
		qx, qy := float64(sx)-150+rnd.Float64()*100, float64(sy)-150+rnd.Float64()*100
		l := document.Lupe{Quelle: document.R(document.Pt(qx, qy), document.Pt(qx+330.3, qy+310.7)),
			Ziel: document.R(document.Pt(40.5, 60.25), document.Pt(40.5+700, 60.25+650))}
		n, teil := 0, 0
		in := LupenInhalt(d, l)
		for y := in.Rect.Min.Y; y < in.Rect.Max.Y; y++ {
			for x := in.Rect.Min.X; x < in.Rect.Max.X; x++ {
				c := in.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				n++
				ox, oy := int(c.R)|int(c.B&15)<<8, int(c.G)|int(c.B>>4)<<8
				if !image.Pt(ox, oy).In(crop) || ox >= st[0].Von && ox < st[0].Bis || oy >= st[1].Von && oy < st[1].Bis {
					t.Fatalf("Fall %d: LECK Original %d,%d", it, ox, oy)
				}
				vx, vy := ox-crop.Min.X, oy-crop.Min.Y
				if ox >= st[0].Bis {
					vx -= st[0].Bis - st[0].Von
				}
				if oy >= st[1].Bis {
					vy -= st[1].Bis - st[1].Von
				}
				switch a := ohne.NRGBAAt(vx, vy).A; {
				case a == 0:
					t.Fatalf("Fall %d: Lupe zeigt Original %d,%d (Ansicht %d,%d), das der Spalt ganz radiert", it, ox, oy, vx, vy)
				case a != 255:
					teil++
				}
			}
		}
		t.Logf("Fall %d: %d Pixel geprüft, %d von teilweise radierten", it, n, teil)
	}
}
