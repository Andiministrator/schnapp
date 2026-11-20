// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
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
	WerkzeugAuswahl    Werkzeug = iota // auswählen, verschieben, Griffe
	WerkzeugPfeil                      // Ziehen vom Anfang zur Spitze
	WerkzeugKasten                     // Ziehen von Ecke zu Ecke
	WerkzeugText                       // Klick setzt ein Textfeld
	WerkzeugNummer                     // Klick setzt eine Schritt-Nummer
	WerkzeugSchwaerzen                 // Rahmen ziehen: Pixel schwarz überschreiben (ADR-0022)
	WerkzeugVerpixeln                  // Rahmen ziehen: Pixel verpixeln mit Rauschen (ADR-0022)
)

// Werte des Verpixelns (ADR-0022 Punkt 2), Größen in Pixeln des Originals.
const (
	VerpixelnBlock    = 12
	VerpixelnRauschen = 24
)

// Schwaerzfarbe füllt beim Schwärzen.
var Schwaerzfarbe = color.NRGBA{A: 0xFF}

// schwaerzt meldet, ob w ein Werkzeug ist, das Pixel überschreibt statt ein Objekt anzulegen.
func schwaerzt(w Werkzeug) bool { return w == WerkzeugSchwaerzen || w == WerkzeugVerpixeln }

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
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
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
	// Klick auf einen vorhandenen Text bearbeitet ihn auch bei vollem Dokument (kein neues Objekt nötig).
	if s.Werkzeug == WerkzeugText {
		if id, ok := Treffer(s.D.Objekte(), q, s.toleranz()); ok {
			if o, _ := s.D.Objekt(id); isText(o) {
				s.Auswahl = id
				s.TextBearbeiten()
				return
			}
		}
	}
	if a := s.Voll(); a != AblehnungKeine {
		s.abgelehnt = a // sichtbare Meldung statt stillem Scheitern (F-154)
		return
	}
	switch s.Werkzeug {
	case WerkzeugPfeil, WerkzeugKasten:
		s.zug = zugNeu
	case WerkzeugSchwaerzen, WerkzeugVerpixeln:
		s.zug, s.rahmen = zugNeu, false
	case WerkzeugNummer:
		n := document.Nummer{Mitte: q, Wert: s.D.NaechsteNummer(), Radius: s.Stil.Radius, Farbe: s.Stil.Farbe}
		id, err := s.D.BeginneNeuenZug(n)
		if err != nil {
			return
		}
		n.ID = id
		s.zug, s.neuID, s.startObj = zugNeu, id, n
	case WerkzeugText:
		t := document.Text{Ort: q, Schriftgrad: s.Stil.Schriftgrad, Farbe: Weiss, Hintergrund: s.Stil.Farbe}
		id, err := s.D.BeginneNeuenZug(t)
		if err != nil {
			return
		}
		s.zug, s.textID, s.textNeu, s.Auswahl = zugText, id, true, id
	}
}

func isText(o document.Objekt) bool { _, ok := o.(document.Text); return ok }

// Ablehnung nennt den Grund, aus dem ein Zeichenwerkzeug nichts anlegt (F-154).
type Ablehnung int

const (
	AblehnungKeine   Ablehnung = iota
	AblehnungObjekte           // das Dokument hat document.MaxObjekte Objekte
	AblehnungNummer            // die nächste Nummer wäre größer als document.MaxNummer
)

// Voll meldet, warum das aktive Zeichenwerkzeug jetzt kein neues Objekt anlegen kann (AblehnungKeine, wenn es geht).
// Die Statusleiste verspricht damit keine Nummer, die nicht kommt.
func (s *Sitzung) Voll() Ablehnung {
	switch {
	case s.Werkzeug == WerkzeugAuswahl, schwaerzt(s.Werkzeug):
		return AblehnungKeine // Schwärzen legt kein Objekt an
	case len(s.D.Objekte()) >= document.MaxObjekte:
		return AblehnungObjekte
	case s.Werkzeug == WerkzeugNummer && s.D.NaechsteNummer() > document.MaxNummer:
		return AblehnungNummer
	}
	return AblehnungKeine
}

