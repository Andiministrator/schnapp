// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 19 Punkt 5: Ohne Auswahl gilt die Wahl für folgende Pfeile; mit gewähltem Pfeil ändert sie ihn als
// einen Rückgängig-Schritt und merkt sie fürs Werkzeug. Die rechte Leiste zeigt die Zeilen bei Werkzeug und Auswahl.
func TestPfeilformWerkzeugUndAuswahl(t *testing.T) {
	s, _, _ := sitzung(t)
	if s.Seitenleiste().Pfeilform {
		t.Error("Pfeilform beim Auswahlwerkzeug ohne Pfeil")
	}
	s.SetzeWerkzeug(WerkzeugPfeil)
	if !s.Seitenleiste().Pfeilform {
		t.Error("keine Pfeilform beim Werkzeug Pfeil")
	}
	if k, e := s.Pfeilform(); k != document.KopfDreieck || e != document.EndenEnde {
		t.Errorf("Voreinstellung %v %v", k, e)
	}
	vor := s.D.Stand()
	if s.SetzeKopf(document.KopfPunkt) || s.SetzeEnden(document.EndenBeide) || s.D.Stand() != vor {
		t.Error("ohne Auswahl ein Rückgängig-Schritt")
	}
	zieh(s, f(20, 20), f(150, 80))
	pf, ok := letztes(t, s).(document.Pfeil)
	if !ok || pf.Kopf != document.KopfPunkt || pf.Enden != document.EndenBeide {
		t.Fatalf("neuer Pfeil %+v, %v", pf, ok)
	}
	// Mit Auswahl: ein Schritt, rückgängig zurück; das Werkzeug merkt die Wahl.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = pf.ID
	if !s.Seitenleiste().Pfeilform {
		t.Error("keine Pfeilform am gewählten Pfeil")
	}
	vor = s.D.Stand()
	if !s.SetzeEnden(document.EndenKeine) || s.D.Stand() == vor {
		t.Fatal("SetzeEnden änderte den Pfeil nicht")
	}
	if _, e := s.Pfeilform(); e != document.EndenKeine || s.Stil.Enden != document.EndenKeine {
		t.Errorf("Enden %v, Stil %v", e, s.Stil.Enden)
	}
	if s.SetzeEnden(document.EndenKeine) {
		t.Error("dieselbe Wahl ist ein Schritt")
	}
	if !s.SetzeKopf(document.KopfQuer) {
		t.Fatal("SetzeKopf änderte den Pfeil nicht")
	}
	s.Rueckgaengig()
	if k, _ := s.Pfeilform(); k != document.KopfPunkt {
		t.Errorf("nach Rückgängig Kopf %v", k)
	}
	s.Rueckgaengig()
	if _, e := s.Pfeilform(); e != document.EndenBeide {
		t.Errorf("nach Rückgängig Enden %v", e)
	}
	// Ein Schnellstil ändert Farbe und Breite, nicht die Form (Punkt 6).
	st := Schnellstil{Art: SchnellPfeil, Farbe: Beschriftungsfarben[1], Breite: 12}
	if !s.WendeSchnellstilAn(st) {
		t.Fatal("Schnellstil nicht angewandt")
	}
	if p, _ := s.gewaehlterPfeil(); p.Kopf != document.KopfPunkt || p.Enden != document.EndenBeide || p.Strich.Breite != 12 {
		t.Errorf("nach Schnellstil %+v", p)
	}
	// Während des Speicherns ruht das Bearbeiten.
	s.SetzeRuhe(true)
	if s.SetzeKopf(document.KopfWinkel) {
		t.Error("SetzeKopf in der Ruhe")
	}
}

// Ein Linie trifft man wie einen Pfeil am Schaft; ihre Enden sind rund.
func TestPfeilformTrefferLinie(t *testing.T) {
	linie := document.Pfeil{ID: 1, A: document.Pt(0, 0), B: document.Pt(100, 0), K: document.Pt(50, 0), Strich: rot,
		Enden: document.EndenKeine}
	for _, fall := range []struct {
		p  document.Punkt
		ok bool
	}{{document.Pt(50, 1), true}, {document.Pt(101, 0), true}, {document.Pt(99, 10), false}} {
		if _, ok := Treffer([]document.Objekt{linie}, fall.p, 2); ok != fall.ok {
			t.Errorf("%v: %v, Soll %v", fall.p, ok, fall.ok)
		}
	}
}
