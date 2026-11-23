// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Abschnitt „Ebenen“ der rechten Leiste (ADR-0032 Punkt 6): Deckkraft der gewählten Ebene, Liste von oben (vorderste
// Ebene) nach unten mit Auge, Vorschau, Name und Schloss, zuletzt das Bild; Ziehen ordnet um, Doppelklick oder F2
// benennt um; darunter die Knöpfe ↑ ↓ Doppeln Löschen. Was eine Aktion bewirkt, entscheidet canvas.Sitzung (ebenen.go
// dort); hier stehen Lage, Zeichnen und Ereignisse.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/icons"
)

// Maße der Liste in dp.
const (
	ebeneZeileDp   = 34
	ebeneAugeDp    = 26 // Klickfläche des Auges links und des Schlosses rechts
	ebeneVorschauB = 38
	ebeneVorschauH = 24
	// ebenenMaxZeilen: so viele Zeilen zeigt die Liste höchstens, darüber blättert sie.
	ebenenMaxZeilen = 8
)

// Knöpfe unter der Liste.
const (
	ebKnopfHoch = iota
	ebKnopfRunter
	ebKnopfDoppeln
	ebKnopfLoeschen
	ebKnopfAnzahl
)

// ebenenZeileLage ist eine gezeichnete Zeile: Kennung (0 = Bild) und Lage im Fenster.
type ebenenZeileLage struct {
	id     int
	rahmen image.Rectangle
}

// ebenenLeiste hält Lage und Zustand des Abschnitts.
type ebenenLeiste struct {
	deck       eingabe // Deckkraft der gewählten Ebene
	name       eingabe // Name während der Umbenennung
	umbenennen int     // Kennung der Ebene in der Umbenennung, 0 = keine
	// umbenennenFokus: Das Namensfeld bekommt beim nächsten Zeichnen den Fokus (es ist erst dann angemeldet).
	umbenennenFokus bool
	// nameHatteFokus: Erst ein Fokusverlust danach beendet die Umbenennung (feldTasten).
	nameHatteFokus bool

	knoepfe     [ebKnopfAnzahl]widget.Clickable
	knopfRahmen [ebKnopfAnzahl]image.Rectangle
	kopf        widget.Clickable // Überschrift: klappt den Abschnitt ein und aus
	zu          bool
	// autoZu: Die Leiste reicht nicht für Farbwähler und Ebenen; die Ebenen sind in diesem Bild eingeklappt (setzt
	// zeichneSeitenleiste). vorrang: Der Nutzer hat die Ebenen bei Platzmangel geöffnet, dafür klappt die Farbe ein.
	autoZu, vorrang bool

	liste   image.Rectangle // sichtbare Fläche der Liste beim letzten Zeichnen; der Zeiger ist das Ziel der Ereignisse
	zeilen  []ebenenZeileLage
	zeileH  int // Zeilenhöhe in Pixeln beim letzten Zeichnen
	augeB   int // Breite der Klickflächen von Auge und Schloss
	versatz int // geblättert, in Pixeln
	inhaltH int // Höhe aller Zeilen
	druck   ebenenDruck
	letzter struct {
		id   int
		zeit time.Duration
	}
	gezeigt  bool     // der Abschnitt stand beim letzten Zeichnen offen da (Tests)
	namen    []string // gezeigte Namen von oben nach unten, mit dem Bild (Tests)
	hinweisY int

	// zeigeGewaehlte: Die Liste blättert beim nächsten Zeichnen zur gewählten Ebene (Wahl per Tastatur).
	zeigeGewaehlte bool
}

// ebenenDruck ist ein laufender Druck auf eine Zeile: Wählen, bei Bewegung Ziehen zum Umordnen.
type ebenenDruck struct {
	aktiv  bool
	id     int
	startY float32
	zieht  bool
	slot   int // Einfügestelle in der Liste (0 = über der obersten Zeile), nur beim Ziehen
}

// knoepfe sind die Knöpfe des Abschnitts (Tastaturfokus, seitenKnoepfe.alle).
func (k *ebenenLeiste) alleKnoepfe() []*widget.Clickable {
	a := []*widget.Clickable{&k.kopf}
	for i := range k.knoepfe {
		a = append(a, &k.knoepfe[i])
	}
	return a
}

