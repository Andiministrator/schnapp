// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/document"
)

// Werkzeug ist das aktive Werkzeug des Editors (ADR-0021).
type Werkzeug int

const (
	WerkzeugAuswahl Werkzeug = iota // auswählen, verschieben, Griffe
	WerkzeugPfeil                   // Ziehen vom Anfang zur Spitze
	WerkzeugKasten                  // Ziehen von Ecke zu Ecke
	WerkzeugText                    // Klick setzt ein Textfeld
	WerkzeugNummer                  // Klick setzt eine Schritt-Nummer
)

// MinObjekt ist die Mindestgröße eines neuen Pfeils (Länge) oder Kastens (jede Seite) in Fensterpixeln: Kleiner
// bleibt beim Loslassen nichts liegen — ein versehentlicher Klick legt kein leeres Objekt an (ADR-0021).
const MinObjekt = 6.0

// Beschriftungsfarben sind die Farben der ersten Fassung (ADR-0021 Punkt 4): kräftig genug, dass weiße Schrift
// darauf lesbar bleibt. Die erste ist die Voreinstellung.
var Beschriftungsfarben = []color.NRGBA{
	{R: 0xD9, G: 0x3A, B: 0x33, A: 0xFF}, // Rot
	{R: 0x2F, G: 0x6F, B: 0xDB, A: 0xFF}, // Blau
	{R: 0x2E, G: 0x8B, B: 0x57, A: 0xFF}, // Grün
	{R: 0x1E, G: 0x21, B: 0x24, A: 0xFF}, // Dunkelgrau (grau-900)
}

// Weiss ist die Schriftfarbe auf Text-Schildern (Nummern zeichnen Renderer und Export ohnehin weiß).
var Weiss = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

// Stil sind die Eigenschaften neuer Objekte, Größen in Pixeln des Originals.
type Stil struct {
	Farbe       color.NRGBA
	Breite      float64 // Strich von Pfeil und Kasten
	Eckradius   float64 // Kasten
	Schriftgrad float64 // Text
	Radius      float64 // Nummer
}

// StandardStil ist die Voreinstellung (ADR-0021 Punkt 4).
func StandardStil() Stil {
	return Stil{Farbe: Beschriftungsfarben[0], Breite: 4, Eckradius: 4, Schriftgrad: 20, Radius: 14}
}

func (st Stil) strich() document.Strich { return document.Strich{Farbe: st.Farbe, Breite: st.Breite} }

// Rasten45 liefert b so gedreht um a, dass die Richtung a→b ein Vielfaches von 45° ist; die Länge bleibt.
func Rasten45(a, b document.Punkt) document.Punkt {
	d := b.Minus(a)
	l := d.Laenge()
	if l == 0 {
		return b
	}
	w := math.Round(math.Atan2(d.Y, d.X)/(math.Pi/4)) * (math.Pi / 4)
	return a.Plus(document.Pt(l*math.Cos(w), l*math.Sin(w)))
}

// Quadrat liefert die Ecke gegenüber a eines Quadrats in Richtung b; die Seite ist die größere der beiden Strecken.
func Quadrat(a, b document.Punkt) document.Punkt {
	d := b.Minus(a)
	s := math.Max(math.Abs(d.X), math.Abs(d.Y))
	return a.Plus(document.Pt(math.Copysign(s, d.X), math.Copysign(s, d.Y)))
}

// Umgefaerbt liefert o in der Farbe c: Strich bei Pfeil und Kasten, Schild beim Text, Kreis bei der Nummer.
func Umgefaerbt(o document.Objekt, c color.NRGBA) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		v.Strich.Farbe = c
		return v
	case document.Kasten:
		v.Strich.Farbe = c
		return v
	case document.Text:
		v.Hintergrund = c
		return v
	case document.Nummer:
		v.Farbe = c
		return v
	}
	return o
}

// --- Werkzeug und Farbe ---

// SetzeWerkzeug wechselt das Werkzeug. Eine Texteingabe wird vorher bestätigt; während eines Mauszugs geht es nicht.
func (s *Sitzung) SetzeWerkzeug(w Werkzeug) bool {
	if s.Zieht() {
		return false
	}
	s.TextBestaetigen()
	s.Werkzeug = w
	return true
}

// SetzeFarbe setzt die Farbe für folgende Objekte und färbt das ausgewählte Objekt um (ein Rückgängig-Schritt);
// während der Texteingabe gehört das Umfärben zur Eingabe. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeFarbe(c color.NRGBA) bool {
	if s.Zieht() {
		return false
	}
	s.Stil.Farbe = c
	if s.zug == zugText {
		if o, ok := s.D.Objekt(s.textID); ok {
			_ = s.D.ZugSetze(Umgefaerbt(o, c))
		}
		return false
	}
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok {
		return false
	}
	neu := Umgefaerbt(o, c)
	if neu == o {
		return false
	}
	return s.D.Ersetzen(neu) == nil
}

