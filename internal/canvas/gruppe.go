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

// gruppenMitglieder sind die wählbaren Objekte der Gruppe von id, mit id selbst; ohne wirksame Gruppe nur id. Zwei
// Durchgänge durch den Stapel, ohne Suche je Kennung (F-961).
func (s *Sitzung) gruppenMitglieder(id int) []int {
	g := s.GruppeVon(id)
	if g == 0 {
		return []int{id}
	}
	ids := []int{id}
	for _, o := range s.D.Objekte() {
		if k := o.Kennung(); k != id && o.Ebenenwerte().Gruppe == g && s.waehlbarObjekt(o) {
			ids = append(ids, k)
		}
	}
	return ids
}

// waehlbarObjekt ist waehlbar ohne Suche: sichtbar, nicht gesperrt.
func (s *Sitzung) waehlbarObjekt(o document.Objekt) bool {
	return s.D.ObjektSichtbar(o) && o.Ebenenwerte().Bearbeitbar()
}

// waehlbareJeGruppe zählt die wählbaren Mitglieder je Kennung (ein Durchgang).
func (s *Sitzung) waehlbareJeGruppe() map[int]int {
	n := map[int]int{}
	for _, o := range s.D.Objekte() {
		if g := o.Ebenenwerte().Gruppe; g != 0 && s.waehlbarObjekt(o) {
			n[g]++
		}
	}
	return n
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
// (Rahmen, ADR-0044 Punkt 3). Ein Durchgang durch den Stapel (F-961: vorher je Kennung einer).
func (s *Sitzung) nurGanzeGruppen(ids []int) []int {
	in := make(map[int]bool, len(ids))
	for _, id := range ids {
		in[id] = true
	}
	groesse, waehlbar := s.gruppenGroesse(), s.waehlbareJeGruppe()
	gruppeVon, drin := map[int]int{}, map[int]int{}
	for _, o := range s.D.Objekte() {
		if g := o.Ebenenwerte().Gruppe; in[o.Kennung()] && g != 0 && groesse[g] >= 2 {
			gruppeVon[o.Kennung()] = g
			drin[g]++
		}
	}
	var r []int
	for _, id := range ids {
		if g, ok := gruppeVon[id]; !ok || drin[g] == waehlbar[g] {
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

// KannGruppieren meldet, ob Strg+G wirkt: mindestens zwei gewählt, und sie sind nicht schon genau die wählbaren
// Mitglieder einer Gruppe (F-960: gesperrte oder ausgeblendete Mitglieder zählen dafür nicht).
func (s *Sitzung) KannGruppieren() bool {
	if !s.mehrereBereit(2) {
		return false
	}
	os := s.gewaehlteObjekte()
	g := os[0].Ebenenwerte().Gruppe
	if g == 0 || s.gruppenGroesse()[g] < 2 {
		return true
	}
	for _, o := range os[1:] {
		if o.Ebenenwerte().Gruppe != g {
			return true
		}
	}
	return s.waehlbareJeGruppe()[g] != len(os) // ein Teil einer größeren Gruppe wird eine eigene
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
	return s.ersetzeAlsSchritt(s.mitAufgeraeumt(neu))
}

// KannGruppierungAufheben meldet, ob Strg+Umschalt+G wirkt: ein gewähltes gehört zu einer wirksamen Gruppe (F-962).
func (s *Sitzung) KannGruppierungAufheben() bool {
	if s.zug != keinZug || s.D.ZugLaeuft() || s.gesperrt() {
		return false
	}
	groesse := s.gruppenGroesse()
	for _, o := range s.gewaehlteObjekte() {
		if g := o.Ebenenwerte().Gruppe; g != 0 && groesse[g] >= 2 {
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
	return s.ersetzeAlsSchritt(s.mitAufgeraeumt(neu))
}

// mitAufgeraeumt ergänzt die Änderungen neu um die Objekte, die danach als einziges ihre Kennung trügen: Sie verlieren
// sie im selben Schritt (F-962: keine verwaiste Kennung, kein „Ungroup“ ohne Wirkung).
func (s *Sitzung) mitAufgeraeumt(neu []document.Objekt) []document.Objekt {
	geaendert := make(map[int]document.Objekt, len(neu))
	for _, o := range neu {
		geaendert[o.Kennung()] = o
	}
	zahl := map[int]int{}
	for _, o := range s.D.Objekte() {
		if n, ok := geaendert[o.Kennung()]; ok {
			o = n
		}
		if g := o.Ebenenwerte().Gruppe; g != 0 {
			zahl[g]++
		}
	}
	for _, o := range s.D.Objekte() {
		if _, ok := geaendert[o.Kennung()]; ok {
			continue
		}
		if g := o.Ebenenwerte().Gruppe; g != 0 && zahl[g] == 1 {
			neu = append(neu, mitGruppe(o, 0))
		}
	}
	return neu
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

// gewaehlteGruppe ist die Gruppe von id, wenn sie als Gruppe gewählt ist, sonst 0: mindestens zwei Mitglieder und alle
// wählbaren gewählt (F-959: ein einzeln per Liste gewähltes Mitglied ist es nicht, auch wenn die übrigen gesperrt sind).
func (s *Sitzung) gewaehlteGruppe(id int) int {
	g := s.GruppeVon(id)
	if g == 0 || !s.IstGewaehlt(id) {
		return 0
	}
	n := 0
	for _, o := range s.gewaehlteObjekte() {
		if o.Ebenenwerte().Gruppe == g {
			n++
		}
	}
	if n >= 2 && n == s.waehlbareJeGruppe()[g] {
		return g
	}
	return 0
}

// Ebenenschalter ist Ausblenden (Strg+H) oder Sperren (Strg+L).
type Ebenenschalter int

const (
	SchalterAusblenden Ebenenschalter = iota + 1
	SchalterSperren
)

// feld ist der Schalter an der Ebene e.
func (a Ebenenschalter) feld(e *document.Ebene) *bool {
	if a == SchalterSperren {
		return &e.Gesperrt
	}
	return &e.Ausgeblendet
}

// gruppenSchaltung sind die Objekte, die Strg+H bzw. Strg+L an der Hauptauswahl id als Gruppe schaltet (ADR-0044
// Punkt 4, Nachtrag 2), mit dem Ziel; false: es gilt nur id.
//   - Hat derselbe Schalter eben eine Menge geschaltet und wurde die Auswahl seither nicht anders geändert, schaltet er
//     genau diese Menge zurück (F-958, F-964: auch wenn die Hauptauswahl dabei auf ein Fremdobjekt rückte; F-966: ein
//     vorher einzeln ausgeblendetes Mitglied gehört nicht dazu).
//   - Ist die Gruppe gewählt (gewaehlteGruppe), schaltet er ihre Mitglieder, die noch nicht im Ziel sind.
//   - Ist die Hauptauswahl selbst ausgeblendet bzw. gesperrt und alle Mitglieder ihrer Gruppe sind es auch (etwa nach
//     Rückgängig und Wiederholen, F-965), schaltet er sie alle zurück.
func (s *Sitzung) gruppenSchaltung(id int, a Ebenenschalter) ([]int, bool, bool) {
	m := &s.mf
	if m.schaltArt == a && m.schaltGen == m.gen && len(m.geschaltet) > 0 {
		if o, ok := s.D.Objekt(m.geschaltet[0]); ok {
			e := o.Ebenenwerte()
			return m.geschaltet, !*a.feld(&e), true
		}
	}
	o, ok := s.D.Objekt(id)
	g := s.GruppeVon(id)
	if !ok || g == 0 {
		return nil, false, false
	}
	e := o.Ebenenwerte()
	stand := *a.feld(&e)
	ziel := !stand
	gruppe := s.gewaehlteGruppe(id) != 0
	if !gruppe && stand {
		gruppe = true
		for _, x := range s.D.Objekte() {
			if xe := x.Ebenenwerte(); xe.Gruppe == g && *a.feld(&xe) != stand {
				gruppe = false
			}
		}
	}
	if !gruppe {
		return nil, false, false
	}
	var ids []int
	for _, x := range s.D.Objekte() {
		if xe := x.Ebenenwerte(); xe.Gruppe == g && *a.feld(&xe) != ziel {
			ids = append(ids, x.Kennung())
		}
	}
	return ids, ziel, len(ids) > 0
}

// GruppeSchaltet meldet, ob Strg+H bzw. Strg+L an der Hauptauswahl eine Gruppe schaltet (Menütext „Hide group“,
// F-963).
func (s *Sitzung) GruppeSchaltet(a Ebenenschalter) bool {
	_, _, ok := s.gruppenSchaltung(s.Auswahl, a)
	return ok
}

// SchalteAusgeblendetGruppe blendet id aus bzw. ein, als Gruppe nach gruppenSchaltung (Strg+H). Ein Schritt.
func (s *Sitzung) SchalteAusgeblendetGruppe(id int) bool {
	return s.schalteGruppeEbene(id, SchalterAusblenden, s.SchalteAusgeblendet)
}

// SchalteGesperrtGruppe sperrt id bzw. gibt es frei, als Gruppe nach gruppenSchaltung (Strg+L). Ein Schritt.
func (s *Sitzung) SchalteGesperrtGruppe(id int) bool {
	return s.schalteGruppeEbene(id, SchalterSperren, s.SchalteGesperrt)
}

func (s *Sitzung) schalteGruppeEbene(id int, a Ebenenschalter, einzeln func(int) bool) bool {
	// Erst die Texteingabe bestätigen (sie kann das Objekt löschen und die Auswahl ändern), dann rechnen (F-972).
	if !s.ebenenBereit() {
		return false
	}
	if _, da := s.D.Objekt(id); !da {
		id = s.Auswahl
	}
	m := &s.mf
	zurueck := m.schaltArt == a && m.schaltGen == m.gen && len(m.geschaltet) > 0
	ids, ziel, ok := s.gruppenSchaltung(id, a)
	if !ok {
		if m.schaltArt == a { // das Gedächtnis des anderen Schalters bleibt (F-973)
			m.geschaltet = nil
		}
		return einzeln(id)
	}
	vorHaupt, vorMehr := s.Auswahl, s.Gewaehlt()
	var neu []document.Objekt
	for _, k := range ids {
		if o, ok := s.D.Objekt(k); ok {
			e := o.Ebenenwerte()
			*a.feld(&e) = ziel
			neu = append(neu, document.MitEbene(o, e))
		}
	}
	if !s.ersetzeAlsSchritt(neu) {
		return false
	}
	switch {
	case zurueck && !ziel: // zurückgeschaltet: die Auswahl von vor dem Hinschalten (F-964)
		s.setzeAuswahl(m.schaltVorHaupt, m.schaltVorMehr)
		s.bereinigeAuswahl()
	case !ziel: // sichtbar bzw. offen: die Gruppe wählen (F-965)
		s.waehleGruppe(id)
	default: // ausgeblendet bzw. gesperrt: sie fallen aus der Auswahl, die Hauptauswahl bleibt
		m.schaltVorHaupt, m.schaltVorMehr = vorHaupt, vorMehr
		s.bereinigeAuswahl()
	}
	if ziel {
		m.geschaltet, m.schaltArt, m.schaltGen = ids, a, m.gen
	} else {
		m.geschaltet = nil
	}
	return true
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
