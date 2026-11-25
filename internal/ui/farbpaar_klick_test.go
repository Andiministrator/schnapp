// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// F-817 (ADR-0031 Nachtrag 12): Ein Klick auf das Farbpaar unter der Werkzeugleiste zeigt den Farbwähler der rechten
// Leiste — aufgeklappt, mit Vorrang vor den Ebenen und ins Bild gerollt —, auch wenn die Leiste schon offen ist. Nur
// wenn die Leiste die Farbe nicht zeigen kann, öffnet er die kleine Farbliste.

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

func (l *menueLauf) klickFarbpaar(fuellung bool) {
	l.t.Helper()
	_, farbe := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	vorn, fuel, _ := farbpaarRahmen(farbe)
	if fuellung {
		l.klick(mittenPunkt(fuel))
		return
	}
	l.klick(mittenPunkt(vorn))
}

func TestFarbpaarKlickZeigtFarbwaehler(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if l.e.seiteZu || !l.e.seite.farbeGezeigt {
		t.Fatalf("Vorbedingung: Leiste zu %v, Farbe gezeigt %v", l.e.seiteZu, l.e.seite.farbeGezeigt)
	}
	l.klickFarbpaar(false)
	if l.e.menue.offen {
		t.Error("Leiste offen: Klick öffnete die Farbliste statt den Farbwähler zu zeigen")
	}
	if l.e.s.AktivesFeld() != canvas.FeldVordergrund || !l.e.seite.farbeGezeigt {
		t.Errorf("Feld %v, Farbe gezeigt %v", l.e.s.AktivesFeld(), l.e.seite.farbeGezeigt)
	}
	l.klickFarbpaar(true)
	if l.e.menue.offen || l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Errorf("Füllung: Liste offen %v, Feld %v", l.e.menue.offen, l.e.s.AktivesFeld())
	}
}

func TestFarbpaarKlickKlapptAuf(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.seite.farbeZu = true
	l.setle()
	if l.e.seite.farbeGezeigt {
		t.Fatal("Vorbedingung: Abschnitt Farbe eingeklappt")
	}
	l.klickFarbpaar(false)
	if l.e.seite.farbeZu || !l.e.seite.farbeGezeigt || l.e.menue.offen {
		t.Errorf("eingeklappter Abschnitt: zu %v, gezeigt %v, Liste %v", l.e.seite.farbeZu, l.e.seite.farbeGezeigt,
			l.e.menue.offen)
	}
	l.e.seiteZu = true
	l.setle()
	l.klickFarbpaar(false)
	if l.e.seiteZu || !l.e.seite.farbeGezeigt || l.e.menue.offen {
		t.Errorf("eingeklappte Leiste: zu %v, Farbe %v, Liste %v", l.e.seiteZu, l.e.seite.farbeGezeigt, l.e.menue.offen)
	}
}

// Ebenen mit Vorrang (Platzmangel, ein Klick auf „Ebenen“): Der Klick aufs Farbpaar gibt der Farbe den Vorrang zurück.
func TestFarbpaarKlickNimmtEbenenVorrang(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 560)
	l.setle()
	l.e.ebenen.vorrang = true
	l.setle()
	if l.e.seite.farbeGezeigt {
		t.Skip("bei dieser Höhe kein Platzmangel — Fall nicht herstellbar")
	}
	l.klickFarbpaar(false)
	if !l.e.seite.farbeGezeigt || l.e.ebenen.vorrang {
		t.Errorf("Farbe gezeigt %v, Ebenen-Vorrang %v", l.e.seite.farbeGezeigt, l.e.ebenen.vorrang)
	}
}

// In niedrigen Fenstern rollt die Leiste den Abschnitt Farbe ins Bild.
func TestFarbpaarKlickRolltFarbwaehlerInsBild(t *testing.T) {
	l, _ := ebenenLauf(t)
	geprueft := false
	for h := 320; h <= 600; h += 20 {
		l.fenst = image.Pt(1200, h)
		l.rolle(-1 << 20) // ganz nach oben
		vorn := l.e.seite.paarRahmen[paarVorn]
		if l.imSichtfeld(vorn) {
			continue // Farbwähler schon zu sehen, nichts zu rollen
		}
		geprueft = true
		l.klickFarbpaar(false)
		if !l.imSichtfeld(l.e.seite.paarRahmen[paarVorn]) {
			t.Errorf("Höhe %d: Farbpaar der Leiste %v nach dem Klick nicht im Bild (Versatz %d, Sicht %v)", h,
				l.e.seite.paarRahmen[paarVorn], l.e.seite.roll.rollen, l.e.seite.roll.sicht)
		}
		if !l.e.seite.roll.ziel.Empty() {
			t.Errorf("Höhe %d: Rollziel nicht verbraucht", h)
		}
	}
	if !geprueft {
		t.Fatal("keine Fensterhöhe, in der der Farbwähler oben weggerollt ist")
	}
}

// Zu schmales Fenster: Die Leiste kann die Farbe nicht zeigen, der Klick öffnet die Farbliste am Farbfeld.
func TestFarbpaarKlickSchmalOeffnetListe(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(600, 800))
	if seitePasst(l.gtx()) {
		t.Fatal("Vorbedingung: Fenster zu breit für den Fall")
	}
	l.klickFarbpaar(false)
	if !l.e.menue.offen || l.e.menue.welches != listeFarbe || !l.e.listeAmFarbKnopf {
		t.Errorf("schmal: Liste offen %v, welche %d, am Farbfeld %v", l.e.menue.offen, l.e.menue.welches,
			l.e.listeAmFarbKnopf)
	}
}
