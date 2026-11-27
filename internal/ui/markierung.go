// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Mehrere Vorschauen markieren (ADR-0036 Zusatz Karte #361): Strg+Klick schaltet eine Vorschau, Umschalt+Klick markiert
// den Bereich ab dem Anker, Strg+Umschalt+Klick fügt ihn hinzu; Strg+Leertaste, Umschalt+Pfeile, Strg+A und Esc per
// Tastatur. Leiste und volle Ansicht haben je eine eigene Markierung (libview.Markierung in libview.Leiste bzw.
// libview.Raster); sie gilt nur für Gezeigtes. Entf, Umschalt+F10 und das Kontextmenü gelten der Markierung, wenn die
// Vorschau markiert ist; das ✕ immer nur seiner Vorschau.

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"slices"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// markiereKlick wendet einen Klick mit Umschaltern mods auf die Aufnahme rel der Liste es an: Strg schaltet,
// Umschalt markiert den Bereich ab dem Anker (mit Strg dazu). Ohne Umschalter hebt er die Markierung auf, setzt den
// Anker und liefert false: Der Klick wirkt dann wie bisher (öffnen bzw. Fokus).
func markiereKlick(m *libview.Markierung, es []library.Eintrag, rel string, mods key.Modifiers) bool {
	strg, umsch := mods.Contain(key.ModShortcut), mods.Contain(key.ModShift)
	switch {
	case umsch:
		m.Bereich(es, rel, strg)
	case strg:
		m.Umschalte(rel)
	default:
		m.Leere()
		m.SetzeAnker(rel)
		return false
	}
	return true
}

// leisteMarkiereKlick: Klick auf die Vorschau i der Leiste mit Umschaltern; liefert, ob er markiert hat (dann öffnet er
// nicht). Die Vorschau bekommt den Tastaturfokus, damit Strg+Leertaste und Umschalt+Pfeile dort weitermachen.
func (e *editor) leisteMarkiereKlick(i int, mods key.Modifiers) bool {
	k := &e.unten
	if i < 0 || i >= len(k.gezeigt) || k.gezeigt[i] == "" {
		return false
	}
	if !markiereKlick(k.l.Markierung(), k.l.Eintraege(), k.gezeigt[i], mods) {
		return false
	}
	k.l.SetzeFokus(i)
	k.fokusSoll = i
	return true
}

// markierTasten sind die Tasten der Markierung an einer Vorschau der Leiste (ADR-0036 Zusatz Karte #361 Punkt 3): an
// den Knopf mit Fokus gebunden, Bedienung der Liste wie die Menünavigation, keine Kürzel der Tabelle.
func (k *untenLeiste) markierTasten() []key.Filter {
	var f []key.Filter
	for i := k.von; i < k.bis && i < len(k.knoepfe); i++ {
		c := event.Tag(&k.knoepfe[i])
		for _, n := range []key.Name{key.NameLeftArrow, key.NameRightArrow, key.NameHome, key.NameEnd} {
			f = append(f, key.Filter{Focus: c, Name: n, Required: key.ModShift})
		}
		f = append(f, key.Filter{Focus: c, Name: key.NameSpace, Required: key.ModShortcut},
			key.Filter{Focus: c, Name: "A", Required: key.ModShortcut})
	}
	return f
}

// leisteMarkierTaste behandelt die Tasten der Markierung an der Vorschau i der Leiste (Stelle beim letzten Zeichnen);
// liefert, ob die Taste verbraucht ist. Esc behandelt sie nicht: Es geht wie von der Leinwand über tasten an
// leisteMarkierungEsc (F-890), ein Weg für beide Fälle. Wie der Klick gilt die Taste der gezeigten Aufnahme gezeigt[i],
// nicht Eintraege()[i]: Liest die Leiste zwischen Zeichnen und Taste neu, stünde dort die Nachbarin (F-888).
func (e *editor) leisteMarkierTaste(gtx layout.Context, i int, t key.Event) bool {
	k := &e.unten
	m, es := k.l.Markierung(), k.l.Eintraege()
	if i < 0 || i >= len(k.gezeigt) || k.gezeigt[i] == "" {
		return false
	}
	rel := k.gezeigt[i]
	i = slices.IndexFunc(es, func(x library.Eintrag) bool { return x.Pfad == rel }) // Stelle im jetzigen Stand
	switch {
	case t.Name == key.NameSpace && t.Modifiers == key.ModShortcut:
		if i >= 0 {
			m.Umschalte(rel)
		}
	case t.Name == "A" && t.Modifiers == key.ModShortcut:
		m.Alle(es)
	case t.Modifiers == key.ModShift:
		s, ok := pfeilSchritt(t.Name)
		if !ok || i < 0 {
			return ok // verschwunden: die Taste wirkt nicht, geht aber auch nicht an die Leinwand
		}
		ziel := k.l.Bewege(i, s)
		if ziel < 0 {
			return true
		}
		if m.Anker() == "" {
			m.SetzeAnker(rel)
		}
		m.Bereich(es, es[ziel].Pfad, false)
		if ziel != i {
			k.l.SetzeFokus(ziel)
			k.l.Zeige(ziel, k.masse)
			k.fokusSoll = ziel
		}
	default:
		return false
	}
	gtx.Execute(op.InvalidateCmd{})
	return true
}