// ebenenName ist der Name einer Zeile: der eigene, sonst aus Art und Inhalt (ADR-0032 Punkt 2).
func (e *editor) ebenenName(z canvas.EbenenZeile) string {
	if z.Ebene.Name != "" {
		return z.Ebene.Name
	}
	switch v := z.Objekt.(type) {
	case document.Text:
		zeile, _, _ := strings.Cut(strings.TrimSpace(v.Inhalt), "\n")
		if utf8.RuneCountInString(zeile) > 24 {
			zeile = string([]rune(zeile)[:23]) + "…"
		}
		return fmt.Sprintf(i18n.T("Text “%s”"), document.Zeichenbar(zeile))
	case document.Nummer:
		return fmt.Sprintf(i18n.T("Step %d"), v.Wert)
	}
	return e.werkzeugName(z.Art)
}

// ebenenNameText ist der Name der Ebene in der Umbenennung (Ausgangspunkt des Felds).
func (e *editor) ebenenNameText() string {
	for _, z := range e.s.Ebenen() {
		if z.ID == e.ebenen.umbenennen {
			return e.ebenenName(z)
		}
	}
	return ""
}

// ebenenNameUnveraendert meldet, ob text der gezeigte Name ist: Enter ohne Änderung ist kein Fehler.
func (e *editor) ebenenNameUnveraendert(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" { // leer heißt „Name aus dem Inhalt“: ohne eigenen Namen ist das keine Änderung (F-587)
		eb, ok := e.s.GewaehlteEbene()
		return ok && eb.Name == "" && e.ebenen.umbenennen == e.s.Auswahl
	}
	return t == e.ebenenNameText()
}

// schalteAusgeblendet blendet id aus bzw. ein und meldet beim Ausblenden, dass der Inhalt in der Arbeitsdatei bleibt
// (F-586: der Hinweis der Leiste ist nicht immer zu sehen).
func (e *editor) schalteAusgeblendet(id int) {
	if !e.s.SchalteAusgeblendet(id) {
		return
	}
	if o, ok := e.s.D.Objekt(id); ok && o.Ebenenwerte().Ausgeblendet {
		e.melde(i18n.T("Layer hidden. It stays in the working file; redact to remove content for good."), false)
	}
}

// ebenenFeldWert ist der Text der Felder des Abschnitts ohne Eingabe; ok ist falsch für fremde Felder.
func (e *editor) ebenenFeldWert(f *eingabe) (string, bool) {
	switch f {
	case &e.ebenen.name:
		return e.ebenenNameText(), true
	case &e.ebenen.deck:
		eb, ok := e.s.GewaehlteEbene()
		if !ok {
			return "–", true
		}
		return fmt.Sprintf(i18n.TK("status bar", "%d%%"), eb.Deckkraft()), true
	}
	return "", false
}

// umbenennenBeginnen öffnet das Namensfeld an der gewählten Ebene (F2, Doppelklick). Ist die Leiste nicht zu sehen,
// öffnet es sie (eingeklappt) bzw. meldet, dass das Fenster zu schmal ist.
func (e *editor) umbenennenBeginnen(gtx layout.Context) {
	if !e.s.KannEbene() || !e.seiteAn() {
		return
	}
	if !seitePasst(gtx) {
		e.melde(i18n.T("The window is too narrow for the sidebar; widen it to rename a layer."), false)
		return
	}
	e.s.TextBestaetigen()
	if !e.s.KannEbene() { // eine leere Texteingabe war die Auswahl und ist weg (F-587)
		return
	}
	e.seiteZu, e.ebenen.zu = false, false
	k := &e.ebenen
	k.umbenennen, k.umbenennenFokus, k.zeigeGewaehlte, k.nameHatteFokus = e.s.Auswahl, true, true, false
	k.name.neu, k.name.text, k.name.fehler = true, "", false
}

// umbenennenEnde schließt das Namensfeld, wenn f es ist.
func (e *editor) umbenennenEnde(f *eingabe) {
	if f == &e.ebenen.name {
		e.ebenen.umbenennen, e.ebenen.umbenennenFokus, e.ebenen.nameHatteFokus = 0, false, false
	}
}

