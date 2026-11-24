// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// Die Übersicht F1 ist modal wie die Einstellungstafel (ADR-0031 Nachtrag 11, F-707): Ein Klick neben der Tafel
// schließt sie und wirkt nicht auf das, was dahinter liegt.

// hilfeKlick öffnet die Übersicht, klickt an p (Fensterpixel).
func hilfeKlick(l *menueLauf, p image.Point) {
	l.e.hilfeOffen = true
	l.setle()
	l.klick(p)
}

func TestHilfeKlickAufFarbtauschWirktNicht(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	tx := l.text()
	tx.Hintergrund = canvas.Weiss
	if err := l.e.s.D.Ersetzen(tx); err != nil {
		t.Fatal(err)
	}
	l.e.s.Gespeichert()
	l.setle()
	_, farbe := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	_, _, tausch := farbpaarRahmen(farbe)
	alt, stand := l.text(), l.e.s.D.Stand()
	hilfeKlick(l, mittenPunkt(tausch))
	if l.e.hilfeOffen {
		t.Error("der Klick neben der Tafel hat die Übersicht nicht geschlossen")
	}
	if neu := l.text(); neu.Farbe != alt.Farbe || l.e.s.D.Stand() != stand {
		t.Errorf("Klick auf ⇄ hinter der Tafel hat das Objekt geändert (Schrift %v → %v)", alt.Farbe, neu.Farbe)
	}
}

func TestHilfeKlickInWerkzeugleisteWirktNicht(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	wz := l.e.s.Werkzeug
	rahmen, _ := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	hilfeKlick(l, mittenPunkt(rahmen[1]))
	if l.e.hilfeOffen {
		t.Error("der Klick in die Werkzeugleiste hat die Übersicht nicht geschlossen")
	}
	if l.e.s.Werkzeug != wz {
		t.Errorf("Klick in die Werkzeugleiste hinter der Tafel wechselte das Werkzeug (%v → %v)", wz, l.e.s.Werkzeug)
	}
}

// Die Menüzeile bleibt unter der Tafel stumm: Der Klick schließt nur die Übersicht und öffnet kein Menü.
func TestHilfeKlickAufMenuezeileSchliesstNurDieTafel(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	hilfeKlick(l, image.Pt(40, l.gtx().Dp(menuezeileDp)/2))
	if l.e.hilfeOffen {
		t.Error("der Klick auf die Menüzeile hat die Übersicht nicht geschlossen")
	}
	if l.e.menue.offen {
		t.Error("der Klick auf die Menüzeile hinter der Tafel hat ein Menü geöffnet")
	}
}

func TestHilfeKlickAufFusszeileWirktNicht(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	mx := l.gtx().Constraints.Max
	hilfeKlick(l, image.Pt(mx.X-60, mx.Y-l.gtx().Dp(fusszeileDp)/2)) // Zoomfeld rechts unten
	if l.e.hilfeOffen {
		t.Error("der Klick auf die Fußzeile hat die Übersicht nicht geschlossen")
	}
	if l.e.menue.offen {
		t.Error("der Klick auf ein Fußfeld hinter der Tafel hat eine Liste geöffnet")
	}
}
