// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "github.com/Andiministrator/schnapp/internal/document"

// ZeigerArt ist die Form des Mauszeigers über der Leinwand (Karte #323, ADR-0021 Nachtrag 18). Die Oberfläche bildet
// sie auf die Zeiger des Systems ab; die Entscheidung fällt hier, ohne Gio, nach denselben Regeln wie Druecken.
type ZeigerArt int

const (
	ZeigerStandard    ZeigerArt = iota // Pfeil
	ZeigerFadenkreuz                   // Zeichenwerkzeuge, Pipette, Bereich für Text aus dem Bild
	ZeigerText                         // Text, Sprechblase
	ZeigerVerschieben                  // über einem greifbaren Objekt bzw. beim Verschieben (vier Pfeile)
	ZeigerGreifen                      // über einem Punktgriff: Pfeilende, Bogen, Spitze der Sprechblase
	ZeigerGreifend                     // Ansicht wird gezogen
	ZeigerNWSO                         // Größe ändern, Diagonale links oben – rechts unten
	ZeigerNOSW                         // Größe ändern, Diagonale rechts oben – links unten
	ZeigerNS                           // Größe ändern, senkrecht
	ZeigerOW                           // Größe ändern, waagerecht
)

// griffZeiger ist der Zeiger zu einem Griff: Ecken und Kanten zeigen die Richtung, in der sie die Größe ändern,
// Punktgriffe die greifende Hand.
func griffZeiger(a GriffArt) ZeigerArt {
	switch a {
	case GriffNW, GriffSO, GriffQuelleNW, GriffQuelleSO, GriffZielNW, GriffZielSO:
		return ZeigerNWSO
	case GriffNO, GriffSW, GriffQuelleNO, GriffQuelleSW, GriffZielNO, GriffZielSW:
		return ZeigerNOSW
	case GriffN, GriffS:
		return ZeigerNS
	case GriffO, GriffW:
		return ZeigerOW
	case GriffAnfang, GriffEnde, GriffBogen, GriffSpitze:
		return ZeigerGreifen
	}
	return ZeigerStandard
}

// werkzeugZeiger ist der Zeiger des Werkzeugs, wo kein Griff und kein Objekt etwas anderes sagen.
func (s *Sitzung) werkzeugZeiger() ZeigerArt {
	switch s.Werkzeug {
	case WerkzeugAuswahl:
		return ZeigerStandard
	case WerkzeugText, WerkzeugSprechblase:
		return ZeigerText
	}
	return ZeigerFadenkreuz
}

// Zeiger liefert die Form des Zeigers am Fensterpunkt p (nil: kein Zeiger über der Leinwand). Ein laufender Zug
// behält seine Form, auch wenn der Zeiger dabei den Griff verlässt.
func (s *Sitzung) Zeiger(p *document.Punkt) ZeigerArt {
	switch {
	case s.pipette || s.TextBereichLaeuft():
		return ZeigerFadenkreuz
	case s.pan && s.bewegt, s.zug == zugAnsicht && s.bewegt:
		return ZeigerGreifend
	case s.zug == zugGriff, s.zug == zugZuschnitt:
		return griffZeiger(s.griff)
	case s.zug == zugObjekt:
		return ZeigerVerschieben
	case s.zug != keinZug, p == nil:
		return s.werkzeugZeiger()
	}
	q := s.original(*p)
	if sel, ok := s.D.Objekt(s.Auswahl); ok && !ohneGriffe(s.Werkzeug) && sel.Ebenenwerte().Bearbeitbar() {
		if art, ok := GriffTreffer(sel, q, s.toleranz()); ok {
			return griffZeiger(art)
		}
	}
	if s.Werkzeug == WerkzeugZuschneiden {
		if art, ok := s.zuschnittGriff(*p); ok {
			return griffZeiger(art)
		}
	}
	if s.Werkzeug != WerkzeugAuswahl {
		return s.werkzeugZeiger()
	}
	if _, ok := Treffer(s.sichtbareObjekte(), q, s.toleranz()); ok {
		return ZeigerVerschieben
	}
	return ZeigerStandard
}
