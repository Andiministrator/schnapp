// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// menueEintrag ist eine Zeile des Menüs (ADR-0026 Punkt 2).
type menueEintrag struct {
	akt    aktion
	text   string
	an     bool
	gruppe bool // beginnt eine neue Gruppe (Trennlinie davor)
}

// menueEintraege liefert die Einträge in Reihenfolge, mit dem Zustand der Sitzung (an oder grau).
func (e *editor) menueEintraege() []menueEintrag {
	s := e.s
	return []menueEintrag{
		{akt: aktKopieren, text: i18n.T("Copy"), an: true},
		{akt: aktExport, text: i18n.T("Export…"), an: !e.exportOffen},
		{akt: aktSpeichern, text: i18n.T("Save"), an: true},
		{akt: aktRueckgaengig, text: i18n.T("Undo"), an: s.D.KannRueckgaengig(), gruppe: true},
		{akt: aktWiederholen, text: i18n.T("Redo"), an: s.D.KannWiederholen()},
		{akt: aktTextBearbeiten, text: i18n.T("Edit text"), an: s.KannTextBearbeiten(), gruppe: true},
		{akt: aktLoeschen, text: i18n.T("Delete"), an: s.KannLoeschen()},
		{akt: aktGanzesBild, text: i18n.T("Whole image"), an: s.KannGanzesBild(), gruppe: true},
		{akt: aktZurueckholen, text: i18n.T("Restore cut-out areas"), an: s.KannZurueckholen()},
		{akt: aktKantenform, text: i18n.T("Seam style"), an: s.KannKanteWechseln()},
		{akt: aktEinpassen, text: i18n.T("Fit"), an: true, gruppe: true},
		{akt: aktEins, text: "1:1", an: true},
		{akt: aktSchliessen, text: i18n.T("Close"), an: true, gruppe: true},
	}
}

// kantenMeldung ist die Statusmeldung nach einem Wechsel der Kantenform (Menüeintrag „Seam style“). Jede Form hat ihren
// eigenen Text als Literal, damit die Übersetzung ihn findet (ADR-0014 Nachtrag 1).
func kantenMeldung(k document.Kante) string {
	switch k {
	case document.KanteWelle:
		return i18n.T("Seam style: wave")
	case document.KanteZacken:
		return i18n.T("Seam style: zigzag")
	}
	return i18n.T("Seam style: torn edge")
}

// schliessenEintrag ist der Knopf des Menüeintrags „Schließen“ (nil, wenn es ihn nicht gibt).
func (e *editor) schliessenEintrag() *widget.Clickable {
	for i, en := range e.menueEintraege() {
		if en.akt == aktSchliessen {
			return &e.menue.eintraege[i]
		}
	}
	return nil
}

// menueKuerzel ist das angezeigte Kürzel eines Eintrags; „Text bearbeiten“ geht mit Enter.
func menueKuerzel(a aktion) string {
	if a == aktTextBearbeiten {
		return kuerzelText(aktEnter)
	}
	return kuerzelText(a)
}

// menue ist der Zustand des Menüs.
type menue struct {
	knopf     widget.Clickable
	offen     bool
	eintraege [16]widget.Clickable // je Eintrag einer; mehr als genug für menueEintraege
	aussen    int                  // Ziel für Klicks neben die offene Liste (schließt sie)

	// Tastatur (F-255): fokusNach ist der Eintrag, der beim Zeichnen den Fokus bekommt (die Klickfläche gibt es erst
	// dort); cursor der zuletzt per Taste angesteuerte Eintrag (-1: keiner).
	fokusNach *widget.Clickable
	cursor    int
	// perTasteOffen: Die Liste wurde per Tastatur geöffnet (Esc gibt den Fokus dann an den Menüknopf zurück, sonst an
	// die Leinwand).
	perTasteOffen bool

	// rahmen: Lage der Zeilen im Fenster beim letzten Zeichnen (Tests und Drehbuch-Koordinaten, F-265).
	rahmen [16]image.Rectangle
}

// menueNaechster liefert den Index des nächsten bedienbaren Eintrags von von aus in Richtung dir (+1 abwärts, -1
// aufwärts), die Liste läuft an den Enden um (wie die Menüs in KDE/Qt); graue Einträge werden übersprungen. von = -1:
// keiner angesteuert, dann der erste (abwärts) bzw. letzte (aufwärts). -1, wenn nichts bedienbar ist.
func menueNaechster(en []menueEintrag, von, dir int) int {
	n := len(en)
	if n == 0 {
		return -1
	}
	start := von
	if von < 0 {
		start = -1
		if dir < 0 {
			start = n
		}
	}
	for k := 1; k <= n; k++ {
		i := ((start+dir*k)%n + n) % n
		if en[i].an {
			return i
		}
	}
	return -1
}

