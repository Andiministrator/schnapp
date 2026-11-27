// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// dreiKaesten ist eine Sitzung mit drei Kästen 20 × 20 in einer Zeile (Kennungen 1, 2, 3; x 10, 60, 120) auf einem
// Bild 200 × 100, Zoom 1, Fenster wie sitzung (f rechnet Bild- in Fensterpunkte).
func dreiKaesten(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []float64{10, 60, 120} {
		if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(x, 10), document.Pt(x+20, 30)),
			Fuellung: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	_ = d.VergissVerlauf()
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 148))
	return s
}

func minX(t *testing.T, s *Sitzung, id int) float64 {
	t.Helper()
	o, ok := s.D.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o.Grenzen().Min.X
}

func klick(s *Sitzung, p document.Punkt, umschalt bool) {
	s.Umschalt = umschalt
	s.Druecken(p, TastePrimaer)
	s.Loslassen(p)
	s.Umschalt = false
}

// ADR-0039 Punkt 2: Umschalt+Klick nimmt hinzu (wird Hauptauswahl) und wieder heraus; ein Klick ohne Umschalt wählt nur
// das getroffene, ein Klick ins Leere hebt alles auf.
func TestUmschaltKlickWaehlt(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 3}) || s.Auswahl != 3 {
		t.Fatalf("gewählt %v, Haupt %d", got, s.Auswahl)
	}
	klick(s, f(65, 15), true)
	if s.AnzahlGewaehlt() != 3 || s.Auswahl != 2 {
		t.Fatalf("drei: %v, Haupt %d", s.Gewaehlt(), s.Auswahl)
	}
	klick(s, f(65, 15), true) // die Hauptauswahl heraus: das oberste übrige wird Haupt
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 3}) || s.Auswahl != 3 {
		t.Fatalf("nach Herausnehmen: %v, Haupt %d", got, s.Auswahl)
	}
	klick(s, f(15, 15), false) // ohne Umschalt, ohne Bewegung: nur dieses
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) {
		t.Fatalf("Klick ohne Umschalt: %v", got)
	}
	klick(s, f(125, 15), true)
	klick(s, f(90, 80), false) // ins Leere
	if s.AnzahlGewaehlt() != 0 {
		t.Fatalf("Klick ins Leere: %v", s.Gewaehlt())
	}
	if s.Geaendert() {
		t.Error("Wählen änderte das Dokument")
	}
}

// Punkt 2: Ein Rahmen im Bild wählt die ganz enthaltenen; mit Umschalt kommen sie dazu. Strg+A wählt alle sichtbaren,
// nicht gesperrten.
func TestRahmenWaehlt(t *testing.T) {
	s := dreiKaesten(t)
	s.Druecken(f(1, 1), TastePrimaer) // weiter als die Treffer-Toleranz von Kasten 1
	s.Ziehen(f(50, 50))
	if r, ok := s.AuswahlRahmen(); !ok || !nah(r.Min, f(1, 1)) {
		t.Fatalf("Rahmen %v %v", r, ok)
	}
	s.Ziehen(f(90, 40)) // ganz: 1 und 2; 3 (ab x 120) nicht
	s.Loslassen(f(90, 40))
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) || s.Auswahl != 2 {
		t.Fatalf("Rahmen: %v, Haupt %d", got, s.Auswahl)
	}
	if _, ok := s.AuswahlRahmen(); ok {
		t.Error("Rahmen nach dem Loslassen noch da")
	}
	// Ein Rahmen, der 3 nur berührt, wählt nichts; ohne Umschalt ist die Auswahl dann leer.
	s.Druecken(f(110, 5), TastePrimaer)
	s.Ziehen(f(130, 50))
	s.Loslassen(f(130, 50))
	if s.AnzahlGewaehlt() != 0 {
		t.Fatalf("berührt: %v", s.Gewaehlt())
	}
	klick(s, f(15, 15), false)
	s.Umschalt = true
	s.Druecken(f(110, 5), TastePrimaer)
	s.Ziehen(f(150, 50))
	s.Loslassen(f(150, 50))
	s.Umschalt = false
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 3}) {
		t.Fatalf("Umschalt-Rahmen: %v", got)
	}
	// Ausgeblendet oder gesperrt: weder Rahmen noch Strg+A wählen es.
	if !s.SchalteGesperrt(2) {
		t.Fatal("sperren")
	}
	s.waehleNur(0)
	if !s.WaehleAlle() {
		t.Fatal("WaehleAlle")
	}
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 3}) || s.Auswahl != 3 {
		t.Fatalf("Strg+A: %v, Haupt %d", got, s.Auswahl)
	}
	if s.Geaendert() != true { // das Sperren war eine Änderung
		t.Error("Vorbedingung Sperren")
	}
}

