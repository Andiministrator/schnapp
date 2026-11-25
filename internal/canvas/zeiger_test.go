// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Karte #323 (ADR-0021 Nachtrag 18): Der Mauszeiger folgt der Lage — Griffe zeigen die Richtung, Objekte das
// Verschieben, sonst das Werkzeug; ein laufender Zug behält seine Form.

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

func zeigerBei(s *Sitzung, p document.Punkt) ZeigerArt { return s.Zeiger(&p) }

func TestZeigerUeberGriffenUndObjekten(t *testing.T) {
	s, k, p := sitzung(t) // Kasten (10,10)–(50,50), Pfeil (100,50)–(180,50)
	if z := s.Zeiger(nil); z != ZeigerStandard {
		t.Errorf("ohne Zeiger: %v", z)
	}
	if z := zeigerBei(s, f(5, 90)); z != ZeigerStandard {
		t.Errorf("leere Fläche: %v", z)
	}
	if z := zeigerBei(s, f(30, 30)); z != ZeigerVerschieben {
		t.Errorf("über dem Kasten: %v", z)
	}
	// Ohne Auswahl keine Griffe: die Ecke ist nur Objekt.
	if z := zeigerBei(s, f(10, 10)); z != ZeigerVerschieben {
		t.Errorf("Ecke ohne Auswahl: %v", z)
	}
	s.Auswahl = k
	for _, f0 := range []struct {
		x, y float64
		want ZeigerArt
	}{
		{10, 10, ZeigerNWSO}, {50, 50, ZeigerNWSO}, {50, 10, ZeigerNOSW}, {10, 50, ZeigerNOSW},
		{30, 10, ZeigerNS}, {30, 50, ZeigerNS}, {10, 30, ZeigerOW}, {50, 30, ZeigerOW},
		{30, 30, ZeigerVerschieben},
	} {
		if z := zeigerBei(s, f(f0.x, f0.y)); z != f0.want {
			t.Errorf("Kasten gewählt, (%v,%v): %v, Soll %v", f0.x, f0.y, z, f0.want)
		}
	}
	s.Auswahl = p
	for _, pt := range []document.Punkt{f(100, 50), f(180, 50), f(140, 50)} { // Anfang, Spitze, Bogengriff
		if z := zeigerBei(s, pt); z != ZeigerGreifen {
			t.Errorf("Pfeilgriff %v: %v", pt, z)
		}
	}
}

func TestZeigerJeWerkzeug(t *testing.T) {
	s, k, _ := sitzung(t)
	for _, f0 := range []struct {
		w    Werkzeug
		want ZeigerArt
	}{
		{WerkzeugPfeil, ZeigerFadenkreuz}, {WerkzeugKasten, ZeigerFadenkreuz}, {WerkzeugText, ZeigerText},
		{WerkzeugSprechblase, ZeigerText}, {WerkzeugSchwaerzen, ZeigerFadenkreuz}, {WerkzeugLupe, ZeigerFadenkreuz},
	} {
		s.SetzeWerkzeug(f0.w)
		s.Auswahl = 0
		if z := zeigerBei(s, f(30, 30)); z != f0.want {
			t.Errorf("%v über dem Kasten: %v, Soll %v", f0.w, z, f0.want)
		}
	}
	// Ein Zeichenwerkzeug mit gewähltem Kasten: dessen Griffe gehen vor (wie Druecken).
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Auswahl = k
	if z := zeigerBei(s, f(50, 50)); z != ZeigerNWSO {
		t.Errorf("Griff beim Kastenwerkzeug: %v", z)
	}
	// Zuschneiden: keine Objektgriffe, aber die Kanten des Rahmens.
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if z := zeigerBei(s, f(0, 0)); z != ZeigerNWSO {
		t.Errorf("Zuschneiden, Ecke links oben: %v", z)
	}
	if z := zeigerBei(s, f(100, 0)); z != ZeigerNS {
		t.Errorf("Zuschneiden, Kante oben: %v", z)
	}
	if z := zeigerBei(s, f(50, 50)); z != ZeigerFadenkreuz {
		t.Errorf("Zuschneiden innen: %v", z)
	}
	// Gesperrte Auswahl zeigt keine Griffe.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	o, _ := s.D.Objekt(k)
	kasten := o.(document.Kasten)
	kasten.Gesperrt = true
	if err := s.D.Ersetzen(kasten); err != nil {
		t.Fatal(err)
	}
	s.Auswahl = k
	if z := zeigerBei(s, f(10, 10)); z == ZeigerNWSO {
		t.Error("Griff an gesperrter Auswahl")
	}
}

func TestZeigerImZug(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	s.Druecken(f(50, 50), TastePrimaer) // Griff SO
	s.Ziehen(f(90, 90))                 // weit weg vom Griff
	if z := zeigerBei(s, f(90, 90)); z != ZeigerNWSO {
		t.Errorf("Griffzug: %v", z)
	}
	s.Loslassen(f(90, 90))
	s.Druecken(f(30, 30), TastePrimaer) // Kasten verschieben
	s.Ziehen(f(35, 35))
	if z := zeigerBei(s, f(200, 120)); z != ZeigerVerschieben {
		t.Errorf("Objektzug: %v", z)
	}
	s.Loslassen(f(35, 35))
	s.Auswahl = 0
	s.Druecken(f(5, 95), TastePrimaer) // leere Fläche: Ansicht ziehen
	s.Ziehen(f(40, 120))
	if z := zeigerBei(s, f(40, 120)); z != ZeigerGreifend {
		t.Errorf("Ansicht ziehen: %v", z)
	}
	s.Loslassen(f(40, 120))
	s.SchaltePipette()
	if z := zeigerBei(s, f(30, 30)); z != ZeigerFadenkreuz {
		t.Errorf("Pipette: %v", z)
	}
}

// Kritik K-5: Während Speichern oder Laden (Ruhe) reagieren Griffe und Objekte nicht; der Zeiger verspricht es auch nicht.
func TestZeigerBeiSperre(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	s.SetzeRuhe(true)
	if z := zeigerBei(s, f(50, 50)); z != ZeigerStandard {
		t.Errorf("Griff während der Ruhe: %v", z)
	}
	s.SetzeRuhe(false)
	if z := zeigerBei(s, f(50, 50)); z != ZeigerNWSO {
		t.Errorf("Griff nach der Ruhe: %v", z)
	}
}
