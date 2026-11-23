// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Lupe (ADR-0033 Punkte 8 und 9): Aufziehen wählt die Quelle, das Ziel erscheint mit Vergrößerung 2 daneben. In der
// Auswahl sind Quelle und Ziel getrennt zu verschieben und an ihren Ecken zu greifen; die rechte Leiste wählt die
// Vergrößerung in Stufen.

// LupenStufen sind die Vergrößerungen der rechten Leiste (− und + gehen von Stufe zu Stufe).
var LupenStufen = []float64{2, 3, 4}

// LupenStart ist die Vergrößerung einer neuen Lupe.
const LupenStart = 2.0

// LupenAbstand ist der Abstand des Ziels zur Quelle in Pixeln der Ansicht, mal document.Dokument.KantenFaktor.
const LupenAbstand = 16.0

// Grenzen der Vergrößerung beim Ziehen an einem Griff des Ziels: nicht kleiner als die Quelle, nicht mehr als 16-fach.
const (
	LupeMinFaktor = 1.0
	LupeMaxFaktor = 16.0
)

// LupenTeil ist der Teil einer Lupe, den ein Druck trifft.
type LupenTeil int

const (
	LupeGanz   LupenTeil = iota // Verbindungslinie: die ganze Lupe
	LupeQuelle                  // Inneres oder Rahmen der Quelle
	LupeZiel                    // Inneres oder Rahmen des Ziels
)

// lupenTeil liefert den Teil von l, den p (Original) mit der Toleranz tol trifft. Das Ziel liegt oben (es wird nach
// der Quelle gezeichnet) und geht bei Überlappung vor.
func lupenTeil(l document.Lupe, p document.Punkt, tol float64) (LupenTeil, bool) {
	halb := l.Strich.Breite/2 + tol
	in := func(r document.Rechteck) bool {
		return document.Rechteck{Min: r.Min.Minus(document.Pt(halb, halb)), Max: r.Max.Plus(document.Pt(halb, halb))}.Enthaelt(p)
	}
	switch {
	case in(l.Ziel):
		return LupeZiel, true
	case in(l.Quelle):
		return LupeQuelle, true
	}
	if a, b, ok := l.Linie(); ok && abstandZurStrecke(p, a, b) <= halb {
		return LupeGanz, true
	}
	return LupeGanz, false
}

// ZielPlatz legt ein Ziel der Größe groesse neben die Quelle q, alles in Ansichtskoordinaten einer Ansicht der Größe
// ansicht (ADR-0033 Punkt 8): rechts im Abstand abstand; ragt es dort aus der Ansicht, links, sonst darunter, sonst
// darüber, sonst doch rechts über den Rand. Quer dazu beginnt das Ziel an derselben Kante wie die Quelle (oben bzw.
// links) und rückt, wenn nötig, in die Ansicht; ein Platz gilt nur, wenn das Ziel auch quer in die Ansicht passt.
func ZielPlatz(q document.Rechteck, groesse document.Punkt, abstand float64, ansicht image.Point) document.Rechteck {
	w, h := groesse.X, groesse.Y
	gx, gy := float64(ansicht.X), float64(ansicht.Y)
	quer := func(start, laenge, platz float64) (float64, bool) {
		if laenge > platz {
			return start, false
		}
		return math.Max(0, math.Min(start, platz-laenge)), true
	}
	at := func(x, y float64) document.Rechteck { return document.R(document.Pt(x, y), document.Pt(x+w, y+h)) }
	if y, ok := quer(q.Min.Y, h, gy); ok {
		if x := q.Max.X + abstand; x+w <= gx {
			return at(x, y)
		}
		if x := q.Min.X - abstand - w; x >= 0 {
			return at(x, y)
		}
	}
	if x, ok := quer(q.Min.X, w, gx); ok {
		if y := q.Max.Y + abstand; y+h <= gy {
			return at(x, y)
		}
		if y := q.Min.Y - abstand - h; y >= 0 {
			return at(x, y)
		}
	}
	return at(q.Max.X+abstand, q.Min.Y)
}

