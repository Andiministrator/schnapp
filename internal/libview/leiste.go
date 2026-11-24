// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"github.com/Andiministrator/schnapp/internal/library"
)

// MaxEintraege ist die größte Zahl von Aufnahmen in der Leiste: die neuesten. Ältere erreicht die volle Ansicht
// (ADR-0036 Punkt 3, Teil 2); die Leiste soll schnell bleiben, auch bei einer großen Bibliothek.
const MaxEintraege = 200

// Masse sind die Maße der Leiste in Pixeln, die die Oberfläche aus dp rechnet: Rand links und rechts der Spur, Breite
// einer Kachel, Lücke zwischen zwei Kacheln und die sichtbare Breite der Spur.
type Masse struct {
	Rand, Kachel, Luecke, Breite int
}

// Schritt ist eine Taste, die den Fokus in der Leiste bewegt (ADR-0036 Punkt 1, Fokusfahrt).
type Schritt int

const (
	SchrittLinks Schritt = iota
	SchrittRechts
	SchrittAnfang
	SchrittEnde
)

// Leiste ist der Zustand der Leiste der letzten Aufnahmen: die gezeigten Einträge (neueste links), die offene Aufnahme,
// der Tastaturfokus und der waagerechte Versatz. Ohne Gio; die Oberfläche fragt Lage und sichtbaren Bereich ab.
type Leiste struct {
	eintraege []library.Eintrag
	aktuell   string  // Pfad der offenen Aufnahme relativ zur Ablage; leer: keine aus der Bibliothek
	fokus     int     // Stelle mit Tastaturfokus; -1: keine
	rollen    int     // Versatz der Spur in Pixeln, 0 = ganz links
	rest      float32 // Radweg unter einem Pixel (Touchpad)
}

// NeueLeiste liefert eine leere Leiste ohne Fokus.
func NeueLeiste() *Leiste { return &Leiste{fokus: -1} }

// Setze übernimmt die Aufnahmen aus dem Index (neueste zuerst, wie library.Suchen sie liefert). Agenten-Aufnahmen
// fallen heraus (ADR-0036 Punkt 1), ebenso alles hinter MaxEintraege. Der Fokus bleibt bei derselben Aufnahme; gibt es
// sie nicht mehr, rückt er auf die letzte vorhandene Stelle.
func (l *Leiste) Setze(es []library.Eintrag) {
	var alt string
	if l.fokus >= 0 && l.fokus < len(l.eintraege) {
		alt = l.eintraege[l.fokus].Pfad
	}
	neu := make([]library.Eintrag, 0, min(len(es), MaxEintraege))
	for _, e := range es {
		if e.Agent() {
			continue
		}
		if len(neu) == MaxEintraege {
			break
		}
		neu = append(neu, e)
	}
	l.eintraege = neu
	if l.fokus < 0 {
		return
	}
	for i, e := range neu {
		if e.Pfad == alt {
			l.fokus = i
			return
		}
	}
	l.fokus = min(l.fokus, len(neu)-1)
}

// Eintraege sind die gezeigten Aufnahmen, neueste zuerst.
func (l *Leiste) Eintraege() []library.Eintrag { return l.eintraege }

// SetzeAktuell merkt die offene Aufnahme (Pfad relativ zur Ablage; leer: keine aus der Bibliothek).
func (l *Leiste) SetzeAktuell(rel string) { l.aktuell = rel }

// Aktuell ist der Pfad der offenen Aufnahme relativ zur Ablage.
func (l *Leiste) Aktuell() string { return l.aktuell }

// IstAktuell meldet, ob die Stelle i die offene Aufnahme ist (grün umrandet, ein Klick darauf wechselt nicht).
func (l *Leiste) IstAktuell(i int) bool {
	return l.aktuell != "" && i >= 0 && i < len(l.eintraege) && l.eintraege[i].Pfad == l.aktuell
}

// Fokus ist die Stelle mit Tastaturfokus; -1: keine.
func (l *Leiste) Fokus() int { return l.fokus }

