// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"fmt"
	"image/color"
	"math"
)

// Kante ist die Form der Stauch-Kante eines Dokuments (ADR-0023 Nachtrag 6). Der Nullwert ist der Abriss, die
// Voreinstellung für neue Dokumente; Aufnahme und Öffnen eines einfachen Bilds setzen die Form aus der Einstellung
// `[capture] seam`, eine Arbeitsdatei bringt ihre eigene mit (fehlt sie, gilt KanteZacken).
type Kante uint8

// Die Formen in der Reihenfolge, in der der Menüeintrag „Seam style“ sie reihum wählt.
const (
	KanteAbriss Kante = iota // „tear“: unregelmäßige Abrisskante (Voreinstellung)
	KanteWelle               // „wave“: Sinuswelle
	KanteZacken              // „zigzag“: die Zackenkante bis Nachtrag 6
	anzahlKanten
)

// kantenNamen sind die Namen der Formen in der Arbeitsdatei (`s:kante`) und in der Einstellung `[capture] seam`.
var kantenNamen = [anzahlKanten]string{"tear", "wave", "zigzag"}

// Name ist der Name der Form in Arbeitsdatei und Einstellung; leer bei einem ungültigen Wert.
func (k Kante) Name() string {
	if k >= anzahlKanten {
		return ""
	}
	return kantenNamen[k]
}

func (k Kante) String() string { return k.Name() }

// KanteAus liefert die Form zum Namen; ok ist falsch bei einem unbekannten Namen (Groß-/Kleinschreibung zählt).
func KanteAus(name string) (k Kante, ok bool) {
	for i, n := range kantenNamen {
		if n == name {
			return Kante(i), true
		}
	}
	return KanteZacken, false
}

// Naechste ist die Form nach k, reihum (Menüeintrag „Seam style“).
func (k Kante) Naechste() Kante { return (k + 1) % anzahlKanten }

// Maße der Kante quer zur Naht, in Ansichtspixeln bei Faktor 1 (ADR-0023 Nachtrag 6): Die Mittellinie lenkt höchstens
// KantenAuslenkung aus (Abriss 0,8 · 6,15 = 4,92, Welle und Zacken 4), um sie liegt der Spalt mit KantenSpaltHalb je
// Seite, daneben die abgedunkelte Bildkante der Breite KantenDunkel. ZackenBandHalb ist die Summe: So weit ragt die
// Kante höchstens über die Naht. Alle Maße wachsen mit dem Zackenfaktor des Dokuments (Nachträge 1, 2 und 4).
const (
	KantenAuslenkung = 5.0
	KantenSpaltHalb  = 5.0
	KantenDunkel     = 1.0
	ZackenBandHalb   = KantenAuslenkung + KantenSpaltHalb + KantenDunkel
)

// abrissTerme sind Amplitude, Periode und Phase der Sinusanteile der Abrisskante, abrissFaktor ihr gemeinsamer Faktor
// (ADR-0023 Nachtrag 6). Fest, ohne Zufall: Editor, Renderer und SVG-Export zeichnen dieselbe Linie.
var abrissTerme = [...][3]float64{{2.6, 71, 0.3}, {1.6, 29, 1.9}, {1.0, 13, 4.1}, {0.6, 7.3, 2.2}, {0.35, 3.1, 5.7}}

const abrissFaktor = 0.8

// mitte ist die Auslenkung der Mittellinie quer zur Naht an der Stelle t entlang der Naht, bei Faktor 1.
func (k Kante) mitte(t float64) float64 {
	switch k {
	case KanteWelle:
		return 4 * math.Sin(2*math.Pi*t/32)
	case KanteZacken: // ±4, ein Zahn 16 breit, beginnt bei −4
		p := math.Mod(t, 16) / 16
		if p < 0.5 {
			return -4 + 16*p
		}
		return 4 - 16*(p-0.5)
	}
	v := 0.0
	for _, a := range abrissTerme {
		v += a[0] * math.Sin(2*math.Pi*t/a[1]+a[2])
	}
	return abrissFaktor * v
}

// schritt ist der Abstand der Stützpunkte entlang der Naht bei Faktor 1: Abriss 1, Welle 2, Zacken 8 (die Ecken).
func (k Kante) schritt() float64 {
	switch k {
	case KanteWelle:
		return 2
	case KanteZacken:
		return 8
	}
	return 1
}

