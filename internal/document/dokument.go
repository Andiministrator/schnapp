// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/big"
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
	// geschwaerzt zählt die wirksamen Schwärzungen seit Neu bzw. dem Laden (Rückgängig zieht ab); VergissVerlauf
	// lässt ihn stehen (ADR-0022 Punkt 4).
	geschwaerzt int
	// schwaerzStand ist die Kennung der zuletzt wirksamen Schwärzung (0: keine seit Neu bzw. Laden), schwaerzKennungen
	// die zuletzt vergebene Kennung (SchwaerzStand, F-187).
	schwaerzStand, schwaerzKennungen uint64
	// schwaerzVerworfen: Die Grenze des Verlaufs hat seit dem letzten VergissVerlauf eine Schwärzung verworfen
	// (SchwaerzungenImVerlauf ist dann unvollständig, ADR-0029).
	schwaerzVerworfen bool
}

// Fehler der Änderungen.
var (
	ErrZugLaeuft   = errors.New("document: während eines Zugs nicht möglich")
	ErrKeinZug     = errors.New("document: kein Zug aktiv")
	ErrUnbekannt   = errors.New("document: Objekt unbekannt")
	ErrNichtsDa    = errors.New("document: nichts rückgängig zu machen oder zu wiederholen")
	ErrBildGroesse = errors.New("document: Bildgröße ungültig")
)

// Neu legt ein Dokument mit einer Kopie von bild als Original an. Die Kopie schützt davor, dass der Aufrufer die Pixel
// später ändert oder mit einem anderen Dokument teilt.
func Neu(bild image.Image) (*Dokument, error) {
	b := bild.Bounds()
	if err := pruefeGroesse(b); err != nil {
		return nil, err
	}
	o := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(o, o.Bounds(), bild, b.Min, draw.Src)
	return mitOriginal(o), nil
}

// NeuAus legt ein Dokument an, das o als Original übernimmt, ohne es zu kopieren (F-334): für ein Bild, das der
// Aufrufer eben erst erzeugt hat (Decoder, Ausrichten) und danach nicht mehr verwendet oder weitergibt — sonst teilten
// zwei Stellen die Pixel, und eine Schwärzung oder ein Rückgängig wirkte auf beide. Ein o, das nicht genau ein
// geschlossenes Bild ab (0, 0) ist (Teilbild, anderer Zeilenabstand), wird wie bei Neu kopiert.
func NeuAus(o *image.NRGBA) (*Dokument, error) {
	if o == nil {
		return nil, fmt.Errorf("%w: kein Bild", ErrBildGroesse)
	}
	r := o.Rect
	if r.Min != (image.Point{}) || o.Stride != 4*r.Dx() || len(o.Pix) != o.Stride*r.Dy() {
		return Neu(o)
	}
	if err := pruefeGroesse(r); err != nil {
		return nil, err
	}
	return mitOriginal(o), nil
}

// pruefeGroesse prüft die Maße eines neuen Originals gegen MaxSeite und MaxPixel.
func pruefeGroesse(b image.Rectangle) error {
	if b.Dx() < 1 || b.Dy() < 1 || b.Dx() > MaxSeite || b.Dy() > MaxSeite || b.Dx()*b.Dy() > MaxPixel {
		return fmt.Errorf("%w: %dx%d", ErrBildGroesse, b.Dx(), b.Dy())
	}
	return nil
}

func mitOriginal(o *image.NRGBA) *Dokument {
	d := &Dokument{original: o, naechsteID: 1}
	d.setzeAnsicht(o.Bounds(), nil)
	return d
}

// Original liefert eine Kopie des unbeschnittenen Originals (mit allen Schwärzungen).
func (d *Dokument) Original() *image.NRGBA {
	k := image.NewNRGBA(d.original.Rect)
	copy(k.Pix, d.original.Pix)
	return k
}

