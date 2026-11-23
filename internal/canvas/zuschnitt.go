// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
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
	AnsichtRand                     // das Band reichte bis an den Rand und wurde zum Zuschnitt (F-222, E-21)
)

// rahmenWerkzeug meldet, ob w mit einem gezogenen Rahmen arbeitet statt ein Objekt anzulegen.
func rahmenWerkzeug(w Werkzeug) bool {
	return schwaerzt(w) || w == WerkzeugZuschneiden || w == WerkzeugStauchen
}

// ohneGriffe meldet, ob bei w die Griffe eines ausgewählten Objekts nicht gelten: Zuschneiden und Stauchen ziehen
// immer einen Rahmen (F-174). SetzeWerkzeug hebt die Auswahl auf, Druecken wertet dann auch keine Griffe aus.
func ohneGriffe(w Werkzeug) bool { return w == WerkzeugZuschneiden || w == WerkzeugStauchen }

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

// sichtbareObjekte sind die Objekte, die Treffer finden kann: Verborgenes ist nicht zu sehen und nicht zu greifen,
// ausgeblendete und gesperrte Ebenen auch nicht (ADR-0032 Punkte 4 und 5).
func (s *Sitzung) sichtbareObjekte() []document.Objekt {
	var res []document.Objekt
	for _, o := range s.D.Objekte() {
		if s.D.ObjektSichtbar(o) && o.Ebenenwerte().Bearbeitbar() {
			res = append(res, o)
		}
	}
	return res
}

// ansichtGeaendert folgt einer geänderten Ansicht: Bild neu bauen, neu einpassen, eine verborgene Auswahl aufheben.
// verborgenVorher ist VerborgeneObjekte vor der Änderung: Liegen danach mehr Beschriftungen im Verborgenen, meldet
// die Statusleiste es (ADR-0023 Punkt 7: nicht unbemerkt) — nach Zuschneiden, Stauchen, Rückgängig und Wiederholen
// gleich, und nur bei Zuwachs (F-184).
//
// Beim Werkzeug Zuschneiden zeigt die Leinwand die Vollansicht (ADR-0023 Nachtrag 5). Ändert sich nur der Zuschnitt,
// bleibt sie dasselbe Bild; dann bleibt auch die Abbildung, wie der Nutzer sie hatte (KA-5): Eingepasst bleibt
// eingepasst, eine selbst gezoomte oder verschobene Ansicht bleibt stehen. Weil die Ansichtskoordinaten am Zuschnitt
// hängen (die Vollansicht liegt bei −VollVersatz), rückt der Versatz um die Verschiebung von VollVersatz mal Zoom
// nach — so steht jeder Punkt der Vollansicht am selben Fensterpunkt. Sonst wird neu eingepasst.
func (s *Sitzung) ansichtGeaendert(vorher ansichtVorher) {
	s.bildAenderung++
	if s.zuschnittModus() && s.D.VollGroesse() == vorher.vollGroesse && slices.Equal(s.D.Streifen(), vorher.streifen) {
		if s.eingepasst {
			s.Abb = s.einpassAbb()
		} else {
			d := s.D.VollVersatz().Sub(vorher.vollVersatz)
			s.Abb.Versatz = s.Abb.Versatz.Plus(document.Pt(float64(d.X), float64(d.Y)).Mal(s.Abb.Zoom))
		}
	} else {
		s.eingepasst = true
		s.Abb = s.einpassAbb()
	}
	if o, ok := s.D.Objekt(s.Auswahl); ok && !s.D.ObjektSichtbar(o) {
		s.Auswahl = 0
	}
	if s.VerborgeneObjekte() > vorher.verborgen {
		s.ansichtMeldung = AnsichtVerborgen
	}
}

// ansichtVorher hält fest, was ansichtGeaendert vom Stand vor einer Änderung von Zuschnitt oder Streifen braucht.
type ansichtVorher struct {
	verborgen   int // VerborgeneObjekte
	vollGroesse image.Point
	vollVersatz image.Point
	streifen    []document.Streifen
}

// vorAnsicht merkt den Stand vor einer Änderung von Zuschnitt oder Streifen (für ansichtGeaendert).
func (s *Sitzung) vorAnsicht() ansichtVorher {
	return ansichtVorher{verborgen: s.VerborgeneObjekte(), vollGroesse: s.D.VollGroesse(), vollVersatz: s.D.VollVersatz(),
		streifen: s.D.Streifen()}
}