// menueErster und menueLetzter: der erste bzw. letzte bedienbare Eintrag (-1: keiner).
func menueErster(en []menueEintrag) int  { return menueNaechster(en, -1, 1) }
func menueLetzter(en []menueEintrag) int { return menueNaechster(en, -1, -1) }

// menueFokusIndex ist der Eintrag mit Tastaturfokus (-1: keiner).
func (e *editor) menueFokusIndex(gtx layout.Context) int {
	for i := range e.menueEintraege() {
		if gtx.Focused(&e.menue.eintraege[i]) {
			return i
		}
	}
	return -1
}

// menueTaste verarbeitet Pfeile, Pos1 und Ende bei offenem Menü (F-255, ADR-0026 Nachtrag 2): Der Fokus wandert zum
// nächsten bedienbaren Eintrag; die Wirkung folgt beim Zeichnen (menueFokusSetzen).
func (e *editor) menueTaste(gtx layout.Context, name key.Name) {
	m := &e.menue
	en := e.menueEintraege()
	cur := m.cursor
	if m.fokusNach == nil {
		if i := e.menueFokusIndex(gtx); i >= 0 {
			cur = i
		}
	}
	neu := -1
	switch name {
	case key.NameDownArrow:
		neu = menueNaechster(en, cur, 1)
	case key.NameUpArrow:
		neu = menueNaechster(en, cur, -1)
	case key.NameHome:
		neu = menueErster(en)
	case key.NameEnd:
		neu = menueLetzter(en)
	}
	if neu >= 0 {
		m.cursor, m.fokusNach = neu, &m.eintraege[neu]
	}
}

// menueSchliesse schließt die Liste und gibt den Fokus zurück: per Esc an den Menüknopf, nach einer Handlung oder
// einem Klick daneben an die Leinwand (F-255; sonst fiele der Fokus auf nil, getippter Text ginge verloren).
func (e *editor) menueSchliesse(gtx layout.Context, zuKnopf bool) {
	m := &e.menue
	m.offen, m.fokusNach = false, nil
	if zuKnopf {
		gtx.Execute(key.FocusCmd{Tag: &m.knopf})
	} else {
		gtx.Execute(key.FocusCmd{Tag: e})
	}
}

// fuehreAus führt eine Handlung aus, gleich ob per Taste, Knopf oder Menü. vonMaus: Ein laufender Text wird vorher
// bestätigt (wie die Knöpfe); per Taste wirken Kopieren und Export in der Texteingabe nicht (wie bisher).
func (e *editor) fuehreAus(gtx layout.Context, a aktion, vonMaus bool) {
	if vonMaus {
		e.s.TextBestaetigen()
	}
	if w := e.wirkung(a); w != nil {
		w(gtx)
	}
}

// aktionAusfuehrbar meldet, ob fuehreAus für a etwas tut (Test: kein Menüeintrag ohne Wirkung, F-253).
func aktionAusfuehrbar(a aktion) bool { return (&editor{}).wirkung(a) != nil }

