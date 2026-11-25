// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Leiste der letzten Aufnahmen (ADR-0036 Punkte 1, 2, 6 und 7): unten zwischen Leinwand und Fußzeile, so breit wie die
// Leinwand. Vorschaubilder der Bibliothek, neueste links, je mit Zeitangabe; die offene Aufnahme grün umrandet. Klick
// oder Enter öffnet eine Aufnahme im selben Fenster (wechsel.go), Mausrad rollt waagerecht, Tab und Pfeile wandern.
// Liste, Fokus, Rollen, Zeitangabe und Speicher der Vorschauen stehen ohne Gio in internal/libview.

import (
	"fmt"
	"image"
	"path/filepath"
	"time"

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

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/libview"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// Maße der Leiste in dp (ADR-0036 Punkt 1, Muster 174 a).
const (
	leisteUntenDp  = 118 // Höhe; zwei Zeilen unter der Vorschau: Zeit und Auflösung (Karte #311)
	leisteKachelB  = 120 // Vorschau
	leisteKachelH  = 68
	leisteRandDp   = 12 // links und rechts der Spur
	leisteLueckeDp = 12 // zwischen zwei Vorschauen
	leisteObenDp   = 8  // über den Vorschauen
	// leisteMinLeinwandDp: Die Leiste erscheint nur, wenn daneben mindestens so viel Leinwandhöhe bleibt (ADR-0036
	// Punkt 2); sonst blendet sie sich aus, ohne die Einstellung zu ändern.
	leisteMinLeinwandDp = 420
)

// ladeLeiste liest [library] bar beim Öffnen des Editors (ADR-0018 Nachtrag 6), wie die Tafel die Datei frisch liest;
// Tests setzen XDG_CONFIG_HOME (TestMain).
var ladeLeiste = func() bool {
	w, _ := settings.Lade() // Warnungen hat der Start schon ausgegeben
	return w.Leiste
}

// leisteUhr ist die Uhr der Zeitangaben; Naht für Tests.
var leisteUhr = time.Now

// untenLeiste ist der Zustand der Leiste im Editor.
type untenLeiste struct {
	// abgleich: das nächste Lesen gleicht den Index vorher mit der Ablage ab (library.Abgleich: Einträge ohne Datei
	// fallen weg, etwa im Dateimanager gelöscht; F-672). Beim Start und nach einem gescheiterten Öffnen.
	abgleich bool
	an       bool // eingeschaltet ([library] bar bzw. Ansicht → Recent captures)
	l        *libview.Leiste
	v        *libview.Vorschauen
	knoepfe  [libview.MaxEintraege]widget.Clickable
	// loesch sind die ✕ oben rechts in den Vorschauen (Karte #312): sichtbar unter dem Zeiger oder nach langem Druck.
	loesch       [libview.MaxEintraege]widget.Clickable
	loeschRahmen []image.Rectangle // ✕ je gezeichneter Stelle von bis bis, leer ohne ✕ (Tests)
	langDruck    int               // Stelle, deren ✕ ein langer Druck zeigt; -1: keine
	// bibKnopf ist der Knopf „Library“ links in der Leiste: öffnet die volle Ansicht (ADR-0036 Punkte 1 und 3).
	bibKnopf widget.Clickable
	rad      int // Ziel der Mausrad-Ereignisse (nur die Adresse zählt)
	masse    libview.Masse

	// Index im Hintergrund (ADR-0036 Punkt 7): höchstens ein Lesen zugleich; nochmal heißt, nach dem laufenden erneut.
	index        chan indexErgebnis
	indexLaeuft  bool
	indexNochmal bool
	indexFehler  bool

	// Stand beim letzten Zeichnen: Klicks beziehen sich auf die Vorschauen, die der Nutzer gesehen hat.
	gezeigt   []string          // Pfade relativ zur Ablage je Stelle
	rahmen    image.Rectangle   // Lage der Leiste im Fenster (leer: nicht gezeigt)
	kacheln   []image.Rectangle // Lage der gezeichneten Vorschauen je Stelle von von bis bis (Tests)
	von, bis  int               // gezeichnete Stellen
	ops       map[string]leisteBild
	fokusSoll int    // Stelle, die den Tastaturfokus bekommen soll, sobald sie angemeldet ist; -1: keine
	fokusHier int    // Stelle, deren Knopf beim letzten Zeichnen den Tastaturfokus hatte; -1: keine
	hinweis   string // Text für die Statuszeile zur Vorschau mit Fokus oder Zeiger
}

// leisteBild ist die Bild-Op eines Vorschaubilds, gemerkt, damit Gio die Textur nicht je Bild neu hochlädt.
type leisteBild struct {
	bild *image.NRGBA
	op   paint.ImageOp
}

type indexErgebnis struct {
	es  []library.Eintrag
	err error
}

// alle sind die Knöpfe der Leiste in Tab-Reihenfolge (editor.fokussierterKnopf).
func (k *untenLeiste) alle() []*widget.Clickable {
	var a []*widget.Clickable
	if !k.rahmen.Empty() {
		a = append(a, &k.bibKnopf)
	}
	for i := k.von; i < k.bis && i < len(k.knoepfe); i++ {
		a = append(a, &k.knoepfe[i])
	}
	return a
}

// stelle liefert die Stelle des Knopfs c in der Leiste, -1 wenn c keiner ist.
func (k *untenLeiste) stelle(c *widget.Clickable) int {
	for i := range k.knoepfe {
		if c == &k.knoepfe[i] {
			return i
		}
	}
	return -1
}

// ziele sind die Knöpfe, an denen die Leertaste wie Enter wirkt (leertastenZiele).
func (k *untenLeiste) ziele() []event.Tag {
	var z []event.Tag
	for _, c := range k.alle() {
		z = append(z, c)
	}
	return z
}

// leisteStart richtet die Leiste ein (Editor mit Anbindung): Schalter aus der Einstellung, Index im Hintergrund lesen.
func (e *editor) leisteStart() {
	k := &e.unten
	k.fokusSoll, k.fokusHier, k.langDruck = -1, -1, -1
	if e.bib == nil || e.bib.Bibliothek == nil {
		return
	}
	k.an = ladeLeiste() && !probeLeisteAus
	k.l = libview.NeueLeiste()
	wecke := func() {}
	if e.wecker != nil {
		w := e.wecker
		wecke = w.Anfordern
	} else if e.w != nil {
		wecke = e.w.Invalidate
	}
	k.v = libview.NeueVorschauen(libview.LaderFuer(e.bib.Bibliothek), wecke, libview.SpeicherGrenze)
	k.v.SetzeStand(libview.StandFuer(e.bib.Bibliothek)) // fremdes Speichern erkennen (F-677)
	e.leisteAktuell()
	e.bibliothekStart()
	k.abgleich = true
	e.leisteNeuLesen()
}

// leisteAktuell bestimmt die offene Aufnahme relativ zur Ablage (grüner Rand); eine Datei von außen ist keine.
func (e *editor) leisteAktuell() {
	k := &e.unten
	if k.l == nil {
		return
	}
	rel := ""
	if e.pfad != nil {
		if r, ok := e.bib.Bibliothek.Relativ(e.pfad()); ok {
			rel = r
		}
	}
	k.l.SetzeAktuell(rel)
}

// leisteNeuLesen liest den Index im Hintergrund (library.Suchen ohne Agenten-Aufnahmen); läuft schon ein Lesen, folgt
// eins danach.
func (e *editor) leisteNeuLesen() {
	k := &e.unten
	if k.l == nil {
		return
	}
	if k.indexLaeuft {
		k.indexNochmal = true
		return
	}
	k.indexLaeuft = true
	if k.index == nil {
		k.index = make(chan indexErgebnis, 1)
	}
	b, ch, wecker, abgleich := e.bib.Bibliothek, k.index, e.wecker, k.abgleich
	k.abgleich = false
	go func() {
		if abgleich {
			gleicheAb(b)
		}
		es, err := b.Suchen(library.Anfrage{OhneAgenten: true})
		ch <- indexErgebnis{es, err}
		if wecker != nil {
			wecker.Anfordern()
		}
	}()
}

// leisteIndexAbholen nimmt einen gelesenen Index an (Ereignisschleife); liefert, ob einer da war.
func (e *editor) leisteIndexAbholen() bool {
	k := &e.unten
	if !k.indexLaeuft {
		return false
	}
	select {
	case r := <-k.index:
		k.indexLaeuft = false
		k.indexFehler = r.err != nil
		if r.err != nil {
			diagnoseEditor("Index der Leiste: %v", r.err)
		} else {
			vorher := k.l.Fokus()
			k.l.Setze(r.es)
			k.langDruck = -1 // die Stelle zeigt nach dem Neulesen womöglich eine andere Aufnahme (K-3)
			k.v.Pruefe()     // eine andere Instanz kann eine Aufnahme inzwischen geschwärzt haben (F-677)
			// Setze hält den Fokus bei derselben Aufnahme; der Tastaturfokus von Gio hängt aber am Knopf der Stelle. Hatte
			// ein Knopf der Leiste ihn, folgt er der Aufnahme an ihre neue Stelle (F-656) — sonst öffnete Enter nach einer
			// neuen Aufnahme vorn die Nachbarin.
			if f := k.l.Fokus(); f >= 0 && f != vorher && vorher >= 0 && k.fokusHier == vorher {
				k.fokusSoll = f
			}
			if i := k.aktuelleStelle(); i >= 0 && k.masse.Breite > 0 {
				k.l.Zeige(i, k.masse)
			}
		}
		if k.indexNochmal {
			k.indexNochmal = false
			e.leisteNeuLesen()
		}
		return true
	default:
		return false
	}
}

// aktuelleStelle ist die Stelle der offenen Aufnahme in der Liste, -1 ohne.
func (k *untenLeiste) aktuelleStelle() int {
	for i := range k.l.Eintraege() {
		if k.l.IstAktuell(i) {
			return i
		}
	}
	return -1
}

// leisteNachSpeichern: Nach einem eigenen Speichern ist die Vorschau neu geschrieben (vielleicht als neue Aufnahme):
// offene Aufnahme neu bestimmen, ihr Bild neu laden, Index neu lesen (ADR-0036 Punkt 7).
func (e *editor) leisteNachSpeichern() {
	k := &e.unten
	if k.l == nil {
		return
	}
	if alt := k.l.Aktuell(); alt != "" {
		k.v.Vergiss(alt)
	}
	e.leisteAktuell()
	if neu := k.l.Aktuell(); neu != "" {
		k.v.Vergiss(neu)
	}
	e.leisteNeuLesen()
	e.bibliothekNachSpeichern()
}

// leisteNachWechsel: eine andere Aufnahme ist offen.
func (e *editor) leisteNachWechsel() {
	if e.unten.l == nil {
		return
	}
	e.leisteAktuell()
	e.leisteNeuLesen()
}

// leistePasst meldet, ob das Fenster hoch genug für Leiste und Mindest-Leinwand ist.
func leistePasst(gtx layout.Context) bool {
	return gtx.Constraints.Max.Y >= gtx.Dp(menuezeileDp+fusszeileDp+leisteUntenDp+leisteMinLeinwandDp)
}

// leisteSichtbar meldet, ob die Leiste gezeigt wird: mit Bibliothek, eingeschaltet und hoch genug.
func (e *editor) leisteSichtbar(gtx layout.Context) bool {
	return e.unten.l != nil && e.unten.an && leistePasst(gtx)
}

// leisteAn meldet, ob Klicks und Tasten der Leiste wirken (wie die rechte Leiste: nicht während des Speicherns und
// unter der Hilfe; dazu nicht, solange ein Wechsel lädt).
func (e *editor) leisteAn() bool {
	return e.seiteAn() && !e.wechsel.laedt && !e.einst.offen && !e.bibl.arbeitLaeuft // F-676
}

// schalteLeiste schaltet die Leiste ein oder aus und merkt es in settings.ini (ADR-0036 Punkt 2). Im Probe-Lauf eines
// Drehbuchs (probeLeisteAus) wird nichts gemerkt, sonst kippte der nächste Lauf (Lehre aus F-539).
func (e *editor) schalteLeiste(gtx layout.Context) {
	k := &e.unten
	if k.l == nil {
		return
	}
	k.an = !k.an
	if !k.an {
		for _, c := range k.alle() {
			if gtx.Focused(c) {
				gtx.Execute(key.FocusCmd{Tag: e}) // der Knopf mit Fokus verschwindet
			}
		}
	}
	if probeLeisteAus {
		return
	}
	wert := "no"
	if k.an {
		wert = "yes"
	}
	if err := merkeEinstellung(settings.Aenderung{Abschnitt: "library", Schluessel: "bar", Wert: wert}); err != nil {
		e.melde(fmt.Sprintf(i18n.T("Could not remember the setting: %v"), fehlerGrund(err)), true)
	}
}

// leisteGrau ist die Erklärung am grauen Menüeintrag, wenn das Fenster keine Bibliothek kennt (ADR-0026 Nachtrag 8).
func (e *editor) leisteGrau() string {
	if e.unten.l != nil {
		return ""
	}
	return i18n.T("Not available in this window: it has no library.")
}

// leisteMasseFuer rechnet die Maße der Spur in Pixeln für die sichtbare Breite breite.
func leisteMasseFuer(gtx layout.Context, breite int) libview.Masse {
	return libview.Masse{Rand: gtx.Dp(leisteRandDp), Kachel: gtx.Dp(leisteKachelB), Luecke: gtx.Dp(leisteLueckeDp), Breite: breite}
}

// leisteAuswerten verarbeitet Klicks und Mausrad (vor dem Zeichnen, wie die anderen Leisten).
func (e *editor) leisteAuswerten(gtx layout.Context) {
	k := &e.unten
	if k.l == nil {
		return
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &k.rad, Kinds: pointer.Scroll,
			ScrollX: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Scroll {
			k.l.Rad(pe.Scroll.Y+pe.Scroll.X, k.masse) // senkrechtes Rad rollt waagerecht (ADR-0036 Punkt 1)
		}
	}
	if k.bibKnopf.Clicked(gtx) && e.leisteAn() {
		e.bibliothekOeffnen(gtx)
	}
	for i := range k.gezeigt {
		if k.loesch[i].Clicked(gtx) && e.leisteAn() && !e.loeschRuht() {
			k.langDruck = -1
			e.leistePapierkorb(i)
			continue
		}
		if k.knoepfe[i].Clicked(gtx) && e.leisteAn() && !e.loeschRuht() { // zweiter Klick nach dem ✕ öffnet nichts (K-2)
			if langerDruck(k.knoepfe[i].History()) {
				k.langDruck = i // zeigt das ✕, statt zu öffnen
				continue
			}
			k.langDruck = -1
			e.leisteOeffne(i)
		}
	}
}

// langerDruck meldet, ob der letzte beendete Druck der Historie mindestens langDruckZeit dauerte (Karte #312).
func langerDruck(h []widget.Press) bool {
	if len(h) == 0 {
		return false
	}
	p := h[len(h)-1]
	return !p.Cancelled && !p.End.IsZero() && p.End.Sub(p.Start) >= langDruckZeit
}

// leistePapierkorb legt die Aufnahme an der Stelle i in den Papierkorb (✕, Entf); die offene ist ausgenommen.
func (e *editor) leistePapierkorb(i int) {
	k := &e.unten
	if i < 0 || i >= len(k.gezeigt) || k.gezeigt[i] == "" {
		return
	}
	abs := filepath.Join(e.bib.Bibliothek.Orte.Ablage, filepath.FromSlash(k.gezeigt[i]))
	e.papierkorbAnfrage(k.gezeigt[i], k.l.IstAktuell(i) || e.istOffen(abs))
}

// leisteOeffne öffnet die Aufnahme an der Stelle i, wie sie beim letzten Zeichnen stand. Die offene übergeht
// wechselAnfrage (eine Stelle für Leiste und volle Ansicht).
func (e *editor) leisteOeffne(i int) {
	k := &e.unten
	if i < 0 || i >= len(k.gezeigt) || k.gezeigt[i] == "" {
		return
	}
	e.wechselAnfrage(filepath.Join(e.bib.Bibliothek.Orte.Ablage, filepath.FromSlash(k.gezeigt[i])))
}

// leisteTaste behandelt Pfeile links und rechts, Pos1 und Ende, solange eine Vorschau den Fokus hat; Enter und
// Leertaste laufen über knopfPerTaste. Liefert, ob die Taste verbraucht ist.
func (e *editor) leisteTaste(gtx layout.Context, taste key.Event) bool {
	k := &e.unten
	if k.l == nil || taste.Modifiers != 0 {
		return false
	}
	i := -1
	for _, c := range k.alle() {
		if gtx.Focused(c) {
			i = k.stelle(c)
		}
	}
	if i < 0 {
		return false
	}
	var s libview.Schritt
	switch taste.Name {
	case key.NameDeleteForward: // in den Papierkorb, wie in der vollen Ansicht (Karte #312)
		e.leistePapierkorb(i)
		return true
	case key.NameLeftArrow:
		s = libview.SchrittLinks
	case key.NameRightArrow:
		s = libview.SchrittRechts
	case key.NameHome:
		s = libview.SchrittAnfang
	case key.NameEnd:
		s = libview.SchrittEnde
	default:
		return false
	}
	ziel := k.l.Bewege(i, s)
	if ziel >= 0 && ziel != i {
		k.l.SetzeFokus(ziel)
		k.l.Zeige(ziel, k.masse)
		k.fokusSoll = ziel
		gtx.Execute(op.InvalidateCmd{})
	}
	return true
}

// zeichneLeisteUnten zeichnet die Leiste in r (Fensterpixel).
func (e *editor) zeichneLeisteUnten(gtx layout.Context, r image.Rectangle) {
	k := &e.unten
	k.rahmen, k.kacheln, k.hinweis, k.von, k.bis, k.loeschRahmen = r, nil, "", 0, 0, k.loeschRahmen[:0]
	paint.FillShape(gtx.Ops, farbeLeisteVoll, clip.Rect(r).Op())
	paint.FillShape(gtx.Ops, farbeLinie, clip.Rect(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+gtx.Dp(1))).Op())
	// Links der Knopf „Library“ (Teil 2, Muster 174 a), dahinter die Spur der Vorschauen.
	kn := image.Rect(r.Min.X+gtx.Dp(leisteKnopfRand), r.Min.Y+gtx.Dp(leisteObenDp), r.Min.X+gtx.Dp(leisteKnopfRand+leisteKnopfB),
		r.Min.Y+gtx.Dp(leisteObenDp+leisteKachelH))
	spur := image.Rect(kn.Max.X, r.Min.Y, max(kn.Max.X, r.Max.X), r.Max.Y)
	k.masse = leisteMasseFuer(gtx, spur.Dx())
	es := k.l.Eintraege()
	k.gezeigt = k.gezeigt[:0]
	for _, eintrag := range es {
		k.gezeigt = append(k.gezeigt, eintrag.Pfad)
	}
	// Das Mausrad ist im Ausschnitt der Leiste vor den Vorschauen angemeldet: Gio gibt es so auch dann an die Leiste,
	// wenn eine Vorschau unter dem Zeiger liegt (Lehre aus F-598).
	c := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, &k.rad)
	e.zeichneLeisteKnopf(gtx, kn)
	r = spur
	cs := clip.Rect(spur).Push(gtx.Ops) // gerollte Vorschauen laufen nicht über den Knopf
	if len(es) == 0 {
		text := i18n.T("No captures in the library yet.")
		if k.indexLaeuft {
			text = ""
		} else if k.indexFehler {
			text = i18n.T("The library index could not be read.")
		}
		e.leisteText(gtx, text, image.Pt(r.Min.X+k.masse.Rand, r.Min.Y+gtx.Dp(leisteObenDp+leisteKachelH/2)), r.Dx(), false)
		cs.Pop()
		c.Pop()
		k.ops = nil
		return
	}
	if f := k.fokusSoll; f >= 0 && f < len(es) {
		k.l.Zeige(f, k.masse) // vor dem Zeichnen: die Vorschau muss angemeldet werden, damit sie den Fokus bekommt
	}
	k.von, k.bis = k.l.Sichtbar(k.masse)
	k.fokusHier = -1
	jetzt := leisteUhr()
	neueOps := map[string]leisteBild{}
	if e.unten.v != nil {
		e.unten.v.Runde() // Aufträge weggerollter Vorschauen verfallen (F-669)
	}
	y0 := r.Min.Y + gtx.Dp(leisteObenDp)
	for i := k.von; i < k.bis; i++ {
		x0, x1 := k.l.Lage(i, k.masse)
		kr := image.Rect(r.Min.X+x0, y0, r.Min.X+x1, y0+gtx.Dp(leisteKachelH))
		k.kacheln = append(k.kacheln, kr)
		eintrag := es[i]
		aktuell := k.l.IstAktuell(i)
		e.zeichneLeistenBild(gtx, eintrag.Pfad, kr, neueOps)
		switch {
		case aktuell:
			paint.FillShape(gtx.Ops, farbeAuswahl, clip.Stroke{Path: clip.Rect(kr.Inset(gtx.Dp(1))).Path(),
				Width: float32(gtx.Dp(2))}.Op())
		case k.knoepfe[i].Hovered():
			paint.FillShape(gtx.Ops, farbeNebentext, clip.Stroke{Path: clip.Rect(kr).Path(), Width: float32(gtx.Dp(1))}.Op())
		}
		seiteKnopf(gtx, &k.knoepfe[i], kr)
		// ✕ oben rechts (Karte #312), über der Vorschau angemeldet, damit es den Klick bekommt.
		var lr image.Rectangle
		if !aktuell && !e.loeschRuht() && (k.knoepfe[i].Hovered() || k.loesch[i].Hovered() || k.langDruck == i) {
			lr = loeschRahmen(gtx, kr)
			zeichneLoeschX(gtx, lr, k.loesch[i].Hovered())
			seiteKnopf(gtx, &k.loesch[i], lr)
			if k.loesch[i].Hovered() {
				k.hinweis = loeschHinweis()
			}
		}
		k.loeschRahmen = append(k.loeschRahmen, lr)
		fokus := gtx.Focused(&k.knoepfe[i])
		if fokus {
			k.fokusHier = i
		}
		if fokus && k.l.Fokus() != i {
			k.l.SetzeFokus(i) // per Tab hierher: ganz zeigen
			k.l.Zeige(i, k.masse)
			gtx.Execute(op.InvalidateCmd{})
		}
		if k.fokusSoll == i {
			if fokus {
				k.fokusSoll = -1
			} else {
				gtx.Execute(key.FocusCmd{Tag: &k.knoepfe[i]})
				gtx.Execute(op.InvalidateCmd{})
			}
		}
		farbe := farbeNebentext
		if aktuell {
			farbe = farbeText
		}
		e.leisteText(gtx, libview.Zeitangabe(eintrag.Erstellt, jetzt), image.Pt(kr.Min.X, kr.Max.Y+gtx.Dp(4)), kr.Dx(), farbe == farbeText)
		// Auflösung der Ansicht darunter, immer im Nebentext (Karte #311).
		e.leisteText(gtx, leisteAufloesung(eintrag), image.Pt(kr.Min.X, kr.Max.Y+gtx.Dp(18)), kr.Dx(), false)
		if fokus || (k.hinweis == "" && k.knoepfe[i].Hovered()) {
			k.hinweis = leisteHinweis(eintrag, aktuell)
		}
	}
	cs.Pop()
	c.Pop()
	k.ops = neueOps
}

