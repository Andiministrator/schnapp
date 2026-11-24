// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Lupe im Editor (ADR-0033): Zeichnen auf Leinwand und in der Vorschau der Ebenenliste, Abschnitt „Vergrößerung“ der
// rechten Leiste. Das Bild im Ziel rechnet render.LupenInhalt ohne Gio (dieselben Pixel wie der Renderer); Gio zeichnet
// es mit dem Filter „nächster Nachbar“.

import (
	"fmt"
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// auswahlRechtecke sind die Rechtecke, um die die Auswahl einen Rahmen zeichnet (Original): bei den meisten Objekten ihre
// Grenzen, bei einer Lupe je ein Rahmen um Quelle und um Ziel statt einer Hülle über beide, die bei weit auseinander
// liegenden Rechtecken einen großen Rahmen über leere Fläche zeigte (F-606, Karte #231). Jedes Rechteck mit dem halben
// Strich, wie Lupe.Grenzen.
func auswahlRechtecke(o document.Objekt) []document.Rechteck {
	if t, ok := o.(document.Text); ok { // Sprechblase: Schild samt Spitze (ADR-0021 Nachtrag 13 Punkt 4)
		return []document.Rechteck{t.Huelle()}
	}
	l, ok := o.(document.Lupe)
	if !ok {
		return []document.Rechteck{o.Grenzen()}
	}
	h := l.Strich.Breite / 2
	weit := func(r document.Rechteck) document.Rechteck {
		return document.R(r.Min.Minus(document.Pt(h, h)), r.Max.Plus(document.Pt(h, h)))
	}
	return []document.Rechteck{weit(l.Quelle), weit(l.Ziel)}
}

// lupenVorschauSeite ist die größte Seite des Bilds einer Lupe in der Vorschau der Ebenenliste (Pixel).
const lupenVorschauSeite = 96

// lupenBild ist ein gemerktes Bild im Ziel einer Lupe: Neu gerechnet wird nur, wenn sich die Lupe, der Verlaufsstand
// (Schwärzen, Zuschnitt, Streifen, Rückgängig) oder das Ansichtsbild der Sitzung geändert hat — sonst lüde Gio das Bild
// in jedem Bild neu hoch.
type lupenBild struct {
	l         document.Lupe
	stand     uint64
	bild      uint64
	op        paint.ImageOp
	rahmen    image.Rectangle // Lage in Ansichtskoordinaten (Leinwand) bzw. Größe ab 0 (Vorschau)
	gueltig   bool
	vorhanden bool
}

type lupenSchluessel struct {
	id       int
	vorschau bool
}

// lupenLeiste ist der Abschnitt „Vergrößerung“ der rechten Leiste: − und + über canvas.LupenStufen.
type lupenLeiste struct {
	knoepfe [2]widget.Clickable
	rahmen  [2]image.Rectangle
	text    string // angezeigte Vergrößerung beim letzten Zeichnen (Tests)
	bilder  map[lupenSchluessel]lupenBild
}

// alle sind die Knöpfe des Abschnitts (Tastaturfokus).
func (k *lupenLeiste) alle() []*widget.Clickable {
	return []*widget.Clickable{&k.knoepfe[0], &k.knoepfe[1]}
}

// lupenInhalt liefert das gemerkte Bild im Ziel von l (Leinwand: Ansichtskoordinaten; Vorschau: ohne Ansicht, ab 0).
func (e *editor) lupenInhalt(l document.Lupe, vorschau bool) (paint.ImageOp, image.Rectangle, bool) {
	k := &e.lupe
	if k.bilder == nil {
		k.bilder = map[lupenSchluessel]lupenBild{}
	}
	s := lupenSchluessel{l.ID, vorschau}
	stand, bild := e.s.D.Stand(), e.s.BildAenderung()
	if b, ok := k.bilder[s]; ok && b.gueltig && b.l == l && b.stand == stand && b.bild == bild {
		return b.op, b.rahmen, b.vorhanden
	}
	var img *image.NRGBA
	if vorschau {
		img = render.LupenVorschau(e.s.D, l, lupenVorschauSeite)
	} else {
		img = render.LupenInhalt(e.s.D, l)
	}
	b := lupenBild{l: l, stand: stand, bild: bild, gueltig: true}
	if img != nil {
		b.op, b.rahmen, b.vorhanden = paint.NewImageOp(img), img.Rect, true
		b.op.Filter = paint.FilterNearest // Pixel bleiben scharfe Quadrate (ADR-0033 Punkt 3)
	}
	k.bilder[s] = b
	return b.op, b.rahmen, b.vorhanden
}

// lupenAufraeumen verwirft nach jedem Bild die gemerkten Bilder von Lupen, die es nicht mehr gibt (gelöscht, Anlegen
// rückgängig gemacht; ADR-0033 Nachtrag 2 Punkt 5): Sie hielten sonst Bilder bis zur Größe der Ansicht bis zum Ende des
// Editors, auch solche von vor einer späteren Schwärzung.
func (e *editor) lupenAufraeumen() {
	for k := range e.lupe.bilder {
		if o, ok := e.s.D.Objekt(k.id); !ok {
			delete(e.lupe.bilder, k)
		} else if _, ist := o.(document.Lupe); !ist {
			delete(e.lupe.bilder, k)
		}
	}
}

// zeichneLupe zeichnet l wie der Renderer: Linie, Rahmen um die Quelle, Bild im Ziel, Rahmen um das Ziel. an bildet
// Originalpunkte in die Zeichenfläche ab; in der Vorschau ist das das Original selbst, das Bild wird dann aufs Ziel
// gestreckt.
func (e *editor) zeichneLupe(gtx layout.Context, l document.Lupe, an func(document.Punkt) f32.Point, vorschau bool) {
	strich := l.Strich.Breite > 0 && l.Strich.Farbe.A > 0
	if strich {
		if a, b, ok := l.Linie(); ok {
			var p clip.Path
			p.Begin(gtx.Ops)
			p.MoveTo(an(a))
			p.LineTo(an(b))
			paint.FillShape(gtx.Ops, l.Strich.Farbe, clip.Stroke{Path: p.End(), Width: float32(l.Strich.Breite)}.Op())
		}
		e.lupenRahmen(gtx, l.Quelle, l.Strich, an)
	}
	if bildOp, r, ok := e.lupenInhalt(l, vorschau); ok {
		var t op.TransformStack
		if vorschau {
			a, b := an(l.Ziel.Min), an(l.Ziel.Max)
			t = op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt((b.X-a.X)/float32(r.Dx()), (b.Y-a.Y)/float32(r.Dy()))).
				Offset(a)).Push(gtx.Ops)
		} else {
			t = op.Offset(r.Min).Push(gtx.Ops)
		}
		c := clip.Rect(image.Rectangle{Max: r.Size()}).Push(gtx.Ops)
		bildOp.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		c.Pop()
		t.Pop()
	}
	if strich {
		e.lupenRahmen(gtx, l.Ziel, l.Strich, an)
	}
}

