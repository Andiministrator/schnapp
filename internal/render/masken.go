// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// Gemerkte Masken (F-430). Die Kanten der Stauch-Nähte einer Achse sind querverschobene Kopien derselben Linie: Nähte
// im Abstand ganzer Pixel ergeben dieselben Pfade im Ausschnitt (Koordinaten relativ zu dessen Ecke, als float32), und
// der Rasterkern rechnet aus denselben Pfaden dieselbe Deckung. Der Renderer merkt sich deshalb die Deckung großer
// Füllungen (Maske je Pixel, 16 Bit) unter ihren genauen Pfadkoordinaten und spielt sie bei der nächsten Füllung mit
// denselben Koordinaten ab, statt 65 000 Strecken neu zu rastern. Bitgleich ist das, weil die Maske nur von Größe und
// Koordinaten des Pfades abhängt und der Schlüssel beide vollständig vergleicht (kein Hash allein); eine Füllung, deren
// Pfad auch nur um eine Stufe anders ausfällt (Rundung der Koordinaten), rastert wie bisher.

// maskenCacheGrenze ist die Höchstzahl Bytes, die der Zwischenspeicher je Bild hält (Masken und Koordinaten); 0 schaltet
// ihn ab (TestMaskenCacheBitgleich, TestFuellenBitgleich). Darüber hinaus wird nichts mehr gemerkt, nie verdrängt.
var maskenCacheGrenze = 128 << 20

// maskenMinPunkte: Füllungen mit weniger Pfadpunkten rastert der Renderer ohne Zwischenspeicher (Beschriftungen).
const maskenMinPunkte = 2048

// Das Abspielen einer gemerkten Maske teilt sich auf höchstens maskenArbeiter Spaltenstreifen auf, je mindestens
// maskenMinProArbeiter Pixel (darunter kostet das Starten mehr, als es spart).
const (
	maskenArbeiter       = 8
	maskenMinProArbeiter = 100_000
)

// Zähler für Tests: Wiederverwendungen und Neuberechnungen gemerkter Masken, dazu die größte Bytezahl, die ein
// Zwischenspeicher je gehalten hat (maskenBytesMax, F-430: sie darf maskenCacheGrenze nie übersteigen).
var maskenTreffer, maskenFehl, maskenBytesMax atomic.Int64

// merkeBytes hebt maskenBytesMax auf n, wenn n größer ist.
func merkeBytes(n int) {
	for {
		alt := maskenBytesMax.Load()
		if int64(n) <= alt || maskenBytesMax.CompareAndSwap(alt, int64(n)) {
			return
		}
	}
}

type maskenSchluessel struct {
	w, h int
	hash uint64
}

type maskenEintrag struct {
	pkt   []uint32 // Pfad, vollständig verglichen (pfadAufnahme)
	maske []uint16 // Deckung je Pixel, Zeile für Zeile
}

type maskenCache struct {
	eintraege map[maskenSchluessel][]*maskenEintrag
	bytes     int
	scratch   pfadAufnahme
}

func neuerMaskenCache() *maskenCache {
	return &maskenCache{eintraege: map[maskenSchluessel][]*maskenEintrag{}}
}

// pfadAufnahme zeichnet die Befehle eines Pfades auf (und hasht sie) statt zu rastern.
type pfadAufnahme struct {
	pkt  []uint32 // Bitmuster der Koordinaten; MoveTo und ClosePath tragen eine Kennung davor bzw. allein (NaN, kommt als Koordinate nie vor)
	hash uint64
}

const (
	fnvStart = 14695981039346656037
	fnvPrim  = 1099511628211

	kennMoveTo    = 0x7fc00001
	kennClosePath = 0x7fc00002
)

func (a *pfadAufnahme) reset() { a.pkt, a.hash = a.pkt[:0], fnvStart }

func (a *pfadAufnahme) MoveTo(x, y float32) {
	u, v := math.Float32bits(x), math.Float32bits(y)
	a.pkt = append(a.pkt, kennMoveTo, u, v)
	a.hash = (((a.hash^kennMoveTo)*fnvPrim^uint64(u))*fnvPrim ^ uint64(v)) * fnvPrim
}

func (a *pfadAufnahme) LineTo(x, y float32) {
	u, v := math.Float32bits(x), math.Float32bits(y)
	a.pkt = append(a.pkt, u, v)
	a.hash = ((a.hash^uint64(u))*fnvPrim ^ uint64(v)) * fnvPrim
}

