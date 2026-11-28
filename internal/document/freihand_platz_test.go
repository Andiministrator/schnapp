// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"image"
	"slices"
	"testing"
)

// vollerStrich liefert einen Freihand-Strich mit n Punkten im Zickzack (Länge > 0).
func vollerStrich(n int) Freihand {
	pts := make([]Punkt, n)
	for i := range pts {
		pts[i] = Pt(float64(i%100), float64(i%2))
	}
	return NeuerFreihand(pts, Strich{Breite: 2})
}

// fastVoll liefert ein Dokument, in dem nur noch rest Freihand-Punkte Platz haben.
func fastVoll(t *testing.T, rest int) *Dokument {
	t.Helper()
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	fehlt := MaxFreihandGesamt - rest
	var os []Objekt
	for fehlt > 0 {
		n := min(fehlt, MaxFreihandPunkte)
		if fehlt-n == 1 { // kein Strich mit einem Punkt übrig lassen
			n--
		}
		os = append(os, vollerStrich(n))
		fehlt -= n
	}
	if _, err := d.HinzufuegenMehrere(os); err != nil {
		t.Fatal(err)
	}
	if p := d.FreihandPlatz(); p != rest {
		t.Fatalf("Platz %d, Soll %d", p, rest)
	}
	return d
}

// F-1051: Jeder Weg, auf dem ein Objekt ins Dokument kommt, prüft die Gesamtgrenze der Freihand-Punkte; genau bis an
// die Grenze geht es, ein Punkt mehr nicht.
func TestFreihandGesamtgrenzeBeimHinzufuegen(t *testing.T) {
	voll := func(err error) bool { return errors.Is(err, ErrFreihandVoll) }

	d := fastVoll(t, 10)
	if _, err := d.Hinzufuegen(vollerStrich(11)); !voll(err) {
		t.Errorf("Hinzufuegen 11 bei Platz 10: %v", err)
	}
	if _, err := d.FuegeEin(vollerStrich(11), 0); !voll(err) {
		t.Errorf("FuegeEin: %v", err)
	}
	if _, err := d.HinzufuegenMehrere([]Objekt{vollerStrich(6), vollerStrich(5)}); !voll(err) {
		t.Errorf("HinzufuegenMehrere: %v", err)
	}
	erster := d.Objekte()[0].Kennung()
	if _, err := d.FuegeEinMehrere([]Objekt{vollerStrich(6), vollerStrich(5)}, []int{erster, erster}); !voll(err) {
		t.Errorf("FuegeEinMehrere: %v", err)
	}
	// Andere Objekte gehen weiter.
	if _, err := d.Hinzufuegen(GeraderPfeil(Pt(1, 1), Pt(9, 9), Strich{Breite: 2})); err != nil {
		t.Errorf("Pfeil: %v", err)
	}
	// Genau an die Grenze geht es.
	if _, err := d.Hinzufuegen(vollerStrich(10)); err != nil {
		t.Errorf("Hinzufuegen 10 bei Platz 10: %v", err)
	}
	if d.FreihandPlatz() != 0 {
		t.Errorf("Platz %d", d.FreihandPlatz())
	}

	// Ein neuer Zug braucht Platz für einen ganzen Strich, weil er bis MaxFreihandPunkte wächst.
	d = fastVoll(t, MaxFreihandPunkte-1)
	if _, err := d.BeginneNeuenZug(vollerStrich(2)); !voll(err) {
		t.Errorf("BeginneNeuenZug ohne Platz für einen ganzen Strich: %v", err)
	}
	if _, err := d.BeginneNeuenZug(GeraderPfeil(Pt(1, 1), Pt(9, 9), Strich{Breite: 2})); err != nil {
		t.Errorf("Pfeil-Zug: %v", err)
	}
	d.BrecheZugAb()
	d = fastVoll(t, MaxFreihandPunkte)
	if _, err := d.BeginneNeuenZug(vollerStrich(2)); err != nil {
		t.Errorf("BeginneNeuenZug mit Platz: %v", err)
	}
	d.BrecheZugAb()

	// SetzeAngaben (Rettungskopie) und PruefeInhalt prüfen die Summe.
	os := slices.Repeat([]Objekt{vollerStrich(MaxFreihandPunkte)}, MaxFreihandGesamt/MaxFreihandPunkte)
	if err := d.SetzeAngaben(os, image.Rect(0, 0, 200, 100), nil, KanteAbriss); err != nil {
		t.Errorf("SetzeAngaben genau an der Grenze: %v", err)
	}
	if err := d.SetzeAngaben(append(os, vollerStrich(2)), image.Rect(0, 0, 200, 100), nil, KanteAbriss); !voll(err) {
		t.Errorf("SetzeAngaben über der Grenze: %v", err)
	}
	if _, err := PruefeInhalt(image.Pt(200, 100), image.Rect(0, 0, 200, 100), nil, append(os, vollerStrich(2))); !voll(err) {
		t.Errorf("PruefeInhalt über der Grenze: %v", err)
	}
}
