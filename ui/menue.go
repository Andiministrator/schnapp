// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/i18n"
)

// menueEintrag ist eine Zeile des Menüs (ADR-0026 Punkt 2).
type menueEintrag struct {
	akt    aktion
	text   string
	an     bool
	gruppe bool // beginnt eine neue Gruppe (Trennlinie davor)
}

// menueEintraege liefert die Einträge in Reihenfolge, mit dem Zustand der Sitzung (an oder grau).
func (e *editor) menueEintraege() []menueEintrag {
	s := e.s
	return []menueEintrag{
		{akt: aktKopieren, text: i18n.T("Copy"), an: true},
		{akt: aktExport, text: i18n.T("Export…"), an: !e.exportOffen},
		{akt: aktSpeichern, text: i18n.T("Save"), an: true},
		{akt: aktRueckgaengig, text: i18n.T("Undo"), an: s.D.KannRueckgaengig(), gruppe: true},
		{akt: aktWiederholen, text: i18n.T("Redo"), an: s.D.KannWiederholen()},
		{akt: aktTextBearbeiten, text: i18n.T("Edit text"), an: s.KannTextBearbeiten(), gruppe: true},
		{akt: aktLoeschen, text: i18n.T("Delete"), an: s.KannLoeschen()},
		{akt: aktGanzesBild, text: i18n.T("Whole image"), an: s.KannGanzesBild(), gruppe: true},
		{akt: aktZurueckholen, text: i18n.T("Restore cut-out areas"), an: s.KannZurueckholen()},
		{akt: aktEinpassen, text: i18n.T("Fit"), an: true, gruppe: true},
		{akt: aktEins, text: "1:1", an: true},
		{akt: aktSchliessen, text: i18n.T("Close"), an: true, gruppe: true},
	}
}

// menueKuerzel ist das angezeigte Kürzel eines Eintrags; „Text bearbeiten“ geht mit Enter.
func menueKuerzel(a aktion) string {
	if a == aktTextBearbeiten {
		return kuerzelText(aktEnter)
	}
	return kuerzelText(a)
}

// menue ist der Zustand des Menüs.
type menue struct {
	knopf     widget.Clickable
	offen     bool
	eintraege [16]widget.Clickable // je Eintrag einer; mehr als genug für menueEintraege
	aussen    int                  // Ziel für Klicks neben die offene Liste (schließt sie)
}

// fuehreAus führt eine Handlung aus, gleich ob per Taste, Knopf oder Menü. vonMaus: Ein laufender Text wird vorher
// bestätigt (wie die Knöpfe); per Taste wirken Kopieren und Export in der Texteingabe nicht (wie bisher).
func (e *editor) fuehreAus(gtx layout.Context, a aktion, vonMaus bool) {
	if vonMaus {
		e.s.TextBestaetigen()
	}
	if w := e.wirkung(a); w != nil {
		w(gtx)
	}
}

// aktionAusfuehrbar meldet, ob fuehreAus für a etwas tut (Test: kein Menüeintrag ohne Wirkung, F-253).
func aktionAusfuehrbar(a aktion) bool { return (&editor{}).wirkung(a) != nil }