// leisteAufloesung ist die zweite Zeile unter einer Vorschau: Breite × Höhe der Ansicht aus dem Index.
func leisteAufloesung(e library.Eintrag) string {
	if e.Breite <= 0 || e.Hoehe <= 0 {
		return ""
	}
	return fmt.Sprintf(i18n.TK("status bar", "%d × %d"), e.Breite, e.Hoehe)
}

// leisteVerborgen räumt den Stand des letzten Zeichnens ab, wenn die Leiste nicht gezeigt wird: kein Hinweis in der
// Statuszeile, keine Vorschau für Klicks und die Fokusfahrt.
func (e *editor) leisteVerborgen() {
	k := &e.unten
	k.rahmen, k.von, k.bis, k.kacheln, k.gezeigt, k.hinweis = image.Rectangle{}, 0, 0, nil, nil, ""
}

// zeichneLeistenBild zeichnet das Vorschaubild von rel eingepasst in r, bis es geladen ist ein graues Feld.
func (e *editor) zeichneLeistenBild(gtx layout.Context, rel string, r image.Rectangle, ops map[string]leisteBild) {
	zeichneVorschau(gtx, e.unten.v, e.unten.ops, rel, r, ops)
}

// leisteText setzt eine Zeile ab der linken oberen Ecke p, höchstens breite Pixel breit.
func (e *editor) leisteText(gtx layout.Context, s string, p image.Point, breite int, hell bool) {
	if s == "" {
		return
	}
	lbl := material.Label(e.thema, unit.Sp(11), s)
	lbl.Color = farbeNebentext
	if hell {
		lbl.Color = farbeText
	}
	lbl.MaxLines = 1
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, breite), gtx.Dp(20))}
	t := op.Offset(p).Push(gtx.Ops)
	lbl.Layout(g)
	t.Pop()
}

// leisteHinweis ist der Text der Statuszeile zu einer Vorschau: Datum, Uhrzeit und Größe, bei der offenen der Hinweis,
// dass sie offen ist.
func leisteHinweis(e library.Eintrag, aktuell bool) string {
	zeit := e.Erstellt.Local().Format(i18n.TK("date format", "2006-01-02 15:04"))
	if aktuell {
		return fmt.Sprintf(i18n.T("Capture of %s, %d × %d px — open now"), zeit, e.Breite, e.Hoehe)
	}
	return fmt.Sprintf(i18n.T("Capture of %s, %d × %d px — click or Enter opens it here (this one is saved first)"), zeit, e.Breite, e.Hoehe)
}
