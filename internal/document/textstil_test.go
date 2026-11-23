// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

func fastGleich(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestTextAusrichtung: Jede Zeile steht links, mittig oder rechts in der Breite der längsten (ADR-0021 Nachtrag 10
// Punkt 6); das Schild ist für alle drei gleich breit.
func TestTextAusrichtung(t *testing.T) {
	basis := Text{Ort: Pt(10, 20), Inhalt: "WWWW\nx", Schriftgrad: 20}
	var breiten []float64
	for _, a := range []Ausrichtung{AusrichtungLinks, AusrichtungMitte, AusrichtungRechts} {
		tx := basis
		tx.Ausrichtung = a
		z := tx.Zeilen()
		if len(z) != 2 || z[0].X != 10 || z[0].Vorschub <= z[1].Vorschub {
			t.Fatalf("%d: Zeilen %+v", a, z)
		}
		want := map[Ausrichtung]float64{AusrichtungLinks: 10, AusrichtungMitte: 10 + (z[0].Vorschub-z[1].Vorschub)/2,
			AusrichtungRechts: 10 + z[0].Vorschub - z[1].Vorschub}[a]
		if !fastGleich(z[1].X, want) || !fastGleich(z[1].Grundlinie, tx.Grundlinie(1)) {
			t.Errorf("%d: zweite Zeile bei %v, erwartet %v", a, z[1].X, want)
		}
		breiten = append(breiten, tx.Grenzen().Breite())
	}
	if !fastGleich(breiten[0], breiten[1]) || !fastGleich(breiten[0], breiten[2]) {
		t.Errorf("Schildbreiten %v", breiten)
	}
	// Die Tinte folgt der Ausrichtung: Eine Zeile mit Tinte neben einer längeren aus Leerzeichen wandert nach rechts.
	links := Text{Ort: Pt(10, 20), Inhalt: "x\n          ", Schriftgrad: 20}
	rechts := links
	rechts.Ausrichtung = AusrichtungRechts
	rl, _, _ := textTinte(links)
	rr, ok, _ := textTinte(rechts)
	z := rechts.Zeilen()
	if !ok || !fastGleich(rr.Min.X-rl.Min.X, z[1].Vorschub-z[0].Vorschub) {
		t.Errorf("Tinte rechtsbündig %v, linksbündig %v, Vorschübe %+v", rr, rl, z)
	}
}

// TestTextStriche: Unterstrich unter, Durchstrich über der Grundlinie, Dicke mindestens 1 px, über den Vorschub jeder
// Zeile; sie gehören zur Tinte und zum Schild (Punkt 7).
func TestTextStriche(t *testing.T) {
	tx := Text{Ort: Pt(0, 0), Inhalt: "ab\n\nlang", Schriftgrad: 40, Unterstrichen: true, Durchgestrichen: true}
	st := tx.Striche()
	z := tx.Zeilen()
	if len(st) != 4 { // leere Zeile ohne Vorschub: keine Striche
		t.Fatalf("%d Striche: %v", len(st), st)
	}
	d := StrichDicke * 40
	u, du := st[0], st[1]
	if !fastGleich(u.Min.Y, z[0].Grundlinie+UnterstrichAbstand*40) || !fastGleich(u.Hoehe(), d) ||
		!fastGleich(u.Min.X, z[0].X) || !fastGleich(u.Breite(), z[0].Vorschub) {
		t.Errorf("Unterstrich %v, Zeile %+v", u, z[0])
	}
	if !fastGleich((du.Min.Y+du.Max.Y)/2, z[0].Grundlinie-DurchstrichHoehe*40) || !fastGleich(du.Hoehe(), d) {
		t.Errorf("Durchstrich %v", du)
	}
	if klein := (Text{Inhalt: "a", Schriftgrad: 4, Unterstrichen: true}).Striche(); len(klein) != 1 || klein[0].Hoehe() != 1 {
		t.Errorf("Mindestdicke: %v", klein)
	}
	// Bei kleinem Grad reicht der Unterstrich unter die Zeile: Schild und Tinte schließen ihn ein.
	k := Text{Ort: Pt(5, 5), Inhalt: "ab", Schriftgrad: 4, Unterstrichen: true}
	s := k.Striche()[0]
	g := k.Grenzen()
	if g.Max.Y < s.Max.Y+TextInnenabstand*4-1e-9 {
		t.Errorf("Schild %v schließt den Unterstrich %v nicht ein", g, s)
	}
	ohne := k
	ohne.Unterstrichen = false
	if !(g.Max.Y > ohne.Grenzen().Max.Y) {
		t.Errorf("Schild mit Unterstrich %v nicht größer als ohne %v", g, ohne.Grenzen())
	}
	// Nur Leerraum mit Unterstrich hat Tinte, ohne keine.
	leer := Text{Ort: Pt(5, 5), Inhalt: "   ", Schriftgrad: 20}
	if _, ok, _ := textTinte(leer); ok {
		t.Error("Leerraum hat Tinte")
	}
	leer.Unterstrichen = true
	if _, ok, _ := textTinte(leer); !ok {
		t.Error("unterstrichener Leerraum hat keine Tinte")
	}
}

// TestTextStilPruefung: Ein ungültiger Familienname oder eine unbekannte Ausrichtung wird abgelehnt.
func TestTextStilPruefung(t *testing.T) {
	for _, tx := range []Text{
		{Ort: Pt(1, 1), Inhalt: "a", Schriftgrad: 10, Schrift: "a\nb"},
		{Ort: Pt(1, 1), Inhalt: "a", Schriftgrad: 10, Ausrichtung: AusrichtungRechts + 1},
	} {
		if tx.pruefe() == nil {
			t.Errorf("%+v gilt", tx)
		}
	}
	if (Text{Ort: Pt(1, 1), Inhalt: "a", Schriftgrad: 10, Schrift: "Noto Sans", Ausrichtung: AusrichtungMitte}).pruefe() != nil {
		t.Error("gültiger Stil abgelehnt")
	}
}
