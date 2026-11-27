// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"image/color"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
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
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/selection"
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
var ErrAbgebrochen = errors.New("ui: abgebrochen")

// diagnose schreibt nur mit gesetztem SCHNAPP_DEBUG auf stderr (Fehlersuche bei echten Läufen, ohne Monitornamen).
func diagnose(format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		log.Printf("[overlay] "+format, args...)
	}
}

// Abdunkelungs-Alphawerte (F-632, ADR-0020 Nachtrag 14). Vor der Umstellung auf Mischen im sRGB-Raum (26fd556) mischte
// Gio linear: 0x99 (60 %) grau-950 über Weiß ergab sRGB 170 (Linearwert 0,4·1 + 0,6·0,008 = 0,405 → sRGB 0,669).
// Dasselbe ergibt im sRGB-Raum 255 − (255 − 22)·a = 170 für a = 85/233 = 0,365, also 0x5D (93/255 = 0,365). Über
// Grau 128 liefert die alte Linearmischung 85, die neue 89; über Dunkelgrau 40 alt 30, neu 33. Reines Schwarz (Zuschneiden)
// braucht für 170 über Weiß a = 1/3, also 0x55. Geprüft von TestAbdunkelnWieVorDerUmstellung.
const (
	abdunkelnAlpha        = 0x5D
	abdunkelnSchwarzAlpha = 0x55
)

// Farben laut brand/corporate-design.md.
var (
	farbeZuschnittDunkel = color.NRGBA{A: abdunkelnSchwarzAlpha}                     // Zuschneiden: schwarz, wirkt wie 0x99 vor der sRGB-Mischung
	farbeAbdunkeln       = color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: abdunkelnAlpha} // grau-950, durchscheinend
	farbeLeiste          = color.NRGBA{R: 0x2A, G: 0x2E, B: 0x33, A: 0xF0}           // grau-800
	farbeLinie           = color.NRGBA{R: 0x3A, G: 0x3F, B: 0x45, A: 0xFF}           // grau-700
	farbeText            = color.NRGBA{R: 0xEC, G: 0xEA, B: 0xE5, A: 0xFF}           // grau-100
	farbeAuswahl         = color.NRGBA{R: 0x4F, G: 0x94, B: 0x79, A: 0xFF}           // gruen-500
	farbeAuswahlFlaeche  = color.NRGBA{R: 0x4F, G: 0x94, B: 0x79, A: 0x22}           // gruen-500, leicht getönt (ADR-0039)
	farbeAkzent          = color.NRGBA{R: 0xF5, G: 0xA6, B: 0x5B, A: 0xFF}           // orange-500
	farbeGrund           = color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0xFF}           // grau-950
)

// Overlay zeigt je Monitor sein eingefrorenes Bild im randlosen Vollbild und lässt einen Bereich wählen (im
// Monitor-Modus auch einen ganzen Monitor per Klick). Es kehrt zurück, sobald gewählt oder mit Esc abgebrochen ist.
// Vorher schließen sich alle Fenster, jedes in seiner eigenen Ereignisschleife: Window.Perform(ActionClose) aus
// einer fremden Goroutine läuft in Gio v0.10.3 in einen Datenwettlauf und hängt (F-77); unter Wayland führt Gio
// Perform im aufrufenden Goroutine aus, in der eigenen Schleife ist es also wettlauffrei. Muss neben app.Main laufen
// (Starte).
func Overlay(bilder []MonitorBild, m selection.Modus) (OverlayErgebnis, error) {
	if len(bilder) == 0 {
		return OverlayErgebnis{}, errors.New("ui: keine Bilder")
	}
	g := &gruppe{fertig: make(chan ergebnis, 1), zu: make(chan struct{}, len(bilder))}
	g.aktiv.Store(-1)
	for i, b := range bilder {
		// Jedes Fenster läuft in einer eigenen Goroutine; text.Shaper ist nicht nebenläufig nutzbar, darum je
		// Fenster ein eigenes Thema (sonst Absturz im Text-Setzer, erster echter Lauf 19.11.2026).
		th := material.NewTheme()
		th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
		g.fenster = append(g.fenster, &overlayFenster{
			gruppe: g, index: i, ausgabe: b.Name, thema: th, modus: m,
			w:      new(app.Window),
			bildOp: paint.NewImageOp(b.Bild),
			z:      selection.Neu(m, b.Bild.Bounds().Size()),
		})
	}
	for _, f := range g.fenster {
		offeneOverlays.Add(1) // zurück beim DestroyEvent; der Editor wartet darauf (F-82)
		go func() {
			if err := f.laufen(); err != nil {
				g.melde(ergebnis{err: err})
			}
		}()
	}
	e := <-g.fertig
	// Alle Fenster anstoßen; jedes schließt sich beim nächsten Frame selbst. Ein Invalidate verpufft, solange ein
	// Fenster gerade einen Frame bearbeitet (F-79, F-99) — darum bis zum Schließen alle 50 ms erneut anstoßen. Kurz
	// warten, damit das Overlay vor dem Speichern verschwindet, aber nie hängen bleiben.
	frist := time.After(fensterFrist)
	takt := time.NewTicker(50 * time.Millisecond)
	defer takt.Stop()
	for _, f := range g.fenster {
		f.w.Invalidate()
	}
	for offen := len(g.fenster); offen > 0; {
		select {
		case <-g.zu:
			offen--
		case <-takt.C:
			for _, f := range g.fenster {
				f.w.Invalidate() // auf geschlossenen Fenstern wirkungslos
			}
		case <-frist:
			diagnose("Fenster nicht in %v geschlossen", fensterFrist)
			return e.OverlayErgebnis, e.err
		}
	}
	return e.OverlayErgebnis, e.err
}

