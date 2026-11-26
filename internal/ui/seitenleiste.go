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

// Höhen für das Einklappen der Farbe in niedrigen Fenstern (dp): der Farbwähler (zeichneFarbwaehler: Fläche 114,
// zwei Regler, Felder) und der kleinste Ebenen-Abschnitt ohne Hinweis (zeichneEbenen: Trennlinie, Kopf, Deckkraft,
// eine Zeile, Knöpfe, drei Zeilen Hinweis).
const (
	farbwaehlerHoeheDp = 234 // bis unter Hex und Deckkraft, seit #219 mit der Zeile des Farbpaars (24 + 10); die Farbfelder darunter rechnet farbfelderHoehe (#364)
	ebenenMindestDp    = 230
)

// kantenFormen sind die Formen der Stauch-Kante in der Reihenfolge der Knöpfe.
var kantenFormen = []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken}

// seitenKnoepfe sind die Knöpfe der rechten Leiste; die Rahmen halten ihre Lage beim letzten Zeichnen (Tests).
type seitenKnoepfe struct {
	farben     [farbPlaetze]widget.Clickable // zuletzt benutzte Farben (ADR-0031 Nachtrag 5, seit Karte #364 nur diese)
	ff         farbfeldLeiste                // eigene Farbfelder unter Hex und Deckkraft (Karte #364)
	kanten     [3]widget.Clickable           // je Kantenform einer (kantenFormen)
	strich     [2]widget.Clickable           // Strichstärke − und + (ADR-0021 Nachtrag 7)
	strRahmen  [2]image.Rectangle
	strichText string // angezeigte Strichstärke beim letzten Zeichnen (Tests)
	// strRegler ist der Regler der Strichstärke (Karte #352), zugleich Ziel seiner Zeigerereignisse; strZieht: dort
	// wird gezogen.
	strRegler    image.Rectangle
	strZieht     bool
	farbRahmen   [farbPlaetze]image.Rectangle
	fw           farbwaehler
	sw           schriftwahl
	kantRahmen   [3]image.Rectangle
	kopf         string           // Überschrift oben beim letzten Zeichnen (Tests)
	farbeGezeigt bool             // der Abschnitt Farbe stand beim letzten Zeichnen da
	farbeKopf    widget.Clickable // Überschrift „Farbe“: klappt den Abschnitt ein und aus (ADR-0032 Punkt 6)
	farbeZu      bool
	farbeZeigen  bool // nächstes Bild: den Abschnitt Farbe ins Bild rollen (Klick aufs Farbpaar links, F-817)
	// Farbpaar über dem Farbwähler (ADR-0021 Nachtrag 11): Vordergrund, Füllung, Tauschen, Keine; Rahmen für Tests.
	paar       [4]widget.Clickable
	paarRahmen [4]image.Rectangle
	roll       rollZustand // Rollen der Leiste bei wenig Höhe (ADR-0031 Nachtrag 8, F-598)
}

// Plätze in seitenKnoepfe.paar.
const (
	paarVorn = iota
	paarFuellung
	paarTausch
	paarKeine
)

