// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/document"
)

// Zuschneiden und Stauchen (ADR-0023): Beide ändern nur die Angaben Zuschnitt bzw. Streifen des Dokuments, je ein
// Rückgängig-Schritt; das Original und die Beschriftungen bleiben unverändert.

// AnsichtMeldung ist eine einmalige Meldung nach Zuschneiden oder Stauchen für die Statusleiste.
type AnsichtMeldung int

const (
	AnsichtKeine     AnsichtMeldung = iota
	AnsichtVerborgen                // danach liegen Beschriftungen ganz im Verborgenen (Anzahl: VerborgeneObjekte)
	AnsichtLeer                     // das Band hätte die ganze Ansicht genommen; nichts geändert
	AnsichtZuviele                  // zu viele Streifen (document.MaxStreifen, document.MaxKacheln); nichts geändert
)

// rahmenWerkzeug meldet, ob w mit einem gezogenen Rahmen arbeitet statt ein Objekt anzulegen.
func rahmenWerkzeug(w Werkzeug) bool {
	return schwaerzt(w) || w == WerkzeugZuschneiden || w == WerkzeugStauchen
}

// NimmAnsichtMeldung liefert einmal die Meldung des letzten Zuschneidens oder Stauchens; danach ist sie gelöscht.
func (s *Sitzung) NimmAnsichtMeldung() AnsichtMeldung {
	m := s.ansichtMeldung
	s.ansichtMeldung = AnsichtKeine
	return m
}

// VerborgeneObjekte zählt die Beschriftungen, die ganz im Verborgenen liegen (außerhalb des Zuschnitts oder ganz in
// einem Streifen, document.Dokument.Sichtbar). Sie bleiben im Dokument, werden aber nirgends gezeichnet.
func (s *Sitzung) VerborgeneObjekte() int {
	n := 0
	for _, o := range s.D.Objekte() {
		if !s.D.ObjektSichtbar(o) {
			n++
		}
	}
	return n
}

// sichtbareObjekte sind die Objekte, die Treffer finden kann: Verborgenes ist nicht zu sehen und nicht zu greifen.
func (s *Sitzung) sichtbareObjekte() []document.Objekt {
	var res []document.Objekt
	for _, o := range s.D.Objekte() {
		if s.D.ObjektSichtbar(o) {
			res = append(res, o)
		}
	}
	return res
}

// ansichtGeaendert folgt einer geänderten Ansicht: Bild neu bauen, neu einpassen, eine verborgene Auswahl aufheben.
// verborgenVorher ist VerborgeneObjekte vor der Änderung: Liegen danach mehr Beschriftungen im Verborgenen, meldet
// die Statusleiste es (ADR-0023 Punkt 7: nicht unbemerkt) — nach Zuschneiden, Stauchen, Rückgängig und Wiederholen
// gleich, und nur bei Zuwachs (F-184).
func (s *Sitzung) ansichtGeaendert(verborgenVorher int) {
	s.bildAenderung++
	s.eingepasst = true
	s.Abb = Einpassen(s.D.Groesse(), s.fenster, Rand)
	if o, ok := s.D.Objekt(s.Auswahl); ok && !s.D.ObjektSichtbar(o) {
		s.Auswahl = 0
	}
	if s.VerborgeneObjekte() > verborgenVorher {
		s.ansichtMeldung = AnsichtVerborgen
	}
}

// ganzzahlig rundet einen Ansichtsrahmen auf ganze Ansichtspixel innerhalb der Ansicht.
func (s *Sitzung) ganzzahlig(r document.Rechteck) image.Rectangle {
	g := s.D.Groesse()
	k := func(v float64, m int) int { return max(0, min(m, int(math.Round(v)))) }
	return image.Rect(k(r.Min.X, g.X), k(r.Min.Y, g.Y), k(r.Max.X, g.X), k(r.Max.Y, g.Y))
}

