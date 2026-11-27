// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Stil für alle gewählten (ADR-0039 Nachtrag 4): Sind mehrere Objekte gewählt, wirkt eine Änderung der rechten Leiste
// auf alle gewählten, die die Eigenschaft tragen, als ein Rückgängig-Schritt; jedes geänderte Objekt gibt sie an sein
// Werkzeug weiter (ADR-0041 Punkt 2). Die Leiste zeigt die Werte der Hauptauswahl bzw. des obersten Trägers und
// erkennt gemischte Werte (Gemischt, ZweiteFarbe).

import (
	"image/color"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Eigenschaft ist ein Stilwert der rechten Leiste, den mehrere gewählte Objekte verschieden tragen können.
type Eigenschaft int

const (
	EigFarbe         Eigenschaft = iota // Vordergrund (jede Art)
	EigFuellung                         // Füllung (Kasten, Text)
	EigSchattenFarbe                    // Farbe des Schattens (Objekte mit Schatten)
	EigBreite                           // Strichstärke (Pfeil, Kasten, Lupe)
	EigStrichart                        // Strichart (Pfeil, Kasten, Lupe)
	EigKopfAnfang                       // Spitze am Anfang (Pfeil)
	EigKopfEnde                         // Spitze am Ende (Pfeil)
	EigSchatten                         // Schatten an oder aus (jede Art)
	EigSchrift                          // Schriftfamilie (Text, Sprechblase)
	EigGrad                             // Schriftgrad (Text, Sprechblase)
)

// stilMehrere meldet, ob eine Änderung der Leiste mehreren gewählten gilt: mehr als eines gewählt, keine Texteingabe.
func (s *Sitzung) stilMehrere() bool { return s.zug != zugText && s.AnzahlGewaehlt() > 1 }

// eigenschaftWert ist der Wert der Eigenschaft e an o (false: o trägt sie nicht). Die Werte sind vergleichbar.
func eigenschaftWert(o document.Objekt, e Eigenschaft) (any, bool) {
	switch e {
	case EigFarbe:
		return objektFarbe(o, FeldVordergrund)
	case EigFuellung:
		return objektFarbe(o, FeldFuellung)
	case EigSchattenFarbe:
		if sch := o.Ebenenwerte().Schatten; sch.An {
			return sch.Farbe, true
		}
		return nil, false
	case EigSchatten:
		return o.Ebenenwerte().Schatten.An, true
	}
	switch v := o.(type) {
	case document.Pfeil:
		switch e {
		case EigBreite:
			return v.Strich.Breite, true
		case EigStrichart:
			return v.Strich.Art, true
		case EigKopfAnfang:
			return v.KopfAnfang(), true
		case EigKopfEnde:
			return v.Ende, true
		}
	case document.Kasten:
		switch e {
		case EigBreite:
			return v.Strich.Breite, true
		case EigStrichart:
			return v.Strich.Art, true
		}
	case document.Lupe:
		switch e {
		case EigBreite:
			return v.Strich.Breite, true
		case EigStrichart:
			return v.Strich.Art, true
		}
	case document.Text:
		switch e {
		case EigSchrift:
			return v.Schrift, true
		case EigGrad:
			return v.Schriftgrad, true
		}
	}
	return nil, false
}

// feldEigenschaft ist die Eigenschaft des Farbfelds f.
func feldEigenschaft(f Farbfeld) Eigenschaft {
	switch f {
	case FeldFuellung:
		return EigFuellung
	case FeldSchatten:
		return EigSchattenFarbe
	}
	return EigFarbe
}

// traegerReihe sind die gewählten Objekte in der Reihenfolge, in der die Leiste ihre Werte sucht: die Hauptauswahl,
// dann die übrigen von oben nach unten.
func (s *Sitzung) traegerReihe() []document.Objekt {
	os := s.gewaehlteObjekte()
	r := make([]document.Objekt, 0, len(os))
	for _, o := range os {
		if o.Kennung() == s.Auswahl {
			r = append(r, o)
		}
	}
	for i := len(os) - 1; i >= 0; i-- {
		if os[i].Kennung() != s.Auswahl {
			r = append(r, os[i])
		}
	}
	return r
}

// traeger ist das Objekt, dessen Wert der Eigenschaft e die Leiste bei mehreren gewählten zeigt: die Hauptauswahl,
// wenn sie e trägt, sonst das oberste gewählte, das e trägt.
func (s *Sitzung) traeger(e Eigenschaft) (document.Objekt, bool) {
	for _, o := range s.traegerReihe() {
		if _, ok := eigenschaftWert(o, e); ok {
			return o, true
		}
	}
	return nil, false
}

// Gemischt meldet, ob die gewählten Objekte, die e tragen, darin verschiedene Werte haben (nur bei mehreren gewählten,
// außerhalb der Texteingabe).
func (s *Sitzung) Gemischt(e Eigenschaft) bool {
	_, ok := s.zweiterWert(e)
	return ok
}

// zweiterWert ist der erste Wert von e in der traegerReihe, der vom gezeigten (dem ersten) abweicht.
func (s *Sitzung) zweiterWert(e Eigenschaft) (any, bool) {
	if !s.stilMehrere() {
		return nil, false
	}
	var erst any
	hat := false
	for _, o := range s.traegerReihe() {
		v, ok := eigenschaftWert(o, e)
		switch {
		case !ok:
		case !hat:
			erst, hat = v, true
		case v != erst:
			return v, true
		}
	}
	return nil, false
}

// ZweiteFarbe ist bei gemischten Farben im Feld f die nächste andere Farbe nach der gezeigten (das Farbfeld wird
// diagonal geteilt, ADR-0039 Nachtrag 4 Punkt 4); false, wenn alle gleich sind.
func (s *Sitzung) ZweiteFarbe(f Farbfeld) (color.NRGBA, bool) {
	v, ok := s.zweiterWert(feldEigenschaft(f))
	if !ok {
		return color.NRGBA{}, false
	}
	c, ok := v.(color.NRGBA)
	return c, ok
}

// traegtFeld meldet, ob o das Farbfeld f trägt; der Schatten nur, wenn er an ist.
func traegtFeld(o document.Objekt, f Farbfeld) bool {
	_, ok := eigenschaftWert(o, feldEigenschaft(f))
	return ok
}

// traegtStrich meldet, ob o einen Strich hat (Pfeil, Kasten, Lupe).
func traegtStrich(o document.Objekt) bool {
	_, ok := eigenschaftWert(o, EigBreite)
	return ok
}

// merkeWerkzeuge ruft merke einmal je Werkzeug der Objekte os (Text und Sprechblase sind eines).
func merkeWerkzeuge(os []document.Objekt, merke func(Werkzeug)) {
	gesehen := map[Werkzeug]bool{}
	for _, o := range os {
		w, ok := objektWerkzeug(o)
		if !ok || gesehen[paarSchluessel(w)] {
			continue
		}
		gesehen[paarSchluessel(w)] = true
		merke(w)
	}
}

// werkzeugeVon sind die Werkzeuge der Objekte os, je eines.
func werkzeugeVon(os []document.Objekt) []Werkzeug {
	var ws []Werkzeug
	merkeWerkzeuge(os, func(w Werkzeug) { ws = append(ws, w) })
	return ws
}

// setzeMehrere ändert jedes gewählte Objekt, das aendere annimmt (zweiter Wert true), als einen Rückgängig-Schritt;
// merke (darf nil sein) bekommt einmal je Werkzeug der angenommenen Objekte die Gelegenheit, die Änderung im Stil des
// Werkzeugs zu merken. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) setzeMehrere(aendere func(document.Objekt) (document.Objekt, bool), merke func(Werkzeug)) bool {
	var alt, neu []document.Objekt
	for _, o := range s.gewaehlteObjekte() {
		if n, ok := aendere(o); ok {
			alt, neu = append(alt, o), append(neu, n)
		}
	}
	if merke != nil {
		merkeWerkzeuge(alt, merke)
	}
	if len(neu) == 0 {
		return false
	}
	vor := s.D.Stand()
	if s.D.ErsetzeMehrere(neu) != nil {
		return false
	}
	s.merkeAenderung(vor)
	return s.D.Stand() != vor
}

