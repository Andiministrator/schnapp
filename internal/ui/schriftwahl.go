// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Schrift im Text-Werkzeug (ADR-0031 Nachtrag 6, Bedeutung ADR-0021 Nachtrag 10): Feld „Schrift“ mit einer Liste zum
// Blättern, Größe − / +, vier Umschalter für den Stil und drei Knöpfe für die Ausrichtung. Was eine Wahl bewirkt,
// entscheidet canvas.Sitzung; hier stehen Lage, Ereignisse und die Liste.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// stilArten sind die Umschalter in der Reihenfolge der Knöpfe.
var stilArten = []canvas.StilArt{canvas.StilFett, canvas.StilKursiv, canvas.StilUnterstrichen, canvas.StilDurchgestrichen}

// ausrichtungen sind die Knöpfe der Ausrichtung in ihrer Reihenfolge.
var ausrichtungen = []document.Ausrichtung{document.AusrichtungLinks, document.AusrichtungMitte, document.AusrichtungRechts}

// tippPause ist die Zeit, nach der getippte Buchstaben in der Liste einen neuen Anfang suchen.
const tippPause = time.Second

// schriftwahl hält Knöpfe, Lage und Liste der Schrift.
type schriftwahl struct {
	feld       widget.Clickable
	feldRahmen image.Rectangle
	feldText   string // angezeigter Name beim letzten Zeichnen (Tests)
	groesse    [2]widget.Clickable
	grRahmen   [2]image.Rectangle
	grText     string
	stil       [4]widget.Clickable
	stilRahmen [4]image.Rectangle
	ausr       [3]widget.Clickable
	ausrRahmen [3]image.Rectangle
	gezeigt    bool // der Abschnitt stand beim letzten Zeichnen da

	// Liste
	offen     bool
	familien  []string // beim Öffnen gelesen
	cursor    int
	liste     widget.List
	eintraege []widget.Clickable
	rahmen    image.Rectangle // Lage der Liste
	zeilen    map[int]image.Rectangle
	aussen    int // Ziel für Klicks neben die Liste
	tipp      string
	tippZeit  time.Time
	jetzt     func() time.Time // für Tests austauschbar
}

// knoepfe sind die Knöpfe des Abschnitts in Tab-Reihenfolge (fokussierterKnopf).
func (sw *schriftwahl) knoepfe() []*widget.Clickable {
	a := []*widget.Clickable{&sw.feld, &sw.groesse[0], &sw.groesse[1]}
	for i := range sw.stil {
		a = append(a, &sw.stil[i])
	}
	for i := range sw.ausr {
		a = append(a, &sw.ausr[i])
	}
	return a
}

// schriftName ist der angezeigte Name einer Familie (leer = Go).
func schriftName(n string) string {
	if n == "" {
		return fonts.Go
	}
	return n
}

// stilBuchstabe ist die Aufschrift eines Umschalters (übersetzt: F, K, U, S auf Deutsch).
func stilBuchstabe(a canvas.StilArt) string {
	switch a {
	case canvas.StilFett:
		return i18n.TK("text style", "B")
	case canvas.StilKursiv:
		return i18n.TK("text style", "I")
	case canvas.StilUnterstrichen:
		return i18n.TK("text style", "U")
	}
	return i18n.TK("text style", "S")
}

// schriftListeOeffne öffnet die Liste mit dem Cursor auf der gezeigten Familie; der Tastaturfokus geht an die
// Leinwand, damit getippte Buchstaben (EditEvent) und die Pfeile in tasten ankommen.
func (e *editor) schriftListeOeffne(gtx layout.Context) {
	sw := &e.seite.sw
	sw.familien = fonts.Familien()
	sw.cursor = 0
	jetzt := schriftName(e.s.AktuellerTextStil().Schrift)
	for i, f := range sw.familien {
		if f == jetzt {
			sw.cursor = i
		}
	}
	if len(sw.eintraege) < len(sw.familien) {
		sw.eintraege = make([]widget.Clickable, len(sw.familien))
	}
	sw.offen, sw.tipp = true, ""
	sw.liste.Axis = layout.Vertical
	sw.liste.ScrollTo(max(0, sw.cursor-3))
	gtx.Execute(key.FocusCmd{Tag: e})
}

// schriftListeSchliesse schließt die Liste; mit zumFeld geht der Fokus an das Feld zurück, während der Texteingabe
// immer an die Leinwand.
func (e *editor) schriftListeSchliesse(gtx layout.Context, zumFeld bool) {
	sw := &e.seite.sw
	sw.offen = false
	switch {
	case e.s.TextLaeuft():
		// In der Texteingabe gehört der Fokus der Leinwand, sonst gingen weitere Zeichen verloren (Sitzungs-QA B-1).
		gtx.Execute(key.FocusCmd{Tag: e})
	case zumFeld:
		gtx.Execute(key.FocusCmd{Tag: &sw.feld})
	}
}