// wirkung ist die Handlung zu a; nil für aktKeine und die Tasten mit eigener Behandlung (aktEsc, aktEnter).
func (e *editor) wirkung(a aktion) func(gtx layout.Context) {
	switch a {
	case aktKopieren:
		return func(gtx layout.Context) {
			if e.s.KopierenErlaubt() {
				e.kopieren(gtx)
			}
		}
	case aktExport:
		return func(layout.Context) {
			if e.s.KopierenErlaubt() {
				e.exportStarten()
			}
		}
	case aktSpeichern:
		return func(layout.Context) { e.speichernJetzt() }
	case aktRueckgaengig:
		return func(layout.Context) { e.rueckgaengig() }
	case aktWiederholen:
		return func(layout.Context) { e.wiederholen() }
	case aktTextBearbeiten:
		return func(layout.Context) { e.s.TextBearbeiten() }
	case aktLoeschen:
		return func(layout.Context) { e.s.Loeschen() }
	case aktGanzesBild:
		// Zuschnitt aufheben und Randstreifen zurückholen, wie Doppelklick mit Zuschneiden (E-21, F-222).
		return func(layout.Context) {
			if e.s.HebeZuschnittAuf() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktZurueckholen:
		return func(layout.Context) {
			if e.s.HoleGestauchtesZurueck() {
				e.bildNeu()
				e.meldeAnsicht()
			}
		}
	case aktKantenform:
		// Kantenform reihum, ein Rückgängig-Schritt (ADR-0023 Nachtrag 6); die Statusleiste nennt die neue Form. Die
		// Kanten zeichnet zeichneNaehte neu, ihr Zwischenspeicher ist nach der Form geschlüsselt.
		return func(layout.Context) {
			if k, ok := e.s.WechsleKante(); ok {
				e.melde(kantenMeldung(k), false)
			}
		}
	case aktEinpassen:
		return func(layout.Context) { e.s.Einpassen() }
	case aktEins:
		return func(layout.Context) { e.s.EinsZuEins() }
	case aktSchliessen:
		return func(layout.Context) {
			diagnoseEditor("Schließen über Taste oder Menü")
			// In der eigenen Schleife, wie im Overlay (F-77): Die Anfrage kommt als ClosingEvent zurück und geht durch
			// schliessen; läuft ein Speichern, merkt SchliessenAnfrage sie vor (F-299).
			if e.perform != nil {
				e.perform(system.ActionClose)
				return
			}
			e.w.Perform(system.ActionClose)
		}
	}
	return nil
}

// menueUmschalten öffnet oder schließt die Liste. Per Tastatur geöffnet: Fokus auf den ersten bedienbaren Eintrag (F-255).
// Per Maus bleibt der Fokus unberührt, damit kein Ring ohne Tastatur erscheint. Die Art kommt vom Aufrufer, nicht aus
// dem Klick: Gio meldet bei einem Knopfklick keine Quelle, und der Zeigerort sagt nichts (Leertaste bei ruhendem Zeiger,
// F-255 V-2). Enter und Leertaste öffnen sofort statt erst im nächsten Bild, damit eine gleich folgende Pfeiltaste
// die Liste schon findet (F-255 V-3).
func (e *editor) menueUmschalten(perTaste bool) {
	m := &e.menue
	m.offen = !m.offen
	m.cursor = -1
	m.perTasteOffen = m.offen && perTaste
	if m.perTasteOffen {
		if i := menueErster(e.menueEintraege()); i >= 0 {
			m.cursor, m.fokusNach = i, &m.eintraege[i]
		}
	}
}

// knopfPerTaste löst einen per Tastatur fokussierten Knopf aus (Enter): der Menüknopf über menueUmschalten, die
// übrigen als Klick.
func (e *editor) knopfPerTaste(c *widget.Clickable) {
	if c == &e.menue.knopf {
		e.menueUmschalten(true)
		return
	}
	c.Click()
}

// menueAuswerten verarbeitet Klicks auf den Menüknopf, die Einträge und daneben.
func (e *editor) menueAuswerten(gtx layout.Context) {
	m := &e.menue
	// Während des Speicherns ruht das Bearbeiten (ADR-0020 Nachtrag 6); je Klick neu gefragt, denn ein Eintrag im selben
	// Bild kann das Speichern starten (F-280).
	an := func() bool { return !e.s.SpeichernLaeuft() }
	if m.knopf.Clicked(gtx) && an() {
		e.menueUmschalten(false) // ein Klick des Knopfes ist die Maus; Enter und Leertaste laufen in tasten
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.aussen, Kinds: pointer.Press})
		if !ok {
			break
		}
		if p, ok := ev.(pointer.Event); ok && p.Kind == pointer.Press && m.offen {
			e.menueSchliesse(gtx, false)
		}
	}
	for { // Klicks auf die Fläche zwischen den Einträgen: abholen und nichts tun
		if _, ok := gtx.Event(pointer.Filter{Target: m, Kinds: pointer.Press}); !ok {
			break
		}
	}
	for i, en := range e.menueEintraege() {
		// Schließen wirkt auch während des Speicherns wie der Fensterknopf: Die Anfrage wird vorgemerkt (F-299).
		if m.eintraege[i].Clicked(gtx) && m.offen && en.an && (an() || en.akt == aktSchliessen) {
			e.menueSchliesse(gtx, false) // Fokus an die Leinwand: getippter Text nach „Text bearbeiten“ kommt an
			e.fuehreAus(gtx, en.akt, true)
		}
	}
}

