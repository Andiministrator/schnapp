// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 26 Punkt 7: Schalten, Setzen und Ziehen am Werkzeug und am gewählten Objekt; neue Objekte bekommen
// den Schatten des Werkzeugs, wenn er an ist; das Feld Schatten färbt ihn über den Farbwähler.
func TestSchattenWerkzeugUndAuswahl(t *testing.T) {
	s, k, _ := sitzung(t)
	if s.Schatten().An {
		t.Fatal("Schatten an ohne Wahl")
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	vor := s.D.Stand()
	if s.SchalteSchatten() || s.D.Stand() != vor {
		t.Error("ohne Auswahl ein Rückgängig-Schritt")
	}
	if sch := s.Schatten(); !sch.An || sch.Winkel != 135 || sch.Laenge == 0 || sch.Farbe.A != 102 {
		t.Fatalf("Voreinstellung %+v", sch)
	}
	zieh(s, f(20, 20), f(120, 80))
	if o := letztes(t, s); !o.Ebenenwerte().Schatten.An {
		t.Error("neuer Kasten ohne Schatten des Werkzeugs")
	}
	// Feld Schatten: der Farbwähler färbt den Schatten.
	if !s.SetzeFeld(FeldSchatten) || s.AktivesFeld() != FeldSchatten {
		t.Fatal("Feld Schatten nicht aktiv")
	}
	blau := color.NRGBA{B: 200, A: 128}
	s.SetzeFarbe(blau)
	if o := letztes(t, s); o.Ebenenwerte().Schatten.Farbe != blau {
		t.Errorf("Schattenfarbe %v", o.Ebenenwerte().Schatten.Farbe)
	}
	if o := letztes(t, s).(document.Kasten); o.Strich.Farbe == blau {
		t.Error("Feld Schatten färbte den Strich")
	}
	// Am gewählten Kasten ohne Schatten: Einschalten ist ein Schritt, Rückgängig nimmt ihn zurück.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	s.Auswahl = k
	if s.SetzeFeld(FeldSchatten) {
		t.Error("Feld Schatten ohne Schatten am Objekt")
	}
	vor = s.D.Stand()
	if !s.SchalteSchatten() || s.D.Stand() == vor {
		t.Fatal("Einschalten änderte den Kasten nicht")
	}
	o, _ := s.D.Objekt(k)
	if !o.Ebenenwerte().Schatten.An {
		t.Fatal("Kasten ohne Schatten")
	}
	// Ziehen an einem Regler: ein Schritt.
	vor = s.D.Stand()
	if !s.BeginneSchattenZug() {
		t.Fatal("Zug nicht begonnen")
	}
	for _, l := range []float64{8, 12, 20} {
		sch := s.Schatten()
		sch.Laenge = l
		s.SchattenZugSetze(sch)
	}
	s.BeendeBreiteZug()
	if o, _ := s.D.Objekt(k); o.Ebenenwerte().Schatten.Laenge != 20 {
		t.Errorf("Länge nach dem Zug %v", o.Ebenenwerte().Schatten.Laenge)
	}
	s.Rueckgaengig()
	if o, _ := s.D.Objekt(k); o.Ebenenwerte().Schatten.Laenge == 20 || !o.Ebenenwerte().Schatten.An {
		t.Errorf("Rückgängig nahm den Zug nicht als einen Schritt: %+v", o.Ebenenwerte().Schatten)
	}
	// Grenzen und Winkel.
	sch := s.Schatten()
	sch.Winkel, sch.Laenge, sch.Weich = -45, 5000, -3
	s.SetzeSchatten(sch)
	if got := s.Schatten(); got.Winkel != 315 || got.Laenge != document.MaxSchattenLaenge || got.Weich != 0 {
		t.Errorf("begrenzt %+v", got)
	}
	// Ausschalten verlässt das Feld Schatten.
	s.SetzeFeld(FeldSchatten)
	s.SchalteSchatten()
	if s.Schatten().An || s.AktivesFeld() == FeldSchatten {
		t.Error("Ausschalten ließ Schatten oder Feld")
	}
	s.SchalteSchatten() // wieder an: das Feld bleibt der Vordergrund
	if !s.Schatten().An || s.AktivesFeld() != FeldVordergrund {
		t.Errorf("wieder an: Feld %v", s.AktivesFeld())
	}
}
