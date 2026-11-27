// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Mehrfachauswahl (ADR-0039): Auswahl ist die Hauptauswahl (Griffe, rechte Leiste, Farbe, Texteingabe wirken auf sie),
// mehr hält die weiteren gewählten Objekte. Umschalt+Klick schaltet ein Objekt, ein Rahmen im Bild wählt die ganz
// enthaltenen, Strg+A alle; gemeinsam verschieben, löschen, doppeln, ausrichten und verteilen laufen als je ein Schritt.

import (
	"math"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
)

// mehrfach ist der Zustand der Mehrfachauswahl in der Sitzung.
type mehrfach struct {
	// mehr sind die weiteren gewählten Kennungen; sie gelten nur, solange Auswahl == fuer. Setzt eine andere Stelle die
	// Auswahl (Ebenenliste, neues Objekt, Abwählen), sind die weiteren damit hinfällig, ohne dass jede Stelle sie leert.
	mehr []int
	fuer int
	// Rahmen (zugRahmen): Ecken in Fensterpunkten; dazu: mit Umschalt begonnen, die bisherige Auswahl bleibt.
	rahmenA, rahmenB document.Punkt
	dazu             bool
	// Zug über mehrere (zugMehrere): ihre Stände beim Drücken; klick ist das angeklickte Objekt, das ohne Bewegung allein
	// gewählt bleibt.
	startObjs []document.Objekt
	klick     int
}

// waehlbar meldet, ob das Objekt id gewählt sein kann: vorhanden, sichtbar, nicht gesperrt (wie Treffer).
func (s *Sitzung) waehlbar(id int) bool {
	o, ok := s.D.Objekt(id)
	return ok && s.D.ObjektSichtbar(o) && o.Ebenenwerte().Bearbeitbar()
}

// weitere sind die gewählten Objekte außer der Hauptauswahl, soweit sie noch wählbar sind.
func (s *Sitzung) weitere() []int {
	if s.Auswahl == 0 || s.mf.fuer != s.Auswahl {
		return nil
	}
	var aus []int
	for _, id := range s.mf.mehr {
		if id != s.Auswahl && s.waehlbar(id) {
			aus = append(aus, id)
		}
	}
	return aus
}

// Gewaehlt liefert alle gewählten Objekte (Hauptauswahl und weitere) in der Reihenfolge des Stapels, unten zuerst.
func (s *Sitzung) Gewaehlt() []int {
	if s.Auswahl == 0 {
		return nil
	}
	ids := append(s.weitere(), s.Auswahl)
	slices.SortFunc(ids, func(a, b int) int { return s.D.Stelle(a) - s.D.Stelle(b) })
	return ids
}

// AnzahlGewaehlt ist die Zahl der gewählten Objekte.
func (s *Sitzung) AnzahlGewaehlt() int { return len(s.Gewaehlt()) }

// IstGewaehlt meldet, ob das Objekt id gewählt ist (Hauptauswahl oder weiteres).
func (s *Sitzung) IstGewaehlt(id int) bool {
	return id != 0 && (id == s.Auswahl || slices.Contains(s.weitere(), id))
}

// setzeAuswahl macht haupt zur Hauptauswahl und weitere zu den weiteren gewählten.
func (s *Sitzung) setzeAuswahl(haupt int, weitere []int) {
	s.Auswahl, s.mf.fuer = haupt, haupt
	s.mf.mehr = s.mf.mehr[:0]
	for _, id := range weitere {
		if id != haupt && !slices.Contains(s.mf.mehr, id) {
			s.mf.mehr = append(s.mf.mehr, id)
		}
	}
}

// waehleNur wählt genau das Objekt id (0: keins).
func (s *Sitzung) waehleNur(id int) { s.setzeAuswahl(id, nil) }

// obersteVon ist das oberste Objekt aus ids im Stapel (0 ohne).
func (s *Sitzung) obersteVon(ids []int) int {
	best, stelle := 0, -1
	for _, id := range ids {
		if st := s.D.Stelle(id); st > stelle {
			best, stelle = id, st
		}
	}
	return best
}

