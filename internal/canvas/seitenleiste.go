// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Inhalt der rechten Leiste ohne Gio (ADR-0031 Punkt 3, Nachtrag 3): Die Oberfläche setzt die Texte, hier steht nur,
// was die Leiste zeigt.

// SeitenHinweis ist ein Hinweis unter den Eigenschaften (Nebentext), ohne Text: Die Oberfläche übersetzt ihn.
type SeitenHinweis int

const (
	SeitenHinweisKeiner     SeitenHinweis = iota
	SeitenHinweisNichts                   // Auswählen ohne gewähltes Objekt: nichts ausgewählt
	SeitenHinweisEndgueltig               // Schwärzen, Verpixeln: Pixel werden beim Speichern endgültig überschrieben
	SeitenHinweisOriginal                 // Zuschneiden, Stauchen: das Weggenommene bleibt in der Arbeitsdatei
	SeitenHinweisKeineNaht                // Stauchen ohne Naht: die Kantenform wirkt erst mit einer Naht
)

// SeitenInhalt beschreibt die rechte Leiste.
type SeitenInhalt struct {
	// Auswahl: Der Kopf nennt das gewählte Objekt („Auswahl: Pfeil“), sonst das Werkzeug („Werkzeug: Text“). Werkzeug
	// ist dann das Werkzeug, das diese Art Objekt zeichnet; sein Name ist der Name der Art.
	Auswahl        bool
	Werkzeug       Werkzeug
	Farbe          bool // Abschnitt Farbe: Werkzeug oder Objekt hat eine Farbe
	Strich         bool // Strichstärke wählbar (Pfeil, Kasten; ADR-0021 Nachtrag 7)
	Text           bool // Schrift, Größe, Stil und Ausrichtung wählbar (Text; ADR-0021 Nachtrag 10)
	Kante          bool // Kantenform wählbar (Stauchen)
	KanteAn        bool // die Knöpfe der Kantenform wirken (KannKanteWechseln)
	Vergroesserung bool // Vergrößerung wählbar (gewählte Lupe, ADR-0033 Punkt 8)
	Hinweis        SeitenHinweis
}

// objektWerkzeug ist das Werkzeug, das o zeichnet (false: keines, etwa eine unbekannte Art).
func objektWerkzeug(o document.Objekt) (Werkzeug, bool) {
	switch o.(type) {
	case document.Pfeil:
		return WerkzeugPfeil, true
	case document.Kasten:
		return WerkzeugKasten, true
	case document.Text:
		return WerkzeugText, true
	case document.Nummer:
		return WerkzeugNummer, true
	case document.Lupe:
		return WerkzeugLupe, true
	}
	return WerkzeugAuswahl, false
}

// farbWerkzeug meldet, ob ein Werkzeug eine Beschriftungsfarbe verwendet.
func farbWerkzeug(w Werkzeug) bool {
	return w == WerkzeugPfeil || w == WerkzeugKasten || w == WerkzeugText || w == WerkzeugNummer || w == WerkzeugLupe
}

// strichWerkzeug meldet, ob ein Werkzeug einen Strich zeichnet (Pfeil, Kasten, Lupe).
func strichWerkzeug(w Werkzeug) bool {
	return w == WerkzeugPfeil || w == WerkzeugKasten || w == WerkzeugLupe
}

// Seitenleiste liefert den Inhalt der rechten Leiste. Ein gewähltes Objekt geht dem Werkzeug vor: Seine Farbe ändert
// die Farbwahl (SetzeFarbe färbt es um). Ohne Objekt zeigt Auswählen keine Farbe, denn sie wirkte auf nichts.
func (s *Sitzung) Seitenleiste() SeitenInhalt {
	if s.Auswahl != 0 {
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			if w, ok := objektWerkzeug(o); ok {
				return SeitenInhalt{Auswahl: true, Werkzeug: w, Farbe: true,
					Strich: strichWerkzeug(w), Text: w == WerkzeugText, Vergroesserung: w == WerkzeugLupe}
			}
		}
	}
	i := SeitenInhalt{Werkzeug: s.Werkzeug, Farbe: farbWerkzeug(s.Werkzeug),
		Strich: strichWerkzeug(s.Werkzeug), Text: s.Werkzeug == WerkzeugText}
	switch s.Werkzeug {
	case WerkzeugAuswahl:
		i.Hinweis = SeitenHinweisNichts
	case WerkzeugSchwaerzen, WerkzeugVerpixeln:
		i.Hinweis = SeitenHinweisEndgueltig
	case WerkzeugZuschneiden:
		i.Hinweis = SeitenHinweisOriginal
	case WerkzeugStauchen:
		i.Kante, i.KanteAn = true, s.KannKanteWechseln()
		i.Hinweis = SeitenHinweisOriginal
		if len(s.D.Naehte()) == 0 {
			i.Hinweis = SeitenHinweisKeineNaht
		}
	}
	return i
}
