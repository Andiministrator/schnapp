// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/canvas"
	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/i18n"
)

// Weitere Farben laut brand/corporate-design.md.
var (
	farbeFenster    = color.NRGBA{R: 0x1E, G: 0x21, B: 0x24, A: 0xFF} // grau-900
	farbeNebentext  = color.NRGBA{R: 0x92, G: 0x98, B: 0x9F, A: 0xFF} // grau-400
	farbeGruenText  = color.NRGBA{R: 0x86, G: 0xC2, B: 0xA6, A: 0xFF} // gruen-300
	farbeLeisteVoll = color.NRGBA{R: 0x2A, G: 0x2E, B: 0x33, A: 0xFF} // grau-800, deckend
)

// offeneOverlays zählt Overlay-Fenster, die noch nicht zerstört sind (F-82: der Editor öffnet erst danach).
var offeneOverlays atomic.Int32

// ErrOverlayOffen: Nach der Frist waren noch Overlay-Fenster offen; der Editor öffnet dann nicht (F-82).
var ErrOverlayOffen = errors.New("editor: Overlay-Fenster noch offen")

// Speichern speichert das Dokument (Aufrufer: Bibliothek, ADR-0020). Der Fehlertext erscheint in der Statusleiste und
// muss schon übersetzt sein.
type Speichern func(d *document.Dokument) (Gespeichert, error)

// Gespeichert beschreibt ein gelungenes Speichern, das der Nutzer erfahren muss (F-116).
type Gespeichert struct {
	Datei   string // neuer Dateiname für die Titelleiste (nur Name, kein Ordner); leer = unverändert
	Hinweis string // übersetzter Text für die Statusleiste statt „Saved to the library“; leer = gewöhnlich
	Warnung bool   // Hinweis als Warnung zeigen (z. B. Datei nicht aktualisiert, alter Stand liegt noch)
}

// Editor öffnet das Dokument in einem Fenster und kehrt zurück, wenn es geschlossen wird. Ungespeicherte Änderungen
// werden beim Schließen gespeichert (die Bibliothek speichert immer, Kernprinzip 4); scheitert das, bleibt das Fenster
// mit der Meldung offen, und erst eine zweite Schließen-Anfrage verwirft (F-107, ADR-0020 Nachtrag 2). Ein Signal
// (SIGTERM, SIGINT, SIGHUP) speichert und schließt. Rückgabewert ist ein Fehler des Fensters oder des letzten
// Speicherns ohne Rückfrage. titel ist der Dateiname für die Titelleiste. Muss neben app.Main laufen (Starte).
func Editor(titel string, d *document.Dokument, speichern Speichern) error {
	frist := time.Now().Add(2 * time.Second)
	for offeneOverlays.Load() > 0 {
		if time.Now().After(frist) {
			return ErrOverlayOffen
		}
		time.Sleep(20 * time.Millisecond)
	}
	th := material.NewTheme()
	// Nur die Go-Schriften, wie im Renderer: sonst zeigte der Bildschirm Zeichen aus Systemschriften, die in der
	// gespeicherten Datei als Kästchen erscheinen (Kritik A-3).
	th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	e := &editor{w: new(app.Window), thema: th, s: canvas.NeueSitzung(d), titel: titel, speichern: speichern}
	e.bildNeu()
	ende := e.signaleAbfangen()
	defer ende()
	return e.laufen()
}

// signaleAbfangen leitet SIGTERM, SIGINT und SIGHUP (Abmelden, Strg+C im Terminal, kill) in ein geordnetes Schließen
// mit Speichern um (F-107): Die Signal-Goroutine setzt nur eine Marke und weckt das Fenster (Invalidate ist dafür
// gedacht); gespeichert und geschlossen wird in der eigenen Ereignisschleife (F-77). Nach dem ersten Signal gilt
// wieder die Voreinstellung — ein zweites beendet sofort. Die Rückgabe beendet das Abfangen.
func (e *editor) signaleAbfangen() func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	fertig := make(chan struct{})
	go func() {
		select {
		case s := <-sig:
			signal.Stop(sig)
			diagnoseEditor("Signal %v: speichern und schließen", s)
			e.signal.Store(true)
			e.w.Invalidate()
		case <-fertig:
		}
	}()
	return func() {
		signal.Stop(sig)
		close(fertig)
	}
}