// --- Zug über mehrere (Regler, Farbfläche) ---

// beginneMehrZug beginnt einen Zug des Dokuments an allen gewählten, die traegt annimmt, und merkt ihre Stände und die
// Stile ihrer Werkzeuge (für den Abbruch). Liefert, ob ein Zug läuft.
func (s *Sitzung) beginneMehrZug(traegt func(document.Objekt) bool) bool {
	var os []document.Objekt
	var ids []int
	for _, o := range s.gewaehlteObjekte() {
		if traegt(o) {
			os, ids = append(os, o), append(ids, o.Kennung())
		}
	}
	s.merkeZugStil(werkzeugeVon(os)...)
	if len(ids) == 0 || s.D.BeginneZugMehrere(ids) != nil {
		s.mehrZug = nil
		return false
	}
	s.mehrZug = os
	return true
}

// mehrZugSetze setzt während eines Zugs über mehrere den Stand f(Stand beim Beginn) jedes beteiligten Objekts und
// merkt die Änderung über merke im Stil ihrer Werkzeuge.
func (s *Sitzung) mehrZugSetze(f func(document.Objekt) document.Objekt, merke func(Werkzeug)) {
	merkeWerkzeuge(s.mehrZug, merke)
	neu := make([]document.Objekt, len(s.mehrZug))
	for i, o := range s.mehrZug {
		neu[i] = f(o)
	}
	_ = s.D.ZugSetzeMehrere(neu)
}

// --- Schatten ---

// schattenTeil ist eine Menge von Bestandteilen des Schattens, die eine Änderung an der Leiste setzt.
type schattenTeil uint8

const (
	teilAn schattenTeil = 1 << iota
	teilWinkel
	teilLaenge
	teilWeich
	teilFarbe
)

