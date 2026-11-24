// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"slices"
	"sort"
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

// Maße der Kante quer zur Naht, in Ansichtspixeln bei Faktor 1 (ADR-0023 Nachträge 6 und 7): Die Mittellinie lenkt
// höchstens KantenAuslenkung aus (Abriss 0,9 · 6,15 = 5,54, Welle und Zacken 4), um sie liegt der durchsichtige Spalt
// mit KantenSpaltHalb je Seite, daneben der Rand der Bildteile (Fasersaum und Abdunklung) der Breite KantenRand.
// KantenBandHalb ist die Summe: So weit ragt die Kante höchstens über die Naht. Alle Maße wachsen mit dem Kantenfaktor
// des Dokuments (Nachträge 1, 2 und 4).
const (
	KantenAuslenkung = 6.0
	KantenSpaltHalb  = 8.0
	KantenRand       = 2.0
	KantenBandHalb   = KantenAuslenkung + KantenSpaltHalb + KantenRand
	// KantenSichtRest ist der Teil des Bilds, der neben der Kante mindestens sichtbar bleibt (ADR-0023 Nachtrag 8,
	// F-445): Mindestrest je Seite einer Naht = (KantenBandHalb + KantenSichtRest) · Faktor.
	KantenSichtRest = 8.0
)

// abrissTerme sind Amplitude, Periode und Phase der Sinusanteile der Abrisskante, abrissFaktor ihr gemeinsamer Faktor
// (ADR-0023 Nachtrag 6; Faktor 0,9 seit Nachtrag 7). Fest, ohne Zufall: Editor, Renderer und SVG-Export zeichnen
// dieselbe Linie.
var abrissTerme = [...][3]float64{{2.6, 71, 0.3}, {1.6, 29, 1.9}, {1.0, 13, 4.1}, {0.6, 7.3, 2.2}, {0.35, 3.1, 5.7}}

const abrissFaktor = 0.9

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
// mit faktor (KantenFaktor, mindestens 1) multipliziert. Die Querlage ist auf 1/100 Pixel gerundet und als Hundertstel
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