// ganzzahlig rundet einen Ansichtsrahmen auf ganze Ansichtspixel innerhalb der Bildfläche (beim Zuschneiden die
// Vollansicht, ADR-0023 Nachtrag 5).
func (s *Sitzung) ganzzahlig(r document.Rechteck) image.Rectangle {
	f := s.Bildflaeche()
	k := func(v float64, lo, hi int) int { return max(lo, min(hi, int(math.Round(v)))) }
	return image.Rect(k(r.Min.X, f.Min.X, f.Max.X), k(r.Min.Y, f.Min.Y, f.Max.Y), k(r.Max.X, f.Min.X, f.Max.X),
		k(r.Max.Y, f.Min.Y, f.Max.Y))
}

// vollBereich rechnet einen Bereich [von, bis) der Vollansicht (ganze Pixel, bis > von) einer Achse ins Original um,
// wie originalBereich: vom ersten Pixel bis hinter das letzte.
func (s *Sitzung) vollBereich(von, bis int, zeilen bool) (int, int) {
	o := func(v int) int {
		p := s.D.VollZuOriginal(document.Pt(float64(v), float64(v)))
		if zeilen {
			return int(math.Round(p.Y))
		}
		return int(math.Round(p.X))
	}
	return o(von), o(bis-1) + 1
}

// zuschnittGriff liefert den Griff des Zuschnittrahmens am Fensterpunkt p (ADR-0023 Nachtrag 5): Greifbar ist die
// ganze Kante innerhalb von Toleranz Fensterpixeln, an einer Ecke gewinnt die Ecke; liegen zwei gegenüberliegende Kanten
// gleich nah (sehr kleiner Zuschnitt), die nähere, bei Gleichstand die linke bzw. obere.
func (s *Sitzung) zuschnittGriff(p document.Punkt) (GriffArt, bool) {
	g := s.D.Groesse()
	a, b := s.Abb.ZuFenster(document.Pt(0, 0)), s.Abb.ZuFenster(document.Pt(float64(g.X), float64(g.Y)))
	seite := func(v, lo, hi float64) int { // -1 Anfang, +1 Ende, 0 keine
		d0, d1 := math.Abs(v-lo), math.Abs(v-hi)
		switch {
		case d0 <= Toleranz && d0 <= d1:
			return -1
		case d1 <= Toleranz:
			return 1
		}
		return 0
	}
	innen := func(v, lo, hi float64) bool { return v >= lo-Toleranz && v <= hi+Toleranz }
	var sx, sy int
	if innen(p.Y, a.Y, b.Y) {
		sx = seite(p.X, a.X, b.X)
	}
	if innen(p.X, a.X, b.X) {
		sy = seite(p.Y, a.Y, b.Y)
	}
	art := map[[2]int]GriffArt{
		{-1, -1}: GriffNW, {0, -1}: GriffN, {1, -1}: GriffNO, {1, 0}: GriffO,
		{1, 1}: GriffSO, {0, 1}: GriffS, {-1, 1}: GriffSW, {-1, 0}: GriffW,
	}[[2]int{sx, sy}]
	return art, art != 0
}

// zieheZuschnitt führt Kante oder Ecke des Zuschnittrahmens zum Fensterpunkt p, begrenzt auf die Vollansicht. Der
// Rahmen geht immer vom Zuschnitt beim Druck aus; über die Gegenkante gezogen, klappt er um.
func (s *Sitzung) zieheZuschnitt(p document.Punkt) {
	g := s.D.Groesse()
	a, b := document.Pt(0, 0), document.Pt(float64(g.X), float64(g.Y))
	q := s.klemmeAnsicht(s.Abb.ZuAnsicht(p))
	switch s.griff {
	case GriffNW, GriffW, GriffSW:
		a.X = q.X
	case GriffNO, GriffO, GriffSO:
		b.X = q.X
	}
	switch s.griff {
	case GriffNW, GriffN, GriffNO:
		a.Y = q.Y
	case GriffSW, GriffS, GriffSO:
		b.Y = q.Y
	}
	s.rahmenA, s.rahmenB = a, b
}

// ZuschnittRahmen liefert beim Werkzeug Zuschneiden den Rahmen, den die Leinwand mit Griffen zeigt (Fensterpunkte):
// während eines Zugs den gezogenen, sonst den Zuschnitt (ADR-0023 Nachtrag 5). ok = false bei anderen Werkzeugen.
func (s *Sitzung) ZuschnittRahmen() (a, b document.Punkt, ok bool) {
	if !s.zuschnittModus() {
		return document.Punkt{}, document.Punkt{}, false
	}
	if a, b, ok := s.Rahmen(); ok {
		return a, b, true
	}
	g := s.D.Groesse()
	return s.Abb.ZuFenster(document.Pt(0, 0)), s.Abb.ZuFenster(document.Pt(float64(g.X), float64(g.Y))), true
}

