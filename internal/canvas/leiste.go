// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "image"

// LeistenMass sind die Maße der Werkzeugleiste in Fensterpixeln (F-148).
type LeistenMass struct {
	RandX      int // links und rechts
	RandY      int // oben und unten
	Abstand    int // zwischen zwei Knöpfen einer Zeile und zwischen zwei Zeilen; vor einer Gruppe zusätzlich
	KnopfHoehe int
	Speichern  int // Breite des Knopfs „Speichern“, rechts in der ersten Zeile
	Breite     int // verfügbare Breite des Fensters
}

// LeistenKnopf ist ein Knopf der linken Seite: seine Breite und ob er eine neue Gruppe beginnt.
type LeistenKnopf struct {
	Breite int
	Gruppe bool
}

// LeisteAnordnen legt die Knöpfe der Reihe nach in Zeilen. Eine Zeile nimmt Knöpfe, solange der nächste samt Rand
// noch hineinpasst; die erste Zeile endet vor „Speichern“ (samt Abstand), die weiteren am rechten Rand. Ein Knopf
// wird nie geteilt: Ist einer breiter als eine ganze Zeile, steht er allein am Zeilenanfang und ragt über (mehr
// kann die Leiste nicht tun). Eine Gruppe, die am Zeilenanfang steht, bekommt dort keinen Zusatzabstand.
// Rückgabe: die linke obere Ecke je Knopf (in der Reihenfolge der Eingabe), die Höhe der ganzen Leiste samt
// Rändern oben und unten und die Zahl der Zeilen. Ohne Knöpfe bleibt eine Zeile (Höhe wie mit einem Knopf).
func LeisteAnordnen(knoepfe []LeistenKnopf, m LeistenMass) (pos []image.Point, hoehe, zeilen int) {
	pos = make([]image.Point, len(knoepfe))
	zeilen = 1
	x := m.RandX
	anfang := true
	// Grenze der aktuellen Zeile: rechts der Rand, in der ersten Zeile davor noch „Speichern“ und ein Abstand.
	grenze := m.Breite - m.RandX - m.Speichern - m.Abstand
	for i, k := range knoepfe {
		zusatz := 0
		if k.Gruppe && !anfang {
			zusatz = m.Abstand
		}
		if !anfang && x+zusatz+k.Breite > grenze {
			// Umbruch: neue Zeile, volle Breite bis zum rechten Rand.
			zeilen++
			x = m.RandX
			zusatz = 0
			grenze = m.Breite - m.RandX
		}
		x += zusatz
		anfang = false
		pos[i] = image.Pt(x, m.RandY+(zeilen-1)*(m.KnopfHoehe+m.Abstand))
		x += k.Breite + m.Abstand
	}
	hoehe = 2*m.RandY + zeilen*m.KnopfHoehe + (zeilen-1)*m.Abstand
	return pos, hoehe, zeilen
}