// originalBereich rechnet einen Bereich [von, bis) der Ansicht (ganze Pixel, bis > von) einer Achse ins Original um:
// Anfang ist das Original des ersten Pixels, Ende das des letzten Pixels + 1. So endet der Bereich nie mitten in oder
// hinter einem Streifen an seinem Rand (ADR-0023 Punkt 1).
func (s *Sitzung) originalBereich(von, bis int, zeilen bool) (int, int) {
	o := func(v int) int {
		p := s.D.ZuOriginal(document.Pt(float64(v), float64(v)))
		if zeilen {
			return int(math.Round(p.Y))
		}
		return int(math.Round(p.X))
	}
	return o(von), o(bis-1) + 1
}

// zuschneideRahmen setzt beim Loslassen den Zuschnitt auf den gezogenen Rahmen (ADR-0023 Punkt 1). Unter der
// Mindestgröße passiert nichts.
func (s *Sitzung) zuschneideRahmen() {
	if !s.rahmen {
		return
	}
	s.rahmen = false
	r := document.R(s.rahmenA, s.rahmenB)
	grenze := MinObjekt / s.Abb.Zoom
	if r.Breite() < grenze || r.Hoehe() < grenze {
		return
	}
	a := s.ganzzahlig(r)
	if a.Empty() || a == (image.Rectangle{Max: s.D.Groesse()}) {
		// Rahmen über die ganze Ansicht: nichts zu tun. An der Ansicht geprüft, nicht am Zuschnitt — liegt ein Streifen
		// am Rand, wiche der umgerechnete Zuschnitt vom alten ab, ohne dass sich etwas Sichtbares ändert (F-182).
		return
	}
	x0, x1 := s.originalBereich(a.Min.X, a.Max.X, false)
	y0, y1 := s.originalBereich(a.Min.Y, a.Max.Y, true)
	z := image.Rect(x0, y0, x1, y1)
	if z == s.D.Zuschnitt() {
		return
	}
	verborgen := s.VerborgeneObjekte()
	if s.D.SetzeZuschnitt(z) == nil {
		s.ansichtGeaendert(verborgen)
	}
}

// HebeZuschnittAuf zeigt wieder das ganze Original (Doppelklick mit dem Werkzeug Zuschneiden, ADR-0023 Punkt 2), ein
// Rückgängig-Schritt. Die Streifen bleiben. Liefert, ob sich etwas geändert hat.
func (s *Sitzung) HebeZuschnittAuf() bool {
	if s.zug != keinZug {
		return false
	}
	ganz := s.D.Original().Rect
	if s.D.Zuschnitt() == ganz {
		return false
	}
	vor, verborgen := s.D.Stand(), s.VerborgeneObjekte()
	if s.D.SetzeZuschnitt(ganz) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.ansichtGeaendert(verborgen)
	return true
}

// stauchBand liefert das Band des laufenden Stauchen-Zugs in ganzen Ansichtspixeln (ADR-0023 Punkt 3): die Achse aus
// der Zugrichtung (mehr senkrecht gezogen: Zeilen), den Bereich [von, bis) auf dieser Achse und ob es die Mindestgröße
// erreicht.
func (s *Sitzung) stauchBand() (zeilen bool, von, bis int, ok bool) {
	d := s.rahmenB.Minus(s.rahmenA)
	zeilen = math.Abs(d.Y) > math.Abs(d.X)
	lo, hi := math.Min(s.rahmenA.X, s.rahmenB.X), math.Max(s.rahmenA.X, s.rahmenB.X)
	g := s.D.Groesse().X
	if zeilen {
		lo, hi = math.Min(s.rahmenA.Y, s.rahmenB.Y), math.Max(s.rahmenA.Y, s.rahmenB.Y)
		g = s.D.Groesse().Y
	}
	von = max(0, min(g, int(math.Round(lo))))
	bis = max(0, min(g, int(math.Round(hi))))
	return zeilen, von, bis, hi-lo >= MinObjekt/s.Abb.Zoom && bis > von
}

// bandRahmen ist das Band als Rechteck der Ansicht über die ganze Breite bzw. Höhe (Vorschau beim Ziehen).
func (s *Sitzung) bandRahmen() (document.Punkt, document.Punkt) {
	zeilen, von, bis, _ := s.stauchBand()
	g := s.D.Groesse()
	if zeilen {
		return document.Pt(0, float64(von)), document.Pt(float64(g.X), float64(bis))
	}
	return document.Pt(float64(von), 0), document.Pt(float64(bis), float64(g.Y))
}

