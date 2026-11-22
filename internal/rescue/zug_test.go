// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"image/color"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-409: Eine Kopie, die während eines Zugs entsteht, trägt das Objekt im Zug nicht — auch kein leeres oder
// entartetes (leere Texteingabe, Pfeil der Länge null), das beim Anwenden ein unsichtbares Geisterobjekt hinterließe.
func TestErzeugeOhneObjektImZug(t *testing.T) {
	rot := document.Strich{Farbe: color.NRGBA{255, 0, 0, 255}, Breite: 3}
	faelle := []struct {
		name string
		o    document.Objekt
	}{
		{"leerer Text", document.Text{Ort: document.Pt(10, 10), Inhalt: "", Schriftgrad: 20, Farbe: color.NRGBA{0, 0, 0, 255}}},
		{"Pfeil der Länge null", document.GeraderPfeil(document.Pt(5, 5), document.Pt(5, 5), rot)},
		{"Text", document.Text{Ort: document.Pt(10, 10), Inhalt: "noch nicht übernommen", Schriftgrad: 20, Farbe: color.NRGBA{0, 0, 0, 255}}},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			d, err := document.Neu(bild(200, 100))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.Hinzufuegen(document.GeraderPfeil(document.Pt(1, 1), document.Pt(50, 50), rot)); err != nil {
				t.Fatal(err)
			}
			if _, err := d.BeginneNeuenZug(f.o); err != nil {
				t.Fatal(err)
			}
			kp, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			neu, err := document.Neu(bild(200, 100))
			if err != nil {
				t.Fatal(err)
			}
			if err := Anwenden(neu, &kp, nil); err != nil {
				t.Fatal(err)
			}
			if n := len(neu.Objekte()); n != 1 {
				t.Errorf("nach dem Anwenden %d Objekte, erwartet 1 (nur der übernommene Pfeil): %#v", n, neu.Objekte())
			}
		})
	}
}

// F-409: Während eines Verschiebens trägt die Kopie die Lage vor dem Zug.
func TestErzeugeVerschiebenImZug(t *testing.T) {
	d, err := document.Neu(bild(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	k := document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(40, 30)),
		Strich: document.Strich{Farbe: color.NRGBA{0, 0, 255, 255}, Breite: 2}}
	id, err := d.Hinzufuegen(k)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := d.Objekt(id)
	if err := d.BeginneZug(id); err != nil {
		t.Fatal(err)
	}
	weg := o.(document.Kasten)
	weg.Bereich = document.R(document.Pt(100, 50), document.Pt(130, 70))
	if err := d.ZugSetze(weg); err != nil {
		t.Fatal(err)
	}
	kp, err := Erzeuge(d, Kennung{Pfad: "/x"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	neu, err := document.Neu(bild(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	if err := Anwenden(neu, &kp, nil); err != nil {
		t.Fatal(err)
	}
	got := neu.Objekte()
	if len(got) != 1 {
		t.Fatalf("%d Objekte", len(got))
	}
	if r := got[0].(document.Kasten).Bereich; r != k.Bereich {
		t.Errorf("Kasten in der Kopie bei %v, erwartet die Lage vor dem Zug %v", r, k.Bereich)
	}
}
