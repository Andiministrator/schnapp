// SPDX-License-Identifier: GPL-3.0-or-later

package oberflaeche

import (
	"errors"
	"image"
	"image/color"
	"os"
	"sync"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/auswahl"
	"github.com/Andiministrator/schnapp/sprache"
)

// Starte führt f neben der Gio-Hauptschleife aus und beendet das Programm mit dem Rückgabewert von f. Gio verlangt
// app.Main auf dem Haupt-Goroutine; jedes Programm, das ein Fenster öffnen kann, ruft Starte aus main auf.
func Starte(f func() int) {
	go func() { os.Exit(f()) }()
	app.Main()
}

// MonitorBild ist das eingefrorene Bild eines Monitors für das Overlay.
type MonitorBild struct {
	Name string      // Wayland-Ausgabe, auf der das Vollbild liegt
	Bild image.Image // in echten Pixeln des Monitors
}

// OverlayErgebnis: welcher Monitor, welcher Bereich (in Bildpixeln dieses Monitors).
type OverlayErgebnis struct {
	Monitor int
	Bereich image.Rectangle
}

// ErrAbgebrochen: Der Nutzer hat das Overlay mit Esc verlassen.
var ErrAbgebrochen = errors.New("overlay: abgebrochen")

// Farben laut marke/corporate-design.md.
var (
	farbeAbdunkeln = color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0x99} // grau-950, durchscheinend
	farbeLeiste    = color.NRGBA{R: 0x2A, G: 0x2E, B: 0x33, A: 0xF0} // grau-800
	farbeLinie     = color.NRGBA{R: 0x3A, G: 0x3F, B: 0x45, A: 0xFF} // grau-700
	farbeText      = color.NRGBA{R: 0xEC, G: 0xEA, B: 0xE5, A: 0xFF} // grau-100
	farbeAuswahl   = color.NRGBA{R: 0x4F, G: 0x94, B: 0x79, A: 0xFF} // gruen-500
	farbeAkzent    = color.NRGBA{R: 0xF5, G: 0xA6, B: 0x5B, A: 0xFF} // orange-500
	farbeGrund     = color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0xFF} // grau-950
)

// Overlay zeigt je Monitor sein eingefrorenes Bild im randlosen Vollbild und lässt einen Bereich wählen (im
// Monitor-Modus auch einen ganzen Monitor per Klick). Es kehrt zurück, sobald gewählt oder mit Esc abgebrochen ist.
// Die Fenster schließt es nicht: Window.Perform(ActionClose) aus einer fremden Goroutine läuft in Gio v0.10.3 in
// einen Datenwettlauf und wartet danach ewig auf das Ende der Fenster (erster echter Lauf 19.11.2026, F-77). Die
// Fenster zeigen weiter das eingefrorene Bild und verschwinden mit dem Programmende. Muss neben app.Main laufen
// (Starte); ein zweites Overlay im selben Prozess ist so nicht vorgesehen.
func Overlay(bilder []MonitorBild, m auswahl.Modus) (OverlayErgebnis, error) {
	if len(bilder) == 0 {
		return OverlayErgebnis{}, errors.New("overlay: keine Bilder")
	}
	g := &gruppe{fertig: make(chan ergebnis, 1)}
	for i, b := range bilder {
		// Jedes Fenster läuft in einer eigenen Goroutine; text.Shaper ist nicht nebenläufig nutzbar, darum je
		// Fenster ein eigenes Thema (sonst Absturz im Text-Setzer, erster echter Lauf 19.11.2026).
		th := material.NewTheme()
		th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
		g.fenster = append(g.fenster, &overlayFenster{
			gruppe: g, index: i, ausgabe: b.Name, thema: th, modus: m,
			w:      new(app.Window),
			bildOp: paint.NewImageOp(b.Bild),
			z:      auswahl.Neu(m, b.Bild.Bounds().Size()),
		})
	}
	for _, f := range g.fenster {
		go func() {
			if err := f.laufen(); err != nil {
				g.melde(ergebnis{err: err})
			}
		}()
	}
	e := <-g.fertig
	return e.OverlayErgebnis, e.err
}

