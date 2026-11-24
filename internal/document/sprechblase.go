// SPDX-License-Identifier: GPL-3.0-or-later

package document

import "math"

// Sprechblase: ein Text mit Spitze (ADR-0021 Nachtrag 13). Keine eigene Objektart — `Text.Spitze` zeigt auf die Stelle,
// auf die die Blase deutet (Original). Alle Ausgänge (Renderer, SVG-Export, Editor, Treffer, Einrasten) rechnen mit den
// Funktionen dieser Datei, nie mit eigenen Formeln.

// SpitzeAbstand ist der kleinste Abstand der Spitze vom Schild in Schriftgraden; näher (oder im Schild) entfällt das
// Dreieck (ADR-0021 Nachtrag 13 Punkt 2). Ebenso der Mindestabstand der Basis von jeder Ecke.
const SpitzeAbstand = 0.25

// SpitzeBasis ist die Breite der Dreiecksbasis in Schriftgraden (höchstens die halbe Seitenlänge ohne Eckabstände).
const SpitzeBasis = 1.0

// Spitzendreieck liefert das Dreieck der Blase: zwei Basispunkte auf der Schildseite, auf die die Spitze zeigt, in
// Umlaufrichtung (im Uhrzeigersinn, y nach unten), und die Spitze selbst. ok ist false, wenn es kein Dreieck gibt:
// keine Spitze, Spitze im Schild oder näher als SpitzeAbstand Schriftgrade am Schild, oder die Seite ist zu kurz
// (höchstens zweimal der Eckabstand).
//
// Die Basis liegt auf der Seite, die der Strahl von der Schildmitte zur Spitze schneidet; ihre Mitte ist dieser
// Schnittpunkt, so verschoben, dass die Basis ganz auf der Seite liegt und von jeder Ecke mindestens SpitzeAbstand
// Schriftgrade Abstand hält. Breite: SpitzeBasis Schriftgrade, höchstens die Hälfte der Seitenlänge abzüglich der
// beiden Eckabstände (Auslegung der Wendung „halbe Länge abzüglich der Eckabstände“, ADR-0021 Nachtrag 13).
func (t Text) Spitzendreieck() (basis1, spitze, basis2 Punkt, ok bool) {
	if t.Spitze == nil {
		return
	}
	basis1, spitze, basis2, _, ok = spitzendreieck(t.Grenzen(), *t.Spitze, t.Schriftgrad)
	return
}

// spitzendreieck ist die Rechnung von Spitzendreieck auf dem Schild g (testbar ohne Schrift).
// seite ist 0 oben, 1 rechts, 2 unten, 3 links.
func spitzendreieck(g Rechteck, s Punkt, grad float64) (basis1, spitze, basis2 Punkt, seite int, ok bool) {
	e := SpitzeAbstand * grad
	// Abstand der Spitze vom Schild (0 im Schild).
	dx := math.Max(0, math.Max(g.Min.X-s.X, s.X-g.Max.X))
	dy := math.Max(0, math.Max(g.Min.Y-s.Y, s.Y-g.Max.Y))
	if math.Hypot(dx, dy) < e || !(g.Breite() > 0 && g.Hoehe() > 0) {
		return
	}
	m := Pt((g.Min.X+g.Max.X)/2, (g.Min.Y+g.Max.Y)/2)
	d := s.Minus(m)
	hb, hh := g.Breite()/2, g.Hoehe()/2
	// Welche Seite der Strahl trifft: die Achse, deren Schildhälfte zuerst erreicht wird.
	tx, ty := math.Inf(1), math.Inf(1)
	if d.X != 0 {
		tx = hb / math.Abs(d.X)
	}
	if d.Y != 0 {
		ty = hh / math.Abs(d.Y)
	}
	senkrecht := tx < ty // Seite links oder rechts (Länge = Höhe)
	laenge := g.Breite()
	if senkrecht {
		laenge = g.Hoehe()
	}
	b := math.Min(SpitzeBasis*grad, (laenge-2*e)/2)
	if !(b > 0) {
		return
	}
	// Schnittpunkt des Strahls mit der Seite, gemessen entlang der Seite (0 = Anfang der Seite).
	t := math.Min(tx, ty)
	p := m.Plus(d.Mal(t))
	var c float64
	if senkrecht {
		c = p.Y - g.Min.Y
	} else {
		c = p.X - g.Min.X
	}
	c = math.Min(math.Max(c, e+b/2), laenge-e-b/2)
	von, bis := c-b/2, c+b/2
	switch {
	case !senkrecht && d.Y < 0: // oben, von links nach rechts
		return Pt(g.Min.X+von, g.Min.Y), s, Pt(g.Min.X+bis, g.Min.Y), 0, true
	case senkrecht && d.X > 0: // rechts, von oben nach unten
		return Pt(g.Max.X, g.Min.Y+von), s, Pt(g.Max.X, g.Min.Y+bis), 1, true
	case !senkrecht && d.Y > 0: // unten, von rechts nach links
		return Pt(g.Min.X+bis, g.Max.Y), s, Pt(g.Min.X+von, g.Max.Y), 2, true
	case senkrecht && d.X < 0: // links, von unten nach oben
		return Pt(g.Min.X, g.Min.Y+bis), s, Pt(g.Min.X, g.Min.Y+von), 3, true
	}
	return
}

