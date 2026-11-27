// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"slices"
	"testing"
)

// drei legt drei Kästen an (Kennungen 1, 2, 3, von unten nach oben).
func drei(t *testing.T) *Dokument {
	t.Helper()
	d := neu(t, 100, 100)
	for i := range 3 {
		x := float64(10 + 20*i)
		if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(x, 10), Pt(x+10, 20)), Strich: rot}); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func kennungen(d *Dokument) []int {
	var ids []int
	for _, o := range d.Objekte() {
		ids = append(ids, o.Kennung())
	}
	return ids
}

func xVon(t *testing.T, d *Dokument, id int) float64 {
	t.Helper()
	o, ok := d.Objekt(id)
	if !ok {
		t.Fatalf("Objekt %d fehlt", id)
	}
	return o.(Kasten).Bereich.Min.X
}

func kastenUm(t *testing.T, d *Dokument, id int, dx float64) Objekt {
	t.Helper()
	o, _ := d.Objekt(id)
	k := o.(Kasten)
	k.Bereich.Min.X += dx
	k.Bereich.Max.X += dx
	return k
}

// ADR-0039 Punkt 3: Ein Zug über mehrere Objekte landet als ein Schritt; Abbrechen stellt alle her; während des Zugs
// zählt für Speichern und Rettung der Stand davor.
func TestZugMehrere(t *testing.T) {
	d := drei(t)
	vor := d.Stand()
	if err := d.BeginneZugMehrere([]int{1, 3}); err != nil {
		t.Fatal(err)
	}
	if err := d.ZugSetze(kastenUm(t, d, 1, 5)); err == nil {
		t.Error("ZugSetze an einem Zug über mehrere angenommen")
	}
	if err := d.ZugSetzeMehrere([]Objekt{kastenUm(t, d, 2, 5)}); !errors.Is(err, ErrUnbekannt) {
		t.Errorf("fremdes Objekt im Zug: %v", err)
	}
	if err := d.ZugSetzeMehrere([]Objekt{kastenUm(t, d, 1, 5), kastenUm(t, d, 3, 5)}); err != nil {
		t.Fatal(err)
	}
	if x := xVon(t, d, 3); x != 55 {
		t.Errorf("während des Zugs x %v", x)
	}
	for _, o := range d.ObjekteOhneZug() {
		if k := o.(Kasten); k.Bereich.Min.X == 15 || k.Bereich.Min.X == 55 {
			t.Errorf("ObjekteOhneZug trägt den Zug: %v", k.Bereich)
		}
	}
	if err := d.BeendeZug(); err != nil {
		t.Fatal(err)
	}
	if d.Stand() == vor || xVon(t, d, 1) != 15 || xVon(t, d, 3) != 55 || xVon(t, d, 2) != 30 {
		t.Fatalf("nach dem Zug: Stand gleich %v, x %v %v %v", d.Stand() == vor, xVon(t, d, 1), xVon(t, d, 2), xVon(t, d, 3))
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if d.Stand() != vor || xVon(t, d, 1) != 10 || xVon(t, d, 3) != 50 {
		t.Errorf("Rückgängig nimmt nicht beide in einem Schritt zurück: x %v %v", xVon(t, d, 1), xVon(t, d, 3))
	}
	// Abbrechen
	_ = d.BeginneZugMehrere([]int{1, 2})
	_ = d.ZugSetzeMehrere([]Objekt{kastenUm(t, d, 1, 7), kastenUm(t, d, 2, 7)})
	if err := d.BrecheZugAb(); err != nil {
		t.Fatal(err)
	}
	if d.Stand() != vor || xVon(t, d, 1) != 10 || xVon(t, d, 2) != 30 {
		t.Errorf("Abbrechen: x %v %v", xVon(t, d, 1), xVon(t, d, 2))
	}
	// Unveränderter Zug hinterlässt nichts.
	_ = d.BeginneZugMehrere([]int{1, 2})
	_ = d.BeendeZug()
	if d.Stand() != vor {
		t.Error("unveränderter Zug legte einen Schritt an")
	}
	for _, ids := range [][]int{nil, {1, 9}, {2, 2}} {
		if err := d.BeginneZugMehrere(ids); err == nil {
			t.Errorf("BeginneZugMehrere(%v) angenommen", ids)
			_ = d.BrecheZugAb()
		}
	}
}

func TestErsetzeMehrere(t *testing.T) {
	d := drei(t)
	vor := d.Stand()
	if err := d.ErsetzeMehrere([]Objekt{kastenUm(t, d, 1, 3), kastenUm(t, d, 2, 0), kastenUm(t, d, 3, 3)}); err != nil {
		t.Fatal(err)
	}
	if xVon(t, d, 1) != 13 || xVon(t, d, 3) != 53 {
		t.Fatal("nicht ersetzt")
	}
	_ = d.Rueckgaengig()
	if d.Stand() != vor || xVon(t, d, 1) != 10 || xVon(t, d, 3) != 50 {
		t.Error("kein einzelner Schritt")
	}
	if err := d.ErsetzeMehrere([]Objekt{kastenUm(t, d, 2, 0)}); err != nil || d.Stand() != vor {
		t.Errorf("unverändert: %v, Stand neu %v", err, d.Stand() != vor)
	}
	k := kastenUm(t, d, 1, 1).(Kasten)
	k.ID = 9
	if err := d.ErsetzeMehrere([]Objekt{kastenUm(t, d, 2, 1), k}); !errors.Is(err, ErrUnbekannt) || xVon(t, d, 2) != 30 {
		t.Errorf("unbekannte Kennung: %v, x2 %v (alles oder nichts)", err, xVon(t, d, 2))
	}
	if err := d.ErsetzeMehrere([]Objekt{kastenUm(t, d, 2, 1), kastenUm(t, d, 2, 2)}); !errors.Is(err, ErrUnbekannt) {
		t.Errorf("doppelte Kennung: %v", err)
	}
}

func TestEntferneMehrere(t *testing.T) {
	d := drei(t)
	if err := d.EntferneMehrere([]int{3, 1}); err != nil {
		t.Fatal(err)
	}
	if got := kennungen(d); !slices.Equal(got, []int{2}) {
		t.Fatalf("übrig %v", got)
	}
	_ = d.Rueckgaengig()
	if got := kennungen(d); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Rückgängig stellt die Reihenfolge nicht her: %v", got)
	}
	if err := d.EntferneMehrere([]int{1, 7}); !errors.Is(err, ErrUnbekannt) || len(d.Objekte()) != 3 {
		t.Errorf("unbekannte Kennung: %v, %d Objekte", err, len(d.Objekte()))
	}
}

func TestFuegeEinMehrere(t *testing.T) {
	d := drei(t)
	a, _ := d.Objekt(1)
	c, _ := d.Objekt(3)
	ids, err := d.FuegeEinMehrere([]Objekt{a, c}, []int{1, 3})
	if err != nil {
		t.Fatal(err)
	}
	// Jede Kopie direkt über ihrem Original.
	if got := kennungen(d); !slices.Equal(got, []int{1, ids[0], 2, 3, ids[1]}) {
		t.Fatalf("Reihenfolge %v, neue %v", got, ids)
	}
	_ = d.Rueckgaengig()
	if got := kennungen(d); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Rückgängig: %v", got)
	}
	if _, err := d.FuegeEinMehrere([]Objekt{a}, []int{8}); !errors.Is(err, ErrUnbekannt) {
		t.Errorf("unbekanntes Original: %v", err)
	}
}
