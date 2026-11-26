// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Abschnitt „Shadow“ der rechten Leiste (ADR-0021 Nachtrag 26 Punkt 7, Karte #363 Teil 2): Schalter an/aus, bei an
// „Direction“ (− / Wert / + in 15°-Schritten), „Distance“ und „Softness“ als Regler (0–40 bzw. 0–20 px mal Faktor,
// Ziehen ein Rückgängig-Schritt wie am Regler der Strichstärke) und „Color“, ein Farbfeld, das den Schatten zum aktiven
// Feld des Farbwählers macht. Die Logik steht in canvas (schatten.go); hier Lage, Zeichnen und Klicks.

import (
	"fmt"
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Regler des Abschnitts (schattenLeiste.zieht).
const (
	schattenReglerKeiner = iota
	schattenReglerLaenge
	schattenReglerWeich
)

// schattenLeiste hält Knöpfe und Lagen des Abschnitts; die Rahmen sind Inhaltskoordinaten der Leiste beim letzten
// Zeichnen (leer: nicht gezeigt), für Tests und den Zeiger.
type schattenLeiste struct {
	schalter widget.Clickable
	richtung [2]widget.Clickable // − und +
	// stufe sind − und + neben den Reglern: [0] Länge, [1] Weich (Tastaturweg, Kritik K-4 #363b).
	stufe [2][2]widget.Clickable
	farbe widget.Clickable

	schalterRahmen image.Rectangle
	richtRahmen    [2]image.Rectangle
	stufeRahmen    [2][2]image.Rectangle
	farbRahmen     image.Rectangle
	// laenge und weich sind die Regler, zugleich Ziele ihrer Zeigerereignisse.
	laenge, weich image.Rectangle
	zieht         int    // schattenRegler…: an diesem Regler wird gezogen
	richtText     string // angezeigte Richtung beim letzten Zeichnen (Tests)
}

// alle sind die gezeigten Knöpfe (fokussierterKnopf).
func (k *schattenLeiste) alle() []*widget.Clickable {
	if k.schalterRahmen.Empty() {
		return nil
	}
	a := []*widget.Clickable{&k.schalter}
	if !k.farbRahmen.Empty() {
		a = append(a, &k.richtung[0], &k.richtung[1], &k.stufe[0][0], &k.stufe[0][1], &k.stufe[1][0], &k.stufe[1][1], &k.farbe)
	}
	return a
}

// leeren vergisst die Lagen vor dem Zeichnen (zeichneSeitenleiste).
func (k *schattenLeiste) leeren() {
	k.schalterRahmen, k.farbRahmen, k.laenge, k.weich = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
	k.richtRahmen, k.stufeRahmen, k.richtText = [2]image.Rectangle{}, [2][2]image.Rectangle{}, ""
}

// schattenAuswerten verarbeitet Klicks und Ziehen des Abschnitts; während des Speicherns ruht das Bearbeiten.
func (e *editor) schattenAuswerten(gtx layout.Context) {
	k := &e.schattenL
	if k.schalter.Clicked(gtx) && e.seiteAn() {
		e.s.SchalteSchatten()
	}
	for j, dir := range []int{-1, 1} {
		if k.richtung[j].Clicked(gtx) && e.seiteAn() {
			e.s.SchattenRichtungStufe(dir)
		}
	}
	for i := range k.stufe {
		for j, dir := range []int{-1, 1} {
			if k.stufe[i][j].Clicked(gtx) && e.seiteAn() {
				e.s.SchattenWertStufe(i == 1, dir)
			}
		}
	}
	if k.farbe.Clicked(gtx) && e.seiteAn() && e.s.SetzeFeld(canvas.FeldSchatten) {
		// Der Farbwähler wirkt jetzt auf den Schatten: Abschnitt „Farbe“ aufklappen und ins Bild rollen, wie ein Klick
		// aufs Farbpaar links (F-817).
		e.seite.farbeZu, e.ebenen.vorrang, e.seite.farbeZeigen = false, false, true
		gtx.Execute(op.InvalidateCmd{})
	}
	e.schattenReglerAuswerten(gtx, &k.laenge, schattenReglerLaenge)
	e.schattenReglerAuswerten(gtx, &k.weich, schattenReglerWeich)
}

// schattenReglerAuswerten verarbeitet Drücken, Ziehen, Loslassen und Abbruch an einem Regler (wie strichReglerAuswerten).
func (e *editor) schattenReglerAuswerten(gtx layout.Context, ziel *image.Rectangle, art int) {
	k := &e.schattenL
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: ziel, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			return
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch pe.Kind {
		case pointer.Press:
			if pe.Buttons != pointer.ButtonPrimary || k.zieht != schattenReglerKeiner || !e.seiteAn() {
				continue
			}
			if e.s.BeginneSchattenZug() {
				k.zieht = art
				e.schattenZiehen(art, pe.Position)
			}
		case pointer.Drag, pointer.Release:
			if k.zieht != art {
				continue
			}
			e.schattenZiehen(art, pe.Position)
			if pe.Kind == pointer.Release {
				e.s.BeendeBreiteZug()
				k.zieht = schattenReglerKeiner
			}
		case pointer.Cancel:
			if k.zieht == art {
				e.s.BrecheBreiteZugAb()
				k.zieht = schattenReglerKeiner
			}
		}
	}
}

