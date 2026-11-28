// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Abschnitt „Schnellstile“ der rechten Leiste (ADR-0037, Karte #192): Kacheln mit je einer Mini-Darstellung des Stils
// auf hellem Schachbrett, orange der Rand des Stils, der zum Stand passt, graue Ecke an eigenen Stilen, hinter den
// Kacheln der Knopf „+“. Was ein Klick bewirkt, entscheidet canvas.Sitzung (schnellstile.go dort); hier stehen Lage,
// Zeichnen, Tastatur und das Lesen und Schreiben von styles.ini beim Öffnen und bei „+“ und Entfernen.

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Maße der Kacheln in dp (ADR-0037 Punkt 2) und die größte Zahl von Kacheln einer Art: sieben eingebaute, dazu die
// eigenen.
const (
	schnellKachelDp = 40
	schnellLueckeDp = 5
	schnellSpalten  = 6
	schnellMax      = 8 + canvas.MaxEigeneSchnellstile
)

// Lesen und Schreiben der Datei; Tests tauschen sie nicht, sie setzen XDG_CONFIG_HOME (TestMain).
var (
	ladeStile = settings.LadeStile
	// aendereStile liest styles.ini unter einer Sperre neu und wendet die Änderung an (F-627); der Aufruf heißt nicht
	// Schreibe…, denn die Prüfung in cmd/schnapp erlaubt hier nur den Schreiber der Stile.
	aendereStile = settings.AendereStile
)

// schnellLeiste hält Knöpfe, Lage und Zustand des Abschnitts.
type schnellLeiste struct {
	kacheln [schnellMax]widget.Clickable
	rechts  [schnellMax]byte // Zeigerziele für den Rechtsklick (nur die Adresse zählt)
	plus    widget.Clickable
	kopf    widget.Clickable // Überschrift: klappt den Abschnitt ein und aus
	zu      bool

	// Stand beim letzten Zeichnen: Klicks und Tasten beziehen sich auf die Kacheln, die der Nutzer gesehen hat.
	gezeigt    bool
	art        canvas.SchnellArt
	liste      []canvas.Schnellstil
	rahmen     [schnellMax]image.Rectangle // Lage der Kacheln im Fenster (Inhaltskoordinaten, Tests)
	plusRahmen image.Rectangle
	plusAn     bool   // „+“ wirkt beim letzten Zeichnen
	hinweis    string // Name der Kachel mit Fokus oder Zeiger für die Statuszeile
}

// alle sind die Knöpfe des Abschnitts in Tab-Reihenfolge (Tastaturfokus, editor.fokussierterKnopf).
func (k *schnellLeiste) alle() []*widget.Clickable {
	a := []*widget.Clickable{&k.kopf}
	for i := range k.kacheln {
		a = append(a, &k.kacheln[i])
	}
	return append(a, &k.plus)
}

// ziele sind die Knöpfe, an denen die Leertaste wie Enter wirkt (leertastenZiele).
func (k *schnellLeiste) ziele() []event.Tag {
	var z []event.Tag
	for i := range k.kacheln {
		z = append(z, &k.kacheln[i])
	}
	return append(z, &k.plus)
}

// stelle liefert die Stelle von c unter den Kacheln (0 bis len(liste)-1) und „+“ (len(liste)); -1, wenn c keine ist.
func (k *schnellLeiste) stelle(c *widget.Clickable) int {
	if c == &k.plus {
		return len(k.liste)
	}
	for i := range k.liste {
		if c == &k.kacheln[i] {
			return i
		}
	}
	return -1
}

// knopfBei ist der Knopf an der Stelle i (siehe stelle).
func (k *schnellLeiste) knopfBei(i int) *widget.Clickable {
	if i == len(k.liste) {
		return &k.plus
	}
	return &k.kacheln[i]
}

// schriftFehlt meldet, ob die Familie des Stils auf diesem Rechner fehlt. Solange die Suche nach Systemschriften läuft,
// gilt keine als fehlend (sonst blitzte jede beim Öffnen auf).
func schriftFehlt(st canvas.Schnellstil) bool {
	return st.Schrift != "" && fonts.SucheFertig() && !fonts.Bekannt(st.Schrift)
}