// Kantenband liefert ein Vieleck (Ansicht) entlang der Naht n über die Länge laenge (Breite bzw. Höhe der Ansicht): die
// Mittellinie der Form k, quer zur Naht um von bis bis verschoben (von < bis; negativ heißt oben bzw. links). Erst die
// Kante bei von vorwärts, dann die bei bis rückwärts. Die Stützpunkte beginnen bei 0 und reichen höchstens einen
// Schritt über laenge hinaus (Renderer und SVG beschneiden auf die Ansicht). Auslenkung, Schritt, von und bis werden
// mit faktor (ZackenFaktor, mindestens 1) multipliziert. Die Querlage ist auf 1/100 Pixel gerundet und als Hundertstel
// gerechnet: Renderer, Editor und SVG-Export bekommen genau dieselben Zahlen, und das SVG schreibt sie kurz.
func Kantenband(k Kante, n Naht, laenge, faktor int, von, bis float64) []Punkt {
	return NeueKantenlinie(k, laenge, faktor).Band(nil, n, von, bis)
}

// Kantenlinie ist die Mittellinie einer Kante, einmal je Form, Länge und Faktor gerechnet (F-418): Lage der Stützpunkte
// entlang der Naht und ihre Auslenkung in ganzen Hundertsteln. Alle Schichten aller Nähte derselben Achse verschieben
// sie nur quer (Band); vorher rechnete Kantenband die Mittellinie (beim Abriss fünf Sinus je Stützpunkt) für jede
// Schicht jeder Naht neu. Die Zahlen sind dieselben wie bisher: Band liefert genau das Vieleck von Kantenband.
type Kantenlinie struct {
	f int
	t []float64 // Lage entlang der Naht je Stützpunkt
	m []int     // Auslenkung quer zur Naht in Hundertsteln, schon mit dem Faktor multipliziert
}

// NeueKantenlinie rechnet die Mittellinie der Form k über die Länge laenge beim Faktor faktor (mindestens 1); Schritt,
// Zahl und Lage der Stützpunkte wie in Kantenband beschrieben.
func NeueKantenlinie(k Kante, laenge, faktor int) *Kantenlinie {
	f := max(faktor, 1)
	s := k.schritt()
	anz := int(math.Ceil(float64(max(laenge, 1)) / (s * float64(f))))
	l := &Kantenlinie{f: f, t: make([]float64, anz+1), m: make([]int, anz+1)}
	for i := 0; i <= anz; i++ {
		tb := float64(i) * s
		l.t[i] = tb * float64(f)
		l.m[i] = hundertstel(k.mitte(tb)) * f
	}
	return l
}

// Band liefert das Vieleck der Linie an der Naht n, quer um von bis bis verschoben (Ansichtspixel bei Faktor 1, mit dem
// Faktor multipliziert), wie Kantenband. Es schreibt in dst, wenn dessen Kapazität reicht (der Renderer nutzt so einen
// Puffer für alle Schichten), sonst in eine neue Liste.
func (l *Kantenlinie) Band(dst []Punkt, n Naht, von, bis float64) []Punkt {
	anz := len(l.t) - 1
	vonH, bisH := hundertstel(von)*l.f, hundertstel(bis)*l.f
	if cap(dst) < 2*(anz+1) {
		dst = make([]Punkt, 2*(anz+1))
	}
	dst = dst[:2*(anz+1)]
	lage := n.Lage * 100
	for i, t := range l.t {
		m := lage + l.m[i]
		dst[i] = quer(n, t, float64(m+vonH)/100)
		dst[2*anz+1-i] = quer(n, t, float64(m+bisH)/100)
	}
	return dst
}

// hundertstel rundet v auf ganze Hundertstel.
func hundertstel(v float64) int { return int(math.Round(v * 100)) }

// quer baut einen Punkt aus der Lage t entlang der Naht und q quer dazu.
func quer(n Naht, t, q float64) Punkt {
	if n.Zeilen {
		return Punkt{t, q}
	}
	return Punkt{q, t}
}

// Schicht ist ein Band der Kante: von Von bis Bis quer zur Mittellinie (Ansichtspixel bei Faktor 1, negativ heißt oben
// bzw. links), gefüllt mit Farbe; A ist die Deckkraft (nicht vormultipliziert).
type Schicht struct {
	Von, Bis float64
	Farbe    color.NRGBA
}

// Farben der Kante: der helle Spalt, der Schatten in ihn hinein und die Abdunklung der Bildkanten (Deckkraft in A).
var (
	KantenSpalt    = color.NRGBA{R: 0xD3, G: 0xD7, B: 0xDC, A: 0xFF}
	kantenSchatten = color.NRGBA{R: 0x20, G: 0x24, B: 0x28}
	kantenDunkel   = color.NRGBA{}
)