// schriftWaehle setzt die Familie i der Liste und schließt sie.
func (e *editor) schriftWaehle(gtx layout.Context, i int) {
	sw := &e.seite.sw
	if i >= 0 && i < len(sw.familien) && e.seiteAn() {
		e.s.SetzeSchrift(sw.familien[i])
	}
	e.schriftListeSchliesse(gtx, true)
}

// schriftListeTaste bedient die offene Liste: Pfeile hoch und runter, Pos1, Ende, Enter, Esc; andere Tasten wirken bei
// offener Liste nicht.
func (e *editor) schriftListeTaste(gtx layout.Context, n key.Name) {
	sw := &e.seite.sw
	letzter := len(sw.familien) - 1
	switch n {
	case key.NameUpArrow:
		sw.cursor = max(0, sw.cursor-1)
	case key.NameDownArrow:
		sw.cursor = min(letzter, sw.cursor+1)
	case key.NameHome:
		sw.cursor = 0
	case key.NameEnd:
		sw.cursor = letzter
	case key.NameReturn, key.NameEnter:
		e.schriftWaehle(gtx, sw.cursor)
		return
	case key.NameEscape:
		e.schriftListeSchliesse(gtx, true)
		return
	default:
		return
	}
	e.schriftCursorSichtbar()
}

// schriftCursorSichtbar blättert die Liste so, dass der Cursor sichtbar ist.
func (e *editor) schriftCursorSichtbar() {
	sw := &e.seite.sw
	p := sw.liste.Position
	if sw.cursor < p.First {
		sw.liste.ScrollTo(sw.cursor)
	} else if p.Count > 0 && sw.cursor >= p.First+p.Count-1 {
		sw.liste.ScrollTo(max(0, sw.cursor-p.Count+2))
	}
}

// schriftListeTippe springt zur ersten Familie, die mit den getippten Buchstaben beginnt (Groß- und Kleinschreibung
// egal); nach tippPause beginnt ein neuer Anfang.
func (e *editor) schriftListeTippe(text string) {
	sw := &e.seite.sw
	jetzt := time.Now
	if sw.jetzt != nil {
		jetzt = sw.jetzt
	}
	t := jetzt()
	if t.Sub(sw.tippZeit) > tippPause {
		sw.tipp = ""
	}
	sw.tippZeit = t
	sw.tipp += strings.ToLower(text)
	for i, f := range sw.familien {
		if strings.HasPrefix(strings.ToLower(f), sw.tipp) {
			sw.cursor = i
			e.schriftCursorSichtbar()
			return
		}
	}
}

