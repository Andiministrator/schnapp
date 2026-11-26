// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Werkzeug ist das aktive Werkzeug des Editors (ADR-0021).
type Werkzeug int

const (
	WerkzeugAuswahl     Werkzeug = iota // auswählen, verschieben, Griffe
	WerkzeugPfeil                       // Ziehen vom Anfang zur Spitze
	WerkzeugKasten                      // Ziehen von Ecke zu Ecke
	WerkzeugText                        // Klick setzt ein Textfeld
	WerkzeugNummer                      // Klick setzt eine Schritt-Nummer
	WerkzeugSchwaerzen                  // Rahmen ziehen: Pixel schwarz überschreiben (ADR-0022)
	WerkzeugVerpixeln                   // Rahmen ziehen: Mosaik ohne Bezug zum Inhalt (ADR-0022 Nachtrag 1)
	WerkzeugZuschneiden                 // Rahmen ziehen: sichtbarer Ausschnitt (ADR-0023)
	WerkzeugStauchen                    // Band ziehen: Zeilen oder Spalten herausnehmen (ADR-0023)
	WerkzeugLupe                        // Quelle aufziehen, das Ziel erscheint vergrößert daneben (ADR-0033)
	WerkzeugSprechblase                 // Spitze setzen, zum Ort des Texts ziehen, tippen (ADR-0021 Nachtrag 13)
)

