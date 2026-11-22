// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Regressionstests aus der Kritik zu Sprint 11 (Gegenproben von Kritiker A, Befunde KA-1 und KB-11; ADR-0023
// Nachtrag 5).

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// KA-1: Esc ohne Zug und ohne Auswahl verlässt das Werkzeug Zuschneiden über denselben Weg wie SetzeWerkzeug: Die
// Leinwand baut ihr Bild neu (Vollansicht aus) und die Abbildung ist dieselbe wie nach SetzeWerkzeug(WerkzeugAuswahl).
func TestEscVerlaesstZuschneiden(t *testing.T) {
	zuschneiden := func() *Sitzung {
		s := verlaufSitzung(t)
		s.SetzeWerkzeug(WerkzeugZuschneiden)
		s.Druecken(f(200, 50), TastePrimaer)
		s.Ziehen(f(150, 50))
		s.Loslassen(f(150, 50))
		if z := s.D.Zuschnitt(); z != image.Rect(0, 0, 150, 100) {
			t.Fatalf("Vorbereitung: Zuschnitt %v", z)
		}
		return s
	}
	s := zuschneiden()
	stand := s.BildAenderung()
	s.Esc()
	if s.Werkzeug != WerkzeugAuswahl {
		t.Fatalf("Esc: Werkzeug %v", s.Werkzeug)
	}
	if s.BildAenderung() == stand {
		t.Errorf("BildAenderung unverändert (%d): der Editor zeigte weiter die Vollansicht", stand)
	}
	if s.Bildflaeche() != image.Rect(0, 0, 150, 100) {
		t.Errorf("Bildfläche nach Esc %v, Soll die Ansicht", s.Bildflaeche())
	}
	s2 := zuschneiden()
	s2.SetzeWerkzeug(WerkzeugAuswahl)
	if s.Abb != s2.Abb {
		t.Errorf("Abbildung nach Esc %+v, nach SetzeWerkzeug %+v", s.Abb, s2.Abb)
	}
}

