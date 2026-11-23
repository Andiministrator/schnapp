// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"strings"
	"testing"
)

func ebenenDok(t *testing.T, n int) (*Dokument, []int) {
	t.Helper()
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 50, 40)))
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for i := 0; i < n; i++ {
		id, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(float64(i), 1), Pt(float64(i)+5, 6)), Strich: Strich{Breite: 1}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return d, ids
}

func reihenfolge(d *Dokument) []int {
	var r []int
	for _, o := range d.Objekte() {
		r = append(r, o.Kennung())
	}
	return r
}

func gleich(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ADR-0032 Punkt 3: Umordnen ist ein Rückgängig-Schritt, Grenzen werden geklemmt, ohne Wirkung kein Schritt.
func TestUmordnen(t *testing.T) {
	d, ids := ebenenDok(t, 3)
	vor := d.Stand()
	if err := d.Umordnen(ids[0], 2); err != nil {
		t.Fatal(err)
	}
	if r := reihenfolge(d); !gleich(r, []int{ids[1], ids[2], ids[0]}) {
		t.Fatalf("nach Umordnen nach oben: %v", r)
	}
	if err := d.Rueckgaengig(); err != nil || !gleich(reihenfolge(d), ids) || d.Stand() != vor {
		t.Fatalf("Rückgängig: %v, Reihenfolge %v", err, reihenfolge(d))
	}
	if err := d.Wiederholen(); err != nil || !gleich(reihenfolge(d), []int{ids[1], ids[2], ids[0]}) {
		t.Fatalf("Wiederholen: %v, %v", err, reihenfolge(d))
	}
	// geklemmt
	if err := d.Umordnen(ids[0], -5); err != nil || !gleich(reihenfolge(d), ids) {
		t.Fatalf("nach unten geklemmt: %v, %v", err, reihenfolge(d))
	}
	s := d.Stand()
	if err := d.Umordnen(ids[0], 0); err != nil || d.Stand() != s {
		t.Error("Umordnen an dieselbe Stelle legt einen Schritt an")
	}
	if err := d.Umordnen(99, 0); err != ErrUnbekannt {
		t.Errorf("unbekannte Kennung: %v", err)
	}
	if _, err := d.BeginneNeuenZug(Kasten{Strich: Strich{Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := d.Umordnen(ids[0], 2); err != ErrZugLaeuft {
		t.Errorf("während eines Zugs: %v", err)
	}
}

// FuegeEin legt an der Stelle ein (Doppeln: direkt über dem Original), mit neuer Kennung, als ein Schritt.
func TestFuegeEin(t *testing.T) {
	d, ids := ebenenDok(t, 3)
	o, _ := d.Objekt(ids[0])
	id, err := d.FuegeEin(o, d.Stelle(ids[0])+1)
	if err != nil {
		t.Fatal(err)
	}
	if id == ids[0] || !gleich(reihenfolge(d), []int{ids[0], id, ids[1], ids[2]}) {
		t.Fatalf("Kennung %d, Reihenfolge %v", id, reihenfolge(d))
	}
	if d.Rueckgaengig() != nil || !gleich(reihenfolge(d), ids) {
		t.Fatalf("Rückgängig: %v", reihenfolge(d))
	}
	if _, err := d.FuegeEin(Kasten{Strich: Strich{Breite: -1}}, 0); err == nil {
		t.Error("ungültiges Objekt eingefügt")
	}
	if id, err := d.FuegeEin(o, 99); err != nil || d.Stelle(id) != 3 {
		t.Errorf("Stelle über dem Ende: %v, %d", err, d.Stelle(id))
	}
}

// Angaben der Ebene: über Ersetzen als ein Schritt, geprüft (Name, Deckkraft), im Abzug erhalten.
func TestEbenenAngaben(t *testing.T) {
	d, ids := ebenenDok(t, 1)
	o, _ := d.Objekt(ids[0])
	if e := o.Ebenenwerte(); e != (Ebene{}) || e.Deckkraft() != 100 || !e.Bearbeitbar() {
		t.Fatalf("Nullwert: %+v, Deckkraft %d", e, e.Deckkraft())
	}
	neu := Ebene{Name: "Rahmen", Ausgeblendet: true, Gesperrt: true}.MitDeckkraft(40)
	if err := d.Ersetzen(MitEbene(o, neu)); err != nil {
		t.Fatal(err)
	}
	o2, _ := d.Objekt(ids[0])
	if o2.Ebenenwerte() != neu || neu.Deckkraft() != 40 || neu.Bearbeitbar() {
		t.Fatalf("gesetzt: %+v", o2.Ebenenwerte())
	}
	a, err := d.Abzug()
	if err != nil {
		t.Fatal(err)
	}
	if oa, _ := a.Objekt(ids[0]); oa.Ebenenwerte() != neu {
		t.Error("Abzug verliert die Angaben der Ebene")
	}
	if d.Rueckgaengig() != nil {
		t.Fatal("Rückgängig")
	}
	if o3, _ := d.Objekt(ids[0]); o3.Ebenenwerte() != (Ebene{}) {
		t.Error("Rückgängig stellt die Angaben nicht her")
	}
	for _, e := range []Ebene{{Name: strings.Repeat("x", MaxEbenenName+1)}, {Name: "a\nb"}, {Name: "\xff"}, {Transparenz: 101}} {
		if err := d.Ersetzen(MitEbene(o, e)); err == nil {
			t.Errorf("ungültige Angaben angenommen: %+v", e)
		}
	}
	if !EbenenNameGueltig(strings.Repeat("ä", MaxEbenenName)) || !EbenenNameGueltig("") {
		t.Error("gültiger Name abgelehnt")
	}
	if (Ebene{}).MitDeckkraft(150).Deckkraft() != 100 || (Ebene{}).MitDeckkraft(-3).Deckkraft() != 0 {
		t.Error("MitDeckkraft klemmt nicht")
	}
	// alle vier Arten tragen die Angaben
	for _, x := range []Objekt{Pfeil{}, Kasten{}, Text{}, Nummer{}} {
		if MitEbene(x, neu).Ebenenwerte() != neu {
			t.Errorf("%T trägt die Angaben nicht", x)
		}
	}
	if MitEbene(nil, neu) != nil {
		t.Error("MitEbene(nil)")
	}
}
