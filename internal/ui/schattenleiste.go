// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Abschnitt „Shadow“ der rechten Leiste (ADR-0021 Nachtrag 26 Punkt 7, Karte #363 Teil 2; neu gestaltet nach ADR-0031
// Nachtrag 13, Karte #369): Kästchen „Shadow“ an/aus, bei an „Direction“ als Scheibe (Klicken oder Ziehen setzt die
// Fallrichtung zum Zeiger, 15°-Raster, Umschalt frei; Pfeile ±15° mit Fokus), „Distance“ und „Softness“ als Regler
// (0–40 bzw. 0–20 px mal Faktor, Ziehen ein Rückgängig-Schritt wie am Regler der Strichstärke) und „Color“, ein
// Farbfeld, das den Reiter „Shadow“ des Farbwählers wählt. Die Logik steht in canvas (schatten.go); hier Lage, Zeichnen
// und Klicks.

import (
	"fmt"
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Regler des Abschnitts (schattenLeiste.zieht).
const (
	schattenReglerKeiner = iota
	schattenReglerLaenge
	schattenReglerWeich
	schattenReglerRichtung // die Scheibe
)

// scheibeDp ist der Durchmesser der Richtungsscheibe.
const scheibeDp = 56

// schattenLeiste hält Knöpfe und Lagen des Abschnitts; die Rahmen sind Inhaltskoordinaten der Leiste beim letzten
// Zeichnen (leer: nicht gezeigt), für Tests und den Zeiger.
type schattenLeiste struct {
	schalter widget.Clickable
	scheibe  widget.Clickable // Tastaturfokus der Richtungsscheibe (Pfeile ±15°)
	// stufe sind − und + neben den Reglern: [0] Länge, [1] Weich (Tastaturweg, Kritik K-4 #363b).
	stufe [2][2]widget.Clickable
	farbe widget.Clickable

	schalterRahmen image.Rectangle
	scheibeRahmen  image.Rectangle // Richtungsscheibe, zugleich Ziel ihrer Zeigerereignisse
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
		a = append(a, &k.scheibe, &k.stufe[0][0], &k.stufe[0][1], &k.stufe[1][0], &k.stufe[1][1], &k.farbe)
	}
	return a
}

// leeren vergisst die Lagen vor dem Zeichnen (zeichneSeitenleiste).
func (k *schattenLeiste) leeren() {
	k.schalterRahmen, k.farbRahmen, k.laenge, k.weich = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
	k.scheibeRahmen, k.stufeRahmen, k.richtText = image.Rectangle{}, [2][2]image.Rectangle{}, ""
}

// schattenAuswerten verarbeitet Klicks und Ziehen des Abschnitts; während des Speicherns ruht das Bearbeiten.
func (e *editor) schattenAuswerten(gtx layout.Context) {
	k := &e.schattenL
	if k.schalter.Clicked(gtx) && e.seiteAn() {
		e.s.SchalteSchatten()
	}
	for i := range k.stufe {
		for j, dir := range []int{-1, 1} {
			if k.stufe[i][j].Clicked(gtx) && e.seiteAn() {
				e.s.SchattenWertStufe(i == 1, dir)
			}
		}
	}
	if k.farbe.Clicked(gtx) && e.seiteAn() && e.s.SetzeFeld(canvas.FeldSchatten) {
		// Der Farbwähler wirkt jetzt auf den Schatten (Reiter „Shadow“): Abschnitt „Farbe“ aufklappen und ins Bild
		// rollen, wie ein Klick aufs Farbpaar links (F-817).
		e.seite.farbeZu, e.ebenen.vorrang, e.seite.farbeZeigen = false, false, true
		gtx.Execute(op.InvalidateCmd{})
	}
	e.schattenReglerAuswerten(gtx, &k.laenge, schattenReglerLaenge)
	e.schattenReglerAuswerten(gtx, &k.weich, schattenReglerWeich)
	e.schattenReglerAuswerten(gtx, &k.scheibeRahmen, schattenReglerRichtung)
}

// scheibeTaste: Pfeile an der Richtungsscheibe mit Fokus drehen den Winkel um 15° (rechts und hoch erhöhen, links und
// runter verringern, wie die früheren Knöpfe + und −). Liefert, ob die Taste verbraucht ist.
func (e *editor) scheibeTaste(gtx layout.Context, taste key.Event) bool {
	k := &e.schattenL
	if k.scheibeRahmen.Empty() || !gtx.Focused(&k.scheibe) || taste.Modifiers != 0 {
		return false
	}
	switch taste.Name {
	case key.NameRightArrow, key.NameUpArrow:
		e.s.SchattenRichtungStufe(1)
	case key.NameLeftArrow, key.NameDownArrow:
		e.s.SchattenRichtungStufe(-1)
	default:
		return false
	}
	return true
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
				e.schattenZiehen(art, pe.Position, pe.Modifiers.Contain(key.ModShift))
			}
		case pointer.Drag, pointer.Release:
			if k.zieht != art {
				continue
			}
			e.schattenZiehen(art, pe.Position, pe.Modifiers.Contain(key.ModShift))
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