// Vereinfacht liefert die Linie mit weniger Stützpunkten (Douglas-Peucker, F-422): Jeder weggelassene Punkt liegt höchstens
// toleranz Pixel (quer zur Naht gemessen) neben der Verbindung seiner Nachbarn, Anfang und Ende bleiben. Alle Schichten
// einer Achse sind um feste Beträge quer verschobene Kopien derselben Linie, also weicht jede Kante des Vielecks um
// höchstens toleranz von der des Renderers ab; weil Spalt und Schichten dieselbe Linie nutzen, bleiben sie
// zueinander genau deckungsgleich. Nur der SVG-Export vereinfacht (ADR-0023 Nachtrag 10), der Renderer rechnet voll.
func (l *Kantenlinie) Vereinfacht(toleranz float64) *Kantenlinie {
	n := len(l.t)
	if n <= 2 {
		return l
	}
	tol := toleranz * 100 // m liegt in Hundertsteln
	behalte := make([]bool, n)
	behalte[0], behalte[n-1] = true, true
	stapel := [][2]int{{0, n - 1}}
	for len(stapel) > 0 {
		a, b := stapel[len(stapel)-1][0], stapel[len(stapel)-1][1]
		stapel = stapel[:len(stapel)-1]
		if b-a < 2 {
			continue
		}
		weit, wo := -1.0, -1
		for i := a + 1; i < b; i++ {
			// Querabstand von der Verbindung a–b an der Stelle t[i]
			soll := float64(l.m[a]) + float64(l.m[b]-l.m[a])*(l.t[i]-l.t[a])/(l.t[b]-l.t[a])
			if d := math.Abs(float64(l.m[i]) - soll); d > weit {
				weit, wo = d, i
			}
		}
		if weit > tol {
			behalte[wo] = true
			stapel = append(stapel, [2]int{a, wo}, [2]int{wo, b})
		}
	}
	v := &Kantenlinie{f: l.f}
	for i, b := range behalte {
		if b {
			v.t = append(v.t, l.t[i])
			v.m = append(v.m, l.m[i])
		}
	}
	return v
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

// quer liefert die Querlage der Kanten des Bands von bis bis (wie Band) an der Stelle t entlang der Naht n, in Pixeln
// der Ansicht: zwischen zwei Stützpunkten linear wie die Kanten des Vielecks, vor dem ersten und hinter dem letzten wie
// diese. So prüft die Lupe, ob ein Ansichtspixel im durchsichtigen Spalt liegt (ADR-0033 Nachtrag 2 Punkt 1).
func (l *Kantenlinie) quer(n Naht, t, von, bis float64) (a, b float64) {
	i := sort.SearchFloat64s(l.t, t) // erster Stützpunkt mit l.t[i] >= t
	var m float64
	switch {
	case i == 0:
		m = float64(l.m[0])
	case i == len(l.t):
		m = float64(l.m[len(l.m)-1])
	default:
		anteil := (t - l.t[i-1]) / (l.t[i] - l.t[i-1])
		m = float64(l.m[i-1]) + anteil*float64(l.m[i]-l.m[i-1])
	}
	mitte := float64(n.Lage*100) + m
	return (mitte + float64(hundertstel(von)*l.f)) / 100, (mitte + float64(hundertstel(bis)*l.f)) / 100
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

// Farben der Kante (Deckkraft in A): der helle Fasersaum am Rand der Bildteile, ihre Abdunklung und der Schatten der
// oberen bzw. linken Kante in den Spalt.
var (
	kantenSaum     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF}
	kantenDunkel   = color.NRGBA{}
	kantenSchatten = color.NRGBA{R: 0x10, G: 0x12, B: 0x14}
)

func mitDeckkraft(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// kantenSchichten ist die Gestaltung aller Formen (ADR-0023 Nachtrag 7, Papier-Abriss), unten zuerst gezeichnet, nach
// dem Referenzbild docs/adr/bilder/0023-naht-durchsichtig-muster.png (mittlere Zeile, 16 px). Der Spalt selbst
// (±KantenSpaltHalb) ist keine Schicht: Er ist durchsichtig, Renderer und SVG-Export lassen ihn leer, der Editor
// zeigt dort ein Schachbrett; die Schichten liegen danach darüber. Alle Bänder einer Gruppe beginnen an der
// Spaltkante ihres Teils, wo mehrere übereinanderliegen, wird es kräftiger; die Deckkraft ist so gewählt, dass das
// Produkt je Tiefe s (Abstand von der Spaltkante) das Muster über die Stufe gemittelt trifft:
//   - Fasersaum, weiß, im Muster 0,75·(1 − s/1,6) (ohne die Schwankung des Musters, die sich im SVG nicht gleich
//     zeigen ließe): Tiefe 0 bis 0,8 Deckung 0,56, 0,8 bis 1,6 Deckung 0,19;
//   - Abdunklung, schwarz, im Muster 0,18·e^(−s/0,8): Tiefe 0 bis 1 Deckung 0,10, 1 bis 2 Deckung 0,03; am unteren
//     bzw. rechten Teil × 0,7 (0,07 und 0,02);
//   - Schatten der oberen bzw. linken Kante in den Spalt, dunkel, im Muster 0,35·e^(−s/2,2): Tiefe 0 bis 1, 1 bis 2,
//     2 bis 4 und 4 bis 7 Deckung 0,28, 0,18, 0,09 und 0,03 (darunter unter 0,01, weggelassen).
var kantenSchichten = [...]Schicht{
	// Fasersaum, oben bzw. links
	{-KantenSpaltHalb - 1.6, -KantenSpaltHalb, mitDeckkraft(kantenSaum, 0x30)},
	{-KantenSpaltHalb - 0.8, -KantenSpaltHalb, mitDeckkraft(kantenSaum, 0x76)},
	// Fasersaum, unten bzw. rechts
	{KantenSpaltHalb, KantenSpaltHalb + 1.6, mitDeckkraft(kantenSaum, 0x30)},
	{KantenSpaltHalb, KantenSpaltHalb + 0.8, mitDeckkraft(kantenSaum, 0x76)},
	// Abdunklung, oben bzw. links
	{-KantenSpaltHalb - KantenRand, -KantenSpaltHalb, mitDeckkraft(kantenDunkel, 0x08)},
	{-KantenSpaltHalb - 1, -KantenSpaltHalb, mitDeckkraft(kantenDunkel, 0x13)},
	// Abdunklung, unten bzw. rechts (× 0,7)
	{KantenSpaltHalb, KantenSpaltHalb + KantenRand, mitDeckkraft(kantenDunkel, 0x05)},
	{KantenSpaltHalb, KantenSpaltHalb + 1, mitDeckkraft(kantenDunkel, 0x0D)},
	// Schatten der oberen bzw. linken Kante in den Spalt
	{-KantenSpaltHalb, -1, mitDeckkraft(kantenSchatten, 0x08)},
	{-KantenSpaltHalb, -4, mitDeckkraft(kantenSchatten, 0x10)},
	{-KantenSpaltHalb, -6, mitDeckkraft(kantenSchatten, 0x18)},
	{-KantenSpaltHalb, -7, mitDeckkraft(kantenSchatten, 0x20)},
}

// KantenSchichten liefert die Schichten der Kante in Zeichenreihenfolge (eine Kopie). Renderer, SVG-Export und Editor
// leeren zuerst den Spalt aller Nähte (Band −KantenSpaltHalb bis KantenSpaltHalb, siehe Spaltband) und zeichnen dann je
// Naht und Schicht ein Kantenband.
func KantenSchichten() []Schicht { return append([]Schicht(nil), kantenSchichten[:]...) }

// KantenFlach liefert dieselbe Gestaltung wie KantenSchichten, aber ohne Überlappung (F-422): Je Farbe werden die
// übereinanderliegenden Bänder an allen ihren Grenzen zerlegt, und jedes Stück trägt die Deckkraft, die alle Bänder
// darüber zusammen ergäben (1 − Π(1 − aᵢ), auf 1/255 gerundet). Weil die Bänder einer Farbe gleichfarbig sind, malt das
// dasselbe Bild. Der SVG-Export braucht so weniger verschiedene Bandbreiten (fünf statt dreizehn Pfade je Achse, jede
// Breite einmal, die Stücke als verschobene Verweise). Reihenfolge: je Farbe in der Reihenfolge von KantenSchichten, in
// jeder Farbe von oben bzw. links nach unten bzw. rechts. Grenzen sind Hundertstel, wie in Kantenlinie.Band.
func KantenFlach() []Schicht {
	var res []Schicht
	var fertig []color.NRGBA
	for _, s0 := range kantenSchichten {
		rgb := mitDeckkraft(s0.Farbe, 0)
		if slices.Contains(fertig, rgb) {
			continue
		}
		fertig = append(fertig, rgb)
		var gleich []Schicht
		var grenzen []int
		for _, s := range kantenSchichten {
			if mitDeckkraft(s.Farbe, 0) == rgb {
				gleich = append(gleich, s)
				grenzen = append(grenzen, hundertstel(s.Von), hundertstel(s.Bis))
			}
		}
		slices.Sort(grenzen)
		grenzen = slices.Compact(grenzen)
		for i := 0; i+1 < len(grenzen); i++ {
			von, bis := grenzen[i], grenzen[i+1]
			frei := 1.0 // Anteil, der durch alle Bänder darüber hindurchscheint
			for _, s := range gleich {
				if hundertstel(s.Von) <= von && bis <= hundertstel(s.Bis) {
					frei *= 1 - float64(s.Farbe.A)/255
				}
			}
			if frei == 1 {
				continue // kein Band deckt dieses Stück
			}
			res = append(res, Schicht{float64(von) / 100, float64(bis) / 100, mitDeckkraft(rgb, uint8(math.Round(255*(1-frei))))})
		}
	}
	return res
}

// Spaltband liefert die Grenzen des durchsichtigen Spalts quer zur Mittellinie (Faktor 1), für Kantenlinie.Band.
func Spaltband() (von, bis float64) { return -KantenSpaltHalb, KantenSpaltHalb }

// ImSpalt meldet, ob die Schicht ganz im durchsichtigen Spalt ihrer eigenen Naht liegt (der Schatten, ADR-0023
// Nachtrag 7); die anderen (Fasersaum, Abdunklung) liegen daneben auf den Bildteilen.
func (s Schicht) ImSpalt() bool { return s.Von >= -KantenSpaltHalb && s.Bis <= KantenSpaltHalb }

// NaehteBeruehren meldet, ob die Kante einer Naht in den Spalt einer anderen ragen kann (ADR-0023 Nachtrag 9,
// F-438): wenn es Nähte beider Achsen gibt (sie kreuzen sich) oder zwei Nähte derselben Achse näher als die ganze
// Bandhöhe (2·KantenBandHalb·Faktor) beieinanderliegen. Nur dann leeren Renderer, SVG-Export und Editor die Spalte nach
// dem Fasersaum und der Abdunklung noch einmal; sonst bleibt die Zeichnung wie vor F-438.
func NaehteBeruehren(naehte []Naht, faktor int) bool {
	var zeilen, spalten []int
	for _, n := range naehte {
		if n.Zeilen {
			zeilen = append(zeilen, n.Lage)
		} else {
			spalten = append(spalten, n.Lage)
		}
	}
	if len(zeilen) > 0 && len(spalten) > 0 {
		return true
	}
	nah := func(l []int) bool {
		slices.Sort(l)
		for i := 1; i < len(l); i++ {
			if float64(l[i]-l[i-1]) < 2*KantenBandHalb*float64(max(faktor, 1)) {
				return true
			}
		}
		return false
	}
	return nah(zeilen) || nah(spalten)
}

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