// ebenenAuswerten verarbeitet Knöpfe, Überschrift und Zeiger der Liste (aus seiteAuswerten).
func (e *editor) ebenenAuswerten(gtx layout.Context) {
	k := &e.ebenen
	an := e.seiteAn()
	if k.kopf.Clicked(gtx) {
		if k.autoZu { // bei Platzmangel: Ebenen öffnen, dafür die Farbe einklappen
			k.vorrang, k.zu = true, false
		} else {
			k.zu, k.vorrang = !k.zu, false
		}
	}
	for j := range k.knoepfe {
		if !k.knoepfe[j].Clicked(gtx) || !an {
			continue
		}
		switch j {
		case ebKnopfHoch:
			e.s.EbeneSchritt(1)
		case ebKnopfRunter:
			e.s.EbeneSchritt(-1)
		case ebKnopfDoppeln:
			e.s.Doppeln()
		case ebKnopfLoeschen:
			e.s.Loeschen()
		}
		k.zeigeGewaehlte = true
	}
	if e.ebenen.umbenennen != 0 {
		if _, ok := e.s.D.Objekt(e.ebenen.umbenennen); !ok { // gelöscht oder zurückgenommen
			e.umbenennenEnde(&e.ebenen.name)
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &k.liste,
			Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			break
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		e.ebenenZeiger(gtx, pe, an)
	}
}

// ebenenZeiger verarbeitet ein Zeigerereignis der Liste (Fensterpixel).
func (e *editor) ebenenZeiger(gtx layout.Context, pe pointer.Event, an bool) {
	k := &e.ebenen
	p := pe.Position.Round()
	switch pe.Kind {
	case pointer.Scroll:
		k.versatz = max(0, min(k.inhaltH-k.liste.Dy(), k.versatz+int(math.Round(float64(pe.Scroll.Y)))))
	case pointer.Press:
		if pe.Buttons != pointer.ButtonPrimary || !an {
			return
		}
		z, ok := k.zeileBei(p)
		if !ok || z.id == 0 {
			return // Bild: keine Aktionen
		}
		switch {
		case p.X < k.liste.Min.X+k.augeB:
			e.schalteAusgeblendet(z.id)
		case p.X >= k.liste.Max.X-k.augeB:
			e.s.SchalteGesperrt(z.id)
		default:
			doppel := k.letzter.id == z.id && pe.Time-k.letzter.zeit < doppelklickZeit // dieselbe Grenze wie auf der Leinwand
			k.letzter.id, k.letzter.zeit = z.id, pe.Time
			if !e.s.WaehleEbene(z.id) {
				return
			}
			if doppel {
				e.umbenennenBeginnen(gtx)
				return
			}
			k.druck = ebenenDruck{aktiv: true, id: z.id, startY: pe.Position.Y}
		}
	case pointer.Drag:
		if !k.druck.aktiv {
			return
		}
		if !k.druck.zieht && math.Abs(float64(pe.Position.Y-k.druck.startY)) >= float64(gtx.Dp(4)) {
			k.druck.zieht = true
		}
		if k.druck.zieht {
			// Am Rand der Liste blättert sie mit, damit auch ein weiter Weg in einer langen Liste geht (F-587).
			switch rand := gtx.Dp(8); {
			case p.Y < k.liste.Min.Y+rand:
				k.versatz = max(0, k.versatz-k.zeileH/3)
			case p.Y > k.liste.Max.Y-rand:
				k.versatz = max(0, min(k.inhaltH-k.liste.Dy(), k.versatz+k.zeileH/3))
			}
			k.druck.slot = k.slotBei(p.Y, len(e.s.Ebenen()))
		}
	case pointer.Release:
		d := k.druck
		k.druck = ebenenDruck{}
		if !d.zieht || !an {
			return
		}
		zeilen := e.s.Ebenen()
		von := -1
		for i, z := range zeilen {
			if z.ID == d.id {
				von = i
			}
		}
		if von < 0 {
			return
		}
		neu := d.slot
		if neu > von {
			neu--
		}
		// Liste oben = vorderste Ebene; das Dokument zählt von unten.
		e.s.EbeneNach(d.id, len(zeilen)-1-neu)
	case pointer.Cancel:
		k.druck = ebenenDruck{}
	}
}

// zeileBei liefert die gezeichnete Zeile unter p.
func (k *ebenenLeiste) zeileBei(p image.Point) (ebenenZeileLage, bool) {
	if !p.In(k.liste) {
		return ebenenZeileLage{}, false
	}
	for _, z := range k.zeilen {
		if p.In(z.rahmen) {
			return z, true
		}
	}
	return ebenenZeileLage{}, false
}

// slotBei ist die Einfügestelle für den Zeiger bei y (Fensterpixel): 0 über der obersten Zeile, n unter der untersten
// Ebene (über dem Bild).
func (k *ebenenLeiste) slotBei(y, n int) int {
	if k.zeileH <= 0 {
		return 0
	}
	rel := y - k.liste.Min.Y + k.versatz
	return max(0, min(n, (rel+k.zeileH/2)/k.zeileH))
}

// abschnittKopf zeichnet eine einklappbare Überschrift (Versalien, Dreieck) über die Breite und liefert ihre Höhe.
func (e *editor) abschnittKopf(gtx layout.Context, c *widget.Clickable, text string, zu bool, x, y, breite int) int {
	h := e.seiteText(gtx, strings.ToUpper(text), 12, farbeNebentext, image.Pt(x+gtx.Dp(14), y), breite-gtx.Dp(14), 1)
	m := image.Pt(x+gtx.Dp(5), y+h/2)
	var pf clip.Path
	pf.Begin(gtx.Ops)
	s := float32(gtx.Dp(4))
	mf := f32.Pt(float32(m.X), float32(m.Y))
	if zu { // ▸
		pf.MoveTo(mf.Add(f32.Pt(-s/2, -s)))
		pf.LineTo(mf.Add(f32.Pt(s/2+1, 0)))
		pf.LineTo(mf.Add(f32.Pt(-s/2, s)))
	} else { // ▾
		pf.MoveTo(mf.Add(f32.Pt(-s, -s/2)))
		pf.LineTo(mf.Add(f32.Pt(s, -s/2)))
		pf.LineTo(mf.Add(f32.Pt(0, s/2+1)))
	}
	pf.Close()
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Outline{Path: pf.End()}.Op())
	e.knopfAn(gtx, c, image.Rect(x, y-gtx.Dp(2), x+breite, y+h+gtx.Dp(2)))
	return h
}

