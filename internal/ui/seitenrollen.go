// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Rollen der rechten Leiste (ADR-0031 Nachtrag 8, F-598): Reicht die Höhe des Fensters nicht für den ganzen Inhalt der
// Leiste (Werkzeug-Eigenschaften, Farbwähler, Ebenen), rollt die Leiste senkrecht, statt Hex und Deckkraft unter der
// Fußzeile zu verlieren. Der Inhalt wird wie sonst in Inhaltskoordinaten gezeichnet und ganz um `rollen` Pixel nach oben
// verschoben (op.Offset); die Lagen in den Zustandsstrukturen und die Zeigerlagen der Handler sind deshalb
// Inhaltskoordinaten und stimmen mit der Zeichnung überein. Die Rechnung steht hier ohne Gio.

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

// rollEnde ist der größte Versatz: wie weit der Inhalt (er endet bei inhaltEnde, Inhaltskoordinaten) über die
// Unterkante `unten` der Leiste hinausragt. 0, wenn alles passt.
func rollEnde(inhaltEnde, unten int) int {
	return max(0, inhaltEnde-unten)
}

// rollBegrenze hält einen Versatz zwischen 0 und rollMax.
func rollBegrenze(rollen, rollMax int) int {
	return max(0, min(rollen, rollMax))
}

// rollZeige liefert den Versatz, bei dem r (Inhaltskoordinaten) samt Rand in der Fläche zwischen oben und unten liegt,
// möglichst wenig weg vom bisherigen. Ist r höher als die Fläche, gilt seine Oberkante.
func rollZeige(rollen int, r image.Rectangle, oben, unten, rand, rollMax int) int {
	switch {
	case r.Min.Y-rand < oben+rollen:
		rollen = r.Min.Y - rand - oben
	case r.Max.Y+rand > unten+rollen:
		rollen = min(r.Max.Y+rand-unten, r.Min.Y-rand-oben)
	}
	return rollBegrenze(rollen, rollMax)
}

// rollRad rollt um den Radweg dy; der Bruchteil unter einem Pixel bleibt als Rest für das nächste Ereignis (ein
// Touchpad liefert langsam Wege unter 1 px je Ereignis, F-612). Am Anschlag verfällt der Rest, sonst müsste man den
// aufgesparten Weg erst wieder abrollen, bevor die Gegenrichtung wirkt.
func rollRad(rollen int, rest, dy float32, rollMax int) (int, float32) {
	rest += dy
	n := int(rest)
	rest -= float32(n)
	neu := rollBegrenze(rollen+n, rollMax)
	if neu != rollen+n || (neu == 0 && rest < 0) || (neu == rollMax && rest > 0) {
		rest = 0
	}
	return neu, rest
}

// rollDaumen liefert Oberkante und Länge des Daumens einer Rollleiste der Länge bahn für eine sichtbare Höhe sicht über
// einem Inhalt der Höhe gesamt bei Versatz rollen.
func rollDaumen(bahn, sicht, gesamt, rollen int) (int, int) {
	if gesamt <= sicht || gesamt <= 0 {
		return 0, bahn
	}
	laenge := max(bahn/10, bahn*sicht/gesamt)
	return (bahn - laenge) * rollen / max(1, gesamt-sicht), laenge
}

// rollZustand ist der Zustand der Rollens der Leiste.
type rollZustand struct {
	rollen     int             // Versatz in Pixeln, 0 = ganz oben
	rollMax    int             // größter Versatz beim letzten Zeichnen
	rest       float32         // Radweg unter einem Pixel, der noch nicht gerollt ist (Touchpad, F-612)
	flaeche    int             // Ziel der Mausrad-Ereignisse (Zeiger)
	fokus      image.Rectangle // Lage des Elements mit Tastaturfokus beim letzten Zeichnen (Inhaltskoordinaten)
	fokusAlt   image.Rectangle // dasselbe im Bild davor: nur ein Wechsel rollt, damit das Mausrad nichts bekämpft
	sicht      image.Rectangle // sichtbare Fläche der Leiste in Fensterkoordinaten beim letzten Zeichnen
	inhaltEnde int             // Ende des Inhalts in Inhaltskoordinaten beim letzten Zeichnen
	ziel       image.Rectangle // einmal ins Bild rollen (Inhaltskoordinaten), etwa den Farbwähler nach F-817; danach leer
}

