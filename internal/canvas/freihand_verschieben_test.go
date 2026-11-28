// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// QA #333 (Fable): Runden im Verschieben (F-1036) trifft die neue Längenprüfung (F-1040).
func TestQARundenMachtStrichOhneLaenge(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	st := document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6}
	// Ein Tupfer unter 1/100 px fällt beim Anlegen auf einen Punkt zusammen und wird abgelehnt (F-1045: Punkte liegen auf
	// Hundertsteln); vorher entstand er und scheiterte erst beim Verschieben.
	if _, err := d.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(50.001, 50), document.Pt(50.004, 50)}, st)); err == nil {
		t.Error("Tupfer unter 1/100 px angenommen")
	}
	// Der kürzeste mögliche Tupfer (Länge 0,01) lässt sich verschieben.
	id, err := d.Hinzufuegen(document.NeuerFreihand([]document.Punkt{document.Pt(50.01, 50), document.Pt(50.02, 50)}, st))
	if err != nil {
		t.Fatalf("Tupfer wird schon beim Anlegen abgelehnt: %v", err)
	}
	o, _ := d.Objekt(id)
	if err := d.Ersetzen(Verschiebe(o, document.Pt(10, 0))); err != nil {
		t.Errorf("Verschieben um 10 px scheitert: %v", err)
	}
	// Zusammen mit einem zweiten Objekt (alles oder nichts).
	id2, err := d.Hinzufuegen(document.Kasten{Bereich: document.Rechteck{Min: document.Pt(10, 10), Max: document.Pt(40, 40)}, Strich: st})
	if err != nil {
		t.Fatal(err)
	}
	k, _ := d.Objekt(id2)
	if err := d.ErsetzeMehrere([]document.Objekt{Verschiebe(o, document.Pt(10, 0)), Verschiebe(k, document.Pt(10, 0))}); err != nil {
		t.Errorf("gemeinsames Verschieben scheitert: %v", err)
	}
}

// QA: ein Strich mit ungerundeten Lagen gilt nach Verschieben um 0 nicht mehr als derselbe (Einfügen: stehtSchonDa).
func TestQAUngerundetNichtMehrGleich(t *testing.T) {
	st := document.Strich{Farbe: color.NRGBA{A: 255}, Breite: 6}
	f := document.NeuerFreihand([]document.Punkt{document.Pt(10.123456, 20), document.Pt(60, 40.5)}, st)
	if !document.GleichOhneKennung(Verschiebe(f, document.Pt(0, 0)), f) {
		t.Error("Verschieben um 0 ändert den Strich (Doppel-Erkennung beim Einfügen greift nicht)")
	}
}
