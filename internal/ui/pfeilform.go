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

// pfeilformLeiste ist der Abschnitt „Arrow style“ der rechten Leiste (ADR-0021 Nachträge 19 und 20): je Ende ein
// Klappfeld mit einem kleinen Pfeil als Vorschau und dem Namen der Spitze. Ein Klick (Enter, Leertaste) öffnet die
// Liste aller Spitzenarten über die Menü-Mechanik (listeKopfAnfang, listeKopfEnde), jede mit ihrer Vorschau. Die Wahl
// gilt für den gewählten Pfeil bzw. folgende Pfeile. Jede Vorschau kommt aus derselben Zerlegung wie Leinwand und
// Export.
type pfeilformLeiste struct {
	feld    [2]widget.Clickable // 0: Anfang (bei A), 1: Ende (bei B)
	rahm    [2]image.Rectangle  // Lagen beim letzten Zeichnen (Inhaltskoordinaten der Leiste), leer: nicht gezeigt
	hinweis string              // Text für die Statuszeile zum Feld unter dem Zeiger oder mit Fokus
}

// alle sind die Felder in Tab-Reihenfolge (fokussierterKnopf).
func (k *pfeilformLeiste) alle() []*widget.Clickable {
	var a []*widget.Clickable
	for j := range k.feld {
		if !k.rahm[j].Empty() {
			a = append(a, &k.feld[j])
		}
	}
	return a
}

// kopfName ist der Name einer Spitzenart für Feld, Liste und Statuszeile.
func kopfName(k document.Kopfart) string {
	switch k {
	case document.KopfKeine:
		return i18n.TK("arrow head", "None")
	case document.KopfSchlank:
		return i18n.TK("arrow head", "Narrow")
	case document.KopfKerbe:
		return i18n.TK("arrow head", "Notched")
	case document.KopfWinkel:
		return i18n.TK("arrow head", "Open head")
	case document.KopfHalb:
		return i18n.TK("arrow head", "Half head")
	case document.KopfPunkt:
		return i18n.TK("arrow head", "Dot")
	case document.KopfRaute:
		return i18n.TK("arrow head", "Diamond")
	case document.KopfQuadrat:
		return i18n.TK("arrow head", "Square")
	case document.KopfQuer:
		return i18n.TK("arrow head", "Bar")
	}
	return i18n.TK("arrow head", "Filled triangle")
}

// kopfListe ist die Liste der Spitzenarten für das Klappfeld am Anfang bzw. Ende (menueEintraegeVon).
func (e *editor) kopfListe(amAnfang bool) []menueEintrag {
	anfang, ende := e.s.Pfeilform()
	jetzt := ende
	if amAnfang {
		jetzt = anfang
	}
	var en []menueEintrag
	for _, art := range document.Kopfarten {
		en = append(en, menueEintrag{hatKopf: true, kopf: art, kopfAnfang: amAnfang, text: kopfName(art), an: true,
			aktiv: art == jetzt})
	}
	return en
}

// kopfListeOeffnen öffnet die Liste am Feld j (0 Anfang, 1 Ende); per Tastatur steht der Fokus auf der gewählten Art.
func (e *editor) kopfListeOeffnen(j int, perTaste bool) {
	liste := listeKopfEnde
	if j == 0 {
		liste = listeKopfAnfang
	}
	e.menueOeffne(liste, perTaste)
	if !perTaste || !e.menue.offen {
		return
	}
	for i, en := range e.menueEintraege() {
		if en.aktiv {
			e.menue.cursor, e.menue.fokusNach = i, &e.menue.eintraege[i]
		}
	}
}

// pfeilformAuswerten verarbeitet Klicks auf die Felder; während des Speicherns ruht das Bearbeiten (seiteAn).
func (e *editor) pfeilformAuswerten(gtx layout.Context) {
	k := &e.pfeilform
	for j := range k.feld {
		if k.feld[j].Clicked(gtx) && e.seiteAn() {
			e.kopfListeOeffnen(j, false)
		}
	}
}