// knopfAn meldet den Knopf c bei r an (seiteKnopf) und merkt sich r, wenn er den Tastaturfokus hat (Rollen zum Fokus).
func (e *editor) knopfAn(gtx layout.Context, c *widget.Clickable, r image.Rectangle) {
	if gtx.Focused(c) {
		e.seite.roll.fokus = r
	}
	seiteKnopf(gtx, c, r)
}

// rollAuswerten liest das Mausrad über der Leiste (aus seiteAuswerten).
func (e *editor) rollAuswerten(gtx layout.Context) {
	k := &e.seite.roll
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

// rollAnmelden meldet die Fläche der Leiste für das Mausrad an; vor dem Inhalt, damit dessen Listen (Ebenen, Schriften)
// ihr Mausrad behalten. Der Aufrufer hat r als Ausschnitt schon aufgelegt (zeichneSeitenleiste), und die Anmeldung steht
// bewusst darin und nicht in einem eigenen, wieder abgelegten Ausschnitt: Gio verbindet einen Treffer nur dann mit den
// Bereichen, die später darin gezeichnet werden, wenn der Eintrag im selben Ausschnitt-Bereich bleibt. Sonst deckt jeder
// undurchlässige Bereich darüber (Knopf, Eingabefeld, Farbfläche) das Mausrad ab (Sichtprobe Karte #225).
func (e *editor) rollAnmelden(gtx layout.Context, r image.Rectangle) {
	k := &e.seite.roll
	k.sicht = r
	k.fokus = image.Rectangle{}
	event.Op(gtx.Ops, &k.flaeche)
}

// rollAbschluss rechnet nach dem Zeichnen: größter Versatz, Rollen zum Fokus nach einem Wechsel. Ändert sich der Versatz,
// folgt ein neues Bild.
func (e *editor) rollAbschluss(gtx layout.Context, inhaltEnde int) {
	k := &e.seite.roll
	alt := k.rollen
	k.inhaltEnde = inhaltEnde
	k.rollMax = rollEnde(inhaltEnde, k.sicht.Max.Y)
	k.rollen = rollBegrenze(k.rollen, k.rollMax)
	if !k.fokus.Empty() && k.fokus != k.fokusAlt {
		k.rollen = rollZeige(k.rollen, k.fokus, k.sicht.Min.Y, k.sicht.Max.Y, gtx.Dp(8), k.rollMax)
	}
	k.fokusAlt = k.fokus
	if !k.ziel.Empty() {
		k.rollen = rollZeige(k.rollen, k.ziel, k.sicht.Min.Y, k.sicht.Max.Y, gtx.Dp(8), k.rollMax)
		k.ziel = image.Rectangle{}
	}
	if k.rollen != alt {
		gtx.Execute(op.InvalidateCmd{})
	}
}

// zeichneRollbalken zeichnet rechts in der Leiste einen schmalen eckigen Balken, wenn der Inhalt rollt (Fensterkoordinaten).
func (e *editor) zeichneRollbalken(gtx layout.Context) {
	k := &e.seite.roll
	if k.rollMax <= 0 {
		return
	}
	s := k.sicht
	breite := gtx.Dp(4)
	oben, laenge := rollDaumen(s.Dy(), s.Dy(), s.Dy()+k.rollMax, k.rollen)
	r := image.Rect(s.Max.X-breite-gtx.Dp(2), s.Min.Y+oben, s.Max.X-gtx.Dp(2), s.Min.Y+oben+laenge)
	paint.FillShape(gtx.Ops, farbeNebentext, clip.Rect(r).Op())
}
