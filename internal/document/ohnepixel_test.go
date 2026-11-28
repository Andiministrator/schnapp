// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"testing"
)

// ADR-0045 Punkt 2: Ein Dokument ohne Pixel trägt Größe, Zuschnitt, Streifen und Kante, hat keinen Verlauf und bildet
// zwischen Original und Ansicht ab wie das Dokument mit Pixeln.
func TestOhnePixel(t *testing.T) {
	st := []Streifen{{Von: 10, Bis: 20}}
	d, err := OhnePixel(image.Pt(100, 50), image.Rect(5, 0, 100, 50), st, KanteWelle)
	if err != nil {
		t.Fatal(err)
	}
	if d.KannRueckgaengig() || d.Zuschnitt() != image.Rect(5, 0, 100, 50) || len(d.Streifen()) != 1 || d.Kante() != KanteWelle {
		t.Errorf("Verlauf %v, Zuschnitt %v, Streifen %v, Kante %v", d.KannRueckgaengig(), d.Zuschnitt(), d.Streifen(), d.Kante())
	}
	if p := d.ZuAnsicht(Pt(30, 7)); p != Pt(15, 7) { // 30 − 5 Zuschnitt − 10 Streifen
		t.Errorf("ZuAnsicht %v", p)
	}
	if _, err := OhnePixel(image.Pt(0, 5), image.Rect(0, 0, 0, 5), nil, KanteAbriss); err == nil {
		t.Error("Größe 0 angenommen")
	}
	if _, err := OhnePixel(image.Pt(10, 10), image.Rect(0, 0, 20, 20), nil, KanteAbriss); err == nil {
		t.Error("Zuschnitt außerhalb angenommen")
	}
}

// ADR-0045 Punkt 3: HinzufuegenMehrere legt in Reihenfolge zuoberst ab, als ein Schritt; leer ohne Schritt.
func TestHinzufuegenMehrere(t *testing.T) {
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	a, _ := d.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(2, 2))})
	stand := d.Stand()
	if ids, err := d.HinzufuegenMehrere(nil); err != nil || ids != nil || d.Stand() != stand {
		t.Errorf("leer: %v %v, Stand geändert %v", ids, err, d.Stand() != stand)
	}
	ids, err := d.HinzufuegenMehrere([]Objekt{Nummer{Mitte: Pt(3, 3), Wert: 1, Radius: 2}, Kasten{Bereich: R(Pt(4, 4), Pt(5, 5))}})
	if err != nil || len(ids) != 2 {
		t.Fatal(ids, err)
	}
	os := d.Objekte()
	if os[0].Kennung() != a || os[1].Kennung() != ids[0] || os[2].Kennung() != ids[1] {
		t.Errorf("Reihenfolge %v, erwartet %d %v", os, a, ids)
	}
	if err := d.Rueckgaengig(); err != nil || len(d.Objekte()) != 1 {
		t.Errorf("Rückgängig: %v, %d Objekte", err, len(d.Objekte()))
	}
	if err := d.Wiederholen(); err != nil || len(d.Objekte()) != 3 || d.Objekte()[2].Kennung() != ids[1] {
		t.Errorf("Wiederholen: %v, %v", err, d.Objekte())
	}
	if _, err := d.HinzufuegenMehrere([]Objekt{Kasten{Strich: Strich{Breite: -1}}}); err == nil {
		t.Error("ungültiges Objekt angenommen")
	}
}
