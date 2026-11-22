// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Sperrzeit: so lange nach dem ersten Bild nehmen die Knöpfe keinen Klick an. Der Dialog erscheint, ohne dass der
// Mensch ihn erwartet; ein Klick, der eigentlich einem anderen Fenster galt, darf nicht „Erlauben“ treffen.
const bestaetigenSperre = 600 * time.Millisecond

// Bestaetige zeigt den Bestätigungsdialog für eine Aufnahme über den MCP-Server (ADR-0030 Nachtrag 2 Punkt 4) und
// kehrt mit der Antwort zurück: true nur nach einem Klick auf „Allow“. Esc, Schließen des Fensters und jeder Fehler
// heißen Ablehnen; Enter löst nichts aus. ziel beschreibt, was aufgenommen werden soll (fertig übersetzt), kunde ist
// der vom Agenten gemeldete Name (ungeprüft, schon maskiert und gekürzt; leer = unbekannt). Muss neben app.Main laufen
// (Starte). Eine Frist setzt der Aufrufer von außen (er beendet den Prozess).
func Bestaetige(ziel, kunde string) (bool, error) {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette.Bg = farbeGrund
	th.Palette.Fg = farbeText
	th.Palette.ContrastBg = farbeAuswahl
	th.Palette.ContrastFg = farbeText

	w := new(app.Window)
	w.Option(app.Title(i18n.T("Schnapp — agent request")), app.Size(unit.Dp(480), unit.Dp(200)),
		app.MinSize(unit.Dp(380), unit.Dp(180)))
	var (
		ops                op.Ops
		erlauben, ablehnen widget.Clickable
		antwort            bool
		fertig             bool
		erstesBild         time.Time
	)
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return antwort, e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if fertig {
				e.Frame(gtx.Ops)
				w.Perform(system.ActionClose) // in der eigenen Schleife (F-77)
				continue
			}
			if erstesBild.IsZero() {
				erstesBild = e.Now
			}
			gesperrt := e.Now.Sub(erstesBild) < bestaetigenSperre
			if gesperrt {
				gtx.Execute(op.InvalidateCmd{At: erstesBild.Add(bestaetigenSperre)})
			}
			for {
				ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
				if !ok {
					break
				}
				if k, ok := ev.(key.Event); ok && k.State == key.Press {
					fertig = true
				}
			}
			if erlauben.Clicked(gtx) && !gesperrt {
				antwort, fertig = true, true
			}
			if ablehnen.Clicked(gtx) {
				fertig = true
			}
			if fertig {
				gtx.Execute(op.InvalidateCmd{})
			}
			zeichneBestaetigung(gtx, th, ziel, kunde, &erlauben, &ablehnen, gesperrt)
			e.Frame(gtx.Ops)
		}
	}
}

func zeichneBestaetigung(gtx layout.Context, th *material.Theme, ziel, kunde string, erlauben, ablehnen *widget.Clickable, gesperrt bool) {
	paint.Fill(gtx.Ops, farbeGrund)
	knopf := func(k *widget.Clickable, text string, akzent bool) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			b := material.Button(th, k, text)
			b.CornerRadius = 0 // eckig (Karte #186)
			b.Background = farbeLeiste
			if akzent {
				b.Background = farbeAuswahl
				if gesperrt {
					b.Background = farbeLinie
				}
			}
			return b.Layout(gtx)
		}
	}
	wer := kunde
	if wer == "" {
		wer = i18n.T("unknown")
	}
	layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body1(th, i18n.T("An agent wants to capture:")+" "+ziel)
				l.Font.Weight = font.Bold
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Caption(th, i18n.T("Name reported by the agent (unverified):")+" "+wer)
				l.Color = farbeAkzent
				return l.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Min.X, Y: gtx.Constraints.Min.Y}}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Spacing: layout.SpaceStart}.Layout(gtx,
					layout.Rigid(knopf(ablehnen, i18n.T("Deny"), false)),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(knopf(erlauben, i18n.T("Allow"), true)),
				)
			}),
		)
	})
}
