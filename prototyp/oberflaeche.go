// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"sort"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"
)

type modus int

const (
	modusAuswahl modus = iota // eingefrorenes Bild, Bereich aufziehen
	modusEditor               // Ausschnitt, Pfeile ziehen und biegen
)

// gezogen sagt, woran die Maus gerade zieht.
type gezogen int

const (
	nichts gezogen = iota
	neuerPfeil
	griffK // Biegegriff
	griffA
	griffB
	auswahlRahmen
)

const (
	griffRadius = 9  // Pixel im Fenster
	strichBild  = 6  // Pfeilbreite in Bildpixeln
	trefferAbst = 14 // Fangradius für Griffe in Fensterpixeln
)

var pfeilFarbe = color.NRGBA{R: 0xE5, G: 0x30, B: 0x2A, A: 0xFF}

type ansicht struct {
	fenster *app.Window
	ausgabe string     // Wayland-Ausgabe des Vollbildfensters
	gruppe  []*ansicht // alle Overlay-Fenster (eins je Monitor)
	thema   *material.Theme
	modus   modus

	bild   *image.RGBA
	bildOp paint.ImageOp

	// Abbildung Bild → Fenster: p_fenster = p_bild·zoom + versatz
	zoom    float32
	versatz f32.Point
	groesse image.Point // Fenstergröße in Pixeln beim letzten Frame
	gesetzt bool        // Abbildung für die aktuelle Größe festgelegt

	// Auswahl
	auswahlA, auswahlB f32.Point
	hatAuswahl         bool

	// Editor
	pfeile  []pfeil
	verlauf [][]pfeil // Rückgängig-Stapel: Stand vor jeder Änderung
	aktiv   int       // Index des Pfeils, an dem gezogen wird
	zug     gezogen

	// Messung
	ersterFrame bool
	zugFrames   []time.Time
	zugArbeit   []time.Duration // CPU-Zeit für Aufbau der Ops je Frame
	zugJeFrame  []int           // Zieh-Ereignisse, die ein Frame auf einmal verarbeitet hat
	ereignisse  int             // Zieh-Ereignisse seit dem letzten Frame
	letzteRate  string
	messDauer   time.Duration
	messStart   time.Time
}

func neueAnsicht(b *image.RGBA, m modus, messDauer time.Duration) *ansicht {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	a := &ansicht{fenster: new(app.Window), thema: th, modus: m, messDauer: messDauer, aktiv: -1}
	a.setzeBild(b)
	return a
}

func (a *ansicht) setzeBild(b *image.RGBA) {
	a.bild = b
	a.bildOp = paint.NewImageOp(b)
	a.gesetzt = false
}

func (a *ansicht) laufen() error {
	a.fenster.Option(app.Title("Schnapp-Prototyp"), app.FullscreenOn(a.ausgabe), app.Fullscreen.Option())
	var ops op.Ops
	for {
		switch e := a.fenster.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			arbeit := time.Now()
			gtx := app.NewContext(&ops, e)
			if !a.ersterFrame {
				a.ersterFrame = true
				log.Printf("[%s] Erster Frame: Fenster %dx%d px, PxPerDp %.2f, Bild %dx%d, nach %v (Unix-ns %d)",
					a.ausgabe, e.Size.X, e.Size.Y, e.Metric.PxPerDp, a.bild.Rect.Dx(), a.bild.Rect.Dy(),
					time.Since(programmStart), time.Now().UnixNano())
			}
			a.frame(gtx, e.Now)
			e.Frame(gtx.Ops)
			if a.zug != nichts || a.messDauer > 0 {
				a.zugArbeit = append(a.zugArbeit, time.Since(arbeit))
			}
		}
	}
}

func (a *ansicht) frame(gtx layout.Context, jetzt time.Time) {
	if gtx.Constraints.Max != a.groesse {
		a.groesse = gtx.Constraints.Max
		a.gesetzt = false
	}
	if !a.gesetzt {
		a.abbildungSetzen()
	}
	a.tasten(gtx)
	a.zeiger(gtx)
	if a.messDauer > 0 {
		a.messSchritt(gtx, jetzt)
	} else if a.zug != nichts {
		a.zugFrames = append(a.zugFrames, jetzt)
		a.zugJeFrame = append(a.zugJeFrame, a.ereignisse)
	}
	a.ereignisse = 0

	paint.Fill(gtx.Ops, color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF})
	a.zeichneBild(gtx)
	switch a.modus {
	case modusAuswahl:
		a.zeichneAuswahl(gtx)
		pointer.CursorCrosshair.Add(gtx.Ops)
	case modusEditor:
		a.zeichnePfeile(gtx)
		a.zeichneGriffe(gtx)
	}
	a.zeichneLeiste(gtx)

	// Eingabe für den nächsten Frame anmelden: ganze Fläche.
	bereich := clip.Rect(image.Rectangle{Max: a.groesse}).Push(gtx.Ops)
	event.Op(gtx.Ops, a)
	bereich.Pop()
}