// zeichneMenue legt die offene Liste unter den Menüknopf, rechtsbündig an rechts (Fensterpixel), oben bei y. Darunter
// liegt eine Fläche über dem ganzen Fenster, die einen Klick daneben abfängt.
func (e *editor) zeichneMenue(gtx layout.Context, rechts, y int) {
	m := &e.menue
	if !m.offen {
		return
	}
	mx := gtx.Constraints.Max
	ha := clip.Rect(image.Rectangle{Max: mx}).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.aussen)
	ha.Pop()

	eintraege := e.menueEintraege()
	zeile, innen, trenn := gtx.Dp(32), gtx.Dp(14), gtx.Dp(9)
	// Breite: längster Text plus längstes Kürzel, gemessen ohne Zeichnen.
	mess := func(s string) int {
		if s == "" {
			return 0
		}
		r := op.Record(gtx.Ops)
		g := gtx
		g.Constraints.Min = image.Point{}
		d := material.Label(e.thema, unit.Sp(14), s).Layout(g)
		r.Stop()
		return d.Size.X
	}
	textB, kuerzelB, hoehe := 0, 0, gtx.Dp(6)*2
	for _, en := range eintraege {
		textB = max(textB, mess(en.text))
		kuerzelB = max(kuerzelB, mess(menueKuerzel(en.akt)))
		hoehe += zeile
		if en.gruppe {
			hoehe += trenn
		}
	}
	breite := innen + textB + gtx.Dp(32) + kuerzelB + innen
	// Passt die Liste nicht unter den Knopf, rückt sie nach oben (über die Leiste); ist das Fenster niedriger als die
	// Liste, fallen die Abstände der Gruppen weg und die Zeilen werden niedriger, bis zur Höhe einer Textzeile
	// (F-252: bei der Mindesthöhe 320 dp wären sonst die unteren Einträge nicht erreichbar).
	if y+hoehe > mx.Y {
		y = max(0, mx.Y-hoehe)
	}
	if hoehe > mx.Y {
		trenn = 0
		zeile = menueZeilenhoehe(mx.Y, gtx.Dp(6), gtx.Dp(20), gtx.Dp(32), len(eintraege))
		hoehe = gtx.Dp(6)*2 + zeile*len(eintraege)
		y = 0
	}
	ecke := image.Pt(max(0, rechts-breite), y)
	t := op.Offset(ecke).Push(gtx.Ops)
	flaeche := image.Rectangle{Max: image.Pt(breite, hoehe)}
	rr := clip.UniformRRect(flaeche, gtx.Dp(6))
	paint.FillShape(gtx.Ops, farbeLeisteVoll, rr.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(gtx.Dp(1))}.Op())
	// Die Fläche selbst fängt Klicks zwischen den Einträgen ab, damit sie nicht als „daneben“ gelten.
	hf := clip.Rect(flaeche).Push(gtx.Ops)
	event.Op(gtx.Ops, m)
	hf.Pop()
	oy := gtx.Dp(6)
	for i, en := range eintraege {
		if en.gruppe && i > 0 {
			ly := oy + trenn/2
			paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(innen/2, ly, breite-innen/2, ly+gtx.Dp(1))).Op())
		}
		if en.gruppe {
			oy += trenn
		}
		m.rahmen[i] = image.Rectangle{Min: ecke.Add(image.Pt(0, oy)), Max: ecke.Add(image.Pt(breite, oy+zeile))}
		e.menueZeile(gtx, &m.eintraege[i], en, image.Pt(0, oy), image.Pt(breite, zeile), innen)
		oy += zeile
	}
	t.Pop()
	if m.fokusNach != nil { // die Klickflächen sind jetzt angemeldet: Fokus in die Liste (F-255)
		gtx.Execute(key.FocusCmd{Tag: m.fokusNach})
		gtx.Execute(op.InvalidateCmd{})
		m.fokusNach = nil
	}
}

// menueZeilenhoehe ist die Zeilenhöhe, wenn das Fenster niedriger ist als die Liste: der Platz nach Abzug des
// Randes, geteilt durch die Zeilen, nie unter kleinst und nie über groesst (F-262: sonst würden die Zeilen bei
// Fenstern knapp unter der Listenhöhe höher statt niedriger). Alles in Pixeln.
func menueZeilenhoehe(fensterHoehe, rand, kleinst, groesst, zeilen int) int {
	return min(groesst, max(kleinst, (fensterHoehe-rand*2)/zeilen))
}

