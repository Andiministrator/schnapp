// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"math"
	"testing"
)

// ADR-0021 Nachträge 19 und 20: Schaft und Köpfe eines Pfeils je Spitzenart am Anfang und am Ende.

func pfeilFuer(anfang, ende Kopfart) Pfeil {
	return Pfeil{A: Pt(10, 100), B: Pt(210, 100), K: Pt(110, 20), Strich: Strich{Breite: 6}}.MitKoepfen(anfang, ende)
}

// Ohne Angabe trägt ein Pfeil das Dreieck bei B und keine Spitze bei A (Nullwerte, alte Dateien und Objekte).
func TestPfeilNullwerte(t *testing.T) {
	p := Pfeil{A: Pt(0, 0), B: Pt(100, 0)}
	if p.Ende != KopfDreieck || p.KopfAnfang() != KopfKeine {
		t.Errorf("Nullwerte: Ende %v, Anfang %v", p.Ende, p.KopfAnfang())
	}
	for _, k := range Kopfarten {
		if AnfangMit(k).Art() != k {
			t.Errorf("Anfang %v kommt als %v zurück", k, AnfangMit(k).Art())
		}
	}
}

func TestPfeilFormEnden(t *testing.T) {
	for _, fall := range []struct {
		name          string
		anfang, ende  Kopfart
		koepfe        int
		kAnfang, kEnd bool // ein Kopf bei A bzw. B
	}{
		{"ende", KopfKeine, KopfDreieck, 1, false, true}, {"anfang", KopfDreieck, KopfKeine, 1, true, false},
		{"beide", KopfDreieck, KopfDreieck, 2, true, true}, {"keine", KopfKeine, KopfKeine, 0, false, false},
	} {
		p := pfeilFuer(fall.anfang, fall.ende)
		f := p.Form()
		if len(f.Koepfe) != fall.koepfe {
			t.Fatalf("%s: %d Köpfe", fall.name, len(f.Koepfe))
		}
		erster, letzter := f.Schaft[0], f.Schaft[len(f.Schaft)-1]
		if !fall.kAnfang && erster != p.A || !fall.kEnd && letzter != p.B {
			t.Errorf("%s: Schaft ohne Kopf endet nicht am Ende: %v … %v", fall.name, erster, letzter)
		}
		for _, k := range f.Koepfe {
			ende, schaftEnde := p.B, letzter
			if k.AmAnfang {
				ende, schaftEnde = p.A, erster
			}
			if k.Spitze != ende || k.Schaftende != schaftEnde {
				t.Errorf("%s: Kopf bei %v mit Schaftende %v, Schaft endet bei %v", fall.name, k.Spitze, k.Schaftende, schaftEnde)
			}
			// Der Schaft endet KopfRueckstand vor dem Ende, auf der Kurve.
			if d := k.Schaftende.Minus(ende).Laenge(); math.Abs(d-KopfRueckstand(KopfDreieck, 6)) > 1e-6 {
				t.Errorf("%s: Rückstand %v, Soll %v", fall.name, d, KopfRueckstand(KopfDreieck, 6))
			}
			// Die Richtung zeigt vom Schaft nach außen.
			if r := k.Spitze.Minus(k.Schaftende); r.X*k.Richtung.X+r.Y*k.Richtung.Y <= 0 {
				t.Errorf("%s: Richtung %v zeigt nach innen", fall.name, k.Richtung)
			}
		}
	}
}

