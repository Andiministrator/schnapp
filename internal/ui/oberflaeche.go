// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Rahmen des Editors nach ADR-0031: Menüzeile oben, Werkzeugleiste links, Fußzeile unten; dazwischen die Leinwand.

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/icons"
)

// Maße des Rahmens in dp (ADR-0031 Punkte 1, 2 und 4).
const (
	menuezeileDp    = 30
	werkzeugeDp     = 52 // Breite der Werkzeugleiste
	werkzeugKnopfDp = 40 // Kantenlänge eines Werkzeugknopfs
	fusszeileDp     = 30
)

var farbeRot = color.NRGBA{R: 0xE6, G: 0x78, B: 0x73, A: 0xFF} // rot-500

// Menüs der Menüzeile in Reihenfolge (ADR-0031 Punkt 1).
const (
	menueDatei = iota
	menueBearbeiten
	menueAnsicht
	menueBild
	menueWerkzeuge
	menueHilfe
	menueAnzahl
)

// menueTitel ist der Text eines Menüs der Zeile.
func menueTitel(i int) string {
	switch i {
	case menueDatei:
		return i18n.T("File")
	case menueBearbeiten:
		return i18n.T("Edit")
	case menueAnsicht:
		return i18n.T("View")
	case menueBild:
		return i18n.T("Image")
	case menueWerkzeuge:
		return i18n.T("Tools")
	case menueHilfe:
		return i18n.T("Help")
	}
	return ""
}

// menueEintraegeVon liefert die Einträge des Menüs i mit dem Zustand der Sitzung (an oder grau).
func (e *editor) menueEintraegeVon(i int) []menueEintrag {
	s := e.s
	switch i {
	case menueDatei:
		return []menueEintrag{
			{akt: aktSpeichern, text: i18n.T("Save"), an: true},
			{akt: aktExport, text: i18n.T("Export…"), an: !e.exportOffen},
			{akt: aktKopieren, text: i18n.T("Copy"), an: true},
			{akt: aktSchliessen, text: i18n.T("Close"), an: true, gruppe: true},
		}
	case menueBearbeiten:
		return []menueEintrag{
			{akt: aktRueckgaengig, text: i18n.T("Undo"), an: s.D.KannRueckgaengig()},
			{akt: aktWiederholen, text: i18n.T("Redo"), an: s.D.KannWiederholen()},
			{akt: aktTextBearbeiten, text: i18n.T("Edit text"), an: s.KannTextBearbeiten(), gruppe: true},
			{akt: aktLoeschen, text: i18n.T("Delete"), an: s.KannLoeschen()},
		}
	case menueAnsicht:
		return []menueEintrag{
			{akt: aktEinpassen, text: i18n.T("Fit"), an: true},
			{akt: aktEins, text: "1:1", an: true},
		}
	case menueBild:
		return []menueEintrag{
			{akt: aktGanzesBild, text: i18n.T("Whole image"), an: s.KannGanzesBild()},
			{akt: aktZurueckholen, text: i18n.T("Restore cut-out areas"), an: s.KannZurueckholen()},
			{akt: aktKantenform, text: kantenMeldung(s.D.Kante()), an: s.KannKanteWechseln(), gruppe: true},
		}
	case menueWerkzeuge:
		var en []menueEintrag
		for _, w := range e.werkzeugKnoepfe() {
			en = append(en, menueEintrag{werkzeug: w.w, hatWerkzeug: true, text: w.text, an: true,
				aktiv: e.s.Werkzeug == w.w})
		}
		return en
	case menueHilfe:
		return []menueEintrag{
			{akt: aktTastenkuerzel, text: i18n.T("Keyboard shortcuts"), an: true},
			{akt: aktUeber, text: i18n.T("About Schnapp"), an: true},
		}
	}
	return nil
}

// leinwandFlaeche ist die Fläche der Leinwand im Fenster (Pixel).
func leinwandFlaeche(gtx layout.Context) image.Rectangle {
	mx := gtx.Constraints.Max
	r := image.Rect(gtx.Dp(werkzeugeDp), gtx.Dp(menuezeileDp), mx.X, mx.Y-gtx.Dp(fusszeileDp))
	if r.Dx() < 1 {
		r.Max.X = r.Min.X + 1
	}
	if r.Dy() < 1 {
		r.Max.Y = r.Min.Y + 1
	}
	return r
}

