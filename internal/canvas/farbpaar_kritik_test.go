// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Tests aus der Kritik zu Karte #219 (B-2 bis B-4): Lücken, die Mutanten in farbpaar.go und farbwahl.go offenließen.

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// TestTauschGleicheFarbenKeinSchritt: Ein Kasten mit gleicher Linie und Füllung tauscht ohne Rückgängig-Schritt.
func TestTauschGleicheFarbenKeinSchritt(t *testing.T) {
	s, _, _ := sitzung(t)
	id, err := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(60, 60), document.Pt(90, 90)),
		Strich: rot, Fuellung: rot.Farbe})
	if err != nil {
		t.Fatal(err)
	}
	s.Auswahl = id
	vor := schritte(t, s)
	if !s.TauscheFarben() {
		t.Fatal("Tausch gleicher Farben meldet false")
	}
	if n := schritte(t, s); n != vor {
		t.Errorf("%d Schritte, erwartet %d", n, vor)
	}
}

// TestTauschenUndFeldRuhen: Während eines Farbzugs und des Speicherns wird weder getauscht noch das Feld gewechselt.
func TestTauschenUndFeldRuhen(t *testing.T) {
	s, k, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeFeld(FeldFuellung)
	s.SetzeFarbe(blau)
	stil := s.Stil
	s.BeginneFarbZug()
	if s.KannTauschen() || s.TauscheFarben() || s.Stil != stil {
		t.Error("Tauschen während des Farbzugs")
	}
	if s.SetzeFeld(FeldVordergrund) || s.AktivesFeld() != FeldFuellung {
		t.Error("Feldwechsel während des Farbzugs")
	}
	s.BrecheFarbZugAb()
	s.Stil = stil
	s.hg.laeuft = true
	if s.KannTauschen() || s.TauscheFarben() || s.Stil != stil {
		t.Error("Tauschen am Werkzeug während des Speicherns")
	}
	s.hg.laeuft = false
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	alt, _ := s.D.Objekt(k)
	s.hg.laeuft = true
	if s.TauscheFarben() {
		t.Error("Tauschen am Objekt während des Speicherns")
	}
	if neu, _ := s.D.Objekt(k); neu != alt {
		t.Errorf("Objekt geändert: %+v", neu)
	}
}

// TestTauschAmObjektLaesstPaar: Tauschen am gewählten Kasten lässt das gemerkte Paar des Werkzeugs, wie es ist.
func TestTauschAmObjektLaesstPaar(t *testing.T) {
	s, k, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeFarbe(blau)
	s.SetzeFeld(FeldFuellung)
	s.SetzeFarbe(gelb)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	if !s.TauscheFarben() {
		t.Fatal("Tausch am Kasten")
	}
	if p := s.paar(WerkzeugKasten); p.vorn != blau || p.fuellung != gelb {
		t.Errorf("Paar %v / %v", p.vorn, p.fuellung)
	}
}

// TestFarbZugTexteingabeFuellung: Ein Farbzug in der Texteingabe färbt das Schild, nicht die Schrift, und die Farbe
// kommt in die zuletzt benutzten.
func TestFarbZugTexteingabeFuellung(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(120, 80), TastePrimaer)
	s.Loslassen(f(120, 80))
	s.Tippe("x")
	if !s.SetzeFeld(FeldFuellung) {
		t.Fatal("Füllung in der Texteingabe nicht wählbar")
	}
	want := (HSV{H: 240, S: 1, V: 1}).Farbe(200)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 240, S: 1, V: 1}, 200)
	s.BeendeFarbZug()
	s.Enter(false)
	if tx := letztes(t, s).(document.Text); tx.Hintergrund != want || tx.Farbe != Weiss {
		t.Errorf("Text %v auf %v", tx.Farbe, tx.Hintergrund)
	}
	if z := s.Zuletzt(); len(z) == 0 || z[0] != want {
		t.Errorf("Zuletzt %v", z)
	}
}

// TestZuletztNachZugAmObjekt: Nach einem Farbzug an der Füllung des gewählten Kastens steht genau diese Farbe in
// den zuletzt benutzten.
func TestZuletztNachZugAmObjekt(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	s.SetzeFeld(FeldFuellung)
	want := (HSV{H: 120, S: 1, V: 0.5}).Farbe(255)
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 120, S: 1, V: 0.5}, 255)
	s.BeendeFarbZug()
	if z := s.Zuletzt(); len(z) != 1 || z[0] != want {
		t.Errorf("Zuletzt %v, erwartet [%v]", z, want)
	}
}

// TestKeineBehaeltFarbton: „Keine“ setzt nur die Deckkraft auf 0 und kommt nicht in die zuletzt benutzten.
func TestKeineBehaeltFarbton(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeFeld(FeldFuellung)
	s.SetzeFarbe(blau)
	if !s.KeineFuellung() {
		t.Fatal("Keine")
	}
	if want := (color.NRGBA{B: 200}); s.Stil.Fuellung != want {
		t.Errorf("Füllung %v, erwartet %v", s.Stil.Fuellung, want)
	}
	if z := s.Zuletzt(); len(z) != 0 {
		t.Errorf("Zuletzt %v, erwartet leer", z)
	}
}

// TestZuletztOhneDurchsichtige: Deckkraft 0 (wie „Keine“) kommt nicht in die zuletzt benutzten (Kritik B-3).
func TestZuletztOhneDurchsichtige(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SetzeFeld(FeldFuellung)
	s.SetzeHex("#00AA00")
	s.SetzeDeckkraftText("0")
	for _, c := range s.Zuletzt() {
		if c.A == 0 {
			t.Errorf("ganz durchsichtige Farbe in Zuletzt: %v", c)
		}
	}
	s.BeginneFarbZug()
	s.FarbZugHSV(HSV{H: 30, S: 1, V: 1}, 0)
	s.BeendeFarbZug()
	for _, c := range s.Zuletzt() {
		if c.A == 0 {
			t.Errorf("ganz durchsichtige Farbe nach dem Farbzug in Zuletzt: %v", c)
		}
	}
}

// TestTauschenOhneFarbwerkzeug: Ein Werkzeug ohne Farbe tauscht nicht unsichtbar das Paar des letzten Farbwerkzeugs
// (Kritik B-4); Feld Füllung und „Keine“ sind dort grau.
func TestTauschenOhneFarbwerkzeug(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugText)
	for _, w := range []Werkzeug{WerkzeugSchwaerzen, WerkzeugZuschneiden, WerkzeugStauchen, WerkzeugAuswahl} {
		s.SetzeWerkzeug(w)
		s.Auswahl = 0
		stil := s.Stil
		if s.KannTauschen() || s.TauscheFarben() || s.Stil != stil {
			t.Errorf("Tauschen bei Werkzeug %v: Stil %v/%v", w, s.Stil.Farbe, s.Stil.Fuellung)
		}
		if s.SetzeFeld(FeldFuellung) || s.KeineFuellung() || s.Stil != stil {
			t.Errorf("Füllung bei Werkzeug %v wählbar", w)
		}
	}
}