type ergebnis struct {
	OverlayErgebnis
	err error
}

type gruppe struct {
	fenster []*overlayFenster
	fertig  chan ergebnis
	einmal  sync.Once
}

// melde nimmt nur das erste Ergebnis an (zwei Fenster können fast gleichzeitig fertig werden).
func (g *gruppe) melde(e ergebnis) { g.einmal.Do(func() { g.fertig <- e }) }

type overlayFenster struct {
	gruppe  *gruppe
	index   int
	ausgabe string
	thema   *material.Theme
	modus   auswahl.Modus
	w       *app.Window
	bildOp  paint.ImageOp
	z       *auswahl.Zustand
	groesse image.Point
}

func (f *overlayFenster) laufen() error {
	f.w.Option(app.Title("Schnapp"), app.FullscreenOn(f.ausgabe), app.Fullscreen.Option())
	var ops op.Ops
	for {
		switch e := f.w.Event().(type) {
		case app.DestroyEvent:
			if e.Err != nil {
				return e.Err
			}
			// Vom Compositor oder Nutzer geschlossen, ohne dass gewählt wurde: wie Esc.
			f.gruppe.melde(ergebnis{err: ErrAbgebrochen})
			return nil
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			f.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (f *overlayFenster) frame(gtx layout.Context) {
	if gtx.Constraints.Max != f.groesse {
		f.groesse = gtx.Constraints.Max
		f.z.SetzeAbbildung(f.groesse) // der erste Frame kommt oft in falscher Skala (gio-wayland.md)
	}
	f.tasten(gtx)
	f.zeiger(gtx)

	paint.Fill(gtx.Ops, farbeGrund)
	f.zeichneBild(gtx)
	f.zeichneAuswahl(gtx)
	if p, ok := f.z.Zeiger(); ok {
		f.zeichneLupe(gtx, p)
	}
	f.zeichneHinweis(gtx)

	bereich := clip.Rect(image.Rectangle{Max: f.groesse}).Push(gtx.Ops)
	pointer.CursorCrosshair.Add(gtx.Ops)
	event.Op(gtx.Ops, f)
	bereich.Pop()
}

func (f *overlayFenster) tasten(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: key.NameReturn},
			key.Filter{Name: key.NameEnter},
		)
		if !ok {
			return
		}
		k, ok := ev.(key.Event)
		if !ok || k.State != key.Press {
			continue
		}
		switch k.Name {
		case key.NameEscape:
			f.gruppe.melde(ergebnis{err: ErrAbgebrochen})
		case key.NameReturn, key.NameEnter:
			f.fertig(f.z.Ganz())
		}
	}
}

func (f *overlayFenster) zeiger(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: f,
			Kinds:  pointer.Move | pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Enter | pointer.Leave,
		})
		if !ok {
			return
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		p := auswahl.Punkt{X: e.Position.X, Y: e.Position.Y}
		switch e.Kind {
		case pointer.Move, pointer.Drag, pointer.Enter:
			f.z.Bewegt(p)
		case pointer.Leave:
			f.z.Verlassen()
		case pointer.Press:
			if e.Buttons == pointer.ButtonPrimary {
				f.z.Gedrueckt(p)
			}
		case pointer.Release:
			f.fertig(f.z.Losgelassen(p))
		case pointer.Cancel:
			f.z.Abgebrochen()
		}
	}
}

func (f *overlayFenster) fertig(e auswahl.Ergebnis) {
	if e.Fertig {
		f.gruppe.melde(ergebnis{OverlayErgebnis: OverlayErgebnis{Monitor: f.index, Bereich: e.Bereich}})
	}
}