type editor struct {
	w         *app.Window
	thema     *material.Theme
	s         *canvas.Sitzung
	titel     string
	speichern Speichern
	bildOp    paint.ImageOp
	meldung   string // Zeile in der Statusleiste (Speichern gelungen oder gescheitert)
	fehler    bool
	titelSet  string

	signal     atomic.Bool // ein Signal kam an (Signal-Goroutine setzt, Ereignisschleife liest)
	erzwungen  bool        // Schließen nach Signal: kein ClosingEvent mehr abbrechen
	schlussErr error       // Speicherfehler beim Schließen nach Signal, Rückgabe von Editor

	knoepfe struct {
		auswahl, rueck, wieder, einpassen, eins, speichern widget.Clickable
	}
}

// diagnoseEditor schreibt nur mit gesetztem SCHNAPP_DEBUG (echte Läufe).
func diagnoseEditor(format string, args ...any) {
	if os.Getenv("SCHNAPP_DEBUG") != "" {
		log.Printf("[editor] "+format, args...)
	}
}

func (e *editor) bildNeu() { e.bildOp = paint.NewImageOp(e.s.D.Ansichtsbild()) }

func (e *editor) laufen() error {
	e.w.Option(app.Title(e.fensterTitel()), app.Size(unit.Dp(1200), unit.Dp(800)), app.MinSize(unit.Dp(480), unit.Dp(320)))
	e.titelSet = e.fensterTitel()
	var ops op.Ops
	for {
		switch ev := e.w.Event().(type) {
		case app.DestroyEvent:
			diagnoseEditor("Fenster zu: Fehler %v, ungesichert %v, Zug %v", ev.Err, e.s.Ungesichert(), e.s.Zieht())
			if e.s.Zieht() {
				e.s.Abbrechen()
			}
			// Ein geordnetes Schließen hat schon gespeichert (ClosingEvent). Endet das Fenster ohne Anfrage (Wayland
			// weg), liegt das Dokument noch im Speicher: speichern (Kritik A-4). Nach einem gescheiterten Speichern
			// beim Schließen hat der Nutzer das Verwerfen mit der zweiten Anfrage bestätigt.
			var err error
			if e.s.Ungesichert() {
				var g Gespeichert
				g, err = e.speichern(e.s.D)
				diagnoseEditor("Speichern beim Zerstören: %v", err)
				if err == nil {
					e.s.Gespeichert()
					hinweisAufStderr(g)
				}
			}
			return errors.Join(ev.Err, e.schlussErr, err)
		case *app.ClosingEvent:
			// Gio fragt vor dem Zerstören (Fensterknopf, Strg+W, Compositor); Abort hält das Fenster offen (F-107).
			if !e.erzwungen && !e.schliessen() {
				ev.Abort()
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			e.frame(gtx)
			ev.Frame(gtx.Ops)
			if t := e.fensterTitel(); t != e.titelSet {
				e.titelSet = t
				e.w.Option(app.Title(t))
			}
		}
		if e.signal.Load() && !e.erzwungen {
			e.nachSignal()
		}
	}
}

// hinweisAufStderr: Das Fenster ist (gleich) zu, die Statusleiste weg — ein Hinweis (z. B. neue Aufnahme statt der
// alten Datei) geht auf die Fehlerausgabe, damit er nicht still verloren ist (F-116).
func hinweisAufStderr(g Gespeichert) {
	if g.Hinweis != "" {
		fmt.Fprintln(os.Stderr, g.Hinweis)
	}
}

// schliessen beantwortet eine Schließen-Anfrage; false hält das Fenster offen (Entscheidung in canvas.Sitzung):
// Scheitert das Speichern, steht der Fehler in der Statusleiste und eine zweite Anfrage verwirft; eine Warnung beim
// Speichern (Ausweichen auf eine neue Aufnahme) hält das Fenster einmal offen, damit sie gelesen wird.
func (e *editor) schliessen() bool {
	z, err := e.s.Schliessen(func() (bool, error) {
		g, err := e.speichern(e.s.D)
		if err != nil {
			return false, err
		}
		if g.Warnung {
			e.gespeichertMelden(g)
		} else {
			hinweisAufStderr(g)
		}
		return g.Warnung, nil
	})
	diagnoseEditor("Schließen-Anfrage: %v, %v", z, err)
	switch z {
	case canvas.SchlussZu:
		return true
	case canvas.SchlussFehler:
		e.meldung, e.fehler = fmt.Sprintf(i18n.T("Could not save: %v. Close again to discard the changes."), err), true
	}
	e.w.Invalidate() // Meldung zeigen
	return false
}

// nachSignal speichert nach einem Signal einmal und schließt das Fenster, ohne es offen zu halten.
func (e *editor) nachSignal() {
	e.erzwungen = true
	z, err := e.s.Schliessen(func() (bool, error) {
		g, err := e.speichern(e.s.D)
		if err == nil {
			hinweisAufStderr(g)
		}
		return false, err
	})
	if z == canvas.SchlussFehler {
		e.schlussErr = err
	}
	e.w.Perform(system.ActionClose) // in der eigenen Schleife (F-77)
}

func (e *editor) fensterTitel() string {
	t := e.titel + " — Schnapp"
	if e.s.Geaendert() {
		t = "• " + t
	}
	return t
}

func (e *editor) speichernJetzt() {
	if e.s.Zieht() {
		return
	}
	if !e.s.Geaendert() {
		// Nichts zu schreiben (F-109): sonst würde die Datei neu geschrieben und als bearbeitet markiert.
		e.meldung, e.fehler = i18n.T("No changes to save."), false
		return
	}
	g, err := e.speichern(e.s.D)
	if err != nil {
		e.meldung, e.fehler = fmt.Sprintf(i18n.T("Could not save: %v"), err), true
		return
	}
	e.s.Gespeichert()
	e.gespeichertMelden(g)
}

// gespeichertMelden zeigt ein gelungenes Speichern in Titel- und Statusleiste (F-116).
func (e *editor) gespeichertMelden(g Gespeichert) {
	if g.Datei != "" {
		e.titel = g.Datei
	}
	e.meldung, e.fehler = i18n.T("Saved to the library"), false
	if g.Hinweis != "" {
		e.meldung, e.fehler = g.Hinweis, g.Warnung
	}
}

func (e *editor) frame(gtx layout.Context) {
	e.tasten(gtx)
	e.knoepfeAuswerten(gtx)
	paint.Fill(gtx.Ops, farbeFenster)
	leiste := gtx.Dp(56)
	status := gtx.Dp(28)
	flaeche := image.Rect(0, leiste, gtx.Constraints.Max.X, gtx.Constraints.Max.Y-status)
	if flaeche.Dy() < 1 {
		flaeche.Max.Y = flaeche.Min.Y + 1
	}

	// Leinwand: eigener Ursprung, damit Zeigerpositionen schon Leinwandkoordinaten sind.
	t := op.Offset(flaeche.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(flaeche.Size())
	e.leinwand(g)
	t.Pop()

	e.zeichneLeiste(gtx, leiste)
	e.zeichneStatus(gtx, status)
}

// --- Leinwand ---

func (e *editor) leinwand(gtx layout.Context) {
	groesse := gtx.Constraints.Max
	e.s.SetzeFenster(groesse)
	e.zeiger(gtx)

	c := clip.Rect(image.Rectangle{Max: groesse}).Push(gtx.Ops)
	paint.Fill(gtx.Ops, farbeGrund)
	a := e.s.Abb
	af := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(a.Zoom), float32(a.Zoom))).
		Offset(f32.Pt(float32(a.Versatz.X), float32(a.Versatz.Y)))).Push(gtx.Ops)
	b := e.bildOp
	if a.Zoom >= 2 {
		b.Filter = paint.FilterNearest // einzelne Pixel sichtbar
	}
	// Objekte im selben Ausschnitt wie das Bild: Was außerhalb liegt, fehlt auch in der gespeicherten Datei (Kritik A-7).
	bc := clip.Rect(image.Rectangle{Max: e.s.D.Groesse()}).Push(gtx.Ops)
	b.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	e.zeichneObjekte(gtx)
	bc.Pop()
	af.Pop()
	e.zeichneAuswahl(gtx)

	pointer.CursorDefault.Add(gtx.Ops)
	event.Op(gtx.Ops, e)
	c.Pop()
}