// wirkung ist die Handlung zu a; nil für aktKeine und die Tasten mit eigener Behandlung (aktEsc, aktEnter).
func (e *editor) wirkung(a aktion) func(gtx layout.Context) {
	switch a {
	case aktKopieren:
		return func(gtx layout.Context) {
			if e.s.KopierenErlaubt() {
				e.kopieren(gtx)
			}
		}
	case aktExport:
		return func(layout.Context) {
			if e.s.KopierenErlaubt() {
				e.exportStarten()
			}
		}
	case aktSpeichern:
		return func(layout.Context) { e.speichernJetzt() }
	case aktRueckgaengig:
		return func(layout.Context) { e.rueckgaengig() }
	case aktWiederholen:
		return func(layout.Context) { e.wiederholen() }
	case aktTextBearbeiten:
		return func(layout.Context) { e.s.TextBearbeiten() }
	case aktLoeschen:
		return func(layout.Context) { e.s.Loeschen() }
	case aktGanzesBild:
		// Zuschnitt aufheben und Randstreifen zurückholen, wie Doppelklick mit Zuschneiden (E-21, F-222).
		return func(layout.Context) {
			if e.s.HebeZuschnittAuf() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktZurueckholen:
		return func(layout.Context) {
			if e.s.HoleGestauchtesZurueck() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktEinpassen:
		return func(layout.Context) { e.s.Einpassen() }
	case aktEins:
		return func(layout.Context) { e.s.EinsZuEins() }
	case aktSchliessen:
		return func(layout.Context) {
			diagnoseEditor("Schließen über Taste oder Menü")
			e.w.Perform(system.ActionClose) // in der eigenen Schleife, wie im Overlay (F-77)
		}
	}
	return nil
}

// menueAuswerten verarbeitet Klicks auf den Menüknopf, die Einträge und daneben.
func (e *editor) menueAuswerten(gtx layout.Context) {
	m := &e.menue
	if m.knopf.Clicked(gtx) {
		m.offen = !m.offen
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.aussen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if p, ok := ev.(pointer.Event); ok && p.Kind == pointer.Press {
			m.offen = false
		}
	}
	for { // Klicks auf die Fläche zwischen den Einträgen: abholen und nichts tun
		if _, ok := gtx.Event(pointer.Filter{Target: m, Kinds: pointer.Press}); !ok {
			break
		}
	}
	for i, en := range e.menueEintraege() {
		if m.eintraege[i].Clicked(gtx) && m.offen && en.an {
			m.offen = false
			e.fuehreAus(gtx, en.akt, true)
		}
	}
}

// zeichneMenue legt die offene Liste unter den Menüknopf, rechtsbündig an rechts (Fensterpixel), oben bei y. Darunter
// liegt eine Fläche über dem ganzen Fenster, die einen Klick daneben abfängt.
func (e *editor) zeichneMenue(gtx layout.Context, rechts, y int) {
	m := &e.menue
	if !m.offen {
		return
	}
	mx := gtx.Constraints.Max
	ha := clip.Rect(image.Rectangle{Max: mx}).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.aussen)
	ha.Pop()

	eintraege := e.menueEintraege()
	zeile, innen, trenn := gtx.Dp(32), gtx.Dp(14), gtx.Dp(9)
	// Breite: längster Text plus längstes Kürzel, gemessen ohne Zeichnen.
	mess := func(s string) int {
		if s == "" {
			return 0
		}
		r := op.Record(gtx.Ops)
		g := gtx
		g.Constraints.Min = image.Point{}
		d := material.Label(e.thema, unit.Sp(14), s).Layout(g)
		r.Stop()
		return d.Size.X
	}
	textB, kuerzelB, hoehe := 0, 0, gtx.Dp(6)*2
	for _, en := range eintraege {
		textB = max(textB, mess(en.text))
		kuerzelB = max(kuerzelB, mess(menueKuerzel(en.akt)))
		hoehe += zeile
		if en.gruppe {
			hoehe += trenn
		}
	}
	breite := innen + textB + gtx.Dp(32) + kuerzelB + innen
	// Passt die Liste nicht unter den Knopf, rückt sie nach oben (über die Leiste); ist das Fenster niedriger als die
	// Liste, fallen die Abstände der Gruppen weg und die Zeilen werden niedriger, bis zur Höhe einer Textzeile
	// (F-252: bei der Mindesthöhe 320 dp wären sonst die unteren Einträge nicht erreichbar).
	if y+hoehe > mx.Y {
		y = max(0, mx.Y-hoehe)
	}
	if hoehe > mx.Y {
		trenn = 0
		zeile = max(gtx.Dp(20), (mx.Y-gtx.Dp(6)*2)/len(eintraege))
		hoehe = gtx.Dp(6)*2 + zeile*len(eintraege)
		y = 0
	}
	ecke := image.Pt(max(0, rechts-breite), y)
	t := op.Offset(ecke).Push(gtx.Ops)
	flaeche := image.Rectangle{Max: image.Pt(breite, hoehe)}
	rr := clip.UniformRRect(flaeche, gtx.Dp(6))
	paint.FillShape(gtx.Ops, farbeLeisteVoll, rr.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	// Die Fläche selbst fängt Klicks zwischen den Einträgen ab, damit sie nicht als „daneben“ gelten.
	hf := clip.Rect(flaeche).Push(gtx.Ops)
	event.Op(gtx.Ops, m)
	hf.Pop()
	oy := gtx.Dp(6)
	for i, en := range eintraege {
		if en.gruppe && i > 0 {
			ly := oy + trenn/2
			paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(innen/2, ly, breite-innen/2, ly+gtx.Dp(1))).Op())
		}
		if en.gruppe {
			oy += trenn
		}
		e.menueZeile(gtx, &m.eintraege[i], en, image.Pt(0, oy), image.Pt(breite, zeile), innen)
		oy += zeile
	}
	t.Pop()
}