// nachruecken: Fällt die Hauptauswahl weg (Rückgängig, Löschen von außen), wird das oberste übrige gewählte
// Hauptauswahl (ADR-0039 Punkt 1).
func (s *Sitzung) nachruecken() {
	var rest []int
	if s.mf.fuer == s.Auswahl {
		for _, id := range s.mf.mehr {
			if s.waehlbar(id) {
				rest = append(rest, id)
			}
		}
	}
	s.setzeAuswahl(s.obersteVon(rest), rest)
}

// SchalteGewaehlt nimmt das Objekt id zur Auswahl hinzu (es wird Hauptauswahl) oder heraus (Umschalt+Klick, ADR-0039
// Punkt 2). Nimmt es die Hauptauswahl heraus, wird das oberste übrige Hauptauswahl.
func (s *Sitzung) SchalteGewaehlt(id int) bool {
	if !s.waehlbar(id) || s.zug != keinZug || s.gesperrt() {
		return false
	}
	alle := s.Gewaehlt()
	if slices.Contains(alle, id) {
		alle = slices.DeleteFunc(alle, func(x int) bool { return x == id })
		s.setzeAuswahl(s.obersteVon(alle), alle)
		return true
	}
	s.setzeAuswahl(id, alle)
	return true
}

// KannAlleWaehlen meldet, ob Strg+A etwas zu wählen hat.
func (s *Sitzung) KannAlleWaehlen() bool {
	return s.zug == keinZug && !s.gesperrt() && len(s.sichtbareObjekte()) > 0
}

// WaehleAlle wählt alle sichtbaren, nicht gesperrten Objekte (Strg+A); Hauptauswahl wird das oberste. Eine
// Texteingabe wird vorher bestätigt.
func (s *Sitzung) WaehleAlle() bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	s.TextBestaetigen()
	var ids []int
	for _, o := range s.sichtbareObjekte() {
		ids = append(ids, o.Kennung())
	}
	if len(ids) == 0 {
		return false
	}
	s.setzeAuswahl(ids[len(ids)-1], ids)
	return true
}

// --- Rahmen (ADR-0039 Punkt 2) ---

// AuswahlRahmen liefert den aufgezogenen Rahmen in Fensterpunkten, solange er gezogen wird.
func (s *Sitzung) AuswahlRahmen() (document.Rechteck, bool) {
	if s.zug != zugRahmen || !s.bewegt {
		return document.Rechteck{}, false
	}
	return document.R(s.mf.rahmenA, s.mf.rahmenB), true
}

// beginneRahmen beginnt am Fensterpunkt p einen Rahmen; ohne Umschalt endet die bisherige Auswahl gleich (ein Klick ins
// Leere hebt sie auf).
func (s *Sitzung) beginneRahmen(p document.Punkt) {
	s.mf.rahmenA, s.mf.rahmenB, s.mf.dazu = p, p, s.Umschalt
	if !s.mf.dazu {
		s.waehleNur(0)
	}
	s.zug = zugRahmen
}

// beendeRahmen wählt die Objekte ganz im Rahmen (Original), mit Umschalt dazu.
func (s *Sitzung) beendeRahmen() {
	if !s.bewegt {
		return
	}
	r := document.R(s.original(s.mf.rahmenA), s.original(s.mf.rahmenB))
	var neu []int
	for _, o := range s.sichtbareObjekte() {
		g := o.Grenzen()
		if g.Min.X >= r.Min.X && g.Min.Y >= r.Min.Y && g.Max.X <= r.Max.X && g.Max.Y <= r.Max.Y {
			neu = append(neu, o.Kennung())
		}
	}
	if len(neu) == 0 {
		return
	}
	alle := neu
	if s.mf.dazu {
		alle = append(s.Gewaehlt(), neu...)
	}
	s.setzeAuswahl(s.obersteVon(neu), alle)
}

// --- Gemeinsam verschieben (ADR-0039 Punkt 3) ---

// beginneMehrere startet am angeklickten, schon gewählten Objekt id einen Zug über alle gewählten.
func (s *Sitzung) beginneMehrere(id int) bool {
	ids := s.Gewaehlt()
	if len(ids) < 2 || s.D.BeginneZugMehrere(ids) != nil {
		return false
	}
	s.mf.startObjs = s.mf.startObjs[:0]
	for _, i := range ids {
		o, _ := s.D.Objekt(i)
		s.mf.startObjs = append(s.mf.startObjs, o)
	}
	s.mf.klick = id
	s.setzeAuswahl(id, ids)
	s.zug = zugMehrere
	return true
}