// lupenRahmen zeichnet den eckigen Rahmen um r mittig auf der Kante (document.EckigerRahmen, wie der Renderer).
func (e *editor) lupenRahmen(gtx layout.Context, r document.Rechteck, s document.Strich, an func(document.Punkt) f32.Point) {
	a, b := an(r.Min), an(r.Max)
	q := document.R(document.Pt(float64(a.X), float64(a.Y)), document.Pt(float64(b.X), float64(b.Y)))
	var p clip.Path
	p.Begin(gtx.Ops)
	for _, k := range document.EckigerRahmen(q, s.Breite) {
		p.MoveTo(f32.Pt(float32(k.Min.X), float32(k.Min.Y)))
		p.LineTo(f32.Pt(float32(k.Max.X), float32(k.Min.Y)))
		p.LineTo(f32.Pt(float32(k.Max.X), float32(k.Max.Y)))
		p.LineTo(f32.Pt(float32(k.Min.X), float32(k.Max.Y)))
		p.Close()
	}
	paint.FillShape(gtx.Ops, s.Farbe, clip.Outline{Path: p.End()}.Op())
}

// vergroesserungText ist die angezeigte Vergrößerung, auf Zehntel gerundet („2×“, „2.5×“ bzw. „2,5×“).
func vergroesserungText(k float64) string {
	z := int(math.Round(k * 10))
	if z%10 == 0 {
		return fmt.Sprintf(i18n.TK("status bar", "%d×"), z/10)
	}
	return fmt.Sprintf(i18n.TK("status bar", "%d.%d×"), z/10, z%10)
}

// lupeAuswerten verarbeitet − und + der Vergrößerung; während des Speicherns ruht das Bearbeiten.
func (e *editor) lupeAuswerten(gtx layout.Context) {
	for j, dir := range []int{-1, 1} {
		if e.lupe.knoepfe[j].Clicked(gtx) && e.seiteAn() {
			if k, ok := e.s.VergroesserungStufe(dir); ok {
				e.s.SetzeVergroesserung(k)
			}
		}
	}
}

// zeichneVergroesserung zeichnet die Zeile „Vergrößerung − 2× +“ bei y (wie die Strichstärke) und liefert das neue y.
func (e *editor) zeichneVergroesserung(gtx layout.Context, x, y, spalte int) int {
	k := &e.lupe
	hoch, kb := gtx.Dp(26), gtx.Dp(30)
	e.seiteText(gtx, i18n.T("Magnification"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	wert := image.Rect(x+spalte+kb, y, x+spalte+kb+gtx.Dp(64), y+hoch)
	for j, dir := range []int{-1, 1} {
		r := image.Rect(x+spalte, y, x+spalte+kb, y+hoch)
		zeichen := "−"
		if dir > 0 {
			r = image.Rect(wert.Max.X, y, wert.Max.X+kb, y+hoch)
			zeichen = "+"
		}
		k.rahmen[j] = r
		_, an := e.s.VergroesserungStufe(dir)
		rnd, schrift := farbeLinie, farbeText
		switch {
		case !an:
			schrift = farbeNebentext
		case k.knoepfe[j].Hovered():
			rnd = farbeNebentext
		}
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
		e.textMittig(gtx, zeichen, 14, schrift, r)
		seiteKnopf(gtx, &k.knoepfe[j], r) // auch grau angemeldet, wie die Strichstärke (Sitzungs-QA A-5)
	}
	v, _ := e.s.Vergroesserung()
	k.text = vergroesserungText(v)
	e.textMittig(gtx, k.text, 13, farbeText, wert)
	return y + hoch + gtx.Dp(10)
}