// zeichneEbenen zeichnet den Abschnitt samt Trennlinie unter dem Inhalt darüber (endet bei y) und liefert sein Ende.
// Er hängt am unteren Rand der Leiste (unten), solange Platz ist: Wählt ein Klick in der Liste ein anderes Objekt, ändert
// sich der Inhalt darüber (Schrift, Strichstärke, Farbe), die Liste aber bleibt unter dem Zeiger stehen — sonst träfe
// der zweite Klick eines Doppelklicks oder ein Ziehen eine andere Zeile. Fehlt Platz, rückt er an y und die Liste
// schrumpft (mindestens eine Zeile, sie blättert).
func (e *editor) zeichneEbenen(gtx layout.Context, x, y, breite, spalte, unten int) int {
	k := &e.ebenen
	zu := k.zu || k.autoZu // vom Nutzer oder aus Platzmangel eingeklappt (Verify V23-2)
	k.gezeigt, k.namen, k.zeilen = false, nil, nil
	k.liste, k.deck.rahmen, k.name.rahmen = image.Rectangle{}, image.Rectangle{}, image.Rectangle{}
	k.knopfRahmen = [ebKnopfAnzahl]image.Rectangle{}
	fh := gtx.Dp(26)
	k.zeileH, k.augeB = gtx.Dp(ebeneZeileDp), gtx.Dp(ebeneAugeDp)
	zeilen := e.s.Ebenen()
	k.inhaltH = (len(zeilen) + 1) * k.zeileH
	hinweisH := gtx.Dp(46) // drei Zeilen Nebentext
	trennH, kopfH := gtx.Dp(28), gtx.Dp(28)
	listeH := max(k.zeileH, min(k.inhaltH, ebenenMaxZeilen*k.zeileH))
	rest := trennH + kopfH + fh + gtx.Dp(10) + gtx.Dp(8) + fh + gtx.Dp(8) + hinweisH // alles außer der Liste
	if zu {
		rest, listeH = trennH+kopfH, 0
	}
	oben := unten - gtx.Dp(8) - rest - listeH
	if oben < y {
		oben = y
		if !zu {
			listeH = max(k.zeileH, unten-gtx.Dp(8)-rest-y)
		}
	}
	ly := oben + trennH/2
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(x-gtx.Dp(8), ly, x+breite+gtx.Dp(8), ly+gtx.Dp(1))).Op())
	y = oben + trennH
	y += e.abschnittKopf(gtx, &k.kopf, i18n.T("Layers"), zu, x, y, breite) + gtx.Dp(12)
	if zu {
		return y
	}
	k.gezeigt = true

	// Deckkraft der gewählten Ebene
	e.seiteText(gtx, i18n.T("Opacity"), 13, farbeNebentext, image.Pt(x, y+gtx.Dp(5)), spalte-gtx.Dp(4), 1)
	k.deck.rahmen = image.Rect(x+spalte, y, x+spalte+gtx.Dp(70), y+fh)
	e.zeichneEingabe(gtx, &k.deck)
	y += fh + gtx.Dp(10)

	hinweis := i18n.T("Redaction is part of the image (permanent). Hidden layers stay in the working file.")
	x0 := x - gtx.Dp(8) // Zeilen reichen etwas über den Innenrand hinaus (Muster v2)
	k.liste = image.Rect(x0, y, x+breite+gtx.Dp(8), y+listeH)
	// Wahl per Tastatur oder Knopf: zur gewählten Zeile blättern.
	if k.zeigeGewaehlte {
		k.zeigeGewaehlte = false
		for i, z := range zeilen {
			if z.Gewaehlt {
				oben, unten := i*k.zeileH, (i+1)*k.zeileH
				if oben < k.versatz {
					k.versatz = oben
				} else if unten > k.versatz+k.liste.Dy() {
					k.versatz = unten - k.liste.Dy()
				}
			}
		}
	}
	k.versatz = max(0, min(k.versatz, k.inhaltH-k.liste.Dy()))

	lc := clip.Rect(k.liste).Push(gtx.Ops)
	for i := 0; i <= len(zeilen); i++ {
		r := image.Rect(k.liste.Min.X, y+i*k.zeileH-k.versatz, k.liste.Max.X, y+(i+1)*k.zeileH-k.versatz)
		if i < len(zeilen) {
			z := zeilen[i]
			name := e.ebenenName(z)
			k.namen = append(k.namen, name)
			if r.Max.Y > k.liste.Min.Y && r.Min.Y < k.liste.Max.Y {
				e.zeichneEbenenZeile(gtx, z, name, r)
			}
			k.zeilen = append(k.zeilen, ebenenZeileLage{id: z.ID, rahmen: r})
			continue
		}
		k.namen = append(k.namen, i18n.T("Image"))
		if r.Max.Y > k.liste.Min.Y && r.Min.Y < k.liste.Max.Y {
			e.zeichneBildZeile(gtx, r)
		}
		k.zeilen = append(k.zeilen, ebenenZeileLage{rahmen: r})
	}
	if k.druck.zieht { // Einfügestelle
		ly := y + k.druck.slot*k.zeileH - k.versatz
		paint.FillShape(gtx.Ops, farbeAkzent, clip.Rect(image.Rect(k.liste.Min.X, ly-gtx.Dp(1), k.liste.Max.X, ly+gtx.Dp(1))).Op())
	}
	// Ereignisfläche der Liste: unter den Feldern der Zeile (Namensfeld), weil vorher angemeldet.
	ec := clip.Rect(k.liste).Push(gtx.Ops)
	event.Op(gtx.Ops, &k.liste)
	ec.Pop()
	if e.ebenen.umbenennen != 0 {
		for _, z := range k.zeilen {
			if z.id == e.ebenen.umbenennen {
				k.name.rahmen = image.Rect(z.rahmen.Min.X+k.augeB+gtx.Dp(ebeneVorschauB)+gtx.Dp(10), z.rahmen.Min.Y+gtx.Dp(4),
					z.rahmen.Max.X-k.augeB, z.rahmen.Max.Y-gtx.Dp(4))
				e.zeichneEingabe(gtx, &k.name)
			}
		}
	}
	lc.Pop()
	y = k.liste.Max.Y + gtx.Dp(8)

	// Knöpfe ↑ ↓ Doppeln Löschen
	kann := []bool{e.s.KannEbeneSchritt(1), e.s.KannEbeneSchritt(-1), e.s.KannEbene(), e.s.KannLoeschen()}
	bx := x
	for j := 0; j < ebKnopfAnzahl; j++ {
		b := fh + gtx.Dp(4)
		text := ""
		switch j {
		case ebKnopfDoppeln:
			text = i18n.T("Duplicate")
		case ebKnopfLoeschen:
			text = i18n.T("Delete")
		}
		if text != "" {
			b = e.feldTextBreite(gtx, text, 13) + gtx.Dp(20)
		}
		r := image.Rect(bx, y, bx+b, y+fh)
		bx = r.Max.X + gtx.Dp(6)
		k.knopfRahmen[j] = r
		e.knopfFlaeche(gtx, &k.knoepfe[j], r, false, kann[j])
		vorn := farbeText
		if !kann[j] {
			vorn = farbeNebentext
		}
		switch j {
		case ebKnopfHoch:
			zeichneSymbol(gtx, icons.Hoch, r.Min.Add(image.Pt((r.Dx()-gtx.Dp(16))/2, (r.Dy()-gtx.Dp(16))/2)), gtx.Dp(16), vorn, vorn)
		case ebKnopfRunter:
			zeichneSymbol(gtx, icons.Runter, r.Min.Add(image.Pt((r.Dx()-gtx.Dp(16))/2, (r.Dy()-gtx.Dp(16))/2)), gtx.Dp(16), vorn, vorn)
		default:
			e.textMittig(gtx, text, 13, vorn, r)
		}
		e.knopfAn(gtx, &k.knoepfe[j], r)
	}
	y += fh + gtx.Dp(8)
	k.hinweisY = y
	y += e.seiteText(gtx, hinweis, 12, farbeNebentext, image.Pt(x, y), breite, 3)
	return y
}

