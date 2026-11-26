// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// ADR-0021 Nachtrag 19: Schaft und Köpfe eines Pfeils je Spitzenart und Enden.

func pfeilFuer(k Kopfart, e PfeilEnden) Pfeil {
	return Pfeil{A: Pt(10, 100), B: Pt(210, 100), K: Pt(110, 20), Strich: Strich{Breite: 6}, Kopf: k, Enden: e}
}

func TestPfeilFormEnden(t *testing.T) {
	for _, fall := range []struct {
		e      PfeilEnden
		koepfe int
		anfang bool // ein Kopf bei A
		ende   bool // ein Kopf bei B
	}{
		{EndenEnde, 1, false, true}, {EndenAnfang, 1, true, false}, {EndenBeide, 2, true, true}, {EndenKeine, 0, false, false},
	} {
		p := pfeilFuer(KopfDreieck, fall.e)
		f := p.Form()
		if len(f.Koepfe) != fall.koepfe {
			t.Fatalf("%s: %d Köpfe", fall.e.Name(), len(f.Koepfe))
		}
		erster, letzter := f.Schaft[0], f.Schaft[len(f.Schaft)-1]
		if !fall.anfang && erster != p.A || !fall.ende && letzter != p.B {
			t.Errorf("%s: Schaft ohne Kopf endet nicht am Ende: %v … %v", fall.e.Name(), erster, letzter)
		}
		for _, k := range f.Koepfe {
			ende, schaftEnde := p.B, letzter
			if k.AmAnfang {
				ende, schaftEnde = p.A, erster
			}
			if k.Spitze != ende || k.Schaftende != schaftEnde {
				t.Errorf("%s: Kopf bei %v mit Schaftende %v, Schaft endet bei %v", fall.e.Name(), k.Spitze, k.Schaftende, schaftEnde)
			}
			// Der Schaft endet KopfRueckstand vor dem Ende, auf der Kurve.
			if d := k.Schaftende.Minus(ende).Laenge(); math.Abs(d-KopfRueckstand(KopfDreieck, 6)) > 1e-6 {
				t.Errorf("%s: Rückstand %v, Soll %v", fall.e.Name(), d, KopfRueckstand(KopfDreieck, 6))
			}
			// Die Richtung zeigt vom Schaft nach außen.
			if r := k.Spitze.Minus(k.Schaftende); r.X*k.Richtung.X+r.Y*k.Richtung.Y <= 0 {
				t.Errorf("%s: Richtung %v zeigt nach innen", fall.e.Name(), k.Richtung)
			}
		}
	}
}

// Der Umriss beginnt und endet an den Schaftpunkten ∓ Normale·hw, die Normale ist die der äußersten Schaftstrecke.
func TestPfeilKopfUmrissSchliesstAmSchaft(t *testing.T) {
	for _, art := range Kopfarten {
		p := pfeilFuer(art, EndenBeide)
		f := p.Form()
		hw := p.Strich.Breite / 2
		for _, k := range f.Koepfe {
			u := k.Umriss(hw)
			a, b := f.Schaft[len(f.Schaft)-2], f.Schaft[len(f.Schaft)-1]
			if k.AmAnfang {
				a, b = f.Schaft[1], f.Schaft[0]
			}
			r := b.Minus(a).Mal(1 / b.Minus(a).Laenge())
			n := Pt(-r.Y, r.X)
			if u[0].Minus(b.Minus(n.Mal(hw))).Laenge() > 1e-9 || u[len(u)-1].Minus(b.Plus(n.Mal(hw))).Laenge() > 1e-9 {
				t.Errorf("%s am Anfang %v: Umriss %v … %v, Schaftende %v ± %v", art.Name(), k.AmAnfang, u[0], u[len(u)-1], b, n.Mal(hw))
			}
		}
	}
}

