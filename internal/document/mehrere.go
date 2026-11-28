// SPDX-License-Identifier: GPL-3.0-or-later

package document

// Änderungen an mehreren Objekten als ein Schritt im Verlauf (ADR-0039): gemeinsam verschieben (Zug über mehrere),
// Ausrichten und Verteilen (ErsetzeMehrere), Löschen (EntferneMehrere) und Doppeln (FuegeEinMehrere) der
// Mehrfachauswahl. Rückgängig nimmt jede dieser Handlungen als Ganzes zurück.

import (
	"fmt"
	"slices"
)

// gruppe ist ein Befehl aus mehreren, die zusammen ausgeführt und in umgekehrter Reihenfolge zurückgenommen werden.
type gruppe []befehl

func (g gruppe) ausfuehren(d *Dokument) {
	for _, b := range g {
		b.ausfuehren(d)
	}
}

func (g gruppe) zuruecknehmen(d *Dokument) {
	for i := len(g) - 1; i >= 0; i-- {
		g[i].zuruecknehmen(d)
	}
}

// ErsetzeMehrere setzt jedes Objekt aus neu an die Stelle des Objekts mit derselben Kennung, als ein Schritt.
// Unveränderte (Gleich) zählen nicht; ändert sich nichts, entsteht kein Schritt. Ist eine Kennung unbekannt, doppelt oder
// ein Objekt ungültig, geschieht nichts.
func (d *Dokument) ErsetzeMehrere(neu []Objekt) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	var g gruppe
	var alt, neuO []Objekt // die geänderten, für die Gesamtgrenze der Freihand-Punkte (F-1055)
	gesehen := map[int]bool{}
	for _, o := range neu {
		if o == nil || o.Kennung() == 0 || gesehen[o.Kennung()] {
			return ErrUnbekannt
		}
		gesehen[o.Kennung()] = true
		i := d.index(o.Kennung())
		if i < 0 {
			return ErrUnbekannt
		}
		if err := o.pruefe(); err != nil {
			return err
		}
		if !Gleich(o, d.objekte[i]) {
			g = append(g, ersetzen{d.objekte[i], o})
			alt, neuO = append(alt, d.objekte[i]), append(neuO, o)
		}
	}
	if err := d.pruefeFreihandStatt(alt, neuO); err != nil {
		return err
	}
	if len(g) > 0 {
		d.fuehreAus(g)
	}
	return nil
}

// EntferneMehrere nimmt die Objekte mit den Kennungen ids heraus, als ein Schritt. Eine unbekannte Kennung macht nichts.
func (d *Dokument) EntferneMehrere(ids []int) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	var stellen []int
	for _, id := range ids {
		i := d.index(id)
		if i < 0 {
			return ErrUnbekannt
		}
		if !slices.Contains(stellen, i) {
			stellen = append(stellen, i)
		}
	}
	if len(stellen) == 0 {
		return nil
	}
	// Von hinten nach vorn: Jedes Entfernen lässt die Stellen davor unberührt; Rückgängig fügt in umgekehrter
	// Reihenfolge wieder ein, also von vorn nach hinten an genau dieselben Stellen.
	slices.Sort(stellen)
	slices.Reverse(stellen)
	g := make(gruppe, 0, len(stellen))
	for _, i := range stellen {
		g = append(g, entfernen{d.objekte[i], i})
	}
	d.fuehreAus(g)
	return nil
}

// FuegeEinMehrere fügt jedes Objekt aus os direkt über dem Objekt mit der Kennung ueber[i] ein (Doppeln der Auswahl),
// als ein Schritt, und liefert die neuen Kennungen in der Reihenfolge von os.
func (d *Dokument) FuegeEinMehrere(os []Objekt, ueber []int) ([]int, error) {
	if d.zug != nil {
		return nil, ErrZugLaeuft
	}
	if len(os) != len(ueber) {
		return nil, ErrUnbekannt
	}
	if len(os) == 0 {
		return nil, nil // nichts einzufügen: kein leerer Schritt im Verlauf (F-898)
	}
	if len(d.objekte)+len(os) > MaxObjekte {
		return nil, fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
	}
	if err := d.pruefeFreihandDazu(os...); err != nil {
		return nil, err
	}
	type teil struct {
		o      Objekt
		stelle int // Stelle des Originals vor allen Einfügungen
		nr     int // Stelle in os
	}
	teile := make([]teil, 0, len(os))
	for nr, o := range os {
		if o == nil {
			return nil, ErrUnbekannt
		}
		i := d.index(ueber[nr])
		if i < 0 {
			return nil, ErrUnbekannt
		}
		if err := o.pruefe(); err != nil {
			return nil, err
		}
		teile = append(teile, teil{o, i, nr})
	}
	// Von hinten nach vorn einfügen: eine Einfügung hinter Stelle i verschiebt die Stellen davor nicht.
	slices.SortStableFunc(teile, func(a, b teil) int { return b.stelle - a.stelle })
	ids := make([]int, len(os))
	g := make(gruppe, 0, len(teile))
	for _, t := range teile {
		o := t.o.mitKennung(d.naechsteID)
		d.naechsteID++
		ids[t.nr] = o.Kennung()
		g = append(g, einfuegen{o, t.stelle + 1})
	}
	d.fuehreAus(g)
	return ids, nil
}