// --- Menüzeile ---

// zeichneMenuezeile zeichnet die Zeile mit dem „s“, den Menütiteln und dem Dateinamen; die Lage der Titel steht danach
// in e.menue.titelRahmen.
func (e *editor) zeichneMenuezeile(gtx layout.Context) {
	mx := gtx.Constraints.Max
	h := gtx.Dp(menuezeileDp)
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, 0, mx.X, h)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(0, h-gtx.Dp(1), mx.X, h)).Op())
	// Die Zeile fängt Klicks ab, die keinen Titel treffen (schließt eine offene Liste); die Titel darüber, weil sie
	// danach angemeldet werden.
	za := clip.Rect(image.Rect(0, 0, mx.X, h)).Push(gtx.Ops)
	event.Op(gtx.Ops, &e.menue.zeile)
	za.Pop()
	x := gtx.Dp(8)
	x += zeichneLogoS(gtx, image.Pt(x, gtx.Dp(4)), h-gtx.Dp(8)) + gtx.Dp(10)
	for i := 0; i < menueAnzahl; i++ {
		t := menueTitel(i)
		lbl := material.Label(e.thema, unit.Sp(14), t)
		lbl.Color = farbeText
		g := gtx
		g.Constraints.Min = image.Point{}
		m := op.Record(gtx.Ops)
		d := lbl.Layout(g)
		call := m.Stop()
		r := image.Rect(x, 0, x+d.Size.X+gtx.Dp(20), h-gtx.Dp(1))
		e.menue.titelRahmen[i] = r
		c := &e.menue.titel[i]
		if (e.menue.offen && e.menue.welches == i) || c.Hovered() {
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		}
		if gtx.Focused(c) {
			paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(gtx.Dp(2))).Path(),
				Width: float32(gtx.Dp(2))}.Op())
		}
		tt := op.Offset(image.Pt(x+gtx.Dp(10), (h-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
		ft := op.Offset(r.Min).Push(gtx.Ops)
		g.Constraints = layout.Exact(r.Size())
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
		pointer.CursorPointer.Add(gtx.Ops)
		ft.Pop()
		x = r.Max.X
	}
	// Dateiname rechts, gekürzt.
	if e.titel != "" {
		lbl := material.Label(e.thema, unit.Sp(12), e.titel)
		lbl.Color = farbeNebentext
		lbl.MaxLines = 1
		g := gtx
		g.Constraints.Min = image.Point{}
		g.Constraints.Max.X = max(0, mx.X-x-gtx.Dp(24))
		m := op.Record(gtx.Ops)
		d := lbl.Layout(g)
		call := m.Stop()
		tt := op.Offset(image.Pt(mx.X-gtx.Dp(12)-d.Size.X, (h-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
}

// zeichneLogoS zeichnet das „s“ des Logos (Maße aus internal/icons, dort gegen brand/ geprüft) in der Höhe hoehe bei
// ecke und liefert die Breite.
func zeichneLogoS(gtx layout.Context, ecke image.Point, hoehe int) int {
	box := icons.LogoSBox
	k := float32(hoehe) / float32(box[3])
	breite := int(float32(box[2]) * k)
	zu := func(p icons.Punkt) f32.Point {
		return f32.Pt(float32(ecke.X)+float32(p.X-box[0])*k, float32(ecke.Y)+float32(p.Y-box[1])*k)
	}
	linie := icons.LogoSLinie(12)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(zu(linie[0]))
	for _, q := range linie[1:] {
		p.LineTo(zu(q))
	}
	w := float32(icons.LogoSStrich) * k
	paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: p.End(), Width: w}.Op())
	kreis := func(m icons.Punkt, r float32, c color.NRGBA) {
		z := zu(m)
		var k clip.Path
		k.Begin(gtx.Ops)
		k.MoveTo(f32.Pt(z.X+r, z.Y))
		k.Arc(f32.Pt(-r, 0), f32.Pt(-r, 0), 2*math.Pi)
		k.Close()
		paint.FillShape(gtx.Ops, c, clip.Outline{Path: k.End()}.Op())
	}
	kreis(linie[0], w/2, farbeAkzent) // runde Enden
	kreis(linie[len(linie)-1], w/2, farbeAkzent)
	a, pu := icons.LogoSAugapfel, icons.LogoSPupille
	kreis(icons.Punkt{X: a[0], Y: a[1]}, float32(a[2]+1)*k, farbeAkzent) // Rand 2 um den Augapfel
	kreis(icons.Punkt{X: a[0], Y: a[1]}, float32(a[2]-1)*k, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	kreis(icons.Punkt{X: pu[0], Y: pu[1]}, float32(pu[2])*k, farbeFenster)
	return breite
}

// --- Werkzeugleiste ---

// werkzeugSymbol ist das Symbol eines Werkzeugs (Karte #188).
func werkzeugSymbol(w canvas.Werkzeug) icons.Symbol {
	switch w {
	case canvas.WerkzeugPfeil:
		return icons.Pfeil
	case canvas.WerkzeugKasten:
		return icons.Kasten
	case canvas.WerkzeugText:
		return icons.Text
	case canvas.WerkzeugNummer:
		return icons.Nummer
	case canvas.WerkzeugSchwaerzen:
		return icons.Schwaerzen
	case canvas.WerkzeugVerpixeln:
		return icons.Verpixeln
	case canvas.WerkzeugZuschneiden:
		return icons.Zuschneiden
	case canvas.WerkzeugStauchen:
		return icons.Stauchen
	}
	return icons.Auswahl
}

// werkzeugGruppeBeginnt: vor diesen Werkzeugen ein größerer Abstand (ADR-0031 Punkt 2).
func werkzeugGruppeBeginnt(w canvas.Werkzeug) bool {
	return w == canvas.WerkzeugPfeil || w == canvas.WerkzeugSchwaerzen || w == canvas.WerkzeugZuschneiden
}

// werkzeugRahmen liefert die Lage der Werkzeugknöpfe und des Farbfelds (Fensterpixel).
func werkzeugRahmen(gtx layout.Context, ws []werkzeugKnopf) (knoepfe []image.Rectangle, farbe image.Rectangle) {
	x := (gtx.Dp(werkzeugeDp) - gtx.Dp(werkzeugKnopfDp)) / 2
	y := gtx.Dp(menuezeileDp) + gtx.Dp(8)
	k := gtx.Dp(werkzeugKnopfDp)
	for _, w := range ws {
		if werkzeugGruppeBeginnt(w.w) {
			y += gtx.Dp(8)
		}
		knoepfe = append(knoepfe, image.Rect(x, y, x+k, y+k))
		y += k + gtx.Dp(4)
	}
	y += gtx.Dp(12)
	return knoepfe, image.Rect(x, y, x+k, y+k)
}

// zeichneWerkzeugleiste zeichnet die Leiste links: Werkzeuge mit Rand und Grund, das aktive grün, darunter das
// Farbfeld; Hinweis mit Name und Kürzel rechts neben dem Knopf (ADR-0031 Punkt 2).
func (e *editor) zeichneWerkzeugleiste(gtx layout.Context) {
	mx := gtx.Constraints.Max
	b := gtx.Dp(werkzeugeDp)
	oben, unten := gtx.Dp(menuezeileDp), mx.Y-gtx.Dp(fusszeileDp)
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, oben, b, unten)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(b-gtx.Dp(1), oben, b, unten)).Op())
	ws := e.werkzeugKnoepfe()
	rahmen, farbe := werkzeugRahmen(gtx, ws)
	for i, w := range ws {
		e.werkzeugKnopfZeichnen(gtx, w.c, rahmen[i], werkzeugSymbol(w.w), e.s.Werkzeug == w.w)
	}
	e.farbKnopf(gtx, farbe)
	if e.menue.offen {
		return
	}
	for _, w := range e.hinweisKnoepfe(gtx) {
		for i, x := range ws {
			if x.c == w.c {
				e.zeichneHinweis(gtx, werkzeugHinweis(w), image.Pt(rahmen[i].Max.X+gtx.Dp(8),
					rahmen[i].Min.Y+gtx.Dp(8)))
			}
		}
	}
}

// werkzeugHinweis ist der Text des Hinweises am Werkzeugknopf: Name und Kürzel (ADR-0031 Punkt 2).
func werkzeugHinweis(w werkzeugKnopf) string { return w.text + " (" + werkzeugKuerzel(w.w) + ")" }

// werkzeugKnopfZeichnen: eckiger Knopf mit Rand, Grund und Symbol; aktiv grün gefüllt; Fokusring 2 px orange.
func (e *editor) werkzeugKnopfZeichnen(gtx layout.Context, c *widget.Clickable, r image.Rectangle, s icons.Symbol, aktiv bool) {
	grund, rand := farbeGrund, farbeLinie
	switch {
	case aktiv:
		grund, rand = farbeAuswahl, farbeAuswahl
	case c.Hovered():
		rand = farbeNebentext
	}
	paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	if gtx.Focused(c) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(-gtx.Dp(3))).Path(),
			Width: float32(gtx.Dp(2))}.Op())
	}
	seite := gtx.Dp(24)
	ecke := r.Min.Add(image.Pt((r.Dx()-seite)/2, (r.Dy()-seite)/2))
	zeichneSymbol(gtx, s, ecke, seite, farbeText, farbeAkzent)
	t := op.Offset(r.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(r.Size())
	c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
	pointer.CursorPointer.Add(gtx.Ops)
	t.Pop()
}