// Abgelehnt liefert den Grund, aus dem der letzte Druck kein Objekt angelegt hat (AblehnungKeine sonst). Die
// Oberfläche zeigt dazu eine Meldung.
func (s *Sitzung) Abgelehnt() Ablehnung { return s.abgelehnt }

// zieheNeu setzt das neue Objekt auf den Fensterpunkt p. Der Endpunkt bleibt im sichtbaren Bild (F-150).
func (s *Sitzung) zieheNeu(p document.Punkt) {
	a := s.original(s.start)
	b := s.original(s.Abb.ZuFenster(s.klemmeAnsicht(s.Abb.ZuAnsicht(p))))
	var o document.Objekt
	switch s.neuArt {
	case WerkzeugPfeil:
		if s.Umschalt {
			b = s.umschaltEnde(WerkzeugPfeil, p)
		}
		pf := document.GeraderPfeil(a, b, s.Stil.strich())
		pf.ID = s.neuID
		o = pf
	case WerkzeugKasten:
		if s.Umschalt {
			b = s.umschaltEnde(WerkzeugKasten, p)
		}
		o = document.Kasten{ID: s.neuID, Bereich: document.R(a, b), Strich: s.Stil.strich(), Eckradius: s.Stil.Eckradius}
	case WerkzeugNummer:
		o = Verschiebe(s.startObj, b.Minus(a))
	case WerkzeugSchwaerzen, WerkzeugVerpixeln:
		s.rahmen = true
		s.rahmenA = s.klemmeAnsicht(s.Abb.ZuAnsicht(s.start))
		s.rahmenB = s.klemmeAnsicht(s.Abb.ZuAnsicht(p))
		return
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

// umschaltEnde liefert den Endpunkt (Original) eines Pfeils oder Kastens mit gedrückter Umschalttaste. Gerastet wird in
// der Ansicht (nach Zuschnitt und Stauchen), nicht im Original: Der Endpunkt liegt so nie in einem herausgeschnittenen
// Streifen (F-157, ADR-0021 Nachtrag 1). Am Bildrand wird nicht je Achse geklemmt, sondern entlang der Richtung
// gekürzt (F-156): Der Pfeil bleibt auf 45°, der Kasten ein Quadrat mit der Seite, die der Platz zulässt.
func (s *Sitzung) umschaltEnde(art Werkzeug, p document.Punkt) document.Punkt {
	g := s.D.Groesse()
	w, h := float64(g.X), float64(g.Y)
	a := s.klemmeAnsicht(s.Abb.ZuAnsicht(s.start))
	b := s.klemmeAnsicht(s.Abb.ZuAnsicht(p))
	var e document.Punkt
	if art == WerkzeugPfeil {
		e = Rasten45(a, b)
		d := e.Minus(a)
		t := math.Min(1, math.Min(platzAchse(d.X, a.X, w), platzAchse(d.Y, a.Y, h)))
		e = a.Plus(d.Mal(t))
	} else {
		e = Quadrat(a, b)
		d := e.Minus(a)
		if d.X == 0 || d.Y == 0 {
			return s.D.ZuOriginal(a)
		}
		seite := math.Min(math.Abs(d.X), math.Abs(d.Y))
		seite = math.Min(seite, math.Min(platzAchse(d.X, a.X, w)*math.Abs(d.X), platzAchse(d.Y, a.Y, h)*math.Abs(d.Y)))
		e = a.Plus(document.Pt(math.Copysign(seite, d.X), math.Copysign(seite, d.Y)))
	}
	return s.D.ZuOriginal(s.klemmeAnsicht(e))
}

// platzAchse liefert, wie oft die Schrittweite d von a aus bis zum Rand [0, max] passt (+Inf bei d = 0).
func platzAchse(d, a, max float64) float64 {
	switch {
	case d > 0:
		return (max - a) / d
	case d < 0:
		return a / -d
	}
	return math.Inf(1)
}

// beendeNeu übernimmt das neue Objekt als einen Schritt in den Verlauf und wählt es aus — Pfeil und Kasten nur, wenn
// sie die Mindestgröße erreichen.
func (s *Sitzung) beendeNeu() {
	if schwaerzt(s.neuArt) {
		s.schwaerzeRahmen()
		return
	}
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

// --- Schwärzen (ADR-0022) ---

// Rahmen liefert den gerade gezogenen Schwärzungsrahmen in Fensterpunkten (ok = false, wenn keiner gezogen wird).
func (s *Sitzung) Rahmen() (a, b document.Punkt, ok bool) {
	if s.zug != zugNeu || !s.rahmen {
		return document.Punkt{}, document.Punkt{}, false
	}
	return s.Abb.ZuFenster(s.rahmenA), s.Abb.ZuFenster(s.rahmenB), true
}

// schwaerzeRahmen überschreibt beim Loslassen die Pixel im Rahmen. Der Rahmen gilt in der Ansicht; seine Ecken werden
// ins Original umgerechnet und nach außen auf ganze Pixel gerundet, ein verborgener Stauch-Streifen dazwischen wird
// mitgeschwärzt. Unter der Mindestgröße passiert nichts.
func (s *Sitzung) schwaerzeRahmen() {
	if !s.rahmen {
		return
	}
	s.rahmen = false
	r := document.R(s.rahmenA, s.rahmenB)
	grenze := MinObjekt / s.Abb.Zoom
	if r.Breite() < grenze || r.Hoehe() < grenze {
		return
	}
	o1, o2 := s.D.ZuOriginal(document.Pt(r.Min.X, r.Min.Y)), s.D.ZuOriginal(document.Pt(r.Max.X, r.Max.Y))
	px := image.Rect(int(math.Floor(math.Min(o1.X, o2.X))), int(math.Floor(math.Min(o1.Y, o2.Y))),
		int(math.Ceil(math.Max(o1.X, o2.X))), int(math.Ceil(math.Max(o1.Y, o2.Y))))
	sw := document.Schwaerzung{Farbe: Schwaerzfarbe}
	if s.neuArt == WerkzeugVerpixeln {
		kurz := min(px.Dx(), px.Dy())
		switch {
		case kurz >= VerpixelnBlock:
			sw = document.Schwaerzung{Block: VerpixelnBlock, Rauschen: VerpixelnRauschen}
		case kurz >= document.MinBlock:
			sw = document.Schwaerzung{Block: document.MinBlock, Rauschen: VerpixelnRauschen}
		default:
			s.ersatzSchwarz = true // zu schmal zum Verpixeln: schwärzen (ADR-0022 Punkt 2)
		}
	}
	if s.D.Schwaerzen(px, sw) == nil {
		s.bildAenderung++
	}
}

// VerpixelnErsetzt meldet, dass der letzte Rahmen zu schmal zum Verpixeln war und geschwärzt wurde.
func (s *Sitzung) VerpixelnErsetzt() bool { return s.ersatzSchwarz }

// BildAenderung zählt die Änderungen an den Pixeln des Originals durch Schwärzen; die Oberfläche baut ihr Bild neu,
// wenn sich der Wert ändert.
func (s *Sitzung) BildAenderung() uint64 { return s.bildAenderung }

// --- Texteingabe ---

// KopierenErlaubt meldet, ob Strg+C das Bild in die Zwischenablage legen darf: nicht während einer Texteingabe, dort
// bleibt die Taste dem Text vorbehalten (Karte #39).
func (s *Sitzung) KopierenErlaubt() bool { return !s.TextLaeuft() }

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
	vor := s.D.Stand()
	defer s.merkeAenderung(vor)
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