// Werte des Verpixelns (ADR-0022 Nachtrag 1), Größen in Pixeln des Originals. Die Blockgröße ist nur noch Optik: Die
// Blockfarben hängen nicht vom Inhalt ab.
const (
	VerpixelnBlock  = 12
	VerpixelnMosaik = 24
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

// StandardFarbfelder sind die eigenen Farbfelder, solange es keine colors.ini gibt (ADR-0021 Nachtrag 24, Karte #364):
// die Beschriftungsfarben, dazu Weiß, Gelb und Orange.
var StandardFarbfelder = append(append([]color.NRGBA(nil), Beschriftungsfarben...),
	color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, // Weiß
	color.NRGBA{R: 0xF2, G: 0xC2, B: 0x30, A: 0xFF}, // Gelb
	color.NRGBA{R: 0xF5, G: 0x8A, B: 0x1F, A: 0xFF}, // Orange
)

// Weiss ist die Schriftfarbe auf Text-Schildern (Nummern zeichnen Renderer und Export ohnehin weiß).
var Weiss = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

// Stil sind die Eigenschaften neuer Objekte, Größen in Pixeln des Originals.
type Stil struct {
	// Farbe ist der Vordergrund, Fuellung die Fläche von Kasten und Text-Schild (Deckkraft 0: keine), beide aus dem
	// Paar des zuletzt gewählten Werkzeugs mit Farbe (ADR-0021 Nachtrag 11, farbpaar.go).
	Farbe       color.NRGBA
	Fuellung    color.NRGBA
	Breite      float64 // Strich von Pfeil und Kasten
	Eckradius   float64 // Kasten
	Schriftgrad float64 // Text
	Radius      float64 // Nummer
	// Schrift, Stil und Ausrichtung neuer Texte (ADR-0021 Nachtrag 10); leere Schrift ist Go.
	Schrift         string
	Fett, Kursiv    bool
	Unterstrichen   bool
	Durchgestrichen bool
	Ausrichtung     document.Ausrichtung
	// Spitzenarten neuer Pfeile je Ende (ADR-0021 Nachträge 19 und 20); Nullwerte: Dreieck bei B, keine bei A.
	KopfEnde   document.Kopfart
	KopfAnfang document.Anfangskopf
	// Strichart neuer Pfeile, Kästen und Lupen (ADR-0021 Nachtrag 25), wie die Breite ein Wert für alle drei.
	Strichart document.Strichart
	// Schatten neuer Objekte (ADR-0021 Nachtrag 26), ein Wert für alle Werkzeuge; aus, mit der Voreinstellung als Werten.
	Schatten document.Schatten
}

// StandardStil ist die Voreinstellung (ADR-0021 Punkt 4, Strich seit Nachtrag 16).
func StandardStil() Stil {
	sch := document.StandardSchatten(1)
	sch.An = false // aus, mit der Voreinstellung als Werten (ADR-0021 Nachtrag 26)
	return Stil{Farbe: Beschriftungsfarben[0], Breite: 6, Eckradius: 0, Schriftgrad: 20, Radius: 14, Schatten: sch}
}

// StilFuer ist die Voreinstellung für ein Bild der Größe g (ADR-0021 Nachtrag 7): Strich, Schriftgrad und
// Nummernradius wachsen mit demselben ganzzahligen Faktor wie die Stauch-Kante (document.KantenFaktor: kurze Seite /
// 1080), damit Beschriftungen auf 4K- und HiDPI-Aufnahmen nicht dünn und klein wirken.
func StilFuer(g image.Point) Stil { return StilMitFaktor(document.KantenFaktor(g)) }

// StilMitFaktor ist die Voreinstellung mal dem ganzzahligen Faktor f (mindestens 1). NeueSitzung nimmt den Faktor des
// Dokuments (document.Dokument.KantenFaktor), der wie bei der Kante durch die aktuelle Ansicht gedeckelt ist: ein
// 4K-Original, auf 300×80 zugeschnitten, beginnt mit Faktor 1 (Sitzungs-QA B-6).
func StilMitFaktor(faktor int) Stil {
	st := StandardStil()
	f := float64(max(1, faktor))
	st.Breite *= f
	st.Schriftgrad *= f
	st.Radius *= f
	st.Schatten.Laenge *= f
	st.Schatten.Weich *= f
	return st
}

// StrichStufen sind die Strichstärken der rechten Leiste in Pixeln des Originals (− und + gehen von Stufe zu Stufe).
var StrichStufen = []float64{1, 2, 3, 4, 6, 8, 12, 16, 24, 32}

func (st Stil) strich() document.Strich {
	return document.Strich{Farbe: st.Farbe, Breite: st.Breite, Art: st.Strichart}
}

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

// Umgefaerbt liefert o mit dem Vordergrund c: Strich bei Pfeil und Kasten, Schrift beim Text, Kreis bei der Nummer
// (ADR-0021 Nachtrag 11; das Schild des Texts ist die Füllung, UmgefaerbtIn).
func Umgefaerbt(o document.Objekt, c color.NRGBA) document.Objekt {
	return UmgefaerbtIn(o, FeldVordergrund, c)
}

// --- Werkzeug und Farbe ---

// SetzeWerkzeug wechselt das Werkzeug. Eine Texteingabe wird vorher bestätigt; während eines Mauszugs geht es nicht.
func (s *Sitzung) SetzeWerkzeug(w Werkzeug) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	s.TextBestaetigen()
	s.pipette = false
	s.tb = textBereich{} // ein Werkzeugwechsel beendet den Bereichsmodus (ADR-0035)
	if ohneGriffe(w) {
		s.Auswahl = 0 // F-174: sonst zöge ein Zug nahe einem Griff den Griff statt den Rahmen
	}
	if s.feld == FeldSchatten && w != s.Werkzeug {
		// Das Feld Schatten wählt man bewusst am Farbfeld der Leiste; mit einem anderen Werkzeug färbte die Farbwahl
		// sonst unerwartet den Schatten statt der Beschriftung (Sichtprobe #363).
		s.feld, s.hsvGemerkt = FeldVordergrund, false
	}
	vorher := s.zuschnittModus()
	s.Werkzeug = w
	s.wechslePaar(w)
	if s.zuschnittModus() != vorher {
		// Vollansicht an oder aus (ADR-0023 Nachtrag 5): anderes Bild; die Ansichtskoordinaten bleiben, neu eingepasst
		// wird nur, wenn der Nutzer nicht selbst gezoomt oder verschoben hat.
		s.bildAenderung++
		if s.eingepasst {
			s.Abb = s.einpassAbb()
		}
	}
	return true
}

