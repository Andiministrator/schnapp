// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-386, ADR-0023 Nachtrag 5: Mit der Wahl des Werkzeugs Zuschneiden liegt ein Rahmen über der ganzen Ansicht, mit
// acht Griffen an Ecken und Kantenmitten; die ganze Kante ist greifbar, eine Ecke gewinnt.
func TestZuschnittRahmenNachDerWahl(t *testing.T) {
	s := verlaufSitzung(t) // 200×100 in 248×148, Zoom 1, Versatz 24
	if _, _, ok := s.ZuschnittRahmen(); ok {
		t.Fatal("Rahmen ohne das Werkzeug Zuschneiden")
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	a, b, ok := s.ZuschnittRahmen()
	if !ok || a != f(0, 0) || b != f(200, 100) {
		t.Fatalf("Rahmen %v %v %v, Soll die ganze Ansicht", a, b, ok)
	}
	for _, fall := range []struct {
		p   document.Punkt
		art GriffArt
	}{
		{f(0, 0), GriffNW}, {f(100, 0), GriffN}, {f(200, 0), GriffNO}, {f(200, 50), GriffO},
		{f(200, 100), GriffSO}, {f(100, 100), GriffS}, {f(0, 100), GriffSW}, {f(0, 50), GriffW},
		{f(30, -5), GriffN}, {f(203, 80), GriffO}, {f(4, 4), GriffNW}, {f(-6, 50), GriffW},
	} {
		if art, ok := s.zuschnittGriff(fall.p); !ok || art != fall.art {
			t.Errorf("Griff bei %v: %v %v, Soll %v", fall.p, art, ok, fall.art)
		}
	}
	for _, p := range []document.Punkt{f(100, 50), f(7, 50), f(100, 7), f(-7, 50), f(100, 108)} {
		if art, ok := s.zuschnittGriff(p); ok {
			t.Errorf("Griff bei %v: %v, Soll keiner", p, art)
		}
	}
	if s.Bildflaeche() != image.Rect(0, 0, 200, 100) {
		t.Errorf("Bildfläche ohne Zuschnitt %v", s.Bildflaeche())
	}
}

// Ein Griff nach innen verkleinert den Zuschnitt beim Loslassen (ein Rückgängig-Schritt); die Ecke bewegt zwei Kanten;
// zurückgeschoben (nach außen in den abgedunkelten Teil) kommt das Weggeschnittene wieder.
func TestZuschnittGriffZiehen(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.Druecken(f(200, 50), TastePrimaer) // rechte Kante
	s.Ziehen(f(150, 50))
	if a, b, ok := s.ZuschnittRahmen(); !ok || a != f(0, 0) || b != f(150, 100) {
		t.Errorf("Rahmen beim Ziehen %v %v", a, b)
	}
	if s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Error("Zuschnitt schon vor dem Loslassen gesetzt")
	}
	s.Loslassen(f(150.4, 50))
	if z := s.D.Zuschnitt(); z != image.Rect(0, 0, 150, 100) {
		t.Fatalf("Zuschnitt nach Griff O %v", z)
	}
	if n := schritte(t, s); n != 1 { // nimmt alles zurück und stellt es wieder her
		t.Errorf("%d Rückgängig-Schritte, Soll 1", n)
	}
	zieh(s, fv(s, 0, 0), fv(s, 20, 10)) // Ecke links oben: zwei Kanten
	if z := s.D.Zuschnitt(); z != image.Rect(20, 10, 150, 100) {
		t.Fatalf("Zuschnitt nach Griff NW %v", z)
	}
	if a, b, _ := s.ZuschnittRahmen(); a != fv(s, 0, 0) || b != fv(s, 130, 90) {
		t.Errorf("Rahmen liegt nicht auf dem neuen Zuschnitt: %v %v", a, b)
	}
	// Zurückschieben: links über den Zuschnitt hinaus bis an den Rand der Vollansicht (klemmt), rechts ein Stück.
	zieh(s, fv(s, 0, 40), fv(s, -50, 40))
	zieh(s, fv(s, 150, 40), fv(s, 170, 40))
	if z := s.D.Zuschnitt(); z != image.Rect(0, 10, 170, 100) {
		t.Fatalf("Zuschnitt nach dem Zurückschieben %v, Soll (0,10)-(170,100)", z)
	}
	// Griff über die Gegenkante: klappt um.
	zieh(s, fv(s, 85, 0), fv(s, 85, 120)) // obere Kante unter die untere (klemmt am Rand bei 90)
	if z := s.D.Zuschnitt(); z != image.Rect(0, 10, 170, 100) {
		t.Errorf("leerer Rahmen nach dem Umklappen auf den Rand änderte %v", z)
	}
	zieh(s, fv(s, 85, 0), fv(s, 85, 120-50)) // obere Kante auf 70: Zuschnitt (0,80)-(170,100)
	if z := s.D.Zuschnitt(); z != image.Rect(0, 80, 170, 100) {
		t.Errorf("Zuschnitt nach Griff N %v", z)
	}
}