// zeichnePfeilform zeichnet Überschrift und die beiden Felder bei y und liefert das neue y.
func (e *editor) zeichnePfeilform(gtx layout.Context, x, y, breite, spalte int) int {
	k := &e.pfeilform
	anfang, ende := e.s.Pfeilform()
	y += e.seiteText(gtx, i18n.T("Arrow style"), 12, farbeNebentext, image.Pt(x, y), breite, 1) + gtx.Dp(4)
	hoch := gtx.Dp(28)
	for j, art := range []document.Kopfart{anfang, ende} {
		titel := i18n.TK("arrow end", "Start")
		if j == 1 {
			titel = i18n.TK("arrow end", "End")
		}
		e.seiteText(gtx, titel, 13, farbeNebentext, image.Pt(x, y+gtx.Dp(5)), spalte-gtx.Dp(4), 1)
		r := image.Rect(x+spalte, y, x+breite, y+hoch)
		k.rahm[j] = r
		rnd := farbeLinie
		offen := e.menue.offen && e.menue.welches == []int{listeKopfAnfang, listeKopfEnde}[j]
		switch {
		case offen:
			rnd = farbeAuswahl
		case k.feld[j].Hovered():
			rnd = farbeNebentext
		}
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
		// Vorschau links, Name daneben, Aufklapp-Pfeil rechts (wie die Felder der Fußzeile).
		vorschau := image.Rect(r.Min.X+gtx.Dp(6), r.Min.Y+gtx.Dp(4), r.Min.X+gtx.Dp(50), r.Max.Y-gtx.Dp(4))
		zeichneKopfMuster(gtx, art, j == 0, vorschau, farbeText)
		textX := vorschau.Max.X + gtx.Dp(8)
		e.seiteText(gtx, kopfName(art), 13, farbeText, image.Pt(textX, y+gtx.Dp(5)), r.Max.X-textX-gtx.Dp(18), 1)
		zeichneAufklappPfeil(gtx, image.Pt(r.Max.X-gtx.Dp(10), (r.Min.Y+r.Max.Y)/2), farbeNebentext)
		e.knopfAn(gtx, &k.feld[j], r)
		if k.feld[j].Hovered() || gtx.Focused(&k.feld[j]) {
			if j == 0 {
				k.hinweis = i18n.T("Arrow style at the start") + ": " + kopfName(art)
			} else {
				k.hinweis = i18n.T("Arrow style at the end") + ": " + kopfName(art)
			}
		}
		y += hoch + gtx.Dp(6)
	}
	return y + gtx.Dp(4)
}

// zeichneKopfMuster zeichnet in r einen waagerechten Pfeil mit der Spitze art am linken (amAnfang) bzw. rechten Ende
// und ohne Spitze am anderen (Strich 2 dp).
func zeichneKopfMuster(gtx layout.Context, art document.Kopfart, amAnfang bool, r image.Rectangle, farbe color.NRGBA) {
	anfang, ende := document.KopfKeine, art
	if amAnfang {
		anfang, ende = art, document.KopfKeine
	}
	zeichnePfeilMuster(gtx, anfang, ende, r, farbe)
}

// zeichnePfeilMuster zeichnet einen waagerechten Pfeil mit den Spitzen anfang (links) und ende (rechts) in r
// (Strich 2 dp).
func zeichnePfeilMuster(gtx layout.Context, anfang, ende document.Kopfart, r image.Rectangle, farbe color.NRGBA) {
	pf := musterPfeil(anfang, ende, r, float64(gtx.Dp(2)), farbe)
	pz := render.ZerlegePfeil(pf)
	an := func(q document.Punkt) f32.Point { return f32.Pt(float32(q.X), float32(q.Y)) }
	fuelleTeile(gtx, append(pz.Teile, pz.Einzeln...), farbe, an)
}

// musterPfeil ist der waagerechte Pfeil der Vorschau in r mit Strichstärke w. Punkt, Quadrat und Balken ragen über das
// Ende hinaus: Das Ende rückt um ihr Maß ein.
func musterPfeil(anfang, ende document.Kopfart, r image.Rectangle, w float64, farbe color.NRGBA) document.Pfeil {
	ym := float64(r.Min.Y+r.Max.Y) / 2
	a := document.Pt(float64(r.Min.X)+document.KopfUeberstand(anfang, w), ym)
	b := document.Pt(float64(r.Max.X)-document.KopfUeberstand(ende, w), ym)
	return document.GeraderPfeil(a, b, document.Strich{Farbe: farbe, Breite: w}).MitKoepfen(anfang, ende)
}
