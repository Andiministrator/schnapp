// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"slices"
	"testing"
)

// gruppiert sind dreiKaesten mit 1 und 2 als Gruppe (Strg+G über die Mehrfachauswahl).
func gruppiert(t *testing.T) *Sitzung {
	t.Helper()
	s := dreiKaesten(t)
	s.setzeAuswahl(2, []int{1})
	if !s.KannGruppieren() || !s.Gruppieren() {
		t.Fatal("Gruppieren")
	}
	if s.GruppeVon(1) == 0 || s.GruppeVon(1) != s.GruppeVon(2) || s.GruppeVon(3) != 0 {
		t.Fatalf("Gruppen %d %d %d", s.GruppeVon(1), s.GruppeVon(2), s.GruppeVon(3))
	}
	s.waehleNur(0)
	return s
}

// ADR-0044 Punkt 2 und 3: Gruppieren ist ein Schritt; ein Klick auf ein Mitglied wählt die Gruppe, das angeklickte wird
// Hauptauswahl; Ziehen verschiebt die Gruppe.
func TestGruppeKlickWaehltGanz(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(65, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) || s.Auswahl != 2 {
		t.Fatalf("Klick: %v Haupt %d", got, s.Auswahl)
	}
	klick(s, f(125, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Klick auf ungruppiertes: %v", got)
	}
	// Ziehen an einem nicht gewählten Mitglied verschiebt die ganze Gruppe.
	x1, x2 := minX(t, s, 1), minX(t, s, 2)
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(25, 15))
	s.Loslassen(f(25, 15))
	if minX(t, s, 1) != x1+10 || minX(t, s, 2) != x2+10 || s.Auswahl != 1 {
		t.Errorf("Ziehen: %v %v Haupt %d", minX(t, s, 1), minX(t, s, 2), s.Auswahl)
	}
	s.Rueckgaengig()
	if minX(t, s, 1) != x1 || minX(t, s, 2) != x2 {
		t.Error("Verschieben war nicht ein Schritt")
	}
	s.Rueckgaengig()
	if s.GruppeVon(1) != 0 {
		t.Error("Rückgängig hebt das Gruppieren nicht auf")
	}
}

// Umschalt+Klick nimmt die Gruppe dazu und heraus; der Rahmen nimmt eine Gruppe nur ganz.
func TestGruppeUmschaltUndRahmen(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(125, 15), false)
	klick(s, f(15, 15), true)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2, 3}) || s.Auswahl != 1 {
		t.Fatalf("Umschalt dazu: %v Haupt %d", got, s.Auswahl)
	}
	klick(s, f(65, 15), true)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Umschalt heraus: %v", got)
	}
	// Rahmen um Kasten 2 und 3: die Gruppe ist nicht ganz drin, nur 3 wird gewählt.
	s.waehleNur(0)
	s.Druecken(f(45, 70), TastePrimaer)
	s.Ziehen(f(150, 2))
	s.Loslassen(f(150, 2))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Errorf("Rahmen über halbe Gruppe: %v", got)
	}
	s.waehleNur(0)
	s.Druecken(f(2, 70), TastePrimaer)
	s.Ziehen(f(95, 2))
	s.Loslassen(f(95, 2))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Rahmen um die Gruppe: %v", got)
	}
}

// Die Ebenenliste wählt zeilenweise; Aufheben nimmt die gewählten aus der Gruppe.
func TestGruppeEbenenlisteUndAufheben(t *testing.T) {
	s := gruppiert(t)
	s.WaehleEbene(1)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) {
		t.Fatalf("Liste: %v", got)
	}
	z := s.Ebenen()
	if z[0].Gruppe != 0 || z[1].Gruppe == 0 || z[1].Gruppe != z[2].Gruppe {
		t.Errorf("Gruppe in der Liste: %d %d %d", z[0].Gruppe, z[1].Gruppe, z[2].Gruppe)
	}
	if !s.KannGruppierungAufheben() || !s.GruppierungAufheben() {
		t.Fatal("Aufheben")
	}
	// Kasten 2 trägt die Kennung noch, ist aber allein: keine wirksame Gruppe.
	if s.GruppeVon(1) != 0 || s.GruppeVon(2) != 0 {
		t.Errorf("nach Aufheben: %d %d", s.GruppeVon(1), s.GruppeVon(2))
	}
	klick(s, f(65, 15), false)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{2}) {
		t.Errorf("einzelnes Restmitglied wählt %v", got)
	}
	s.waehleNur(0)
	if s.KannGruppieren() {
		t.Error("Gruppieren ohne Auswahl möglich")
	}
	s.setzeAuswahl(2, []int{1})
	s.Gruppieren()
	s.waehleNur(0)
	klick(s, f(15, 15), false)
	if s.KannGruppieren() {
		t.Error("eine ganze Gruppe lässt sich noch einmal gruppieren")
	}
}

// Punkt 4: Strg+H und Strg+L an einer gewählten Gruppe wirken auf alle Mitglieder (ein Schritt); Doppeln gibt den
// Kopien eine neue Gruppe, die Kopie eines einzelnen Mitglieds steht allein.
func TestGruppeAusblendenSperrenDoppeln(t *testing.T) {
	s := gruppiert(t)
	klick(s, f(15, 15), false)
	g := s.GruppeVon(1)
	if !s.SchalteGesperrtGruppe(s.Auswahl) {
		t.Fatal("Sperren")
	}
	for _, id := range []int{1, 2} {
		if o, _ := s.D.Objekt(id); !o.Ebenenwerte().Gesperrt {
			t.Errorf("Mitglied %d nicht gesperrt", id)
		}
	}
	s.Rueckgaengig()
	if o, _ := s.D.Objekt(2); o.Ebenenwerte().Gesperrt {
		t.Error("Sperren der Gruppe war nicht ein Schritt")
	}
	klick(s, f(15, 15), false)
	s.SchalteAusgeblendetGruppe(s.Auswahl)
	if o, _ := s.D.Objekt(2); !o.Ebenenwerte().Ausgeblendet {
		t.Error("Mitglied 2 nicht ausgeblendet")
	}
	s.Rueckgaengig()
	klick(s, f(15, 15), false)
	n := len(s.D.Objekte())
	if !s.Doppeln() || len(s.D.Objekte()) != n+2 {
		t.Fatal("Doppeln der Gruppe")
	}
	ids := s.Gewaehlt()
	g2 := s.GruppeVon(ids[0])
	if g2 == 0 || g2 == g || s.GruppeVon(ids[1]) != g2 {
		t.Errorf("Kopien: Gruppen %d %d, Original %d", g2, s.GruppeVon(ids[1]), g)
	}
	s.WaehleEbene(1)
	s.Doppeln()
	if s.GruppeVon(s.Auswahl) != 0 {
		t.Error("Kopie eines einzelnen Mitglieds gehört zur Gruppe")
	}
	if o, _ := s.D.Objekt(s.Auswahl); o.Ebenenwerte().Gruppe != 0 {
		t.Error("Kopie trägt noch die Kennung")
	}
}

// Punkt 3: Rechtsklick auf ein nicht gewähltes Mitglied wählt die ganze Gruppe.
func TestGruppeRechtsklick(t *testing.T) {
	s := gruppiert(t)
	if _, objekt := s.KontextBei(f(15, 15)); !objekt {
		t.Fatal("kein Objekt getroffen")
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) || s.Auswahl != 1 {
		t.Errorf("Rechtsklick: %v Haupt %d", got, s.Auswahl)
	}
}