func (a *pfadAufnahme) ClosePath() {
	a.pkt = append(a.pkt, kennClosePath)
	a.hash = (a.hash ^ kennClosePath) * fnvPrim
}

func (c *maskenCache) suche(w, h int, a *pfadAufnahme) *maskenEintrag {
	for _, e := range c.eintraege[maskenSchluessel{w, h, a.hash}] {
		if slices.Equal(e.pkt, a.pkt) {
			return e
		}
	}
	return nil
}

// maskenWert sind die Maskenbreiten, die die Zeilenfunktionen kennen: 16 Bit gemerkt, 32 Bit frisch gerastert.
type maskenWert interface{ ~uint16 | ~uint32 }

// wendeBlockAn wendet einen Maskenblock (Zeilen ab y0 des Ausschnitts) mit zeile auf bild an.
func wendeBlockAn[M maskenWert](bild *image.NRGBA, ausschnitt, sichtbar image.Rectangle, y0 int, maske []M, zeile func([]M, []byte)) {
	breite := ausschnitt.Dx()
	von := max(sichtbar.Min.Y, ausschnitt.Min.Y+y0)
	bis := min(sichtbar.Max.Y, ausschnitt.Min.Y+y0+len(maske)/breite)
	for y := von; y < bis; y++ {
		z := maske[(y-ausschnitt.Min.Y-y0)*breite:]
		i := bild.PixOffset(sichtbar.Min.X, y)
		z = z[sichtbar.Min.X-ausschnitt.Min.X : sichtbar.Max.X-ausschnitt.Min.X]
		zeile(z, bild.Pix[i:i+4*len(z)])
	}
}

// ruht meldet, dass ein Pixel (vier Bytes NRGBA als Zahl, Alpha oben) ohne Deckung unverändert bleibt: deckend oder ganz
// (0,0,0,0), das bliebe (0,0,0,0), misch liefert dasselbe (F-430).
func ruht(px uint32) bool { return px>>24 == 0xff || px == 0 }

// mischeZeile mischt eine Zeile: maske hat je Pixel eine Deckung, pix vier Bytes je Pixel. Vier Pixel ohne jede Deckung
// auf ruhendem Grund (der Regelfall neben einem dünnen Band) überspringt sie mit einem Vergleich (F-430); die übrigen
// laufen einzeln, mit denselben Ausdrücken wie vorher, also bitgleich.
func mischeZeile[M maskenWert](e *einmischer, maske []M, pix []byte) {
	k := 0
	for ; k+4 <= len(maske); k += 4 {
		q := pix[4*k : 4*k+16 : 4*k+16]
		a0, a1, a2, a3 := binary.LittleEndian.Uint32(q[0:]), binary.LittleEndian.Uint32(q[4:]),
			binary.LittleEndian.Uint32(q[8:]), binary.LittleEndian.Uint32(q[12:])
		if maske[k]|maske[k+1]|maske[k+2]|maske[k+3] == 0 && ruht(a0) && ruht(a1) && ruht(a2) && ruht(a3) {
			continue
		}
		if (a0>>24)&(a1>>24)&(a2>>24)&(a3>>24) == 0xff {
			// Vier deckende Pixel: ohne Fallunterscheidung je Pixel (mischDeckend gilt für jede Deckung). An den Rändern der
			// Bänder wechseln Deckung 0, teilweise und voll unregelmäßig; Fehlsprünge kosteten dort mehr als die Rechnung.
			if e.sa == 0xffff && maske[k]&maske[k+1]&maske[k+2]&maske[k+3] == 0xffff {
				// Volle Deckung einer deckenden Quelle (Inneres bildgroßer Flächen): die Farbe, ohne Rechnung.
				v := uint32(e.voll.R) | uint32(e.voll.G)<<8 | uint32(e.voll.B)<<16 | 0xff<<24
				binary.LittleEndian.PutUint32(q[0:], v)
				binary.LittleEndian.PutUint32(q[4:], v)
				binary.LittleEndian.PutUint32(q[8:], v)
				binary.LittleEndian.PutUint32(q[12:], v)
				continue
			}
			e.mischDeckend(q[0:4:4], uint32(maske[k]))
			e.mischDeckend(q[4:8:8], uint32(maske[k+1]))
			e.mischDeckend(q[8:12:12], uint32(maske[k+2]))
			e.mischDeckend(q[12:16:16], uint32(maske[k+3]))
			continue
		}
		for j := k; j < k+4; j++ {
			e.mischeEins(uint32(maske[j]), pix[4*j:4*j+4:4*j+4])
		}
	}
	for ; k < len(maske); k++ {
		e.mischeEins(uint32(maske[k]), pix[4*k:4*k+4:4*k+4])
	}
}

