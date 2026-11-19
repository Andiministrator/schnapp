// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/document"
)

// Fensterwerte der Bedienung (Pixel des Fensterpuffers, unabhängig vom Zoom).
const (
	// Toleranz für Treffer auf Objekte und Griffe.
	Toleranz = 6.0
	// Rand beim Einpassen.
	Rand = 24.0
	// Ab dieser Strecke ist ein Druck ein Ziehen (darunter ein Klick).
	Ziehschwelle = 3.0
)

// Taste unterscheidet die Maustaste beim Drücken.
type Taste int

const (
	TastePrimaer Taste = iota // links: auswählen, verschieben, Griffe; auf leerer Fläche verschieben der Ansicht
	TasteMitte                // Mitte: Ansicht verschieben
)

type zugArt int

const (
	keinZug zugArt = iota
	zugObjekt
	zugGriff
	zugAnsicht
)

// Sitzung ist der Zustand des Editors ohne Gio (ADR-0020 Punkt 2): Dokument, Abbildung, Auswahl und laufender Zug.
// Alle Punkte kommen in Fensterkoordinaten. Änderungen am Dokument laufen über dessen Züge und Befehle, damit
// Rückgängig sie als je eine Änderung kennt.
type Sitzung struct {
	D   *document.Dokument
	Abb Abbildung
	// Auswahl ist die Kennung des ausgewählten Objekts, 0 = keins.
	Auswahl int
	// Geaendert ist wahr, sobald seit dem Öffnen oder letzten Speichern etwas am Dokument geändert wurde.
	Geaendert bool

	fenster    image.Point
	eingepasst bool // Abbildung folgt der Fenstergröße, bis der Nutzer zoomt oder verschiebt
	zug        zugArt
	griff      GriffArt
	start      document.Punkt  // Fensterpunkt beim Drücken
	startObj   document.Objekt // Objekt beim Drücken (ZieheGriff/Verschiebe wirken immer darauf)
	startAbb   Abbildung
	bewegt     bool
}

// NeueSitzung beginnt mit eingepasster Ansicht und ohne Auswahl.
func NeueSitzung(d *document.Dokument) *Sitzung {
	return &Sitzung{D: d, Abb: Abbildung{Zoom: 1}, eingepasst: true}
}

// SetzeFenster meldet die Fenstergröße. Solange der Nutzer nicht selbst gezoomt oder verschoben hat, wird neu
// eingepasst (der erste Frame kommt unter Wayland oft in falscher Größe).
func (s *Sitzung) SetzeFenster(g image.Point) {
	s.fenster = g
	if s.eingepasst {
		s.Abb = Einpassen(s.D.Groesse(), g, Rand)
	}
}

// Einpassen zeigt die ganze Ansicht im Fenster (Strg+0).
func (s *Sitzung) Einpassen() {
	s.eingepasst = true
	s.Abb = Einpassen(s.D.Groesse(), s.fenster, Rand)
}

// EinsZuEins zeigt die Ansicht 1:1 um die Fenstermitte (Strg+1).
func (s *Sitzung) EinsZuEins() {
	s.eingepasst = false
	s.Abb = s.Abb.Eins(document.Pt(float64(s.fenster.X)/2, float64(s.fenster.Y)/2))
}

// Scrollen zoomt zum Zeiger: dy < 0 vergrößert (Rad nach oben). Ein Rastschritt (dy = ±1) ändert um 10 %.
func (s *Sitzung) Scrollen(p document.Punkt, dy float64) {
	if dy == 0 || math.IsNaN(dy) || math.IsInf(dy, 0) {
		return
	}
	s.eingepasst = false
	s.Abb = s.Abb.ZoomUm(p, math.Pow(1.1, -math.Max(-10, math.Min(10, dy))))
}

// original rechnet einen Fensterpunkt ins Original um.
func (s *Sitzung) original(p document.Punkt) document.Punkt {
	return s.D.ZuOriginal(s.Abb.ZuAnsicht(p))
}

// toleranz ist die Fenster-Toleranz in Pixeln des Originals.
func (s *Sitzung) toleranz() float64 { return Toleranz / s.Abb.Zoom }

