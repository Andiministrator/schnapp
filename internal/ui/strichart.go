// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// strichartLeiste ist das Klappfeld „Line style“ der rechten Leiste (ADR-0021 Nachtrag 25 Punkt 7) unter dem Regler der
// Strichstärke: eine Vorschau der Strichart und ihr Name. Ein Klick (Enter, Leertaste) öffnet die Liste aller Arten über
// die Menü-Mechanik (listeStrichart), jede mit ihrer Vorschau, wie die Felder des Pfeil-Stils (pfeilform.go).
type strichartLeiste struct {
	feld    widget.Clickable
	rahm    image.Rectangle // Lage beim letzten Zeichnen (Inhaltskoordinaten der Leiste), leer: nicht gezeigt
	hinweis string          // Text für die Statuszeile, wenn das Feld unter dem Zeiger liegt oder den Fokus hat
}

// alle ist das Feld, wenn es gezeigt wird (fokussierterKnopf).
func (k *strichartLeiste) alle() []*widget.Clickable {
	if k.rahm.Empty() {
		return nil
	}
	return []*widget.Clickable{&k.feld}
}

// strichartName ist der Name einer Strichart für Feld, Liste und Statuszeile.
func strichartName(a document.Strichart) string {
	switch a {
	case document.StrichGestrichelt:
		return i18n.TK("line style", "Dashed")
	case document.StrichGepunktet:
		return i18n.TK("line style", "Dotted")
	}
	return i18n.TK("line style", "Solid")
}

// strichartListe ist die Liste der Stricharten (menueEintraegeVon).
func (e *editor) strichartListe() []menueEintrag {
	jetzt := e.s.Strichart()
	var en []menueEintrag
	for _, art := range document.Stricharten {
		en = append(en, menueEintrag{hatStrichart: true, strichart: art, text: strichartName(art), an: true,
			aktiv: art == jetzt})
	}
	return en
}

// strichartListeOeffnen öffnet die Liste am Feld; per Tastatur steht der Fokus auf der gewählten Art.
func (e *editor) strichartListeOeffnen(perTaste bool) {
	e.menueOeffne(listeStrichart, perTaste)
	if !perTaste || !e.menue.offen {
		return
	}
	for i, en := range e.menueEintraege() {
		if en.aktiv {
			e.menue.cursor, e.menue.fokusNach = i, &e.menue.eintraege[i]
		}
	}
}

// strichartAuswerten verarbeitet Klicks auf das Feld; während des Speicherns ruht das Bearbeiten (seiteAn).
func (e *editor) strichartAuswerten(gtx layout.Context) {
	if e.strichart.feld.Clicked(gtx) && e.seiteAn() {
		e.strichartListeOeffnen(false)
	}
}

// zeichneStrichart zeichnet Beschriftung und Feld bei y und liefert das neue y.
func (e *editor) zeichneStrichart(gtx layout.Context, x, y, breite, spalte int) int {
	k := &e.strichart
	art := e.s.Strichart()
	hoch := gtx.Dp(28)
	e.seiteText(gtx, i18n.T("Line style"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(5)), spalte-gtx.Dp(4), 1)
	r := image.Rect(x+spalte, y, x+breite, y+hoch)
	k.rahm = r
	rnd := farbeLinie
	switch {
	case e.menue.offen && e.menue.welches == listeStrichart:
		rnd = farbeAuswahl
	case k.feld.Hovered():
		rnd = farbeNebentext
	}
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	// Vorschau links, Name daneben, Aufklapp-Pfeil rechts (wie die Felder des Pfeil-Stils).
	vorschau := image.Rect(r.Min.X+gtx.Dp(6), r.Min.Y+gtx.Dp(4), r.Min.X+gtx.Dp(50), r.Max.Y-gtx.Dp(4))
	zeichneStrichMuster(gtx, art, vorschau, farbeText)
	textX := vorschau.Max.X + gtx.Dp(8)
	name := strichartName(art)
	if e.s.Gemischt(canvas.EigStrichart) {
		name = gemischtText()
	}
	e.seiteText(gtx, name, 13, farbeText, image.Pt(textX, y+gtx.Dp(5)), r.Max.X-textX-gtx.Dp(18), 1)
	zeichneAufklappPfeil(gtx, image.Pt(r.Max.X-gtx.Dp(10), (r.Min.Y+r.Max.Y)/2), farbeNebentext)
	e.knopfAn(gtx, &k.feld, r)
	if k.feld.Hovered() || gtx.Focused(&k.feld) {
		k.hinweis = i18n.T("Line style") + ": " + name
	}
	return y + hoch + gtx.Dp(10)
}

// zeichneStrichMuster zeichnet in r eine waagerechte Linie in der Strichart art (Strich 2 dp), aus derselben Zerlegung
// wie Leinwand und Export (render.ZerlegeStrich).
func zeichneStrichMuster(gtx layout.Context, art document.Strichart, r image.Rectangle, farbe color.NRGBA) {
	w := float64(gtx.Dp(2))
	ym := float64(r.Min.Y+r.Max.Y) / 2
	a, b := document.Pt(float64(r.Min.X)+w/2, ym), document.Pt(float64(r.Max.X)-w/2, ym)
	sz := render.ZerlegeStrich([]document.Punkt{a, b}, document.Strich{Farbe: farbe, Breite: w, Art: art})
	fuelleTeile(gtx, sz.Teile, farbe, punktF32)
}
