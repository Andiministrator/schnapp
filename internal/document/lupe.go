// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"image"
	"math"
)

// Lupe ist ein vergrößerter Bildausschnitt als eigenes Objekt (ADR-0033): Quelle ist der Ausschnitt im Original, Ziel
// das Rechteck, in dem er vergrößert erscheint, beide in Pixeln des Originals. Die Vergrößerung folgt aus den beiden
// Rechtecken (Ziel/Quelle je Achse). Gezeichnet werden die Verbindungslinie, der Rahmen um die Quelle, das Bild im Ziel
// und der Rahmen um das Ziel, alles in Farbe und Breite von Strich. Der Inhalt ist die vergrößerte Ansicht unter der
// Quelle (Pixel des Originals mit allen Schwärzungen, nur soweit die Ansicht sie zeigt, LupenQuelle); die Lupe hält
// selbst keine Pixel.
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

// --- Inhalt der Lupe (ADR-0033 Punkt 2 und 3, Nachtrag 2 Punkt 1) ---

// LupenQuelle ordnet den Pixeln im Ziel einer Lupe die Pixel des Originals zu, die sie zeigen: die eine Rechnung für
// Renderer, Editor (beide über render) und SVG-Export. Die Lupe vergrößert die Ansicht, nicht das Original: Ihre Quelle
// wird über ZuAnsicht in die Ansicht abgebildet (QA, ohne Klemmung am Zuschnitt: weggestauchte Streifen fallen heraus,
// was außerhalb des Zuschnitts liegt, behält seine Breite). Das Zielpixel mit dem Mittelpunkt z (Original) nimmt den
// Ansichtspunkt QA.Min + (z − Ziel.Min) · QA.Größe / Ziel.Größe, abgerundet auf sein Pixel (nächster Nachbar), und
// dessen Pixel im Original. Liegt es außerhalb der Ansicht (des Zuschnitts) oder im durchsichtigen Spalt einer
// Stauch-Naht, bleibt das Zielpixel leer: gezeigt wird nur, was die Ansicht zeigt. Die Pixel liest sie bei jedem Aufruf
// neu aus dem Original (mit allen Schwärzungen); sie gilt nur, solange sich Zuschnitt, Streifen und Kante nicht ändern.
type LupenQuelle struct {
	d     *Dokument
	l     Lupe
	qa    Rechteck
	spalt []spaltNaht
}

// spaltNaht ist eine Naht, deren Spalt die Quelle in der Ansicht erreichen kann, mit der Mittellinie ihrer Achse.
type spaltNaht struct {
	n     Naht
	linie *Kantenlinie
}

// LupenQuelle liefert die Zuordnung für l im jetzigen Stand der Ansicht.
func (d *Dokument) LupenQuelle(l Lupe) *LupenQuelle {
	q := &LupenQuelle{d: d, l: l, qa: R(d.ZuAnsicht(l.Quelle.Min), d.ZuAnsicht(l.Quelle.Max))}
	f, g := d.KantenFaktor(), d.Groesse()
	reich := (KantenAuslenkung+KantenSpaltHalb)*float64(f) + 1 // so weit reicht der Spalt höchstens über die Naht
	var linien [2]*Kantenlinie
	for _, n := range d.Naehte() {
		lo, hi, achse, laenge := q.qa.Min.X, q.qa.Max.X, 1, g.Y // senkrechte Naht: Linie entlang y (wie render.kanten)
		if n.Zeilen {
			lo, hi, achse, laenge = q.qa.Min.Y, q.qa.Max.Y, 0, g.X
		}
		if float64(n.Lage)+reich <= lo || float64(n.Lage)-reich >= hi {
			continue
		}
		if linien[achse] == nil {
			linien[achse] = NeueKantenlinie(d.kante, laenge, f)
		}
		q.spalt = append(q.spalt, spaltNaht{n, linien[achse]})
	}
	return q
}

// InAnsicht ist die Quelle in der Ansicht (QA). Liegt sie in einer Achse ganz in einem Streifen, ist sie dort 0 breit.
func (q *LupenQuelle) InAnsicht() Rechteck { return q.qa }

// lupenAchse ist die Ansichtsspalte bzw. -zeile für den Zielmittelpunkt z: floor(amin + (z − zmin) · (amax − amin) /
// (zmax − zmin)); ok = false, wenn z nicht im Ziel [zmin, zmax) liegt oder die Quelle in der Ansicht keine Ausdehnung hat.
func lupenAchse(z, zmin, zmax, amin, amax float64) (int, bool) {
	if !(z >= zmin && z < zmax) || !(amax > amin) {
		return 0, false
	}
	return int(math.Floor(amin + (z-zmin)*(amax-amin)/(zmax-zmin))), true
}

// Spalte liefert die Ansichtsspalte, die das Zielpixel mit dem Mittelpunkt x (Original) zeigt (siehe LupenQuelle).
func (q *LupenQuelle) Spalte(x float64) (int, bool) {
	return lupenAchse(x, q.l.Ziel.Min.X, q.l.Ziel.Max.X, q.qa.Min.X, q.qa.Max.X)
}

