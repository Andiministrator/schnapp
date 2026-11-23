// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Rechte Leiste des Editors (ADR-0031 Punkt 3, Nachtrag 3): oben Werkzeug bzw. Auswahl mit ihren Eigenschaften,
// darunter die Farbe. Was sie zeigt, entscheidet canvas.Sitzung.Seitenleiste ohne Gio; hier stehen Texte, Lage und
// Klicks.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// kantenFormen sind die Formen der Stauch-Kante in der Reihenfolge der Knöpfe.
var kantenFormen = []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken}

// seitenKnoepfe sind die Knöpfe der rechten Leiste; die Rahmen halten ihre Lage beim letzten Zeichnen (Tests).
type seitenKnoepfe struct {
	farben       [8]widget.Clickable // je Beschriftungsfarbe einer
	kanten       [3]widget.Clickable // je Kantenform einer (kantenFormen)
	strich       [2]widget.Clickable // Strichstärke − und + (ADR-0021 Nachtrag 7)
	strRahmen    [2]image.Rectangle
	farbRahmen   [8]image.Rectangle
	kantRahmen   [3]image.Rectangle
	kopf         string // Überschrift oben beim letzten Zeichnen (Tests)
	farbeGezeigt bool   // der Abschnitt Farbe stand beim letzten Zeichnen da
}

// alle sind die Knöpfe der Leiste (Tastaturfokus, editor.fokussierterKnopf).
func (k *seitenKnoepfe) alle() []*widget.Clickable {
	var a []*widget.Clickable
	for i := range canvas.Beschriftungsfarben {
		a = append(a, &k.farben[i])
	}
	for i := range k.kanten {
		a = append(a, &k.kanten[i])
	}
	return append(a, &k.strich[0], &k.strich[1])
}

// kantenName ist der Name einer Kantenform auf ihrem Knopf.
func kantenName(k document.Kante) string {
	switch k {
	case document.KanteWelle:
		return i18n.T("Wave")
	case document.KanteZacken:
		return i18n.T("Zigzag")
	}
	return i18n.T("Torn edge")
}

// werkzeugName ist der Name eines Werkzeugs (wie in der Werkzeugleiste und im Menü „Werkzeuge“).
func (e *editor) werkzeugName(w canvas.Werkzeug) string {
	for _, k := range e.werkzeugKnoepfe() {
		if k.w == w {
			return k.text
		}
	}
	return ""
}

// seitenKopf ist die Überschrift oben in der Leiste: „Werkzeug: Text“ bzw. „Auswahl: Pfeil“, in Versalien gesetzt.
func (e *editor) seitenKopf(i canvas.SeitenInhalt) string {
	f := i18n.T("Tool: %s")
	if i.Auswahl {
		f = i18n.T("Selection: %s")
	}
	return strings.ToUpper(fmt.Sprintf(f, e.werkzeugName(i.Werkzeug)))
}

// seitenHinweisText ist der Nebentext unter den Eigenschaften.
func seitenHinweisText(h canvas.SeitenHinweis) string {
	switch h {
	case canvas.SeitenHinweisNichts:
		return i18n.T("Nothing selected. Click an object to change it, or pick a tool on the left.")
	case canvas.SeitenHinweisEndgueltig:
		return i18n.T("The pixels are overwritten for good once saved.")
	case canvas.SeitenHinweisOriginal:
		return i18n.T("Removed parts stay in the working file; redact what must go.")
	case canvas.SeitenHinweisKeineNaht:
		return i18n.T("The seam style applies once a part is cut out.")
	}
	return ""
}

// seiteAuswerten verarbeitet die Klicks der Leiste; während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6).
func (e *editor) seiteAuswerten(gtx layout.Context) {
	an := func() bool { return !e.s.SpeichernLaeuft() }
	k := &e.seite
	for j, c := range canvas.Beschriftungsfarben {
		if k.farben[j].Clicked(gtx) && an() {
			e.s.SetzeFarbe(c)
		}
	}
	for j, f := range kantenFormen {
		if k.kanten[j].Clicked(gtx) && an() && e.s.SetzeKante(f) {
			e.melde(kantenMeldung(f), false)
		}
	}
	for j, dir := range []int{-1, 1} {
		if k.strich[j].Clicked(gtx) && an() {
			if b, ok := e.s.BreiteStufe(dir); ok {
				e.s.SetzeBreite(b)
			}
		}
	}
}