// stileLaden liest styles.ini beim Öffnen des Editors; Warnungen gehen auf die Fehlerausgabe wie bei settings.ini.
func (e *editor) stileLaden() {
	roh, warn := ladeStile()
	for _, w := range warn {
		fmt.Fprintln(fehlerAusgabe, w)
	}
	e.eigeneAusDatei(roh)
}

// stileSpeichern ändert styles.ini um genau die eigene Änderung aend (F-627: die Datei wird unter einer Sperre neu
// gelesen, damit ein zweites Fenster nichts zurückholt oder verliert), setzt danach die eigenen Stile der Sitzung auf
// den Stand der Datei und sagt in der Statuszeile, was geschah; ein Fehler bleibt dort. Musste eine Datei mit
// Unlesbarem ersetzt werden, steht in der Meldung, wo die alte liegt (F-629 b).
func (e *editor) stileSpeichern(erfolg string, aend func([]settings.Stil) []settings.Stil) {
	neu, gesichert, err := aendereStile(aend)
	if err != nil {
		e.melde(fmt.Sprintf(i18n.T("Could not save the styles: %v"), fehlerGrund(err)), true)
		return
	}
	e.eigeneAusDatei(neu)
	if gesichert {
		erfolg = i18n.T("Style saved. Unreadable parts of the old styles file are kept in styles.ini.bak.")
	}
	e.melde(erfolg, false)
}

// eigeneAusDatei setzt die eigenen Stile der Sitzung auf die Liste der Datei.
func (e *editor) eigeneAusDatei(roh []settings.Stil) {
	var liste []canvas.Schnellstil
	for _, r := range roh {
		if st, ok := canvas.SchnellstilAus(r); ok {
			liste = append(liste, st)
		}
	}
	e.s.SetzeEigeneSchnellstile(liste)
}

// schnellAuswerten verarbeitet Klicks, Rechtsklicks und die Überschrift (aus seiteAuswerten). Sie beziehen sich auf die
// Kacheln des letzten Bilds; zeigt die Leiste inzwischen eine andere Art, wirken sie nicht.
func (e *editor) schnellAuswerten(gtx layout.Context) {
	k := &e.schnell
	if k.kopf.Clicked(gtx) {
		k.zu = !k.zu
	}
	art, ok := e.s.SchnellArt()
	gueltig := ok && k.gezeigt && art == k.art
	for j := range k.kacheln {
		if k.kacheln[j].Clicked(gtx) && e.seiteAn() && gueltig && j < len(k.liste) {
			e.s.WendeSchnellstilAn(k.liste[j])
		}
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: &k.rechts[j], Kinds: pointer.Press})
			if !ok {
				break
			}
			if pe, ok := ev.(pointer.Event); ok && pe.Buttons == pointer.ButtonSecondary && e.seiteAn() && gueltig {
				e.schnellEntferne(gtx, j)
			}
		}
	}
	if k.plus.Clicked(gtx) && e.seiteAn() && gueltig {
		if st, r := e.s.FuegeSchnellstilHinzu(art); r == canvas.HinzuOK {
			neu := st.Einstellung()
			e.stileSpeichern(i18n.T("Style saved"), func(alt []settings.Stil) []settings.Stil { return settings.StilHinzu(alt, neu) })
		}
	}
}

// schnellEntferne entfernt die eigene Kachel j (Rechtsklick, Entf); eine eingebaute bleibt. Der Fokus bleibt an der
// Stelle, rückt bei der letzten Kachel aber auf „+“, sonst ginge er mit der Kachel verloren.
func (e *editor) schnellEntferne(gtx layout.Context, j int) {
	k := &e.schnell
	if j < 0 || j >= len(k.liste) || !k.liste[j].Eigen() {
		return
	}
	if aktuell := e.s.Schnellstile(k.art); j >= len(aktuell) || aktuell[j] != k.liste[j] {
		return
	}
	weg := k.liste[j].Einstellung()
	if !e.s.EntferneSchnellstil(k.art, j) {
		return
	}
	if gtx.Focused(&k.kacheln[j]) && j >= len(e.s.Schnellstile(k.art)) {
		gtx.Execute(key.FocusCmd{Tag: &k.plus})
	}
	e.stileSpeichern(i18n.T("Style removed"), func(alt []settings.Stil) []settings.Stil { return settings.StilEntfernt(alt, weg) })
}