// originalBereich rechnet einen Bereich [von, bis) der Ansicht (ganze Pixel, bis > von) einer Achse ins Original um:
// Anfang ist das Original des ersten Pixels, Ende das des letzten Pixels + 1. So endet der Bereich nie mitten in oder
// hinter einem Streifen an seinem Rand (ADR-0023 Punkt 1).
func (s *Sitzung) originalBereich(von, bis int, zeilen bool) (int, int) {
	return originalBereich(s.D, von, bis, zeilen)
}

// originalBereich ist Sitzung.originalBereich für ein Dokument ohne Sitzung (SchwaerzBereich).
func originalBereich(d *document.Dokument, von, bis int, zeilen bool) (int, int) {
	o := func(v int) int {
		p := d.ZuOriginal(document.Pt(float64(v), float64(v)))
		if zeilen {
			return int(math.Round(p.Y))
		}
		return int(math.Round(p.X))
	}
	return o(von), o(bis-1) + 1
}

// zuschneideRahmen setzt beim Loslassen den Zuschnitt auf den gezogenen Rahmen (ADR-0023 Punkt 1, Nachtrag 5): ein
// neuer Rahmen oder der mit einem Griff geänderte Zuschnitt, beides in der Vollansicht. Unter der Mindestgröße
// passiert nichts.
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
	a = a.Add(s.D.VollVersatz()) // in die Vollansicht: Der Rahmen darf über den bisherigen Zuschnitt hinausreichen
	x0, x1 := s.vollBereich(a.Min.X, a.Max.X, false)
	y0, y1 := s.vollBereich(a.Min.Y, a.Max.Y, true)
	z := image.Rect(x0, y0, x1, y1)
	if z == s.D.Zuschnitt() {
		return
	}
	verborgen := s.vorAnsicht()
	if s.D.SetzeZuschnitt(z) == nil {
		s.ansichtGeaendert(verborgen)
	}
}

// HebeZuschnittAuf zeigt wieder das ganze Bild (Knopf „Ganzes Bild“, Doppelklick mit dem Werkzeug Zuschneiden;
// ADR-0023 Punkt 2 und Nachtrag 3), ein Rückgängig-Schritt: Der Zuschnitt wird das ganze Original, und Streifen, die
// dann am Rand des Originals anliegen, fallen weg — sie hätten keine Naht und wären sonst nicht mehr zurückzuholen
// (F-222; so entstanden sie vor dem Nachtrag, etwa in älteren Dateien). Alle anderen Streifen bleiben. Liefert, ob sich
// etwas geändert hat.
func (s *Sitzung) HebeZuschnittAuf() bool {
	if s.zug != keinZug || s.gesperrt() {
		return false
	}
	ganz, st, ok := s.ganzesBild()
	if !ok {
		return false
	}
	vor, verborgen := s.D.Stand(), s.vorAnsicht()
	if s.D.SetzeAnsicht(ganz, st) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.ansichtGeaendert(verborgen)
	return true
}

// KannGanzesBild meldet, ob HebeZuschnittAuf etwas ändern würde (Knopf „Ganzes Bild“ an oder grau).
func (s *Sitzung) KannGanzesBild() bool {
	_, _, ok := s.ganzesBild()
	return ok && s.zug == keinZug && !s.gesperrt()
}