// seiteText setzt text bei p (links oben) in höchstens breite Pixeln, umbrochen, und liefert die Höhe.
func (e *editor) seiteText(gtx layout.Context, text string, sp float32, c color.NRGBA, p image.Point, breite int, zeilen int) int {
	lbl := material.Label(e.thema, unit.Sp(sp), text)
	lbl.Color = c
	lbl.MaxLines = zeilen
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, breite), gtx.Constraints.Max.Y)}
	t := op.Offset(p).Push(gtx.Ops)
	d := lbl.Layout(g)
	t.Pop()
	return d.Size.Y
}

// seiteKnopf meldet einen Knopf bei r an (Klickfläche, Zeiger) und zeichnet den Fokusring.
func seiteKnopf(gtx layout.Context, c *widget.Clickable, r image.Rectangle) {
	if gtx.Focused(c) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(-gtx.Dp(3))).Path(),
			Width: float32(gtx.Dp(2))}.Op())
	}
	t := op.Offset(r.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(r.Size())
	c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
	pointer.CursorPointer.Add(gtx.Ops)
	t.Pop()
}

// zeichneSeitenleiste zeichnet die rechte Leiste, wenn sie sichtbar ist (seiteSichtbar).
func (e *editor) zeichneSeitenleiste(gtx layout.Context) {
	k := &e.seite
	k.kopf, k.farbeGezeigt = "", false
	k.farbRahmen, k.kantRahmen, k.strRahmen = [8]image.Rectangle{}, [3]image.Rectangle{}, [2]image.Rectangle{}
	if !e.seiteSichtbar(gtx) {
		return
	}
	mx := gtx.Constraints.Max
	x0, oben, unten := mx.X-gtx.Dp(seiteDp), gtx.Dp(menuezeileDp), mx.Y-gtx.Dp(fusszeileDp)
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(x0, oben, mx.X, unten)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(x0, oben, x0+gtx.Dp(1), unten)).Op())
	// Nichts läuft unter die Fußzeile (niedrige Fenster).
	rand := clip.Rect(image.Rect(x0, oben, mx.X, unten)).Push(gtx.Ops)
	defer rand.Pop()

	innen := gtx.Dp(16)
	x, breite := x0+innen, gtx.Dp(seiteDp)-2*innen
	spalte := gtx.Dp(88) // Beschriftungen links, Bedienelemente rechts davon (Muster v2)
	y := oben + gtx.Dp(16)
	trenne := func() {
		y += gtx.Dp(14)
		paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(x0+gtx.Dp(8), y, mx.X-gtx.Dp(8), y+gtx.Dp(1))).Op())
		y += gtx.Dp(14)
	}

	inhalt := e.s.Seitenleiste()
	k.kopf = e.seitenKopf(inhalt)
	y += e.seiteText(gtx, k.kopf, 12, farbeNebentext, image.Pt(x, y), breite, 1) + gtx.Dp(12)

	if inhalt.Kante {
		hoch := gtx.Dp(26)
		e.seiteText(gtx, i18n.T("Seam style"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
		kb := (breite - spalte - 2*gtx.Dp(4)) / len(kantenFormen)
		jetzt := e.s.D.Kante()
		for j, f := range kantenFormen {
			r := image.Rect(x+spalte+j*(kb+gtx.Dp(4)), y, x+spalte+j*(kb+gtx.Dp(4))+kb, y+hoch)
			k.kantRahmen[j] = r
			grund, rnd, schrift := farbeGrund, farbeLinie, farbeText
			switch {
			case !inhalt.KanteAn:
				schrift = farbeNebentext
			case f == jetzt:
				grund, rnd = farbeAuswahl, farbeAuswahl
			case k.kanten[j].Hovered():
				rnd = farbeNebentext
			}
			paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
			paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
			e.textMittig(gtx, kantenName(f), 12, schrift, r)
			if inhalt.KanteAn {
				seiteKnopf(gtx, &k.kanten[j], r)
			}
		}
		y += hoch + gtx.Dp(10)
	}
	if inhalt.Strich {
		hoch, kb := gtx.Dp(26), gtx.Dp(30)
		e.seiteText(gtx, i18n.T("Line width"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
		wert := image.Rect(x+spalte+kb, y, x+spalte+kb+gtx.Dp(64), y+hoch)
		for j, dir := range []int{-1, 1} {
			r := image.Rect(x+spalte, y, x+spalte+kb, y+hoch)
			zeichen := "−"
			if dir > 0 {
				r = image.Rect(wert.Max.X, y, wert.Max.X+kb, y+hoch)
				zeichen = "+"
			}
			k.strRahmen[j] = r
			_, an := e.s.BreiteStufe(dir)
			rnd, schrift := farbeLinie, farbeText
			switch {
			case !an:
				schrift = farbeNebentext
			case k.strich[j].Hovered():
				rnd = farbeNebentext
			}
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
			paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
			e.textMittig(gtx, zeichen, 14, schrift, r)
			if an {
				seiteKnopf(gtx, &k.strich[j], r)
			}
		}
		e.textMittig(gtx, fmt.Sprintf(i18n.TK("status bar", "%d px"), int(math.Round(e.s.Breite()))), 13, farbeText, wert)
		y += hoch + gtx.Dp(10)
	}
	if t := seitenHinweisText(inhalt.Hinweis); t != "" {
		y += e.seiteText(gtx, t, 12, farbeNebentext, image.Pt(x, y), breite, 4)
	}

	if !inhalt.Farbe {
		return
	}
	trenne()
	k.farbeGezeigt = true
	y += e.seiteText(gtx, strings.ToUpper(i18n.T("Color")), 12, farbeNebentext, image.Pt(x, y), breite, 1) + gtx.Dp(12)
	q, abst := gtx.Dp(28), gtx.Dp(8)
	for j, c := range canvas.Beschriftungsfarben {
		r := image.Rect(x+j*(q+abst), y, x+j*(q+abst)+q, y+q)
		k.farbRahmen[j] = r
		if c == e.s.Stil.Farbe { // gewählt: grüner Rahmen außen (Grün heißt aktiv, Orange bleibt dem Fokus)
			paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r.Inset(-gtx.Dp(2))).Op())
		} else if k.farben[j].Hovered() {
			paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(r.Inset(-gtx.Dp(1))).Op())
		}
		paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
		seiteKnopf(gtx, &k.farben[j], r)
	}
	y += q + gtx.Dp(12)
	f := e.s.Stil.Farbe
	e.seiteText(gtx, i18n.T("Hex"), 13, farbeNebentext, image.Pt(x, y), spalte, 1)
	e.seiteText(gtx, fmt.Sprintf("#%02X%02X%02X", f.R, f.G, f.B), 13, farbeText, image.Pt(x+gtx.Dp(40), y), spalte, 1)
	e.seiteText(gtx, i18n.T("Opacity"), 13, farbeNebentext, image.Pt(x+spalte+gtx.Dp(20), y), spalte, 1)
	e.seiteText(gtx, fmt.Sprintf(i18n.TK("status bar", "%d%%"), canvas.Deckkraft(f)), 13, farbeText,
		image.Pt(x+breite-gtx.Dp(44), y), gtx.Dp(44), 1)
}

// textMittig setzt text mittig in r.
func (e *editor) textMittig(gtx layout.Context, text string, sp float32, c color.NRGBA, r image.Rectangle) {
	lbl := material.Label(e.thema, unit.Sp(sp), text)
	lbl.Color = c
	lbl.MaxLines = 1
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	m := op.Record(gtx.Ops)
	d := lbl.Layout(g)
	call := m.Stop()
	t := op.Offset(r.Min.Add(r.Size().Sub(d.Size).Div(2))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}
