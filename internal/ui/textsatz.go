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
	if len(ts.faces) > 256 {
		clear(ts.faces)
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
func (ts *textsatz) zeichneZeile(gtx layout.Context, s *fonts.Schrift, text string, x, y, grad float64, c color.NRGBA,
	sicht image.Rectangle) {
	oben, unten, seitlich := s.Huelle() // wie im Renderer (F-576)
	if text == "" || y-grad*oben > float64(sicht.Max.Y) || y+grad*unten < float64(sicht.Min.Y) {
		return
	}
	face := ts.face(s, grad)
	if face == nil {
		return
	}
	ppem := fixed.Int26_6(math.Round(grad * 64))
	links, rechts := float64(sicht.Min.X)-seitlich*grad, float64(sicht.Max.X)+seitlich*grad
	dot := fixed.Int26_6(math.Round(x * 64))
	gy := float32(fixed.Int26_6(math.Round(y*64))) / 64
	var p clip.Path
	p.Begin(gtx.Ops)
	leer := true
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
				}
			}
		}
		dot += adv
	}
	spec := p.End()
	if !leer {
		paint.FillShape(gtx.Ops, c, clip.Outline{Path: spec}.Op())
	}
}