// Ein neuer Rahmen darf im abgedunkelten Teil beginnen und über den Zuschnitt hinausreichen; das ist der Weg bei einem
// Zuschnitt, der zu klein ist, um darin zu beginnen (Widerlegungsbedingung F-386).
func TestZuschnittNeuerRahmenImAbgedunkelten(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(100, 50, 108, 58)); err != nil { // 8×8, Kanten näher als zwei Toleranzen
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if s.Bildflaeche() != image.Rect(-100, -50, 100, 50) {
		t.Fatalf("Bildfläche %v", s.Bildflaeche())
	}
	if _, ok := s.zuschnittGriff(fv(s, 4, 4)); !ok {
		t.Error("Mitte eines kleinen Zuschnitts trifft keine Kante")
	}
	zieh(s, fv(s, -90, -40), fv(s, 50, 30))
	if z := s.D.Zuschnitt(); z != image.Rect(10, 10, 150, 80) {
		t.Fatalf("Zuschnitt %v, Soll (10,10)-(150,80)", z)
	}
	// Beginnt neben der Vollansicht: nichts (F-150).
	zieh(s, fv(s, -200, -200), fv(s, 0, 0))
	if z := s.D.Zuschnitt(); z != image.Rect(10, 10, 150, 80) {
		t.Errorf("Zug neben dem Bild änderte den Zuschnitt: %v", z)
	}
}

// Esc während eines Griff-Zugs ändert nichts; der Rahmen liegt danach wieder auf dem Zuschnitt.
func TestZuschnittGriffEsc(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.Druecken(f(0, 50), TastePrimaer)
	s.Ziehen(f(80, 50))
	s.Esc()
	if _, _, ok := s.Rahmen(); ok || s.Zieht() {
		t.Error("Zug läuft nach Esc")
	}
	s.Loslassen(f(80, 50))
	if s.Geaendert() || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Errorf("Esc: Zuschnitt %v", s.D.Zuschnitt())
	}
	if a, b, _ := s.ZuschnittRahmen(); a != f(0, 0) || b != f(200, 100) {
		t.Errorf("Rahmen nach Esc %v %v", a, b)
	}
}

// Mit Stauch-Streifen: Die Vollansicht behält die Streifen, der Zuschnitt liegt darin verschoben; ein Griff nach außen
// über einen Streifen vor dem Zuschnitt rechnet richtig ins Original zurück (erstes Pixel bis hinter das letzte).
func TestZuschnittGriffMitStreifen(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeAnsicht(image.Rect(60, 0, 200, 100), []document.Streifen{{Von: 20, Bis: 40}, {Von: 100, Bis: 110}}); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	// Vollansicht: 200 − 20 − 10 = 170 breit; der Zuschnitt beginnt bei Original 60 = Vollansicht 40.
	if s.Bildflaeche() != image.Rect(-40, 0, 130, 100) {
		t.Fatalf("Bildfläche %v", s.Bildflaeche())
	}
	zieh(s, fv(s, 0, 50), fv(s, -30, 50)) // linke Kante auf Vollansicht 10 = Original 10
	if z := s.D.Zuschnitt(); z != image.Rect(10, 0, 200, 100) {
		t.Fatalf("Zuschnitt %v, Soll (10,0)-(200,100)", z)
	}
	zieh(s, fv(s, 0, 50), fv(s, 15, 50)) // linke Kante auf Vollansicht 25 = hinter dem Streifen 20–40: Original 45
	if z := s.D.Zuschnitt(); z != image.Rect(45, 0, 200, 100) {
		t.Errorf("Zuschnitt %v, Soll (45,0)-(200,100)", z)
	}
	if len(s.D.Streifen()) != 2 {
		t.Errorf("Streifen verändert: %v", s.D.Streifen())
	}
}

// Ein Werkzeugwechsel zeigt die Vollansicht bzw. wieder die Ansicht; eingepasst wird nur, solange der Nutzer nicht
// selbst gezoomt hat — sonst bleibt die Ansicht, wo sie ist (die Ansichtskoordinaten ändern sich nicht).
func TestZuschnittModusWechsel(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(50, 25, 150, 75)); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	vor := s.BildAenderung()
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if s.BildAenderung() != vor+1 || s.Abb.ZuFenster(document.Pt(-50, -25)) != Einpassen(image.Pt(200, 100), image.Pt(248, 148), Rand).Versatz {
		t.Errorf("Vollansicht nicht eingepasst: %+v, BildAenderung %d", s.Abb, s.BildAenderung())
	}
	if g := s.Leinwandbild().Rect.Size(); g != image.Pt(200, 100) {
		t.Errorf("Leinwandbild %v, Soll die Vollansicht", g)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if g := s.Leinwandbild().Rect.Size(); g != image.Pt(100, 50) || s.BildAenderung() != vor+2 {
		t.Errorf("Leinwandbild %v nach dem Wechsel", g)
	}
	s.Scrollen(f(10, 10), -2) // Nutzer zoomt selbst
	abb := s.Abb
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	if s.Abb != abb {
		t.Errorf("gezoomte Ansicht beim Wechsel verändert: %+v → %+v", abb, s.Abb)
	}
	s.SetzeWerkzeug(WerkzeugStauchen) // Stauchen zeigt die Ansicht, nicht die Vollansicht
	if s.Bildflaeche() != image.Rect(0, 0, 100, 50) {
		t.Errorf("Bildfläche beim Stauchen %v", s.Bildflaeche())
	}
}
