// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// Schatten je Objekt (ADR-0021 Nachtrag 26 Punkt 7): Die rechte Leiste zeigt den des gewählten Objekts, sonst den des
// Werkzeugs (Stil.Schatten, ein Wert für alle Werkzeuge wie die Strichstärke). Setzen ändert ein gewähltes Objekt als
// einen Rückgängig-Schritt; Ziehen an einem Regler ist einer (Zug wie am Regler der Strichstärke).

// SchattenObjekt liefert das gewählte Objekt (jede Art trägt einen Schatten).
func (s *Sitzung) schattenObjekt() (document.Objekt, bool) {
	id := s.Auswahl
	if s.zug == zugText {
		id = s.textID
	}
	return s.D.Objekt(id)
}

// Schatten ist der Schatten, den die rechte Leiste zeigt.
func (s *Sitzung) Schatten() document.Schatten {
	if o, ok := s.schattenObjekt(); ok {
		return o.Ebenenwerte().Schatten
	}
	return s.Stil.Schatten
}

// mitSchatten liefert o mit dem Schatten sch.
func mitSchatten(o document.Objekt, sch document.Schatten) document.Objekt {
	e := o.Ebenenwerte()
	e.Schatten = sch
	return document.MitEbene(o, e)
}

// mitStilSchatten gibt einem neuen Objekt den Schatten des Werkzeugs, wenn er an ist (sonst bleibt der Nullwert, damit
// Objekte ohne Schatten nach Speichern und Laden gleich bleiben).
func (s *Sitzung) mitStilSchatten(o document.Objekt) document.Objekt {
	if !s.Stil.Schatten.An {
		return o
	}
	return mitSchatten(o, s.Stil.Schatten)
}

// gueltigerSchatten bringt sch in die Grenzen von document (Winkel 0 bis 360, Länge und Weich begrenzt).
func gueltigerSchatten(sch document.Schatten) document.Schatten {
	sch.Winkel = document.NormWinkel(sch.Winkel)
	sch.Laenge = max(0, min(document.MaxSchattenLaenge, sch.Laenge))
	sch.Weich = max(0, min(document.MaxSchattenWeich, sch.Weich))
	return sch
}

// SetzeSchatten setzt den Schatten sch für folgende Objekte und ändert ein gewähltes als einen Rückgängig-Schritt
// (in der Texteingabe als Teil der Eingabe). Liefert, ob sich das Dokument geändert hat.
func (s *Sitzung) SetzeSchatten(sch document.Schatten) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	sch = gueltigerSchatten(sch)
	s.Stil.Schatten = sch // das Werkzeug behält die Werte auch ausgeschaltet
	o, ok := s.schattenObjekt()
	if !ok {
		return false
	}
	if !sch.An {
		// Am Objekt ist ein ausgeschalteter Schatten der Nullwert: Die Datei schreibt ihn nicht, nach Speichern und Laden
		// wäre das Objekt sonst ein anderes (Sitzungs-Kritik 26.11. (5)).
		sch = document.Schatten{}
	}
	neu := mitSchatten(o, sch)
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

// SchalteSchatten schaltet den Schatten an oder aus. Beim Einschalten eines Schattens ohne eigene Werte (etwa aus einer
// Datei ohne Schatten) gelten die des Werkzeugs, sonst die Voreinstellung zum Faktor des Bilds.
func (s *Sitzung) SchalteSchatten() bool {
	sch := s.Schatten()
	if sch.An {
		sch.An = false
		if s.feld == FeldSchatten {
			s.feld, s.hsvGemerkt = FeldVordergrund, false
		}
		return s.SetzeSchatten(sch)
	}
	if sch == (document.Schatten{}) {
		sch = s.Stil.Schatten
		if sch.Laenge == 0 && sch.Weich == 0 && sch.Farbe == (document.Schatten{}).Farbe {
			sch = document.StandardSchatten(s.faktor())
		}
	}
	sch.An = true
	return s.SetzeSchatten(sch)
}

// BeginneSchattenZug beginnt das Ziehen an einem Regler des Schattens (Länge, Weich): wie am Regler der Strichstärke
// ändert sich das gewählte Objekt bis zum Loslassen mit und wird ein Rückgängig-Schritt (BeendeBreiteZug,
// BrecheBreiteZugAb beenden beide Züge).
func (s *Sitzung) BeginneSchattenZug() bool {
	if s.Zieht() || s.gesperrt() || s.zug == zugText {
		return false
	}
	s.breiteZug, s.breiteZugObj = true, false
	if o, ok := s.schattenObjekt(); ok && s.D.BeginneZug(o.Kennung()) == nil {
		s.breiteZugObj, s.breiteZugVor = true, s.D.Stand()
	}
	return true
}

// SchattenZugSetze setzt während des Ziehens den Schatten sch für folgende Objekte und das gewählte.
func (s *Sitzung) SchattenZugSetze(sch document.Schatten) {
	if !s.breiteZug {
		return
	}
	sch = gueltigerSchatten(sch)
	s.Stil.Schatten = sch
	if !sch.An {
		sch = document.Schatten{} // am Objekt wie in SetzeSchatten der Nullwert (V-4)
	}
	if s.breiteZugObj {
		if o, ok := s.D.Objekt(s.Auswahl); ok {
			_ = s.D.ZugSetze(mitSchatten(o, sch))
		}
	}
}
