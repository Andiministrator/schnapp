// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 19 Punkt 5, Nachtrag 20 (je Ende): Ohne Auswahl gilt die Wahl für folgende Pfeile; mit gewähltem Pfeil ändert sie ihn als
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
	if a, e := s.Pfeilform(); a != document.KopfKeine || e != document.KopfDreieck {
		t.Errorf("Voreinstellung %v %v", a, e)
	}
	vor := s.D.Stand()
	if s.SetzeKopf(true, document.KopfPunkt) || s.SetzeKopf(false, document.KopfKerbe) || s.D.Stand() != vor {
		t.Error("ohne Auswahl ein Rückgängig-Schritt")
	}
	zieh(s, f(20, 20), f(150, 80))
	pf, ok := letztes(t, s).(document.Pfeil)
	if !ok || pf.KopfAnfang() != document.KopfPunkt || pf.Ende != document.KopfKerbe {
		t.Fatalf("neuer Pfeil %+v, %v", pf, ok)
	}
	// Mit Auswahl: ein Schritt, rückgängig zurück; das Werkzeug merkt die Wahl.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = pf.ID
	if !s.Seitenleiste().Pfeilform {
		t.Error("keine Pfeilform am gewählten Pfeil")
	}
	vor = s.D.Stand()
	if !s.SetzeKopf(false, document.KopfKeine) || s.D.Stand() == vor {
		t.Fatal("SetzeKopf am Ende änderte den Pfeil nicht")
	}
	if _, e := s.Pfeilform(); e != document.KopfKeine || s.Stil.KopfEnde != document.KopfKeine {
		t.Errorf("Ende %v, Stil %v", e, s.Stil.KopfEnde)
	}
	if s.SetzeKopf(false, document.KopfKeine) {
		t.Error("dieselbe Wahl ist ein Schritt")
	}
	if !s.SetzeKopf(true, document.KopfQuer) {
		t.Fatal("SetzeKopf am Anfang änderte den Pfeil nicht")
	}
	if a, _ := s.Pfeilform(); a != document.KopfQuer || s.Stil.KopfAnfang.Art() != document.KopfQuer {
		t.Errorf("Anfang %v, Stil %v", a, s.Stil.KopfAnfang.Art())
	}
	s.Rueckgaengig()
	if a, _ := s.Pfeilform(); a != document.KopfPunkt {
		t.Errorf("nach Rückgängig Anfang %v", a)
	}
	s.Rueckgaengig()
	if _, e := s.Pfeilform(); e != document.KopfKerbe {
		t.Errorf("nach Rückgängig Ende %v", e)
	}
	// Ein Schnellstil ändert Farbe und Breite, nicht die Form (Punkt 6).
	st := Schnellstil{Art: SchnellPfeil, Farbe: Beschriftungsfarben[1], Breite: 12}
	if !s.WendeSchnellstilAn(st) {
		t.Fatal("Schnellstil nicht angewandt")
	}
	if p, _ := s.gewaehlterPfeil(); p.KopfAnfang() != document.KopfPunkt || p.Ende != document.KopfKerbe || p.Strich.Breite != 12 {
		t.Errorf("nach Schnellstil %+v", p)
	}
	// Während des Speicherns ruht das Bearbeiten.
	s.SetzeRuhe(true)
	if s.SetzeKopf(false, document.KopfWinkel) {
		t.Error("SetzeKopf in der Ruhe")
	}
}

// Ein Linie trifft man wie einen Pfeil am Schaft; ihre Enden sind rund.
func TestPfeilformTrefferLinie(t *testing.T) {
	linie := document.Pfeil{ID: 1, A: document.Pt(0, 0), B: document.Pt(100, 0), K: document.Pt(50, 0), Strich: rot,
		Ende: document.KopfKeine}
	for _, fall := range []struct {
		p  document.Punkt
		ok bool
	}{{document.Pt(50, 1), true}, {document.Pt(101, 0), true}, {document.Pt(99, 10), false}} {
		if _, ok := Treffer([]document.Objekt{linie}, fall.p, 2); ok != fall.ok {
			t.Errorf("%v: %v, Soll %v", fall.p, ok, fall.ok)
		}
	}
}