// schnellTaste behandelt Pfeile und Entf, solange eine Kachel oder „+“ den Fokus hat (ADR-0037 Punkt 7); Enter und
// Leertaste laufen über knopfPerTaste. Liefert, ob die Taste verbraucht ist. Entf gehört der Kachel: Sonst löschte es
// das gewählte Objekt hinter ihr.
func (e *editor) schnellTaste(gtx layout.Context, taste key.Event) bool {
	k := &e.schnell
	i := -1
	for j := 0; j <= len(k.liste) && i < 0; j++ {
		if gtx.Focused(k.knopfBei(j)) {
			i = j
		}
	}
	if i < 0 || !k.gezeigt || taste.Modifiers != 0 {
		return false
	}
	ziel := i
	switch taste.Name {
	case key.NameLeftArrow:
		ziel = i - 1
	case key.NameRightArrow:
		ziel = i + 1
	case key.NameUpArrow:
		ziel = i - schnellSpalten
	case key.NameDownArrow:
		ziel = i + schnellSpalten
	case key.NameHome:
		ziel = 0
	case key.NameEnd:
		ziel = len(k.liste)
	case key.NameDeleteForward:
		e.schnellEntferne(gtx, i)
		return true
	default:
		return false
	}
	if ziel >= 0 && ziel <= len(k.liste) && ziel != i {
		gtx.Execute(key.FocusCmd{Tag: k.knopfBei(ziel)})
	}
	return true
}

// schnellName ist der Text der Statuszeile zu einer Kachel: der Name eines eingebauten Stils, bei einem eigenen der
// Hinweis zum Entfernen; fehlt die Schrift, sagt er es dazu.
func schnellName(st canvas.Schnellstil) string {
	t := st.Name()
	if st.Eigen() {
		t = i18n.T("Custom style. Right-click or Del removes it.")
		if st.Schatten.An {
			t = i18n.T("Custom style with drop shadow. Right-click or Del removes it.")
		}
	}
	if schriftFehlt(st) {
		t = fmt.Sprintf(i18n.T("%s Font “%s” is not installed."), t, st.Schrift)
	}
	return t
}

// schnellPlusText ist der Hinweis zum Knopf „+“.
func (e *editor) schnellPlusText() string {
	k := &e.schnell
	switch e.s.PruefeSchnellstilHinzu(k.art) {
	case canvas.HinzuOK:
		return i18n.T("Save the current settings as a custom style")
	case canvas.HinzuUnbrauchbar:
		return i18n.T("The current settings cannot be saved as a style (stroke width, font name).")
	}
	n := 0
	for _, st := range k.liste {
		if st.Eigen() {
			n++
		}
	}
	if n >= canvas.MaxEigeneSchnellstile {
		return fmt.Sprintf(i18n.T("At most %d custom styles per kind."), canvas.MaxEigeneSchnellstile)
	}
	return i18n.T("The current settings are already a style.")
}