// ganzesBild liefert Zuschnitt und Streifen für „Ganzes Bild“ und ob sie vom jetzigen Stand abweichen.
func (s *Sitzung) ganzesBild() (image.Rectangle, []document.Streifen, bool) {
	ganz := image.Rectangle{Max: s.D.OriginalGroesse()}
	var bleiben []document.Streifen
	for _, st := range s.D.Streifen() {
		lo, hi := ganz.Min.X, ganz.Max.X
		if st.Zeilen {
			lo, hi = ganz.Min.Y, ganz.Max.Y
		}
		if st.Von <= lo || st.Bis >= hi {
			continue // liegt am Rand des Originals an: ohne Naht
		}
		bleiben = append(bleiben, st)
	}
	return ganz, bleiben, s.D.Zuschnitt() != ganz || len(bleiben) != len(s.D.Streifen())
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
	ok = hi-lo >= MinObjekt/s.Abb.Zoom && bis > von
	// Mindestrest je Seite (ADR-0023 Nachträge 4 und 8, F-226, F-445): Zwischen Naht und Bildrand müssen mindestens die
	// halbe Bandhöhe plus KantenSichtRest sichtbare Pixel bleiben, sonst bestünde die Seite nur aus der Kante. Ein kleinerer Rest ginge ohnehin im Band unter; das Band
	// reicht dann bis an den Rand (und wird, wie jedes Band am Rand, zum Zuschnitt, F-222). Vorschau (bandRahmen) und
	// Ergebnis (staucheRahmen) lesen dasselbe Band (F-240).
	halb := s.D.KantenFaktor() * (document.KantenBandHalb + document.KantenSichtRest) // F-445, ADR-0023 Nachtrag 8
	if ok && von < halb {
		von = 0
	}
	if ok && g-bis < halb {
		bis = g
	}
	return zeilen, von, bis, ok
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
	// Das Band trägt den Mindestrest schon (stauchBand, F-226/F-240).
	// Bleibt weniger als die Mindestgröße eines Rahmens übrig, gilt das wie „nichts übrig“: Ein Zug kann nur im Bild
	// beginnen (F-150) und trifft die erste Zeile selten genau; ein Rest von 4 Pixeln war kein gewolltes Ergebnis
	// (Echtlauf #92).
	if laenge := map[bool]int{false: g.X, true: g.Y}[zeilen]; float64(laenge-(bis-von)) < MinObjekt/s.Abb.Zoom {
		s.ansichtMeldung = AnsichtLeer
		return
	}
	if laenge := map[bool]int{false: g.X, true: g.Y}[zeilen]; von == 0 || bis == laenge {
		s.randZuschnitt(zeilen, von, bis, laenge)
		return
	}
	neu := document.Streifen{Zeilen: zeilen}
	neu.Von, neu.Bis = s.originalBereich(von, bis, zeilen)
	st := Verschmolzen(s.D.Streifen(), neu)
	verborgen := s.vorAnsicht()
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

// randZuschnitt nimmt ein Band, das bis an den Rand der Ansicht reicht, als Zuschnitt weg statt als Streifen (F-222,
// E-21, ADR-0023 Nachtrag 3): Am Rand gäbe es keine Naht, über die es zurückzuholen wäre; den Zuschnitt hebt „Ganzes
// Bild“ auf. Die andere Achse behält ihren Zuschnitt. laenge ist die Länge der Ansicht auf der Achse des Bands.
func (s *Sitzung) randZuschnitt(zeilen bool, von, bis, laenge int) {
	rv, rb := bis, laenge // was bleibt: hinter dem Band (Band am Anfang) …
	if von > 0 {
		rv, rb = 0, von // … oder davor (Band am Ende)
	}
	a, b := s.originalBereich(rv, rb, zeilen)
	z := s.D.Zuschnitt()
	if zeilen {
		z.Min.Y, z.Max.Y = a, b
	} else {
		z.Min.X, z.Max.X = a, b
	}
	if z == s.D.Zuschnitt() {
		return
	}
	verborgen := s.vorAnsicht()
	if s.D.SetzeZuschnitt(z) != nil {
		return
	}
	s.ansichtGeaendert(verborgen)
	if s.ansichtMeldung == AnsichtKeine {
		s.ansichtMeldung = AnsichtRand
	}
}

// EntferneNaht nimmt am Fensterpunkt p die nächste Stauch-Naht zurück (Doppelklick mit dem Werkzeug Stauchen,
// ADR-0023 Punkt 4): alle Streifen, die an dieser Naht verborgen sind, in einem Rückgängig-Schritt. Gefunden wird eine
// Naht, wenn p höchstens Toleranz Fensterpixel neben ihrer Kante liegt (halbe Bandhöhe, ADR-0023 Nachtrag 6). Liefert,
// ob sich etwas geändert hat.
func (s *Sitzung) EntferneNaht(p document.Punkt) bool {
	if s.zug != keinZug || s.gesperrt() || !s.imBild(p) {
		return false
	}
	a := s.Abb.ZuAnsicht(p)
	tol := Toleranz/s.Abb.Zoom + float64(s.D.KantenFaktor())*document.KantenBandHalb
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
	vor, verborgen := s.D.Stand(), s.vorAnsicht()
	if s.D.SetzeStreifen(bleiben) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.ansichtGeaendert(verborgen)
	return true
}