// schriftAuswerten verarbeitet die Klicks des Abschnitts und der Liste.
func (e *editor) schriftAuswerten(gtx layout.Context) {
	sw := &e.seite.sw
	if sw.feld.Clicked(gtx) && e.seiteAn() {
		if sw.offen {
			e.schriftListeSchliesse(gtx, true)
		} else {
			e.schriftListeOeffne(gtx)
		}
	}
	for j, dir := range []int{-1, 1} {
		if sw.groesse[j].Clicked(gtx) && e.seiteAn() {
			if g, ok := e.s.SchriftgradStufe(dir); ok {
				e.s.SetzeSchriftgrad(g)
			}
		}
	}
	for j, a := range stilArten {
		if sw.stil[j].Clicked(gtx) && e.seiteAn() {
			e.s.SchalteStil(a)
		}
	}
	for j, a := range ausrichtungen {
		if sw.ausr[j].Clicked(gtx) && e.seiteAn() {
			e.s.SetzeAusrichtung(a)
		}
	}
	for i := range sw.eintraege {
		if sw.eintraege[i].Clicked(gtx) && sw.offen {
			e.schriftWaehle(gtx, i)
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &sw.aussen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if p, ok := ev.(pointer.Event); ok && p.Kind == pointer.Press && sw.offen {
			e.schriftListeSchliesse(gtx, false)
		}
	}
	// Ohne Abschnitt (anderes Werkzeug, Auswahl weg) bleibt keine Liste offen.
	if sw.offen && (!e.s.Seitenleiste().Text || !e.seiteSichtbar(gtx)) { // auch ohne sichtbare Leiste (B-6)
		sw.offen = false
	}
}

// zeichneSchriftwahl zeichnet den Abschnitt ab y und liefert das neue y.
func (e *editor) zeichneSchriftwahl(gtx layout.Context, x, y, breite, spalte int) int {
	sw := &e.seite.sw
	sw.gezeigt = true
	st := e.s.AktuellerTextStil()
	hoch := gtx.Dp(26)
	rechts := x + breite // rechter Rand des Schriftfelds

	// Schrift: Feld mit Namen und ▾.
	e.seiteText(gtx, i18n.T("Font"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	sw.feldRahmen = image.Rect(x+spalte, y, rechts, y+hoch)
	sw.feldText = schriftName(st.Schrift)
	if st.Fehlt {
		sw.feldText = fmt.Sprintf(i18n.T("%s (missing)"), sw.feldText)
	}
	e.knopfFlaeche(gtx, &sw.feld, sw.feldRahmen, sw.offen, true)
	e.seiteText(gtx, sw.feldText, 13, farbeText, sw.feldRahmen.Min.Add(image.Pt(gtx.Dp(6), gtx.Dp(4))),
		sw.feldRahmen.Dx()-gtx.Dp(26), 1)
	zeichneAufklappPfeil(gtx, image.Pt(sw.feldRahmen.Max.X-gtx.Dp(12), y+hoch/2-gtx.Dp(2)), farbeNebentext)
	seiteKnopf(gtx, &sw.feld, sw.feldRahmen)
	y += hoch + gtx.Dp(8)

	// Größe: − Wert +.
	kb := gtx.Dp(30)
	e.seiteText(gtx, i18n.T("Size"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	wert := image.Rect(x+spalte+kb, y, x+spalte+kb+gtx.Dp(64), y+hoch)
	for j, dir := range []int{-1, 1} {
		r := image.Rect(x+spalte, y, x+spalte+kb, y+hoch)
		zeichen := "−"
		if dir > 0 {
			r, zeichen = image.Rect(wert.Max.X, y, wert.Max.X+kb, y+hoch), "+"
		}
		sw.grRahmen[j] = r
		_, an := e.s.SchriftgradStufe(dir)
		e.knopfFlaeche(gtx, &sw.groesse[j], r, false, an)
		schrift := farbeText
		if !an {
			schrift = farbeNebentext
		}
		e.textMittig(gtx, zeichen, 14, schrift, r)
		seiteKnopf(gtx, &sw.groesse[j], r) // auch grau angemeldet, wie die Strichstärke (Sitzungs-QA A-5)
	}
	sw.grText = fmt.Sprintf(i18n.TK("status bar", "%d px"), int(math.Round(st.Grad)))
	e.textMittig(gtx, sw.grText, 13, farbeText, wert)
	y += hoch + gtx.Dp(8)

	// Stil: F K U S.
	e.seiteText(gtx, i18n.T("Style"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	q, abst := gtx.Dp(26), gtx.Dp(4)
	for j, a := range stilArten {
		r := image.Rect(x+spalte+j*(q+abst), y, x+spalte+j*(q+abst)+q, y+hoch)
		sw.stilRahmen[j] = r
		e.knopfFlaeche(gtx, &sw.stil[j], r, st.An(a), true)
		e.zeichneStilBuchstabe(gtx, a, r)
		seiteKnopf(gtx, &sw.stil[j], r)
	}
	y += hoch + gtx.Dp(8)

	// Ausrichtung: eigene Zeile (neben dem Stil reicht die Breite der Leiste nicht für sieben Knöpfe).
	e.seiteText(gtx, i18n.T("Align"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	for j, a := range ausrichtungen {
		r := image.Rect(x+spalte+j*(q+abst), y, x+spalte+j*(q+abst)+q, y+hoch)
		sw.ausrRahmen[j] = r
		e.knopfFlaeche(gtx, &sw.ausr[j], r, st.Ausrichtung == a, true)
		zeichneAusrichtung(gtx, a, r.Inset(gtx.Dp(6)))
		seiteKnopf(gtx, &sw.ausr[j], r)
	}
	return y + hoch
}

// knopfFlaeche zeichnet Grund und Rand eines Knopfs der Leiste: aktiv grün, sonst Rand heller beim Überfahren.
func (e *editor) knopfFlaeche(gtx layout.Context, c *widget.Clickable, r image.Rectangle, aktiv, an bool) {
	grund, rnd := farbeGrund, farbeLinie
	switch {
	case aktiv:
		grund, rnd = farbeAuswahl, farbeAuswahl
	case an && c.Hovered():
		rnd = farbeNebentext
	}
	paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
}

// zeichneStilBuchstabe setzt die Aufschrift eines Umschalters, Unter- und Durchstrich als Linie dazu.
func (e *editor) zeichneStilBuchstabe(gtx layout.Context, a canvas.StilArt, r image.Rectangle) {
	lbl := material.Label(e.thema, 13, stilBuchstabe(a))
	lbl.Color = farbeText
	switch a {
	case canvas.StilFett:
		lbl.Font.Weight = font.Bold
	case canvas.StilKursiv:
		lbl.Font.Style = font.Italic
	}
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	m := op.Record(gtx.Ops)
	d := lbl.Layout(g)
	call := m.Stop()
	ecke := r.Min.Add(r.Size().Sub(d.Size).Div(2))
	t := op.Offset(ecke).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
	w := gtx.Dp(1)
	switch a {
	case canvas.StilUnterstrichen:
		yy := ecke.Y + d.Size.Y - d.Baseline + gtx.Dp(2)
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(ecke.X, yy, ecke.X+d.Size.X, yy+w)).Op())
	case canvas.StilDurchgestrichen:
		yy := ecke.Y + d.Size.Y - d.Baseline - d.Size.Y*3/10
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(ecke.X-w, yy, ecke.X+d.Size.X+w, yy+w)).Op())
	}
}

// zeichneAusrichtung zeichnet das Symbol der Ausrichtung: vier Linien, kurze links, mittig oder rechts.
func zeichneAusrichtung(gtx layout.Context, a document.Ausrichtung, r image.Rectangle) {
	h := max(1, gtx.Dp(2))
	for i, anteil := range []int{100, 60, 100, 60} {
		l := r.Dx() * anteil / 100
		x := r.Min.X
		switch a {
		case document.AusrichtungMitte:
			x += (r.Dx() - l) / 2
		case document.AusrichtungRechts:
			x += r.Dx() - l
		}
		y := r.Min.Y + i*(r.Dy()-h)/3
		paint.FillShape(gtx.Ops, farbeText, clip.Rect(image.Rect(x, y, x+l, y+h)).Op())
	}
}

// zeichneSchriftliste zeichnet die offene Liste über allem unter dem Feld (nach Leisten und Fußzeile).
func (e *editor) zeichneSchriftliste(gtx layout.Context) {
	sw := &e.seite.sw
	sw.rahmen, sw.zeilen = image.Rectangle{}, nil
	if !sw.offen || !sw.gezeigt || len(sw.familien) == 0 {
		return
	}
	mx := gtx.Constraints.Max
	// Klicks daneben schließen die Liste (wie bei den Menüs).
	ac := clip.Rect(image.Rectangle{Max: mx}).Push(gtx.Ops)
	event.Op(gtx.Ops, &sw.aussen)
	ac.Pop()

	zeile := gtx.Dp(26)
	unten := mx.Y - gtx.Dp(fusszeileDp)
	hoehe := min(len(sw.familien)*zeile, gtx.Dp(320), unten-sw.feldRahmen.Max.Y)
	if hoehe < zeile {
		return
	}
	sw.rahmen = image.Rect(sw.feldRahmen.Min.X, sw.feldRahmen.Max.Y, sw.feldRahmen.Max.X, sw.feldRahmen.Max.Y+hoehe)
	t := op.Offset(sw.rahmen.Min).Push(gtx.Ops)
	flaeche := image.Rectangle{Max: sw.rahmen.Size()}
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(flaeche).Op())
	hf := clip.Rect(flaeche).Push(gtx.Ops)
	event.Op(gtx.Ops, sw) // die Fläche selbst fängt Klicks zwischen den Zeilen ab
	g := gtx
	g.Constraints = layout.Exact(flaeche.Size())
	sw.zeilen = map[int]image.Rectangle{}
	material.List(e.thema, &sw.liste).Layout(g, len(sw.familien), func(gtx layout.Context, i int) layout.Dimensions {
		r := image.Rectangle{Max: image.Pt(flaeche.Dx(), zeile)}
		grund := color.NRGBA{}
		switch {
		case i == sw.cursor:
			grund = farbeAuswahl
		case sw.eintraege[i].Hovered():
			grund = farbeLinie
		}
		if grund.A > 0 {
			paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
		}
		e.seiteText(gtx, sw.familien[i], 13, farbeText, image.Pt(gtx.Dp(8), gtx.Dp(5)), r.Dx()-gtx.Dp(16), 1)
		cc := clip.Rect(r).Push(gtx.Ops)
		sw.eintraege[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
		pointer.CursorPointer.Add(gtx.Ops)
		cc.Pop()
		return layout.Dimensions{Size: r.Size()}
	})
	p := sw.liste.Position
	for i := p.First; i < p.First+p.Count && i < len(sw.familien); i++ {
		oy := (i-p.First)*zeile - p.Offset
		sw.zeilen[i] = image.Rect(sw.rahmen.Min.X, sw.rahmen.Min.Y+oy, sw.rahmen.Max.X, sw.rahmen.Min.Y+oy+zeile)
	}
	hf.Pop()
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(flaeche).Path(), Width: float32(gtx.Dp(1))}.Op())
	t.Pop()
}
