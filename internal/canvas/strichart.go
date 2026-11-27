// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Strichart von Pfeil, Kasten und Lupe (ADR-0021 Nachtrag 25): Die rechte Leiste zeigt die des gewählten Objekts, sonst
// die des Werkzeugs; Setzen ändert ein gewähltes Objekt als einen Rückgängig-Schritt und merkt die Wahl für folgende
// Objekte, wie die Strichstärke (SetzeBreite).

// Strichart liefert die Strichart, die die rechte Leiste zeigt.
func (s *Sitzung) Strichart() document.Strichart {
	o, _ := s.strichObjekt()
	switch v := o.(type) {
	case document.Pfeil:
		return v.Strich.Art
	case document.Kasten:
		return v.Strich.Art
	case document.Lupe:
		return v.Strich.Art
	}
	return s.Stil.Strichart
}

// SetzeStrichart setzt die Strichart a für folgende Pfeile, Kästen und Lupen und ändert ein gewähltes solches Objekt als
// einen Rückgängig-Schritt. Liefert, ob sich das Dokument geändert hat; dieselbe Art ist kein Schritt.
func (s *Sitzung) SetzeStrichart(a document.Strichart) bool {
	if s.Zieht() || s.gesperrt() || int(a) >= len(document.Stricharten) {
		return false
	}
	s.aendereZielStil(func(st *Stil) { st.Strichart = a })
	o, ok := s.strichObjekt()
	if !ok {
		return false
	}
	neu := mitStrichart(o, a)
	if document.Gleich(neu, o) {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// mitStrichart liefert Pfeil, Kasten oder Lupe o mit der Strichart a; andere Objekte unverändert.
func mitStrichart(o document.Objekt, a document.Strichart) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		v.Strich.Art = a
		return v
	case document.Kasten:
		v.Strich.Art = a
		return v
	case document.Lupe:
		v.Strich.Art = a
		return v
	}
	return o
}
