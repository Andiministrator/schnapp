// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
)

// Kritik B-2 (Sprint 13, Regression aus F-705): Über der Liste der Übersicht F1 liegt das Zeigerziel des Rollens.
// Ein Klick auf die Liste schließt die Übersicht trotzdem, wie ein Klick auf die Überschrift oder neben die Tafel
// („click or Esc to close“, ADR-0031 Nachtrag 11).
func TestHilfeKlickAufDieListeSchliesst(t *testing.T) {
	for _, f := range []image.Point{{1200, 800}, {800, 500}} {
		l := neuerMenueLauf(t, f)
		l.e.hilfeOffen = true
		l.setle()
		fl := l.e.leinwandFlaeche(l.gtx())
		l.klick(fl.Min.Add(fl.Size().Div(2)))
		l.setle()
		if l.e.hilfeOffen {
			t.Errorf("Fenster %v: Klick auf die Liste der Übersicht schließt sie nicht", f)
		}
		l.e.hilfeOffen = true
		l.setle()
		ecke := fl.Min.Add(fl.Size().Sub(l.e.hilfeGroesse).Div(2))
		l.klick(ecke.Add(image.Pt(20, 8)))
		l.setle()
		if l.e.hilfeOffen {
			t.Errorf("Fenster %v: Klick auf die Überschrift schließt die Übersicht nicht", f)
		}
	}
}