// abbildungSetzen: Auswahl zeigt das Bild 1:1 ab der Ecke (pixelgenau, wenn Bild = Fenster),
// sonst eingepasst; der Editor passt den Ausschnitt mit Rand ein, nie größer als 1:1.
func (a *ansicht) abbildungSetzen() {
	a.gesetzt = true
	if a.groesse.X == 0 || a.groesse.Y == 0 {
		return
	}
	bx, by := float32(a.bild.Rect.Dx()), float32(a.bild.Rect.Dy())
	fx, fy := float32(a.groesse.X), float32(a.groesse.Y)
	rand := float32(0)
	if a.modus == modusEditor {
		rand = 40
	}
	z := float32(math.Min(float64((fx-2*rand)/bx), float64((fy-2*rand)/by)))
	if a.modus == modusEditor && z > 1 {
		z = 1
	}
	if a.modus == modusAuswahl && a.bild.Rect.Dx() == a.groesse.X && a.bild.Rect.Dy() == a.groesse.Y {
		z = 1
	}
	a.zoom = z
	a.versatz = f32.Pt((fx-bx*z)/2, (fy-by*z)/2)
	log.Printf("[%s] Abbildung: Modus %d, Zoom %.4f, Versatz %.1f,%.1f (pixelgenau: %v)",
		a.ausgabe, a.modus, z, a.versatz.X, a.versatz.Y, z == 1 && a.versatz == f32.Point{})
}

func (a *ansicht) zuBild(p f32.Point) f32.Point    { return p.Sub(a.versatz).Mul(1 / a.zoom) }
func (a *ansicht) zuFenster(p f32.Point) f32.Point { return p.Mul(a.zoom).Add(a.versatz) }

func (a *ansicht) tasten(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: key.NameReturn},
			key.Filter{Name: key.NameEnter},
			key.Filter{Name: "Z", Required: key.ModShortcut},
		)
		if !ok {
			return
		}
		k, ok := ev.(key.Event)
		if !ok || k.State != key.Press {
			continue
		}
		log.Printf("Taste %q (Mod %v)", k.Name, k.Modifiers)
		switch {
		case k.Name == key.NameEscape:
			log.Printf("Esc: beendet")
			exitNachLog()
		case (k.Name == key.NameReturn || k.Name == key.NameEnter) && a.modus == modusAuswahl:
			a.inEditor()
		case k.Name == "Z" && a.modus == modusEditor:
			a.rueckgaengig()
		}
	}
}

func (a *ansicht) zeiger(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  a,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -100000, Max: 100000},
		})
		if !ok {
			return
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if e.Buttons != pointer.ButtonPrimary {
				continue
			}
			a.drueck(e.Position)
		case pointer.Drag:
			a.ereignisse++
			a.zieh(e.Position)
		case pointer.Release, pointer.Cancel:
			a.loslassen()
		case pointer.Scroll:
			if a.modus == modusEditor && e.Scroll.Y != 0 {
				a.zoomen(e.Position, e.Scroll.Y)
			}
		}
	}
}

func (a *ansicht) drueck(p f32.Point) {
	a.zugFrames = a.zugFrames[:0]
	a.zugArbeit = a.zugArbeit[:0]
	a.zugJeFrame = a.zugJeFrame[:0]
	if a.modus == modusAuswahl {
		a.auswahlA, a.auswahlB, a.hatAuswahl = p, p, true
		a.zug = auswahlRahmen
		return
	}
	a.merken()
	for i := len(a.pfeile) - 1; i >= 0; i-- {
		pf := a.pfeile[i]
		for _, g := range []struct {
			art gezogen
			pkt f32.Point
		}{{griffK, pf.griff()}, {griffB, pf.B}, {griffA, pf.A}} {
			if laenge(a.zuFenster(g.pkt).Sub(p)) <= trefferAbst {
				a.aktiv, a.zug = i, g.art
				return
			}
		}
	}
	b := a.zuBild(p)
	a.pfeile = append(a.pfeile, gerade(b, b))
	a.aktiv, a.zug = len(a.pfeile)-1, neuerPfeil
}