// Punkt 3: Ziehen an einem gewählten verschiebt alle gewählten als einen Schritt; ohne Bewegung bleibt nur das
// angeklickte gewählt. Esc während des Zugs stellt alle her.
func TestMehrereVerschieben(t *testing.T) {
	s := dreiKaesten(t)
	s.SchalteEinrasten() // ohne Einrasten: der Weg ist genau der gezogene (Einrasten: TestMehrereRasten)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	vor := s.D.Stand()
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(25, 35))
	s.Loslassen(f(25, 35))
	if minX(t, s, 1) != 20 || minX(t, s, 3) != 130 || minX(t, s, 2) != 60 {
		t.Fatalf("x %v %v %v", minX(t, s, 1), minX(t, s, 2), minX(t, s, 3))
	}
	if s.AnzahlGewaehlt() != 2 || s.Auswahl != 1 {
		t.Errorf("nach dem Zug: %v, Haupt %d", s.Gewaehlt(), s.Auswahl)
	}
	if !s.Rueckgaengig() || s.D.Stand() != vor || minX(t, s, 1) != 10 || minX(t, s, 3) != 120 {
		t.Fatalf("ein Schritt zurück: x %v %v", minX(t, s, 1), minX(t, s, 3))
	}
	s.Druecken(f(125, 15), TastePrimaer)
	s.Ziehen(f(145, 15))
	s.Abbrechen()
	if minX(t, s, 1) != 10 || minX(t, s, 3) != 120 || s.D.ZugLaeuft() {
		t.Fatalf("Abbrechen: x %v %v", minX(t, s, 1), minX(t, s, 3))
	}
	klick(s, f(125, 15), false) // Klick ohne Bewegung auf ein gewähltes: nur dieses
	if got := s.Gewaehlt(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Klick: %v", got)
	}
	// Begrenzt: das rechte bleibt im Bild, auch das linke wandert nur so weit.
	klick(s, f(15, 15), true)
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(400, 15))
	s.Loslassen(f(400, 15))
	if x1, x3 := minX(t, s, 1), minX(t, s, 3); x3-x1 != 110 || x3 >= 200 {
		t.Errorf("begrenzt: x %v %v", x1, x3)
	}
}

// Punkt 3: Eingerastet wird der Rahmen um alle gewählten, an den übrigen Objekten (nicht an den gewählten selbst).
func TestMehrereRasten(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	// Rahmen 10…140, Mitte 75; um 10 nach rechts wäre die Mitte 85, fünf vom rechten Rand von Kasten 2 (80): rastet ein.
	s.Druecken(f(15, 15), TastePrimaer)
	s.Ziehen(f(25, 15))
	if len(s.Hilfslinien()) == 0 || minX(t, s, 1) != 15 {
		t.Errorf("eingerastet: x %v, Linien %v", minX(t, s, 1), s.Hilfslinien())
	}
	s.Loslassen(f(25, 15))
}

// Punkt 3: Löschen und Doppeln wirken auf alle gewählten, je ein Schritt; die Kopien sind danach gewählt.
func TestMehrereLoeschenUndDoppeln(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if !s.Doppeln() {
		t.Fatal("Doppeln")
	}
	if len(s.D.Objekte()) != 5 || s.AnzahlGewaehlt() != 2 || s.IstGewaehlt(1) || s.IstGewaehlt(3) {
		t.Fatalf("nach Doppeln: %d Objekte, gewählt %v", len(s.D.Objekte()), s.Gewaehlt())
	}
	if !s.Loeschen() || len(s.D.Objekte()) != 3 || s.AnzahlGewaehlt() != 0 {
		t.Fatalf("Löschen der Kopien: %d Objekte", len(s.D.Objekte()))
	}
	klick(s, f(15, 15), false)
	klick(s, f(65, 15), true)
	if !s.Loeschen() || len(s.D.Objekte()) != 1 {
		t.Fatalf("Löschen: %d Objekte", len(s.D.Objekte()))
	}
	// Rückgängig holt beide; die Auswahl ist leer, weil die Hauptauswahl beim Löschen endete.
	if !s.Rueckgaengig() || len(s.D.Objekte()) != 3 {
		t.Fatalf("Rückgängig: %d", len(s.D.Objekte()))
	}
}

// Punkt 1: Verschwindet die Hauptauswahl durch Rückgängig, rückt das oberste übrige nach.
func TestHauptauswahlRueckt(t *testing.T) {
	s := dreiKaesten(t)
	k, _ := s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(160, 60), document.Pt(180, 80)),
		Fuellung: color.NRGBA{A: 255}})
	klick(s, f(15, 15), false)
	klick(s, f(170, 70), true)
	if s.Auswahl != k {
		t.Fatalf("Haupt %d", s.Auswahl)
	}
	s.Rueckgaengig() // entfernt k wieder
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1}) || s.Auswahl != 1 {
		t.Errorf("nachgerückt: %v, Haupt %d", got, s.Auswahl)
	}
}