// geaenderteTeile sind die Bestandteile, in denen sich neu von vorher unterscheidet.
func geaenderteTeile(vorher, neu document.Schatten) schattenTeil {
	var t schattenTeil
	if neu.An != vorher.An {
		t |= teilAn
	}
	if neu.Winkel != vorher.Winkel {
		t |= teilWinkel
	}
	if neu.Laenge != vorher.Laenge {
		t |= teilLaenge
	}
	if neu.Weich != vorher.Weich {
		t |= teilWeich
	}
	if neu.Farbe != vorher.Farbe {
		t |= teilFarbe
	}
	return t
}

// schattenUebertrag setzt die Bestandteile teile von neu auf den Schatten cur (ADR-0039 Nachtrag 4 Punkt 3): An/Aus
// (beim Einschalten ohne eigene Werte die aus ersatz), dann jeden Wert, wenn der Schatten an ist. amObjekt: Ein
// ausgeschalteter Schatten am Objekt ist der Nullwert; das Werkzeug behält seine Werte auch aus.
func schattenUebertrag(cur, neu document.Schatten, teile schattenTeil, ersatz func() document.Schatten, amObjekt bool) document.Schatten {
	if teile&teilAn != 0 {
		switch {
		case neu.An && !cur.An:
			if cur.Laenge == 0 && cur.Weich == 0 && cur.Farbe == (color.NRGBA{}) {
				cur = ersatz()
			}
			cur.An = true
		case !neu.An && amObjekt:
			cur = document.Schatten{}
		case !neu.An:
			cur.An = false
		}
	}
	if !cur.An && amObjekt {
		return cur
	}
	if teile&teilWinkel != 0 {
		cur.Winkel = neu.Winkel
	}
	if teile&teilLaenge != 0 {
		cur.Laenge = neu.Laenge
	}
	if teile&teilWeich != 0 {
		cur.Weich = neu.Weich
	}
	if teile&teilFarbe != 0 {
		cur.Farbe = neu.Farbe
	}
	return gueltigerSchatten(cur)
}

// schattenErsatz sind die Werte, die ein Schatten ohne eigene Werte beim Einschalten bekommt: die des Werkzeugs w,
// sonst die Voreinstellung zum Faktor des Bilds (wie SchalteSchatten).
func (s *Sitzung) schattenErsatz(w Werkzeug) func() document.Schatten {
	return func() document.Schatten {
		sch := s.stilVon(w).Schatten
		if sch.Laenge == 0 && sch.Weich == 0 && sch.Farbe == (color.NRGBA{}) {
			sch = document.StandardSchatten(s.faktor())
		}
		return sch
	}
}

// objektMitSchatten ist o mit den übertragenen Bestandteilen; false, wenn sich o dadurch nicht ändert (F-943: dann
// merkt sein Werkzeug auch nichts).
func (s *Sitzung) objektMitSchatten(o document.Objekt, neu document.Schatten, teile schattenTeil) (document.Objekt, bool) {
	w, _ := objektWerkzeug(o)
	n := mitSchatten(o, schattenUebertrag(o.Ebenenwerte().Schatten, neu, teile, s.schattenErsatz(w), true))
	return n, !document.Gleich(n, o)
}

// merkeSchatten überträgt die Bestandteile auf den Schatten des Werkzeugs w, ausgehend von basis.
func (s *Sitzung) merkeSchatten(w Werkzeug, basis Stil, neu document.Schatten, teile schattenTeil) {
	s.aendereStil(w, func(st *Stil) {
		*st = basis
		st.Schatten = schattenUebertrag(basis.Schatten, neu, teile, func() document.Schatten {
			return document.StandardSchatten(s.faktor())
		}, false)
	})
}

// setzeSchattenMehrere überträgt die Änderung von vorher nach neu auf alle gewählten, die sich dadurch ändern, und
// deren Werkzeuge.
func (s *Sitzung) setzeSchattenMehrere(vorher, neu document.Schatten) bool {
	teile := geaenderteTeile(vorher, neu)
	return s.setzeMehrere(func(o document.Objekt) (document.Objekt, bool) {
		return s.objektMitSchatten(o, neu, teile)
	}, func(w Werkzeug) { s.merkeSchatten(w, s.stilVon(w), neu, teile) })
}

// schalteSchattenMehrere schaltet den Schatten aller gewählten ein, wenn mindestens einer aus ist, sonst alle aus.
func (s *Sitzung) schalteSchattenMehrere() bool {
	an := false
	for _, o := range s.gewaehlteObjekte() {
		if !o.Ebenenwerte().Schatten.An {
			an = true
		}
	}
	vorher := s.Schatten()
	vorher.An = !an
	neu := vorher
	neu.An = an
	if !an && s.feld == FeldSchatten {
		s.feld, s.hsvGemerkt = FeldVordergrund, false
	}
	return s.setzeSchattenMehrere(vorher, neu)
}

// hatSchatten meldet, ob o einen eingeschalteten Schatten trägt (Teilnehmer eines Schattenzugs über mehrere).
func hatSchatten(o document.Objekt) bool { return o.Ebenenwerte().Schatten.An }
