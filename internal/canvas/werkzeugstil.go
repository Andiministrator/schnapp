// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Stil je Werkzeug (ADR-0041): Jedes Zeichenwerkzeug hat seinen eigenen Stil; der des zuletzt gewählten steht in
// Sitzung.Stil (paarW), die der übrigen in Sitzung.stile. Text und Sprechblase teilen sich einen (paarSchluessel).
// Das Farbpaar je Werkzeug (ADR-0021 Nachtrag 11) ist ein Teil davon.

import (
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// stilWerkzeuge sind die Werkzeuge mit eigenem Stil, in der Reihenfolge von tools.ini.
var stilWerkzeuge = []Werkzeug{WerkzeugPfeil, WerkzeugKasten, WerkzeugText, WerkzeugNummer, WerkzeugLupe, WerkzeugEllipse,
	WerkzeugTextmarker, WerkzeugFreihand}

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
	case WerkzeugEllipse:
		return settings.ArtEllipse
	case WerkzeugTextmarker:
		return settings.ArtTextmarker
	case WerkzeugFreihand:
		return settings.ArtFreihand
	}
	return ""
}

// stileAngleichen rechnet alle Stile der Sitzung (Stil, stile, der Merker eines laufenden Zugs) um, wenn sich der
// Faktor des Dokuments seit dem letzten Abgleich geändert hat (Zuschnitt, Stauchen, Rückgängig; ADR-0041 Nachtrag 2,
// F-934). Damit gilt in der Sitzung immer ein Faktor: Schnellstile, Regler und tools.ini rechnen gleich, und die
// Strichstärke eines Werkzeugs folgt dem Bild wie die Voreinstellung (F-927 bleibt behoben: tools.ini ändert sich
// durch einen Zuschnitt nicht).
func (s *Sitzung) stileAngleichen() {
	f := s.dokFaktor()
	if s.stilFaktor < 1 {
		s.stilFaktor = f
		return
	}
	if f == s.stilFaktor {
		return
	}
	k := f / s.stilFaktor
	s.Stil = stilMal(s.Stil, k)
	for w, st := range s.stile {
		s.stile[w] = stilMal(st, k)
	}
	for w, st := range s.zugStil.vor {
		s.zugStil.vor[w] = stilMal(st, k)
	}
	s.stilFaktor = f
}

// standardStilFuer ist die Voreinstellung des Werkzeugs w zum Faktor des Dokuments.
func (s *Sitzung) standardStilFuer(w Werkzeug) Stil {
	st := StilMitFaktor(int(s.faktor()))
	p := standardPaar(w)
	st.Farbe, st.Fuellung = p.vorn, p.fuellung
	if paarSchluessel(w) == WerkzeugTextmarker { // ADR-0048 Punkt 5
		st.Breite = TextmarkerBreite * st.Breite / StandardStil().Breite
	}
	return st
}