// huelle ist der Rahmen um die Grenzen aller os.
func huelle(os []document.Objekt) document.Rechteck {
	r := os[0].Grenzen()
	for _, o := range os[1:] {
		g := o.Grenzen()
		r.Min.X, r.Min.Y = math.Min(r.Min.X, g.Min.X), math.Min(r.Min.Y, g.Min.Y)
		r.Max.X, r.Max.Y = math.Max(r.Max.X, g.Max.X), math.Max(r.Max.Y, g.Max.Y)
	}
	return r
}

// zieheMehrere verschiebt alle gewählten um den Weg vom Druck bis zum Fensterpunkt p: der Rahmen um alle rastet ein,
// begrenzt wird je Achse so, dass jedes im sichtbaren Bild bleibt.
func (s *Sitzung) zieheMehrere(p document.Punkt) {
	d := s.original(p).Minus(s.original(s.start))
	ohne := make([]int, len(s.mf.startObjs))
	for i, o := range s.mf.startObjs {
		ohne[i] = o.Kennung()
	}
	d, linien := s.rasteRechteck(huelle(s.mf.startObjs), d, ohne...)
	e := d
	for _, o := range s.mf.startObjs {
		b := s.begrenzeVerschieben(o, d)
		if math.Abs(b.X) < math.Abs(e.X) {
			e.X = b.X
		}
		if math.Abs(b.Y) < math.Abs(e.Y) {
			e.Y = b.Y
		}
	}
	s.linien = nurWennGleich(linien, d, e)
	neu := make([]document.Objekt, len(s.mf.startObjs))
	for i, o := range s.mf.startObjs {
		neu[i] = Verschiebe(o, e)
	}
	_ = s.D.ZugSetzeMehrere(neu)
}

// loslassenMehrere beendet den Zug über mehrere; ohne Bewegung war es ein Klick, der nur das angeklickte wählt.
func (s *Sitzung) loslassenMehrere() {
	if s.bewegt {
		_ = s.D.BeendeZug()
		return
	}
	_ = s.D.BrecheZugAb()
	s.waehleNur(s.mf.klick)
}

// --- Löschen und Doppeln mehrerer ---

// loescheMehrere entfernt alle gewählten als einen Schritt (Loeschen mit mehreren).
func (s *Sitzung) loescheMehrere() bool {
	ids := s.Gewaehlt()
	vor := s.D.Stand()
	if s.D.EntferneMehrere(ids) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.waehleNur(0)
	return true
}

// doppleMehrere doppelt alle gewählten, je direkt über dem Original, als einen Schritt; die Kopien sind danach gewählt.
func (s *Sitzung) doppleMehrere() bool {
	ids := s.Gewaehlt()
	v := DoppelVersatz / s.Abb.Zoom
	kopien := make([]document.Objekt, 0, len(ids))
	for _, id := range ids {
		o, _ := s.D.Objekt(id)
		kopien = append(kopien, Verschiebe(o, document.Pt(v, v)))
	}
	vor := s.D.Stand()
	neu, err := s.D.FuegeEinMehrere(kopien, ids)
	if err != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.setzeAuswahl(s.obersteVon(neu), neu)
	return true
}

// --- Ausrichten und Verteilen (ADR-0039 Punkt 3) ---

// Ausrichtung ist eine Kante oder Mitte, an der Ausrichten die gewählten Objekte ausrichtet.
type Ausrichtung int

const (
	AusrichtenLinks Ausrichtung = iota
	AusrichtenMitteWaagerecht
	AusrichtenRechts
	AusrichtenOben
	AusrichtenMitteSenkrecht
	AusrichtenUnten
)

// Ausrichtungen sind alle Ausrichtungen in der Reihenfolge des Menüs.
var Ausrichtungen = []Ausrichtung{AusrichtenLinks, AusrichtenMitteWaagerecht, AusrichtenRechts, AusrichtenOben,
	AusrichtenMitteSenkrecht, AusrichtenUnten}

// KannAusrichten meldet, ob Ausrichten geht: mindestens zwei gewählt, kein Zug, nicht gesperrt.
func (s *Sitzung) KannAusrichten() bool { return s.mehrereBereit(2) }

// KannVerteilen meldet, ob Verteilen geht: mindestens drei gewählt.
func (s *Sitzung) KannVerteilen() bool { return s.mehrereBereit(3) }