func (e *editor) zeiger(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  e,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6},
		})
		if !ok {
			return
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		p := document.Pt(float64(pe.Position.X), float64(pe.Position.Y))
		switch pe.Kind {
		case pointer.Press:
			switch {
			case pe.Buttons.Contain(pointer.ButtonTertiary):
				e.s.Druecken(p, canvas.TasteMitte)
			case pe.Buttons.Contain(pointer.ButtonPrimary):
				e.s.Druecken(p, canvas.TastePrimaer)
			}
			gtx.Execute(key.FocusCmd{Tag: e})
		case pointer.Drag:
			e.s.Ziehen(p)
		case pointer.Release:
			e.s.Loslassen(p)
			diagnoseEditor("Loslassen: Auswahl %d, geändert %v", e.s.Auswahl, e.s.Geaendert())
		case pointer.Cancel:
			e.s.Abbrechen()
		case pointer.Scroll:
			// Gio liefert Pixel; ein Rastschritt des Mausrads ergibt je nach System 10–120. Auf Schritte von etwa
			// 10 % umrechnen und begrenzen, damit ein Touchpad nicht springt.
			dy := float64(pe.Scroll.Y) / 40
			if dy != 0 {
				dy = math.Copysign(math.Min(3, math.Max(0.25, math.Abs(dy))), dy)
			}
			e.s.Scrollen(p, dy)
		}
	}
}

