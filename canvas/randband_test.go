// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// randSitzung ist ein Bild 200×200 mit Pixeln, die ihre Lage tragen, im Fenster 248×248.
func randSitzung(t *testing.T) *Sitzung {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x + 1), uint8(y + 1), 77, 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(248, 248))
	return s
}

// schwarzeZeilen liefert die Zeilen des Originals, in denen Spalte x geschwärzt ist.
func schwarzeZeilen(o *image.NRGBA, x int) (res []int) {
	for y := o.Rect.Min.Y; y < o.Rect.Max.Y; y++ {
		if (o.NRGBAAt(x, y) == color.NRGBA{A: 255}) {
			res = append(res, y)
		}
	}
	return res
}

// F-222 (E-21): Ein Stauch-Band bis an den Rand wird ein Zuschnitt, kein Streifen ohne Naht. Auch ohne Verlauf (nach
// dem Speichern mit Schwärzung oder Wiederöffnen) holt „Ganzes Bild“ das Band zurück.
func TestStauchenAmRandWirdZuschnitt(t *testing.T) {
	for _, fall := range []struct {
		name     string
		von, bis document.Punkt
		soll     image.Rectangle
	}{
		{"unten", document.Pt(100, 150), document.Pt(100, 900), image.Rect(0, 0, 200, 150)},
		{"oben", document.Pt(100, 40), document.Pt(100, -300), image.Rect(0, 40, 200, 200)},
		{"links", document.Pt(30, 100), document.Pt(-300, 100), image.Rect(30, 0, 200, 200)},
		{"rechts", document.Pt(170, 100), document.Pt(900, 100), image.Rect(0, 0, 170, 200)},
	} {
		t.Run(fall.name, func(t *testing.T) {
			s := randSitzung(t)
			s.SetzeWerkzeug(WerkzeugStauchen)
			zieh(s, fv(s, fall.von.X, fall.von.Y), fv(s, fall.bis.X, fall.bis.Y))
			if len(s.D.Streifen()) != 0 || s.D.Zuschnitt() != fall.soll {
				t.Fatalf("Streifen %v, Zuschnitt %v (Soll %v)", s.D.Streifen(), s.D.Zuschnitt(), fall.soll)
			}
			if m := s.NimmAnsichtMeldung(); m != AnsichtRand {
				t.Errorf("Meldung %v, Soll AnsichtRand", m)
			}
			if err := s.D.VergissVerlauf(); err != nil {
				t.Fatal(err)
			}
			if !s.KannGanzesBild() || !s.HebeZuschnittAuf() || s.D.Zuschnitt() != image.Rect(0, 0, 200, 200) {
				t.Errorf("Ganzes Bild: Zuschnitt %v", s.D.Zuschnitt())
			}
			if s.KannGanzesBild() {
				t.Error("Knopf bleibt an, obwohl schon alles zu sehen ist")
			}
		})
	}
}

// Das Rand-Band nach einem Zuschnitt und neben einem Streifen: Die andere Achse behält den Zuschnitt, ein Streifen
// mitten im Bild bleibt mit seiner Naht; Rückgängig nimmt das Band in einem Schritt zurück.
func TestStauchenAmRandMitZuschnittUndStreifen(t *testing.T) {
	s := randSitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(20, 0, 180, 200)); err != nil {
		t.Fatal(err)
	}
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 50, Bis: 60}}); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	vor := s.D.Stand()
	s.SetzeWerkzeug(WerkzeugStauchen)
	zieh(s, fv(s, 80, 140), fv(s, 80, 500)) // Ansicht 140 = Original 150
	if z := s.D.Zuschnitt(); z != image.Rect(20, 0, 180, 150) {
		t.Errorf("Zuschnitt %v, Soll (20,0)-(180,150)", z)
	}
	if st := s.D.Streifen(); len(st) != 1 || len(s.D.Naehte()) != 1 {
		t.Errorf("Streifen %v, Nähte %v", st, s.D.Naehte())
	}
	if !s.Rueckgaengig() || s.D.Stand() != vor || s.D.Zuschnitt() != image.Rect(20, 0, 180, 200) {
		t.Errorf("Rückgängig: Zuschnitt %v", s.D.Zuschnitt())
	}
}