// schattenZiehen setzt Länge bzw. Weich aus der Lage p (relativ zum Regler), an der Scheibe die Richtung (frei:
// Umschalt gehalten, ganze Grad statt 15°-Raster).
func (e *editor) schattenZiehen(art int, p f32.Point, frei bool) {
	k := &e.schattenL
	if art == schattenReglerRichtung {
		m := float64(k.scheibeRahmen.Dx()) / 2
		if w, ok := canvas.SchattenWinkelZu(float64(p.X)-m, float64(p.Y)-m, frei); ok {
			sch := e.s.Schatten()
			sch.Winkel = w
			e.s.SchattenZugSetze(sch)
		}
		return
	}
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

	// Kästchen mit Wort, wie „Show agent captures“ der Bibliothek (ADR-0031 Nachtrag 13 Punkt 2): Kästchen und Wort
	// sind eine Klickfläche.
	q := gtx.Dp(18)
	kr := image.Rect(x, y+(hoch-q)/2, x+q, y+(hoch-q)/2+q)
	zeichneKaestchen(gtx, kr, sch.An)
	e.seiteText(gtx, i18n.TK("shadow", "Shadow"), 13, farbeText, image.Pt(kr.Max.X+gtx.Dp(8), y+gtx.Dp(4)),
		breite-q-gtx.Dp(8), 1)
	r := image.Rect(x, y, x+spalte+gtx.Dp(40), y+hoch)
	k.schalterRahmen = r
	e.knopfAn(gtx, &k.schalter, r)
	y += hoch + gtx.Dp(10)
	if !sch.An {
		return y
	}

	// Richtung als Scheibe, rechts daneben der Lichtwinkel als Zahl (ADR-0031 Nachtrag 13 Punkt 3).
	beschrift(i18n.TK("shadow", "Direction"), y)
	d := gtx.Dp(scheibeDp)
	sr := image.Rect(x+spalte, y, x+spalte+d, y+d)
	k.scheibeRahmen = sr
	e.zeichneScheibe(gtx, sr, sch.Winkel, gtx.Focused(&k.scheibe))
	e.knopfAn(gtx, &k.scheibe, sr)
	reglerAnmelden(gtx, &k.scheibeRahmen, sr)
	// Gerundet modulo 360: 359,6° aus einer Datei hieße sonst „360°“ (Kritik K-6 #363b).
	k.richtText = fmt.Sprintf(i18n.TK("shadow", "%d°"), int(math.Round(sch.Winkel))%360)
	e.seiteText(gtx, k.richtText, 13, farbeText, image.Pt(sr.Max.X+gtx.Dp(12), y+(d-gtx.Dp(16))/2), gtx.Dp(60), 1)
	y += d + gtx.Dp(10)

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

// zeichneScheibe zeichnet die Richtungsscheibe in r: Kreis mit Marken alle 45°, in der Mitte ein kleines Quadrat mit
// seinem Schatten in Fallrichtung (Versatz des Lichtwinkels w) und ein Punkt am Rand in derselben Richtung.
func (e *editor) zeichneScheibe(gtx layout.Context, r image.Rectangle, w float64, fokus bool) {
	m := f32.Pt(float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+r.Dy()/2))
	rad := float32(r.Dx()) / 2
	kreis := func(c f32.Point, rd float32) clip.Ellipse {
		return clip.Ellipse{Min: c.Sub(f32.Pt(rd, rd)).Round(), Max: c.Add(f32.Pt(rd, rd)).Round()}
	}
	paint.FillShape(gtx.Ops, farbeGrund, kreis(m, rad).Op(gtx.Ops))
	rnd := farbeLinie
	if fokus {
		rnd = farbeAkzent
	}
	paint.FillShape(gtx.Ops, rnd, clip.Stroke{Path: kreis(m, rad-0.5).Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	for i := 0; i < 8; i++ { // Marken alle 45°
		a := float64(i) * math.Pi / 4
		v := f32.Pt(float32(math.Cos(a)), float32(math.Sin(a)))
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(m.Add(v.Mul(rad - float32(gtx.Dp(5)))))
		p.LineTo(m.Add(v.Mul(rad - float32(gtx.Dp(2)))))
		paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(1))}.Op())
	}
	v := document.Schatten{An: true, Winkel: w, Laenge: 1}.Versatz()
	richt := f32.Pt(float32(v.X), float32(v.Y))
	h := float32(gtx.Dp(7))
	quadrat := func(c f32.Point) image.Rectangle {
		return image.Rectangle{Min: c.Sub(f32.Pt(h, h)).Round(), Max: c.Add(f32.Pt(h, h)).Round()}
	}
	schatten := farbeNebentext
	schatten.A = 140
	paint.FillShape(gtx.Ops, schatten, clip.Rect(quadrat(m.Add(richt.Mul(float32(gtx.Dp(6)))))).Op())
	paint.FillShape(gtx.Ops, farbeText, clip.Rect(quadrat(m)).Op())
	paint.FillShape(gtx.Ops, farbeAkzent, kreis(m.Add(richt.Mul(rad-float32(gtx.Dp(8)))), float32(gtx.Dp(3))).Op(gtx.Ops))
}