// radiereZeile senkt die Deckung jedes Pixels der Zeile um die Deckung der Maske (ausradieren).
func radiereZeile[M maskenWert](maske []M, pix []byte) {
	for k, m := range maske {
		ma := uint32(m)
		if ma == 0 {
			continue
		}
		p := pix[4*k : 4*k+4 : 4*k+4]
		a := uint32(p[3]) * 0x101 * (0xffff - ma) / 0xffff >> 8
		if a == 0 {
			p[0], p[1], p[2], p[3] = 0, 0, 0, 0
			continue
		}
		p[3] = uint8(a)
	}
}

// rasterGemerkt füllt (oder radiert) wie einmischen bzw. ausradieren, mit gemerkter Maske: Ist der Pfad im
// Zwischenspeicher, spielt es die Maske ab; sonst rastert es und merkt sie sich, soweit Platz ist. false: nicht
// zuständig (Zwischenspeicher aus oder Pfad zu klein), der Aufrufer rastert wie bisher.
func (z zeichner) rasterGemerkt(punkte int, pfade func(pfad), ausschnitt image.Rectangle, farbe color.NRGBA, loeschen bool) bool {
	c := z.masken
	if c == nil || maskenCacheGrenze <= 0 || punkte < maskenMinPunkte {
		return false
	}
	w, h := ausschnitt.Dx(), ausschnitt.Dy()
	c.scratch.reset()
	pfade(&c.scratch)
	e := neuerEinmischer(z.bild, farbe)
	sichtbar := ausschnitt.Intersect(z.bild.Rect)
	zeile16 := func(m []uint16, pix []byte) {
		if loeschen {
			radiereZeile(m, pix)
		} else {
			mischeZeile(&e, m, pix)
		}
	}
	zeile32 := func(m []uint32, pix []byte) {
		if loeschen {
			radiereZeile(m, pix)
		} else {
			mischeZeile(&e, m, pix)
		}
	}
	if g := c.suche(w, h, &c.scratch); g != nil {
		maskenTreffer.Add(1)
		// Jedes Pixel hängt nur von sich und seiner Deckung ab: Spaltenstreifen darf man gleichzeitig abspielen, das Ergebnis
		// ist dasselbe wie Zeile für Zeile.
		teile := min(runtime.GOMAXPROCS(0), maskenArbeiter, w*h/maskenMinProArbeiter)
		if teile < 2 || sichtbar.Dx() < teile*64 {
			teile = 1
		}
		var wg sync.WaitGroup
		for t := 0; t < teile; t++ {
			streifen := sichtbar
			streifen.Min.X = sichtbar.Min.X + sichtbar.Dx()*t/teile
			streifen.Max.X = sichtbar.Min.X + sichtbar.Dx()*(t+1)/teile
			abspielen := func() {
				for y0 := 0; y0 < h; y0 += 64 {
					saveprogress.Puls()
					wendeBlockAn(z.bild, ausschnitt, streifen, y0, g.maske[y0*w:min(y0+64, h)*w], zeile16)
				}
			}
			if teile == 1 {
				abspielen()
				break
			}
			wg.Add(1)
			go func() { defer wg.Done(); abspielen() }()
		}
		wg.Wait()
		return true
	}
	maskenFehl.Add(1)
	var neu *maskenEintrag
	if bedarf := 2*w*h + 4*len(c.scratch.pkt); c.bytes+bedarf <= maskenCacheGrenze {
		neu = &maskenEintrag{maske: make([]uint16, w*h)}
		c.bytes += bedarf
		merkeBytes(c.bytes)
	}
	z.ras.Reset(w, h)
	pfade(z.ras)
	z.ras.Bloecke(64, func(y0 int, m []uint32) {
		saveprogress.Puls()
		if neu != nil {
			dst := neu.maske[y0*w : y0*w+len(m)]
			for i, v := range m {
				dst[i] = uint16(v)
			}
		}
		wendeBlockAn(z.bild, ausschnitt, sichtbar, y0, m, zeile32)
	})
	if neu != nil {
		neu.pkt = slices.Clone(c.scratch.pkt)
		k := maskenSchluessel{w, h, c.scratch.hash}
		c.eintraege[k] = append(c.eintraege[k], neu)
	}
	return true
}