// stilVon ist der gemerkte Stil des Werkzeugs w (Voreinstellung, wenn es noch keinen hat).
func (s *Sitzung) stilVon(w Werkzeug) Stil {
	s.stileAngleichen()
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
	s.stileAngleichen()
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
	s.stileAngleichen()
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

// gerundet rundet die Größen auf Tausendstel (F-930): Hin und zurück über einen Faktor 3, 5 oder 7 bliebe sonst ein
// Rest in der letzten Stelle, und tools.ini würde ohne Änderung neu geschrieben.
func gerundet(st Stil) Stil {
	r := func(v float64) float64 { return math.Round(v*1000) / 1000 }
	st.Breite, st.Schriftgrad, st.Radius, st.Eckradius = r(st.Breite), r(st.Schriftgrad), r(st.Radius), r(st.Eckradius)
	st.Schatten.Laenge, st.Schatten.Weich = r(st.Schatten.Laenge), r(st.Schatten.Weich)
	return st
}

// WerkzeugStile sind die Stile der Werkzeuge, die einen haben (benutzt oder geladen), auf Faktor 1 für tools.ini.
func (s *Sitzung) WerkzeugStile() []settings.WerkzeugStil {
	s.stileAngleichen()
	var r []settings.WerkzeugStil
	for _, w := range stilWerkzeuge {
		st, ok := s.stile[w]
		if w == s.paarWerkzeug() {
			st, ok = s.Stil, true
		}
		if !ok {
			continue
		}
		st = gerundet(stilMal(st, 1/s.faktor()))
		r = append(r, settings.WerkzeugStil{
			Stil: settings.Stil{Art: werkzeugAbschnitt(w), Farbe: st.Farbe, Fuellung: st.Fuellung, Breite: st.Breite,
				Schriftgrad: st.Schriftgrad, Radius: st.Radius, Schrift: st.Schrift, Fett: st.Fett, Kursiv: st.Kursiv,
				Unterstr: st.Unterstrichen, Durchgestr: st.Durchgestrichen, Ausrichtung: st.Ausrichtung,
				Schatten: st.Schatten},
			Strichart: st.Strichart, KopfEnde: st.KopfEnde, KopfAnfang: st.KopfAnfang.Art(), Eckradius: st.Eckradius,
			Freihand: st.Freihand,
		})
	}
	return r
}

// SetzeWerkzeugStile übernimmt gemerkte Stile (Faktor 1, aus tools.ini oder einer vorigen Sitzung): Schlüssel, die
// nicht gesetzt sind, behalten die Voreinstellung des Werkzeugs; ein Wert, der mal dem Faktor eine Grenze sprengte,
// wird auf sie begrenzt (F-929).
func (s *Sitzung) SetzeWerkzeugStile(liste []settings.WerkzeugStil) {
	s.stileAngleichen()
	f := s.faktor()
	// begrenze nimmt einen Wert der Datei mal f, höchstens bis zur Grenze (F-929: nicht still die Voreinstellung).
	// Nach dem Malnehmen auf Hundertstel gerundet (F-935): 7 px bei Faktor 3 stehen als 2.333 in der Datei und werden
	// wieder 7, nicht 6,999.
	begrenze := func(v, grenze float64) float64 { return math.Min(math.Round(v*f*100)/100, grenze) }
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
		if gilt("width") && e.Breite > 0 && e.Breite <= document.MaxGroesse {
			st.Breite = begrenze(e.Breite, document.MaxGroesse)
		}
		if gilt("size") && e.Schriftgrad > 0 && e.Schriftgrad <= document.MaxSchrift {
			st.Schriftgrad = begrenze(e.Schriftgrad, document.MaxSchrift)
		}
		if gilt("radius") && e.Radius > 0 && e.Radius <= document.MaxSchrift {
			st.Radius = begrenze(e.Radius, document.MaxSchrift)
		}
		if gilt("corner_radius") && e.Eckradius >= 0 && e.Eckradius <= document.MaxGroesse {
			st.Eckradius = begrenze(e.Eckradius, document.MaxGroesse)
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
		if gilt("freehand") && w == WerkzeugTextmarker {
			st.Freihand = e.Freihand
		}
		if gilt("start_head") {
			st.KopfAnfang = document.AnfangMit(e.KopfAnfang)
		}
		if gilt("shadow") {
			an := st.Schatten.An
			sch := e.Schatten
			sch.Laenge = math.Round(sch.Laenge*f*100) / 100
			sch.Weich = math.Round(sch.Weich*f*100) / 100
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

// zugStilStand merkt die Stile der Werkzeuge vor einem Zug (Regler, Farbwähler); an einem Zug über mehrere gewählte
// sind es die Werkzeuge aller beteiligten (ADR-0039 Nachtrag 4 Punkt 2).
type zugStilStand struct {
	vor map[Werkzeug]Stil
}

// merkeZugStil hält die Stile der Werkzeuge ws vor einem Zug fest.
func (s *Sitzung) merkeZugStil(ws ...Werkzeug) {
	vor := map[Werkzeug]Stil{}
	for _, w := range ws {
		vor[paarSchluessel(w)] = s.stilVon(w)
	}
	s.zugStil = zugStilStand{vor: vor}
}

// zugStilZurueck stellt beim Abbruch eines Zugs die gemerkten Stile wieder her (F-931: abgebrochen heißt, nichts ist
// geschehen — auch nicht in tools.ini). Nach einem abgeschlossenen Zug bleibt der Merker stehen; jeder neue Zug
// überschreibt ihn, und nur ein laufender Zug lässt sich abbrechen.
func (s *Sitzung) zugStilZurueck() {
	for w, vor := range s.zugStil.vor {
		s.aendereStil(w, func(st *Stil) { *st = vor })
	}
	s.zugStil = zugStilStand{}
}
