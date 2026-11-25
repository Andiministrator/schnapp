// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Text der Beschriftungen im Editor (ADR-0021 Nachtrag 10 Punkt 5): aus den Glyphenumrissen derselben Schriftdatei, mit
// denselben Orten wie der Renderer (Vorschub aus opentype.Face, Paarunterschneidung im Grad über document.KernPaar, ohne
// Hinting), statt über den Shaper von Gio, der anders setzt (eigenes Shaping) und so andere Breiten ergäbe als Messung,
// Renderer und Export.

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// umrissGrenze ist die Zahl der gemerkten Glyphenumrisse; darüber wird der Speicher geleert.
const umrissGrenze = 20000

// facesGrenze ist die Zahl der gemerkten Faces (Schrift und Grad); darüber werden Faces und Umrisse geleert.
const facesGrenze = 2 * fonts.MaxGeladen

type faceSchluessel struct {
	f    *sfnt.Font
	grad float64
}

type umrissSchluessel struct {
	f    *sfnt.Font
	gi   sfnt.GlyphIndex
	ppem fixed.Int26_6
}

// textsatz hält Faces und Umrisse über die Bilder hinweg (nur aus der Gio-Schleife benutzt, nicht nebenläufig).
type textsatz struct {
	faces   map[faceSchluessel]font.Face
	umrisse map[umrissSchluessel][]sfnt.Segment
	puffer  sfnt.Buffer
}