// schattenZiehen setzt Länge bzw. Weich aus der Lage p (relativ zum Regler).
func (e *editor) schattenZiehen(art int, p f32.Point) {
	k := &e.schattenL
	r := k.laenge
	if art == schattenReglerWeich {
		r = k.weich
	}
	if r.Dx() <= 0 {
		return
	}
	laengeBis, weichBis := e.s.SchattenReglerMax()
	sch := e.s.Schatten()
	lage := float64(p.X) / float64(r.Dx())
	if art == schattenReglerWeich {
		sch.Weich = canvas.SchattenAusRegler(lage, weichBis)
	} else {
		sch.Laenge = canvas.SchattenAusRegler(lage, laengeBis)
	}
	e.s.SchattenZugSetze(sch)
}

// zeichneSchattenLeiste zeichnet den Abschnitt ab y und liefert das neue y.
func (e *editor) zeichneSchattenLeiste(gtx layout.Context, x, y, breite, spalte int) int {
	k := &e.schattenL
	sch := e.s.Schatten()
	hoch := gtx.Dp(26)
	beschrift := func(text string, y int) {
		e.seiteText(gtx, text, 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	}

	// Schalter (wie „Tail“, spitze.go)
	beschrift(i18n.TK("shadow", "Shadow"), y)
	r := image.Rect(x+spalte, y, x+spalte+hoch, y+hoch)
	k.schalterRahmen = r
	rnd := farbeLinie
	if k.schalter.Hovered() {
		rnd = farbeNebentext
	}
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	if sch.An {
		paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r.Inset(gtx.Dp(6))).Op())
	}
	e.knopfAn(gtx, &k.schalter, r)
	y += hoch + gtx.Dp(10)
	if !sch.An {
		return y
	}

	// Richtung − / Wert / +
	beschrift(i18n.TK("shadow", "Direction"), y)
	kb := gtx.Dp(30)
	wert := image.Rect(x+spalte+kb, y, x+spalte+kb+gtx.Dp(64), y+hoch)
	for j := range k.richtung {
		r := image.Rect(x+spalte, y, x+spalte+kb, y+hoch)
		zeichen := "−"
		if j == 1 {
			r = image.Rect(wert.Max.X, y, wert.Max.X+kb, y+hoch)
			zeichen = "+"
		}
		k.richtRahmen[j] = r
		rnd := farbeLinie
		if k.richtung[j].Hovered() {
			rnd = farbeNebentext
		}
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
		e.textMittig(gtx, zeichen, 14, farbeText, r)
		e.knopfAn(gtx, &k.richtung[j], r)
	}
	// Gerundet modulo 360: 359,6° aus einer Datei hieße sonst „360°“ (Kritik K-6 #363b).
	k.richtText = fmt.Sprintf(i18n.TK("shadow", "%d°"), int(math.Round(sch.Winkel))%360)
	e.textMittig(gtx, k.richtText, 13, farbeText, wert)
	y += hoch + gtx.Dp(10)

	// Regler Länge und Weich, rechts daneben der Wert
	laengeBis, weichBis := e.s.SchattenReglerMax()
	y = e.zeichneSchattenRegler(gtx, &k.laenge, 0, i18n.TK("shadow", "Distance"), sch.Laenge, laengeBis, x, y, breite, spalte)
	y = e.zeichneSchattenRegler(gtx, &k.weich, 1, i18n.TK("shadow", "Softness"), sch.Weich, weichBis, x, y, breite, spalte)

	// Farbfeld
	beschrift(i18n.TK("shadow", "Color"), y)
	r = image.Rect(x+spalte, y, x+spalte+hoch, y+hoch)
	k.farbRahmen = r
	e.farbFeld(gtx, e.s.Farbe(canvas.FeldSchatten), r, e.s.AktivesFeld() == canvas.FeldSchatten, k.farbe.Hovered())
	e.knopfAn(gtx, &k.farbe, r)
	return y + hoch + gtx.Dp(10)
}

