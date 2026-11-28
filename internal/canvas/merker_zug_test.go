// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import "testing"

// F-974: Im Zug mehrerer (Ziehen) und im Stilzug über mehrere (Regler) rechnet der Merker der Auswahl einmal, nicht
// je Abfrage oder Schritt (F-953); nach dem Zug wieder neu.
func TestMerkerImZugEinmal(t *testing.T) {
	n := 0
	merkerRechnet = func() { n++ }
	t.Cleanup(func() { merkerRechnet = nil })

	s := dreiKaesten(t)
	s.setzeAuswahl(3, []int{1, 2, 3})
	s.Druecken(f(15, 15), TastePrimaer)
	if s.zug != zugMehrere {
		t.Fatalf("Vorbedingung: Zug %v", s.zug)
	}
	n = 0
	for i := range 5 {
		s.Ziehen(f(20+float64(i), 15))
		_ = s.Gewaehlt()
		_ = s.AnzahlGewaehlt()
	}
	if n > 1 {
		t.Errorf("Zug mehrerer: %d Berechnungen, erwartet höchstens 1", n)
	}
	s.Loslassen(f(25, 15))

	if !s.BeginneBreiteZug() {
		t.Fatal("Breitezug beginnt nicht")
	}
	n = 0
	for i := range 5 {
		s.BreiteZugSetze(float64(3 + i))
		_ = s.Gewaehlt()
	}
	if n > 1 {
		t.Errorf("Stilzug über mehrere: %d Berechnungen, erwartet höchstens 1", n)
	}
	s.BeendeBreiteZug()
	n = 0
	_ = s.Gewaehlt()
	if n != 1 {
		t.Errorf("nach dem Zug: %d Berechnungen, erwartet 1 (neuer Stand)", n)
	}
}