// leisteMarkierungEsc hebt beim Esc eine Markierung der Leiste auf, gleich wo der Tastaturfokus liegt (Vorschau,
// Leinwand, rechte Leiste), und liefert, ob es das tat (ADR-0036 Zusatz Karte #361 Punkt 3, F-890): Esc in Stufen, erst die Markierung, dann wie
// bisher. Ein laufender Bereichsmodus, Zug, eine Texteingabe oder Pipette gehen vor, sie beendet Esc wie bisher zuerst.
func (e *editor) leisteMarkierungEsc() bool {
	k := &e.unten
	// Nur eine sichtbare Markierung verbraucht Esc (Verify N-2): Bei ausgeblendeter Leiste wirkt Esc auf die Leinwand.
	if k.l == nil || k.rahmen.Empty() || k.l.Markierung().Anzahl() == 0 ||
		e.s.TextBereichLaeuft() || e.s.Zieht() || e.s.TextLaeuft() || e.s.PipetteLaeuft() {
		return false
	}
	k.l.Markierung().Leere()
	return true
}

// pfeilSchritt ist der Schritt der Leiste zu einer Taste (links, rechts, Pos1, Ende).
func pfeilSchritt(n key.Name) (libview.Schritt, bool) {
	switch n {
	case key.NameLeftArrow:
		return libview.SchrittLinks, true
	case key.NameRightArrow:
		return libview.SchrittRechts, true
	case key.NameHome:
		return libview.SchrittAnfang, true
	case key.NameEnd:
		return libview.SchrittEnde, true
	}
	return 0, false
}

// mehrereMarkiert liefert die markierten Aufnahmen der Liste, wenn rel darunter ist und es mehr als eine sind; sonst nil
// (Entf, Kontextmenü und Umschalt+F10 gelten dann nur rel, Punkt 5).
func mehrereMarkiert(m *libview.Markierung, es []library.Eintrag, rel string) []string {
	if !m.Enthaelt(rel) || m.Anzahl() < 2 {
		return nil
	}
	return m.Pfade(es)
}

// --- Papierkorb mehrerer ---

// papierkorbMarkiert legt die markierten Aufnahmen der Leiste (inLeiste) bzw. der Ansicht in den Papierkorb (Punkt 7).
func (e *editor) papierkorbMarkiert(inLeiste bool) {
	es, m := e.bibl.r.Eintraege(), e.bibl.r.Markierung()
	if inLeiste {
		es, m = e.unten.l.Eintraege(), e.unten.l.Markierung()
	}
	rels := m.Pfade(es)
	if len(rels) == 0 {
		return
	}
	offen := ""
	for _, rel := range rels {
		if e.istOffen(e.absVon(rel)) {
			offen = rel
		}
	}
	var nachbarn []string
	if offen != "" {
		nachbarn = e.unmarkierteNachbarn(es, m, offen)
		if len(nachbarn) == 0 && !inLeiste && e.bibGefiltert() && e.unten.l != nil {
			// Die gefilterte Ansicht zeigt keine andere: die der Leiste, ohne die markierten (wie bibNachbarn, F-827).
			nachbarn = e.unmarkierteNachbarn(e.unten.l.Eintraege(), m, offen)
		}
	}
	e.papierkorbMehrere(rels, offen, nachbarn, inLeiste)
}

// nochMarkiert sind die Aufnahmen aus rels, die in der Leiste (inLeiste) bzw. der vollen Ansicht jetzt noch gezeigt und
// markiert sind, in der Reihenfolge von rels (F-887).
func (e *editor) nochMarkiert(rels []string, inLeiste bool) []string {
	var m *libview.Markierung
	var es []library.Eintrag
	switch {
	case inLeiste && e.unten.l != nil:
		m, es = e.unten.l.Markierung(), e.unten.l.Eintraege()
	case !inLeiste && e.bibl.r != nil:
		m, es = e.bibl.r.Markierung(), e.bibl.r.Eintraege()
	default:
		return nil
	}
	jetzt := m.Pfade(es)
	var aus []string
	for _, rel := range rels {
		if slices.Contains(jetzt, rel) {
			aus = append(aus, rel)
		}
	}
	return aus
}