// Druecken beginnt einen Zug: an einem Griff des ausgewählten Objekts, an einem Objekt (wählt es aus) oder auf leerer
// Fläche bzw. mit der mittleren Taste an der Ansicht. Während eines Zugs ist ein weiterer Druck wirkungslos.
func (s *Sitzung) Druecken(p document.Punkt, t Taste) {
	if s.zug != keinZug || s.D.ZugLaeuft() {
		return
	}
	s.start, s.bewegt, s.startAbb = p, false, s.Abb
	if t == TasteMitte {
		s.zug = zugAnsicht
		return
	}
	o := s.D.Objekte()
	q := s.original(p)
	if sel, ok := s.D.Objekt(s.Auswahl); ok {
		if art, ok := GriffTreffer(sel, q, s.toleranz()); ok && s.D.BeginneZug(sel.Kennung()) == nil {
			s.zug, s.griff, s.startObj = zugGriff, art, sel
			return
		}
	}
	if id, ok := Treffer(o, q, s.toleranz()); ok {
		s.Auswahl = id
		obj, _ := s.D.Objekt(id)
		if s.D.BeginneZug(id) == nil {
			s.zug, s.startObj = zugObjekt, obj
		}
		return
	}
	s.Auswahl = 0
	s.zug = zugAnsicht
}

// Ziehen führt den Zug zum Fensterpunkt p.
func (s *Sitzung) Ziehen(p document.Punkt) {
	if s.zug == keinZug {
		return
	}
	if !s.bewegt && p.Minus(s.start).Laenge() < Ziehschwelle {
		return
	}
	s.bewegt = true
	switch s.zug {
	case zugAnsicht:
		s.eingepasst = false
		s.Abb = s.startAbb.Verschoben(p.Minus(s.start))
	case zugObjekt:
		d := s.original(p).Minus(s.original(s.start))
		_ = s.D.ZugSetze(Verschiebe(s.startObj, d))
	case zugGriff:
		_ = s.D.ZugSetze(ZieheGriff(s.startObj, s.griff, s.original(p)))
	}
}

// Loslassen beendet den Zug; ein Objektzug landet als eine Änderung im Verlauf (ohne Bewegung als keine).
func (s *Sitzung) Loslassen(p document.Punkt) {
	s.Ziehen(p)
	if s.zug == zugObjekt || s.zug == zugGriff {
		if s.bewegt {
			vorher := s.startObj
			_ = s.D.BeendeZug()
			if jetzt, ok := s.D.Objekt(vorher.Kennung()); ok && jetzt != vorher {
				s.Geaendert = true
			}
		} else {
			_ = s.D.BrecheZugAb()
		}
	}
	s.zug = keinZug
}

// Abbrechen stellt den Stand vor dem Zug wieder her (Esc während des Ziehens, Zeiger-Cancel).
func (s *Sitzung) Abbrechen() {
	switch s.zug {
	case zugObjekt, zugGriff:
		_ = s.D.BrecheZugAb()
	case zugAnsicht:
		s.Abb = s.startAbb
	}
	s.zug = keinZug
}

// Zieht meldet einen laufenden Zug.
func (s *Sitzung) Zieht() bool { return s.zug != keinZug }

// Rueckgaengig nimmt die letzte Änderung zurück (nicht während eines Zugs, F-18). Verschwindet dabei das
// ausgewählte Objekt, endet die Auswahl.
func (s *Sitzung) Rueckgaengig() bool { return s.verlauf(s.D.Rueckgaengig) }

// Wiederholen stellt die zuletzt zurückgenommene Änderung wieder her.
func (s *Sitzung) Wiederholen() bool { return s.verlauf(s.D.Wiederholen) }

func (s *Sitzung) verlauf(f func() error) bool {
	if s.zug != keinZug || f() != nil {
		return false
	}
	s.Geaendert = true
	if _, ok := s.D.Objekt(s.Auswahl); !ok {
		s.Auswahl = 0
	}
	return true
}

// Loeschen entfernt das ausgewählte Objekt.
func (s *Sitzung) Loeschen() bool {
	if s.zug != keinZug || s.Auswahl == 0 || s.D.Entfernen(s.Auswahl) != nil {
		return false
	}
	s.Auswahl, s.Geaendert = 0, true
	return true
}

// Abwaehlen hebt die Auswahl auf (Esc ohne Zug).
func (s *Sitzung) Abwaehlen() { s.Auswahl = 0 }

// Gespeichert meldet, dass der aktuelle Stand gespeichert ist.
func (s *Sitzung) Gespeichert() { s.Geaendert = false }