// Maße je Art (w = 6): Dreieck 23 lang und 33 breit, Winkel mit Schenkeln w dick, Punkt Radius 12, Balken 36 lang.
func TestPfeilKopfMasse(t *testing.T) {
	p := GeraderPfeil(Pt(0, 0), Pt(200, 0), Strich{Breite: 6})
	kopf := func(art Kopfart) Kopfform {
		q := p
		q.Kopf = art
		return q.Form().Koepfe[0]
	}
	d := kopf(KopfDreieck)
	if d.Kontur[1] != Pt(200, 0) || d.Kontur[0] != Pt(177, 16.5) && d.Kontur[0] != Pt(177, -16.5) {
		t.Errorf("Dreieck %v", d.Kontur)
	}
	w := kopf(KopfWinkel)
	// Innere Kante (Kontur[0] zum Schaftpunkt) parallel zur äußeren (Kontur[1] zur Spitze), Abstand 6.
	aussenA, aussenB := w.Kontur[1], w.Kontur[2]
	innen := w.Kontur[0]
	ab := aussenB.Minus(aussenA)
	abstand := math.Abs(ab.X*(innen.Y-aussenA.Y)-ab.Y*(innen.X-aussenA.X)) / ab.Laenge()
	if math.Abs(abstand-6) > 1e-9 {
		t.Errorf("Winkel: Schenkel %v dick, Soll 6", abstand)
	}
	pu := kopf(KopfPunkt)
	for _, q := range pu.Kontur {
		if math.Abs(q.Minus(Pt(200, 0)).Laenge()-12) > 1e-9 {
			t.Fatalf("Punkt: %v liegt nicht auf dem Kreis vom Radius 12", q)
		}
	}
	if pu.Kontur[len(pu.Kontur)/2].X < 211 { // die Mitte des Bogens liegt vorn
		t.Errorf("Punkt: Bogen läuft nicht über vorn: %v", pu.Kontur[len(pu.Kontur)/2])
	}
	qu := kopf(KopfQuer)
	if l := qu.Kontur[1].Minus(qu.Kontur[2]).Laenge(); math.Abs(l-36) > 1e-9 || qu.Kontur[1].X != 203 {
		t.Errorf("Balken: %v", qu.Kontur)
	}
}

// Kürzer als die Köpfe: kein Schaft, die Köpfe bleiben; Grenzen der Prüfung.
func TestPfeilKurzUndPruefung(t *testing.T) {
	p := GeraderPfeil(Pt(0, 0), Pt(10, 0), Strich{Breite: 6})
	p.Enden = EndenBeide
	f := p.Form()
	if len(f.Schaft) != 0 || len(f.Koepfe) != 2 {
		t.Fatalf("kurzer Pfeil: Schaft %d Punkte, %d Köpfe", len(f.Schaft), len(f.Koepfe))
	}
	// Jeder Kopf hat seine volle Form: das gedachte Schaftende liegt 23 vor der Spitze auf der Sehne (sonst läge es am
	// anderen Ende, 10 entfernt, und der Umriss hätte eine Kerbe).
	for _, k := range f.Koepfe {
		if d := k.Schaftende.Minus(k.Spitze).Laenge(); math.Abs(d-23) > 1e-9 {
			t.Errorf("Kopf bei %v: Schaftende %v, %v entfernt, Soll 23", k.Spitze, k.Schaftende, d)
		}
	}
	p.Kopf = KopfQuer + 1
	if p.pruefe() == nil {
		t.Error("unbekannte Spitzenart angenommen")
	}
	p.Kopf, p.Enden = KopfDreieck, EndenKeine+1
	if p.pruefe() == nil {
		t.Error("unbekannte Enden angenommen")
	}
	for _, n := range []string{"triangle", "open", "dot", "bar"} {
		if k, ok := KopfartAus(n); !ok || k.Name() != n {
			t.Errorf("Name %q", n)
		}
	}
	if k, ok := KopfartAus("arrow"); ok || k != KopfDreieck {
		t.Error("unbekannter Name")
	}
	for _, n := range []string{"end", "start", "both", "none"} {
		if e, ok := PfeilEndenAus(n); !ok || e.Name() != n {
			t.Errorf("Name %q", n)
		}
	}
}

// Teilstueck: Die Teilkurve deckt sich mit der Kurve zwischen ta und tb.
func TestPfeilTeilstueck(t *testing.T) {
	p := pfeilFuer(KopfDreieck, EndenEnde)
	a, k, b := p.Teilstueck(0.2, 0.7)
	teil := Pfeil{A: a, K: k, B: b}
	for _, s := range []float64{0, 0.3, 0.5, 1} {
		if d := teil.Punkt(s).Minus(p.Punkt(0.2 + 0.5*s)).Laenge(); d > 1e-9 {
			t.Errorf("s=%v: Abstand %v", s, d)
		}
	}
}