// NaechsteFarbe schaltet reihum zur nächsten Beschriftungsfarbe (Knopf „Farbe“).
func (s *Sitzung) NaechsteFarbe() bool {
	i := 0
	for j, c := range Beschriftungsfarben {
		if c == s.Stil.Farbe {
			i = j + 1
			break
		}
	}
	return s.SetzeFarbe(Beschriftungsfarben[i%len(Beschriftungsfarben)])
}

// --- Anlegen ---

// beginneNeu startet mit einem Zeichenwerkzeug an q (Original). Pfeil und Kasten entstehen erst beim Ziehen über die
// Schwelle, Nummer und Text sofort.
func (s *Sitzung) beginneNeu(q document.Punkt) {
	s.neuArt, s.neuID = s.Werkzeug, 0
	switch s.Werkzeug {
	case WerkzeugPfeil, WerkzeugKasten:
		s.zug = zugNeu
	case WerkzeugNummer:
		n := document.Nummer{Mitte: q, Wert: s.D.NaechsteNummer(), Radius: s.Stil.Radius, Farbe: s.Stil.Farbe}
		id, err := s.D.BeginneNeuenZug(n)
		if err != nil {
			return
		}
		n.ID = id
		s.zug, s.neuID, s.startObj = zugNeu, id, n
	case WerkzeugText:
		if id, ok := Treffer(s.D.Objekte(), q, s.toleranz()); ok {
			if o, _ := s.D.Objekt(id); isText(o) {
				s.Auswahl = id
				s.TextBearbeiten()
				return
			}
		}
		t := document.Text{Ort: q, Schriftgrad: s.Stil.Schriftgrad, Farbe: Weiss, Hintergrund: s.Stil.Farbe}
		id, err := s.D.BeginneNeuenZug(t)
		if err != nil {
			return
		}
		s.zug, s.textID, s.textNeu, s.Auswahl = zugText, id, true, id
	}
}

func isText(o document.Objekt) bool { _, ok := o.(document.Text); return ok }

// zieheNeu setzt das neue Objekt auf den Fensterpunkt p.
func (s *Sitzung) zieheNeu(p document.Punkt) {
	a, b := s.original(s.start), s.original(p)
	var o document.Objekt
	switch s.neuArt {
	case WerkzeugPfeil:
		if s.Umschalt {
			b = Rasten45(a, b)
		}
		pf := document.GeraderPfeil(a, b, s.Stil.strich())
		pf.ID = s.neuID
		o = pf
	case WerkzeugKasten:
		if s.Umschalt {
			b = Quadrat(a, b)
		}
		o = document.Kasten{ID: s.neuID, Bereich: document.R(a, b), Strich: s.Stil.strich(), Eckradius: s.Stil.Eckradius}
	case WerkzeugNummer:
		o = Verschiebe(s.startObj, b.Minus(a))
	default:
		return
	}
	if s.neuID == 0 {
		id, err := s.D.BeginneNeuenZug(o)
		if err == nil {
			s.neuID = id
		}
		return
	}
	_ = s.D.ZugSetze(o)
}

// beendeNeu übernimmt das neue Objekt als einen Schritt in den Verlauf und wählt es aus — Pfeil und Kasten nur, wenn
// sie die Mindestgröße erreichen.
func (s *Sitzung) beendeNeu() {
	if s.neuID == 0 {
		return // Klick ohne Ziehen: kein Pfeil, kein Kasten
	}
	o, _ := s.D.Objekt(s.neuID)
	min := MinObjekt / s.Abb.Zoom
	gross := true
	switch v := o.(type) {
	case document.Pfeil:
		gross = v.B.Minus(v.A).Laenge() >= min
	case document.Kasten:
		gross = v.Bereich.Breite() >= min && v.Bereich.Hoehe() >= min
	}
	if !gross {
		_ = s.D.BrecheZugAb()
		return
	}
	if s.D.BeendeZug() == nil {
		s.Auswahl = s.neuID
	}
}

// --- Texteingabe ---

// TextLaeuft meldet eine laufende Texteingabe.
func (s *Sitzung) TextLaeuft() bool { return s.zug == zugText }

// TextObjekt liefert den Text, der gerade eingegeben wird (für die Einfügemarke).
func (s *Sitzung) TextObjekt() (document.Text, bool) {
	if s.zug != zugText {
		return document.Text{}, false
	}
	o, ok := s.D.Objekt(s.textID)
	t, ist := o.(document.Text)
	return t, ok && ist
}