// zeichneObjekte zeichnet in Ansichtskoordinaten (die Affine-Abbildung liegt schon auf dem Stapel). Schrift in
// Ansichtspixeln: eigene Metrik mit 1 px je sp.
func (e *editor) zeichneObjekte(gtx layout.Context) {
	d := e.s.D
	an := func(p document.Punkt) f32.Point {
		q := d.ZuAnsicht(p)
		return f32.Pt(float32(q.X), float32(q.Y))
	}
	for _, o := range d.Objekte() {
		switch v := o.(type) {
		case document.Pfeil:
			var pfad clip.Path
			pfad.Begin(gtx.Ops)
			pfad.MoveTo(an(v.A))
			const n = 48
			for i := 1; i <= n; i++ {
				pfad.LineTo(an(v.Punkt(float64(i) / n)))
			}
			paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Stroke{Path: pfad.End(), Width: float32(v.Strich.Breite)}.Op())
			r := v.RichtungAmEnde()
			gl := math.Max(8, 4*v.Strich.Breite)
			nrm := document.Pt(-r.Y, r.X)
			fuss := v.B.Minus(r.Mal(gl))
			var spitze clip.Path
			spitze.Begin(gtx.Ops)
			spitze.MoveTo(an(v.B))
			spitze.LineTo(an(fuss.Plus(nrm.Mal(gl / 2))))
			spitze.LineTo(an(fuss.Minus(nrm.Mal(gl / 2))))
			spitze.Close()
			paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Outline{Path: spitze.End()}.Op())
		case document.Kasten:
			min, max := an(v.Bereich.Min), an(v.Bereich.Max)
			r := image.Rect(int(math.Round(float64(min.X))), int(math.Round(float64(min.Y))),
				int(math.Round(float64(max.X))), int(math.Round(float64(max.Y))))
			rr := clip.UniformRRect(r, int(math.Round(v.Eckradius)))
			if v.Fuellung.A > 0 {
				paint.FillShape(gtx.Ops, v.Fuellung, rr.Op(gtx.Ops))
			}
			if v.Strich.Breite > 0 && v.Strich.Farbe.A > 0 {
				paint.FillShape(gtx.Ops, v.Strich.Farbe, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(v.Strich.Breite)}.Op())
			}
		case document.Nummer:
			m := an(v.Mitte)
			rad := float32(v.Radius)
			kr := image.Rect(int(math.Round(float64(m.X-rad))), int(math.Round(float64(m.Y-rad))),
				int(math.Round(float64(m.X+rad))), int(math.Round(float64(m.Y+rad))))
			paint.FillShape(gtx.Ops, v.Farbe, clip.Ellipse(kr).Op(gtx.Ops))
			e.textBei(gtx, fmt.Sprint(v.Wert), float32(v.Radius*1.2), m, true, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		case document.Text:
			if v.Hintergrund.A > 0 {
				g := v.Grenzen()
				a, b := an(g.Min), an(g.Max)
				paint.FillShape(gtx.Ops, v.Hintergrund, clip.Rect(image.Rect(int(a.X), int(a.Y), int(b.X), int(b.Y))).Op())
			}
			ort := an(v.Ort)
			for i, zeile := range strings.Split(v.Inhalt, "\n") {
				zeile = document.Zeichenbar(zeile) // wie im Renderer
				p := ort.Add(f32.Pt(0, float32(v.Schriftgrad*1.3*float64(i))))
				e.textBei(gtx, zeile, float32(v.Schriftgrad), p, false, v.Farbe)
			}
		}
	}
}

