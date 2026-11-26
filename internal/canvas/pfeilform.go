// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Spitzenart und Enden des Pfeils (ADR-0021 Nachtrag 19): Die rechte Leiste zeigt die des gewählten Pfeils, sonst die des
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

// Pfeilform liefert Spitzenart und Enden, die die rechte Leiste zeigt.
func (s *Sitzung) Pfeilform() (document.Kopfart, document.PfeilEnden) {
	if p, ok := s.gewaehlterPfeil(); ok {
		return p.Kopf, p.Enden
	}
	return s.Stil.Kopf, s.Stil.Enden
}

// SetzeKopf setzt die Spitzenart k. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeKopf(k document.Kopfart) bool {
	return s.setzePfeilform(func(p *document.Pfeil) { p.Kopf = k }, func() { s.Stil.Kopf = k })
}

// SetzeEnden setzt die Enden e. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeEnden(e document.PfeilEnden) bool {
	return s.setzePfeilform(func(p *document.Pfeil) { p.Enden = e }, func() { s.Stil.Enden = e })
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