func (ts *textsatz) face(s *fonts.Schrift, grad float64) font.Face {
	if ts.faces == nil {
		ts.faces = map[faceSchluessel]font.Face{}
	}
	k := faceSchluessel{s.Font, grad}
	if f, ok := ts.faces[k]; ok {
		return f
	}
	f, err := opentype.NewFace(s.Font, &opentype.FaceOptions{Size: grad, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	// Über facesGrenze Faces fangen Faces und Umrisse leer neu an: Beide halten ihr *sfnt.Font samt Dateibytes fest, und
	// seit der Verdrängung (F-697) kommt dieselbe Datei womöglich als neues *sfnt.Font wieder (Kritik B-1).
	if len(ts.faces) >= facesGrenze {
		clear(ts.faces)
		clear(ts.umrisse)
	}
	ts.faces[k] = f
	return f
}

func (ts *textsatz) umriss(f *sfnt.Font, gi sfnt.GlyphIndex, ppem fixed.Int26_6) []sfnt.Segment {
	if ts.umrisse == nil {
		ts.umrisse = map[umrissSchluessel][]sfnt.Segment{}
	}
	k := umrissSchluessel{f, gi, ppem}
	if u, ok := ts.umrisse[k]; ok {
		return u
	}
	seg, err := f.LoadGlyph(&ts.puffer, gi, ppem, nil)
	if err != nil {
		seg = nil
	}
	u := append([]sfnt.Segment(nil), seg...) // LoadGlyph liefert den Puffer, der beim nächsten Aufruf überschrieben wird
	if len(ts.umrisse) >= umrissGrenze {
		clear(ts.umrisse)
	}
	ts.umrisse[k] = u
	return u
}

// zeichneZeile zeichnet text in der Schrift s im Grad grad mit dem Anfang x auf der Grundlinie y (Ansichtspixel, unter
// der Zoom-Abbildung der Leinwand). sicht ist der sichtbare Bereich (Ansicht): Glyphen, deren Ursprung mehr als die Hülle der Schrift (mindestens
// 1,25 Schriftgrade, fonts.Schrift.Huelle) daneben liegt, entfallen wie im Renderer; Vorschub und Unterschneidung laufen weiter.
// Das Ergebnis ist die Zahl der gezeichneten Glyphenumrisse (für Tests ohne Fenster, F-590).
func (ts *textsatz) zeichneZeile(gtx layout.Context, s *fonts.Schrift, text string, x, y, grad float64, c color.NRGBA,
	sicht image.Rectangle) int {
	n, _ := ts.setzeZeile(gtx, s, text, x, y, grad, c, sicht, 0)
	return n
}

// zeilenLauf ist ein Lauf aufeinanderfolgender Glyphen einer Zeile, einzeln aufgezeichnet, mit der Lage seiner Tinte
// (von, bis) entlang der Zeile: Ursprung der ersten Glyphe minus, der letzten plus die seitliche Hülle der Schrift.
type zeilenLauf struct {
	aufruf   op.CallOp
	von, bis float64
}

// zeileInLaeufen zeichnet die Zeile wie zeichneZeile, aber in Läufen, deren Glyphenursprünge höchstens vier seitliche
// Hüllen auseinanderliegen, jeden für sich aufgezeichnet (F-765, ADR-0023 Nachtrag 18): Der Editor spielt einen Lauf
// nur in den Spalten ab, die seine Tinte berührt. Vorher spielte er die ganze Zeile in jeder Spalte rechts ihres Anfangs
// ab, über 1023 Spaltenstreifen auch eine Zeile „x“ tausendfach und jede lange Zeile mit allen Glyphen je Spalte. Das
// Ergebnis sind die Läufe mit mindestens einer gezeichneten Glyphe und deren Zahl.
func (ts *textsatz) zeileInLaeufen(gtx layout.Context, s *fonts.Schrift, text string, x, y, grad float64, c color.NRGBA,
	sicht image.Rectangle) ([]zeilenLauf, int) {
	_, _, seitlich := s.Huelle()
	n, laeufe := ts.setzeZeile(gtx, s, text, x, y, grad, c, sicht, 4*seitlich*grad)
	return laeufe, n
}

// setzeZeile ist zeichneZeile und zeileInLaeufen: Ohne lauf (0) eine Füllung für die ganze Zeile in gtx, sonst je Lauf
// von höchstens lauf Breite (Glyphenursprünge) eine aufgezeichnete Füllung.
func (ts *textsatz) setzeZeile(gtx layout.Context, s *fonts.Schrift, text string, x, y, grad float64, c color.NRGBA,
	sicht image.Rectangle, lauf float64) (int, []zeilenLauf) {
	oben, unten, seitlich := s.Huelle() // wie im Renderer (F-576)
	if text == "" || y-grad*oben > float64(sicht.Max.Y) || y+grad*unten < float64(sicht.Min.Y) {
		return 0, nil
	}
	face := ts.face(s, grad)
	if face == nil {
		return 0, nil
	}
	var (
		laeufe     []zeilenLauf
		aufnahme   op.MacroOp
		anfang, zu float64 // Ursprung der ersten und der letzten Glyphe des laufenden Laufs
	)
	if lauf > 0 {
		aufnahme = op.Record(gtx.Ops)
	}
	ppem := fixed.Int26_6(math.Round(grad * 64))
	links, rechts := float64(sicht.Min.X)-seitlich*grad, float64(sicht.Max.X)+seitlich*grad
	dot := fixed.Int26_6(math.Round(x * 64))
	gy := float32(fixed.Int26_6(math.Round(y*64))) / 64
	var p clip.Path
	p.Begin(gtx.Ops)
	leer := true
	gezeichnet := 0
	vorher := rune(-1)
	for _, r := range text {
		if vorher >= 0 {
			dot += document.KernPaar(s.Font, &ts.puffer, vorher, r, grad) // wie Renderer und Messung
		}
		vorher = r
		px := float64(dot) / 64
		if px > rechts {
			break
		}
		adv, _ := face.GlyphAdvance(r)
		if px >= links {
			gi, err := s.Font.GlyphIndex(&ts.puffer, r)
			if err == nil {
				if lauf > 0 && !leer && px-anfang > lauf {
					// Lauf voll: abschließen und einen neuen beginnen.
					paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
					laeufe = append(laeufe, zeilenLauf{aufnahme.Stop(), anfang - seitlich*grad, zu + seitlich*grad})
					aufnahme = op.Record(gtx.Ops)
					p.Begin(gtx.Ops)
					leer = true
				}
				warLeer := leer
				o := f32.Pt(float32(dot)/64, gy)
				pt := func(a fixed.Point26_6) f32.Point { return o.Add(f32.Pt(float32(a.X)/64, float32(a.Y)/64)) }
				offen := false
				for _, seg := range ts.umriss(s.Font, gi, ppem) {
					switch seg.Op {
					case sfnt.SegmentOpMoveTo:
						if offen {
							p.Close()
						}
						p.MoveTo(pt(seg.Args[0]))
						offen, leer = true, false
					case sfnt.SegmentOpLineTo:
						p.LineTo(pt(seg.Args[0]))
					case sfnt.SegmentOpQuadTo:
						p.QuadTo(pt(seg.Args[0]), pt(seg.Args[1]))
					case sfnt.SegmentOpCubeTo:
						p.CubeTo(pt(seg.Args[0]), pt(seg.Args[1]), pt(seg.Args[2]))
					}
				}
				if offen {
					p.Close()
					gezeichnet++
					if warLeer {
						anfang = px
					}
					zu = px
				}
			}
		}
		dot += adv
	}
	spec := p.End()
	if !leer {
		paint.FillShape(gtx.Ops, c, clip.Outline{Path: spec}.Op())
	}
	if lauf > 0 {
		aufruf := aufnahme.Stop()
		if !leer {
			laeufe = append(laeufe, zeilenLauf{aufruf, anfang - seitlich*grad, zu + seitlich*grad})
		}
	}
	return gezeichnet, laeufe
}
