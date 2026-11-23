// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image/color"
	"math/rand"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-430: Mit gemerkten Masken (masken.go) ist Bild bitgleich zu Bild ohne Zwischenspeicher, in jeder Form und auf
// deckendem wie durchsichtigem Grund, und der Zwischenspeicher wird wirklich benutzt (Treffer), auch wenn er zu
// klein ist, um alles zu merken (maskenCacheGrenze, Rest wird frisch gerastert).
func TestMaskenCacheBitgleich(t *testing.T) {
	alt := maskenCacheGrenze
	defer func() { maskenCacheGrenze = alt }()
	for _, f := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		for _, opak := range []bool{false, true} {
			var d *document.Dokument
			breite := 6000
			if f == document.KanteZacken {
				breite = 24000 // Schritt 8: erst so lang hat ein Band genug Pfadpunkte (maskenMinPunkte)
			}
			if opak {
				d = opakDokument(t, breite, 500, f, 20)
			} else {
				d = nahtDokument(t, breite, 500, f, 20)
			}
			hinzu(t, d, document.Kasten{Bereich: document.R(document.Pt(100, 20), document.Pt(5000, 480)),
				Eckradius: 9, Fuellung: color.NRGBA{R: 200, G: 30, B: 90, A: 77}})
			maskenCacheGrenze = 0
			soll := bild(t, d)
			for _, grenze := range []int{alt, 3 << 20} {
				maskenCacheGrenze = grenze
				t0, f0 := maskenTreffer.Load(), maskenFehl.Load()
				maskenBytesMax.Store(0)
				ist := bild(t, d)
				if got := maskenBytesMax.Load(); got > int64(grenze) {
					t.Errorf("%v opak=%v Grenze %d: Zwischenspeicher hielt bis zu %d Bytes (Kritik A-3, B-5)", f, opak, grenze, got)
				}
				if grenze == 3<<20 && maskenFehl.Load() == f0 {
					t.Errorf("%v opak=%v Grenze %d: kein Fehlgriff, die Masken passen nicht alle in 3 MiB und müssen frisch gerastert werden", f, opak, grenze)
				}
				if !bytes.Equal(soll.Pix, ist.Pix) {
					t.Errorf("%v opak=%v Grenze %d: Bild weicht vom Rastern ohne Zwischenspeicher ab", f, opak, grenze)
				}
				if maskenTreffer.Load() == t0 {
					t.Errorf("%v opak=%v Grenze %d: keine Treffer (%d Fehlgriffe)", f, opak, grenze, maskenFehl.Load()-f0)
				}
			}
		}
	}
}

// F-430: Der Zwischenspeicher vergleicht den Pfad vollständig, nicht nur Hash und Länge: Zwei Aufnahmen mit gleichem Hash
// und gleicher Länge, aber einem anderen Punkt, finden einander nicht (Kritik A-3, B-5, Mutant E19).
func TestMaskenSucheVergleichtVollstaendig(t *testing.T) {
	c := neuerMaskenCache()
	a := pfadAufnahme{}
	a.reset()
	a.MoveTo(1, 2)
	a.LineTo(30, 4)
	a.LineTo(5, 60)
	a.ClosePath()
	c.eintraege[maskenSchluessel{10, 10, a.hash}] = []*maskenEintrag{{pkt: slices.Clone(a.pkt), maske: make([]uint16, 100)}}
	if c.suche(10, 10, &a) == nil {
		t.Fatal("dieselbe Aufnahme nicht gefunden")
	}
	b := pfadAufnahme{pkt: slices.Clone(a.pkt), hash: a.hash} // gleicher Hash, gleiche Länge
	b.pkt[5]++                                                // ein anderer Punkt
	if g := c.suche(10, 10, &b); g != nil {
		t.Errorf("Aufnahme mit anderem Punkt fand einen Eintrag (%v)", g)
	}
	if g := c.suche(11, 10, &a); g != nil {
		t.Errorf("andere Breite fand einen Eintrag")
	}
}

// mischDeckend rechnet dieselben Zahlen wie misch (F-430): alle Deckungen an den Rändern und zufällige, mehrere Quellen,
// deckende Pixel in allen Farbwerten.
func TestMischDeckendGleich(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	quellen := []color.NRGBA{{R: 255, G: 255, B: 255, A: 0x30}, {A: 0x08}, {R: 16, G: 18, B: 20, A: 0x20}, {R: 1, G: 2, B: 3, A: 1},
		{R: 200, G: 30, B: 90, A: 254}, {R: 9, G: 99, B: 199, A: 255}}
	deckungen := []uint32{0, 1, 2, 0x7fff, 0x8000, 0xfffe, 0xffff}
	for i := 0; i < 20000; i++ {
		deckungen = append(deckungen, uint32(r.Intn(0x10000)))
	}
	for _, q := range quellen {
		e := neuerEinmischer(nil, q)
		for _, ma := range deckungen {
			for k := 0; k < 4; k++ {
				p := []byte{byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), 0xff}
				if k == 0 {
					p[0], p[1], p[2] = 0, 255, byte(ma)
				}
				a, b := append([]byte(nil), p...), append([]byte(nil), p...)
				e.misch(a, ma)
				e.mischDeckend(b, ma)
				if !bytes.Equal(a, b) {
					t.Fatalf("Quelle %v, Deckung %#x, Pixel %v: misch %v, mischDeckend %v", q, ma, p, a, b)
				}
			}
		}
	}
}