// zeichneSchnellstile zeichnet Überschrift und Kacheln ab y und liefert das neue y. Die Lagen sind Inhaltskoordinaten
// der Leiste (wie bei den anderen Abschnitten).
func (e *editor) zeichneSchnellstile(gtx layout.Context, art canvas.SchnellArt, x, y, breite int) int {
	k := &e.schnell
	k.gezeigt, k.liste, k.hinweis = false, nil, ""
	k.rahmen, k.plusRahmen = [schnellMax]image.Rectangle{}, image.Rectangle{}
	y += e.abschnittKopf(gtx, &k.kopf, i18n.T("Quick styles"), k.zu, x, y, breite) + gtx.Dp(12)
	if k.zu {
		return y - gtx.Dp(12)
	}
	liste := e.s.Schnellstile(art)
	if len(liste) > schnellMax {
		liste = liste[:schnellMax]
	}
	k.gezeigt, k.art, k.liste = true, art, liste
	k.plusAn = e.s.KannSchnellstilHinzufuegen(art)
	kachel, luecke := gtx.Dp(schnellKachelDp), gtx.Dp(schnellLueckeDp)
	lage := func(j int) image.Rectangle {
		x0 := x + (j%schnellSpalten)*(kachel+luecke)
		y0 := y + (j/schnellSpalten)*(kachel+luecke)
		return image.Rect(x0, y0, x0+kachel, y0+kachel)
	}
	for j, st := range liste {
		r := lage(j)
		k.rahmen[j] = r
		passt := e.s.SchnellstilPasst(st)
		e.zeichneKachel(gtx, st, r, passt, k.kacheln[j].Hovered())
		e.knopfAn(gtx, &k.kacheln[j], r)
		rechtsAnmelden(gtx, &k.rechts[j], r)
		// Ein Tastaturfokus geht vor dem Zeiger: Der Hinweis gehört der Kachel, die man bedient.
		if gtx.Focused(&k.kacheln[j]) || (k.hinweis == "" && k.kacheln[j].Hovered()) {
			k.hinweis = schnellName(st)
		}
	}
	r := lage(len(liste))
	k.plusRahmen = r
	zeichnePlus(gtx, r, k.plusAn, k.plus.Hovered())
	e.knopfAn(gtx, &k.plus, r)
	if gtx.Focused(&k.plus) || (k.hinweis == "" && k.plus.Hovered()) {
		k.hinweis = e.schnellPlusText()
	}
	return lage(len(liste)).Max.Y
}

// rechtsAnmelden meldet die Fläche r als Ziel von Rechtsklicks an. Der Zeiger läuft weiter zum Knopf darunter, damit der
// Linksklick ihn erreicht.
func rechtsAnmelden(gtx layout.Context, tag event.Tag, r image.Rectangle) {
	t := op.Offset(r.Min).Push(gtx.Ops)
	p := pointer.PassOp{}.Push(gtx.Ops)
	c := clip.Rect(image.Rectangle{Max: r.Size()}).Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	c.Pop()
	p.Pop()
	t.Pop()
}

// zeichnePlus zeichnet den Knopf „+“: grau, wenn er nichts tut.
func zeichnePlus(gtx layout.Context, r image.Rectangle, an, hover bool) {
	paint.FillShape(gtx.Ops, farbeLeiste, clip.Rect(r).Op())
	rand := farbeLinie
	if hover && an {
		rand = farbeNebentext
	}
	paint.FillShape(gtx.Ops, rand, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	c := farbeText
	if !an {
		c = farbeLinie
	}
	m, arm, dick := r.Min.Add(r.Size().Div(2)), gtx.Dp(8), max(1, gtx.Dp(2)/2*2)
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(m.X-arm, m.Y-dick/2, m.X+arm, m.Y+dick/2)).Op())
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(m.X-dick/2, m.Y-arm, m.X+dick/2, m.Y+arm)).Op())
}