func (a *ansicht) zieh(p f32.Point) {
	if a.zug == auswahlRahmen {
		a.auswahlB = p
		return
	}
	if a.aktiv < 0 {
		return
	}
	pf := &a.pfeile[a.aktiv]
	b := a.zuBild(p)
	switch a.zug {
	case neuerPfeil:
		*pf = gerade(pf.A, b)
	case griffK:
		pf.setzeGriff(b)
	case griffA, griffB:
		h := pf.griff() // Biegung halten: Griff bleibt, wo er ist
		if a.zug == griffA {
			pf.A = b
		} else {
			pf.B = b
		}
		pf.setzeGriff(h)
	}
}

func (a *ansicht) loslassen() {
	switch a.zug {
	case nichts:
		return
	case auswahlRahmen:
		a.zug = nichts
		r := a.auswahlRechteck()
		if r.Dx() >= 4 && r.Dy() >= 4 {
			a.inEditor()
		}
		return
	case neuerPfeil:
		if pf := a.pfeile[a.aktiv]; laenge(a.zuFenster(pf.B).Sub(a.zuFenster(pf.A))) < 4 {
			a.pfeile = a.pfeile[:a.aktiv]
			a.verlauf = a.verlauf[:len(a.verlauf)-1]
		}
	}
	a.letzteRate = auswerten(a.zugFrames, a.zugArbeit, "Ziehen")
	stauAuswerten(a.zugJeFrame)
	a.zug, a.aktiv = nichts, -1
}

// auswahlRechteck rechnet den aufgezogenen Rahmen in Bildpixel um, auf das Bild beschnitten.
func (a *ansicht) auswahlRechteck() image.Rectangle {
	p, q := a.zuBild(a.auswahlA), a.zuBild(a.auswahlB)
	r := image.Rect(int(math.Floor(float64(p.X))), int(math.Floor(float64(p.Y))),
		int(math.Ceil(float64(q.X))), int(math.Ceil(float64(q.Y)))).Canon()
	return r.Intersect(a.bild.Rect)
}

func (a *ansicht) inEditor() {
	r := a.bild.Rect
	if a.hatAuswahl {
		if s := a.auswahlRechteck(); s.Dx() >= 4 && s.Dy() >= 4 {
			r = s
		}
	}
	aus := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(aus, aus.Rect, a.bild, r.Min, draw.Src)
	log.Printf("[%s] Auswahl %v → Editor (%dx%d)", a.ausgabe, r, r.Dx(), r.Dy())
	for _, o := range a.gruppe {
		if o != a {
			o.fenster.Perform(system.ActionClose)
		}
	}
	a.modus = modusEditor
	a.setzeBild(aus)
	a.fenster.Invalidate()
}

func (a *ansicht) merken() {
	a.verlauf = append(a.verlauf, append([]pfeil(nil), a.pfeile...))
}

func (a *ansicht) rueckgaengig() {
	if len(a.verlauf) == 0 {
		return
	}
	a.pfeile = a.verlauf[len(a.verlauf)-1]
	a.verlauf = a.verlauf[:len(a.verlauf)-1]
	log.Printf("Rückgängig: %d Pfeile", len(a.pfeile))
}

// zoomen um den Mauspunkt; Gio liefert Scroll in Pixeln, positiv = nach unten.
func (a *ansicht) zoomen(um f32.Point, scroll float32) {
	f := float32(math.Pow(1.0015, float64(-scroll)))
	neu := a.zoom * f
	if neu < 0.05 || neu > 32 {
		return
	}
	b := a.zuBild(um)
	a.zoom = neu
	a.versatz = um.Sub(b.Mul(neu))
	log.Printf("Zoom %.0f %%", neu*100)
}

func (a *ansicht) zeichneBild(gtx layout.Context) {
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(a.zoom, a.zoom)).Offset(a.versatz)).Push(gtx.Ops)
	bop := a.bildOp
	if a.zoom > 1.5 {
		bop.Filter = paint.FilterNearest
	}
	c := clip.Rect(a.bild.Rect).Push(gtx.Ops)
	bop.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	c.Pop()
	t.Pop()
}

