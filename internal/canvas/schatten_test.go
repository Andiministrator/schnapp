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
	if o, _ := s.D.Objekt(k); o.Ebenenwerte().Schatten != (document.Schatten{}) {
		t.Errorf("ausgeschaltet bleibt am Objekt %+v", o.Ebenenwerte().Schatten)
	}
	s.SchalteSchatten() // wieder an: das Feld bleibt der Vordergrund
	if !s.Schatten().An || s.AktivesFeld() != FeldVordergrund {
		t.Errorf("wieder an: Feld %v", s.AktivesFeld())
	}
}

// F-847: Jeder Anlegeweg gibt einem neuen Objekt den Schatten des Werkzeugs — Pfeil, Lupe, Nummer, Text, Sprechblase
// per Klick und per Ziehen; in der Texteingabe gehört eine Änderung zur Eingabe; ein abgebrochener Reglerzug stellt das
// Objekt wieder her.
func TestSchattenAlleAnlegewege(t *testing.T) {
	s := blasenSitzung(t)
	s.SetzeWerkzeug(WerkzeugPfeil)
	s.SchalteSchatten()
	hat := func(name string) {
		t.Helper()
		if o := letztes(t, s); !o.Ebenenwerte().Schatten.An {
			t.Errorf("%s (%T) ohne Schatten", name, o)
		}
	}
	zieh(s, f(20, 20), f(120, 60))
	hat("Pfeil")
	s.SetzeWerkzeug(WerkzeugLupe)
	zieh(s, f(20, 120), f(60, 150))
	hat("Lupe")
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Druecken(f(300, 40), TastePrimaer)
	s.Loslassen(f(300, 40))
	hat("Nummer")
	s.SetzeWerkzeug(WerkzeugText)
	s.Druecken(f(300, 200), TastePrimaer)
	s.Loslassen(f(300, 200))
	s.Tippe("Hi")
	// In der Eingabe: Schatten ändern ist Teil der Eingabe, kein eigener Schritt.
	sch := s.Schatten()
	sch.Laenge = 3
	vor := s.D.Stand()
	s.SetzeSchatten(sch)
	if s.D.Stand() != vor {
		t.Error("Schatten in der Texteingabe ist ein eigener Schritt")
	}
	s.Enter(false)
	hat("Text")
	if o := letztes(t, s); o.Ebenenwerte().Schatten.Laenge != 3 {
		t.Errorf("Text: Länge %v", o.Ebenenwerte().Schatten.Laenge)
	}
	s.SetzeWerkzeug(WerkzeugSprechblase)
	s.Druecken(f(200, 100), TastePrimaer)
	s.Loslassen(f(200, 100))
	bestaetigeBlase(t, s, "Klick")
	hat("Sprechblase per Klick")
	s.SetzeWerkzeug(WerkzeugSprechblase)
	zieh(s, f(100, 250), f(200, 200))
	bestaetigeBlase(t, s, "Zug")
	hat("Sprechblase per Ziehen")
	// Reglerzug abbrechen: das Objekt wie vorher.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	o := letztes(t, s)
	s.Auswahl = o.Kennung()
	if !s.BeginneSchattenZug() {
		t.Fatal("Zug nicht begonnen")
	}
	sch = s.Schatten()
	sch.Laenge = 30
	s.SchattenZugSetze(sch)
	s.BrecheBreiteZugAb()
	if n, _ := s.D.Objekt(o.Kennung()); n.Ebenenwerte().Schatten != o.Ebenenwerte().Schatten {
		t.Errorf("nach Abbruch %+v, vorher %+v", n.Ebenenwerte().Schatten, o.Ebenenwerte().Schatten)
	}
}

// V-4 der Sitzungs-Kritik 26.11. (5): Ein Reglerzug an einem Objekt ohne Schatten schreibt den Nullwert, nicht einen
// ausgeschalteten Schatten mit Werten; das Werkzeug merkt die Werte.
func TestSchattenZugOhneSchatten(t *testing.T) {
	s, k, _ := sitzung(t)
	s.Auswahl = k
	if !s.BeginneSchattenZug() {
		t.Fatal("Zug nicht begonnen")
	}
	sch := s.Schatten()
	sch.Laenge = 12
	s.SchattenZugSetze(sch)
	s.BeendeBreiteZug()
	if o, _ := s.D.Objekt(k); o.Ebenenwerte().Schatten != (document.Schatten{}) {
		t.Errorf("am Objekt %+v", o.Ebenenwerte().Schatten)
	}
	if s.Stil.Schatten.Laenge != 12 {
		t.Errorf("Werkzeug merkt %v", s.Stil.Schatten.Laenge)
	}
}

