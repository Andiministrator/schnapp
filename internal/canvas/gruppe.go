// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Flache Gruppen (ADR-0044): Eine Gruppe ist die Kennung document.Ebene.Gruppe an jedem Mitglied. Ein Klick auf ein
// Mitglied wählt die ganze Gruppe (Hauptauswahl wird das angeklickte), Umschalt+Klick nimmt sie dazu oder heraus, der
// Rahmen nimmt sie nur ganz; die Ebenenliste wählt weiter zeilenweise. Strg+G gruppiert, Strg+Umschalt+G hebt auf.

import (
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
)

// gruppenGroesse zählt die Mitglieder je Gruppe im ganzen Dokument (auch gesperrte und ausgeblendete).
func (s *Sitzung) gruppenGroesse() map[int]int {
	n := map[int]int{}
	for _, o := range s.D.Objekte() {
		if g := o.Ebenenwerte().Gruppe; g != 0 {
			n[g]++
		}
	}
	return n
}

// GruppeVon ist die wirksame Gruppe des Objekts id: seine Kennung, wenn die Gruppe mindestens zwei Mitglieder hat,
// sonst 0 (ADR-0044 Punkt 1).
func (s *Sitzung) GruppeVon(id int) int {
	o, ok := s.D.Objekt(id)
	if !ok {
		return 0
	}
	g := o.Ebenenwerte().Gruppe
	if g == 0 || s.gruppenGroesse()[g] < 2 {
		return 0
	}
	return g
}

// gruppenMitglieder sind die wählbaren Objekte der Gruppe von id, mit id selbst; ohne wirksame Gruppe nur id.
func (s *Sitzung) gruppenMitglieder(id int) []int {
	g := s.GruppeVon(id)
	if g == 0 {
		return []int{id}
	}
	ids := []int{id}
	for _, o := range s.D.Objekte() {
		if k := o.Kennung(); k != id && o.Ebenenwerte().Gruppe == g && s.waehlbar(k) {
			ids = append(ids, k)
		}
	}
	return ids
}

// waehleGruppe wählt das Objekt id mit seiner Gruppe; id wird Hauptauswahl.
func (s *Sitzung) waehleGruppe(id int) { s.setzeAuswahl(id, s.gruppenMitglieder(id)) }

// schalteGruppe nimmt die Gruppe von id zur Auswahl hinzu (id wird Hauptauswahl) oder, ist id schon gewählt, heraus
// (Umschalt+Klick auf der Leinwand, ADR-0044 Punkt 3).
func (s *Sitzung) schalteGruppe(id int) bool {
	if !s.waehlbar(id) || s.zug != keinZug || s.gesperrt() {
		return false
	}
	mit := s.gruppenMitglieder(id)
	alle := s.Gewaehlt()
	if slices.Contains(alle, id) {
		alle = slices.DeleteFunc(alle, func(x int) bool { return slices.Contains(mit, x) })
		s.setzeAuswahl(s.obersteVon(alle), alle)
		return true
	}
	s.setzeAuswahl(id, append(alle, mit...))
	return true
}

// nurGanzeGruppen lässt aus ids die Mitglieder von Gruppen weg, deren wählbare Mitglieder nicht alle in ids stehen
// (Rahmen, ADR-0044 Punkt 3).
func (s *Sitzung) nurGanzeGruppen(ids []int) []int {
	var r []int
	for _, id := range ids {
		ganz := true
		for _, m := range s.gruppenMitglieder(id) {
			if !slices.Contains(ids, m) {
				ganz = false
				break
			}
		}
		if ganz {
			r = append(r, id)
		}
	}
	return r
}

// freieGruppe ist die kleinste Kennung, die kein Objekt trägt.
func (s *Sitzung) freieGruppe(auch map[int]bool) int {
	belegt := map[int]bool{}
	for _, o := range s.D.Objekte() {
		belegt[o.Ebenenwerte().Gruppe] = true
	}
	for g := 1; ; g++ {
		if !belegt[g] && !auch[g] {
			return g
		}
	}
}

// mitGruppe ist o mit der Gruppe g.
func mitGruppe(o document.Objekt, g int) document.Objekt {
	e := o.Ebenenwerte()
	e.Gruppe = g
	return document.MitEbene(o, e)
}

// KannGruppieren meldet, ob Strg+G wirkt: mindestens zwei gewählt, und sie sind nicht schon genau eine Gruppe.
func (s *Sitzung) KannGruppieren() bool {
	if !s.mehrereBereit(2) {
		return false
	}
	ids := s.Gewaehlt()
	g := s.GruppeVon(ids[0])
	if g == 0 {
		return true
	}
	for _, id := range ids[1:] {
		if s.GruppeVon(id) != g {
			return true
		}
	}
	return s.gruppenGroesse()[g] != len(ids) // ein Teil einer größeren Gruppe wird eine eigene
}

