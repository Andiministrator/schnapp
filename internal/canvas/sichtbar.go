// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// MinSichtbar ist, wie weit ein Objekt beim Verschieben oder Ziehen an einem Griff mindestens ins sichtbare Bild
// ragen muss, in Fensterpixeln (F-160, ADR-0021 Nachtrag 2): Es lässt sich nie ganz aus dem Bild schieben, bleibt also
// greifbar. Ein Objekt, das kleiner ist, muss ganz im Bild bleiben.
const MinSichtbar = 8.0

// sichtbarIn meldet je Achse, ob die Gestalt von o das sichtbare Bild (Ansicht) um mindestens MinSichtbar Fensterpixel
// schneidet (bei kleineren Objekten: um ihre ganze Ausdehnung).
func (s *Sitzung) sichtbarIn(o document.Objekt) (x, y bool) {
	g, gr := gestalt(o), s.D.Groesse()
	lo, hi := s.D.ZuAnsicht(g.Min), s.D.ZuAnsicht(g.Max)
	mind := MinSichtbar / s.Abb.Zoom
	ok := func(lo, hi, max float64) bool {
		need := math.Min(mind, hi-lo)
		return math.Min(hi, max)-math.Max(lo, 0) >= need-1e-9
	}
	return ok(lo.X, hi.X, float64(gr.X)), ok(lo.Y, hi.Y, float64(gr.Y))
}

// gestalt ist das Rechteck um das Sichtbare von o: beim Pfeil samt Köpfen und ohne Kontrollpunkt (Pfeil.Gestalt,
// F-829), sonst die Grenzen.
func gestalt(o document.Objekt) document.Rechteck {
	if p, ok := o.(document.Pfeil); ok {
		return p.Gestalt()
	}
	return o.Grenzen()
}

// begrenzeSichtbar liefert den größten Faktor t in [0, 1] (Bisektion), für den ok(t) gilt: ok(0) ist der Ausgangsstand,
// ok(1) das Ziel. Gilt schon ok(1), ist t = 1; ist auch der Ausgang ungültig (etwa eine Datei, die so geladen wurde),
// bleibt es bei t = 0.
func begrenzeSichtbar(ok func(t float64) bool) float64 {
	if ok(1) {
		return 1
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 40; i++ {
		m := (lo + hi) / 2
		if ok(m) {
			lo = m
		} else {
			hi = m
		}
	}
	return lo
}

// begrenzeVerschieben kürzt die Verschiebung d (Original) des Objekts o je Achse so, dass es im Bild sichtbar bleibt.
func (s *Sitzung) begrenzeVerschieben(o document.Objekt, d document.Punkt) document.Punkt {
	tx := begrenzeSichtbar(func(t float64) bool {
		x, _ := s.sichtbarIn(Verschiebe(o, document.Pt(d.X*t, 0)))
		return x
	})
	ty := begrenzeSichtbar(func(t float64) bool {
		_, y := s.sichtbarIn(Verschiebe(o, document.Pt(0, d.Y*t)))
		return y
	})
	return document.Pt(d.X*tx, d.Y*ty)
}

// begrenzeGriff liefert o mit dem Griff art an q (Original), wobei q je Achse so weit zum Ausgangsort des Griffs
// zurückgenommen wird, dass o im Bild sichtbar bleibt.
func (s *Sitzung) begrenzeGriff(o document.Objekt, art GriffArt, q document.Punkt) document.Objekt {
	h := q
	for _, g := range Griffe(o) {
		if g.Art == art {
			h = g.Ort
		}
	}
	tx := begrenzeSichtbar(func(t float64) bool {
		x, _ := s.sichtbarIn(s.zieheGriff(o, art, document.Pt(h.X+(q.X-h.X)*t, h.Y)))
		return x
	})
	ty := begrenzeSichtbar(func(t float64) bool {
		_, y := s.sichtbarIn(s.zieheGriff(o, art, document.Pt(h.X, h.Y+(q.Y-h.Y)*t)))
		return y
	})
	return s.zieheGriff(o, art, document.Pt(h.X+(q.X-h.X)*tx, h.Y+(q.Y-h.Y)*ty))
}