// zeichneEbenenZeile zeichnet eine Ebene: Grund (gewählt grün), Auge, Vorschau, Name, Schloss.
func (e *editor) zeichneEbenenZeile(gtx layout.Context, z canvas.EbenenZeile, name string, r image.Rectangle) {
	k := &e.ebenen
	if z.Gewaehlt {
		paint.FillShape(gtx.Ops, farbeAuswahl, clip.Rect(r).Op())
	}
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(r.Min.X, r.Max.Y-gtx.Dp(1), r.Max.X, r.Max.Y)).Op())
	vorn := farbeText
	if z.Ebene.Ausgeblendet {
		vorn = farbeNebentext
	}
	s := gtx.Dp(16)
	auge := icons.Sichtbar
	if z.Ebene.Ausgeblendet {
		auge = icons.Verborgen
	}
	zeichneSymbol(gtx, auge, image.Pt(r.Min.X+(k.augeB-s)/2+gtx.Dp(2), r.Min.Y+(r.Dy()-s)/2), s, vorn, vorn)
	v := image.Rect(r.Min.X+k.augeB+gtx.Dp(4), r.Min.Y+(r.Dy()-gtx.Dp(ebeneVorschauH))/2,
		r.Min.X+k.augeB+gtx.Dp(4)+gtx.Dp(ebeneVorschauB), r.Min.Y+(r.Dy()+gtx.Dp(ebeneVorschauH))/2)
	e.zeichneVorschau(gtx, z.Objekt, v)
	if z.ID != k.umbenennen {
		e.seiteText(gtx, name, 13, vorn, image.Pt(v.Max.X+gtx.Dp(10), r.Min.Y+(r.Dy()-gtx.Dp(17))/2),
			r.Max.X-k.augeB-v.Max.X-gtx.Dp(12), 1)
	}
	schloss := farbeLinie
	if z.Ebene.Gesperrt {
		schloss = farbeText
	}
	zeichneSymbol(gtx, icons.Sperren, image.Pt(r.Max.X-k.augeB+(k.augeB-s)/2-gtx.Dp(2), r.Min.Y+(r.Dy()-s)/2), s, schloss, schloss)
}