func mitDeckkraft(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// kantenSchichten ist die Gestaltung aller Formen (ADR-0023 Nachtrag 6), unten zuerst gezeichnet, abgestimmt am
// Referenzbild Muster 4 (docs/adr/bilder/0023-kante-muster-4.png): Der Spalt (±5) liegt deckend über dem Bild. Den
// weichen Schatten werfen beide Teile als gestufte, halbdurchsichtige Bänder, die alle an der Spaltkante des Teils
// beginnen; wo mehrere übereinanderliegen, wird es dunkler. Das Muster (Schatten 0,55·e^(−s/3) vom oberen,
// 0,25·e^(−s/2) vom unteren Teil, s = Abstand von dessen Kante) ergibt je Pixeltiefe 0 bis 1, 1 bis 2, 2 bis 4 und 4
// bis 7 vom oberen Teil die Deckung 0,47, 0,34, 0,21 und 0,13, vom unteren 0,20, 0,12 und 0,06 (bis Tiefe 4) — die
// Deckkraft der Bänder ist so gewählt, dass ihr Produkt diese Werte trifft. Zuletzt die Abdunklung der Bildkanten
// (1 px, Deckung 0,11, im Muster 0,18·e^(−s) gemittelt über das erste Pixel).
var kantenSchichten = [...]Schicht{
	{-KantenSpaltHalb, KantenSpaltHalb, KantenSpalt},
	// Schatten des oberen bzw. linken Teils, kräftiger (Licht von oben links)
	{-5, 2, mitDeckkraft(kantenSchatten, 0x21)},
	{-5, -1, mitDeckkraft(kantenSchatten, 0x16)},
	{-5, -3, mitDeckkraft(kantenSchatten, 0x2A)},
	{-5, -4, mitDeckkraft(kantenSchatten, 0x33)},
	// Schatten des unteren bzw. rechten Teils
	{1, 5, mitDeckkraft(kantenSchatten, 0x0F)},
	{3, 5, mitDeckkraft(kantenSchatten, 0x10)},
	{4, 5, mitDeckkraft(kantenSchatten, 0x17)},
	// Abdunklung der Bildkanten
	{-KantenSpaltHalb - KantenDunkel, -KantenSpaltHalb, mitDeckkraft(kantenDunkel, 0x1C)},
	{KantenSpaltHalb, KantenSpaltHalb + KantenDunkel, mitDeckkraft(kantenDunkel, 0x1C)},
}

// KantenSchichten liefert die Schichten der Kante in Zeichenreihenfolge (eine Kopie). Renderer, SVG-Export und Editor
// zeichnen je Naht und Schicht ein Kantenband.
func KantenSchichten() []Schicht { return append([]Schicht(nil), kantenSchichten[:]...) }

// Kante liefert die Form der Stauch-Kante des Dokuments.
func (d *Dokument) Kante() Kante { return d.kante }

type kanteAendern struct{ alt, neu Kante }

func (b kanteAendern) ausfuehren(d *Dokument)    { d.kante = b.neu }
func (b kanteAendern) zuruecknehmen(d *Dokument) { d.kante = b.alt }

// pruefeKante meldet eine ungültige Form.
func pruefeKante(k Kante) error {
	if k >= anzahlKanten {
		return fmt.Errorf("document: Kantenform %d unbekannt", k)
	}
	return nil
}

// SetzeKante setzt die Form der Stauch-Kante als einen Rückgängig-Schritt (Menüeintrag „Seam style“, ADR-0023
// Nachtrag 6). Dieselbe Form noch einmal ist kein Schritt.
func (d *Dokument) SetzeKante(k Kante) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	if err := pruefeKante(k); err != nil {
		return err
	}
	if k == d.kante {
		return nil
	}
	d.fuehreAus(kanteAendern{d.kante, k})
	return nil
}

// ErrVerlaufNichtLeer meldet, dass VorgabeKante nach einer Änderung kam.
var ErrVerlaufNichtLeer = errors.New("document: Verlauf nicht leer")

// VorgabeKante setzt die Form ohne Rückgängig-Schritt, für ein Dokument, das eben entstanden ist (Aufnahme, Öffnen
// eines einfachen Bilds, Laden einer Arbeitsdatei): Die Form ist dann keine Änderung des Nutzers, und der Stand bleibt.
// Nach der ersten Änderung (Verlauf nicht leer) oder während eines Zugs ist das ein Fehler: Ein Rückgängig-Schritt, der
// auf die alte Form zurückführt, käme sonst durcheinander.
func (d *Dokument) VorgabeKante(k Kante) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	if err := pruefeKante(k); err != nil {
		return err
	}
	if len(d.verlauf.erledigt) > 0 || len(d.verlauf.zurueckgenommen) > 0 {
		return ErrVerlaufNichtLeer
	}
	d.kante = k
	return nil
}