// OriginalBild liefert das Original ohne Kopie, nur zum Lesen (Kodieren, Ausschnitte): Wer es verändert oder
// festhält, während das Dokument sich ändert, verfälscht den Stand. Für alles, was ändert oder aufbewahrt, gilt Original.
func (d *Dokument) OriginalBild() *image.NRGBA { return d.original }

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
		return fmt.Errorf("document: Zuschnitt %v außerhalb des Bilds %v", z, d.original.Rect)
	}
	if len(st) > MaxStreifen {
		return fmt.Errorf("document: mehr als %d Streifen", MaxStreifen)
	}
	zeilen := 0
	for _, s := range st {
		if s.Zeilen {
			zeilen++
		}
	}
	if (zeilen+1)*(len(st)-zeilen+1) > MaxKacheln {
		return fmt.Errorf("document: Streifen ergeben mehr als %d Kacheln", MaxKacheln)
	}
	grenze := map[bool]int{false: d.original.Rect.Dx(), true: d.original.Rect.Dy()}
	for i, s := range st {
		if s.Von < 0 || s.Bis > grenze[s.Zeilen] || s.Von >= s.Bis {
			return fmt.Errorf("document: Streifen %v ungültig", s)
		}
		if i > 0 && st[i-1].Zeilen == s.Zeilen && st[i-1].Bis > s.Von {
			return fmt.Errorf("document: Streifen %v und %v überlappen", st[i-1], s)
		}
	}
	// Die Ansicht darf nicht leer werden.
	if baueAbbildung(z, st, false).groesse() < 1 || baueAbbildung(z, st, true).groesse() < 1 {
		return fmt.Errorf("document: Ansicht wäre leer")
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
	// Mosaik > 0 verpixelt ohne Bezug zum Inhalt (ADR-0022 Nachtrag 1): Jeder Block bekommt das Mittel des ganzen
	// Bereichs, verschoben um einen Zufallswert in [-Mosaik, Mosaik] (für R, G und B derselbe, damit es nach Mosaik
	// aussieht und nicht nach Konfetti). Das Blockmittel je Block (Mosaik = 0) lässt sich bei mittlerer Schriftgröße
	// zurückrechnen, auch mit Rauschen je Block (Kritik Fable zu Karte #37, F-163).
	Mosaik int
}

// MaxMosaik begrenzt Schwaerzung.Mosaik.
const MaxMosaik = 64

// zufall liefert eine gleichverteilte Zahl in [0, n); Tests ersetzen ihn.
var zufall = func(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // crypto/rand scheitert unter Linux nicht (getrandom); ohne Zufall kein sicheres Mosaik
	}
	return int(v.Int64())
}

// MinBlock ist die kleinste Blockgröße beim Verpixeln; kleinere Blöcke lassen Text lesbar.
const MinBlock = 8

// schwaerze überschreibt die Pixel im Bereich r des Originals (r liegt schon im Bild).
func (d *Dokument) schwaerze(r image.Rectangle, s Schwaerzung) {
	o := d.original
	if s.Block <= 0 {
		fuelle(o, r, s.Farbe)
		return
	}
	var gesamt color.NRGBA
	if s.Mosaik > 0 {
		gesamt = mittel(o, r)
		// Auch das Gesamtmittel trägt etwas Information (Tintenmenge); einmal verschieben (Schritt 2 unten je Block).
		gesamt = verschoben(gesamt, zufall(2*s.Mosaik+1)-s.Mosaik)
	}
	for _, ys := range bloecke(r.Min.Y, r.Max.Y, s.Block) {
		for _, xs := range bloecke(r.Min.X, r.Max.X, s.Block) {
			blk := image.Rect(xs[0], ys[0], xs[1], ys[1])
			if s.Mosaik > 0 {
				c := verschoben(gesamt, zufall(2*s.Mosaik+1)-s.Mosaik)
				fuelle(o, blk, c)
				continue
			}
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
			fuelle(o, blk, color.NRGBA{uint8(sum[0] / n), uint8(sum[1] / n), uint8(sum[2] / n), uint8(sum[3] / n)})
		}
	}
}

// mittel liefert die mittlere Farbe von r in o.
func mittel(o *image.NRGBA, r image.Rectangle) color.NRGBA {
	var sum [4]int
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := o.PixOffset(x, y)
			for k := 0; k < 4; k++ {
				sum[k] += int(o.Pix[i+k])
			}
		}
	}
	n := r.Dx() * r.Dy()
	return color.NRGBA{uint8(sum[0] / n), uint8(sum[1] / n), uint8(sum[2] / n), uint8(sum[3] / n)}
}

// fuelle setzt alle Pixel von r in o auf c.
func fuelle(o *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			o.SetNRGBA(x, y, c)
		}
	}
}

// verschoben addiert v zu R, G und B von c, begrenzt auf 0..255 (Alpha bleibt).
func verschoben(c color.NRGBA, v int) color.NRGBA {
	for _, p := range []*uint8{&c.R, &c.G, &c.B} {
		*p = uint8(max(0, min(255, int(*p)+v)))
	}
	return c
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
