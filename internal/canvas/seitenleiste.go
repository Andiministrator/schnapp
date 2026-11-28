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
	Spitze         bool // Schalter „Spitze“ (gewählter Text oder Sprechblase, ADR-0021 Nachtrag 13 Punkt 6)
	Vergroesserung bool // Vergrößerung wählbar (gewählte Lupe, ADR-0033 Punkt 8)
	Pfeilform      bool // Spitzenart und Enden wählbar (Werkzeug oder gewählter Pfeil, ADR-0021 Nachtrag 19)
	Freihand       bool // Kästchen „Freehand“ (Werkzeug Textmarker ohne gewähltes Objekt, ADR-0049 Punkt 5)
	// Anzahl ist die Zahl der gewählten Objekte, wenn mehrere gewählt sind (sonst 0): Der Kopf nennt sie, und die
	// Abschnitte sind die aller gewählten Arten (ADR-0039 Nachtrag 4 Punkt 4); Werkzeug ist das der Hauptauswahl.
	Anzahl  int
	Hinweis SeitenHinweis
}

// objektWerkzeug ist das Werkzeug, das o zeichnet (false: keines, etwa eine unbekannte Art).
func objektWerkzeug(o document.Objekt) (Werkzeug, bool) {
	switch v := o.(type) {
	case document.Pfeil:
		if v.Multipliziert() { // ein multiplizierender Pfeil ist ein Textmarker (ADR-0048 Punkt 2)
			return WerkzeugTextmarker, true
		}
		return WerkzeugPfeil, true
	case document.Freihand:
		if v.Multipliziert() { // freier Textmarker (ADR-0049 Punkt 5)
			return WerkzeugTextmarker, true
		}
		return WerkzeugFreihand, true
	case document.Kasten:
		if v.Form == document.FormEllipse {
			return WerkzeugEllipse, true
		}
		return WerkzeugKasten, true
	case document.Text:
		if v.Spitze != nil { // ein Text mit Spitze ist eine Sprechblase (ADR-0021 Nachtrag 13)
			return WerkzeugSprechblase, true
		}
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
	return w == WerkzeugPfeil || w == WerkzeugKasten || w == WerkzeugText || w == WerkzeugNummer || w == WerkzeugLupe ||
		w == WerkzeugSprechblase || w == WerkzeugEllipse || w == WerkzeugTextmarker || w == WerkzeugFreihand
}

// textWerkzeug meldet, ob w Text setzt (Text, Sprechblase): Schrift, Größe, Stil und Ausrichtung gelten dafür.
func textWerkzeug(w Werkzeug) bool { return w == WerkzeugText || w == WerkzeugSprechblase }

// strichWerkzeug meldet, ob ein Werkzeug einen Strich zeichnet (Pfeil, Kasten, Ellipse, Lupe, Textmarker).
func strichWerkzeug(w Werkzeug) bool {
	return w == WerkzeugPfeil || w == WerkzeugKasten || w == WerkzeugEllipse || w == WerkzeugLupe ||
		w == WerkzeugTextmarker || w == WerkzeugFreihand
}

// Seitenleiste liefert den Inhalt der rechten Leiste. Ein gewähltes Objekt geht dem Werkzeug vor: Seine Farbe ändert
// die Farbwahl (SetzeFarbe färbt es um). Ohne Objekt zeigt Auswählen keine Farbe, denn sie wirkte auf nichts.
func (s *Sitzung) Seitenleiste() SeitenInhalt {
	if s.Auswahl != 0 {
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			if w, ok := objektWerkzeug(o); ok {
				i := SeitenInhalt{Auswahl: true, Werkzeug: w, Farbe: true,
					Strich: strichWerkzeug(w), Text: textWerkzeug(w), Spitze: textWerkzeug(w), Vergroesserung: w == WerkzeugLupe,
					Pfeilform: w == WerkzeugPfeil}
				if s.stilMehrere() { // Abschnitte aller gewählten Arten; Spitze und Vergrößerung bleiben der Hauptauswahl
					i.Anzahl = s.AnzahlGewaehlt()
					for _, g := range s.gewaehlteObjekte() {
						if gw, ok := objektWerkzeug(g); ok {
							i.Strich = i.Strich || strichWerkzeug(gw)
							i.Text = i.Text || textWerkzeug(gw)
							i.Pfeilform = i.Pfeilform || gw == WerkzeugPfeil
						}
					}
				}
				return i
			}
		}
	}
	i := SeitenInhalt{Werkzeug: s.Werkzeug, Farbe: farbWerkzeug(s.Werkzeug),
		Strich: strichWerkzeug(s.Werkzeug), Text: textWerkzeug(s.Werkzeug), Pfeilform: s.Werkzeug == WerkzeugPfeil,
		Freihand: s.Werkzeug == WerkzeugTextmarker}
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