// Zeile ist Spalte für die Zeile.
func (q *LupenQuelle) Zeile(y float64) (int, bool) {
	return lupenAchse(y, q.l.Ziel.Min.Y, q.l.Ziel.Max.Y, q.qa.Min.Y, q.qa.Max.Y)
}

// pixel liefert die Spalte bzw. Zeile des Originals, die die Ansicht an v zeigt; ok = false außerhalb der Ansicht.
func (m abbildung) pixel(v int) (int, bool) {
	if v < 0 || v >= m.groesse() {
		return 0, false
	}
	return int(m.zuOriginal(float64(v))), true // ganzzahlig exakt; eine Schnittkante gehört zum Bereich danach
}

// ImSpalt meldet, ob das Ansichtspixel (vx, vy) im durchsichtigen Spalt einer Stauch-Naht liegt: sein Mittelpunkt im
// Band Spaltband um die Mittellinie, das der Renderer radiert.
func (q *LupenQuelle) ImSpalt(vx, vy int) bool {
	von, bis := Spaltband()
	for _, s := range q.spalt {
		t, quer := float64(vy)+0.5, float64(vx)+0.5
		if s.n.Zeilen {
			t, quer = quer, t
		}
		if a, b := s.linie.quer(s.n, t, von, bis); quer >= a && quer < b {
			return true
		}
	}
	return false
}

// ansichtspixel liefert die vier Bytes (NRGBA, nur lesen) des Originals, die das Ansichtspixel (vx, vy) zeigt, oder nil
// (außerhalb der Ansicht oder im Spalt).
func (q *LupenQuelle) ansichtspixel(vx, vy int) []byte {
	ox, okx := q.d.abbX.pixel(vx)
	oy, oky := q.d.abbY.pixel(vy)
	if !okx || !oky || q.ImSpalt(vx, vy) {
		return nil
	}
	o := q.d.original.PixOffset(ox, oy)
	return q.d.original.Pix[o : o+4 : o+4]
}

// Abtasten ruft f für jedes Pixel (i, j) eines Rasters von n Spalten und m Zeilen, deren Mittelpunkte im Original bei
// spalte(i) bzw. zeile(j) liegen, das ein Pixel des Originals zeigt, mit dessen vier Bytes (NRGBA, nur lesen); Pixel,
// die nichts zeigen, übergeht es. nachZeile (darf nil sein) folgt jeder Zeile (Puls für lange Schleifen).
func (q *LupenQuelle) Abtasten(n, m int, spalte, zeile func(int) float64, f func(i, j int, px []byte), nachZeile func(j int)) {
	sx := make([]int, n)
	for i := range sx {
		sx[i] = -1 << 62
		if v, ok := q.Spalte(spalte(i)); ok {
			sx[i] = v
		}
	}
	for j := 0; j < m; j++ {
		if vy, ok := q.Zeile(zeile(j)); ok {
			for i, vx := range sx {
				if vx == -1<<62 {
					continue
				}
				if px := q.ansichtspixel(vx, vy); px != nil {
					f(i, j, px)
				}
			}
		}
		if nachZeile != nil {
			nachZeile(j)
		}
	}
}

// AnsichtsAusschnitt liefert die Ansichtspixel unter der Quelle (die ganzen Pixel unter QA, abgerundet bzw.
// aufgerundet, auf die Ansicht begrenzt), in Ansichtskoordinaten (die Grenzen des Ergebnisses); Pixel im Spalt einer
// Naht sind durchsichtig. Mit der viewBox QA über dem Ziel zeigt es jedes Zielpixel wie Abtasten (SVG-Export, ADR-0033
// Nachtrag 2 Punkt 3). Leer, wenn die Quelle nicht in die Ansicht reicht.
func (q *LupenQuelle) AnsichtsAusschnitt() *image.NRGBA {
	r := image.Rect(int(math.Floor(q.qa.Min.X)), int(math.Floor(q.qa.Min.Y)), int(math.Ceil(q.qa.Max.X)),
		int(math.Ceil(q.qa.Max.Y))).Intersect(image.Rectangle{Max: q.d.Groesse()})
	aus := image.NewNRGBA(r)
	for vy := r.Min.Y; vy < r.Max.Y; vy++ {
		for vx := r.Min.X; vx < r.Max.X; vx++ {
			if px := q.ansichtspixel(vx, vy); px != nil {
				copy(aus.Pix[aus.PixOffset(vx, vy):], px)
			}
		}
	}
	return aus
}

// Zielbild liefert das fertige Bild im Ziel auf dem Pixelraster des Originals in r (Grenzen des Ergebnisses): jedes
// Pixel mit Mittelpunkt im Ziel wie Abtasten, alle anderen durchsichtig. Der SVG-Export bettet es statt des
// Ansichtsausschnitts ein, wenn es weniger Pixel hat (Vergrößerung kleiner 1).
func (q *LupenQuelle) Zielbild(r image.Rectangle) *image.NRGBA {
	aus := image.NewNRGBA(r)
	q.Abtasten(r.Dx(), r.Dy(),
		func(i int) float64 { return float64(r.Min.X+i) + 0.5 },
		func(j int) float64 { return float64(r.Min.Y+j) + 0.5 },
		func(i, j int, px []byte) { copy(aus.Pix[aus.PixOffset(r.Min.X+i, r.Min.Y+j):], px) }, nil)
	return aus
}