// SetzeFarbe setzt die Farbe c im aktiven Feld (AktivesFeld) für folgende Objekte und färbt das Objekt um, auf das die
// Farbwahl wirkt (ein Rückgängig-Schritt); während der Texteingabe gehört das Umfärben zur Eingabe. Das Paar des
// Werkzeugs, das dieses Objekt zeichnet, übernimmt die Farbe. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeFarbe(c color.NRGBA) bool {
	if s.Zieht() || s.gesperrt() { // während des Speicherns auch keine neue Farbe (F-280): Bearbeiten ruht ganz
		return false
	}
	f := s.AktivesFeld()
	s.merkePaarFarbe(s.zielWerkzeug(), f, c)
	o, ok := s.farbObjekt()
	if !ok {
		return false
	}
	neu := UmgefaerbtIn(o, f, c)
	if s.zug == zugText {
		_ = s.D.ZugSetze(neu)
		return false
	}
	if document.Gleich(neu, o) {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// strichObjekt liefert das gewählte Objekt, wenn es einen Strich hat (Pfeil, Kasten).
func (s *Sitzung) strichObjekt() (document.Objekt, bool) {
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok {
		return nil, false
	}
	switch o.(type) {
	case document.Pfeil, document.Kasten, document.Lupe:
		return o, true
	}
	return nil, false
}

// Breite ist die Strichstärke, die die rechte Leiste zeigt: die des gewählten Pfeils oder Kastens, sonst die des
// Werkzeugs.
func (s *Sitzung) Breite() float64 {
	o, _ := s.strichObjekt()
	switch v := o.(type) {
	case document.Pfeil:
		return v.Strich.Breite
	case document.Kasten:
		return v.Strich.Breite
	case document.Lupe:
		return v.Strich.Breite
	}
	return s.Stil.Breite
}

// SetzeBreite setzt die Strichstärke b (Pixel des Originals) für folgende Pfeile und Kästen und ändert ein gewähltes
// solches Objekt als einen Rückgängig-Schritt, wie SetzeFarbe (ADR-0021 Nachtrag 7). Liefert, ob sich das Dokument
// geändert hat; dieselbe Stärke ist kein Schritt.
func (s *Sitzung) SetzeBreite(b float64) bool {
	if s.Zieht() || s.gesperrt() || !(b > 0) || b > document.MaxGroesse {
		return false
	}
	s.Stil.Breite = b
	o, ok := s.strichObjekt()
	if !ok {
		return false
	}
	neu := mitBreite(o, b)
	if document.Gleich(neu, o) {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// BreiteStufe geht von der wirksamen Breite zur nächsten Stufe aus StrichStufen (dir > 0: dicker, sonst dünner); eine
// Breite zwischen zwei Stufen geht zur nächsten in der Richtung. Liefert die neue Breite und ob es eine gab.
func (s *Sitzung) BreiteStufe(dir int) (float64, bool) {
	jetzt := s.Breite()
	if dir > 0 {
		for _, b := range StrichStufen {
			if b > jetzt {
				return b, true
			}
		}
		return jetzt, false
	}
	for i := len(StrichStufen) - 1; i >= 0; i-- {
		if StrichStufen[i] < jetzt {
			return StrichStufen[i], true
		}
	}
	return jetzt, false
}

// mitBreite liefert Pfeil oder Kasten o mit der Strichstärke b; andere Objekte unverändert.
func mitBreite(o document.Objekt, b float64) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		v.Strich.Breite = b
		return v
	case document.Kasten:
		v.Strich.Breite = b
		return v
	case document.Lupe:
		v.Strich.Breite = b
		return v
	}
	return o
}

// --- Anlegen ---

// beginneNeu startet mit einem Zeichenwerkzeug an q (Original). Pfeil und Kasten entstehen erst beim Ziehen über die
// Schwelle, Nummer und Text sofort.
func (s *Sitzung) beginneNeu(q document.Punkt) {
	s.neuArt, s.neuID = s.Werkzeug, 0
	// Klick auf einen vorhandenen Text bearbeitet ihn auch bei vollem Dokument (kein neues Objekt nötig).
	if s.Werkzeug == WerkzeugText || s.Werkzeug == WerkzeugSprechblase {
		if id, ok := Treffer(s.sichtbareObjekte(), q, s.toleranz()); ok {
			if o, _ := s.D.Objekt(id); isText(o) {
				s.Auswahl = id
				s.TextBearbeiten() // bei verborgener Schrift nur Auswahl und Meldung (F-740)
				return
			}
		}
	}
	if a := s.Voll(); a != AblehnungKeine {
		s.abgelehnt = a // sichtbare Meldung statt stillem Scheitern (F-154)
		return
	}
	switch s.Werkzeug {
	case WerkzeugPfeil, WerkzeugKasten, WerkzeugLupe, WerkzeugSprechblase:
		s.zug = zugNeu
	case WerkzeugSchwaerzen, WerkzeugVerpixeln, WerkzeugZuschneiden, WerkzeugStauchen:
		s.zug, s.rahmen = zugNeu, false
	case WerkzeugNummer:
		n := document.Nummer{Mitte: q, Wert: s.D.NaechsteNummer(), Radius: s.Stil.Radius, Farbe: s.Stil.Farbe}
		n = s.mitStilSchatten(n).(document.Nummer)
		id, err := s.D.BeginneNeuenZug(n)
		if err != nil {
			return
		}
		n.ID = id
		s.zug, s.neuID, s.startObj = zugNeu, id, n
	case WerkzeugText:
		t := document.Text{Ort: q, Schriftgrad: s.Stil.Schriftgrad, Farbe: s.Stil.Farbe, Hintergrund: s.Stil.Fuellung}
		s.Stil.aufText(&t)
		id, err := s.D.BeginneNeuenZug(s.mitStilSchatten(t))
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
	// AblehnungTextVerborgen: Der angeklickte Text liegt ganz in einem Stauch-Streifen, nur das Dreieck seiner Sprechblase
	// ist zu sehen; er wird nicht zum Bearbeiten geöffnet (F-740).
	AblehnungTextVerborgen
)

// Voll meldet, warum das aktive Zeichenwerkzeug jetzt kein neues Objekt anlegen kann (AblehnungKeine, wenn es geht).
// Die Statusleiste verspricht damit keine Nummer, die nicht kommt.
func (s *Sitzung) Voll() Ablehnung {
	switch {
	case s.Werkzeug == WerkzeugAuswahl, rahmenWerkzeug(s.Werkzeug):
		return AblehnungKeine // Schwärzen, Zuschneiden und Stauchen legen kein Objekt an
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
	s.linien = nil
	switch s.neuArt {
	case WerkzeugPfeil:
		if s.Umschalt {
			b = s.umschaltEnde(WerkzeugPfeil, p)
		} else {
			b, s.linien = s.rastePunkt(b, s.neuID) // Umschalt hat Vorrang: 45° und Quadrat bleiben genau
		}
		pf := document.GeraderPfeil(a, b, s.Stil.strich())
		pf.Ende, pf.Anfang = s.Stil.KopfEnde, s.Stil.KopfAnfang
		pf.ID = s.neuID
		o = pf
	case WerkzeugKasten:
		if s.Umschalt {
			b = s.umschaltEnde(WerkzeugKasten, p)
		} else {
			b, s.linien = s.rastePunkt(b, s.neuID)
		}
		o = document.Kasten{ID: s.neuID, Bereich: document.R(a, b), Strich: s.Stil.strich(), Fuellung: s.Stil.Fuellung,
			Eckradius: s.Stil.Eckradius}
	case WerkzeugLupe: // Aufziehen wählt die Quelle wie einen Kasten, das Ziel liegt daneben (ADR-0033 Punkt 8)
		if !s.Umschalt {
			b, s.linien = s.rastePunkt(b, s.neuID)
		} else {
			b = s.umschaltEnde(WerkzeugKasten, p) // Quadrat, ohne Einrasten
		}
		l := s.neueLupe(document.R(a, b))
		l.ID = s.neuID
		o = l
	case WerkzeugSprechblase: // Druckpunkt = Spitze, Zeigerort = Ort des Texts (ADR-0021 Nachtrag 13 Punkt 5)
		b2, l1 := s.rastePunkt(a, s.neuID)
		t := s.neueBlase(b2, b)
		g := t.Grenzen()
		d, l2 := s.rasteRechteck(g, document.Punkt{}, s.neuID)
		t.Ort = t.Ort.Plus(d)
		s.linien = append(l1, l2...)
		t.ID = s.neuID
		o = t
	case WerkzeugNummer:
		d, linien := s.rasteRechteck(s.startObj.Grenzen(), b.Minus(a), s.neuID)
		s.linien = linien
		o = Verschiebe(s.startObj, d)
	case WerkzeugSchwaerzen, WerkzeugVerpixeln, WerkzeugZuschneiden, WerkzeugStauchen:
		s.rahmen = true
		s.rahmenA = s.klemmeAnsicht(s.Abb.ZuAnsicht(s.start))
		s.rahmenB = s.klemmeAnsicht(s.Abb.ZuAnsicht(p))
		return
	default:
		return
	}
	o = s.mitStilSchatten(o) // ADR-0021 Nachtrag 26
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
	switch {
	case schwaerzt(s.neuArt):
		s.schwaerzeRahmen()
		return
	case s.neuArt == WerkzeugZuschneiden:
		s.zuschneideRahmen()
		return
	case s.neuArt == WerkzeugStauchen:
		s.staucheRahmen()
		return
	}
	if s.neuArt == WerkzeugSprechblase {
		s.beendeBlase()
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
	case document.Lupe:
		gross = v.Quelle.Breite() >= min && v.Quelle.Hoehe() >= min
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

// Rahmen liefert den gerade gezogenen Rahmen (Schwärzen, Verpixeln, Zuschneiden) in Fensterpunkten, beim Stauchen das
// Band über die ganze Breite bzw. Höhe des Bilds (ok = false, wenn keiner gezogen wird).
func (s *Sitzung) Rahmen() (a, b document.Punkt, ok bool) {
	if (s.zug != zugNeu && s.zug != zugZuschnitt) || !s.rahmen {
		return document.Punkt{}, document.Punkt{}, false
	}
	if s.neuArt == WerkzeugStauchen {
		a, b := s.bandRahmen()
		return s.Abb.ZuFenster(a), s.Abb.ZuFenster(b), true
	}
	return s.Abb.ZuFenster(s.rahmenA), s.Abb.ZuFenster(s.rahmenB), true
}

// schwaerzeRahmen überschreibt beim Loslassen die Pixel im Rahmen. Der Rahmen gilt in der Ansicht und wird nach außen
// auf ganze Ansichtspixel gerundet und ins Original umgerechnet; ein verborgener Stauch-Streifen mitten darin wird
// mitgeschwärzt, einer hinter seinem Rand nicht (F-223). Unter der Mindestgröße passiert nichts.
func (s *Sitzung) schwaerzeRahmen() {
	if !s.rahmen {
		return
	}
	s.rahmen = false
	r := document.R(s.rahmenA, s.rahmenB)
	grenze := MinObjekt / s.Abb.Zoom
	if r.Breite() < grenze || r.Hoehe() < grenze {
		s.rahmenZuKlein = r.Breite() > 0 || r.Hoehe() > 0 // ein bloßer Klick meldet nichts; nicht still ignorieren: bei Zoom 0,1 sind das 60 Originalpixel (F-227)
		return
	}
	// Nach außen auf ganze Ansichtspixel, auf die Ansicht begrenzt (ADR-0023 Punkt 8), dann je Achse wie beim
	// Zuschneiden ins Original: vom ersten bis hinter das letzte Pixel. So endet der Rahmen an einer Naht oder an der
	// Kante der Ansicht vor dem verborgenen Streifen dahinter (F-223); ein Streifen mitten im Rahmen wird mitgeschwärzt.
	px, sw, ersatz, ok := SchwaerzBereich(s.D, r, s.neuArt == WerkzeugVerpixeln)
	if !ok {
		return
	}
	s.ersatzSchwarz = s.ersatzSchwarz || ersatz
	if s.D.Schwaerzen(px, sw) == nil {
		s.bildAenderung++
	}
}

// SchwaerzBereich rechnet einen Rahmen r der Ansicht in den Bereich des Originals und die Art der Schwärzung um, wie
// sie Editor und Kommandozeile (ADR-0030 Nachtrag 10) gleich anwenden: nach außen auf ganze Ansichtspixel, auf die
// Ansicht begrenzt (ADR-0023 Punkt 8), dann je Achse wie beim Zuschneiden ins Original — vom ersten bis hinter das
// letzte Pixel. So endet der Rahmen an einer Naht oder an der Kante der Ansicht vor dem verborgenen Streifen dahinter
// (F-223); ein Streifen mitten im Rahmen wird mitgeschwärzt. verpixeln wählt das Mosaik; ist der Bereich dafür zu
// schmal, wird geschwärzt und ersatz gemeldet (ADR-0022 Punkt 2). ok ist false, wenn nichts von r im Bild liegt.
func SchwaerzBereich(d *document.Dokument, r document.Rechteck, verpixeln bool) (px image.Rectangle,
	sw document.Schwaerzung, ersatz, ok bool) {
	g := d.Groesse()
	k := func(v float64, m int) int { return max(0, min(m, int(v))) }
	a := image.Rect(k(math.Floor(r.Min.X), g.X), k(math.Floor(r.Min.Y), g.Y), k(math.Ceil(r.Max.X), g.X),
		k(math.Ceil(r.Max.Y), g.Y))
	if a.Empty() {
		return image.Rectangle{}, sw, false, false
	}
	x0, x1 := originalBereich(d, a.Min.X, a.Max.X, false)
	y0, y1 := originalBereich(d, a.Min.Y, a.Max.Y, true)
	px = image.Rect(x0, y0, x1, y1)
	sw = document.Schwaerzung{Farbe: Schwaerzfarbe}
	if verpixeln {
		kurz := min(px.Dx(), px.Dy())
		switch {
		case kurz >= VerpixelnBlock:
			sw = document.Schwaerzung{Block: VerpixelnBlock, Mosaik: VerpixelnMosaik}
		case kurz >= document.MinBlock:
			sw = document.Schwaerzung{Block: document.MinBlock, Mosaik: VerpixelnMosaik}
		default:
			ersatz = true // zu schmal zum Verpixeln: schwärzen (ADR-0022 Punkt 2)
		}
	}
	return px, sw, ersatz, true
}

// NimmVerpixelnErsetzt meldet einmal, dass der letzte Rahmen zu schmal zum Verpixeln war und geschwärzt wurde; danach
// ist die Marke gelöscht (sonst zeigte jedes weitere Loslassen die Meldung erneut, F-170).
func (s *Sitzung) NimmVerpixelnErsetzt() bool {
	e := s.ersatzSchwarz
	s.ersatzSchwarz = false
	return e
}

// NimmRahmenZuKlein meldet einmal, dass der letzte Schwärzungs- oder Verpixelungsrahmen unter der Mindestgröße lag und
// nichts getan hat; danach ist die Marke gelöscht (F-227).
func (s *Sitzung) NimmRahmenZuKlein() bool {
	e := s.rahmenZuKlein
	s.rahmenZuKlein = false
	return e
}

// BildAenderung zählt die Änderungen am Ansichtsbild (Schwärzen, Zuschneiden, Stauchen, ADR-0023); die Oberfläche
// baut ihr Bild neu, wenn sich der Wert ändert.
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
	if s.zug != keinZug || s.gesperrt() {
		return false
	}
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok || !isText(o) || !o.Ebenenwerte().Bearbeitbar() {
		return false
	}
	// Liegen Schrift und Schild ganz in einem Stauch-Streifen (nur das Dreieck einer Sprechblase ist zu sehen), beginnt
	// keine Eingabe: Man tippte blind, und der Text erschiene erst, wenn das Schild aus dem Streifen wächst (F-740,
	// ADR-0023 Nachtrag 13). Die Auswahl bleibt, die Statusleiste sagt warum (Abgelehnt).
	if s.textVerborgen(o) {
		s.abgelehnt = AblehnungTextVerborgen
		return false
	}
	if s.D.BeginneZug(s.Auswahl) != nil {
		return false
	}
	s.zug, s.textID, s.textNeu = zugText, s.Auswahl, false
	return true
}

// textVerborgen meldet, ob von Text o weder Glyphen noch Schild in der Ansicht liegen (document.TextSichtbar).
func (s *Sitzung) textVerborgen(o document.Objekt) bool {
	t, ok := o.(document.Text)
	if !ok {
		return false
	}
	schrift, _ := s.D.TextSichtbar(t)
	return !schrift
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
	if !ok || s.gesperrt() {
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
	if !ok || s.gesperrt() {
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
	if s.zug != zugText || s.gesperrt() {
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
// zum Werkzeug Auswählen. Während des Speicherns wirkt Esc nicht (F-280).
func (s *Sitzung) Esc() {
	if s.gesperrt() {
		return
	}
	switch {
	case s.farbZug: // laufendes Ziehen im Farbwähler abbrechen (ADR-0021 Punkt 2, Sitzungs-QA A-4)
		s.BrecheFarbZugAb()
	case s.breiteZug: // ebenso am Regler der Strichstärke (Karte #352)
		s.BrecheBreiteZugAb()
	case s.tb.an: // Bereichsmodus „Text aus dem Bild kopieren“ abbrechen, auch mitten im Zug (ADR-0035)
		s.BeendeTextBereich()
	case s.pipette:
		s.pipette = false
	case s.zug == zugText:
		s.TextAbbrechen()
	case s.zug != keinZug:
		s.Abbrechen()
	case s.Auswahl != 0:
		s.Auswahl = 0
	default:
		s.SetzeWerkzeug(WerkzeugAuswahl) // über den gemeinsamen Weg: Vollansicht aus, Bild neu, Einpassen (KA-1)
	}
}

// Enter bestätigt die Texteingabe (mit umschalt: neue Zeile) oder beginnt sie am ausgewählten Text.
func (s *Sitzung) Enter(umschalt bool) bool {
	if s.gesperrt() {
		return false
	}
	if s.tb.an { // Bereichsmodus: Enter nimmt das ganze sichtbare Bild (ADR-0035 Punkt 5)
		return s.TextBereichGanz()
	}
	if s.zug == zugText {
		if umschalt {
			return s.TextZeilenumbruch()
		}
		return s.TextBestaetigen()
	}
	return s.TextBearbeiten()
}

// Doppelklick am Fensterpunkt p: Mit dem Werkzeug Auswählen bearbeitet er den getroffenen Text; mit Zuschneiden hebt er
// den Zuschnitt auf, mit Stauchen entfernt er die Naht unter dem Zeiger (ADR-0023). Sonst und daneben wirkt er wie
// ein Druck der linken Taste (der erste Klick hat dann schon gewirkt, ADR-0021).
func (s *Sitzung) Doppelklick(p document.Punkt) {
	if s.gesperrt() {
		return
	}
	if s.textBereichDruck(p, TastePrimaer) { // im Bereichsmodus beginnt auch der zweite Druck nur einen Zug
		return
	}
	s.TextBestaetigen()
	s.abgelehnt, s.ersatzSchwarz, s.rahmenZuKlein, s.ansichtMeldung = AblehnungKeine, false, false, AnsichtKeine
	switch {
	case s.Werkzeug == WerkzeugZuschneiden && s.HebeZuschnittAuf():
		return
	case s.Werkzeug == WerkzeugStauchen && s.EntferneNaht(p):
		return
	}
	if s.Werkzeug == WerkzeugAuswahl && s.zug == keinZug && !s.D.ZugLaeuft() {
		if id, ok := Treffer(s.sichtbareObjekte(), s.original(p), s.toleranz()); ok {
			if o, _ := s.D.Objekt(id); isText(o) {
				s.Auswahl = id
				s.TextBearbeiten()
				return
			}
		}
	}
	s.Druecken(p, TastePrimaer)
}
