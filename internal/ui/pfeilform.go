// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// pfeilformLeiste sind die Zeilen „Head“ und „Ends“ der rechten Leiste (ADR-0021 Nachtrag 19): je vier Symbolknöpfe,
// die Spitzenart und Enden des gewählten Pfeils bzw. folgender Pfeile setzen. Jeder Knopf zeigt einen kleinen Pfeil aus
// derselben Zerlegung wie Leinwand und Export.
type pfeilformLeiste struct {
	kopf      [4]widget.Clickable
	enden     [4]widget.Clickable
	kopfRahm  [4]image.Rectangle // Lagen beim letzten Zeichnen (Inhaltskoordinaten der Leiste), leer: nicht gezeigt
	endenRahm [4]image.Rectangle
	hinweis   string // Name des Knopfs unter dem Zeiger oder mit Fokus, für die Statuszeile
}

// kopfName ist der Name einer Spitzenart für Hinweis und Statuszeile.
func kopfName(k document.Kopfart) string {
	switch k {
	case document.KopfWinkel:
		return i18n.T("Open head")
	case document.KopfPunkt:
		return i18n.T("Dot")
	case document.KopfQuer:
		return i18n.T("Bar")
	}
	return i18n.T("Filled triangle")
}

// endenName ist der Name einer Wahl der Enden.
func endenName(e document.PfeilEnden) string {
	switch e {
	case document.EndenKeine:
		return i18n.T("No head (line)")
	case document.EndenAnfang:
		return i18n.T("Head at the start")
	case document.EndenBeide:
		return i18n.T("Heads at both ends")
	}
	return i18n.T("Head at the end")
}

// pfeilformAuswerten verarbeitet Klicks; während des Speicherns ruht das Bearbeiten (seiteAn).
func (e *editor) pfeilformAuswerten(gtx layout.Context) {
	k := &e.pfeilform
	for j, art := range document.Kopfarten {
		if k.kopf[j].Clicked(gtx) && e.seiteAn() {
			e.s.SetzeKopf(art)
		}
	}
	for j, en := range document.PfeilEndenAlle {
		if k.enden[j].Clicked(gtx) && e.seiteAn() {
			e.s.SetzeEnden(en)
		}
	}
}

// zeichnePfeilform zeichnet die beiden Zeilen bei y und liefert das neue y.
func (e *editor) zeichnePfeilform(gtx layout.Context, x, y, breite, spalte int) int {
	k := &e.pfeilform
	kopf, enden := e.s.Pfeilform()
	hoch, luecke := gtx.Dp(26), gtx.Dp(4)
	kb := (breite - spalte - 3*luecke) / 4
	zeile := func(titel string, knoepfe []widget.Clickable, rahmen []image.Rectangle, n int, gewaehlt func(int) bool,
		name func(int) string, muster func(int) document.Pfeil) {
		e.seiteText(gtx, titel, 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
		for j := range n {
			r := image.Rect(x+spalte+j*(kb+luecke), y, x+spalte+j*(kb+luecke)+kb, y+hoch)
			rahmen[j] = r
			grund, rnd, tinte := farbeGrund, farbeLinie, farbeText
			switch {
			case gewaehlt(j):
				grund, rnd = farbeAuswahl, farbeAuswahl
			case knoepfe[j].Hovered():
				rnd = farbeNebentext
			}
			paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
			paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
			zeichnePfeilMuster(gtx, muster(j), r.Inset(gtx.Dp(5)), tinte)
			e.knopfAn(gtx, &knoepfe[j], r)
			if knoepfe[j].Hovered() || gtx.Focused(&knoepfe[j]) {
				k.hinweis = name(j)
			}
		}
		y += hoch + gtx.Dp(8)
	}
	zeile(i18n.T("Head"), k.kopf[:], k.kopfRahm[:], len(document.Kopfarten),
		func(j int) bool { return document.Kopfarten[j] == kopf },
		func(j int) string { return kopfName(document.Kopfarten[j]) },
		func(j int) document.Pfeil {
			return document.Pfeil{Kopf: document.Kopfarten[j], Enden: document.EndenEnde}
		})
	zeile(i18n.T("Ends"), k.enden[:], k.endenRahm[:], len(document.PfeilEndenAlle),
		func(j int) bool { return document.PfeilEndenAlle[j] == enden },
		func(j int) string { return endenName(document.PfeilEndenAlle[j]) },
		func(j int) document.Pfeil {
			return document.Pfeil{Kopf: kopf, Enden: document.PfeilEndenAlle[j]}
		})
	return y + gtx.Dp(2)
}

// zeichnePfeilMuster zeichnet einen waagerechten Pfeil mit Kopf und Enden von p in r (Strich 2 dp).
func zeichnePfeilMuster(gtx layout.Context, p document.Pfeil, r image.Rectangle, farbe color.NRGBA) {
	w := float64(gtx.Dp(2))
	ym := float64(r.Min.Y+r.Max.Y) / 2
	a, b := document.Pt(float64(r.Min.X), ym), document.Pt(float64(r.Max.X), ym)
	// Die Köpfe ragen nach außen nicht über die Enden hinaus, außer Punkt und Balken: dort um ihr Maß einrücken.
	rein := 0.0
	switch p.Kopf {
	case document.KopfPunkt:
		rein = 1.5*w + 3
	case document.KopfQuer:
		rein = w / 2
	}
	if p.Enden != document.EndenKeine && p.Enden != document.EndenAnfang {
		b.X -= rein
	}
	if p.Enden == document.EndenAnfang || p.Enden == document.EndenBeide {
		a.X += rein
	}
	pf := document.GeraderPfeil(a, b, document.Strich{Farbe: farbe, Breite: w})
	pf.Kopf, pf.Enden = p.Kopf, p.Enden
	pz := render.ZerlegePfeil(pf)
	an := func(q document.Punkt) f32.Point { return f32.Pt(float32(q.X), float32(q.Y)) }
	fuelleTeile(gtx, append(pz.Teile, pz.Einzeln...), farbe, an)
}