// fensterFrist: so lange wartet Overlay höchstens auf das Schließen der Fenster. Läuft sie ab, kehrt Overlay
// trotzdem mit dem Ergebnis zurück; noch offene Fenster schließen sich, sobald sie einen Frame bekommen, oder enden
// mit dem Prozess. Der Editor im selben Prozess wartet deshalb, bis offeneOverlays null ist, und öffnet sonst nicht
// (ErrOverlayOffen, F-82).
const fensterFrist = time.Second

type ergebnis struct {
	OverlayErgebnis
	err error
}

type gruppe struct {
	fenster []*overlayFenster
	fertig  chan ergebnis
	einmal  sync.Once
	beendet atomic.Bool   // gesetzt, sobald ein Ergebnis da ist: die Fenster schließen sich
	zu      chan struct{} // je geschlossenem Fenster ein Wert
	// aktiv ist der Monitor, auf dem der Zeiger zuletzt war (-1: noch keiner). Gio meldet das Verlassen eines
	// Fensters nicht (gio_onPointerLeave schickt nur bei Compositor-Zügen ein Cancel), darum erfährt das alte Fenster
	// es über die Gruppe — sonst bliebe Fadenkreuz und Lupe dort stehen.
	aktiv atomic.Int32
}

// zeigerAuf meldet, dass der Zeiger auf Monitor i ist, und lässt das zuvor aktive Fenster neu zeichnen.
func (g *gruppe) zeigerAuf(i int) {
	if alt := int(g.aktiv.Swap(int32(i))); alt != i && alt >= 0 && alt < len(g.fenster) {
		g.fenster[alt].w.Invalidate() // Invalidate ist aus jeder Goroutine erlaubt
	}
}

// melde nimmt nur das erste Ergebnis an (zwei Fenster können fast gleichzeitig fertig werden).
func (g *gruppe) melde(e ergebnis) {
	g.einmal.Do(func() {
		g.beendet.Store(true)
		g.fertig <- e
	})
}

type overlayFenster struct {
	gruppe  *gruppe
	index   int
	ausgabe string
	thema   *material.Theme
	modus   selection.Modus
	w       *app.Window
	bildOp  paint.ImageOp
	z       *selection.Zustand
	groesse image.Point
}