// farbKnopf ist das Farbfeld unter den Werkzeugen: Klick nimmt die nächste Farbe (bis die rechte Leiste die Farbwahl
// trägt, ADR-0031 Schritt b).
func (e *editor) farbKnopf(gtx layout.Context, r image.Rectangle) {
	c := &e.knoepfe.farbe
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
	rand := farbeLinie
	if c.Hovered() {
		rand = farbeNebentext
	}
	paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	innen := r.Inset(gtx.Dp(8))
	paint.FillShape(gtx.Ops, e.s.Stil.Farbe, clip.Rect(innen).Op())
	paint.FillShape(gtx.Ops, farbeText, clip.Stroke{Path: clip.Rect(innen).Path(), Width: float32(gtx.Dp(1))}.Op())
	if gtx.Focused(c) {
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(-gtx.Dp(3))).Path(),
			Width: float32(gtx.Dp(2))}.Op())
	}
	t := op.Offset(r.Min).Push(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(r.Size())
	c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
	pointer.CursorPointer.Add(gtx.Ops)
	t.Pop()
	if e.farbHinweisZeigen(gtx) {
		e.zeichneHinweis(gtx, i18n.T("Color"), image.Pt(r.Max.X+gtx.Dp(8), r.Min.Y+gtx.Dp(8)))
	}
}