// menueZeile zeichnet einen Eintrag: Text links, Kürzel rechts, grau wenn nicht bedienbar.
func (e *editor) menueZeile(gtx layout.Context, c *widget.Clickable, en menueEintrag, ecke, groesse image.Point, innen int) {
	t := op.Offset(ecke).Push(gtx.Ops)
	fokus := en.an && gtx.Focused(c)
	if en.an && (c.Hovered() || fokus) {
		paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(image.Rectangle{Max: groesse}).Op())
	}
	if fokus { // Tastaturfokus immer sichtbar: 2 px orangener Ring (corporate-design.md §6). Die Zeilen stoßen aneinander,
		// außen ist kein Platz: Der Ring liegt innen mit 2 px Abstand zum Zeilenrand (ADR-0026 Nachtrag 3).
		r := image.Rectangle{Max: groesse}.Inset(gtx.Dp(3))
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Stroke{Path: clip.UniformRRect(r, gtx.Dp(4)).Path(gtx.Ops),
			Width: float32(gtx.Dp(2))}.Op())
	}
	farbe, neben := farbeText, farbeNebentext
	if !en.an {
		farbe = farbeNebentext
	}
	g := gtx
	g.Constraints.Min = image.Point{}
	schreibe := func(s string, c color.NRGBA, rechtsbuendig bool) {
		r := op.Record(gtx.Ops)
		lbl := material.Label(e.thema, unit.Sp(14), s)
		lbl.Color = c
		d := lbl.Layout(g)
		call := r.Stop()
		x := innen
		if rechtsbuendig {
			x = groesse.X - innen - d.Size.X
		}
		tt := op.Offset(image.Pt(x, (groesse.Y-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tt.Pop()
	}
	schreibe(en.text, farbe, false)
	if k := menueKuerzel(en.akt); k != "" {
		schreibe(k, neben, true)
	}
	if en.an {
		g.Constraints = layout.Exact(groesse)
		c.Layout(g, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: groesse} })
		pointer.CursorPointer.Add(gtx.Ops)
	}
	t.Pop()
}

// hinweisKnoepfe sind die Werkzeugknöpfe, über denen ein Hinweis steht: bei Zeiger und bei Tastaturfokus; ist ein
// Knopf fokussiert, nur dieser, damit nicht zwei Hinweise zugleich stehen (F-255).
func (e *editor) hinweisKnoepfe(gtx layout.Context) []werkzeugKnopf {
	var fokus, hover []werkzeugKnopf
	for _, w := range e.werkzeugKnoepfe() {
		if gtx.Focused(w.c) {
			fokus = append(fokus, w)
		} else if w.c.Hovered() {
			hover = append(hover, w)
		}
	}
	if len(fokus) > 0 {
		return fokus
	}
	return hover
}

// zeichneWerkzeugHinweis zeigt unter dem Werkzeugknopf, über dem der Zeiger steht, Namen und Kürzel (ADR-0026 Punkt 3).
func (e *editor) zeichneWerkzeugHinweis(gtx layout.Context, links []leistenKnopf, pos []image.Point) {
	if e.menue.offen {
		return
	}
	for _, w := range e.hinweisKnoepfe(gtx) {
		for i, l := range links {
			if l.c != w.c {
				continue
			}
			text := w.text + " (" + werkzeugKuerzel(w.w) + ")"
			r := op.Record(gtx.Ops)
			g := gtx
			g.Constraints.Min = image.Point{}
			lbl := material.Label(e.thema, unit.Sp(12), text)
			lbl.Color = farbeText
			d := lbl.Layout(g)
			call := r.Stop()
			rand := gtx.Dp(6)
			ecke := pos[i].Add(image.Pt(0, gtx.Dp(36)+gtx.Dp(4)))
			ecke.X = max(0, min(ecke.X, gtx.Constraints.Max.X-d.Size.X-2*rand))
			t := op.Offset(ecke).Push(gtx.Ops)
			fl := image.Rectangle{Max: d.Size.Add(image.Pt(2*rand, rand))}
			paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.UniformRRect(fl, gtx.Dp(4)).Op(gtx.Ops))
			paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.UniformRRect(fl, gtx.Dp(4)).Path(gtx.Ops),
				Width: float32(gtx.Dp(1))}.Op())
			tt := op.Offset(image.Pt(rand, rand/2)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			tt.Pop()
			t.Pop()
		}
	}
}
