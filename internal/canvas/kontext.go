// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Kontextmenü der Leinwand (ADR-0038 Punkt 2): Der Rechtsklick wählt das Ziel, die Oberfläche zeigt die Einträge dazu.

// KontextBei bereitet das Kontextmenü am Fensterpunkt p vor: Eine laufende Texteingabe wird bestätigt, das Objekt unter
// dem Zeiger gewählt (dieselbe Trefferregel wie das Auswählen, gleich welches Werkzeug aktiv ist), sonst die Auswahl
// aufgehoben; Pipette und Bereichsmodus der Texterkennung enden. offen: das Menü darf aufgehen (nicht während eines
// Zugs, des Speicherns oder der Ruhe beim Wechsel);
// objekt: es gilt einem Objekt. Zuschneiden und Stauchen wählen keine Objekte (ohneGriffe), dort gilt es dem Bild und
// die Auswahl bleibt.
func (s *Sitzung) KontextBei(p document.Punkt) (offen, objekt bool) {
	if s.Zieht() || s.gesperrt() {
		return false, false
	}
	s.TextBestaetigen()
	// Pipette und Bereichsmodus der Texterkennung enden wie mit Esc: Sonst wirkte der nächste Linksklick als Pipette auf
	// das per Rechtsklick gewählte Objekt bzw. zöge einen Bereich hinter dem Menü (Kritik K-3 #313a).
	s.pipette = false
	s.BeendeTextBereich()
	s.abgelehnt = AblehnungKeine // keine alte Ablehnungsmeldung wiederholen (K-5)
	if ohneGriffe(s.Werkzeug) {
		return true, false // die Auswahl bleibt, wie bei KontextAnker (K-7)
	}
	if id, ok := Treffer(s.sichtbareObjekte(), s.original(p), s.toleranz()); ok {
		// Ein schon gewähltes wird Hauptauswahl, die übrigen bleiben gewählt (ADR-0039 Punkt 3, F-898); ein anderes
		// steht danach allein, auch wenn es früher Hauptauswahl war (F-895).
		s.waehleHaupt(id)
		s.lupenTeil = LupeGanz
		return true, true
	}
	s.waehleNur(0)
	return true, false
}

// KontextAnker ist der Fensterpunkt, an dem das Kontextmenü per Tastatur (Umschalt+F10) aufgeht: die Mitte des gewählten
// Objekts, sonst die Mitte der Leinwand; objekt meldet, ob es einem Objekt gilt (ADR-0038 Punkt 6).
func (s *Sitzung) KontextAnker() (p document.Punkt, objekt bool) {
	if o, ok := s.D.Objekt(s.Auswahl); ok && !ohneGriffe(s.Werkzeug) {
		g := o.Grenzen()
		m := document.Pt((g.Min.X+g.Max.X)/2, (g.Min.Y+g.Max.Y)/2)
		return s.Abb.ZuFenster(s.D.ZuAnsicht(m)), true
	}
	return document.Pt(float64(s.fenster.X)/2, float64(s.fenster.Y)/2), false
}
