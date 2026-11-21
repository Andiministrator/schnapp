// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// F-109: Stand kehrt mit Rückgängig/Wiederholen zu früheren Kennungen zurück, vergibt eine Kennung nie zweimal und
// bleibt über VergissVerlauf erhalten.
func TestStandKennungen(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 20, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if d.Stand() != 0 {
		t.Fatalf("neues Dokument: Stand %d (Soll 0)", d.Stand())
	}
	d.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(5, 5)), Fuellung: color.NRGBA{A: 255}})
	s1 := d.Stand()
	if s1 != 1 {
		t.Errorf("nach Hinzufuegen: Stand %d (Soll 1)", s1)
	}
	d.Rueckgaengig()
	if d.Stand() != 0 {
		t.Errorf("nach Rückgängig: Stand %d (Soll 0)", d.Stand())
	}
	d.Wiederholen()
	if d.Stand() != s1 {
		t.Errorf("nach Wiederholen: Stand %d (Soll %d)", d.Stand(), s1)
	}
	// Neuer Schritt nach Rückgängig: neue Kennung, nicht die verworfene 1 und nicht 0.
	d.Rueckgaengig()
	d.Hinzufuegen(Kasten{Bereich: R(Pt(2, 2), Pt(6, 6)), Fuellung: color.NRGBA{A: 255}})
	s2 := d.Stand()
	if s2 != 2 {
		t.Errorf("neuer Schritt nach Rückgängig: Stand %d (Soll 2)", s2)
	}
	// Ein Zug zählt erst mit BeendeZug; ein Zug ohne Wirkung hinterlässt keinen neuen Stand.
	id := d.Objekte()[0].Kennung()
	d.BeginneZug(id)
	if d.Stand() != s2 {
		t.Error("Stand ändert sich während des Zugs")
	}
	d.BeendeZug()
	if d.Stand() != s2 {
		t.Errorf("Zug ohne Wirkung: Stand %d (Soll %d)", d.Stand(), s2)
	}
	// VergissVerlauf behält den Stand, danach geht die Zählung weiter.
	d.VergissVerlauf()
	if d.Stand() != s2 {
		t.Errorf("nach VergissVerlauf: Stand %d (Soll %d)", d.Stand(), s2)
	}
	d.Hinzufuegen(Kasten{Bereich: R(Pt(3, 3), Pt(7, 7)), Fuellung: color.NRGBA{A: 255}})
	if d.Stand() != 3 {
		t.Errorf("nach VergissVerlauf und neuem Schritt: Stand %d (Soll 3)", d.Stand())
	}
	d.Rueckgaengig()
	if d.Stand() != s2 {
		t.Errorf("Rückgängig bis zur Basis: Stand %d (Soll %d)", d.Stand(), s2)
	}
}