// ADR-0021 Nachtrag 26 Punkt 7: − / + an „Direction“ gehen in 15°-Schritten über 0/360 hinweg, ein Winkel zwischen den
// Stufen springt erst auf das Raster; die Regler bilden 0 bis 40 bzw. 20 px (mal Faktor) ab, Größeres steht am Ende.
func TestSchattenRichtungUndRegler(t *testing.T) {
	for _, f := range []struct {
		w    float64
		dir  int
		soll float64
	}{
		{135, 1, 150}, {135, -1, 120}, {345, 1, 0}, {0, -1, 345}, {137, 1, 150}, {137, -1, 135}, {359.9, 1, 0},
		{0.5, -1, 0}, {-15, 1, 0},
	} {
		if g := NaechsteSchattenRichtung(f.w, f.dir); g != f.soll {
			t.Errorf("%v %+d: %v, Soll %v", f.w, f.dir, g, f.soll)
		}
	}
	for _, f := range []struct{ w, bis, soll float64 }{{0, 40, 0}, {10, 40, 0.25}, {40, 40, 1}, {90, 40, 1}, {-3, 40, 0}, {5, 0, 0}} {
		if g := SchattenReglerLage(f.w, f.bis); g != f.soll {
			t.Errorf("Lage %v/%v: %v, Soll %v", f.w, f.bis, g, f.soll)
		}
	}
	for _, f := range []struct{ lage, bis, soll float64 }{{0.5, 40, 20}, {-1, 40, 0}, {2, 20, 20}, {0.26, 20, 5}} {
		if g := SchattenAusRegler(f.lage, f.bis); g != f.soll {
			t.Errorf("Wert %v/%v: %v, Soll %v", f.lage, f.bis, g, f.soll)
		}
	}

	s, _, _ := sitzung(t)
	if l, w := s.SchattenReglerMax(); l != SchattenLaengeRegler || w != SchattenWeichRegler {
		t.Errorf("Regler bei Faktor 1: %v %v", l, w)
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if w := s.Schatten().Winkel; s.SchattenRichtungStufe(1) || s.Schatten().Winkel != w {
		t.Error("Richtung ändert sich bei ausgeschaltetem Schatten")
	}
	s.SchalteSchatten()
	zieh(s, f(20, 20), f(120, 80))
	vor := s.D.Stand()
	if !s.SchattenRichtungStufe(1) || letztes(t, s).Ebenenwerte().Schatten.Winkel != 150 || s.D.Stand() == vor {
		t.Errorf("Richtung am gewählten Objekt: %+v", letztes(t, s).Ebenenwerte().Schatten)
	}
	if s.Stil.Schatten.Winkel != 150 {
		t.Errorf("Werkzeug behält die Richtung nicht: %v", s.Stil.Schatten.Winkel)
	}
}

// Sichtprobe #363: Ein Werkzeugwechsel gibt das Feld Schatten auf, die Farbwahl wirkt wieder auf den Vordergrund;
// dasselbe Werkzeug noch einmal wählen lässt es stehen.
func TestSchattenFeldEndetBeimWerkzeugwechsel(t *testing.T) {
	s, _, _ := sitzung(t)
	s.SetzeWerkzeug(WerkzeugKasten)
	s.SchalteSchatten()
	if !s.SetzeFeld(FeldSchatten) {
		t.Fatal("Feld Schatten nicht wählbar")
	}
	s.SetzeWerkzeug(WerkzeugKasten)
	if s.AktivesFeld() != FeldSchatten {
		t.Error("dasselbe Werkzeug beendet das Feld Schatten")
	}
	s.SetzeWerkzeug(WerkzeugText)
	if s.AktivesFeld() != FeldVordergrund {
		t.Errorf("nach dem Wechsel zum Text: Feld %v", s.AktivesFeld())
	}
}