// neueLupe liefert eine Lupe mit der Quelle quelle (Original), der Vergrößerung LupenStart und dem Ziel nach ZielPlatz
// in der Ansicht, im Stil der Sitzung.
func (s *Sitzung) neueLupe(quelle document.Rechteck) document.Lupe {
	qv := document.R(s.D.ZuAnsicht(quelle.Min), s.D.ZuAnsicht(quelle.Max))
	g := document.Pt(quelle.Breite()*LupenStart, quelle.Hoehe()*LupenStart)
	zv := ZielPlatz(qv, g, LupenAbstand*float64(s.D.KantenFaktor()), s.D.Groesse())
	min := s.D.ZuOriginal(zv.Min)
	return document.Lupe{Quelle: quelle, Ziel: document.Rechteck{Min: min, Max: min.Plus(g)}, Strich: s.Stil.strich()}
}

// lupenRechteck liefert das Rechteck des Teils t von l (LupeGanz: die Quelle).
func lupenRechteck(l document.Lupe, t LupenTeil) document.Rechteck {
	if t == LupeZiel {
		return l.Ziel
	}
	return l.Quelle
}

// zieheLupenTeil verschiebt beim Ziehen nur Quelle oder Ziel der Lupe vom Zugbeginn um d (Original): Einrasten wie ein
// Kasten (ADR-0033 Punkt 9), Begrenzen so, dass der Teil im Bild sichtbar bleibt.
func (s *Sitzung) zieheLupenTeil(l document.Lupe, d document.Punkt) {
	r := lupenRechteck(l, s.lupenTeil)
	d, linien := s.rasteRechteck(r, d, l.ID)
	e := s.begrenzeVerschieben(document.Kasten{Bereich: r}, d)
	s.linien = nurWennGleich(linien, d, e)
	verschoben := document.Rechteck{Min: r.Min.Plus(e), Max: r.Max.Plus(e)}
	if s.lupenTeil == LupeZiel {
		l.Ziel = verschoben
	} else {
		l.Quelle = verschoben
	}
	_ = s.D.ZugSetze(l)
}

// --- Vergrößerung (rechte Leiste) ---

// gewaehlteLupe liefert die gewählte Lupe.
func (s *Sitzung) gewaehlteLupe() (document.Lupe, bool) {
	o, ok := s.D.Objekt(s.Auswahl)
	l, ist := o.(document.Lupe)
	return l, ok && ist
}

// Vergroesserung ist die waagerechte Vergrößerung der gewählten Lupe (ok = false ohne gewählte Lupe).
func (s *Sitzung) Vergroesserung() (float64, bool) {
	l, ok := s.gewaehlteLupe()
	if !ok {
		return 0, false
	}
	x, _ := l.Vergroesserung()
	return x, true
}

// VergroesserungStufe geht von der Vergrößerung der gewählten Lupe zur nächsten Stufe aus LupenStufen (dir > 0:
// größer, sonst kleiner); ein Wert zwischen zwei Stufen springt zur nächsten in der Richtung. Liefert die neue
// Vergrößerung und ob es eine gibt.
func (s *Sitzung) VergroesserungStufe(dir int) (float64, bool) {
	jetzt, ok := s.Vergroesserung()
	if !ok {
		return 0, false
	}
	const eps = 1e-9
	if dir > 0 {
		for _, k := range LupenStufen {
			if k > jetzt+eps {
				return k, true
			}
		}
		return jetzt, false
	}
	for i := len(LupenStufen) - 1; i >= 0; i-- {
		if LupenStufen[i] < jetzt-eps {
			return LupenStufen[i], true
		}
	}
	return jetzt, false
}

