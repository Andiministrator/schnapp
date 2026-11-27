// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import "github.com/Andiministrator/schnapp/internal/library"

// Markierung sind die markierten Vorschauen der Leiste oder der vollen Ansicht (ADR-0036 Zusatz Karte #361): eine Menge
// von Pfaden relativ zur Ablage und ein Anker, von dem aus Umschalt+Klick und Umschalt+Pfeil einen Bereich markieren.
// Sie gilt nur für Gezeigtes: Bereinige wirft Pfade hinaus, die nicht mehr in der Liste stehen. Ohne Gio.
type Markierung struct {
	pfade map[string]bool
	anker string // leer: keiner
}

// Anzahl ist die Zahl der markierten Aufnahmen.
func (m *Markierung) Anzahl() int { return len(m.pfade) }

// Enthaelt meldet, ob die Aufnahme rel markiert ist.
func (m *Markierung) Enthaelt(rel string) bool { return rel != "" && m.pfade[rel] }

// Anker ist der Pfad, von dem ein Bereich ausgeht; leer: keiner.
func (m *Markierung) Anker() string { return m.anker }

// SetzeAnker setzt den Anker auf rel, ohne die Markierung zu ändern (Klick ohne Taste, Fokus per Pfeil).
func (m *Markierung) SetzeAnker(rel string) { m.anker = rel }

// Leere hebt die Markierung auf; der Anker bleibt.
func (m *Markierung) Leere() { m.pfade = nil }

// setze markiert rel.
func (m *Markierung) setze(rel string) {
	if rel == "" {
		return
	}
	if m.pfade == nil {
		m.pfade = map[string]bool{}
	}
	m.pfade[rel] = true
}

// Umschalte nimmt rel in die Markierung auf oder heraus (Strg+Klick, Strg+Leertaste); rel wird Anker.
func (m *Markierung) Umschalte(rel string) {
	if rel == "" {
		return
	}
	if m.pfade[rel] {
		delete(m.pfade, rel)
	} else {
		m.setze(rel)
	}
	m.anker = rel
}

// Bereich markiert die Aufnahmen vom Anker bis ziel in der Reihenfolge von es (Umschalt+Klick, Umschalt+Pfeil). dazu:
// der Bereich kommt zur Markierung hinzu (Strg+Umschalt+Klick), sonst ersetzt er sie. Steht der Anker nicht in es, ist
// ziel der Anker; der Anker selbst bleibt, damit ein weiterer Umschalt+Klick vom selben Punkt aus misst.
func (m *Markierung) Bereich(es []library.Eintrag, ziel string, dazu bool) {
	z := stelleIn(es, ziel)
	if z < 0 {
		return
	}
	a := stelleIn(es, m.anker)
	if a < 0 {
		a, m.anker = z, ziel
	}
	if !dazu {
		m.pfade = nil
	}
	von, bis := min(a, z), max(a, z)
	for i := von; i <= bis; i++ {
		m.setze(es[i].Pfad)
	}
}

// Alle markiert alle Aufnahmen in es (Strg+A).
func (m *Markierung) Alle(es []library.Eintrag) {
	for _, e := range es {
		m.setze(e.Pfad)
	}
}

// Bereinige wirft alles hinaus, was nicht mehr in es steht, auch den Anker (ADR-0036 Zusatz Karte #361 Punkt 1: es
// geht nie eine Aufnahme in den Papierkorb, die nicht gezeigt wird).
func (m *Markierung) Bereinige(es []library.Eintrag) {
	if len(m.pfade) == 0 && m.anker == "" {
		return
	}
	da := make(map[string]bool, len(es))
	for _, e := range es {
		da[e.Pfad] = true
	}
	for p := range m.pfade {
		if !da[p] {
			delete(m.pfade, p)
		}
	}
	if !da[m.anker] {
		m.anker = ""
	}
}

// Pfade sind die markierten Aufnahmen in der Reihenfolge von es (neueste zuerst wie die Liste).
func (m *Markierung) Pfade(es []library.Eintrag) []string {
	var aus []string
	for _, e := range es {
		if m.pfade[e.Pfad] {
			aus = append(aus, e.Pfad)
		}
	}
	return aus
}

// stelleIn ist die Stelle von rel in es, -1 ohne.
func stelleIn(es []library.Eintrag, rel string) int {
	if rel == "" {
		return -1
	}
	for i, e := range es {
		if e.Pfad == rel {
			return i
		}
	}
	return -1
}
