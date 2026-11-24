// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Regressionstests zum Navigator (Sammelkritik Sprint 12 KS-7, F-613, Karte #236).

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)


// KS-7 (#197): Sehr flache oder schmale Bilder (Leisten-Aufnahme 3840×44, gestauchte Streifen) ergeben einen Kasten von
// 1 bis 2 Pixeln Höhe. Die Oberfläche füllt Flächen bis 2 px ganz mit dem Rand (randStreifen): vom Bild ist nichts zu
// sehen, das Ziel für den Zeiger ist ein Strich.
func TestNavigatorSehrFlachesBild(t *testing.T) {
	for _, g := range []image.Point{{3840, 44}, {32768, 10}, {30, 4000}} {
		n, ok := NeuerNavigator(image.Rectangle{Max: g}, image.Pt(1600, 1000), NavigatorSeite, NavigatorRand)
		t.Logf("Bild %v: Kasten %v (%d × %d), gezeigt %v", g, n.Flaeche, n.Flaeche.Dx(), n.Flaeche.Dy(), ok)
		if ok && (n.Flaeche.Dx() <= 2 || n.Flaeche.Dy() <= 2) {
			t.Errorf("Bild %v: Kasten %d × %d wird gezeigt, ist aber nur Rand (kein Bildpixel, kaum zu treffen)", g,
				n.Flaeche.Dx(), n.Flaeche.Dy())
		}
	}
}

// Prüfung (#197): Hin- und Rückweg der Abbildung bei Ursprung ≠ 0 und nicht quadratischem Bild; Ausschnitt bei einem
// Fenster, das in einer Achse größer ist als das Bild.
func TestNavigatorRundweg(t *testing.T) {
	bild := image.Rect(-300, -200, 900, 600)
	n, ok := NeuerNavigator(bild, image.Pt(1000, 700), 160, 12)
	if !ok {
		t.Fatal("kein Navigator")
	}
	for _, a := range []document.Punkt{document.Pt(-300, -200), document.Pt(0, 0), document.Pt(900, 600), document.Pt(123.5, 77.25)} {
		if b := n.ZuAnsicht(n.ZuKasten(a)); b.Minus(a).Laenge() > 1e-9 {
			t.Errorf("Rundweg %v → %v", a, b)
		}
	}
	// Zoom 1, Bild 1200×800 im Fenster 1000×700 mittig in keiner Achse; Zoom 0,9: Breite 1080 > 1000, Höhe 720 > 700.
	abb := ZentriertAuf(Abbildung{Zoom: 2}, document.Pt(300, 200), bild, image.Pt(1000, 700))
	a := n.Ausschnitt(abb, image.Pt(1000, 700))
	// Sichtbar: 500 × 350 Ansichtspixel um (300, 200) → im Kasten Faktor 160/1200.
	f := 160.0 / 1200
	wantB, wantH := 500*f, 350*f
	if b, h := a.Max.X-a.Min.X, a.Max.Y-a.Min.Y; b < wantB-1e-6 || b > wantB+1e-6 || h < wantH-1e-6 || h > wantH+1e-6 {
		t.Errorf("Ausschnitt %v: %v × %v, Soll %v × %v", a, b, h, wantB, wantH)
	}
	m := n.ZuKasten(document.Pt(300, 200))
	if mx, my := (a.Min.X+a.Max.X)/2, (a.Min.Y+a.Max.Y)/2; mx < m.X-1e-6 || mx > m.X+1e-6 || my < m.Y-1e-6 || my > m.Y+1e-6 {
		t.Errorf("Mitte des Ausschnitts (%v, %v), Soll %v", mx, my, m)
	}
}
