// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"image"
	"image/color"
	"testing"
)

// TestZugOhneWirkungSprechblase (F-694): Ein Zug, der eine Sprechblase mit wertgleicher Spitze, aber neuem Zeiger
// zurücksetzt (wie canvas.Verschiebe mit Versatz 0), hinterlässt keinen Rückgängig-Schritt und ändert den Stand nicht.
// Ein Zug, der die Spitze wirklich verschiebt, legt dagegen einen Schritt an.
func TestZugOhneWirkungSprechblase(t *testing.T) {
	for _, mitSpitze := range []bool{false, true} {
		d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 64, 48)))
		if err != nil {
			t.Fatal(err)
		}
		tx := Text{Ort: Pt(10, 10), Inhalt: "Hi", Schriftgrad: 12, Farbe: color.NRGBA{A: 255}}
		if mitSpitze {
			sp := Pt(40, 40)
			tx.Spitze = &sp
		}
		id, err := d.Hinzufuegen(tx)
		if err != nil {
			t.Fatal(err)
		}
		d.VergissVerlauf()
		vor := d.Stand()
		if err := d.BeginneZug(id); err != nil {
			t.Fatal(err)
		}
		o, _ := d.Objekt(id)
		v := o.(Text)
		v.Ort = v.Ort.Plus(Pt(0, 0))
		if v.Spitze != nil {
			s := v.Spitze.Plus(Pt(0, 0))
			v.Spitze = &s // gleicher Punkt, neuer Zeiger
		}
		if err := d.ZugSetze(v); err != nil {
			t.Fatal(err)
		}
		if err := d.BeendeZug(); err != nil {
			t.Fatal(err)
		}
		if d.KannRueckgaengig() || d.Stand() != vor {
			t.Errorf("mitSpitze=%v: Zug ohne Wirkung legt einen Schritt an (KannRueckgaengig=%v, Stand geändert=%v)",
				mitSpitze, d.KannRueckgaengig(), d.Stand() != vor)
		}
		if !mitSpitze {
			continue
		}
		// Gegenprobe: eine wirklich verschobene Spitze ist ein Schritt.
		if err := d.BeginneZug(id); err != nil {
			t.Fatal(err)
		}
		w := v
		s := Pt(41, 40)
		w.Spitze = &s
		if err := d.ZugSetze(w); err != nil {
			t.Fatal(err)
		}
		if err := d.BeendeZug(); err != nil {
			t.Fatal(err)
		}
		if !d.KannRueckgaengig() {
			t.Error("verschobene Spitze legt keinen Rückgängig-Schritt an")
		}
	}
}

// TestGleich (F-694): Gleich vergleicht die Spitze dem Wert nach und schreibt nicht durch die Zeiger.
func TestGleich(t *testing.T) {
	a, b, c := Pt(1, 2), Pt(1, 2), Pt(1, 3)
	t1 := Text{Inhalt: "x", Spitze: &a}
	t2 := Text{Inhalt: "x", Spitze: &b}
	t3 := Text{Inhalt: "x", Spitze: &c}
	t4 := Text{Inhalt: "x"}
	t5 := Text{Inhalt: "y", Spitze: &b}
	faelle := []struct {
		name string
		x, y Objekt
		soll bool
	}{
		{"gleicher Punkt, anderer Zeiger", t1, t2, true},
		{"anderer Punkt", t1, t3, false},
		{"eine ohne Spitze", t1, t4, false},
		{"beide ohne Spitze", t4, t4, true},
		{"anderer Inhalt", t2, t5, false},
		{"andere Art", t4, Kasten{}, false},
		{"Kasten gleich", Kasten{ID: 3}, Kasten{ID: 3}, true},
	}
	for _, f := range faelle {
		if got := Gleich(f.x, f.y); got != f.soll {
			t.Errorf("%s: Gleich = %v, soll %v", f.name, got, f.soll)
		}
	}
	if a != Pt(1, 2) || b != Pt(1, 2) || t1.Spitze != &a || t2.Spitze != &b {
		t.Error("Gleich hat die Spitze verändert")
	}
}