// zeichneKachel zeichnet eine Kachel: Schachbrett, Mini-Darstellung, graue Ecke bei eigenen, orange Rand bei passendem.
func (e *editor) zeichneKachel(gtx layout.Context, st canvas.Schnellstil, r image.Rectangle, passt, hover bool) {
	zelle := max(1, r.Dx()/8)
	paint.FillShape(gtx.Ops, schachDunkel, clip.Rect(r).Op())
	for zy := 0; zy*zelle < r.Dy(); zy++ {
		for zx := 0; zx*zelle < r.Dx(); zx++ {
			if (zx+zy)%2 == 0 {
				z := image.Rect(r.Min.X+zx*zelle, r.Min.Y+zy*zelle, r.Min.X+(zx+1)*zelle, r.Min.Y+(zy+1)*zelle).Intersect(r)
				paint.FillShape(gtx.Ops, schachHell, clip.Rect(z).Op())
			}
		}
	}
	c := clip.Rect(r).Push(gtx.Ops)
	e.zeichneMini(gtx, st, r)
	c.Pop()
	if st.Eigen() { // graue Ecke oben rechts
		s := float32(gtx.Dp(9))
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(float32(r.Max.X)-s, float32(r.Min.Y)))
		p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
		p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)+s))
		p.Close()
		paint.FillShape(gtx.Ops, farbeNebentext, clip.Outline{Path: p.End()}.Op())
	}
	if st.Schatten.An { // Stil mit Schatten: dunkler Winkel unten rechts (Kritik K-3 #363b)
		d := gtx.Dp(3)
		schatten := color.NRGBA{A: 0xB0}
		paint.FillShape(gtx.Ops, schatten, clip.Rect(image.Rect(r.Min.X+gtx.Dp(6), r.Max.Y-d, r.Max.X, r.Max.Y)).Op())
		paint.FillShape(gtx.Ops, schatten, clip.Rect(image.Rect(r.Max.X-d, r.Min.Y+gtx.Dp(6), r.Max.X, r.Max.Y-d)).Op())
	}
	switch {
	case passt:
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.Rect(r.Inset(gtx.Dp(1))).Path(), Width: float32(gtx.Dp(2))}.Op())
	case hover:
		paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(1))}.Op())
	}
}

// strichDp ist die Breite der Linie in der Mini-Darstellung: der Strich des Stils (Faktor 1) verkleinert, zwischen
// zwei und fünf dp, damit „dick“ dicker aussieht als „normal“.
func strichDp(breite float64) float32 {
	return float32(math.Max(2, math.Min(5, breite*0.6)))
}