func (s *Sitzung) mehrereBereit(n int) bool {
	return s.zug == keinZug && !s.D.ZugLaeuft() && !s.gesperrt() && s.AnzahlGewaehlt() >= n
}

// gewaehlteObjekte liefert die gewählten Objekte in Stapelreihenfolge.
func (s *Sitzung) gewaehlteObjekte() []document.Objekt {
	var os []document.Objekt
	for _, id := range s.Gewaehlt() {
		if o, ok := s.D.Objekt(id); ok {
			os = append(os, o)
		}
	}
	return os
}

// Ausrichten richtet die gewählten Objekte an der Kante bzw. Mitte a des Rahmens um alle aus, als ein Schritt.
// Bliebe eins dabei nicht im sichtbaren Bild, geschieht nichts (Ablehnung AblehnungSichtbar).
func (s *Sitzung) Ausrichten(a Ausrichtung) bool {
	if !s.KannAusrichten() {
		return false
	}
	s.TextBestaetigen()
	os := s.gewaehlteObjekte()
	h := huelle(os)
	neu := make([]document.Objekt, len(os))
	for i, o := range os {
		g := o.Grenzen()
		var d document.Punkt
		switch a {
		case AusrichtenLinks:
			d.X = h.Min.X - g.Min.X
		case AusrichtenMitteWaagerecht:
			d.X = (h.Min.X+h.Max.X)/2 - (g.Min.X+g.Max.X)/2
		case AusrichtenRechts:
			d.X = h.Max.X - g.Max.X
		case AusrichtenOben:
			d.Y = h.Min.Y - g.Min.Y
		case AusrichtenMitteSenkrecht:
			d.Y = (h.Min.Y+h.Max.Y)/2 - (g.Min.Y+g.Max.Y)/2
		case AusrichtenUnten:
			d.Y = h.Max.Y - g.Max.Y
		}
		neu[i] = Verschiebe(o, d)
	}
	return s.ersetzeGewaehlte(neu)
}

// Verteilen verteilt die gewählten Objekte waagerecht bzw. senkrecht: die äußeren bleiben, dazwischen gleich große
// Lücken, nach Lage der linken bzw. oberen Kante sortiert. Ein Schritt.
func (s *Sitzung) Verteilen(waagerecht bool) bool {
	if !s.KannVerteilen() {
		return false
	}
	s.TextBestaetigen()
	os := s.gewaehlteObjekte()
	von := func(r document.Rechteck) float64 {
		if waagerecht {
			return r.Min.X
		}
		return r.Min.Y
	}
	bis := func(r document.Rechteck) float64 {
		if waagerecht {
			return r.Max.X
		}
		return r.Max.Y
	}
	slices.SortStableFunc(os, func(a, b document.Objekt) int {
		x, y := von(a.Grenzen()), von(b.Grenzen())
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	})
	h := huelle(os)
	summe := 0.0
	for _, o := range os {
		summe += bis(o.Grenzen()) - von(o.Grenzen())
	}
	luecke := (bis(h) - von(h) - summe) / float64(len(os)-1)
	neu := make([]document.Objekt, len(os))
	ort := von(h)
	for i, o := range os {
		g := o.Grenzen()
		v := ort - von(g)
		if waagerecht {
			neu[i] = Verschiebe(o, document.Pt(v, 0))
		} else {
			neu[i] = Verschiebe(o, document.Pt(0, v))
		}
		ort += bis(g) - von(g) + luecke
	}
	// Die äußeren bleiben genau, wo sie waren (Rundung).
	neu[0], neu[len(os)-1] = os[0], os[len(os)-1]
	return s.ersetzeGewaehlte(neu)
}

// ersetzeGewaehlte setzt neu als einen Schritt, wenn jedes Objekt im sichtbaren Bild bleibt.
func (s *Sitzung) ersetzeGewaehlte(neu []document.Objekt) bool {
	s.abgelehnt = AblehnungKeine
	for _, o := range neu {
		if x, y := s.sichtbarIn(o); !x || !y {
			s.abgelehnt = AblehnungSichtbar
			return false
		}
	}
	vor := s.D.Stand()
	if s.D.ErsetzeMehrere(neu) != nil {
		return false
	}
	s.merkeAenderung(vor)
	return true
}