// zeichneSchattenRegler zeichnet eine Zeile mit Beschriftung, − , Regler (Wert w von 0 bis bis), + und Wert in px;
// meldet den Regler unter ziel und die Knöpfe stufe[nr] an und liefert das neue y.
func (e *editor) zeichneSchattenRegler(gtx layout.Context, ziel *image.Rectangle, nr int, text string, w, bis float64, x, y, breite, spalte int) int {
	k := &e.schattenL
	hoch, schiene, wertBreite, kb := gtx.Dp(26), gtx.Dp(4), gtx.Dp(44), gtx.Dp(22)
	e.seiteText(gtx, text, 13, farbeNebentext, image.Pt(x, y+gtx.Dp(4)), spalte-gtx.Dp(4), 1)
	minus := image.Rect(x+spalte, y, x+spalte+kb, y+hoch)
	plus := image.Rect(x+breite-wertBreite-kb, y, x+breite-wertBreite, y+hoch)
	for j, r := range []image.Rectangle{minus, plus} {
		k.stufeRahmen[nr][j] = r
		rnd := farbeLinie
		if k.stufe[nr][j].Hovered() {
			rnd = farbeNebentext
		}
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
		e.textMittig(gtx, [2]string{"−", "+"}[j], 14, farbeText, r)
		e.knopfAn(gtx, &k.stufe[nr][j], r)
	}
	links, rechts := minus.Max.X+gtx.Dp(6), plus.Min.X-gtx.Dp(6)
	*ziel = image.Rect(links, y+gtx.Dp(5), rechts, y+hoch-gtx.Dp(5))
	mitte := y + hoch/2
	px := links + int(math.Round(canvas.SchattenReglerLage(w, bis)*float64(rechts-links)))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(links, mitte-schiene/2, rechts, mitte+schiene/2)).Op())
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(image.Rect(links, mitte-schiene/2, px, mitte+schiene/2)).Op())
	b := gtx.Dp(3)
	marke(gtx, image.Rect(max(links, px-b), ziel.Min.Y+gtx.Dp(1), min(rechts, px+b), ziel.Max.Y-gtx.Dp(1)))
	reglerAnmelden(gtx, ziel, *ziel)
	wert := fmt.Sprintf(i18n.TK("status bar", "%d px"), int(math.Round(w)))
	e.textMittig(gtx, wert, 13, farbeText, image.Rect(x+breite-wertBreite, y, x+breite, y+hoch))
	return y + hoch + gtx.Dp(6)
}