// zeichneBildZeile zeichnet die unterste Zeile: das Bild mit Schloss, ohne Aktionen.
func (e *editor) zeichneBildZeile(gtx layout.Context, r image.Rectangle) {
	k := &e.ebenen
	v := image.Rect(r.Min.X+k.augeB+gtx.Dp(4), r.Min.Y+(r.Dy()-gtx.Dp(ebeneVorschauH))/2,
		r.Min.X+k.augeB+gtx.Dp(4)+gtx.Dp(ebeneVorschauB), r.Min.Y+(r.Dy()+gtx.Dp(ebeneVorschauH))/2)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, clip.Rect(v).Op())
	if g := e.bildOp.Size(); g.X > 0 && g.Y > 0 {
		f := math.Min(float64(v.Dx())/float64(g.X), float64(v.Dy())/float64(g.Y))
		b := image.Pt(int(math.Round(float64(g.X)*f)), int(math.Round(float64(g.Y)*f)))
		ziel := image.Rectangle{Min: v.Min.Add(v.Size().Sub(b).Div(2))}
		ziel.Max = ziel.Min.Add(b)
		bildIn(gtx, e.bildOp, ziel, paint.FilterLinear)
	}
	paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(v).Path(), Width: float32(gtx.Dp(1))}.Op())
	e.seiteText(gtx, i18n.T("Image"), 13, farbeText, image.Pt(v.Max.X+gtx.Dp(10), r.Min.Y+(r.Dy()-gtx.Dp(17))/2),
		r.Max.X-k.augeB-v.Max.X-gtx.Dp(12), 1)
	s := gtx.Dp(16)
	zeichneSymbol(gtx, icons.Sperren, image.Pt(r.Max.X-k.augeB+(k.augeB-s)/2-gtx.Dp(2), r.Min.Y+(r.Dy()-s)/2), s,
		farbeNebentext, farbeNebentext)
}

