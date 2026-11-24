// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Navigator bei flachen oder schmalen Bildern und beim Zuschneiden (F-613, KS-7 und KS-8, Karte #236).

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Der Kasten eines flachen oder schmalen Bilds ist mindestens NavigatorMinSeite dick, das Bild steht darin mittig im
// richtigen Maßstab, und die Abbildung rechnet mit der Lage des Bilds, nicht mit der des Kastens.
func TestNavigatorFlachesBildStehtMittigImKasten(t *testing.T) {
	for _, c := range []struct {
		bild   image.Rectangle
		kasten image.Point
		inhalt image.Point
	}{
		{image.Rect(0, 0, 3840, 44), image.Pt(160, 16), image.Pt(160, 2)},
		{image.Rect(0, 0, 30, 4000), image.Pt(16, 160), image.Pt(1, 160)},
		{image.Rect(100, 50, 3100, 250), image.Pt(160, 16), image.Pt(160, 11)},
	} {
		n, ok := NeuerNavigator(c.bild, image.Pt(1600, 1000), NavigatorSeite, NavigatorRand)
		if !ok {
			t.Fatalf("Bild %v: kein Kasten", c.bild)
		}
		if n.Flaeche.Size() != c.kasten || n.Inhalt.Size() != c.inhalt {
			t.Errorf("Bild %v: Kasten %v Inhalt %v, Soll %v und %v", c.bild, n.Flaeche.Size(), n.Inhalt.Size(), c.kasten, c.inhalt)
		}
		if !n.Inhalt.In(n.Flaeche) {
			t.Errorf("Bild %v: Inhalt %v liegt nicht im Kasten %v", c.bild, n.Inhalt, n.Flaeche)
		}
		// Mittig: links/oben und rechts/unten bleibt gleich viel (höchstens ein Pixel Rundung) frei.
		if dx := (n.Inhalt.Min.X - n.Flaeche.Min.X) - (n.Flaeche.Max.X - n.Inhalt.Max.X); dx < -1 || dx > 1 {
			t.Errorf("Bild %v: Inhalt %v in X nicht mittig im Kasten %v", c.bild, n.Inhalt, n.Flaeche)
		}
		if dy := (n.Inhalt.Min.Y - n.Flaeche.Min.Y) - (n.Flaeche.Max.Y - n.Inhalt.Max.Y); dy < -1 || dy > 1 {
			t.Errorf("Bild %v: Inhalt %v in Y nicht mittig im Kasten %v", c.bild, n.Inhalt, n.Flaeche)
		}
		if !gleichf(n.Faktor, 160/float64(max(c.bild.Dx(), c.bild.Dy()))) {
			t.Errorf("Bild %v: Faktor %v verzerrt das Seitenverhältnis", c.bild, n.Faktor)
		}
		// Bildmitte liegt in der Mitte des Inhalts, die Ecke an seiner Ecke, und der Rückweg stimmt.
		m := document.Pt(float64(c.bild.Min.X+c.bild.Max.X)/2, float64(c.bild.Min.Y+c.bild.Max.Y)/2)
		k := n.ZuKasten(m)
		mx, my := float64(n.Inhalt.Min.X+n.Inhalt.Max.X)/2, float64(n.Inhalt.Min.Y+n.Inhalt.Max.Y)/2
		if k.X < mx-1 || k.X > mx+1 || k.Y < my-1 || k.Y > my+1 {
			t.Errorf("Bild %v: Bildmitte im Kasten bei %v, Soll (%v, %v)", c.bild, k, mx, my)
		}
		e := n.ZuKasten(document.Pt(float64(c.bild.Min.X), float64(c.bild.Min.Y)))
		if e.X != float64(n.Inhalt.Min.X) || e.Y != float64(n.Inhalt.Min.Y) {
			t.Errorf("Bild %v: obere linke Ecke im Kasten bei %v, Soll %v", c.bild, e, n.Inhalt.Min)
		}
		if b := n.ZuAnsicht(k); b.Minus(m).Laenge() > 1e-9 {
			t.Errorf("Bild %v: Rundweg %v → %v", c.bild, m, b)
		}
	}
	// Ein Bild mit normaler kurzer Seite behält seinen Kasten unverändert (kein Mindestmaß nötig).
	if n, _ := NeuerNavigator(image.Rect(0, 0, 1200, 800), image.Pt(900, 600), NavigatorSeite, NavigatorRand); n.Inhalt != n.Flaeche {
		t.Errorf("normales Bild: Inhalt %v, Soll gleich dem Kasten %v", n.Inhalt, n.Flaeche)
	}
	// Das Mindestmaß wächst mit der Skala (seite = 320 px bei Skala 2 → 32 px).
	if n, _ := NeuerNavigator(image.Rect(0, 0, 3840, 44), image.Pt(1600, 1000), 320, 24); n.Flaeche.Dy() != 32 {
		t.Errorf("Skala 2: Kastenhöhe %d, Soll 32", n.Flaeche.Dy())
	}
}

// KS-8, KM6: Beim Zuschneiden gilt die Vollansicht (Bildflaeche), nicht die Größe des Dokuments.
func TestSitzungNavigatorBeimZuschneidenNimmtDieVollansicht(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(300, 200, 900, 600)); err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(900, 600))
	s.SetzeWerkzeug(WerkzeugZuschneiden)
	s.SetzeZoom(400)
	n, ok := s.Navigator(NavigatorSeite, NavigatorRand)
	if bf := s.Bildflaeche(); !ok || n.Bild != bf || bf == image.Rect(0, 0, 600, 400) {
		t.Errorf("Zuschneiden: Navigator ok %v Bild %v, Soll die Vollansicht %v", ok, n.Bild, bf)
	}
}