// textBei setzt Text in Ansichtspixeln: mittig um p (mitte) oder mit der linken oberen Ecke bei p.
func (e *editor) textBei(gtx layout.Context, s string, groesse float32, p f32.Point, mitte bool, c color.NRGBA) {
	g := gtx
	g.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	lbl := material.Label(e.thema, unit.Sp(groesse), s)
	lbl.Color = c
	lbl.MaxLines = 1
	if mitte {
		lbl.Font.Weight = 700 // fett wie im Renderer
	}
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	call := m.Stop()
	ecke := p
	if mitte {
		ecke = p.Sub(f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2))
	}
	t := op.Affine(f32.Affine2D{}.Offset(ecke)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// zeichneAuswahl zeichnet Rahmen und Griffe des ausgewählten Objekts in Fensterpixeln (unabhängig vom Zoom).
func (e *editor) zeichneAuswahl(gtx layout.Context) {
	o, ok := e.s.D.Objekt(e.s.Auswahl)
	if !ok {
		return
	}
	fe := func(p document.Punkt) image.Point {
		q := e.s.Abb.ZuFenster(e.s.D.ZuAnsicht(p))
		return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	}
	g := o.Grenzen()
	r := image.Rectangle{Min: fe(g.Min), Max: fe(g.Max)}.Canon()
	rahmen(gtx.Ops, r.Inset(-gtx.Dp(2)), gtx.Dp(1), farbeAuswahl)
	h := gtx.Dp(5)
	for _, gr := range canvas.Griffe(o) {
		m := fe(gr.Ort)
		q := image.Rect(m.X-h, m.Y-h, m.X+h, m.Y+h)
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(q).Op())
		rahmen(gtx.Ops, q, gtx.Dp(1), farbeAuswahl)
	}
}

// --- Tasten ---

func (e *editor) tasten(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: key.NameDeleteForward},
			key.Filter{Name: key.NameDeleteBackward},
			key.Filter{Name: "Z", Required: key.ModShortcut, Optional: key.ModShift},
			key.Filter{Name: "Y", Required: key.ModShortcut},
			key.Filter{Name: "S", Required: key.ModShortcut},
			key.Filter{Name: "0", Required: key.ModShortcut},
			key.Filter{Name: "1", Required: key.ModShortcut},
			key.Filter{Name: "W", Required: key.ModShortcut},
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
			if e.s.Zieht() {
				e.s.Abbrechen()
			} else {
				e.s.Abwaehlen()
			}
		case key.NameDeleteForward, key.NameDeleteBackward:
			e.s.Loeschen()
		case "Z":
			if k.Modifiers.Contain(key.ModShift) {
				e.wiederholen()
			} else {
				e.rueckgaengig()
			}
		case "Y":
			e.wiederholen()
		case "S":
			e.speichernJetzt()
		case "0":
			e.s.Einpassen()
		case "1":
			e.s.EinsZuEins()
		case "W":
			diagnoseEditor("Strg+W")
			e.w.Perform(system.ActionClose) // in der eigenen Schleife, wie im Overlay (F-77)
		}
	}
}

func (e *editor) rueckgaengig() {
	if e.s.Rueckgaengig() {
		e.bildNeu() // Schwärzen und Zuschnitt ändern das Bild selbst
	}
}

func (e *editor) wiederholen() {
	if e.s.Wiederholen() {
		e.bildNeu()
	}
}

// --- Leiste ---

func (e *editor) knoepfeAuswerten(gtx layout.Context) {
	k := &e.knoepfe
	if k.rueck.Clicked(gtx) {
		e.rueckgaengig()
	}
	if k.wieder.Clicked(gtx) {
		e.wiederholen()
	}
	if k.einpassen.Clicked(gtx) {
		e.s.Einpassen()
	}
	if k.eins.Clicked(gtx) {
		e.s.EinsZuEins()
	}
	if k.speichern.Clicked(gtx) {
		e.speichernJetzt()
	}
	k.auswahl.Clicked(gtx) // einziges Werkzeug bis zur Werkzeug-Karte; immer aktiv
}

type knopfArt int

const (
	knopfNormal knopfArt = iota
	knopfAktiv           // aktives Werkzeug: grüner Rand, grüner Text
	knopfHaupt           // Hauptaktion: grüne Fläche
)