// zeichneVorschau zeichnet o verkleinert und mittig in v auf hellem Grund, mit denselben Zeichenwegen wie die Leinwand
// (zeichneObjekt), ohne Stauchen und Zuschnitt: die Vorschau zeigt das ganze Objekt.
func (e *editor) zeichneVorschau(gtx layout.Context, o document.Objekt, v image.Rectangle) {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, clip.Rect(v).Op())
	defer func() {
		paint.FillShape(gtx.Ops, farbeLinie, clip.Stroke{Path: clip.Rect(v).Path(), Width: float32(gtx.Dp(1))}.Op())
	}()
	g := o.Grenzen()
	if p, ok := o.(document.Pfeil); ok { // die Spitze ragt über die Grenzen hinaus
		m := math.Max(8, 4*p.Strich.Breite) / 2
		g = document.Rechteck{Min: g.Min.Minus(document.Pt(m, m)), Max: g.Max.Plus(document.Pt(m, m))}
	}
	gb, gh := g.Breite(), g.Hoehe()
	if !(gb > 0) || !(gh > 0) {
		return
	}
	innen := v.Inset(gtx.Dp(3))
	f := math.Min(float64(innen.Dx())/gb, float64(innen.Dy())/gh)
	f = math.Min(f, 4) // winzige Objekte nicht aufblähen
	ox := float64(innen.Min.X) + (float64(innen.Dx())-gb*f)/2 - g.Min.X*f
	oy := float64(innen.Min.Y) + (float64(innen.Dy())-gh*f)/2 - g.Min.Y*f
	// Dünne Striche verschwänden in der Verkleinerung: mindestens 1,5 Pixel der Vorschau (nur hier, nicht im Dokument).
	switch w := o.(type) {
	case document.Pfeil:
		w.Strich.Breite = math.Max(w.Strich.Breite, 1.5/f)
		o = w
	case document.Kasten:
		w.Strich.Breite = math.Max(w.Strich.Breite, 1.5/f)
		o = w
	case document.Lupe:
		w.Strich.Breite = math.Max(w.Strich.Breite, 1.5/f)
		o = w
	}
	c := clip.Rect(v).Push(gtx.Ops)
	t := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(f), float32(f))).
		Offset(f32.Pt(float32(ox), float32(oy)))).Push(gtx.Ops)
	an := func(p document.Punkt) f32.Point { return f32.Pt(float32(p.X), float32(p.Y)) }
	sicht := image.Rect(int(math.Floor(g.Min.X)), int(math.Floor(g.Min.Y)), int(math.Ceil(g.Max.X)), int(math.Ceil(g.Max.Y)))
	e.zeichneObjekt(gtx, o, an, sicht, true)
	t.Pop()
	c.Pop()
}