func (f *overlayFenster) laufen() error {
	f.w.Option(app.Title("Schnapp"), app.FullscreenOn(f.ausgabe), app.Fullscreen.Option())
	var ops op.Ops
	for {
		switch e := f.w.Event().(type) {
		case app.DestroyEvent:
			offeneOverlays.Add(-1)
			f.gruppe.zu <- struct{}{}
			if e.Err != nil {
				return e.Err
			}
			// Vom Compositor oder Nutzer geschlossen, ohne dass gewählt wurde: wie Esc (nach einem Ergebnis wirkungslos).
			f.gruppe.melde(ergebnis{err: ErrAbgebrochen})
			return nil
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if f.gruppe.beendet.Load() {
				e.Frame(gtx.Ops)
				f.w.Perform(system.ActionClose) // in der eigenen Schleife, siehe Overlay
				continue
			}
			f.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (f *overlayFenster) frame(gtx layout.Context) {
	if gtx.Constraints.Max != f.groesse {
		f.groesse = gtx.Constraints.Max
		f.z.SetzeAbbildung(f.groesse) // der erste Frame kommt oft in falscher Skala (gio-wayland.md)
		diagnose("Monitor %d: Fenster %v px, PxPerDp %.3f, Bild %v, Zoom %.4f, Versatz %.1f,%.1f",
			f.index+1, f.groesse, gtx.Metric.PxPerDp, f.z.Groesse, f.z.Abb.Zoom, f.z.Abb.Versatz.X, f.z.Abb.Versatz.Y)
	}
	f.tasten(gtx)
	f.zeiger(gtx)
	if probeAutoWahl && f.index == 0 && f.groesse.X > 0 {
		// Nur im Probe-Build (Tag probe, probe_an.go): wählt ohne Eingabe den ganzen ersten Monitor, damit ein
		// virtueller KWin den Weg Overlay → Ablegen → Editor ohne Nutzer durchläuft.
		f.gruppe.melde(ergebnis{OverlayErgebnis: OverlayErgebnis{Monitor: 0, Bereich: image.Rectangle{Max: f.z.Groesse}}})
	}
	if f.gruppe.beendet.Load() {
		// Ergebnis kam in diesem Frame (Loslassen, Esc, Enter): Ein Invalidate aus Overlay verpufft, solange dieses
		// Fenster einen Frame bearbeitet (Gio setzt mayInvalidate erst danach). Darum den nächsten Frame selbst
		// anfordern — in ihm schließt sich das Fenster (F-79).
		gtx.Execute(op.InvalidateCmd{})
	}
	if int(f.gruppe.aktiv.Load()) != f.index {
		f.z.Verlassen()
	}

	paint.Fill(gtx.Ops, farbeGrund)
	f.zeichneBild(gtx)
	f.zeichneAuswahl(gtx)
	if p, ok := f.z.Zeiger(); ok {
		f.zeichneFadenkreuz(gtx, p)
		f.zeichneLupe(gtx, p)
	}
	f.zeichneHinweis(gtx)

	bereich := clip.Rect(image.Rectangle{Max: f.groesse}).Push(gtx.Ops)
	// Eigenes Fadenkreuz statt System-Zeiger: Gio lädt das Zeigerbild in der vorab geschätzten Skala und setzt den
	// Hotspot mit der späteren, sodass Zeigerbild und Klickpunkt bei gemischten Skalen auseinanderfallen (F-78).
	pointer.CursorNone.Add(gtx.Ops)
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
			// Tasten kommen im Fenster mit dem Tastaturfokus an; gemeint ist aber der Monitor unter dem Zeiger
			// (Hinweisleiste, grüner Rand). Ohne bekannten Zeigermonitor tut Enter nichts (F-80).
			if i := int(f.gruppe.aktiv.Load()); i >= 0 && i < len(f.gruppe.fenster) {
				ziel := f.gruppe.fenster[i]
				f.gruppe.melde(ergebnis{OverlayErgebnis: OverlayErgebnis{Monitor: i, Bereich: image.Rectangle{Max: ziel.z.Groesse}}})
			}
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
		p := selection.Punkt{X: e.Position.X, Y: e.Position.Y}
		switch e.Kind {
		case pointer.Move, pointer.Drag, pointer.Enter:
			if int(f.gruppe.aktiv.Load()) != f.index {
				diagnose("Monitor %d: Zeiger kommt an bei %.2f,%.2f", f.index+1, p.X, p.Y)
			}
			f.gruppe.zeigerAuf(f.index)
			f.z.Bewegt(p)
		case pointer.Leave:
			f.z.Verlassen()
			gtx.Execute(op.InvalidateCmd{}) // Fadenkreuz und Lupe sofort entfernen, nicht erst beim nächsten Ereignis
		case pointer.Press:
			if e.Buttons == pointer.ButtonSecondary {
				// Ausweg per Maus, falls kein Overlay-Fenster den Tastaturfokus hat (F-81).
				f.gruppe.melde(ergebnis{err: ErrAbgebrochen})
				continue
			}
			if e.Buttons == pointer.ButtonPrimary {
				f.gruppe.zeigerAuf(f.index)
				f.z.Gedrueckt(p)
				diagnose("Monitor %d: Druck bei %.2f,%.2f → Pixel %v", f.index+1, p.X, p.Y, f.z.Abb.Pixel(p, f.z.Groesse))
			}
		case pointer.Release:
			e := f.z.Losgelassen(p)
			diagnose("Monitor %d: Loslassen bei %.2f,%.2f → fertig %v, Bereich %v", f.index+1, p.X, p.Y, e.Fertig, e.Bereich)
			f.fertig(e)
		case pointer.Cancel:
			f.z.Abgebrochen()
		}
	}
}

func (f *overlayFenster) fertig(e selection.Ergebnis) {
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

	call, groesse := f.schild(gtx, selection.MassText(r), true)
	ecke := selection.MassPlatz(fr, groesse, f.groesse)
	t := op.Offset(ecke).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// zeichneFadenkreuz: waagerechte und senkrechte Linie durch das Pixel unter dem Zeiger, hell mit dunklem Saum,
// damit sie auf jedem Grund sichtbar ist.
func (f *overlayFenster) zeichneFadenkreuz(gtx layout.Context, p selection.Punkt) {
	x, y := int(p.X), int(p.Y)
	g := f.groesse
	saum := color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0xC0}
	paint.FillShape(gtx.Ops, saum, clip.Rect(image.Rect(0, y-1, g.X, y+2)).Op())
	paint.FillShape(gtx.Ops, saum, clip.Rect(image.Rect(x-1, 0, x+2, g.Y)).Op())
	paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(0, y, g.X, y+1)).Op())
	paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(x, 0, x+1, g.Y)).Op())
}