func (e *editor) zeichneLeiste(gtx layout.Context, hoehe int) {
	breite := gtx.Constraints.Max.X
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, 0, breite, hoehe)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(0, hoehe-gtx.Dp(1), breite, hoehe)).Op())
	k := &e.knoepfe
	x := gtx.Dp(12)
	y := gtx.Dp(10)
	abstand := gtx.Dp(8)
	links := []struct {
		c    *widget.Clickable
		text string
		art  knopfArt
		an   bool
	}{
		{&k.auswahl, i18n.T("Select"), knopfAktiv, true},
		{&k.rueck, i18n.T("Undo"), knopfNormal, e.s.D.KannRueckgaengig()},
		{&k.wieder, i18n.T("Redo"), knopfNormal, e.s.D.KannWiederholen()},
		{&k.einpassen, i18n.T("Fit"), knopfNormal, true},
		{&k.eins, "1:1", knopfNormal, true},
	}
	for i, kn := range links {
		if i == 1 || i == 3 {
			x += abstand // Gruppen trennen: Werkzeuge | Verlauf | Ansicht
		}
		w := e.knopf(gtx, kn.c, kn.text, kn.art, kn.an, image.Pt(x, y))
		x += w + abstand
	}
	// Speichern rechts.
	m := op.Record(gtx.Ops)
	w := e.knopf(gtx, &k.speichern, i18n.T("Save"), knopfHaupt, true, image.Point{})
	call := m.Stop()
	t := op.Offset(image.Pt(breite-gtx.Dp(12)-w, y)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}

// knopf zeichnet eine Schaltfläche (Radius 6, Höhe 36 dp) bei ecke und liefert ihre Breite.
func (e *editor) knopf(gtx layout.Context, c *widget.Clickable, s string, art knopfArt, an bool, ecke image.Point) int {
	lbl := material.Label(e.thema, unit.Sp(14), s)
	lbl.Color = farbeText
	switch {
	case !an:
		lbl.Color = farbeNebentext
	case art == knopfAktiv:
		lbl.Color = farbeGruenText
	case art == knopfHaupt:
		lbl.Color = farbeGrund
	}
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	textCall := m.Stop()
	innen := image.Pt(gtx.Dp(14), (gtx.Dp(36)-dims.Size.Y)/2)
	groesse := image.Pt(dims.Size.X+2*innen.X, gtx.Dp(36))

	t := op.Offset(ecke).Push(gtx.Ops)
	r := image.Rectangle{Max: groesse}
	rr := clip.UniformRRect(r, gtx.Dp(6))
	switch {
	case art == knopfHaupt && an:
		paint.FillShape(gtx.Ops, farbeAuswahl, rr.Op(gtx.Ops))
	default:
		paint.FillShape(gtx.Ops, farbeGrund, rr.Op(gtx.Ops))
		rand := farbeLinie
		if art == knopfAktiv {
			rand = farbeAuswahl
		}
		paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	}
	if gtx.Focused(c) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.UniformRRect(r.Inset(-gtx.Dp(2)), gtx.Dp(8)).Path(gtx.Ops),
			Width: float32(gtx.Dp(2))}.Op())
	}
	tt := op.Offset(innen).Push(gtx.Ops)
	textCall.Add(gtx.Ops)
	tt.Pop()
	if an {
		g.Constraints = layout.Exact(groesse)
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: groesse} })
		pointer.CursorPointer.Add(gtx.Ops)
	}
	t.Pop()
	return groesse.X
}

func (e *editor) zeichneStatus(gtx layout.Context, hoehe int) {
	mx := gtx.Constraints.Max
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, mx.Y-hoehe, mx.X, mx.Y)).Op())
	g := e.s.D.Groesse()
	rechts := fmt.Sprintf("%d × %d · %d %%", g.X, g.Y, int(math.Round(e.s.Abb.Zoom*100)))
	e.statusText(gtx, rechts, farbeNebentext, mx.Y-hoehe, false)
	if e.meldung != "" {
		c := farbeGruenText
		if e.fehler {
			c = color.NRGBA{R: 0xE6, G: 0x78, B: 0x73, A: 0xFF} // rot-500
		}
		e.statusText(gtx, e.meldung, c, mx.Y-hoehe, true)
	}
}

func (e *editor) statusText(gtx layout.Context, s string, c color.NRGBA, y int, links bool) {
	lbl := material.Label(e.thema, unit.Sp(12), s)
	lbl.Color = c
	lbl.MaxLines = 1
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	dims := lbl.Layout(g)
	call := m.Stop()
	x := gtx.Dp(12)
	if !links {
		x = gtx.Constraints.Max.X - gtx.Dp(12) - dims.Size.X
	}
	t := op.Offset(image.Pt(x, y+(gtx.Dp(28)-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}