// Der Umriss beginnt und endet an den Schaftpunkten ∓ Normale·hw, die Normale ist die der äußersten Schaftstrecke.
func TestPfeilKopfUmrissSchliesstAmSchaft(t *testing.T) {
	for _, art := range Kopfarten {
		if art == KopfKeine {
			continue
		}
		p := pfeilFuer(art, art)
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
		q.Ende = art
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
	// Nachtrag 20: schlank 31 lang, 22 breit; Raute 32 lang, 24 breit, Spitze am Ende; Quadrat Seite 2·10,3 um das Ende.
	sl := kopf(KopfSchlank)
	if sl.Kontur[1] != Pt(200, 0) || math.Abs(sl.Kontur[0].X-169) > 1e-9 || math.Abs(math.Abs(sl.Kontur[0].Y)-11) > 1e-9 {
		t.Errorf("schlank %v", sl.Kontur)
	}
	ra := kopf(KopfRaute)
	if ra.Kontur[1] != Pt(200, 0) || math.Abs(ra.Kontur[0].X-184) > 1e-9 || math.Abs(math.Abs(ra.Kontur[0].Y)-12) > 1e-9 {
		t.Errorf("Raute %v", ra.Kontur)
	}
	qd := kopf(KopfQuadrat)
	if math.Abs(qd.Kontur[1].X-210.3) > 1e-9 || math.Abs(qd.Kontur[0].X-189.7) > 1e-9 || math.Abs(math.Abs(qd.Kontur[0].Y)-10.3) > 1e-9 {
		t.Errorf("Quadrat %v", qd.Kontur)
	}
	ha := kopf(KopfHalb)
	if len(ha.Kontur) != 2 || ha.Kontur[0] != Pt(200, 0) || math.Abs(ha.Kontur[1].X-177) > 1e-9 {
		t.Errorf("halbe Spitze %v", ha.Kontur)
	}
}

// Kerbe und Raute: Die Schaftpunkte liegen auf den hinteren Kanten der Spitze (sonst bliebe zwischen Schaft und Spitze
// eine Lücke oder ein Zipfel), die Spitze der Kerbe liegt vor dem Schaftende.
func TestPfeilKerbeUndRauteSchliessenAnDenSchaft(t *testing.T) {
	p := GeraderPfeil(Pt(0, 0), Pt(200, 0), Strich{Breite: 6})
	for _, art := range []Kopfart{KopfKerbe, KopfRaute} {
		q := p
		q.Ende = art
		k := q.Form().Koepfe[0]
		u := k.Umriss(3)
		l, h := dreieckMass(art, 6)
		ecke, innen := Pt(200-l, -h), Pt(200-l+kerbeTiefe*l, 0) // hintere Ecke rechts und Spitze der Kerbe
		if art == KopfRaute {
			ecke, innen = Pt(200-l/2, -h), Pt(200-l, 0)
		}
		if u[1].Y > 0 { // Kontur beginnt je nach Normale oben oder unten
			ecke.Y = -ecke.Y
		}
		// u[0] (Schaftpunkt) liegt auf der Strecke ecke–innen.
		d := innen.Minus(ecke)
		abstand := math.Abs(d.X*(u[0].Y-ecke.Y)-d.Y*(u[0].X-ecke.X)) / d.Laenge()
		if abstand > 1e-9 {
			t.Errorf("%s: Schaftpunkt %v liegt %v neben der Kante %v–%v", art.Name(), u[0], abstand, ecke, innen)
		}
		if art == KopfKerbe && innen.X <= k.Schaftende.X {
			t.Errorf("Kerbe: Spitze der Kerbe %v hinter dem Schaftende %v", innen, k.Schaftende)
		}
	}
}

// Kürzer als die Köpfe: kein Schaft, die Köpfe bleiben; Grenzen der Prüfung.
func TestPfeilKurzUndPruefung(t *testing.T) {
	p := GeraderPfeil(Pt(0, 0), Pt(10, 0), Strich{Breite: 6}).MitKoepfen(KopfDreieck, KopfDreieck)
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
	p.Ende = KopfKeine + 1
	if p.pruefe() == nil {
		t.Error("unbekannte Spitzenart am Ende angenommen")
	}
	p.Ende, p.Anfang = KopfDreieck, Anfangskopf(KopfKeine)+1
	if p.pruefe() == nil {
		t.Error("unbekannte Spitzenart am Anfang angenommen")
	}
	for _, n := range []string{"triangle", "narrow", "notched", "open", "half", "dot", "diamond", "square", "bar", "none"} {
		if k, ok := KopfartAus(n); !ok || k.Name() != n {
			t.Errorf("Name %q", n)
		}
	}
	if len(KopfNamen()) != len(Kopfarten) {
		t.Errorf("%d Namen für %d Arten", len(KopfNamen()), len(Kopfarten))
	}
	if k, ok := KopfartAus("arrow"); ok || k != KopfDreieck {
		t.Error("unbekannter Name")
	}
}

// Teilstueck: Die Teilkurve deckt sich mit der Kurve zwischen ta und tb.
func TestPfeilTeilstueck(t *testing.T) {
	p := pfeilFuer(KopfKeine, KopfDreieck)
	a, k, b := p.Teilstueck(0.2, 0.7)
	teil := Pfeil{A: a, K: k, B: b}
	for _, s := range []float64{0, 0.3, 0.5, 1} {
		if d := teil.Punkt(s).Minus(p.Punkt(0.2 + 0.5*s)).Laenge(); d > 1e-9 {
			t.Errorf("s=%v: Abstand %v", s, d)
		}
	}
}

// Kritik K-4: Eine Haarnadel, deren Kurve überall näher am Ende bleibt als der Kopf lang ist, behält ihren Schaft
// (Schaftende nach Bogenlänge); vor #315 war die Kurve gezeichnet.
func TestPfeilHaarnadelBehaeltSchaft(t *testing.T) {
	for _, breite := range []float64{6, 12} {
		p := Pfeil{A: Pt(100, 100), K: Pt(110, 170), B: Pt(120, 100), Strich: Strich{Breite: breite}}
		if f := p.Form(); len(f.Schaft) < 2 || len(f.Koepfe) != 1 {
			t.Errorf("Strich %v: %d Schaftpunkte, %d Köpfe", breite, len(f.Schaft), len(f.Koepfe))
		}
	}
}
