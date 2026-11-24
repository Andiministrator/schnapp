// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Rollen der Tafeln (ADR-0031 Nachtrag 10, F-705, F-709): Die Übersicht der Tastenkürzel (F1) und die Einstellungstafel
// rollen ihren Inhalt senkrecht, wenn er nicht in die Tafel passt (kleines Fenster), statt Zeilen oder Felder
// abzuschneiden. Die Rechnung teilen sie mit der rechten Leiste (`seitenrollen.go`: rollEnde, rollZeige, rollRad,
// rollDaumen); hier stehen die Teile, die beide Tafeln brauchen, soweit ohne Gio möglich.

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// rollTaste liefert den Versatz nach einer Rolltaste (Pfeil auf/ab, Bild auf/ab, Pos1, Ende) und ob die Taste eine
// Rolltaste war. zeile ist die Höhe einer Zeile, seite die sichtbare Höhe; eine Seite rollt um eine Seite minus eine Zeile.
func rollTaste(n key.Name, rollen, rollMax, zeile, seite int) (int, bool) {
	switch n {
	case key.NameUpArrow:
		rollen -= zeile
	case key.NameDownArrow:
		rollen += zeile
	case key.NamePageUp:
		rollen -= max(zeile, seite-zeile)
	case key.NamePageDown:
		rollen += max(zeile, seite-zeile)
	case key.NameHome:
		rollen = 0
	case key.NameEnd:
		rollen = rollMax
	default:
		return rollen, false
	}
	return rollBegrenze(rollen, rollMax), true
}

// radLesen liest das Mausrad über der Fläche k.flaeche (angemeldet mit anmelden).
func (k *rollZustand) radLesen(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &k.flaeche, Kinds: pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			return
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Scroll {
			k.rollen, k.rest = rollRad(k.rollen, k.rest, pe.Scroll.Y, k.rollMax)
		}
	}
}

// anmelden meldet die sichtbare Fläche r für das Mausrad an. Der Aufrufer hat r als Ausschnitt schon aufgelegt und
// zeichnet den Inhalt danach in diesem Ausschnitt (Begründung: rollAnmelden in seitenrollen.go).
func (k *rollZustand) anmelden(r image.Rectangle, gtx layout.Context) {
	k.sicht = r
	k.fokus = image.Rectangle{}
	event.Op(gtx.Ops, &k.flaeche)
}

// abschluss rechnet nach dem Zeichnen: größter Versatz und, nach einem Fokuswechsel, Rollen zum Fokus. Ändert sich der
// Versatz, folgt ein neues Bild.
func (k *rollZustand) abschluss(gtx layout.Context, inhaltEnde, rand int) {
	alt := k.rollen
	k.inhaltEnde = inhaltEnde
	k.rollMax = rollEnde(inhaltEnde, k.sicht.Max.Y)
	k.rollen = rollBegrenze(k.rollen, k.rollMax)
	if !k.fokus.Empty() && k.fokus != k.fokusAlt {
		k.rollen = rollZeige(k.rollen, k.fokus, k.sicht.Min.Y, k.sicht.Max.Y, rand, k.rollMax)
	}
	k.fokusAlt = k.fokus
	if k.rollen != alt {
		gtx.Execute(op.InvalidateCmd{})
	}
}

// balken zeichnet rechts in der Fläche s einen schmalen eckigen Balken, wenn der Inhalt rollt.
func (k *rollZustand) balken(gtx layout.Context, s image.Rectangle) {
	if k.rollMax <= 0 {
		return
	}
	breite := gtx.Dp(4)
	oben, laenge := rollDaumen(s.Dy(), s.Dy(), s.Dy()+k.rollMax, k.rollen)
	r := image.Rect(s.Max.X-breite-gtx.Dp(2), s.Min.Y+oben, s.Max.X-gtx.Dp(2), s.Min.Y+oben+laenge)
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(r).Op())
}
