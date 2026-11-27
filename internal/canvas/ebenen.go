// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"strconv"
	"strings"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Ebenen der rechten Leiste und des Menüs „Ebene“ ohne Gio (ADR-0032). Jede Aktion ist genau ein Rückgängig-Schritt;
// während des Speicherns oder eines Zugs geschieht nichts. Eine laufende Texteingabe wird vorher bestätigt.

// EbenenZeile beschreibt eine Zeile der Ebenenliste (ohne Text: die Oberfläche setzt den Namen).
type EbenenZeile struct {
	ID       int
	Art      Werkzeug // das Werkzeug, das diese Art zeichnet (Name der Art)
	Ebene    document.Ebene
	Objekt   document.Objekt // für Vorschau und Namen aus dem Inhalt
	Gewaehlt bool
}

// Ebenen liefert die Zeilen der Liste von oben (vorderste Ebene) nach unten; das Bild steht nicht darin.
func (s *Sitzung) Ebenen() []EbenenZeile {
	o := s.D.ObjekteOhneZug()
	z := make([]EbenenZeile, 0, len(o))
	for i := len(o) - 1; i >= 0; i-- {
		w, _ := objektWerkzeug(o[i])
		z = append(z, EbenenZeile{ID: o[i].Kennung(), Art: w, Ebene: o[i].Ebenenwerte(), Objekt: o[i],
			Gewaehlt: o[i].Kennung() == s.Auswahl})
	}
	return z
}

// ebenenBereit meldet, ob eine Aktion an den Ebenen jetzt wirken darf, und bestätigt eine laufende Texteingabe.
func (s *Sitzung) ebenenBereit() bool {
	if s.gesperrt() || s.Zieht() {
		return false
	}
	s.TextBestaetigen()
	return s.zug == keinZug && !s.D.ZugLaeuft()
}

// WaehleEbene wählt das Objekt id (Klick auf eine Zeile): Werkzeug Auswählen, damit Griffe und Eigenschaften gelten.
func (s *Sitzung) WaehleEbene(id int) bool {
	if _, ok := s.D.Objekt(id); !ok || !s.ebenenBereit() {
		return false
	}
	if s.Werkzeug != WerkzeugAuswahl {
		s.SetzeWerkzeug(WerkzeugAuswahl)
	}
	s.pipette = false
	s.Auswahl = id
	return true
}

// WaehleNachbar wählt die Ebene über (dir > 0) bzw. unter (dir < 0) der gewählten; ohne Auswahl die oberste bzw.
// unterste. Am Rand bleibt die Auswahl.
func (s *Sitzung) WaehleNachbar(dir int) bool {
	o := s.D.ObjekteOhneZug()
	if len(o) == 0 {
		return false
	}
	i := s.D.Stelle(s.Auswahl)
	switch {
	case i < 0 && dir > 0:
		i = len(o) - 1
	case i < 0:
		i = 0
	default:
		i = max(0, min(len(o)-1, i+dir))
	}
	return s.WaehleEbene(o[i].Kennung())
}

// KannEbene meldet, ob die Menüeinträge der gewählten Ebene wirken (eine Auswahl, kein Zug, kein Speichern).
func (s *Sitzung) KannEbene() bool {
	_, ok := s.D.Objekt(s.Auswahl)
	return ok && !s.gesperrt() && !s.Zieht()
}

// GewaehlteEbene liefert die Angaben der gewählten Ebene.
func (s *Sitzung) GewaehlteEbene() (document.Ebene, bool) {
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok {
		return document.Ebene{}, false
	}
	return o.Ebenenwerte(), true
}

// aendereEbene setzt die Angaben von id über f als einen Rückgängig-Schritt.
func (s *Sitzung) aendereEbene(id int, f func(document.Ebene) document.Ebene) bool {
	if !s.ebenenBereit() {
		return false
	}
	o, ok := s.D.Objekt(id)
	if !ok {
		return false
	}
	alt := o.Ebenenwerte()
	neu := f(alt)
	if neu == alt {
		return false
	}
	vor := s.D.Stand()
	if s.D.Ersetzen(document.MitEbene(o, neu)) != nil {
		return false
	}
	s.merkeAenderung(vor)
	return true
}