// zeichneMini zeichnet das Objekt der Art im Stil st in die Kachel r (ein Ausschnitt ist schon aufgelegt): klein und
// eckig, nur mit Pfaden und Flächen.
func (e *editor) zeichneMini(gtx layout.Context, st canvas.Schnellstil, r image.Rectangle) {
	d := func(v float32) float32 { return float32(gtx.Dp(unit.Dp(v))) }
	o := f32.Pt(float32(r.Min.X), float32(r.Min.Y))
	pt := func(px, py float32) f32.Point { return o.Add(f32.Pt(d(px), d(py))) }
	rechteck := func(a, b f32.Point) clip.PathSpec {
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(a)
		p.LineTo(f32.Pt(b.X, a.Y))
		p.LineTo(b)
		p.LineTo(f32.Pt(a.X, b.Y))
		p.Close()
		return p.End()
	}
	rahmen := func(a, b f32.Point, c color.NRGBA, w float32) {
		paint.FillShape(gtx.Ops, c, clip.Stroke{Path: rechteck(a, b), Width: w}.Op())
	}
	flaeche := func(a, b f32.Point, c color.NRGBA) {
		paint.FillShape(gtx.Ops, c, clip.Outline{Path: rechteck(a, b)}.Op())
	}
	dreieck := func(a, b, c f32.Point, f color.NRGBA) {
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(a)
		p.LineTo(b)
		p.LineTo(c)
		p.Close()
		paint.FillShape(gtx.Ops, f, clip.Outline{Path: p.End()}.Op())
	}
	w := d(strichDp(st.Breite))
	switch st.Art {
	case canvas.SchnellPfeil:
		a, b := pt(9, 31), pt(28, 12)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(a)
		p.LineTo(b)
		paint.FillShape(gtx.Ops, st.Farbe, clip.Stroke{Path: p.End(), Width: w}.Op())
		// Spitze: Dreieck am Ende der Linie, Richtung a nach b.
		spitze := pt(32, 8)
		dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
		l := math.Hypot(dx, dy)
		ex, ey := float32(dx/l), float32(dy/l)
		nx, ny := -ey, ex
		fuss := f32.Pt(spitze.X-ex*d(11), spitze.Y-ey*d(11))
		dreieck(spitze, fuss.Add(f32.Pt(nx*d(6), ny*d(6))), fuss.Sub(f32.Pt(nx*d(6), ny*d(6))), st.Farbe)
	case canvas.SchnellKasten:
		a, b := pt(9, 9), pt(31, 31)
		if st.Fuellung.A > 0 {
			flaeche(a, b, st.Fuellung)
		}
		rahmen(a, b, st.Farbe, w)
	case canvas.SchnellTextmarker: // ADR-0048 Punkt 6: zwei dunkle „Zeilen“, darüber der Strich multiplizierend
		flaeche(pt(8, 13), pt(32, 16), color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF})
		flaeche(pt(8, 24), pt(26, 27), color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF})
		m := paint.PushMultiply(gtx.Ops, 1)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(pt(6, 20))
		p.LineTo(pt(34, 20))
		paint.FillShape(gtx.Ops, st.Farbe, clip.Stroke{Path: p.End(), Width: max(w, d(10))}.Op())
		m.Pop()
	case canvas.SchnellEllipse: // ADR-0047 Punkt 5
		a, b := pt(6, 10), pt(34, 30)
		oval := clip.Ellipse{Min: image.Pt(int(a.X), int(a.Y)), Max: image.Pt(int(b.X), int(b.Y))}
		if st.Fuellung.A > 0 {
			paint.FillShape(gtx.Ops, st.Fuellung, oval.Op(gtx.Ops))
		}
		paint.FillShape(gtx.Ops, st.Farbe, clip.Stroke{Path: oval.Path(gtx.Ops), Width: w}.Op())
	case canvas.SchnellText, canvas.SchnellSprechblase:
		a, b := pt(6, 9), pt(34, 29)
		if st.Art == canvas.SchnellSprechblase {
			a, b = pt(5, 6), pt(35, 25)
		}
		if st.Fuellung.A > 0 {
			flaeche(a, b, st.Fuellung)
		}
		if st.Art == canvas.SchnellSprechblase { // Spitze unten links
			f := st.Fuellung
			if f.A == 0 {
				f = st.Farbe
			}
			dreieck(pt(10, 25), pt(18, 25), pt(8, 34), f)
		}
		fehlt := schriftFehlt(st)
		c := st.Farbe
		if fehlt {
			c.A /= 2 // ersetzt: Go statt der gewählten Familie, blass und schräg
		}
		ar := image.Rect(int(a.X), int(a.Y), int(b.X), int(b.Y))
		e.aaMittig(gtx, ar, c, 15, st.Fett, st.Kursiv || fehlt)
	case canvas.SchnellNummer:
		m := pt(20, 20)
		rad := d(12)
		paint.FillShape(gtx.Ops, st.Farbe, clip.Ellipse{Min: image.Pt(int(m.X-rad), int(m.Y-rad)),
			Max: image.Pt(int(m.X+rad), int(m.Y+rad))}.Op(gtx.Ops))
		z := farbeWeiss
		if (299*int(st.Farbe.R)+587*int(st.Farbe.G)+114*int(st.Farbe.B))/1000 > 150 {
			z = farbeGrund
		}
		e.textMittig(gtx, "1", 14, z, image.Rect(int(m.X-rad), int(m.Y-rad), int(m.X+rad), int(m.Y+rad)))
	case canvas.SchnellLupe:
		rahmen(pt(7, 7), pt(18, 18), st.Farbe, w)
		rahmen(pt(15, 15), pt(33, 33), st.Farbe, w)
	}
}

// aaMittig setzt „Aa“ mittig in r, auf Wunsch fett und kursiv.
func (e *editor) aaMittig(gtx layout.Context, r image.Rectangle, c color.NRGBA, sp float32, fett, kursiv bool) {
	lbl := material.Label(e.thema, unit.Sp(sp), "Aa")
	lbl.Color = c
	lbl.MaxLines = 1
	if fett {
		lbl.Font.Weight = font.Bold
	}
	if kursiv {
		lbl.Font.Style = font.Italic
	}
	g := gtx
	g.Constraints = layout.Constraints{Max: r.Size()}
	m := op.Record(gtx.Ops)
	dim := lbl.Layout(g)
	call := m.Stop()
	t := op.Offset(r.Min.Add(r.Size().Sub(dim.Size).Div(2))).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
}