// alle sind die Knöpfe der Leiste (Tastaturfokus, editor.fokussierterKnopf).
func (k *seitenKnoepfe) alle() []*widget.Clickable {
	var a []*widget.Clickable
	for i := range k.paar {
		a = append(a, &k.paar[i])
	}
	for i := range k.farben {
		a = append(a, &k.farben[i])
	}
	for i := range k.kanten {
		a = append(a, &k.kanten[i])
	}
	a = append(a, k.sw.knoepfe()...)
	a = append(a, k.ff.alle()...)
	return append(a, &k.strich[0], &k.strich[1], &k.fw.pipette, &k.farbeKopf)
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
//
// Liegt die Übersicht der Tastenkürzel offen, wirken die Klicks nicht: Die Tafel verdeckt das Objekt, das sie ändern
// würden (wie die Tasten, Sitzungs-QA A-8).
func (e *editor) seiteAuswerten(gtx layout.Context) {
	an := e.seiteAn
	k := &e.seite
	farben := e.s.Zuletzt()
	for j := range k.farben {
		if k.farben[j].Clicked(gtx) && an() && j < len(farben) {
			e.s.SetzeFarbe(farben[j])
		}
	}
	e.farbfelderAuswerten(gtx)
	if k.paar[paarVorn].Clicked(gtx) && an() {
		e.s.SetzeFeld(canvas.FeldVordergrund)
	}
	if k.paar[paarFuellung].Clicked(gtx) && an() {
		e.s.SetzeFeld(canvas.FeldFuellung)
	}
	if k.paar[paarTausch].Clicked(gtx) && an() {
		e.s.TauscheFarben()
	}
	if k.paar[paarKeine].Clicked(gtx) && an() && e.s.SetzeFeld(canvas.FeldFuellung) {
		e.s.KeineFuellung()
	}
	if k.farbeKopf.Clicked(gtx) {
		if e.ebenen.vorrang && !k.farbeZu { // bei Platzmangel eingeklappt: Farbe öffnen, Ebenen wieder einklappen
			e.ebenen.vorrang = false
		} else {
			k.farbeZu = !k.farbeZu
		}
	}
	e.rollAuswerten(gtx)
	e.schnellAuswerten(gtx)
	e.farbeAuswerten(gtx)
	e.schriftAuswerten(gtx)
	e.ebenenAuswerten(gtx)
	e.lupeAuswerten(gtx)
	e.spitzeAuswerten(gtx)
	e.pfeilformAuswerten(gtx)
	for j, f := range kantenFormen {
		if k.kanten[j].Clicked(gtx) && an() && e.s.SetzeKante(f) {
			e.melde(kantenMeldung(f), false)
		}
	}
	e.strichReglerAuswerten(gtx)
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
	handZeiger(gtx, r.Size())
	t.Pop()
}

// handZeiger setzt die Zeigehand für die Fläche (0,0)–groesse am aktuellen Ursprung. Die Op braucht einen eigenen
// Ausschnitt: Nach Clickable.Layout ist dessen Ausschnitt schon abgelegt, ein bloßes pointer.CursorPointer.Add gälte dann
// für den umgebenden Bereich — bis Karte #342 lag so die Hand über der ganzen Leinwand und verdeckte die Zeigerformen
// der Griffe (F-818).
func handZeiger(gtx layout.Context, groesse image.Point) {
	c := clip.Rect(image.Rectangle{Max: groesse}).Push(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	c.Pop()
}

// zeichneSeitenleiste zeichnet die rechte Leiste, wenn sie sichtbar ist (seiteSichtbar).
func (e *editor) zeichneSeitenleiste(gtx layout.Context) {
	k := &e.seite
	k.kopf, k.farbeGezeigt, k.sw.gezeigt = "", false, false
	e.spitze.rahmen = image.Rectangle{}
	e.pfeilform.rahm, e.pfeilform.hinweis = [2]image.Rectangle{}, ""
	e.schnell.gezeigt, e.schnell.hinweis = false, ""
	k.farbRahmen, k.kantRahmen, k.strRahmen = [farbPlaetze]image.Rectangle{}, [3]image.Rectangle{}, [2]image.Rectangle{}
	k.strRegler = image.Rectangle{}
	k.ff.rahmen, k.ff.plusRahmen, k.ff.hinweis = [len(k.ff.rahmen)]image.Rectangle{}, image.Rectangle{}, ""
	k.fw.flaeche, k.fw.ton, k.fw.deck, k.fw.pipRahmen = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
	k.fw.hex.rahmen, k.fw.deckkraft.rahmen = image.Rectangle{}, image.Rectangle{}
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
	e.rollAnmelden(gtx, image.Rect(x0, oben, mx.X, unten))
	versatz := op.Offset(image.Pt(0, -k.roll.rollen)).Push(gtx.Ops)

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

	if art, ok := e.s.SchnellArt(); ok { // Schnellstile über den Eigenschaften (ADR-0037 Punkt 2)
		y = e.zeichneSchnellstile(gtx, art, x, y, breite)
		trenne()
	}
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
			// Auch grau angemeldet, wie die Stufenknöpfe: Der Fokus fällt nicht ins Leere, wenn die letzte Naht
			// wegfällt; ein Klick tut dann nichts, weil SetzeKante ohne Naht nichts setzt (Sitzungs-QA N-9).
			e.knopfAn(gtx, &k.kanten[j], r)
		}
		y += hoch + gtx.Dp(10)
	}
	if inhalt.Text { // Schrift, Größe, Stil, Ausrichtung (ADR-0031 Nachtrag 6)
		y = e.zeichneSchriftwahl(gtx, x, y, breite, spalte) + gtx.Dp(10)
	}
	if inhalt.Spitze { // Schalter „Spitze“ (ADR-0021 Nachtrag 13 Punkt 6)
		y = e.zeichneSpitze(gtx, x, y, spalte)
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
			// Auch grau angemeldet: Sonst fiele der Tastaturfokus nach dem Schritt auf die letzte Stufe ins Leere
			// (Sitzungs-QA A-5); ein Klick dort tut nichts, weil BreiteStufe keine Stufe mehr findet.
			e.knopfAn(gtx, &k.strich[j], r)
		}
		k.strichText = fmt.Sprintf(i18n.TK("status bar", "%d px"), int(math.Round(e.s.Breite())))
		e.textMittig(gtx, k.strichText, 13, farbeText, wert)
		y += hoch + gtx.Dp(6)
		y = e.zeichneStrichRegler(gtx, x, y, breite) + gtx.Dp(10)
	}
	if inhalt.Pfeilform { // Spitzenart und Enden (ADR-0021 Nachtrag 19)
		y = e.zeichnePfeilform(gtx, x, y, breite, spalte)
	}
	if inhalt.Vergroesserung { // gewählte Lupe (ADR-0033 Punkt 8)
		y = e.zeichneVergroesserung(gtx, x, y, spalte)
	}
	if t := seitenHinweisText(inhalt.Hinweis); t != "" {
		y += e.seiteText(gtx, t, 12, farbeNebentext, image.Pt(x, y), breite, 4)
	}

	// Reicht die Leiste nicht für Farbwähler und den kleinsten Ebenen-Abschnitt (Verify V23-2, V23-3), klappen die
	// Ebenen in diesem Bild ein — die Farbe braucht man öfter; ein Klick auf „Ebenen“ gibt ihnen den Vorrang.
	eng := inhalt.Farbe && !k.farbeZu && !e.ebenen.zu &&
		unten-y-gtx.Dp(30) < gtx.Dp(unit.Dp(farbwaehlerHoeheDp+ebenenMindestDp))+e.farbfelderHoehe(gtx, breite)
	if !eng {
		e.ebenen.vorrang = false
	}
	e.ebenen.autoZu = eng && !e.ebenen.vorrang
	if inhalt.Farbe {
		trenne()
		zu := k.farbeZu || (eng && e.ebenen.vorrang)
		kopfY := y
		y += e.abschnittKopf(gtx, &k.farbeKopf, i18n.T("Color"), zu, x, y, breite) + gtx.Dp(12)
		if !zu {
			k.farbeGezeigt = true
			y = e.zeichneFarbwaehler(gtx, x, y, breite, spalte)
			if k.farbeZeigen {
				k.roll.ziel = image.Rect(x, kopfY, x+breite, y)
			}
		}
	}
	k.farbeZeigen = false
	y = e.zeichneEbenen(gtx, x, y, breite, spalte, unten) // mit eigener Trennlinie, am unteren Rand (ADR-0032 Punkt 6)
	versatz.Pop()
	e.rollAbschluss(gtx, y)
	e.zeichneRollbalken(gtx)
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