// farbHinweisZeigen: Der Hinweis „Color“ steht bei Zeiger und bei Tastaturfokus, wie die Hinweise der Werkzeuge nicht
// über einer offenen Liste (F-515).
func (e *editor) farbHinweisZeigen(gtx layout.Context) bool {
	c := &e.knoepfe.farbe
	return !e.menue.offen && (c.Hovered() || gtx.Focused(c))
}

// zeichneSymbol zeichnet ein Symbol aus internal/icons als gefüllte Vielecke im Quadrat (ecke, seite).
func zeichneSymbol(gtx layout.Context, s icons.Symbol, ecke image.Point, seite int, vorn, akzent color.NRGBA) {
	k := float32(seite) / icons.Raster
	for _, t := range s.Teile {
		var p clip.Path
		p.Begin(gtx.Ops)
		for i, q := range t.Punkte {
			pt := f32.Pt(float32(ecke.X)+float32(q.X)*k, float32(ecke.Y)+float32(q.Y)*k)
			if i == 0 {
				p.MoveTo(pt)
			} else {
				p.LineTo(pt)
			}
		}
		p.Close()
		c := vorn
		if t.Akzent {
			c = akzent
		}
		paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
	}
}

// zeichneHinweis zeichnet ein eckiges Schild mit text, links oben bei ecke (im Fenster gehalten).
func (e *editor) zeichneHinweis(gtx layout.Context, text string, ecke image.Point) {
	r := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min = image.Point{}
	lbl := material.Label(e.thema, unit.Sp(12), text)
	lbl.Color = farbeText
	d := lbl.Layout(g)
	call := r.Stop()
	rand := gtx.Dp(6)
	fl := image.Rectangle{Max: d.Size.Add(image.Pt(2*rand, rand))}
	ecke.X = max(0, min(ecke.X, gtx.Constraints.Max.X-fl.Dx()))
	t := op.Offset(ecke).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(fl).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(fl).Path(), Width: float32(gtx.Dp(1))}.Op())
	tt := op.Offset(image.Pt(rand, rand/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	tt.Pop()
	t.Pop()
}

// --- Fußzeile ---

// fussText ist der Text eines Felds der Fußzeile in der Sprache (ADR-0014; Zahlenformat über den Kontext).
func fussText(f canvas.FussFeld) string {
	switch f.Art {
	case canvas.FeldAuswahl:
		return fmt.Sprintf(i18n.TK("status bar", "Selection %d × %d px"), f.Groesse.X, f.Groesse.Y)
	case canvas.FeldZeiger:
		return fmt.Sprintf(i18n.TK("status bar", "at %d, %d"), f.Ort.X, f.Ort.Y)
	case canvas.FeldFarbe:
		return fmt.Sprintf(i18n.TK("status bar", "#%02X%02X%02X · %d%%"), f.Farbe.R, f.Farbe.G, f.Farbe.B,
			canvas.Deckkraft(f.Farbe))
	case canvas.FeldSchrift:
		return fmt.Sprintf(i18n.TK("status bar", "%d px"), f.Grad)
	case canvas.FeldBild:
		if f.Gestaucht {
			return fmt.Sprintf(i18n.TK("status bar", "Image %d × %d (cut out)"), f.Groesse.X, f.Groesse.Y)
		}
		return fmt.Sprintf(i18n.TK("status bar", "Image %d × %d"), f.Groesse.X, f.Groesse.Y)
	case canvas.FeldZoom:
		return fmt.Sprintf(i18n.TK("status bar", "%d%%"), f.Zoom)
	case canvas.FeldStand:
		if f.Ungesichert {
			return i18n.T("unsaved")
		}
		return i18n.T("saved")
	}
	return ""
}

// fussKnopf ist das Clickable eines anklickbaren Felds (nil: nur Anzeige).
func (e *editor) fussKnopf(a canvas.FeldArt) *widget.Clickable {
	switch a {
	case canvas.FeldFarbe:
		return &e.knoepfe.fussFarbe
	case canvas.FeldZoom:
		return &e.knoepfe.fussZoom
	}
	return nil
}

// zeichneFusszeile: links die Meldung bzw. der Hinweis des Werkzeugs (canvas.StatusZeile), rechts die Felder
// (ADR-0031 Punkt 4).
func (e *editor) zeichneFusszeile(gtx layout.Context) {
	mx := gtx.Constraints.Max
	h := gtx.Dp(fusszeileDp)
	y0 := mx.Y - h
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(image.Rect(0, y0, mx.X, mx.Y)).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(0, y0, mx.X, y0+gtx.Dp(1))).Op())
	// Meldung zuerst (Sitzungs-QA B-1): Eine Meldung (Speichern, Warnung, Fehler) behält ihren ganzen Text, solange
	// zwei Drittel der Breite reichen; die Felder weichen von links her. Der bloße Bedienhinweis des Werkzeugs weicht
	// dagegen den Feldern (mindestens ein Drittel bleibt ihm).
	links, fehler, meldung := canvas.StatusZeile(&e.meldung, e.s.D.Stand(), e.s.Werkzeug, e.hinweis(), e.s.TextLaeuft())
	grenze := mx.X / 3
	if (meldung || fehler) && links != "" {
		lbl := material.Label(e.thema, unit.Sp(12), links)
		g := gtx
		g.Constraints.Min = image.Point{}
		g.Constraints.Max.X = mx.X
		m := op.Record(gtx.Ops)
		d := lbl.Layout(g)
		m.Stop()
		grenze = max(grenze, min(mx.X*2/3, d.Size.X+gtx.Dp(36)))
	}
	felder := e.s.FussFelder(e.zeigerAnsicht())
	e.fussRahmen = e.fussRahmen[:0]
	// Die Felder werden von rechts her gelegt (links weichen sie der Meldung); die Klickflächen kommen danach von
	// links nach rechts an die Reihe, damit Tab sie in der Reihenfolge des Bildschirms besucht (F-515).
	type fussKlick struct {
		r image.Rectangle
		k *widget.Clickable
	}
	var klicks []fussKlick
	x := mx.X
	for i := len(felder) - 1; i >= 0; i-- {
		f := felder[i]
		text := fussText(f)
		c := farbeNebentext
		if f.Art == canvas.FeldStand {
			c = farbeGruenText
			if f.Ungesichert {
				c = farbeAkzent
			}
		}
		lbl := material.Label(e.thema, unit.Sp(12), text)
		lbl.Color = c
		lbl.MaxLines = 1
		g := gtx
		g.Constraints.Min = image.Point{}
		m := op.Record(gtx.Ops)
		d := lbl.Layout(g)
		call := m.Stop()
		innen := gtx.Dp(10)
		feld := 0
		if f.Art == canvas.FeldFarbe {
			feld = gtx.Dp(12 + 6)
		}
		breite := d.Size.X + 2*innen + feld
		r := image.Rect(x-breite, y0+gtx.Dp(1), x, mx.Y)
		if x-breite < grenze { // links bleibt Platz für die Meldung; überzählige Felder fallen weg
			break
		}
		e.fussRahmen = append(e.fussRahmen, r)
		k := e.fussKnopf(f.Art)
		if k != nil && f.Klickbar && k.Hovered() {
			paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(r).Op())
		}
		if k != nil && gtx.Focused(k) {
			paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(gtx.Dp(2))).Path(),
				Width: float32(gtx.Dp(2))}.Op())
		}
		paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(r.Min.X, r.Min.Y+gtx.Dp(6), r.Min.X+gtx.Dp(1),
			r.Max.Y-gtx.Dp(6))).Op())
		if feld > 0 {
			q := gtx.Dp(12)
			fr := image.Rect(r.Min.X+innen, y0+(h-q)/2, r.Min.X+innen+q, y0+(h-q)/2+q)
			paint.FillShape(gtx.Ops, f.Farbe, clip.Rect(fr).Op())
			paint.FillShape(gtx.Ops, farbeText, clip.Stroke{Path: clip.Rect(fr).Path(), Width: float32(gtx.Dp(1))}.Op())
		}
		tt := op.Offset(image.Pt(r.Min.X+innen+feld, y0+(h-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
		if k != nil && f.Klickbar {
			klicks = append(klicks, fussKlick{r, k})
		}
		x = r.Min.X
	}
	for i := len(klicks) - 1; i >= 0; i-- {
		r, k := klicks[i].r, klicks[i].k
		ft := op.Offset(r.Min).Push(gtx.Ops)
		g := gtx
		g.Constraints = layout.Exact(r.Size())
		k.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: r.Size()} })
		pointer.CursorPointer.Add(gtx.Ops)
		ft.Pop()
	}
	// Meldung bzw. Hinweis links, bis vor die Felder (gekürzt).
	if links == "" {
		return
	}
	c := farbeNebentext
	switch {
	case fehler:
		c = farbeRot
	case meldung:
		c = farbeGruenText
	}
	lbl := material.Label(e.thema, unit.Sp(12), links)
	lbl.Color = c
	lbl.MaxLines = 1
	g := gtx
	g.Constraints.Min = image.Point{}
	g.Constraints.Max.X = max(0, x-gtx.Dp(24))
	m := op.Record(gtx.Ops)
	d := lbl.Layout(g)
	call := m.Stop()
	tt := op.Offset(image.Pt(gtx.Dp(12), y0+(h-d.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	tt.Pop()
}

// fussAuswerten: Klick auf das Farbfeld nimmt die nächste Farbe, Klick auf den Zoom wechselt zwischen Einpassen und
// 1:1 (bis die Felder Listen aufklappen, ADR-0031 Schritt b).
func (e *editor) fussAuswerten(gtx layout.Context) {
	an := func() bool { return !e.s.SpeichernLaeuft() }
	if e.knoepfe.fussFarbe.Clicked(gtx) && an() {
		e.s.NaechsteFarbe()
	}
	if e.knoepfe.fussZoom.Clicked(gtx) && an() {
		if e.s.Abb.Zoom == 1 {
			e.s.Einpassen()
		} else {
			e.s.EinsZuEins()
		}
	}
}