func (f *overlayFenster) zeichneBild(gtx layout.Context) {
	a := f.z.Abb
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(a.Zoom, a.Zoom)).Offset(f32.Pt(a.Versatz.X, a.Versatz.Y))).Push(gtx.Ops)
	b := f.bildOp
	b.Filter = paint.FilterNearest // pixelgenau, auch wenn die erste Abbildung vergrößert
	c := clip.Rect(image.Rectangle{Max: f.z.Groesse}).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	c.Pop()
	t.Pop()
}

func (f *overlayFenster) zeichneAuswahl(gtx layout.Context) {
	ganz := image.Rectangle{Max: f.groesse}
	if f.z.Hervorgehoben() {
		rahmen(gtx.Ops, ganz.Inset(gtx.Dp(2)), gtx.Dp(4), farbeAuswahl)
		return
	}
	r, ok := f.z.Rahmen()
	if !ok {
		paint.FillShape(gtx.Ops, farbeAbdunkeln, clip.Rect(ganz).Op())
		return
	}
	fr := f.z.Abb.RechteckImFenster(r)
	for _, s := range []image.Rectangle{
		image.Rect(0, 0, ganz.Max.X, fr.Min.Y), image.Rect(0, fr.Max.Y, ganz.Max.X, ganz.Max.Y),
		image.Rect(0, fr.Min.Y, fr.Min.X, fr.Max.Y), image.Rect(fr.Max.X, fr.Min.Y, ganz.Max.X, fr.Max.Y),
	} {
		paint.FillShape(gtx.Ops, farbeAbdunkeln, clip.Rect(s).Op())
	}
	rahmen(gtx.Ops, fr.Inset(-1), 1, farbeAuswahl)
	ecken(gtx.Ops, fr, gtx.Dp(14), gtx.Dp(3), farbeAkzent)

	call, groesse := f.schild(gtx, auswahl.MassText(r), true)
	ecke := auswahl.MassPlatz(fr, groesse, f.groesse)
	t := op.Offset(ecke).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// zeichneLupe: vergrößerter Ausschnitt mit Pixelraster und markiertem Mittelpixel, darunter die Position.
func (f *overlayFenster) zeichneLupe(gtx layout.Context, p auswahl.Punkt) {
	l := f.z.LupeAn(p, f.groesse)
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(l.Ziel).Op())
	c := clip.Rect(l.Ziel).Push(gtx.Ops)
	k := float32(l.Faktor)
	ab := f32.Pt(float32(l.Ziel.Min.X)-float32(l.Quelle.Min.X)*k, float32(l.Ziel.Min.Y)-float32(l.Quelle.Min.Y)*k)
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k)).Offset(ab)).Push(gtx.Ops)
	b := f.bildOp
	b.Filter = paint.FilterNearest
	ic := clip.Rect(image.Rectangle{Max: f.z.Groesse}).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	ic.Pop()
	t.Pop()
	raster := color.NRGBA{R: 0x3A, G: 0x3F, B: 0x45, A: 0x80}
	for i := 1; i < auswahl.LupePixel; i++ {
		x := l.Ziel.Min.X + i*l.Faktor
		y := l.Ziel.Min.Y + i*l.Faktor
		paint.FillShape(gtx.Ops, raster, clip.Rect(image.Rect(x, l.Ziel.Min.Y, x+1, l.Ziel.Max.Y)).Op())
		paint.FillShape(gtx.Ops, raster, clip.Rect(image.Rect(l.Ziel.Min.X, y, l.Ziel.Max.X, y+1)).Op())
	}
	h := auswahl.LupePixel / 2 * l.Faktor
	mitte := image.Rect(l.Ziel.Min.X+h, l.Ziel.Min.Y+h, l.Ziel.Min.X+h+l.Faktor, l.Ziel.Min.Y+h+l.Faktor)
	rahmen(gtx.Ops, mitte.Inset(-1), 2, farbeAkzent)
	c.Pop()
	rahmen(gtx.Ops, l.Ziel.Inset(-1), 1, farbeLinie)

	call, groesse := f.schild(gtx, auswahl.PositionText(l.Mitte), true)
	ecke := image.Pt(l.Ziel.Min.X, l.Ziel.Max.Y+gtx.Dp(4))
	if ecke.Y+groesse.Y > f.groesse.Y {
		ecke.Y = l.Ziel.Min.Y - gtx.Dp(4) - groesse.Y
	}
	t2 := op.Offset(ecke).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t2.Pop()
}