func (a *ansicht) zeichneAuswahl(gtx layout.Context) {
	dunkel := color.NRGBA{A: 0x90}
	if !a.hatAuswahl {
		paint.FillShape(gtx.Ops, dunkel, clip.Rect(image.Rectangle{Max: a.groesse}).Op())
		return
	}
	r := image.Rectangle{
		Min: image.Pt(int(a.auswahlA.X), int(a.auswahlA.Y)),
		Max: image.Pt(int(a.auswahlB.X), int(a.auswahlB.Y)),
	}.Canon()
	g := a.groesse
	for _, s := range []image.Rectangle{
		image.Rect(0, 0, g.X, r.Min.Y), image.Rect(0, r.Max.Y, g.X, g.Y),
		image.Rect(0, r.Min.Y, r.Min.X, r.Max.Y), image.Rect(r.Max.X, r.Min.Y, g.X, r.Max.Y),
	} {
		paint.FillShape(gtx.Ops, dunkel, clip.Rect(s).Op())
	}
	weiss := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	for _, s := range []image.Rectangle{
		image.Rect(r.Min.X-1, r.Min.Y-1, r.Max.X+1, r.Min.Y), image.Rect(r.Min.X-1, r.Max.Y, r.Max.X+1, r.Max.Y+1),
		image.Rect(r.Min.X-1, r.Min.Y, r.Min.X, r.Max.Y), image.Rect(r.Max.X, r.Min.Y, r.Max.X+1, r.Max.Y),
	} {
		paint.FillShape(gtx.Ops, weiss, clip.Rect(s).Op())
	}
}

func (a *ansicht) zeichnePfeile(gtx layout.Context) {
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(a.zoom, a.zoom)).Offset(a.versatz)).Push(gtx.Ops)
	for _, pf := range a.pfeile {
		zeichnePfeil(gtx.Ops, pf, strichBild)
	}
	t.Pop()
}

// zeichnePfeil: Kurve als Strich, Spitze als gefülltes Dreieck. Die Kurve endet am Spitzenansatz,
// damit der Strich nicht über die Spitze hinausragt.
func zeichnePfeil(ops *op.Ops, pf pfeil, breite float32) {
	d := pf.richtungAmEnde()
	n := f32.Pt(-d.Y, d.X)
	spLaenge, spBreite := breite*4.5, breite*2.2
	basis := pf.B.Sub(d.Mul(spLaenge))
	kurz := pf
	kurz.B = pf.B.Sub(d.Mul(spLaenge * 0.8))

	var p clip.Path
	p.Begin(ops)
	p.MoveTo(kurz.A)
	p.QuadTo(kurz.K, kurz.B)
	paint.FillShape(ops, pfeilFarbe, clip.Stroke{Path: p.End(), Width: breite}.Op())

	var s clip.Path
	s.Begin(ops)
	s.MoveTo(pf.B)
	s.LineTo(basis.Add(n.Mul(spBreite)))
	s.LineTo(basis.Sub(n.Mul(spBreite)))
	s.Close()
	paint.FillShape(ops, pfeilFarbe, clip.Outline{Path: s.End()}.Op())
}

func (a *ansicht) zeichneGriffe(gtx layout.Context) {
	for _, pf := range a.pfeile {
		for i, pkt := range []f32.Point{pf.A, pf.B, pf.griff()} {
			c := a.zuFenster(pkt)
			r := image.Rect(int(c.X)-griffRadius, int(c.Y)-griffRadius, int(c.X)+griffRadius, int(c.Y)+griffRadius)
			paint.FillShape(gtx.Ops, color.NRGBA{A: 0xC0}, clip.Ellipse(r).Op(gtx.Ops))
			innen := r.Inset(2)
			farbe := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if i == 2 {
				farbe = color.NRGBA{R: 0x30, G: 0xA0, B: 0xFF, A: 255} // Biegegriff blau
			}
			paint.FillShape(gtx.Ops, farbe, clip.Ellipse(innen).Op(gtx.Ops))
		}
	}
}