// BeginneZugMehrere startet einen Zug an den Objekten ids (gemeinsam verschieben). ZugSetzeMehrere setzt ihren Stand,
// BeendeZug übernimmt alle Änderungen als einen Schritt, BrecheZugAb stellt alle her.
func (d *Dokument) BeginneZugMehrere(ids []int) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	if len(ids) == 0 {
		return ErrUnbekannt
	}
	alt := make([]Objekt, 0, len(ids))
	for _, id := range ids {
		i := d.index(id)
		if i < 0 {
			return ErrUnbekannt
		}
		for _, a := range alt {
			if a.Kennung() == id {
				return ErrUnbekannt
			}
		}
		alt = append(alt, d.objekte[i])
	}
	d.zug = &zug{mehrere: alt}
	return nil
}

// ZugSetzeMehrere setzt den aktuellen Stand der gezogenen Objekte; jedes muss zum Zug gehören.
func (d *Dokument) ZugSetzeMehrere(os []Objekt) error {
	if d.zug == nil {
		return ErrKeinZug
	}
	if d.zug.mehrere == nil {
		return fmt.Errorf("document: Zug gilt einem Objekt")
	}
	for _, o := range os {
		if o == nil || !d.zug.gehoert(o.Kennung()) {
			return ErrUnbekannt
		}
		if err := o.pruefe(); err != nil {
			return err
		}
	}
	alt := make([]Objekt, len(os))
	for k, o := range os {
		alt[k] = d.objekte[d.index(o.Kennung())]
	}
	if err := d.pruefeFreihandStatt(alt, os); err != nil {
		return err
	}
	for _, o := range os {
		d.objekte[d.index(o.Kennung())] = o
	}
	return nil
}

// gehoert meldet, ob das Objekt id Teil des Zugs über mehrere ist.
func (z *zug) gehoert(id int) bool {
	for _, a := range z.mehrere {
		if a.Kennung() == id {
			return true
		}
	}
	return false
}

// altVon liefert den Stand vor dem Zug des Objekts id (nil: nicht Teil des Zugs bzw. neues Objekt).
func (z *zug) altVon(id int) (Objekt, bool) {
	if z.mehrere != nil {
		for _, a := range z.mehrere {
			if a.Kennung() == id {
				return a, true
			}
		}
		return nil, false
	}
	if id == z.id {
		return z.alt, true
	}
	return nil, false
}

// beendeMehrere übernimmt einen Zug über mehrere als einen Schritt (BeendeZug).
func (d *Dokument) beendeMehrere(z *zug) {
	var g gruppe
	for _, a := range z.mehrere {
		i := d.index(a.Kennung())
		neu := d.objekte[i]
		if !Gleich(neu, a) {
			d.objekte[i] = a
			g = append(g, ersetzen{a, neu})
		}
	}
	if len(g) > 0 {
		d.fuehreAus(g)
	}
}

// brecheMehrereAb stellt die Objekte eines Zugs über mehrere her (BrecheZugAb).
func (d *Dokument) brecheMehrereAb(z *zug) {
	for _, a := range z.mehrere {
		d.objekte[d.index(a.Kennung())] = a
	}
}

// HinzufuegenMehrere legt alle Objekte aus os in dieser Reihenfolge zuoberst ab (Einfügen aus der Zwischenablage,
// ADR-0045 Punkt 3), als ein Schritt, und liefert die neuen Kennungen in der Reihenfolge von os.
func (d *Dokument) HinzufuegenMehrere(os []Objekt) ([]int, error) {
	if d.zug != nil {
		return nil, ErrZugLaeuft
	}
	if len(os) == 0 {
		return nil, nil // kein leerer Schritt im Verlauf
	}
	if len(d.objekte)+len(os) > MaxObjekte {
		return nil, fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
	}
	if err := d.pruefeFreihandDazu(os...); err != nil {
		return nil, err
	}
	for _, o := range os {
		if o == nil {
			return nil, ErrUnbekannt
		}
		if err := o.pruefe(); err != nil {
			return nil, err
		}
	}
	ids := make([]int, len(os))
	g := make(gruppe, 0, len(os))
	for i, o := range os {
		o = o.mitKennung(d.naechsteID)
		d.naechsteID++
		ids[i] = o.Kennung()
		g = append(g, einfuegen{o, len(d.objekte) + i})
	}
	d.fuehreAus(g)
	return ids, nil
}