// SchalteAusgeblendet blendet id aus bzw. wieder ein.
func (s *Sitzung) SchalteAusgeblendet(id int) bool {
	return s.aendereEbene(id, func(e document.Ebene) document.Ebene { e.Ausgeblendet = !e.Ausgeblendet; return e })
}

// SchalteGesperrt sperrt id bzw. gibt es frei.
func (s *Sitzung) SchalteGesperrt(id int) bool {
	return s.aendereEbene(id, func(e document.Ebene) document.Ebene { e.Gesperrt = !e.Gesperrt; return e })
}

// SetzeDeckkraft setzt die Deckkraft von id in Prozent (begrenzt auf 0 bis 100).
func (s *Sitzung) SetzeDeckkraft(id, p int) bool {
	return s.aendereEbene(id, func(e document.Ebene) document.Ebene { return e.MitDeckkraft(p) })
}

// SetzeEbenenDeckkraftText setzt die Deckkraft der gewählten Ebene aus dem Feld der Leiste: ganze Zahl 0 bis 100, mit
// oder ohne „%“. Gelingt, wenn der Text gültig ist und eine Ebene gewählt (auch ohne Änderung).
func (s *Sitzung) SetzeEbenenDeckkraftText(text string) bool {
	t := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	p, err := strconv.Atoi(t)
	eb, ok := s.GewaehlteEbene()
	if err != nil || p < 0 || p > 100 || !ok {
		return false
	}
	return eb.Deckkraft() == p || s.SetzeDeckkraft(s.Auswahl, p)
}

// Umbenennen setzt den Namen von id; Leerraum am Rand fällt weg, leer heißt „Name aus dem Inhalt“. Ein ungültiger Name
// (zu lang, Steuerzeichen) ändert nichts.
func (s *Sitzung) Umbenennen(id int, name string) bool {
	name = strings.TrimSpace(name)
	if !document.EbenenNameGueltig(name) {
		return false
	}
	return s.aendereEbene(id, func(e document.Ebene) document.Ebene { e.Name = name; return e })
}

// EbeneNach bringt id an Stelle nach (0 = unterste, wie document.Umordnen).
func (s *Sitzung) EbeneNach(id, nach int) bool {
	if !s.ebenenBereit() {
		return false
	}
	vor := s.D.Stand()
	if s.D.Umordnen(id, nach) != nil || s.D.Stand() == vor {
		return false
	}
	s.merkeAenderung(vor)
	return true
}

// EbeneSchritt bringt die gewählte Ebene eine Stelle nach vorn (dir > 0) bzw. nach hinten (dir < 0).
func (s *Sitzung) EbeneSchritt(dir int) bool {
	i := s.D.Stelle(s.Auswahl)
	if i < 0 {
		return false
	}
	return s.EbeneNach(s.Auswahl, i+dir)
}

// KannEbeneSchritt meldet, ob EbeneSchritt(dir) etwas bewegte (Knöpfe ↑ ↓, Menü).
func (s *Sitzung) KannEbeneSchritt(dir int) bool {
	i := s.D.Stelle(s.Auswahl)
	n := len(s.D.Objekte())
	return s.KannEbene() && i >= 0 && i+dir >= 0 && i+dir < n
}

// DoppelVersatz ist der Versatz einer Kopie in Fensterpixeln (ADR-0032 Punkt 7).
const DoppelVersatz = 10.0

// Doppeln legt eine Kopie der gewählten Ebene direkt darüber, um DoppelVersatz Fensterpixel nach rechts unten
// versetzt, und wählt sie.
func (s *Sitzung) Doppeln() bool {
	if !s.ebenenBereit() {
		return false
	}
	if s.AnzahlGewaehlt() > 1 {
		return s.doppleMehrere() // alle gewählten in einem Schritt (ADR-0039 Punkt 3)
	}
	o, ok := s.D.Objekt(s.Auswahl)
	if !ok {
		return false
	}
	v := DoppelVersatz / s.Abb.Zoom
	kopie := Verschiebe(o, document.Pt(v, v))
	vor := s.D.Stand()
	id, err := s.D.FuegeEin(kopie, s.D.Stelle(o.Kennung())+1)
	if err != nil {
		return false
	}
	s.merkeAenderung(vor)
	s.Auswahl = id
	return true
}