// zeichneHinweis: kurze Bedienhilfe oben in der Mitte.
func (f *overlayFenster) zeichneHinweis(gtx layout.Context) {
	var s string
	if f.modus == auswahl.ModusMonitor {
		s = sprache.T("Click a monitor or drag an area · Enter: this monitor · Esc: cancel")
	} else {
		s = sprache.T("Drag an area · Enter: whole monitor · Esc: cancel")
	}
	call, groesse := f.schild(gtx, s, false)
	t := op.Offset(image.Pt((f.groesse.X-groesse.X)/2, gtx.Dp(16))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// schild zeichnet Text auf einer abgerundeten Leiste und liefert die Aufzeichnung samt Größe (zum Platzieren).
func (f *overlayFenster) schild(gtx layout.Context, s string, mono bool) (op.CallOp, image.Point) {
	m := op.Record(gtx.Ops)
	lbl := material.Label(f.thema, unit.Sp(14), s)
	lbl.Color = farbeText
	if mono {
		lbl.Font = font.Font{Typeface: "Go Mono"}
	}
	innen := image.Pt(gtx.Dp(8), gtx.Dp(4))
	g := gtx
	g.Constraints.Min = image.Point{}
	textMakro := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	text := textMakro.Stop()
	groesse := dims.Size.Add(innen.Mul(2))
	paint.FillShape(gtx.Ops, farbeLeiste, clip.UniformRRect(image.Rectangle{Max: groesse}, gtx.Dp(6)).Op(gtx.Ops))
	t := op.Offset(innen).Push(gtx.Ops)
	text.Add(gtx.Ops)
	t.Pop()
	return m.Stop(), groesse
}

// rahmen zeichnet einen Rahmen der Breite b innen an r.
func rahmen(ops *op.Ops, r image.Rectangle, b int, c color.NRGBA) {
	for _, s := range []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+b), image.Rect(r.Min.X, r.Max.Y-b, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, r.Min.Y+b, r.Min.X+b, r.Max.Y-b), image.Rect(r.Max.X-b, r.Min.Y+b, r.Max.X, r.Max.Y-b),
	} {
		paint.FillShape(ops, c, clip.Rect(s).Op())
	}
}

// ecken zeichnet die vier Auswahlecken (Marke: Motiv für „Bereich“) außen an r.
func ecken(ops *op.Ops, r image.Rectangle, laenge, b int, c color.NRGBA) {
	l := min(laenge, r.Dx()/2+b, r.Dy()/2+b)
	o := r.Inset(-b)
	for _, s := range []image.Rectangle{
		image.Rect(o.Min.X, o.Min.Y, o.Min.X+l, o.Min.Y+b), image.Rect(o.Min.X, o.Min.Y, o.Min.X+b, o.Min.Y+l),
		image.Rect(o.Max.X-l, o.Min.Y, o.Max.X, o.Min.Y+b), image.Rect(o.Max.X-b, o.Min.Y, o.Max.X, o.Min.Y+l),
		image.Rect(o.Min.X, o.Max.Y-b, o.Min.X+l, o.Max.Y), image.Rect(o.Min.X, o.Max.Y-l, o.Min.X+b, o.Max.Y),
		image.Rect(o.Max.X-l, o.Max.Y-b, o.Max.X, o.Max.Y), image.Rect(o.Max.X-b, o.Max.Y-l, o.Max.X, o.Max.Y),
	} {
		paint.FillShape(ops, c, clip.Rect(s).Op())
	}
}
