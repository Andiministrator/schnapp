// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sort"
)

// MaxSeite begrenzt Breite und Höhe des Originals, MaxPixel die Fläche: größer als jede Bildschirmaufnahme
// (drei 4K-Monitore haben rund 25 Mio. Pixel), klein genug, dass eine kleine fremde Datei nicht viele GiB Speicher
// anfordert (F-59; 1<<28 Pixel × 4 Byte = 1 GiB, wie die Grenze der Aufnahmeschicht).
const (
	MaxSeite = 1 << 15
	MaxPixel = 1 << 28
)

// Streifen ist ein herausgeschnittener Bereich beim Stauchen: Zeilen [Von, Bis) bei Zeilen = true (das Bild wird
// niedriger), sonst Spalten [Von, Bis) (das Bild wird schmaler). Angaben in Pixeln des Originals.
type Streifen struct {
	Zeilen   bool
	Von, Bis int
}

// Dokument ist eine Aufnahme mit Beschriftungen. Der Nullwert ist nicht nutzbar; Neu legt eins an.
type Dokument struct {
	original   *image.NRGBA
	objekte    []Objekt
	naechsteID int
	zuschnitt  image.Rectangle // im Original; immer gesetzt, ganzes Bild = keine Beschneidung
	streifen   []Streifen
	abbX, abbY abbildung // aus zuschnitt und streifen abgeleitet, nur über setzeAnsicht ändern
	verlauf    verlauf
	zug        *zug
}

// Fehler der Änderungen.
var (
	ErrZugLaeuft   = errors.New("dokument: während eines Zugs nicht möglich")
	ErrKeinZug     = errors.New("dokument: kein Zug aktiv")
	ErrUnbekannt   = errors.New("dokument: Objekt unbekannt")
	ErrNichtsDa    = errors.New("dokument: nichts rückgängig zu machen oder zu wiederholen")
	ErrBildGroesse = errors.New("dokument: Bildgröße ungültig")
)

// Neu legt ein Dokument mit einer Kopie von bild als Original an.
func Neu(bild image.Image) (*Dokument, error) {
	b := bild.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 || b.Dx() > MaxSeite || b.Dy() > MaxSeite || b.Dx()*b.Dy() > MaxPixel {
		return nil, fmt.Errorf("%w: %dx%d", ErrBildGroesse, b.Dx(), b.Dy())
	}
	o := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(o, o.Bounds(), bild, b.Min, draw.Src)
	d := &Dokument{original: o, naechsteID: 1}
	d.setzeAnsicht(o.Bounds(), nil)
	return d, nil
}

// Original liefert eine Kopie des unbeschnittenen Originals (mit allen Schwärzungen).
func (d *Dokument) Original() *image.NRGBA {
	k := image.NewNRGBA(d.original.Rect)
	copy(k.Pix, d.original.Pix)
	return k
}

// Objekte liefert die Objekte in Zeichenreihenfolge (unten zuerst).
func (d *Dokument) Objekte() []Objekt { return append([]Objekt(nil), d.objekte...) }

// Objekt liefert das Objekt mit der Kennung id.
func (d *Dokument) Objekt(id int) (Objekt, bool) {
	if i := d.index(id); i >= 0 {
		return d.objekte[i], true
	}
	return nil, false
}

// Zuschnitt liefert den sichtbaren Ausschnitt im Original.
func (d *Dokument) Zuschnitt() image.Rectangle { return d.zuschnitt }

// Streifen liefert die Stauch-Streifen, sortiert nach Achse und Lage.
func (d *Dokument) Streifen() []Streifen { return append([]Streifen(nil), d.streifen...) }

// NaechsteNummer ist der Wert für eine neue Schritt-Nummer: größter vorhandener Wert + 1, sonst 1.
func (d *Dokument) NaechsteNummer() int {
	m := 0
	for _, o := range d.objekte {
		if n, ok := o.(Nummer); ok {
			m = max(m, n.Wert)
		}
	}
	return m + 1
}

func (d *Dokument) index(id int) int {
	for i, o := range d.objekte {
		if o.Kennung() == id {
			return i
		}
	}
	return -1
}

// --- Ansicht: Zuschnitt und Stauchen ---

// abbildung ist die fertig berechnete Umrechnung einer Achse (F-136): die entfernten Bereiche, auf den Zuschnitt
// begrenzt, und ihre Präfixsummen. Sie wird beim Ändern von Zuschnitt oder Streifen einmal aufgebaut (setzeAnsicht)
// und danach nur gelesen — so bleibt das Dokument auch bei nebenläufigem Lesen frei von Schreibzugriffen, und jede
// Umrechnung kostet O(log n) ohne Allokation.
type abbildung struct {
	von, bis int
	weg      [][2]int // sortiert, überlappungsfrei, nicht leer
	kum      []int    // kum[i] = Summe der Längen von weg[0..i-1]; len(kum) = len(weg)+1
}