// MitVergroesserung liefert l mit der Vergrößerung k in beiden Achsen: Das Ziel behält seine linke obere Ecke.
func MitVergroesserung(l document.Lupe, k float64) document.Lupe {
	l.Ziel.Max = l.Ziel.Min.Plus(document.Pt(l.Quelle.Breite()*k, l.Quelle.Hoehe()*k))
	return l
}

// SetzeVergroesserung setzt die Vergrößerung der gewählten Lupe auf k, ein Rückgängig-Schritt (ADR-0033 Punkt 8).
// Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeVergroesserung(k float64) bool {
	if s.Zieht() || s.gesperrt() || !(k > 0) {
		return false
	}
	l, ok := s.gewaehlteLupe()
	if !ok || !l.Bearbeitbar() {
		return false
	}
	neu := MitVergroesserung(l, k)
	if neu == l {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// --- Griffe ---

// lupenGriffe sind die vier Ecken der Quelle und des Ziels.
func lupenGriffe(l document.Lupe) []Griff {
	ecken := func(r document.Rechteck, nw GriffArt) []Griff {
		return []Griff{{nw, r.Min}, {nw + 1, document.Pt(r.Max.X, r.Min.Y)}, {nw + 2, r.Max}, {nw + 3, document.Pt(r.Min.X, r.Max.Y)}}
	}
	return append(ecken(l.Quelle, GriffQuelleNW), ecken(l.Ziel, GriffZielNW)...)
}

// gegenueber liefert die Ecke von r gegenüber der Ecke i (0 NW, 1 NO, 2 SO, 3 SW) und die Richtung von ihr zur Ecke i.
func gegenueber(r document.Rechteck, i int) (fest document.Punkt, sx, sy float64) {
	switch i {
	case 0:
		return r.Max, -1, -1
	case 1:
		return document.Pt(r.Min.X, r.Max.Y), 1, -1
	case 2:
		return r.Min, 1, 1
	}
	return document.Pt(r.Max.X, r.Min.Y), -1, 1
}

// richtung ist das Vorzeichen von v, bei 0 die Vorgabe vor.
func richtung(v, vor float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return vor
}

// zieheLupenGriff liefert l mit dem Griff art an p (Original). Ecken der Quelle ändern die Quelle bei gleicher
// Vergrößerung, das Ziel wächst von seiner linken oberen Ecke aus mit; die Quelle bleibt mindestens ein Pixel groß.
// Ecken des Ziels halten das Seitenverhältnis der Quelle und ändern die Vergrößerung (LupeMinFaktor bis
// LupeMaxFaktor); die gegenüberliegende Ecke bleibt stehen (ADR-0033 Punkt 8).
func zieheLupenGriff(l document.Lupe, art GriffArt, p document.Punkt) document.Lupe {
	switch {
	case art >= GriffQuelleNW && art <= GriffQuelleSW:
		kx, ky := l.Vergroesserung()
		fest, sx, sy := gegenueber(l.Quelle, int(art-GriffQuelleNW))
		d := p.Minus(fest)
		bx := math.Max(1, math.Abs(d.X)) * richtung(d.X, sx)
		by := math.Max(1, math.Abs(d.Y)) * richtung(d.Y, sy)
		l.Quelle = document.R(fest, fest.Plus(document.Pt(bx, by)))
		l.Ziel.Max = l.Ziel.Min.Plus(document.Pt(l.Quelle.Breite()*kx, l.Quelle.Hoehe()*ky))
	case art >= GriffZielNW && art <= GriffZielSW:
		fest, sx, sy := gegenueber(l.Ziel, int(art-GriffZielNW))
		d := p.Minus(fest)
		k := math.Max(math.Abs(d.X)/l.Quelle.Breite(), math.Abs(d.Y)/l.Quelle.Hoehe())
		k = math.Max(LupeMinFaktor, math.Min(LupeMaxFaktor, k))
		l.Ziel = document.R(fest, fest.Plus(document.Pt(richtung(d.X, sx)*l.Quelle.Breite()*k, richtung(d.Y, sy)*l.Quelle.Hoehe()*k)))
	}
	return l
}