// SetzeFokus setzt den Fokus auf die Stelle i; außerhalb der Liste: kein Fokus.
func (l *Leiste) SetzeFokus(i int) {
	if i < 0 || i >= len(l.eintraege) {
		i = -1
	}
	l.fokus = i
}

// Bewege liefert die Stelle, auf die der Fokus von i aus mit der Taste s rückt (an den Enden bleibt er stehen); -1 bei
// leerer Liste.
func (l *Leiste) Bewege(i int, s Schritt) int {
	n := len(l.eintraege)
	if n == 0 {
		return -1
	}
	switch s {
	case SchrittLinks:
		i--
	case SchrittRechts:
		i++
	case SchrittAnfang:
		i = 0
	case SchrittEnde:
		i = n - 1
	}
	return max(0, min(i, n-1))
}

// inhalt ist die Breite der ganzen Spur in Pixeln.
func (l *Leiste) inhalt(m Masse) int {
	n := len(l.eintraege)
	if n == 0 {
		return 0
	}
	return 2*m.Rand + n*m.Kachel + (n-1)*m.Luecke
}

// RollMax ist der größte Versatz: wie weit die Spur über die sichtbare Breite hinausragt.
func (l *Leiste) RollMax(m Masse) int { return max(0, l.inhalt(m)-m.Breite) }

// Rollen ist der Versatz, begrenzt auf 0 … RollMax (ein schmaler gewordenes Fenster oder eine kürzere Liste).
func (l *Leiste) Rollen(m Masse) int {
	l.rollen = max(0, min(l.rollen, l.RollMax(m)))
	return l.rollen
}

// Lage liefert linke und rechte Kante der Kachel i in Pixeln ab dem linken Rand der sichtbaren Spur (nach dem Rollen).
func (l *Leiste) Lage(i int, m Masse) (x0, x1 int) {
	x0 = m.Rand + i*(m.Kachel+m.Luecke) - l.Rollen(m)
	return x0, x0 + m.Kachel
}

// Sichtbar liefert die Stellen von bis bis (ausschließlich), die ganz oder teilweise sichtbar sind, dazu je eine
// Kachel Vorlauf auf beiden Seiten (ADR-0036 Punkt 6: nur sie werden geladen).
func (l *Leiste) Sichtbar(m Masse) (von, bis int) {
	n := len(l.eintraege)
	schritt := m.Kachel + m.Luecke
	if n == 0 || schritt <= 0 {
		return 0, 0
	}
	r := l.Rollen(m)
	von = (r - m.Rand - m.Kachel) / schritt // letzte Kachel, deren rechte Kante links vom Sichtbereich liegt
	bis = (r+m.Breite-m.Rand)/schritt + 1   // erste Kachel, die rechts nicht mehr beginnt
	return max(0, von), min(n, bis+1)
}

// Rad rollt um den Radweg d (Pixel; ein senkrechtes Rad rollt waagerecht, ADR-0036 Punkt 1). Der Bruchteil unter einem
// Pixel bleibt als Rest; am Anschlag verfällt er.
func (l *Leiste) Rad(d float32, m Masse) {
	rmax := l.RollMax(m)
	l.rest += d
	n := int(l.rest)
	l.rest -= float32(n)
	neu := max(0, min(l.Rollen(m)+n, rmax))
	if neu != l.rollen+n || (neu == 0 && l.rest < 0) || (neu == rmax && l.rest > 0) {
		l.rest = 0
	}
	l.rollen = neu
}

// Zeige rollt so wenig wie möglich, dass die Kachel i ganz sichtbar ist (Tastaturfokus, offene Aufnahme).
func (l *Leiste) Zeige(i int, m Masse) {
	if i < 0 || i >= len(l.eintraege) {
		return
	}
	x0 := m.Rand + i*(m.Kachel+m.Luecke)
	x1 := x0 + m.Kachel
	r := l.Rollen(m)
	switch {
	case x0-m.Rand < r:
		r = x0 - m.Rand
	case x1+m.Rand > r+m.Breite:
		r = x1 + m.Rand - m.Breite
	}
	l.rollen = max(0, min(r, l.RollMax(m)))
}