// Ältere Dateien können einen Streifen am Rand des Originals tragen (vor F-222): „Ganzes Bild“ entfernt ihn mit dem
// Zuschnitt in einem Schritt, Streifen mit Naht bleiben.
func TestGanzesBildEntferntRandstreifen(t *testing.T) {
	s := randSitzung(t)
	if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 150, Bis: 200}, {Von: 80, Bis: 90}}); err != nil {
		t.Fatal(err)
	}
	if err := s.D.SetzeZuschnitt(image.Rect(10, 0, 200, 200)); err != nil {
		t.Fatal(err)
	}
	if err := s.D.VergissVerlauf(); err != nil {
		t.Fatal(err)
	}
	if !s.HebeZuschnittAuf() {
		t.Fatal("nichts geändert")
	}
	if st := s.D.Streifen(); len(st) != 1 || st[0] != (document.Streifen{Von: 80, Bis: 90}) {
		t.Errorf("Streifen %v (Soll nur der mit Naht)", st)
	}
	if !s.Rueckgaengig() || len(s.D.Streifen()) != 2 || s.D.Zuschnitt() != image.Rect(10, 0, 200, 200) {
		t.Errorf("Rückgängig: Streifen %v, Zuschnitt %v", s.D.Streifen(), s.D.Zuschnitt())
	}
}

// F-223: Ein Schwärzungsrahmen über den Rand der Ansicht hinaus schwärzt keinen verborgenen Randbereich mit.
func TestSchwaerzenAmRandTrifftNichtsVerborgenes(t *testing.T) {
	s := randSitzung(t)
	s.SetzeWerkzeug(WerkzeugStauchen)
	zieh(s, fv(s, 100, 150), fv(s, 100, 900)) // unten weg: Zuschnitt bis 150
	s.SetzeWerkzeug(WerkzeugSchwaerzen)
	zieh(s, fv(s, 50, 100), fv(s, 80, 900))
	z := schwarzeZeilen(s.D.Original(), 60)
	if len(z) == 0 || z[0] != 100 || z[len(z)-1] != 149 {
		t.Errorf("geschwärzte Zeilen %v, Soll 100..149", z)
	}
}

// F-223: Endet oder beginnt der Rahmen genau auf einer Naht, bleibt der Streifen dahinter bzw. davor unberührt;
// liegt er mitten im Rahmen, wird er mitgeschwärzt (ADR-0023 Punkt 8).
func TestSchwaerzenGenauAufNaht(t *testing.T) {
	for _, fall := range []struct {
		name     string
		von, bis float64 // Ansicht; Naht bei 100 (Original 100..150 verborgen)
		erst     int
		letzt    int
	}{
		{"endet auf Naht", 60, 100, 60, 99},
		{"beginnt auf Naht", 100, 130, 150, 179},
		{"endet 0,5 davor", 60, 99.5, 60, 99},
		{"über die Naht", 90, 110, 90, 159},
	} {
		s := randSitzung(t)
		if err := s.D.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 100, Bis: 150}}); err != nil {
			t.Fatal(err)
		}
		s.Abb = Abbildung{Zoom: 1, Versatz: document.Pt(24, 24)}
		s.SetzeWerkzeug(WerkzeugSchwaerzen)
		zieh(s, fv(s, 50, fall.von), fv(s, 80, fall.bis))
		z := schwarzeZeilen(s.D.Original(), 60)
		if len(z) == 0 || z[0] != fall.erst || z[len(z)-1] != fall.letzt || len(z) != fall.letzt-fall.erst+1 {
			t.Errorf("%s: geschwärzt %v, Soll %d..%d", fall.name, z, fall.erst, fall.letzt)
		}
	}
}
