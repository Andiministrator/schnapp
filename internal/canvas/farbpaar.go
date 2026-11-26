// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Zwei Farben je Beschriftung (ADR-0021 Nachtrag 11): Vordergrund (Linie von Pfeil und Kasten, Schrift des Texts,
// Kreis der Nummer) und Füllung (Fläche des Kastens, Schild des Texts). Jedes Zeichenwerkzeug mit Farbe merkt sich
// sein eigenes Paar, solange die Sitzung läuft; das Paar des zuletzt gewählten solchen Werkzeugs steht in Stil.Farbe
// und Stil.Fuellung. Farbwähler, Farbfelder, Hex, Deckkraft und Pipette wirken auf das aktive Feld.

import (
	"image/color"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Farbfeld ist eine der beiden Farben einer Beschriftung.
type Farbfeld int

const (
	FeldVordergrund Farbfeld = iota // Linie, Schrift, Kreis der Nummer
	FeldFuellung                    // Fläche des Kastens, Schild des Texts; Deckkraft 0 ist „keine“
	FeldSchatten                    // Schatten der Ebene (ADR-0021 Nachtrag 26 Punkt 7), nur wenn er an ist
)

type farbpaar struct{ vorn, fuellung color.NRGBA }

// standardPaar ist die Voreinstellung je Werkzeug: Rot, beim Text weiße Schrift auf rotem Schild (ein neuer Text
// sieht aus wie vor Nachtrag 11), der Kasten ohne Füllung.
func standardPaar(w Werkzeug) farbpaar {
	w = paarSchluessel(w)
	rot := Beschriftungsfarben[0]
	if w == WerkzeugText {
		return farbpaar{vorn: Weiss, fuellung: rot}
	}
	return farbpaar{vorn: rot}
}

// StandardPaar ist die Voreinstellung von Vordergrund und Füllung des Werkzeugs w, wie ein neuer Editor sie beginnt
// (für die Kommandozeile, ADR-0030 Nachtrag 10).
func StandardPaar(w Werkzeug) (vorn, fuellung color.NRGBA) {
	p := standardPaar(w)
	return p.vorn, p.fuellung
}

// fuellWerkzeug meldet, ob ein Werkzeug eine Füllung zeichnet (Kasten, Text).
func fuellWerkzeug(w Werkzeug) bool {
	return w == WerkzeugKasten || w == WerkzeugText || w == WerkzeugSprechblase
}

// hatFuellung meldet, ob o eine Füllung hat (Kasten, Text).
func hatFuellung(o document.Objekt) bool {
	switch o.(type) {
	case document.Kasten, document.Text:
		return true
	}
	return false
}

// objektFarbe ist die Farbe des Felds f an o (false: o hat dieses Feld nicht).
func objektFarbe(o document.Objekt, f Farbfeld) (color.NRGBA, bool) {
	if f == FeldSchatten {
		return o.Ebenenwerte().Schatten.Farbe, true
	}
	switch v := o.(type) {
	case document.Pfeil:
		if f == FeldVordergrund {
			return v.Strich.Farbe, true
		}
	case document.Kasten:
		if f == FeldFuellung {
			return v.Fuellung, true
		}
		return v.Strich.Farbe, true
	case document.Text:
		if f == FeldFuellung {
			return v.Hintergrund, true
		}
		return v.Farbe, true
	case document.Nummer:
		if f == FeldVordergrund {
			return v.Farbe, true
		}
	case document.Lupe: // nur Vordergrund: Rahmen und Linie (ADR-0033 Punkt 8)
		if f == FeldVordergrund {
			return v.Strich.Farbe, true
		}
	}
	return color.NRGBA{}, false
}

// UmgefaerbtIn liefert o mit der Farbe c im Feld f; hat o dieses Feld nicht, bleibt o, wie es ist.
func UmgefaerbtIn(o document.Objekt, f Farbfeld, c color.NRGBA) document.Objekt {
	if f == FeldSchatten {
		sch := o.Ebenenwerte().Schatten
		sch.Farbe = c
		return mitSchatten(o, sch)
	}
	switch v := o.(type) {
	case document.Pfeil:
		if f == FeldVordergrund {
			v.Strich.Farbe = c
		}
		return v
	case document.Kasten:
		if f == FeldFuellung {
			v.Fuellung = c
		} else {
			v.Strich.Farbe = c
		}
		return v
	case document.Text:
		if f == FeldFuellung {
			v.Hintergrund = c
		} else {
			v.Farbe = c
		}
		return v
	case document.Nummer:
		if f == FeldVordergrund {
			v.Farbe = c
		}
		return v
	case document.Lupe:
		if f == FeldVordergrund {
			v.Strich.Farbe = c
		}
		return v
	}
	return o
}

// Getauscht liefert o mit vertauschtem Vordergrund und Füllung (Kasten, Text); sonst o selbst.
func Getauscht(o document.Objekt) document.Objekt {
	switch v := o.(type) {
	case document.Kasten:
		v.Strich.Farbe, v.Fuellung = v.Fuellung, v.Strich.Farbe
		return v
	case document.Text:
		v.Farbe, v.Hintergrund = v.Hintergrund, v.Farbe
		return v
	}
	return o
}

// paarWerkzeug ist das Werkzeug, dessen Paar in Stil steht (Voreinstellung Pfeil).
func (s *Sitzung) paarWerkzeug() Werkzeug {
	if farbWerkzeug(s.paarW) {
		return s.paarW
	}
	return WerkzeugPfeil
}

// paar liefert das gemerkte Paar des Werkzeugs w.
func (s *Sitzung) paar(w Werkzeug) farbpaar {
	w = paarSchluessel(w)
	if w == s.paarWerkzeug() {
		return farbpaar{vorn: s.Stil.Farbe, fuellung: s.Stil.Fuellung}
	}
	if p, ok := s.paare[w]; ok {
		return p
	}
	return standardPaar(w)
}

// wechslePaar legt das Paar des Werkzeugs w in Stil (beim Wechsel auf ein Werkzeug mit Farbe).
func (s *Sitzung) wechslePaar(w Werkzeug) {
	w = paarSchluessel(w)
	alt := s.paarWerkzeug()
	if !farbWerkzeug(w) || w == alt {
		return
	}
	if s.paare == nil {
		s.paare = map[Werkzeug]farbpaar{}
	}
	s.paare[alt] = farbpaar{vorn: s.Stil.Farbe, fuellung: s.Stil.Fuellung}
	p := s.paar(w)
	s.paarW = w
	s.Stil.Farbe, s.Stil.Fuellung = p.vorn, p.fuellung
	s.hsvGemerkt = false
}

// merkePaarFarbe setzt die Farbe c im Feld f des Paars von w (ohne Füllung beim Pfeil und der Nummer).
func (s *Sitzung) merkePaarFarbe(w Werkzeug, f Farbfeld, c color.NRGBA) {
	if f == FeldSchatten { // ein Schatten für alle Werkzeuge
		s.Stil.Schatten.Farbe = c
		return
	}
	w = paarSchluessel(w)
	if f == FeldFuellung && !fuellWerkzeug(w) {
		return
	}
	if w == s.paarWerkzeug() {
		if f == FeldFuellung {
			s.Stil.Fuellung = c
		} else {
			s.Stil.Farbe = c
		}
		return
	}
	p := s.paar(w)
	if f == FeldFuellung {
		p.fuellung = c
	} else {
		p.vorn = c
	}
	if s.paare == nil {
		s.paare = map[Werkzeug]farbpaar{}
	}
	s.paare[w] = p
}

// farbObjekt ist das Objekt, auf das die Farbwahl wirkt: der Text in der Eingabe, sonst das gewählte Objekt mit Farbe.
func (s *Sitzung) farbObjekt() (document.Objekt, bool) {
	id := s.Auswahl
	if s.zug == zugText {
		id = s.textID
	}
	o, ok := s.D.Objekt(id)
	if !ok {
		return nil, false
	}
	if _, ok := objektFarbe(o, FeldVordergrund); !ok {
		return nil, false
	}
	return o, true
}

// zielWerkzeug ist das Werkzeug, dessen Paar eine neue Farbe übernimmt: das des Objekts, auf das die Farbwahl wirkt
// (ein umgefärbter Kasten färbt auch die nächsten Kästen), sonst das des Paars in Stil.
func (s *Sitzung) zielWerkzeug() Werkzeug {
	if o, ok := s.farbObjekt(); ok {
		if w, ok := objektWerkzeug(o); ok {
			return w
		}
	}
	return s.paarWerkzeug()
}

// Fuellbar meldet, ob das, worauf die Farbwahl wirkt, eine Füllung hat: das Objekt, sonst das gewählte Werkzeug.
// Ein Werkzeug ohne Farbe (Auswählen ohne Objekt, Schwärzen, Zuschneiden, Stauchen) hat keine Füllung, obwohl Stil
// das Paar des zuletzt benutzten Farbwerkzeugs trägt: Tauschen und „Keine“ wirkten sonst unsichtbar darauf
// (ADR-0021 Nachtrag 11 Punkt 9, Kritik B-4). Einzige Quelle für Feld Füllung, „Keine“ und Tauschen.
func (s *Sitzung) Fuellbar() bool {
	if o, ok := s.farbObjekt(); ok {
		return hatFuellung(o)
	}
	return fuellWerkzeug(s.Werkzeug)
}

// AktivesFeld ist das Feld, auf das die Farbwahl wirkt; ohne Füllung immer der Vordergrund.
func (s *Sitzung) AktivesFeld() Farbfeld {
	if s.feld == FeldFuellung && s.Fuellbar() {
		return FeldFuellung
	}
	if s.feld == FeldSchatten && s.Schatten().An && s.schattenZiel() == s.feldSchattenID {
		return FeldSchatten
	}
	return FeldVordergrund
}

// SetzeFeld macht f zum aktiven Feld (Klick auf eines der beiden Felder). Die Füllung geht nur, wenn es eine gibt;
// während eines Farbzugs bleibt das Feld. Liefert, ob f jetzt aktiv ist.
func (s *Sitzung) SetzeFeld(f Farbfeld) bool {
	if s.farbZug || (f == FeldFuellung && !s.Fuellbar()) || (f == FeldSchatten && !s.Schatten().An) {
		return false
	}
	if f == FeldSchatten {
		s.feldSchattenID = s.schattenZiel()
	}
	if s.feld != f {
		s.feld, s.hsvGemerkt = f, false
	}
	return true
}

// Farbe ist die Farbe des Felds f: die des Objekts, auf das die Farbwahl wirkt, sonst die des Paars in Stil.
func (s *Sitzung) Farbe(f Farbfeld) color.NRGBA {
	if o, ok := s.farbObjekt(); ok {
		if c, ok := objektFarbe(o, f); ok {
			return c
		}
	}
	switch f {
	case FeldFuellung:
		return s.Stil.Fuellung
	case FeldSchatten:
		return s.Stil.Schatten.Farbe
	}
	return s.Stil.Farbe
}

// KeineFuellung setzt die aktive Füllung auf „keine“ (Deckkraft 0, Farbton bleibt). Ohne aktive Füllung geschieht
// nichts. Liefert, ob die Füllung jetzt keine ist.
func (s *Sitzung) KeineFuellung() bool {
	if s.Zieht() || s.gesperrt() || s.AktivesFeld() != FeldFuellung {
		return false
	}
	c := s.Farbe(FeldFuellung)
	c.A = 0
	s.SetzeFarbe(c)
	return true
}

// KannTauschen meldet, ob Farben tauschen wirkt (Knopf, Menü): nur mit Füllung und wenn Bearbeiten nicht ruht.
func (s *Sitzung) KannTauschen() bool { return !s.Zieht() && !s.gesperrt() && s.Fuellbar() }

// TauscheFarben tauscht Vordergrund und Füllung (ADR-0021 Nachtrag 11 Punkt 5): am Objekt, auf das die Farbwahl
// wirkt (ein Rückgängig-Schritt; in der Texteingabe gehört der Tausch zur Eingabe), sonst im Paar des Werkzeugs.
// Liefert, ob getauscht wurde.
func (s *Sitzung) TauscheFarben() bool {
	if !s.KannTauschen() {
		return false
	}
	s.hsvGemerkt = false
	o, ok := s.farbObjekt()
	if !ok {
		s.Stil.Farbe, s.Stil.Fuellung = s.Stil.Fuellung, s.Stil.Farbe
		return true
	}
	neu := Getauscht(o)
	if s.zug == zugText {
		return s.D.ZugSetze(neu) == nil
	}
	if document.Gleich(neu, o) {
		return true // gleiche Farben: nichts zu tauschen, kein Schritt
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}
