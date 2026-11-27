// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Regler der Strichstärke (Karte #352, ADR-0021 Nachtrag 22): zusätzlich zu − / + (StrichStufen) lässt sich die Stärke
// am Regler ziehen. Die Skala ist logarithmisch von ReglerMin bis ReglerMax, gerundet auf ganze Pixel; das Ziehen ist wie
// im Farbwähler ein Rückgängig-Schritt am gewählten Objekt.

// ReglerMin und ReglerMax sind die Enden des Reglers in Pixeln des Originals (die kleinste und größte Stufe).
var (
	ReglerMin = StrichStufen[0]
	ReglerMax = StrichStufen[len(StrichStufen)-1]
)

// BreiteAusRegler ist die Strichstärke zur Lage f (0 links, 1 rechts, begrenzt), auf ganze Pixel gerundet.
func BreiteAusRegler(f float64) float64 {
	f = math.Max(0, math.Min(1, f))
	b := ReglerMin * math.Pow(ReglerMax/ReglerMin, f)
	return math.Max(ReglerMin, math.Min(ReglerMax, math.Round(b)))
}

// ReglerLage ist die Lage (0 bis 1) der Strichstärke b am Regler; Stärken außerhalb stehen am Rand.
func ReglerLage(b float64) float64 {
	if !(b > ReglerMin) {
		return 0
	}
	if b >= ReglerMax {
		return 1
	}
	return math.Log(b/ReglerMin) / math.Log(ReglerMax/ReglerMin)
}

// BreiteZugLaeuft meldet, ob am Regler der Strichstärke gezogen wird.
func (s *Sitzung) BreiteZugLaeuft() bool { return s.breiteZug }

// BeginneBreiteZug beginnt das Ziehen am Regler: Das gewählte Pfeil-, Kasten- oder Lupenobjekt ändert sich bis zum
// Loslassen mit und wird dann ein Rückgängig-Schritt. Liefert false, wenn gerade nicht gezogen werden darf.
func (s *Sitzung) BeginneBreiteZug() bool {
	if s.Zieht() || s.gesperrt() || s.zug == zugText {
		return false
	}
	s.breiteZug, s.breiteZugObj = true, false
	s.merkeZugStil(s.stilZiel())
	if o, ok := s.strichObjekt(); ok && s.D.BeginneZug(o.Kennung()) == nil {
		s.breiteZugObj, s.breiteZugVor = true, s.D.Stand()
	}
	return true
}

// BreiteZugSetze setzt während des Ziehens die Strichstärke b für folgende Objekte und das gewählte.
func (s *Sitzung) BreiteZugSetze(b float64) {
	if !s.breiteZug || !(b > 0) || b > document.MaxGroesse {
		return
	}
	s.aendereZielStil(func(st *Stil) { st.Breite = b })
	if s.breiteZugObj {
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			_ = s.D.ZugSetze(mitBreite(o, b))
		}
	}
}

// BeendeBreiteZug schließt das Ziehen ab: ein Rückgängig-Schritt am Objekt, keiner, wenn die Stärke gleich blieb.
func (s *Sitzung) BeendeBreiteZug() {
	if !s.breiteZug {
		return
	}
	s.breiteZug, s.schattenZugText = false, nil
	if s.breiteZugObj {
		s.breiteZugObj = false
		_ = s.D.BeendeZug()
		s.merkeAenderung(s.breiteZugVor)
	}
}

// BrecheBreiteZugAb stellt das Objekt und den Stil seines Werkzeugs wie vor dem Ziehen her (Esc, Zeiger-Cancel; seit
// F-931 auch den Stil, der über Sitzungen gemerkt wird).
func (s *Sitzung) BrecheBreiteZugAb() {
	if !s.breiteZug {
		return
	}
	s.breiteZug = false
	s.zugStilZurueck()
	if s.schattenZugText != nil { // Schattenzug in der Texteingabe: Text wie vor dem Ziehen (Kritik K-1 #363b)
		if s.zug == zugText {
			_ = s.D.ZugSetze(s.schattenZugText)
		}
		s.schattenZugText = nil
	}
	if s.breiteZugObj {
		s.breiteZugObj = false
		_ = s.D.BrecheZugAb()
	}
}