// Sprechblase liefert den Umriss der Blase als ein Vieleck im Uhrzeigersinn (y nach unten): die vier Ecken des Schilds
// (Text.Grenzen), an der Seite der Spitze das Dreieck eingesetzt (ADR-0021 Nachtrag 13 Punkt 2). Ohne Spitze oder ohne
// Dreieck (Spitzendreieck) sind es nur die vier Ecken, beginnend links oben.
func Sprechblase(t Text) []Punkt {
	g := t.Grenzen()
	tl, tr, br, bl := g.Min, Pt(g.Max.X, g.Min.Y), g.Max, Pt(g.Min.X, g.Max.Y)
	if t.Spitze == nil {
		return []Punkt{tl, tr, br, bl}
	}
	a, sp, b, seite, ok := spitzendreieck(g, *t.Spitze, t.Schriftgrad)
	if !ok {
		return []Punkt{tl, tr, br, bl}
	}
	umriss := make([]Punkt, 0, 7)
	for i, ecke := range []Punkt{tl, tr, br, bl} {
		umriss = append(umriss, ecke)
		if i == seite {
			umriss = append(umriss, a, sp, b)
		}
	}
	return umriss
}

// SchenkelBreite ist die Linienbreite der beiden Schenkel, wenn die Blase keine Füllung hat (ADR-0021 Nachtrag 13
// Punkt 3): wie der Unterstrich (StrichDicke), mindestens ein Pixel.
func (t Text) SchenkelBreite() float64 { return math.Max(1, StrichDicke*t.Schriftgrad) }

// Huelle ist das Schild samt Spitze (ADR-0021 Nachtrag 13 Punkt 4): für Auswahlrahmen, Budget und Ebenen-Zwischenbild.
// Die Spitze zählt auch dann, wenn das Dreieck entfällt. Ohne Spitze ist es Grenzen.
func (t Text) Huelle() Rechteck {
	g := t.Grenzen()
	if t.Spitze == nil {
		return g
	}
	return g.Vereint(Rechteck{Min: *t.Spitze, Max: *t.Spitze})
}

// Enthaelt meldet, ob p in der Blase liegt: im Vieleck von Sprechblase, Rand eingeschlossen (Treffer, Punkt 4). Ohne
// Spitze ist das dasselbe wie Grenzen().Enthaelt(p).
func (t Text) Enthaelt(p Punkt) bool {
	if t.Spitze == nil {
		return t.Grenzen().Enthaelt(p)
	}
	return imVieleck(Sprechblase(t), p)
}

// imVieleck meldet, ob p im einfachen Vieleck v liegt oder auf seinem Rand (Strahlverfahren).
func imVieleck(v []Punkt, p Punkt) bool {
	in := false
	for i, a := range v {
		b := v[(i+1)%len(v)]
		// auf der Strecke ab?
		kreuz := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
		if kreuz == 0 && p.X >= math.Min(a.X, b.X) && p.X <= math.Max(a.X, b.X) &&
			p.Y >= math.Min(a.Y, b.Y) && p.Y <= math.Max(a.Y, b.Y) {
			return true
		}
		if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}