func baueAbbildung(z image.Rectangle, st []Streifen, zeilen bool) abbildung {
	m := abbildung{von: z.Min.X, bis: z.Max.X}
	if zeilen {
		m.von, m.bis = z.Min.Y, z.Max.Y
	}
	m.kum = []int{0}
	for _, s := range st {
		if s.Zeilen != zeilen {
			continue
		}
		a, b := max(s.Von, m.von), min(s.Bis, m.bis)
		if a < b {
			m.weg = append(m.weg, [2]int{a, b}) // st ist sortiert, also auch weg
			m.kum = append(m.kum, m.kum[len(m.kum)-1]+b-a)
		}
	}
	return m
}

func (m abbildung) groesse() int {
	if len(m.kum) == 0 {
		return 0 // Nullwert Dokument{}: keine Ansicht (nur Neu liefert ein nutzbares Dokument)
	}
	return m.bis - m.von - m.kum[len(m.weg)]
}

// zuAnsicht: Punkte in einem Streifen fallen auf dessen Kante.
func (m abbildung) zuAnsicht(v float64) float64 {
	// k = Anzahl der Streifen, die v ganz hinter sich hat (Ende <= v)
	k := sort.Search(len(m.weg), func(i int) bool { return float64(m.weg[i][1]) > v })
	r := v - float64(m.von) - float64(m.kum[k])
	if k < len(m.weg) && v > float64(m.weg[k][0]) {
		r -= v - float64(m.weg[k][0])
	}
	return r
}

// zuOriginal: Eine Schnittkante gehört zum Bereich nach dem Streifen.
//
// Festgelegtes Verhalten an der Kante (F-142): Der Vergleich rechnet exakt — v wird gegen die ganzzahlige Kante
// weg[i][0]-kum[i]-von verglichen, nicht erst mit von verrechnet (diese Addition rundete einen Wert ein ulp vor der
// Kante auf die Kante). Ein Ansichtswert genau auf der Kante ergibt das Streifenende (den Bereich danach), jeder
// kleinere — auch ein ulp davor — den Streifenanfang (linksseitiger Grenzwert, nie ein Wert im Streifen). Die
// frühere Rechnung addierte die Längen vorher in Gleitkomma und lieferte knapp vor der Kante das Streifenende; das
// ist bewusst nicht mehr so.
func (m abbildung) zuOriginal(v float64) float64 {
	// Streifen i wirkt, wenn v + von + kum[i] >= weg[i][0]; a_i - kum[i] - von wächst mit i, die Bedingung gilt also
	// für ein Präfix.
	k := sort.Search(len(m.weg), func(i int) bool { return v < float64(m.weg[i][0]-m.kum[i]-m.von) })
	return v + float64(m.von) + float64(m.kum[k])
}

// setzeAnsicht setzt Zuschnitt und Streifen und baut die Abbildungen neu auf. Jede Stelle, die einen der beiden
// ändert (Verlauf, Rückgängig, Wiederholen, Laden über SetzeStreifen), geht hierüber (F-136).
func (d *Dokument) setzeAnsicht(z image.Rectangle, st []Streifen) {
	d.zuschnitt, d.streifen = z, st
	d.abbX, d.abbY = baueAbbildung(z, st, false), baueAbbildung(z, st, true)
}

// achse liefert für die Ansicht die entfernten Bereiche einer Achse, auf den Zuschnitt begrenzt und sortiert.
func (d *Dokument) achse(zeilen bool) (von, bis int, weg [][2]int) {
	m := d.abbX
	if zeilen {
		m = d.abbY
	}
	return m.von, m.bis, m.weg
}

// Groesse ist die Größe der Ansicht in Pixeln.
func (d *Dokument) Groesse() image.Point { return image.Pt(d.abbX.groesse(), d.abbY.groesse()) }

// ZuAnsicht rechnet einen Punkt des Originals in die Ansicht um. Punkte in einem Streifen fallen auf dessen Kante.
func (d *Dokument) ZuAnsicht(p Punkt) Punkt {
	return Punkt{d.abbX.zuAnsicht(p.X), d.abbY.zuAnsicht(p.Y)}
}

// ZuOriginal rechnet einen Punkt der Ansicht ins Original um. Eine Schnittkante gehört zum Bereich nach dem Streifen.
func (d *Dokument) ZuOriginal(p Punkt) Punkt {
	return Punkt{d.abbX.zuOriginal(p.X), d.abbY.zuOriginal(p.Y)}
}

// Ansichtsbild liefert die Pixel der Ansicht: Original, beschnitten und gestaucht, ohne Objekte.
func (d *Dokument) Ansichtsbild() *image.NRGBA {
	behalten := func(zeilen bool) []int {
		von, bis, weg := d.achse(zeilen)
		var res []int
		j := 0
		for v := von; v < bis; v++ {
			for j < len(weg) && v >= weg[j][1] {
				j++
			}
			if j < len(weg) && v >= weg[j][0] {
				continue
			}
			res = append(res, v)
		}
		return res
	}
	spalten, zeilen := behalten(false), behalten(true)
	ziel := image.NewNRGBA(image.Rect(0, 0, len(spalten), len(zeilen)))
	for y, oy := range zeilen {
		for x, ox := range spalten {
			q := d.original.PixOffset(ox, oy)
			z := ziel.PixOffset(x, y)
			copy(ziel.Pix[z:z+4], d.original.Pix[q:q+4])
		}
	}
	return ziel
}