// staucheRahmen nimmt beim Loslassen das Band als Streifen heraus (ADR-0023 Punkt 3). Überlappende oder anstoßende
// Streifen derselben Achse verschmelzen mit dem neuen, damit keine doppelte Naht entsteht.
func (s *Sitzung) staucheRahmen() {
	if !s.rahmen {
		return
	}
	s.rahmen = false
	zeilen, von, bis, ok := s.stauchBand()
	if !ok {
		return
	}
	g := s.D.Groesse()
	if laenge := map[bool]int{false: g.X, true: g.Y}[zeilen]; von == 0 && bis == laenge {
		s.ansichtMeldung = AnsichtLeer
		return
	}
	neu := document.Streifen{Zeilen: zeilen}
	neu.Von, neu.Bis = s.originalBereich(von, bis, zeilen)
	st := Verschmolzen(s.D.Streifen(), neu)
	verborgen := s.VerborgeneObjekte()
	if s.D.SetzeStreifen(st) != nil {
		s.ansichtMeldung = AnsichtZuviele
		return
	}
	s.ansichtGeaendert(verborgen)
}

// Verschmolzen liefert st mit neu, wobei alle Streifen derselben Achse, die neu überlappen oder an ihn stoßen, in ihm
// aufgehen (auch über mehrere Stufen: a stößt an b, b überlappt neu).
func Verschmolzen(st []document.Streifen, neu document.Streifen) []document.Streifen {
	rest := append([]document.Streifen(nil), st...)
	for {
		weiter := false
		var bleiben []document.Streifen
		for _, a := range rest {
			if a.Zeilen == neu.Zeilen && a.Von <= neu.Bis && a.Bis >= neu.Von {
				neu.Von, neu.Bis = min(neu.Von, a.Von), max(neu.Bis, a.Bis)
				weiter = true
				continue
			}
			bleiben = append(bleiben, a)
		}
		rest = bleiben
		if !weiter {
			break
		}
	}
	return append(rest, neu)
}

// EntferneNaht nimmt am Fensterpunkt p die nächste Stauch-Naht zurück (Doppelklick mit dem Werkzeug Stauchen,
// ADR-0023 Punkt 4): alle Streifen, die an dieser Naht verborgen sind, in einem Rückgängig-Schritt. Gefunden wird eine
// Naht, wenn p höchstens Toleranz Fensterpixel neben ihrem Zackenband liegt. Liefert, ob sich etwas geändert hat.
func (s *Sitzung) EntferneNaht(p document.Punkt) bool {
	if s.zug != keinZug || !s.imBild(p) {
		return false
	}
	a := s.Abb.ZuAnsicht(p)
	tol := Toleranz/s.Abb.Zoom + document.ZackenHoehe + document.ZackenRandHalb
	best, gefunden := math.Inf(1), document.Naht{}
	for _, n := range s.D.Naehte() {
		v := a.X
		if n.Zeilen {
			v = a.Y
		}
		if d := math.Abs(v - float64(n.Lage)); d <= tol && d < best {
			best, gefunden = d, n
		}
	}
	if math.IsInf(best, 1) {
		return false
	}
	z := s.D.Zuschnitt()
	var bleiben []document.Streifen
	for _, st := range s.D.Streifen() {
		lo, hi := z.Min.X, z.Max.X
		if st.Zeilen {
			lo, hi = z.Min.Y, z.Max.Y
		}
		q := s.D.ZuAnsicht(document.Pt(float64(st.Von), float64(st.Von)))
		lage := q.X
		if st.Zeilen {
			lage = q.Y
		}
		if st.Zeilen == gefunden.Zeilen && st.Bis > lo && st.Von < hi && lage == float64(gefunden.Lage) {
			continue
		}
		bleiben = append(bleiben, st)
	}
	vor, verborgen := s.D.Stand(), s.VerborgeneObjekte()
	if s.D.SetzeStreifen(bleiben) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.ansichtGeaendert(verborgen)
	return true
}
