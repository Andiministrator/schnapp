// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// Lupe ist ein vergrößerter Bildausschnitt als eigenes Objekt (ADR-0033): Quelle ist der Ausschnitt im Original, Ziel
// das Rechteck, in dem er vergrößert erscheint, beide in Pixeln des Originals. Die Vergrößerung folgt aus den beiden
// Rechtecken (Ziel/Quelle je Achse). Gezeichnet werden die Verbindungslinie, der Rahmen um die Quelle, das Bild im Ziel
// und der Rahmen um das Ziel, alles in Farbe und Breite von Strich. Der Inhalt sind die Pixel des Originals (mit allen
// Schwärzungen), nur soweit die Ansicht sie zeigt; die Lupe hält selbst keine Pixel.
type Lupe struct {
	Ebene
	ID     int
	Quelle Rechteck
	Ziel   Rechteck
	Strich Strich
}

func (l Lupe) Kennung() int { return l.ID }

// Grenzen umfasst beide Rechtecke mit dem halben Strich; die Verbindungslinie liegt dazwischen.
func (l Lupe) Grenzen() Rechteck { return aufgeweitet(l.Quelle.Vereint(l.Ziel), l.Strich.Breite) }

func (l Lupe) mitKennung(id int) Objekt { l.ID = id; return l }
func (l Lupe) mitEbene(e Ebene) Objekt  { l.Ebene = e; return l }

// pruefe: Breite und Höhe beider Rechtecke größer als 0 und höchstens MaxGroesse, Koordinaten endlich und im Bereich
// (ADR-0033 Punkt 1), Strich wie beim Kasten.
func (l Lupe) pruefe() error {
	for _, r := range []Rechteck{l.Quelle, l.Ziel} {
		if !endlich(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y) || !(r.Breite() > 0) || !(r.Hoehe() > 0) ||
			r.Breite() > MaxGroesse || r.Hoehe() > MaxGroesse {
			return fmt.Errorf("document: Lupe mit ungültigen Werten")
		}
	}
	if err := l.Ebene.pruefe(); err != nil {
		return err
	}
	return l.Strich.pruefe()
}

// Vergroesserung ist der Faktor Ziel/Quelle waagerecht und senkrecht (das Werkzeug hält beide gleich).
func (l Lupe) Vergroesserung() (x, y float64) {
	return l.Ziel.Breite() / l.Quelle.Breite(), l.Ziel.Hoehe() / l.Quelle.Hoehe()
}

// QuellSpalte ist die Spalte des Originals, deren Pixel das Zielpixel mit dem Mittelpunkt x (Original) zeigt: der
// nächste Nachbar, floor(Quelle.Min + (x − Ziel.Min) · Quelle.Breite / Ziel.Breite) (ADR-0033 Punkt 3).
func (l Lupe) QuellSpalte(x float64) int {
	return int(math.Floor(l.Quelle.Min.X + (x-l.Ziel.Min.X)*l.Quelle.Breite()/l.Ziel.Breite()))
}

// QuellZeile ist QuellSpalte für die Zeile.
func (l Lupe) QuellZeile(y float64) int {
	return int(math.Floor(l.Quelle.Min.Y + (y-l.Ziel.Min.Y)*l.Quelle.Hoehe()/l.Ziel.Hoehe()))
}

// ueberlappen meldet, ob sich zwei Rechtecke mit positiver Fläche schneiden.
func ueberlappen(a, b Rechteck) bool {
	return a.Min.X < b.Max.X && b.Min.X < a.Max.X && a.Min.Y < b.Max.Y && b.Min.Y < a.Max.Y
}

// Linie ist die Verbindungslinie zwischen den Mittelpunkten von Quelle und Ziel, abgeschnitten außerhalb beider
// Rechtecke (ADR-0033 Punkt 4): von der Kante der Quelle bis zur Kante des Ziels. Überlappen sich die Rechtecke oder
// bleibt nichts übrig (sie berühren sich), gibt es keine Linie.
func (l Lupe) Linie() (a, b Punkt, ok bool) {
	if ueberlappen(l.Quelle, l.Ziel) {
		return Punkt{}, Punkt{}, false
	}
	mq := l.Quelle.Min.Plus(l.Quelle.Max).Mal(0.5)
	mz := l.Ziel.Min.Plus(l.Ziel.Max).Mal(0.5)
	d := mz.Minus(mq)
	// austritt ist der Anteil der Strecke, nach dem sie ein Rechteck mit den halben Maßen hb, hh um ihren Anfang verlässt.
	austritt := func(hb, hh float64) float64 {
		t := math.Inf(1)
		if d.X != 0 {
			t = math.Min(t, hb/math.Abs(d.X))
		}
		if d.Y != 0 {
			t = math.Min(t, hh/math.Abs(d.Y))
		}
		return t
	}
	tq := austritt(l.Quelle.Breite()/2, l.Quelle.Hoehe()/2)
	tz := 1 - austritt(l.Ziel.Breite()/2, l.Ziel.Hoehe()/2)
	if !(tz-tq > 1e-9) { // berühren sich die Rechtecke, bleibt nur Rundungsrest
		return Punkt{}, Punkt{}, false
	}
	return mq.Plus(d.Mal(tq)), mq.Plus(d.Mal(tz)), true
}

// --- Sichtbare Pixel des Originals (Inhalt der Lupe, ADR-0033 Punkt 2) ---

// behaelt meldet, ob die Ansicht die Spalte bzw. Zeile v des Originals zeigt: im Zuschnitt und in keinem Streifen.
func (m abbildung) behaelt(v int) bool {
	if v < m.von || v >= m.bis {
		return false
	}
	k := sort.Search(len(m.weg), func(i int) bool { return m.weg[i][1] > v })
	return k == len(m.weg) || v < m.weg[k][0]
}

// SpalteSichtbar meldet, ob die Ansicht die Spalte x des Originals zeigt (im Zuschnitt, in keinem Spaltenstreifen).
func (d *Dokument) SpalteSichtbar(x int) bool { return d.abbX.behaelt(x) }

// ZeileSichtbar meldet, ob die Ansicht die Zeile y des Originals zeigt (im Zuschnitt, in keinem Zeilenstreifen).
func (d *Dokument) ZeileSichtbar(y int) bool { return d.abbY.behaelt(y) }

// SichtbarerAusschnitt liefert die Pixel des Originals in r, auf das Original begrenzt (das Ergebnis hat genau diese
// Grenzen): Was die Ansicht nicht zeigt (außerhalb des Zuschnitts, in einem Stauch-Streifen), ist durchsichtig
// (0,0,0,0). Der Inhalt einer Lupe im SVG-Export (ADR-0033 Punkte 2 und 6). Leer, wenn r das Original nicht schneidet.
func (d *Dokument) SichtbarerAusschnitt(r image.Rectangle) *image.NRGBA {
	r = r.Intersect(d.original.Rect)
	aus := image.NewNRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if !d.ZeileSichtbar(y) {
			continue
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			if !d.SpalteSichtbar(x) {
				continue
			}
			q, z := d.original.PixOffset(x, y), aus.PixOffset(x, y)
			copy(aus.Pix[z:z+4], d.original.Pix[q:q+4])
		}
	}
	return aus
}