// absVon ist der absolute Pfad der Aufnahme rel.
func (e *editor) absVon(rel string) string {
	return filepath.Join(e.bib.Bibliothek.Orte.Ablage, filepath.FromSlash(rel))
}

// unmarkierteNachbarn sind die absoluten Pfade der Aufnahmen in es, zu denen der Editor wechseln kann, bevor die offene
// Aufnahme offen mit den markierten in den Papierkorb geht: nach Abstand wie nachbarStellen, ohne markierte und ohne die
// offene, höchstens nachbarGrenze. Anders als nachbarStellen zählt die Grenze nur Treffer: Liegen viele markierte um die
// offene, sucht sie weiter.
func (e *editor) unmarkierteNachbarn(es []library.Eintrag, m *libview.Markierung, offen string) []string {
	i := -1
	for j, eintrag := range es {
		if eintrag.Pfad == offen || e.istOffen(e.absVon(eintrag.Pfad)) {
			i = j
			break
		}
	}
	if i < 0 {
		return nil
	}
	var aus []string
	nimm := func(j int) {
		if j < 0 || j >= len(es) || len(aus) >= nachbarGrenze || m.Enthaelt(es[j].Pfad) {
			return
		}
		if abs := e.absVon(es[j].Pfad); !e.istOffen(abs) {
			aus = append(aus, abs)
		}
	}
	for d := 1; len(aus) < nachbarGrenze && (i+d < len(es) || i-d >= 0); d++ {
		nimm(i + d)
		nimm(i - d)
	}
	return aus
}

// papierkorbMehrere legt die Aufnahmen rels (relativ zur Ablage) in den Papierkorb. offen ist die offene darunter (leer:
// keine), nachbarn die Aufnahmen, zu denen der Editor vorher wechseln kann. Eine einzige geht den bisherigen Weg
// (papierkorbAnfrage). Mit der offenen wechselt der Editor zuerst (wie Karte #347) und nimmt die übrigen nach der
// Übernahme mit, soweit sie dann in der Liste (inLeiste: Leiste, sonst volle Ansicht) noch gezeigt und markiert sind
// (F-887); ohne Nachbarin bleibt die offene, die übrigen gehen.
func (e *editor) papierkorbMehrere(rels []string, offen string, nachbarn []string, inLeiste bool) {
	if len(rels) == 1 {
		e.papierkorbAnfrage(rels[0], offen != "", nachbarn)
		return
	}
	var rest []string
	for _, rel := range rels {
		if rel != offen {
			rest = append(rest, rel)
		}
	}
	if offen != "" && len(nachbarn) > 0 {
		if e.wechselAnfrage(nachbarn[0]) {
			e.wechsel.papierkorb, e.wechsel.weitere, e.wechsel.mitnehmen, e.wechsel.ausLeiste = true, nachbarn[1:], rest, inLeiste
			e.bibl.loeschRuheBis = leisteUhr().Add(loeschRuhe)
		}
		return
	}
	if grund := e.papierkorbGesperrt(); grund != "" {
		e.meldeNebenbei(grund, false)
		return
	}
	e.papierkorbArbeit(rest, offen != "")
}

// papierkorbGesperrt sagt, warum der Papierkorb gerade nicht geht (leer: er geht), mit den Texten von
// papierkorbAnfrage.
func (e *editor) papierkorbGesperrt() string {
	switch {
	case e.s.SpeichernLaeuft():
		return i18n.T("Saving is still running. Please wait a moment, then try again.")
	case e.wechsel.laedt:
		return i18n.T("Another capture is still loading. Please wait a moment, then try again.")
	case e.bibl.arbeitLaeuft:
		return i18n.T("The library is still working. Please wait a moment, then try again.")
	}
	return ""
}

// papierkorbArbeit verschiebt rels im Hintergrund nacheinander in den Papierkorb (library.InPapierkorb, je Aufnahme
// unter der Index-Sperre). offenBleibt: Die offene Aufnahme war markiert und bleibt, die Meldung sagt es. Ohne rels
// tut sie nichts: Beide Aufrufer geben mindestens eine (papierkorbMehrere mit zwei oder mehr markierten ohne die offene,
// wechselAbholen nur mit mitnehmen), eine eigene Meldung für „nur die offene“ gibt es daher nicht (F-893).
func (e *editor) papierkorbArbeit(rels []string, offenBleibt bool) {
	if len(rels) == 0 {
		return
	}
	e.bibl.loeschRuheBis = leisteUhr().Add(loeschRuhe)
	for _, rel := range rels {
		e.verlaeufe.vergiss(e.absVon(rel)) // F-838 je Aufnahme
	}
	b, jetzt := e.bib.Bibliothek, leisteUhr()
	rels = append([]string(nil), rels...)
	e.bibArbeitenMit(bibArbeit{papierkorb: true, rels: rels, offenBleibt: offenBleibt}, func(a *bibArbeit) {
		for _, rel := range rels {
			_, err := b.InPapierkorb(rel, jetzt)
			switch {
			case err == nil:
				a.anzahl++
			case errors.Is(err, library.ErrIndex): // verschoben, nur der Index ist nicht nachgeführt (F-678)
				a.anzahl++
				a.index = true
			default:
				if a.err == nil {
					a.err = err
				}
			}
		}
	})
}

