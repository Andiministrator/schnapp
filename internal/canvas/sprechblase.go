// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Werkzeug Sprechblase (ADR-0021 Nachtrag 13, Karte #194): ein Text mit Spitze. Aufziehen setzt die Spitze auf den
// Druckpunkt und den Text auf den Ort des Loslassens, danach läuft die Texteingabe wie beim Text-Werkzeug; ein Klick
// ohne Zug setzt den Text an die Klickstelle mit der Standard-Spitze. Farbpaar und Stil sind die des Text-Werkzeugs
// (paarSchluessel).

// SpitzeVersatz ist der Abstand der Standard-Spitze von der linken unteren Ecke des Schilds in Schriftgraden, nach
// links und nach unten (ADR-0021 Nachtrag 13 Punkt 5).
const SpitzeVersatz = 3.0

// paarSchluessel ist das Werkzeug, dessen Farbpaar w benutzt: die Sprechblase teilt das Paar „Text“ (ADR-0021
// Nachtrag 11 Punkt 2 und Nachtrag 13 Punkt 5).
func paarSchluessel(w Werkzeug) Werkzeug {
	if w == WerkzeugSprechblase {
		return WerkzeugText
	}
	return w
}

// neueBlase ist der Text einer neuen Sprechblase am Ort ort mit der Spitze spitze (Original), im Stil des Werkzeugs
// und noch ohne Inhalt.
func (s *Sitzung) neueBlase(spitze, ort document.Punkt) document.Text {
	t := document.Text{Ort: ort, Schriftgrad: s.Stil.Schriftgrad, Farbe: s.Stil.Farbe, Hintergrund: s.Stil.Fuellung}
	s.Stil.aufText(&t)
	t.Spitze = &spitze // ein eigener Zeiger je Objekt: Objekte sind Werte, Rückgängig teilt keinen
	return t
}

// StandardSpitze ist die Spitze, die ein Klick (und der Schalter „Spitze“) für t setzt: SpitzeVersatz Schriftgrade
// links und unter der linken unteren Ecke des Schilds, auf das sichtbare Bild begrenzt (Original). Ergibt das nach dem
// Begrenzen kein Dreieck (das Schild sitzt in der Bildecke), kommt die nächste Ecke an die Reihe: rechts unten, links
// oben, rechts oben (F-634); geht keine, bleibt es bei der ersten.
func (s *Sitzung) StandardSpitze(t document.Text) document.Punkt {
	g := t.Grenzen()
	v := SpitzeVersatz * t.Schriftgrad
	ecken := [4]document.Punkt{
		document.Pt(g.Min.X-v, g.Max.Y+v), document.Pt(g.Max.X+v, g.Max.Y+v),
		document.Pt(g.Min.X-v, g.Min.Y-v), document.Pt(g.Max.X+v, g.Min.Y-v),
	}
	var erste document.Punkt
	for i, p := range ecken {
		p = s.D.ZuOriginal(s.klemmeAnsicht(s.D.ZuAnsicht(p)))
		if i == 0 {
			erste = p
		}
		t.Spitze = &p
		if _, _, _, ok := t.Spitzendreieck(); ok {
			return p
		}
	}
	return erste
}

// beendeBlase beendet das Aufziehen einer Sprechblase: Der Text (bei einem Klick ohne Zug erst jetzt, an der
// Klickstelle mit der Standard-Spitze) bleibt im Zug des Dokuments, und die Texteingabe beginnt.
func (s *Sitzung) beendeBlase() {
	if s.neuID == 0 {
		t := document.Text{Ort: s.original(s.start), Schriftgrad: s.Stil.Schriftgrad, Farbe: s.Stil.Farbe,
			Hintergrund: s.Stil.Fuellung}
		s.Stil.aufText(&t)
		sp := s.StandardSpitze(t)
		t.Spitze = &sp
		id, err := s.D.BeginneNeuenZug(s.mitStilSchatten(t))
		if err != nil {
			return
		}
		s.neuID = id
	}
	s.zug, s.textID, s.textNeu, s.Auswahl = zugText, s.neuID, true, s.neuID
}

// spitzeText ist der Text, dessen Spitze der Schalter der rechten Leiste ändert: der in der Eingabe, sonst der
// gewählte (false: keiner, oder die Ebene ist gesperrt oder ausgeblendet).
func (s *Sitzung) spitzeText() (document.Text, bool) {
	o, ok := s.D.Objekt(s.stilObjektID())
	if !ok {
		return document.Text{}, false
	}
	t, ist := o.(document.Text)
	return t, ist && o.Ebenenwerte().Bearbeitbar()
}

// KannSpitze meldet, ob der Schalter „Spitze“ wirkt: Ein Text ist gewählt oder in der Eingabe.
func (s *Sitzung) KannSpitze() bool {
	_, ok := s.spitzeText()
	return ok && !s.Zieht() && !s.gesperrt()
}

// SpitzeAn meldet, ob der gewählte Text eine Spitze hat.
func (s *Sitzung) SpitzeAn() bool {
	t, ok := s.spitzeText()
	return ok && t.Spitze != nil
}

// SchalteSpitze schaltet die Spitze des gewählten Texts um: aus → Standard-Spitze, an → keine. Ein Rückgängig-Schritt;
// in der Texteingabe gehört es zur Eingabe. Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SchalteSpitze() bool {
	t, ok := s.spitzeText()
	if !ok || s.Zieht() || s.gesperrt() {
		return false
	}
	if t.Spitze == nil {
		p := s.StandardSpitze(t)
		t.Spitze = &p
	} else {
		t.Spitze = nil
	}
	if s.zug == zugText {
		_ = s.D.ZugSetze(t)
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(t) == nil
	s.merkeAenderung(vor)
	return ok
}