// pruefeAnsicht prüft Zuschnitt und Streifen gegen das Original.
func (d *Dokument) pruefeAnsicht(z image.Rectangle, st []Streifen) error {
	if z.Empty() || !z.In(d.original.Rect) {
		return fmt.Errorf("dokument: Zuschnitt %v außerhalb des Bilds %v", z, d.original.Rect)
	}
	if len(st) > MaxStreifen {
		return fmt.Errorf("dokument: mehr als %d Streifen", MaxStreifen)
	}
	zeilen := 0
	for _, s := range st {
		if s.Zeilen {
			zeilen++
		}
	}
	if (zeilen+1)*(len(st)-zeilen+1) > MaxKacheln {
		return fmt.Errorf("dokument: Streifen ergeben mehr als %d Kacheln", MaxKacheln)
	}
	grenze := map[bool]int{false: d.original.Rect.Dx(), true: d.original.Rect.Dy()}
	for i, s := range st {
		if s.Von < 0 || s.Bis > grenze[s.Zeilen] || s.Von >= s.Bis {
			return fmt.Errorf("dokument: Streifen %v ungültig", s)
		}
		if i > 0 && st[i-1].Zeilen == s.Zeilen && st[i-1].Bis > s.Von {
			return fmt.Errorf("dokument: Streifen %v und %v überlappen", st[i-1], s)
		}
	}
	// Die Ansicht darf nicht leer werden.
	if baueAbbildung(z, st, false).groesse() < 1 || baueAbbildung(z, st, true).groesse() < 1 {
		return fmt.Errorf("dokument: Ansicht wäre leer")
	}
	return nil
}

func sortiert(st []Streifen) []Streifen {
	st = append([]Streifen(nil), st...)
	sort.Slice(st, func(i, j int) bool {
		if st[i].Zeilen != st[j].Zeilen {
			return !st[i].Zeilen
		}
		return st[i].Von < st[j].Von
	})
	return st
}

// --- Schwärzen ---

// Schwaerzung legt fest, wie ein Bereich unkenntlich gemacht wird.
type Schwaerzung struct {
	// Block > 0 verpixelt mit Blöcken dieser Kantenlänge (Mittelwert je Block); sonst wird mit Farbe gefüllt.
	Block int
	Farbe color.NRGBA
}

// MinBlock ist die kleinste Blockgröße beim Verpixeln; kleinere Blöcke lassen Text lesbar.
const MinBlock = 8

// schwaerze überschreibt die Pixel im Bereich r des Originals (r liegt schon im Bild).
func (d *Dokument) schwaerze(r image.Rectangle, s Schwaerzung) {
	o := d.original
	if s.Block <= 0 {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				o.SetNRGBA(x, y, s.Farbe)
			}
		}
		return
	}
	for _, ys := range bloecke(r.Min.Y, r.Max.Y, s.Block) {
		for _, xs := range bloecke(r.Min.X, r.Max.X, s.Block) {
			blk := image.Rect(xs[0], ys[0], xs[1], ys[1])
			var sum [4]int
			for y := blk.Min.Y; y < blk.Max.Y; y++ {
				for x := blk.Min.X; x < blk.Max.X; x++ {
					i := o.PixOffset(x, y)
					for k := 0; k < 4; k++ {
						sum[k] += int(o.Pix[i+k])
					}
				}
			}
			n := blk.Dx() * blk.Dy()
			c := color.NRGBA{uint8(sum[0] / n), uint8(sum[1] / n), uint8(sum[2] / n), uint8(sum[3] / n)}
			for y := blk.Min.Y; y < blk.Max.Y; y++ {
				for x := blk.Min.X; x < blk.Max.X; x++ {
					o.SetNRGBA(x, y, c)
				}
			}
		}
	}
}

// bloecke teilt [von, bis) in Blöcke der Länge block; ein Rest kürzer als MinBlock wird dem letzten Block
// zugeschlagen, damit kein schmaler Streifen nur in einer Richtung gemittelt wird (F-61). Der Bereich ist
// mindestens MinBlock lang (geprüft in Schwaerzen).
func bloecke(von, bis, block int) [][2]int {
	var res [][2]int
	for a := von; a < bis; a += block {
		res = append(res, [2]int{a, min(a+block, bis)})
	}
	if n := len(res); n > 1 && res[n-1][1]-res[n-1][0] < MinBlock {
		res[n-2][1] = res[n-1][1]
		res = res[:n-1]
	}
	return res
}
