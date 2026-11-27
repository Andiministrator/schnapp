// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Stil je Werkzeug (ADR-0041): Jedes Zeichenwerkzeug hat seinen eigenen Stil; der des zuletzt gewählten steht in
// Sitzung.Stil (paarW), die der übrigen in Sitzung.stile. Text und Sprechblase teilen sich einen (paarSchluessel).
// Das Farbpaar je Werkzeug (ADR-0021 Nachtrag 11) ist ein Teil davon.

import (
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// stilWerkzeuge sind die Werkzeuge mit eigenem Stil, in der Reihenfolge von tools.ini.
var stilWerkzeuge = []Werkzeug{WerkzeugPfeil, WerkzeugKasten, WerkzeugText, WerkzeugNummer, WerkzeugLupe}

// werkzeugAbschnitt ist der Abschnitt von w in tools.ini.
func werkzeugAbschnitt(w Werkzeug) string {
	switch paarSchluessel(w) {
	case WerkzeugPfeil:
		return settings.ArtPfeil
	case WerkzeugKasten:
		return settings.ArtKasten
	case WerkzeugText:
		return settings.ArtText
	case WerkzeugNummer:
		return settings.ArtNummer
	case WerkzeugLupe:
		return settings.ArtLupe
	}
	return ""
}

// standardStilFuer ist die Voreinstellung des Werkzeugs w zum Faktor des Dokuments.
func (s *Sitzung) standardStilFuer(w Werkzeug) Stil {
	st := StilMitFaktor(s.D.KantenFaktor())
	p := standardPaar(w)
	st.Farbe, st.Fuellung = p.vorn, p.fuellung
	return st
}

// stilVon ist der gemerkte Stil des Werkzeugs w (Voreinstellung, wenn es noch keinen hat).
func (s *Sitzung) stilVon(w Werkzeug) Stil {
	w = paarSchluessel(w)
	if w == s.paarWerkzeug() {
		return s.Stil
	}
	if st, ok := s.stile[w]; ok {
		return st
	}
	return s.standardStilFuer(w)
}

// aendereStil ändert den Stil des Werkzeugs w: Sitzung.Stil, wenn w das aktive ist, sonst den gemerkten.
func (s *Sitzung) aendereStil(w Werkzeug, f func(*Stil)) {
	w = paarSchluessel(w)
	if w == s.paarWerkzeug() || !farbWerkzeug(w) {
		f(&s.Stil)
		return
	}
	st := s.stilVon(w)
	f(&st)
	if s.stile == nil {
		s.stile = map[Werkzeug]Stil{}
	}
	s.stile[w] = st
}

// stilZiel ist das Werkzeug, dessen Stil eine Änderung der rechten Leiste merkt (ADR-0041 Punkt 2): das des Objekts,
// an dem geändert wird (Texteingabe, sonst Auswahl), sonst das aktive.
func (s *Sitzung) stilZiel() Werkzeug {
	if o, ok := s.D.Objekt(s.stilObjektID()); ok {
		if w, ok := objektWerkzeug(o); ok {
			return w
		}
	}
	return s.paarWerkzeug()
}

// aendereZielStil ändert den Stil des Werkzeugs, das die Änderung merkt (stilZiel).
func (s *Sitzung) aendereZielStil(f func(*Stil)) { s.aendereStil(s.stilZiel(), f) }

// wechsleStil legt den Stil des Werkzeugs w in Stil (beim Wechsel auf ein Werkzeug mit Stil); der bisherige wird
// gemerkt.
func (s *Sitzung) wechsleStil(w Werkzeug) {
	w = paarSchluessel(w)
	alt := s.paarWerkzeug()
	if !farbWerkzeug(w) || w == alt {
		return
	}
	if s.stile == nil {
		s.stile = map[Werkzeug]Stil{}
	}
	s.stile[alt] = s.Stil
	s.Stil = s.stilVon(w)
	delete(s.stile, w)
	s.paarW = w
	s.hsvGemerkt = false
}

// stilMal liefert st mit allen Größen mal f (Strich, Schrift, Nummer, Eckradius, Schatten).
func stilMal(st Stil, f float64) Stil {
	st.Breite *= f
	st.Schriftgrad *= f
	st.Radius *= f
	st.Eckradius *= f
	st.Schatten.Laenge *= f
	st.Schatten.Weich *= f
	return st
}

// WerkzeugStile sind die Stile der Werkzeuge, die einen haben (benutzt oder geladen), auf Faktor 1 für tools.ini.
func (s *Sitzung) WerkzeugStile() []settings.WerkzeugStil {
	var r []settings.WerkzeugStil
	for _, w := range stilWerkzeuge {
		st, ok := s.stile[w]
		if w == s.paarWerkzeug() {
			st, ok = s.Stil, true
		}
		if !ok {
			continue
		}
		st = stilMal(st, 1/s.faktor())
		r = append(r, settings.WerkzeugStil{
			Stil: settings.Stil{Art: werkzeugAbschnitt(w), Farbe: st.Farbe, Fuellung: st.Fuellung, Breite: st.Breite,
				Schriftgrad: st.Schriftgrad, Radius: st.Radius, Schrift: st.Schrift, Fett: st.Fett, Kursiv: st.Kursiv,
				Unterstr: st.Unterstrichen, Durchgestr: st.Durchgestrichen, Ausrichtung: st.Ausrichtung,
				Schatten: st.Schatten},
			Strichart: st.Strichart, KopfEnde: st.KopfEnde, KopfAnfang: st.KopfAnfang.Art(), Eckradius: st.Eckradius,
		})
	}
	return r
}

// SetzeWerkzeugStile übernimmt gemerkte Stile (Faktor 1, aus tools.ini oder einer vorigen Sitzung): Schlüssel, die
// nicht gesetzt sind, behalten die Voreinstellung des Werkzeugs; ein Wert, der mal dem Faktor die Grenzen sprengt,
// ebenso.
func (s *Sitzung) SetzeWerkzeugStile(liste []settings.WerkzeugStil) {
	f := s.faktor()
	for _, e := range liste {
		var w Werkzeug
		ok := false
		for _, k := range stilWerkzeuge {
			if werkzeugAbschnitt(k) == e.Art {
				w, ok = k, true
			}
		}
		if !ok {
			continue
		}
		gilt := func(k string) bool { return e.Gesetzt == nil || e.Gesetzt[k] }
		st := s.standardStilFuer(w)
		if gilt("color") {
			st.Farbe = e.Farbe
		}
		if gilt("fill") && fuellWerkzeug(w) {
			st.Fuellung = e.Fuellung
		}
		if gilt("width") && e.Breite > 0 && e.Breite*f <= document.MaxGroesse {
			st.Breite = e.Breite * f
		}
		if gilt("size") && e.Schriftgrad > 0 && e.Schriftgrad*f <= document.MaxSchrift {
			st.Schriftgrad = e.Schriftgrad * f
		}
		if gilt("radius") && e.Radius > 0 && e.Radius*f <= document.MaxSchrift {
			st.Radius = e.Radius * f
		}
		if gilt("corner_radius") && e.Eckradius >= 0 && e.Eckradius*f <= document.MaxGroesse {
			st.Eckradius = e.Eckradius * f
		}
		if gilt("font") {
			st.Schrift = e.Schrift
		}
		if gilt("bold") {
			st.Fett = e.Fett
		}
		if gilt("italic") {
			st.Kursiv = e.Kursiv
		}
		if gilt("underline") {
			st.Unterstrichen = e.Unterstr
		}
		if gilt("strike") {
			st.Durchgestrichen = e.Durchgestr
		}
		if gilt("align") {
			st.Ausrichtung = e.Ausrichtung
		}
		if gilt("line_style") && int(e.Strichart) < len(document.Stricharten) {
			st.Strichart = e.Strichart
		}
		if gilt("end_head") {
			st.KopfEnde = e.KopfEnde
		}
		if gilt("start_head") {
			st.KopfAnfang = document.AnfangMit(e.KopfAnfang)
		}
		if gilt("shadow") {
			an := st.Schatten.An
			sch := e.Schatten
			sch.Laenge *= f
			sch.Weich *= f
			st.Schatten = gueltigerSchatten(sch)
			st.Schatten.An = an
		}
		if gilt("shadow_on") {
			st.Schatten.An = e.Schatten.An
		}
		if paarSchluessel(w) == s.paarWerkzeug() {
			s.Stil = st
			s.hsvGemerkt = false
			continue
		}
		if s.stile == nil {
			s.stile = map[Werkzeug]Stil{}
		}
		s.stile[paarSchluessel(w)] = st
	}
}
