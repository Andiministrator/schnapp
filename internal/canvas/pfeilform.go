// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Spitzenarten des Pfeils je Ende (ADR-0021 Nachträge 19 und 20): Die rechte Leiste zeigt die des gewählten Pfeils, sonst die des
// Werkzeugs; Setzen ändert einen gewählten Pfeil als einen Rückgängig-Schritt und merkt die Wahl für folgende Pfeile,
// wie die Strichstärke (SetzeBreite).

// gewaehlterPfeil liefert den gewählten Pfeil.
func (s *Sitzung) gewaehlterPfeil() (document.Pfeil, bool) {
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok {
		return document.Pfeil{}, false
	}
	p, ok := o.(document.Pfeil)
	return p, ok
}

// Pfeilform liefert die Spitzenarten bei A und B, die die rechte Leiste zeigt.
func (s *Sitzung) Pfeilform() (anfang, ende document.Kopfart) {
	if p, ok := s.gewaehlterPfeil(); ok {
		return p.KopfAnfang(), p.Ende
	}
	return s.Stil.KopfAnfang.Art(), s.Stil.KopfEnde
}

// SetzeKopf setzt die Spitzenart k am Anfang (bei A) bzw. am Ende (bei B). Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeKopf(amAnfang bool, k document.Kopfart) bool {
	if amAnfang {
		a := document.AnfangMit(k)
		return s.setzePfeilform(func(p *document.Pfeil) { p.Anfang = a }, func() { s.aendereZielStil(func(st *Stil) { st.KopfAnfang = a }) })
	}
	return s.setzePfeilform(func(p *document.Pfeil) { p.Ende = k }, func() { s.aendereZielStil(func(st *Stil) { st.KopfEnde = k }) })
}

func (s *Sitzung) setzePfeilform(aendere func(*document.Pfeil), merke func()) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	merke()
	p, ok := s.gewaehlterPfeil()
	if !ok {
		return false
	}
	neu := p
	aendere(&neu)
	if document.Gleich(neu, p) {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}
