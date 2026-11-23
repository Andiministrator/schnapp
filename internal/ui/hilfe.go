// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Übersicht der Tastenkürzel (Menü Hilfe, F1; ADR-0031 Punkt 1). Sie entsteht aus denselben Quellen wie Menü und
// Werkzeugleiste (Tastentabelle, Menüeinträge), damit sie nie von ihnen abweicht.

import (
	"image"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Version ist die Version des Programms für „Über Schnapp“; cmd/schnapp setzt sie beim Start.
var Version = "0.0.0-dev"

// hilfeZeile ist eine Zeile der Übersicht: Handlung und Taste; ueberschrift: Menütitel ohne Taste.
type hilfeZeile struct {
	text, taste  string
	ueberschrift bool
}

// hilfeZeilen sammelt alle Kürzel: je Menü die Einträge mit Taste, die Werkzeuge mit Buchstaben, dazu die Tasten ohne
// Menüeintrag (Esc, F10).
func (e *editor) hilfeZeilen() []hilfeZeile {
	var z []hilfeZeile
	for i := 0; i < menueAnzahl; i++ {
		var teil []hilfeZeile
		for _, en := range e.menueEintraegeVon(i) {
			k := menueKuerzel(en)
			if !en.hatWerkzeug && en.akt != aktTextBearbeiten {
				k = strings.Join(alleKuerzel(en.akt), ", ") // auch die zweite Taste (Strg+Y, Rücktaste)
			}
			if k != "" {
				teil = append(teil, hilfeZeile{text: en.text, taste: k})
			}
		}
		if len(teil) > 0 {
			z = append(z, hilfeZeile{text: menueTitel(i), ueberschrift: true})
			z = append(z, teil...)
		}
	}
	return append(z,
		hilfeZeile{text: i18n.T("Other keys"), ueberschrift: true},
		hilfeZeile{text: i18n.T("Menu bar"), taste: "F10"},
		hilfeZeile{text: i18n.T("Cancel, deselect, back to Select"), taste: "Esc"},
		hilfeZeile{text: i18n.T("New line in a text"), taste: i18n.T("Shift") + "+" + i18n.T("Enter")},
	)
}

// zeichneHilfe legt die Übersicht über die Leinwand; ein Klick irgendwohin oder Esc schließt sie.
func (e *editor) zeichneHilfe(gtx layout.Context) {
	if !e.hilfeOffen {
		return
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &e.hilfeOffen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if _, ok := ev.(pointer.Event); ok {
			e.hilfeOffen = false
			return
		}
	}
	flaeche := leinwandFlaeche(gtx)
	ha := clip.Rect(flaeche).Push(gtx.Ops)
	event.Op(gtx.Ops, &e.hilfeOffen)
	ha.Pop()

	zeilen := e.hilfeZeilen()
	hoehe, innen := gtx.Dp(22), gtx.Dp(16)
	spalteText, spalteTaste := gtx.Dp(300), gtx.Dp(170)
	spalteB := spalteText + spalteTaste + innen
	// Zwei Spalten, wenn eine nicht in die Leinwand passt.
	pro := len(zeilen)
	spalten := 1
	if innen*2+pro*hoehe > flaeche.Dy() {
		spalten, pro = 2, (len(zeilen)+1)/2
	}
	groesse := image.Pt(innen*2+spalten*spalteB, min(flaeche.Dy(), innen*2+pro*hoehe+gtx.Dp(30)))
	ecke := flaeche.Min.Add(flaeche.Size().Sub(groesse).Div(2))
	t := op.Offset(ecke).Push(gtx.Ops)
	r := image.Rectangle{Max: groesse}
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	schreibe := func(s string, x, y int, sp unit.Sp, c, fett bool) {
		lbl := material.Label(e.thema, sp, s)
		lbl.Color = farbeText
		if c {
			lbl.Color = farbeNebentext
		}
		if fett {
			lbl.Color = farbeGruenText
		}
		lbl.MaxLines = 1
		g := gtx
		g.Constraints.Min = image.Point{}
		g.Constraints.Max.X = spalteText - gtx.Dp(12)
		m := op.Record(gtx.Ops)
		lbl.Layout(g)
		call := m.Stop()
		tt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
	schreibe(i18n.T("Keyboard shortcuts")+" · "+i18n.T("click or Esc to close"), innen, innen/2, unit.Sp(12), true, false)
	for i, z := range zeilen {
		sp, zeile := i/pro, i%pro
		x := innen + sp*spalteB
		y := innen + gtx.Dp(14) + zeile*hoehe
		if z.ueberschrift {
			schreibe(z.text, x, y, unit.Sp(13), false, true)
			continue
		}
		schreibe(z.text, x+gtx.Dp(8), y, unit.Sp(13), false, false)
		schreibe(z.taste, x+spalteText, y, unit.Sp(13), true, false)
	}
	t.Pop()
}