// Punkt 6: Ein Werkzeugwechsel hebt die weiteren gewählten auf; eine andere Stelle, die die Auswahl setzt (Ebenenliste),
// macht die weiteren hinfällig.
func TestWeitereEnden(t *testing.T) {
	s := dreiKaesten(t)
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SetzeWerkzeug(WerkzeugAuswahl)
	if s.AnzahlGewaehlt() > 1 {
		t.Errorf("nach Werkzeugwechsel: %v", s.Gewaehlt())
	}
	klick(s, f(15, 15), true)
	if !s.WaehleEbene(2) || s.AnzahlGewaehlt() != 1 {
		t.Errorf("Ebenenliste: %v", s.Gewaehlt())
	}	// Ein gewähltes, das gesperrt wird, fällt aus der Auswahl (Punkt 1).
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if !s.SchalteGesperrt(1) || s.IstGewaehlt(1) || !s.IstGewaehlt(3) {
		t.Errorf("gesperrt: %v", s.Gewaehlt())
	}
}

// Punkt 3: Ausrichten an Kanten und Mitten des Rahmens um alle; Verteilen mit gleichen Lücken, die äußeren bleiben;
// je ein Schritt. Ein Objekt, das das Bild verließe, verhindert die Handlung.
func TestAusrichtenUndVerteilen(t *testing.T) {
	s := dreiKaesten(t)
	// 2 nach unten und breiter machen, damit Ausrichten etwas zu tun hat.
	o, _ := s.D.Objekt(2)
	k := o.(document.Kasten)
	k.Bereich = document.R(document.Pt(60, 50), document.Pt(100, 90))
	_ = s.D.Ersetzen(k)
	if s.KannAusrichten() {
		t.Fatal("Ausrichten ohne Auswahl")
	}
	s.WaehleAlle()
	if !s.KannAusrichten() || !s.KannVerteilen() {
		t.Fatal("Kann… mit drei")
	}
	vor := s.D.Stand()
	if !s.Ausrichten(AusrichtenUnten) {
		t.Fatal("Ausrichten")
	}
	for _, id := range []int{1, 2, 3} {
		o, _ := s.D.Objekt(id)
		if g := o.Grenzen(); g.Max.Y != 90 {
			t.Errorf("unten: %d endet bei %v", id, g.Max.Y)
		}
	}
	s.Rueckgaengig()
	if s.D.Stand() != vor {
		t.Fatal("Ausrichten war nicht ein Schritt")
	}
	s.Ausrichten(AusrichtenLinks)
	if minX(t, s, 2) != 10 || minX(t, s, 3) != 10 {
		t.Errorf("links: %v %v", minX(t, s, 2), minX(t, s, 3))
	}
	s.Rueckgaengig()
	s.Ausrichten(AusrichtenMitteWaagerecht) // Rahmen 10…140, Mitte 75
	for _, id := range []int{1, 2, 3} {
		o, _ := s.D.Objekt(id)
		if g := o.Grenzen(); !nah(document.Pt((g.Min.X+g.Max.X)/2, 0), document.Pt(75, 0)) {
			t.Errorf("Mitte: %d bei %v", id, (g.Min.X+g.Max.X)/2)
		}
	}
	s.Rueckgaengig()
	// Verteilen waagerecht: Breiten 20, 40, 20 auf 10…140 → Lücken (130−80)/2 = 25: 2 beginnt bei 55.
	if !s.Verteilen(true) {
		t.Fatal("Verteilen")
	}
	if minX(t, s, 1) != 10 || minX(t, s, 2) != 55 || minX(t, s, 3) != 120 {
		t.Errorf("verteilt: %v %v %v", minX(t, s, 1), minX(t, s, 2), minX(t, s, 3))
	}
	s.Rueckgaengig()
	// Zwei gewählt: Verteilen geht nicht.
	klick(s, f(15, 15), false)
	klick(s, f(125, 15), true)
	if s.KannVerteilen() || s.Verteilen(true) {
		t.Error("Verteilen mit zwei")
	}
	// Zuschnitt bis x 70: Kasten 2 (60…100) ist halb zu sehen, 3 gar nicht. Rechts ausrichten schöbe Kasten 1 nach
	// 80…100, ganz ins Verborgene: abgelehnt, kein Schritt.
	if err := s.D.SetzeZuschnitt(image.Rect(0, 0, 70, 100)); err != nil {
		t.Fatal(err)
	}
	s.WaehleAlle()
	if got := s.Gewaehlt(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("nach Zuschnitt gewählt %v", got)
	}
	vor = s.D.Stand()
	if s.Ausrichten(AusrichtenRechts) || s.D.Stand() != vor || s.Abgelehnt() != AblehnungSichtbar {
		t.Errorf("ins Verborgene ausgerichtet: Stand neu %v, Ablehnung %v", s.D.Stand() != vor, s.Abgelehnt())
	}
	if !s.Ausrichten(AusrichtenLinks) || minX(t, s, 2) != 10 {
		t.Errorf("links geht: x %v", minX(t, s, 2))
	}
}
