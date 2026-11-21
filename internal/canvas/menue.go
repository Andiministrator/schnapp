// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Aktionen des Menüs ohne eigenes Werkzeug (ADR-0026).

// sichtbareStreifen teilt die Stauch-Streifen in die, die den Zuschnitt schneiden (dort hat Herausnehmen sichtbar
// gewirkt), und die übrigen.
func (s *Sitzung) sichtbareStreifen() (drin, sonst []document.Streifen) {
	z := s.D.Zuschnitt()
	for _, st := range s.D.Streifen() {
		lo, hi := z.Min.X, z.Max.X
		if st.Zeilen {
			lo, hi = z.Min.Y, z.Max.Y
		}
		if st.Bis > lo && st.Von < hi {
			drin = append(drin, st)
		} else {
			sonst = append(sonst, st)
		}
	}
	return drin, sonst
}

// KannZurueckholen meldet, ob HoleGestauchtesZurueck etwas ändern würde (Menüeintrag an oder grau).
func (s *Sitzung) KannZurueckholen() bool {
	drin, _ := s.sichtbareStreifen()
	return s.zug == keinZug && !s.gesperrt() && len(drin) > 0
}

// HoleGestauchtesZurueck hebt alle Stauch-Streifen im Zuschnitt in einem Rückgängig-Schritt auf, mit und ohne
// Naht; der Zuschnitt bleibt (ADR-0026 Punkt 4). Das ist die Entsprechung zum Doppelklick mit X für alle Nähte.
func (s *Sitzung) HoleGestauchtesZurueck() bool {
	if !s.KannZurueckholen() {
		return false
	}
	_, sonst := s.sichtbareStreifen()
	vor, verborgen := s.D.Stand(), s.VerborgeneObjekte()
	if s.D.SetzeStreifen(sonst) != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.ansichtGeaendert(verborgen)
	return true
}

// KannLoeschen meldet, ob Loeschen ein Objekt entfernen würde (Menüeintrag „Löschen“).
func (s *Sitzung) KannLoeschen() bool { return s.zug == keinZug && s.Auswahl != 0 && !s.gesperrt() }

// KannTextBearbeiten meldet, ob Enter bzw. der Menüeintrag „Text bearbeiten“ eine Eingabe am ausgewählten Text beginnt.
func (s *Sitzung) KannTextBearbeiten() bool {
	if s.zug != keinZug || s.Auswahl == 0 || s.gesperrt() {
		return false
	}
	o, ok := s.D.Objekt(s.Auswahl)
	return ok && isText(o)
}