// papierkorbMehrereFertig meldet das Ende einer Arbeit mit mehreren Aufnahmen (bibliothekArbeitFertig).
func (e *editor) papierkorbMehrereFertig(r bibArbeit) {
	n := len(r.rels)
	text, fehler := fmt.Sprintf(i18n.N("Moved %d capture to the trash. It can be restored from there.",
		"Moved %d captures to the trash. They can be restored from there.", r.anzahl), r.anzahl), false
	if r.anzahl < n {
		text = fmt.Sprintf(i18n.T("Moved %d of %d captures to the trash. Could not move the others: %v"), r.anzahl, n,
			fehlerGrund(r.err))
		fehler = true
	}
	if r.offenBleibt {
		text += " " + i18n.T("The open capture stays: there is no unselected capture to switch to.")
	}
	e.melde(text, fehler)
	if r.index || r.anzahl < n {
		e.abgleichAnfordern() // Index nachführen bzw. fehlende Dateien austragen (F-672, F-678)
	}
	if e.unten.v != nil {
		for _, rel := range r.rels {
			e.unten.v.Vergiss(rel)
		}
	}
}

// nichtWeggeworfen ist der Zusatz einer Meldung, wenn der Wechsel vor dem Papierkorb scheitert: eine oder mehrere.
func nichtWeggeworfen(mitnehmen []string) string {
	if len(mitnehmen) > 0 {
		return i18n.T("The selected captures were not moved to the trash.")
	}
	return i18n.T("The capture was not moved to the trash.")
}

// --- Darstellung ---

// zeichneMarkiert tönt die Vorschau r orange (ohne Rand, der grüne Rahmen der offenen bleibt sichtbar) und setzt unten links ein angehaktes Kästchen (Punkt 4); oben links steht
// der Punkt „ungesichert“, oben rechts das ✕. Rückgabe ist die Lage des Kästchens.
func zeichneMarkiert(gtx layout.Context, r image.Rectangle) image.Rectangle {
	t := farbeAkzent
	t.A = 0x50
	paint.FillShape(gtx.Ops, t, clip.Rect(r).Op()) // kein eigener Rand: der grüne der offenen und der Fokusring bleiben
	s, rand := gtx.Dp(16), gtx.Dp(5)
	k := image.Rect(r.Min.X+rand, r.Max.Y-rand-s, r.Min.X+rand+s, r.Max.Y-rand)
	paint.FillShape(gtx.Ops, farbeGrund, clip.Rect(k.Inset(-gtx.Dp(2))).Op())
	zeichneHakenFeld(gtx, k, farbeAkzent, farbeGrund)
	return k
}

// zeichneHakenFeld füllt r mit grund und setzt einen Haken in der Farbe haken darauf.
func zeichneHakenFeld(gtx layout.Context, r image.Rectangle, grund, haken color.NRGBA) {
	paint.FillShape(gtx.Ops, grund, clip.Rect(r).Op())
	w, h := float32(r.Dx()), float32(r.Dy())
	o := f32.Pt(float32(r.Min.X), float32(r.Min.Y))
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(o.Add(f32.Pt(w*0.22, h*0.52)))
	p.LineTo(o.Add(f32.Pt(w*0.42, h*0.72)))
	p.LineTo(o.Add(f32.Pt(w*0.78, h*0.30)))
	paint.FillShape(gtx.Ops, haken, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(2))}.Op())
}

// markHinweis ist der Text der Statuszeile zu einer markierten Vorschau. Entf nennt er nur, wenn die Vorschau den
// Tastaturfokus hat (mitEntf): Über dem bloßen Zeiger wirkt Entf nicht auf sie (F-891).
func markHinweis(n int, mitEntf bool) string {
	if !mitEntf {
		return fmt.Sprintf(i18n.N("%d capture selected — Esc clears the selection",
			"%d captures selected — Esc clears the selection", n), n)
	}
	return fmt.Sprintf(i18n.N("%d capture selected — Del moves it to the trash, Esc clears the selection",
		"%d captures selected — Del moves them to the trash, Esc clears the selection", n), n)
}
