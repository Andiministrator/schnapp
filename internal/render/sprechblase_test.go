// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Sprechblase (ADR-0021 Nachtrag 13 Punkte 2 bis 4): Mit Füllung steht das Dreieck in der Füllfarbe, ohne Füllung zwei
// Schenkel in der Schriftfarbe, und ein Text ohne Spitze ändert sich nicht.
func TestSprechblasePixel(t *testing.T) {
	weiss := color.NRGBA{255, 255, 255, 255}
	rot := color.NRGBA{255, 0, 0, 255}
	blau := color.NRGBA{0, 0, 255, 255}
	spitze := document.Pt(50, 33)
	tx := document.Text{Ort: document.Pt(4, 4), Inhalt: "Hi", Schriftgrad: 10, Farbe: color.NRGBA{A: 255}, Hintergrund: rot,
		Spitze: &spitze}
	g := tx.Grenzen()
	a, sp, b, ok := tx.Spitzendreieck()
	if !ok || g.Max.X > 40 {
		t.Fatalf("Aufbau: Dreieck %v, Schild %v", ok, g)
	}
	// Ein Punkt im Dreieck: Mitte zwischen der Basis und der Spitze.
	innen := a.Plus(b).Mal(0.5).Plus(sp.Mal(0.25)).Mal(1 / 1.25)

	zeichne := func(tx document.Text) *image.NRGBA {
		d := weissesDok(t)
		if _, err := d.Hinzufuegen(tx); err != nil {
			t.Fatal(err)
		}
		b, err := Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	// Mit Füllung: im Dreieck rot, ohne Spitze an derselben Stelle weiß.
	if c := zeichne(tx).NRGBAAt(int(innen.X), int(innen.Y)); !nah(c, rot, 2) {
		t.Errorf("Dreieck mit Füllung bei %v: %v", innen, c)
	}
	ohne := tx
	ohne.Spitze = nil
	if c := zeichne(ohne).NRGBAAt(int(innen.X), int(innen.Y)); c != weiss {
		t.Errorf("ohne Spitze bei %v: %v", innen, c)
	}

	// Ohne Füllung: im Dreieck weiß, auf dem Schenkel in der Schriftfarbe, und das Schild bleibt leer.
	tx.Hintergrund = color.NRGBA{}
	tx.Farbe = blau
	bild := zeichne(tx)
	if c := bild.NRGBAAt(int(innen.X), int(innen.Y)); c != weiss {
		t.Errorf("Dreieck ohne Füllung bei %v: %v", innen, c)
	}
	mitte := a.Plus(sp).Mal(0.5)
	if c := bild.NRGBAAt(int(mitte.X), int(mitte.Y)); !nah(c, blau, 90) || c == weiss {
		t.Errorf("Schenkel bei %v: %v", mitte, c)
	}
	if c := bild.NRGBAAt(int(g.Max.X)-1, int(g.Min.Y)+1); c != weiss {
		t.Errorf("Schild ohne Füllung zeigt %v", c)
	}
}

// Die Blase hinter einer Ebene mit Deckkraft: Das Zwischenbild reicht bis zur Spitze (ebenenRahmen mit Hülle), das
// Dreieck wird nicht abgeschnitten.
func TestSprechblaseEbenenRahmen(t *testing.T) {
	rot := color.NRGBA{255, 0, 0, 255}
	spitze := document.Pt(55, 36)
	tx := document.Text{Ebene: document.Ebene{}.MitDeckkraft(50), Ort: document.Pt(4, 4), Inhalt: "Hi", Schriftgrad: 10,
		Farbe: color.NRGBA{A: 255}, Hintergrund: rot, Spitze: &spitze}
	d := weissesDok(t)
	if _, err := d.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	z := zeichner{d: d, rand: image.Rect(0, 0, 60, 40)}
	r := z.ebenenRahmen(tx)
	if !spitzeIn(r, spitze) {
		t.Errorf("Rahmen %v enthält die Spitze %v nicht", r, spitze)
	}
	b, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	a, sp, bb, ok := tx.Spitzendreieck()
	if !ok {
		t.Fatal("kein Dreieck")
	}
	innen := a.Plus(bb).Mal(0.5).Plus(sp.Mal(0.25)).Mal(1 / 1.25)
	if c := b.NRGBAAt(int(innen.X), int(innen.Y)); !nah(c, color.NRGBA{255, 127, 127, 255}, 3) {
		t.Errorf("halbdeckendes Dreieck bei %v: %v", innen, c)
	}
}

func spitzeIn(r image.Rectangle, p document.Punkt) bool {
	return image.Pt(int(p.X), int(p.Y)).In(r)
}

// Die Kosten der Füllung zählen im Trockenlauf mit (Budget): Das Dreieck der Blase kostet mehr als das Schild allein.
func TestSprechblaseBudget(t *testing.T) {
	spitze := document.Pt(55, 36)
	mit := document.Text{Ort: document.Pt(4, 4), Inhalt: "Hi", Schriftgrad: 10, Hintergrund: color.NRGBA{255, 0, 0, 255}, Spitze: &spitze}
	ohne := mit
	ohne.Spitze = nil
	dm, do := weissesDok(t), weissesDok(t)
	if _, err := dm.Hinzufuegen(mit); err != nil {
		t.Fatal(err)
	}
	if _, err := do.Hinzufuegen(ohne); err != nil {
		t.Fatal(err)
	}
	am, ao := Aufwand(dm), Aufwand(do)
	if am <= ao {
		t.Errorf("Aufwand mit Spitze %v kleiner als ohne %v", am, ao)
	}
}