// Was der Rahmen beim Griff-Zug in der Vollansicht zeigt, ist nach dem Loslassen die Ansicht — bei zufälligen
// Streifen, Zuschnitten, Griffen und Zielen (auch außerhalb der Vollansicht, dann geklemmt). Rückgängig stellt den
// alten Zuschnitt her und bleibt im Modus Zuschneiden (Gegenprobe Kritiker A, Sprint 11).
func TestZuschnittGriffZeigtWasKommt(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 90, 70))
	for y := range 70 {
		for x := range 90 {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x*7 + y*3), 255})
		}
	}
	zf := rand.New(rand.NewPCG(11, 386))
	geaendert := 0
	for fall := range 3000 {
		d, err := document.Neu(b)
		if err != nil {
			t.Fatal(err)
		}
		x0, y0 := zf.IntN(40), zf.IntN(30)
		z := image.Rect(x0, y0, x0+20+zf.IntN(90-x0-19), y0+20+zf.IntN(70-y0-19))
		var ok []document.Streifen
		for range zf.IntN(4) {
			zeilen := zf.IntN(2) == 0
			n := 90
			if zeilen {
				n = 70
			}
			von := zf.IntN(n - 1)
			st := document.Streifen{Zeilen: zeilen, Von: von, Bis: von + 1 + zf.IntN(min(8, n-von-1)+1)}
			if d.SetzeAnsicht(z, append(append([]document.Streifen(nil), ok...), st)) == nil {
				ok = append(ok, st)
			}
		}
		if d.SetzeAnsicht(z, ok) != nil {
			continue
		}
		s := NeueSitzung(d)
		s.SetzeFenster(image.Pt(600, 500))
		s.SetzeWerkzeug(WerkzeugZuschneiden)
		g := d.Groesse()
		if g.X < 16 || g.Y < 16 {
			continue
		}
		ecken := []document.Punkt{{X: 0, Y: 0}, {X: float64(g.X) / 2, Y: 0}, {X: float64(g.X), Y: 0}, {X: float64(g.X), Y: float64(g.Y) / 2},
			{X: float64(g.X), Y: float64(g.Y)}, {X: float64(g.X) / 2, Y: float64(g.Y)}, {X: 0, Y: float64(g.Y)}, {X: 0, Y: float64(g.Y) / 2}}
		start := s.Abb.ZuFenster(ecken[zf.IntN(8)])
		art, hat := s.zuschnittGriff(start)
		if !hat {
			t.Fatalf("Fall %d: kein Griff bei %v", fall, start)
		}
		fl := s.Bildflaeche()
		ziel := document.Pt(float64(fl.Min.X-5+zf.IntN(fl.Dx()+10))+0.3, float64(fl.Min.Y-5+zf.IntN(fl.Dy()+10))+0.3)
		voll, vv, zVor := d.VollAnsichtsbild(), d.VollVersatz(), d.Zuschnitt()
		// erwarteter Rahmen in Ansichtskoordinaten
		q := document.Pt(math.Max(float64(fl.Min.X), math.Min(float64(fl.Max.X), ziel.X)),
			math.Max(float64(fl.Min.Y), math.Min(float64(fl.Max.Y), ziel.Y)))
		ax, ay, bx, by := 0.0, 0.0, float64(g.X), float64(g.Y)
		switch art {
		case GriffNW, GriffW, GriffSW:
			ax = q.X
		case GriffNO, GriffO, GriffSO:
			bx = q.X
		}
		switch art {
		case GriffNW, GriffN, GriffNO:
			ay = q.Y
		case GriffSW, GriffS, GriffSO:
			by = q.Y
		}
		r := image.Rect(int(math.Round(ax)), int(math.Round(ay)), int(math.Round(bx)), int(math.Round(by)))
		s.Druecken(start, TastePrimaer)
		s.Ziehen(s.Abb.ZuFenster(ziel))
		s.Loslassen(s.Abb.ZuFenster(ziel))
		if d.Zuschnitt() == zVor {
			continue
		}
		geaendert++
		an := d.Ansichtsbild()
		if an.Rect.Size() != r.Size() {
			t.Fatalf("Fall %d (z %v, st %v, Griff %v, Ziel %v): Ansicht %v, Rahmen %v", fall, zVor, ok, art, ziel, an.Rect.Size(), r)
		}
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				if an.NRGBAAt(x, y) != voll.NRGBAAt(r.Min.X+vv.X+x, r.Min.Y+vv.Y+y) {
					t.Fatalf("Fall %d (z %v, st %v, Griff %v, Ziel %v, Rahmen %v): Pixel (%d,%d) weicht ab; neuer Zuschnitt %v",
						fall, zVor, ok, art, ziel, r, x, y, d.Zuschnitt())
				}
			}
		}
		if !s.Rueckgaengig() || d.Zuschnitt() != zVor || s.Werkzeug != WerkzeugZuschneiden {
			t.Fatalf("Fall %d: Rückgängig %v", fall, d.Zuschnitt())
		}
	}
	if geaendert < 100 {
		t.Errorf("nur %d Fälle mit geändertem Zuschnitt: die Probe greift kaum", geaendert)
	}
}

// KB-11: Ein Griff über die Gegenkante hinaus klappt den Rahmen um, mit nicht leerem Ergebnis: Die linke Kante eines
// Zuschnitts (50..150) rechts über die rechte Kante gezogen ergibt den Bereich zwischen alter rechter Kante und Ziel.
func TestZuschnittGriffKlapptUm(t *testing.T) {
	s := verlaufSitzung(t)
	if err := s.D.SetzeZuschnitt(image.Rect(50, 0, 150, 100)); err != nil {
		t.Fatal(err)
	}
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, fv(s, 0, 50), fv(s, 130, 50)) // linke Kante (Ansicht 0) auf Ansicht 130 = Original 180
	if z := s.D.Zuschnitt(); z != image.Rect(150, 0, 180, 100) {
		t.Errorf("Zuschnitt nach dem Umklappen %v, Soll (150,0)-(180,100)", z)
	}
}

// KB-11: Ein Griff-Zug, der den Rahmen unter die Mindestgröße bringt, ändert nichts (kein Rückgängig-Schritt).
func TestZuschnittGriffMindestgroesse(t *testing.T) {
	s := verlaufSitzung(t)
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	zieh(s, fv(s, 200, 50), fv(s, MinObjekt/2, 50)) // rechte Kante fast auf die linke
	if s.Geaendert() || s.D.Zuschnitt() != image.Rect(0, 0, 200, 100) {
		t.Errorf("zu kleiner Rahmen änderte den Zuschnitt: %v", s.D.Zuschnitt())
	}
	zieh(s, fv(s, 100, 100), fv(s, 100, MinObjekt+1)) // knapp darüber: wirkt
	if z := s.D.Zuschnitt(); z != image.Rect(0, 0, 200, 7) {
		t.Errorf("Rahmen knapp über der Mindestgröße: Zuschnitt %v, Soll (0,0)-(200,7)", z)
	}
}