// menueZeile zeichnet einen Eintrag: Text links, Kürzel rechts, grau wenn nicht bedienbar.
func (e *editor) menueZeile(gtx layout.Context, c *widget.Clickable, en menueEintrag, ecke, groesse image.Point, innen int) {
	t := op.Offset(ecke).Push(gtx.Ops)
	if en.an && (c.Hovered() || gtx.Focused(c)) {
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(image.Rectangle{Max: groesse}).Op())
	}
	farbe, neben := farbeText, farbeNebentext
	if !en.an {
		farbe = farbeNebentext
	}
	g := gtx
	g.Constraints.Min = image.Point{}
	schreibe := func(s string, c color.NRGBA, rechtsbuendig bool) {
		r := op.Record(gtx.Ops)
		lbl := material.Label(e.thema, unit.Sp(14), s)
		lbl.Color = c
		d := lbl.Layout(g)
		call := r.Stop()
		x := innen
		if rechtsbuendig {
			x = groesse.X - innen - d.Size.X
		}
		tt := op.Offset(image.Pt(x, (groesse.Y-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
	schreibe(en.text, farbe, false)
	if k := menueKuerzel(en.akt); k != "" {
		schreibe(k, neben, true)
	}
	if en.an {
		g.Constraints = layout.Exact(groesse)
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: groesse} })
		pointer.CursorPointer.Add(gtx.Ops)
	}
	t.Pop()
}

// zeichneWerkzeugHinweis zeigt unter dem Werkzeugknopf, über dem der Zeiger steht, Namen und Kürzel (ADR-0026 Punkt 3).
func (e *editor) zeichneWerkzeugHinweis(gtx layout.Context, links []leistenKnopf, pos []image.Point) {
	if e.menue.offen {
		return
	}
	for _, w := range e.werkzeugKnoepfe() {
		if !w.c.Hovered() {
			continue
		}
		for i, l := range links {
			if l.c != w.c {
				continue
			}
			text := w.text + " (" + werkzeugKuerzel(w.w) + ")"
			r := op.Record(gtx.Ops)
			g := gtx
			g.Constraints.Min = image.Point{}
			lbl := material.Label(e.thema, unit.Sp(12), text)
			lbl.Color = farbeText
			d := lbl.Layout(g)
			call := r.Stop()
			rand := gtx.Dp(6)
			ecke := pos[i].Add(image.Pt(0, gtx.Dp(36)+gtx.Dp(4)))
			ecke.X = max(0, min(ecke.X, gtx.Constraints.Max.X-d.Size.X-2*rand))
			t := op.Offset(ecke).Push(gtx.Ops)
			fl := image.Rectangle{Max: d.Size.Add(image.Pt(2*rand, rand))}
			paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.UniformRRect(fl, gtx.Dp(4)).Op(gtx.Ops))
			paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.UniformRRect(fl, gtx.Dp(4)).Path(gtx.Ops),
				Width: float32(gtx.Dp(1))}.Op())
			tt := op.Offset(image.Pt(rand, rand/2)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			tt.Pop()
			t.Pop()
		}
	}
}
