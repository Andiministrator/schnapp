// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"context"
	"image"
	"image/color"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/gesture"
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

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Sperrzeit: so lange nach dem ersten Bild nimmt „Allow“ keinen Klick an. Der Dialog erscheint, ohne dass der Mensch
// ihn erwartet; ein Klick, der eigentlich einem anderen Fenster galt, darf nicht „Erlauben“ treffen.
const bestaetigenSperre = 600 * time.Millisecond

// Bestaetige zeigt den Bestätigungsdialog für eine Aufnahme über den MCP-Server (ADR-0030 Nachtrag 2 Punkt 4,
// Nachtrag 4 Punkt 4) und kehrt mit der Antwort zurück: true nur nach einem Mausklick auf „Allow“, dessen Drücken und
// Loslassen nach der Sperre liegen. „Allow“ ist kein fokussierbares Element: Return, Leertaste und Tab erreichen es
// nie. Esc, „Deny“ (auch per Tastatur), Schließen des Fensters, das Ende von ctx und jeder Fehler heißen Ablehnen.
// ziel beschreibt, was aufgenommen werden soll (fertig übersetzt und maskiert), kunde ist der vom Agenten gemeldete
// Name (ungeprüft, schon maskiert und gekürzt; leer = unbekannt). Muss neben app.Main laufen (Starte).
func Bestaetige(ctx context.Context, ziel, kunde string) (bool, error) {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette.Bg = farbeGrund
	th.Palette.Fg = farbeText
	th.Palette.ContrastBg = farbeAuswahl
	th.Palette.ContrastFg = farbeText

	w := new(app.Window)
	w.Option(app.Title(i18n.T("Schnapp — agent request")), app.Size(unit.Dp(480), unit.Dp(200)),
		app.MinSize(unit.Dp(380), unit.Dp(180)))
	// Endet ctx (Frist des Kindes), zeichnet das Fenster neu und schließt sich dabei (F-469).
	stop := context.AfterFunc(ctx, w.Invalidate)
	defer stop()
	var (
		ops      op.Ops
		erlauben gesture.Click
		ablehnen widget.Clickable
		f        freigabe
		antwort  bool
		fertig   bool
	)
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return antwort, e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if ctx.Err() != nil && !fertig {
				antwort, fertig = false, true
			}
			if fertig {
				e.Frame(gtx.Ops)
				w.Perform(system.ActionClose) // in der eigenen Schleife (F-77)
				continue
			}
			f.bild(e.Now)
			gesperrt := f.gesperrt(e.Now)
			if gesperrt {
				gtx.Execute(op.InvalidateCmd{At: f.sperreBis})
			}
			for {
				ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
				if !ok {
					break
				}
				if k, ok := ev.(key.Event); ok && k.State == key.Press && f.taste(k.Name) {
					fertig = true
				}
			}
			for {
				k, ok := erlauben.Update(gtx.Source)
				if !ok {
					break
				}
				switch k.Kind {
				case gesture.KindPress:
					f.druck(e.Now)
				case gesture.KindCancel:
					f.abbruch()
				case gesture.KindClick:
					if f.klick(e.Now) {
						antwort, fertig = true, true
					}
				}
			}
			if ablehnen.Clicked(gtx) {
				antwort, fertig = false, true
			}
			if fertig {
				gtx.Execute(op.InvalidateCmd{})
			}
			zeichneBestaetigung(gtx, th, ziel, kunde, &erlauben, &ablehnen, gesperrt)
			e.Frame(gtx.Ops)
		}
	}
}

func zeichneBestaetigung(gtx layout.Context, th *material.Theme, ziel, kunde string, erlauben *gesture.Click, ablehnen *widget.Clickable, gesperrt bool) {
	paint.Fill(gtx.Ops, farbeGrund)
	wer := kunde
	if wer == "" {
		wer = i18n.T("unknown")
	}
	ablehnenKnopf := func(gtx layout.Context) layout.Dimensions {
		b := material.Button(th, ablehnen, i18n.T("Deny"))
		b.CornerRadius = 0 // eckig (Karte #186)
		b.Background = farbeLeiste
		return b.Layout(gtx)
	}
	erlaubenKnopf := func(gtx layout.Context) layout.Dimensions {
		grund := farbeAuswahl
		if gesperrt {
			grund = farbeLinie
		}
		return knopfNurMaus(gtx, th, erlauben, i18n.T("Allow"), grund)
	}
	layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body1(th, i18n.T("An agent wants to capture:")+" "+ziel)
				l.Font.Weight = font.Bold
				l.MaxLines = 3
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Caption(th, i18n.T("Name reported by the agent (unverified):")+" "+wer)
				l.Color = farbeAkzent
				l.MaxLines = 2
				return l.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Min.X, Y: gtx.Constraints.Min.Y}}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Spacing: layout.SpaceStart}.Layout(gtx,
					layout.Rigid(ablehnenKnopf),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(erlaubenKnopf),
				)
			}),
		)
	})
}

// knopfNurMaus zeichnet einen eckigen Knopf wie material.Button, der nur Zeigerereignisse annimmt (gesture.Click):
// kein Fokus, keine Taste (F-464).
func knopfNurMaus(gtx layout.Context, th *material.Theme, k *gesture.Click, beschriftung string, grund color.NRGBA) layout.Dimensions {
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, grund)
			k.Add(gtx.Ops)
			pointer.CursorPointer.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, th.TextSize, beschriftung)
					l.Color = th.Palette.ContrastFg
					l.Alignment = text.Middle
					l.MaxLines = 1
					return l.Layout(gtx)
				})
		}),
	)
}