// zeichneLupe: vergrößerter Ausschnitt mit Pixelraster und markiertem Mittelpixel, darunter die Position.
func (f *overlayFenster) zeichneLupe(gtx layout.Context, p selection.Punkt) {
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
	for i := 1; i < selection.LupePixel; i++ {
		x := l.Ziel.Min.X + i*l.Faktor
		y := l.Ziel.Min.Y + i*l.Faktor
		paint.FillShape(gtx.Ops, raster, clip.Rect(image.Rect(x, l.Ziel.Min.Y, x+1, l.Ziel.Max.Y)).Op())
		paint.FillShape(gtx.Ops, raster, clip.Rect(image.Rect(l.Ziel.Min.X, y, l.Ziel.Max.X, y+1)).Op())
	}
	h := selection.LupePixel / 2 * l.Faktor
	mitte := image.Rect(l.Ziel.Min.X+h, l.Ziel.Min.Y+h, l.Ziel.Min.X+h+l.Faktor, l.Ziel.Min.Y+h+l.Faktor)
	// Fadenkreuz in der Lupe: Linien durch die Mitte des Mittelpixels, das Mittelpixel selbst bleibt frei.
	mx, my := mitte.Min.X+l.Faktor/2, mitte.Min.Y+l.Faktor/2
	saum := color.NRGBA{R: 0x16, G: 0x18, B: 0x1B, A: 0xC0}
	for _, r := range []image.Rectangle{
		image.Rect(l.Ziel.Min.X, my, mitte.Min.X-2, my+1), image.Rect(mitte.Max.X+2, my, l.Ziel.Max.X, my+1),
		image.Rect(mx, l.Ziel.Min.Y, mx+1, mitte.Min.Y-2), image.Rect(mx, mitte.Max.Y+2, mx+1, l.Ziel.Max.Y),
	} {
		paint.FillShape(gtx.Ops, saum, clip.Rect(r.Inset(-1)).Op())
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(r).Op())
	}
	rahmen(gtx.Ops, mitte.Inset(-1), 2, farbeAkzent)
	c.Pop()
	rahmen(gtx.Ops, l.Ziel.Inset(-1), 1, farbeLinie)

	call, groesse := f.schild(gtx, selection.PositionText(l.Mitte), true)
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
	if f.modus == selection.ModusMonitor {
		s = i18n.T("Click a monitor or drag an area · Enter: monitor under the pointer · Esc or right-click: cancel")
	} else {
		s = i18n.T("Drag an area · Enter: whole monitor under the pointer · Esc or right-click: cancel")
	}
	call, groesse := f.schild(gtx, s, false)
	t := op.Offset(image.Pt((f.groesse.X-groesse.X)/2, gtx.Dp(16))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// schildUmriss ist der Umriss der Schild-Leiste: eckig, ohne Rundung (Karte #186, ADR-0021 Nachtrag 4, CD §5).
func schildUmriss(groesse image.Point) clip.RRect {
	return clip.UniformRRect(image.Rectangle{Max: groesse}, 0)
}

// schild zeichnet Text auf einer eckigen Leiste und liefert die Aufzeichnung samt Größe (zum Platzieren).
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
	paint.FillShape(gtx.Ops, farbeLeiste, schildUmriss(groesse).Op(gtx.Ops))
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