// TextBearbeiten beginnt die Eingabe am ausgewählten Text (Doppelklick, Enter).
func (s *Sitzung) TextBearbeiten() bool {
	if s.zug != keinZug {
		return false
	}
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok || !isText(o) || s.D.BeginneZug(s.Auswahl) != nil {
		return false
	}
	s.zug, s.textID, s.textNeu = zugText, s.Auswahl, false
	return true
}

// Tippe hängt Text an (Steuerzeichen fallen weg, Zeilen beginnt TextZeilenumbruch). Über der Höchstlänge bleibt der
// Text unverändert.
func (s *Sitzung) Tippe(text string) bool {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	return s.aendereText(func(alt string) string { return alt + text })
}

// TextZeilenumbruch beginnt eine neue Zeile (Umschalt+Enter).
func (s *Sitzung) TextZeilenumbruch() bool {
	return s.aendereText(func(alt string) string { return alt + "\n" })
}

// TextZurueck löscht das letzte Zeichen (Rücktaste).
func (s *Sitzung) TextZurueck() bool {
	return s.aendereText(func(alt string) string {
		_, n := utf8.DecodeLastRuneInString(alt)
		return alt[:len(alt)-n]
	})
}

func (s *Sitzung) aendereText(f func(string) string) bool {
	t, ok := s.TextObjekt()
	if !ok {
		return false
	}
	neu := f(t.Inhalt)
	if neu == t.Inhalt {
		return false
	}
	t.Inhalt = neu
	return s.D.ZugSetze(t) == nil
}

// TextBestaetigen beendet die Eingabe (Enter, Klick daneben, Werkzeugwechsel, Speichern, Schließen). Ein leerer Text
// (auch nur Leerraum) wird nicht angelegt; ein vorhandener, leer bestätigter Text wird entfernt — je ein Schritt.
func (s *Sitzung) TextBestaetigen() bool {
	t, ok := s.TextObjekt()
	if !ok {
		return false
	}
	s.zug = keinZug
	if strings.TrimSpace(t.Inhalt) != "" {
		if s.D.BeendeZug() == nil {
			s.Auswahl = s.textID
		}
		return true
	}
	_ = s.D.BrecheZugAb()
	if !s.textNeu {
		_ = s.D.Entfernen(s.textID)
	}
	s.Auswahl = 0
	return true
}

// TextAbbrechen verwirft die Eingabe (Esc): Ein neuer Text verschwindet, ein vorhandener bleibt wie vorher.
func (s *Sitzung) TextAbbrechen() bool {
	if s.zug != zugText {
		return false
	}
	s.zug = keinZug
	_ = s.D.BrecheZugAb()
	if s.textNeu {
		s.Auswahl = 0
	}
	return true
}

// --- Tasten mit Stufen ---

// Esc wirkt in Stufen (ADR-0021 Punkt 2): Texteingabe verwerfen, laufendes Ziehen abbrechen, Auswahl aufheben, zurück
// zum Werkzeug Auswählen.
func (s *Sitzung) Esc() {
	switch {
	case s.zug == zugText:
		s.TextAbbrechen()
	case s.zug != keinZug:
		s.Abbrechen()
	case s.Auswahl != 0:
		s.Auswahl = 0
	default:
		s.Werkzeug = WerkzeugAuswahl
	}
}

// Enter bestätigt die Texteingabe (mit umschalt: neue Zeile) oder beginnt sie am ausgewählten Text.
func (s *Sitzung) Enter(umschalt bool) bool {
	if s.zug == zugText {
		if umschalt {
			return s.TextZeilenumbruch()
		}
		return s.TextBestaetigen()
	}
	return s.TextBearbeiten()
}

// Doppelklick am Fensterpunkt p: Mit dem Werkzeug Auswählen bearbeitet er den getroffenen Text; sonst und daneben
// wirkt er wie ein Druck der linken Taste (der erste Klick hat dann schon gewirkt, ADR-0021).
func (s *Sitzung) Doppelklick(p document.Punkt) {
	s.TextBestaetigen()
	if s.Werkzeug == WerkzeugAuswahl && s.zug == keinZug && !s.D.ZugLaeuft() {
		if id, ok := Treffer(s.D.Objekte(), s.original(p), s.toleranz()); ok {
			if o, _ := s.D.Objekt(id); isText(o) {
				s.Auswahl = id
				s.TextBearbeiten()
				return
			}
		}
	}
	s.Druecken(p, TastePrimaer)
}
