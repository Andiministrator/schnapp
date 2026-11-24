// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Bereichsmodus „Text aus dem Bild kopieren“ (ADR-0035 Punkt 5, Karte #195): Nach dem Menüeintrag bzw. Strg+Umschalt+T
// wählt der nächste Zug mit der linken Taste einen Bereich der Ansicht, Enter das ganze sichtbare Bild, Esc bricht ab.
// Der Modus ist einmalig und liegt über dem Werkzeug: Das Werkzeug bleibt, was es war, und gilt danach wieder. Die
// Erkennung selbst kennt die Sitzung nicht; die Oberfläche holt den gewählten Bereich mit NimmTextBereich ab.

import (
	"image"
	"math"

	"github.com/Andiministrator/schnapp/internal/document"
)

// textBereich ist der Zustand des Bereichsmodus.
type textBereich struct {
	an    bool           // der Modus wartet auf einen Bereich
	zieht bool           // ein Zug mit der linken Taste läuft
	start document.Punkt // Fensterpunkt beim Drücken (Ziehschwelle)
	a, b  document.Punkt // Ecken des Zugs in Ansichtspunkten
	// fertig ist der gewählte Bereich in Pixeln des sichtbaren Bilds, bis NimmTextBereich ihn abholt.
	fertig    image.Rectangle
	hatFertig bool
}

// BeginneTextBereich schaltet den Bereichsmodus ein. Eine Texteingabe wird vorher bestätigt, eine Pipette endet;
// während eines Mauszugs und des Speicherns geht es nicht (false).
func (s *Sitzung) BeginneTextBereich() bool {
	if s.gesperrt() || s.Zieht() {
		return false
	}
	s.TextBestaetigen()
	s.pipette = false
	s.tb = textBereich{an: true}
	return true
}

// TextBereichLaeuft meldet, ob der Bereichsmodus auf einen Bereich wartet (Fadenkreuz, Anleitung in der Statuszeile).
func (s *Sitzung) TextBereichLaeuft() bool { return s.tb.an }

// BeendeTextBereich bricht den Bereichsmodus ab (Esc, Werkzeugwechsel); ein laufender Zug endet ohne Bereich.
func (s *Sitzung) BeendeTextBereich() { s.tb = textBereich{} }

// TextBereichRahmen ist der gerade gezogene Bereich in Fensterpunkten (ok = false, wenn keiner gezogen wird).
func (s *Sitzung) TextBereichRahmen() (a, b document.Punkt, ok bool) {
	if !s.tb.zieht {
		return document.Punkt{}, document.Punkt{}, false
	}
	return s.Abb.ZuFenster(s.tb.a), s.Abb.ZuFenster(s.tb.b), true
}

// TextBereichGanz wählt das ganze sichtbare Bild (Enter im Bereichsmodus); false, wenn der Modus nicht läuft.
func (s *Sitzung) TextBereichGanz() bool {
	if !s.tb.an {
		return false
	}
	s.tb = textBereich{fertig: image.Rectangle{Max: s.D.Groesse()}, hatFertig: true}
	return true
}

// NimmTextBereich liefert einmal den gewählten Bereich in Pixeln des sichtbaren Bilds (render.Bild): Ansichtskoordinaten
// nach Zuschnitt und Stauchen, unabhängig vom Zoom.
func (s *Sitzung) NimmTextBereich() (image.Rectangle, bool) {
	if !s.tb.hatFertig {
		return image.Rectangle{}, false
	}
	r := s.tb.fertig
	s.tb.fertig, s.tb.hatFertig = image.Rectangle{}, false
	return r, true
}

// textBereichDruck beginnt im Bereichsmodus den Zug der linken Taste am Fensterpunkt p; true, wenn der Druck dem
// Modus gehört (dann trifft er kein Objekt und kein Werkzeug).
func (s *Sitzung) textBereichDruck(p document.Punkt, t Taste) bool {
	if !s.tb.an || t != TastePrimaer || s.tb.zieht {
		return false
	}
	q := s.Abb.ZuAnsicht(p)
	s.tb.zieht, s.tb.start, s.tb.a, s.tb.b = true, p, q, q
	return true
}

// textBereichZiehen führt den Zug des Bereichsmodus; true, wenn einer läuft.
func (s *Sitzung) textBereichZiehen(p document.Punkt) bool {
	if !s.tb.zieht {
		return false
	}
	s.tb.b = s.Abb.ZuAnsicht(p)
	return true
}

// textBereichLoslassen beendet den Zug: Liegt ein Bereich mit Fläche im sichtbaren Bild, ist er gewählt und der Modus
// endet; ein Klick (unter der Ziehschwelle) oder ein Zug ganz neben dem Bild lässt den Modus weiter warten.
func (s *Sitzung) textBereichLoslassen(p document.Punkt) bool {
	if !s.tb.zieht {
		return false
	}
	s.tb.b = s.Abb.ZuAnsicht(p)
	s.tb.zieht = false
	if p.Minus(s.tb.start).Laenge() < Ziehschwelle {
		return true
	}
	if r, ok := SichtPixel(s.tb.a, s.tb.b, s.D.Groesse()); ok {
		s.tb = textBereich{fertig: r, hatFertig: true}
	}
	return true
}

// textBereichAbbrechen bricht einen laufenden Zug des Modus ab (Zeiger-Cancel); der Modus wartet weiter.
func (s *Sitzung) textBereichAbbrechen() bool {
	if !s.tb.zieht {
		return false
	}
	s.tb.zieht = false
	return true
}

// SichtPixel rechnet den Bereich zwischen den Ansichtspunkten a und b in ganze Pixel des sichtbaren Bilds der Größe
// groesse um: nach außen gerundet (ein angeschnittenes Zeichen kommt ganz mit) und auf das Bild begrenzt. ok ist
// false, wenn nichts davon im Bild liegt.
func SichtPixel(a, b document.Punkt, groesse image.Point) (image.Rectangle, bool) {
	r := document.R(a, b)
	if math.IsNaN(r.Min.X + r.Min.Y + r.Max.X + r.Max.Y) {
		return image.Rectangle{}, false
	}
	klemme := func(v float64, max int) int { return int(math.Max(0, math.Min(float64(max), v))) }
	px := image.Rect(klemme(math.Floor(r.Min.X), groesse.X), klemme(math.Floor(r.Min.Y), groesse.Y),
		klemme(math.Ceil(r.Max.X), groesse.X), klemme(math.Ceil(r.Max.Y), groesse.Y))
	if px.Empty() {
		return image.Rectangle{}, false
	}
	return px, true
}