// Gruppieren macht alle gewählten zu einer neuen Gruppe, als einen Schritt (Strg+G, ADR-0044 Punkt 2).
func (s *Sitzung) Gruppieren() bool {
	if !s.KannGruppieren() {
		return false
	}
	s.TextBestaetigen()
	g := s.freieGruppe(nil)
	os := s.gewaehlteObjekte()
	neu := make([]document.Objekt, len(os))
	for i, o := range os {
		neu[i] = mitGruppe(o, g)
	}
	return s.ersetzeAlsSchritt(neu)
}

// KannGruppierungAufheben meldet, ob Strg+Umschalt+G wirkt: ein gewähltes gehört zu einer Gruppe.
func (s *Sitzung) KannGruppierungAufheben() bool {
	if s.zug != keinZug || s.D.ZugLaeuft() || s.gesperrt() {
		return false
	}
	for _, o := range s.gewaehlteObjekte() {
		if o.Ebenenwerte().Gruppe != 0 {
			return true
		}
	}
	return false
}

// GruppierungAufheben nimmt alle gewählten aus ihren Gruppen, als einen Schritt (Strg+Umschalt+G). Die Auswahl bleibt.
func (s *Sitzung) GruppierungAufheben() bool {
	if !s.KannGruppierungAufheben() {
		return false
	}
	var neu []document.Objekt
	for _, o := range s.gewaehlteObjekte() {
		if o.Ebenenwerte().Gruppe != 0 {
			neu = append(neu, mitGruppe(o, 0))
		}
	}
	return s.ersetzeAlsSchritt(neu)
}

// ersetzeAlsSchritt setzt neu als einen Rückgängig-Schritt (ohne Prüfung der Sichtbarkeit: nur Ebenenangaben).
func (s *Sitzung) ersetzeAlsSchritt(neu []document.Objekt) bool {
	vor := s.D.Stand()
	if s.D.ErsetzeMehrere(neu) != nil {
		return false
	}
	s.merkeAenderung(vor)
	return s.D.Stand() != vor
}

// gewaehlteGruppe ist die Gruppe von id, wenn sie ganz gewählt ist (alle wählbaren Mitglieder), sonst 0.
func (s *Sitzung) gewaehlteGruppe(id int) int {
	g := s.GruppeVon(id)
	if g == 0 || !s.IstGewaehlt(id) {
		return 0
	}
	for _, m := range s.gruppenMitglieder(id) {
		if !s.IstGewaehlt(m) {
			return 0
		}
	}
	return g
}

// SchalteAusgeblendetGruppe blendet id aus bzw. ein; ist seine Gruppe gewählt, alle Mitglieder mit (Strg+H, ADR-0044
// Punkt 4). Ziel ist der umgekehrte Stand von id. Ein Schritt.
func (s *Sitzung) SchalteAusgeblendetGruppe(id int) bool {
	return s.schalteGruppeEbene(id, s.SchalteAusgeblendet, func(e *document.Ebene) *bool { return &e.Ausgeblendet })
}

// SchalteGesperrtGruppe sperrt id bzw. gibt es frei; ist seine Gruppe gewählt, alle Mitglieder mit (Strg+L).
func (s *Sitzung) SchalteGesperrtGruppe(id int) bool {
	return s.schalteGruppeEbene(id, s.SchalteGesperrt, func(e *document.Ebene) *bool { return &e.Gesperrt })
}

func (s *Sitzung) schalteGruppeEbene(id int, einzeln func(int) bool, feld func(*document.Ebene) *bool) bool {
	g := s.gewaehlteGruppe(id)
	if g == 0 {
		return einzeln(id)
	}
	if !s.ebenenBereit() {
		return false
	}
	o, ok := s.D.Objekt(id)
	if !ok {
		return false
	}
	e := o.Ebenenwerte()
	ziel := !*feld(&e)
	var neu []document.Objekt
	for _, m := range s.D.Objekte() {
		if m.Ebenenwerte().Gruppe == g {
			me := m.Ebenenwerte()
			*feld(&me) = ziel
			neu = append(neu, document.MitEbene(m, me))
		}
	}
	ok = s.ersetzeAlsSchritt(neu)
	s.bereinigeAuswahl()
	return ok
}

// neueGruppenFuer gibt den Kopien os (Doppeln mehrerer) neue Gruppen: dieselbe Aufteilung, andere Kennungen
// (ADR-0044 Punkt 4).
func (s *Sitzung) neueGruppenFuer(os []document.Objekt) []document.Objekt {
	neu := map[int]int{}
	vergeben := map[int]bool{}
	for i, o := range os {
		g := o.Ebenenwerte().Gruppe
		if g == 0 {
			continue
		}
		n, ok := neu[g]
		if !ok {
			n = s.freieGruppe(vergeben)
			neu[g], vergeben[n] = n, true
		}
		os[i] = mitGruppe(o, n)
	}
	return os
}
