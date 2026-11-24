// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Schrift, Größe, Stil und Ausrichtung des Text-Werkzeugs (ADR-0021 Nachtrag 10 Punkt 9): Jede Wahl gilt für neue Texte
// und ändert den gewählten Text als einen Rückgängig-Schritt; während der Texteingabe gehört sie zur Eingabe.

import (
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// SchriftStufen sind die Schriftgrößen der rechten Leiste in Pixeln des Originals (− und + gehen von Stufe zu Stufe).
var SchriftStufen = []float64{8, 10, 12, 14, 16, 20, 24, 28, 32, 40, 48, 64, 80, 96, 128, 160, 200}

// StilArt ist einer der vier Umschalter der Leiste.
type StilArt int

const (
	StilFett StilArt = iota
	StilKursiv
	StilUnterstrichen
	StilDurchgestrichen
)

// TextStil sind die Angaben, die die Leiste beim Text zeigt.
type TextStil struct {
	Schrift         string // leer = Go
	Grad            float64
	Fett, Kursiv    bool
	Unterstrichen   bool
	Durchgestrichen bool
	Ausrichtung     document.Ausrichtung
	Fehlt           bool // die Familie fehlt hier, gesetzt wird in Go
}

// An meldet, ob der Umschalter a an ist.
func (t TextStil) An(a StilArt) bool {
	switch a {
	case StilFett:
		return t.Fett
	case StilKursiv:
		return t.Kursiv
	case StilUnterstrichen:
		return t.Unterstrichen
	}
	return t.Durchgestrichen
}

// aufText überträgt Schrift, Stil und Ausrichtung des Werkzeugs auf einen neuen Text.
func (st Stil) aufText(t *document.Text) {
	t.Schrift, t.Fett, t.Kursiv = st.Schrift, st.Fett, st.Kursiv
	t.Unterstrichen, t.Durchgestrichen, t.Ausrichtung = st.Unterstrichen, st.Durchgestrichen, st.Ausrichtung
}

// textZiel liefert den Text, den eine Wahl ändert: den in der Eingabe (eingabe true), sonst den gewählten.
func (s *Sitzung) textZiel() (document.Text, bool, bool) {
	if t, ok := s.TextObjekt(); ok {
		return t, true, true
	}
	if o, ok := s.D.Objekt(s.Auswahl); ok {
		if t, ok := o.(document.Text); ok {
			return t, false, true
		}
	}
	return document.Text{}, false, false
}

// AktuellerTextStil ist, was die Leiste zeigt: die Angaben des Texts in der Eingabe oder des gewählten Texts, sonst die
// des Werkzeugs.
func (s *Sitzung) AktuellerTextStil() TextStil {
	if t, _, ok := s.textZiel(); ok {
		return TextStil{Schrift: t.Schrift, Grad: t.Schriftgrad, Fett: t.Fett, Kursiv: t.Kursiv,
			Unterstrichen: t.Unterstrichen, Durchgestrichen: t.Durchgestrichen, Ausrichtung: t.Ausrichtung,
			Fehlt: t.SchriftFehlt()}
	}
	st := s.Stil
	return TextStil{Schrift: st.Schrift, Grad: st.Schriftgrad, Fett: st.Fett, Kursiv: st.Kursiv,
		Unterstrichen: st.Unterstrichen, Durchgestrichen: st.Durchgestrichen, Ausrichtung: st.Ausrichtung,
		Fehlt: st.Schrift != "" && !fonts.Bekannt(st.Schrift)}
}

// textAendern setzt eine Angabe am Werkzeug (werkzeug) und am Ziel-Text (text): in der Eingabe als Teil der Eingabe,
// am gewählten Text als ein Rückgängig-Schritt. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) textAendern(werkzeug func(*Stil), text func(*document.Text)) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	werkzeug(&s.Stil)
	t, eingabe, ok := s.textZiel()
	if !ok {
		return false
	}
	neu := t
	text(&neu)
	if document.Gleich(neu, t) {
		return false
	}
	if eingabe {
		_ = s.D.ZugSetze(neu)
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// SetzeSchrift wählt die Familie name („Go“ und leer sind die Voreinstellung); ein ungültiger Name ändert nichts.
func (s *Sitzung) SetzeSchrift(name string) bool {
	if name == fonts.Go {
		name = ""
	}
	if name != "" && !fonts.NameGueltig(name) {
		return false
	}
	return s.textAendern(func(st *Stil) { st.Schrift = name }, func(t *document.Text) { t.Schrift = name })
}

// SetzeSchriftgrad setzt die Schriftgröße (Pixel des Originals, 0 < g ≤ document.MaxSchrift).
func (s *Sitzung) SetzeSchriftgrad(g float64) bool {
	if !(g > 0) || g > document.MaxSchrift {
		return false
	}
	return s.textAendern(func(st *Stil) { st.Schriftgrad = g }, func(t *document.Text) { t.Schriftgrad = g })
}

// SchriftgradStufe geht von der gezeigten Größe zur nächsten Stufe aus SchriftStufen (dir > 0: größer); eine Größe
// zwischen zwei Stufen geht zur nächsten in der Richtung. Liefert die neue Größe und ob es eine gab.
func (s *Sitzung) SchriftgradStufe(dir int) (float64, bool) {
	jetzt := s.AktuellerTextStil().Grad
	if dir > 0 {
		for _, g := range SchriftStufen {
			if g > jetzt && g <= document.MaxSchrift {
				return g, true
			}
		}
		return jetzt, false
	}
	for i := len(SchriftStufen) - 1; i >= 0; i-- {
		if SchriftStufen[i] < jetzt {
			return SchriftStufen[i], true
		}
	}
	return jetzt, false
}

// SchalteStil schaltet einen der vier Umschalter um, ausgehend vom gezeigten Stil.
func (s *Sitzung) SchalteStil(a StilArt) bool {
	an := !s.AktuellerTextStil().An(a)
	setze := func(fett, kursiv, unter, durch *bool) {
		switch a {
		case StilFett:
			*fett = an
		case StilKursiv:
			*kursiv = an
		case StilUnterstrichen:
			*unter = an
		default:
			*durch = an
		}
	}
	return s.textAendern(func(st *Stil) { setze(&st.Fett, &st.Kursiv, &st.Unterstrichen, &st.Durchgestrichen) },
		func(t *document.Text) { setze(&t.Fett, &t.Kursiv, &t.Unterstrichen, &t.Durchgestrichen) })
}

// SetzeAusrichtung wählt die Ausrichtung der Zeilen.
func (s *Sitzung) SetzeAusrichtung(a document.Ausrichtung) bool {
	if a > document.AusrichtungRechts {
		return false
	}
	return s.textAendern(func(st *Stil) { st.Ausrichtung = a }, func(t *document.Text) { t.Ausrichtung = a })
}