func (a *ansicht) zeichneLeiste(gtx layout.Context) {
	var hinweis string
	switch a.modus {
	case modusAuswahl:
		hinweis = "Bereich aufziehen · Enter = ganzes Bild · Esc = abbrechen"
	case modusEditor:
		hinweis = fmt.Sprintf("Pfeil ziehen · blauer Griff biegt · Strg+Z · Mausrad zoomt (%.0f %%) · Esc beendet", a.zoom*100)
	}
	if a.letzteRate != "" {
		hinweis += " · " + a.letzteRate
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0xB0}, clip.Rect(image.Rect(0, 0, a.groesse.X, gtx.Dp(30))).Op())
	t := op.Offset(image.Pt(gtx.Dp(10), gtx.Dp(5))).Push(gtx.Ops)
	l := material.Body2(a.thema, hinweis)
	l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	l.Layout(gtx)
	t.Pop()
}

// messSchritt bewegt den Biegegriff selbst im Kreis und fordert sofort den nächsten Frame an.
func (a *ansicht) messSchritt(gtx layout.Context, jetzt time.Time) {
	if a.messStart.IsZero() {
		a.messStart = jetzt
		b := a.bild.Rect
		a.pfeile = []pfeil{gerade(f32.Pt(float32(b.Dx())*0.2, float32(b.Dy())*0.7), f32.Pt(float32(b.Dx())*0.8, float32(b.Dy())*0.3))}
		a.zugFrames, a.zugArbeit = a.zugFrames[:0], a.zugArbeit[:0]
	}
	a.zugFrames = append(a.zugFrames, jetzt)
	t := jetzt.Sub(a.messStart).Seconds()
	b := a.bild.Rect
	m := f32.Pt(float32(b.Dx())/2, float32(b.Dy())/2)
	r := float32(b.Dy()) * 0.3
	a.pfeile[0].setzeGriff(m.Add(f32.Pt(r*float32(math.Cos(t*3)), r*float32(math.Sin(t*3)))))
	if jetzt.Sub(a.messStart) >= a.messDauer {
		auswerten(a.zugFrames, a.zugArbeit, "Messung 4K")
		exitNachLog()
	}
	gtx.Execute(op.InvalidateCmd{})
}

// auswerten: Bildrate aus den Frame-Zeitpunkten, schlechtester Abstand, 95.-Perzentil, Aufbau-Zeit.
func auswerten(frames []time.Time, arbeit []time.Duration, was string) string {
	if len(frames) < 2 {
		return ""
	}
	abst := make([]time.Duration, 0, len(frames)-1)
	for i := 1; i < len(frames); i++ {
		abst = append(abst, frames[i].Sub(frames[i-1]))
	}
	gesamt := frames[len(frames)-1].Sub(frames[0])
	rate := float64(len(abst)) / gesamt.Seconds()
	sort.Slice(abst, func(i, j int) bool { return abst[i] < abst[j] })
	p95 := abst[len(abst)*95/100]
	var maxArbeit, summe time.Duration
	for _, d := range arbeit {
		summe += d
		if d > maxArbeit {
			maxArbeit = d
		}
	}
	mittel := time.Duration(0)
	if len(arbeit) > 0 {
		mittel = summe / time.Duration(len(arbeit))
	}
	log.Printf("%s: %d Frames in %v → %.1f fps; Abstand p95 %v, max %v; Ops-Aufbau Mittel %v, max %v",
		was, len(frames), gesamt.Round(time.Millisecond), rate, p95.Round(100*time.Microsecond),
		abst[len(abst)-1].Round(100*time.Microsecond), mittel.Round(10*time.Microsecond), maxArbeit.Round(10*time.Microsecond))
	return fmt.Sprintf("%.0f fps (p95 %.1f ms)", rate, float64(p95.Microseconds())/1000)
}

// stauAuswerten trennt Ruckeln von Mauspausen: Hält das Zeichnen mit, verarbeitet jeder Frame höchstens
// ein bis zwei Zieh-Ereignisse. Stauen sich mehr, war ein Frame zu langsam. Frames ohne Ereignis (Maus
// stand still, Gio zeichnet trotzdem) werden mitgezählt, sagen aber nichts über Ruckeln.
func stauAuswerten(jeFrame []int) {
	var summe, maxJe, ueber2, leer int
	for _, n := range jeFrame {
		summe += n
		if n > maxJe {
			maxJe = n
		}
		if n > 2 {
			ueber2++
		}
		if n == 0 {
			leer++
		}
	}
	log.Printf("Stau: %d Zieh-Ereignisse in %d Frames; je Frame max %d; Frames mit >2 Ereignissen: %d; ohne Ereignis: %d",
		summe, len(jeFrame), maxJe, ueber2, leer)
}
