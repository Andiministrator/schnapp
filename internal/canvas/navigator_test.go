// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Navigator (ADR-0031 Nachtrag 9, Karte #197). Bild 1200×800 im Fenster 900×600, Kasten längste Seite 160, Abstand 12:
// Faktor 160/1200, Kasten 160×107 (106,67 gerundet), Ende (888, 588), Anfang (728, 481).

func gleichf(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNavigatorSichtbarRegel(t *testing.T) {
	bild := image.Rect(0, 0, 1200, 800)
	fenster := image.Pt(900, 600)
	for _, fall := range []struct {
		name  string
		bild  image.Rectangle
		zoom  float64
		sicht bool
	}{
		{"100 Prozent, beide Achsen ragen hinaus", bild, 1, true},
		{"Zoom 0,5 passt ganz", bild, 0.5, false},
		{"Zoom 0,75 füllt genau das Fenster", bild, 0.75, false},
		{"Zoom 0,76 ragt in beiden Achsen hinaus", bild, 0.76, true},
		{"nur die Höhe ragt hinaus", image.Rect(0, 0, 100, 1000), 1, true},
		{"nur die Breite ragt hinaus", image.Rect(0, 0, 1000, 100), 1, true},
		{"kleines Bild bei 100 Prozent", image.Rect(0, 0, 400, 300), 1, false},
		{"leere Bildfläche", image.Rectangle{}, 1, false},
	} {
		if got := NavigatorSichtbar(fall.bild, fall.zoom, fenster); got != fall.sicht {
			t.Errorf("%s: %v, Soll %v", fall.name, got, fall.sicht)
		}
	}
	if NavigatorSichtbar(bild, 1, image.Pt(0, 0)) {
		t.Error("Fenster ohne Fläche: kein Navigator")
	}
}

func TestNeuerNavigatorLage(t *testing.T) {
	n, ok := NeuerNavigator(image.Rect(0, 0, 1200, 800), image.Pt(900, 600), 160, 12)
	if !ok {
		t.Fatal("Kasten passt, ok falsch")
	}
	if want := image.Rect(728, 481, 888, 588); n.Flaeche != want {
		t.Errorf("Fläche %v, Soll %v", n.Flaeche, want)
	}
	if !gleichf(n.Faktor, 160.0/1200) {
		t.Errorf("Faktor %v, Soll %v", n.Faktor, 160.0/1200)
	}
	// Hochformat: die Höhe ist die längste Seite.
	h, ok := NeuerNavigator(image.Rect(0, 0, 400, 800), image.Pt(900, 600), 160, 12)
	if !ok || h.Flaeche.Dy() != 160 || h.Flaeche.Dx() != 80 || h.Flaeche.Max != image.Pt(888, 588) {
		t.Errorf("Hochformat: %v ok %v, Soll 80×160 mit Ende (888,588)", h.Flaeche, ok)
	}
	// Bildfläche mit Ursprung ≠ 0 (Vollansicht beim Zuschneiden): Seitenverhältnis der Fläche, nicht des Ursprungs.
	v, ok := NeuerNavigator(image.Rect(-100, -50, 1100, 750), image.Pt(900, 600), 160, 12)
	if !ok || v.Flaeche.Dx() != 160 || v.Flaeche.Dy() != 107 {
		t.Errorf("Vollansicht: %v ok %v, Soll 160×107", v.Flaeche, ok)
	}
	if _, ok := NeuerNavigator(image.Rect(0, 0, 1200, 800), image.Pt(150, 150), 160, 12); ok {
		t.Error("Kasten größer als das Fenster: ok falsch erwartet")
	}
	if _, ok := NeuerNavigator(image.Rectangle{}, image.Pt(900, 600), 160, 12); ok {
		t.Error("leere Bildfläche: ok falsch erwartet")
	}
}

func TestNavigatorAbbildung(t *testing.T) {
	n, _ := NeuerNavigator(image.Rect(0, 0, 1200, 800), image.Pt(900, 600), 160, 12)
	if p := n.ZuKasten(document.Pt(0, 0)); !gleichf(p.X, 728) || !gleichf(p.Y, 481) {
		t.Errorf("Bildecke oben links im Kasten bei %v, Soll (728, 481)", p)
	}
	if p := n.ZuKasten(document.Pt(600, 400)); !gleichf(p.X, 728+80) || !gleichf(p.Y, 481+400*160.0/1200) {
		t.Errorf("Bildmitte im Kasten bei %v", p)
	}
	if p := n.ZuAnsicht(document.Pt(808, 481+400*160.0/1200)); !gleichf(p.X, 600) || !gleichf(p.Y, 400) {
		t.Errorf("Kastenmitte in der Ansicht bei %v, Soll (600, 400)", p)
	}
	// Zeiger außerhalb des Kastens: nächster Rand der Bildfläche.
	if p := n.ZuAnsicht(document.Pt(5000, 5000)); p != document.Pt(1200, 800) {
		t.Errorf("weit unten rechts: %v, Soll (1200, 800)", p)
	}
	if p := n.ZuAnsicht(document.Pt(-5, 300)); !gleichf(p.X, 0) {
		t.Errorf("links außerhalb: %v, Soll x 0", p)
	}
	if !n.Enthaelt(document.Pt(728, 481)) || !n.Enthaelt(document.Pt(888, 588)) || n.Enthaelt(document.Pt(727, 500)) ||
		n.Enthaelt(document.Pt(800, 589)) {
		t.Error("Enthaelt: Rand gehört dazu, ein Pixel daneben nicht")
	}
}

func TestNavigatorAusschnitt(t *testing.T) {
	n, _ := NeuerNavigator(image.Rect(0, 0, 1200, 800), image.Pt(900, 600), 160, 12)
	f := 160.0 / 1200
	// Zoom 4, Versatz (−400, −200): Fenster 900×600 zeigt die Ansicht von (100, 50) bis (325, 200).
	a := Abbildung{Zoom: 4, Versatz: document.Pt(-400, -200)}
	r := n.Ausschnitt(a, image.Pt(900, 600))
	if !gleichf(r.Min.X, 728+100*f) || !gleichf(r.Min.Y, 481+50*f) || !gleichf(r.Max.X, 728+325*f) || !gleichf(r.Max.Y, 481+200*f) {
		t.Errorf("Ausschnitt %v", r)
	}
	// Ragt das Fenster links über das Bild hinaus, bleibt der Ausschnitt am Bildrand: Ansicht von (−300, 0) bis (600, 600).
	r = n.Ausschnitt(Abbildung{Zoom: 1, Versatz: document.Pt(300, 0)}, image.Pt(900, 600))
	if !gleichf(r.Min.X, 728) || !gleichf(r.Max.X, 728+600*f) || !gleichf(r.Min.Y, 481) || !gleichf(r.Max.Y, 481+600*f) {
		t.Errorf("Ausschnitt am Rand %v, Soll links am Kastenrand, rechts bei Ansicht 600", r)
	}
}

func TestZentriertAufBegrenzt(t *testing.T) {
	bild := image.Rect(0, 0, 1200, 800)
	fenster := image.Pt(900, 600)
	a := Abbildung{Zoom: 4, Versatz: document.Pt(-1234, -999)}
	// Nahe der linken oberen Ecke: Halbbreite 112,5 und Halbhöhe 75 sind die Grenzen, der Versatz wird 0.
	if v := ZentriertAuf(a, document.Pt(10, 10), bild, fenster).Versatz; !gleichf(v.X, 0) || !gleichf(v.Y, 0) {
		t.Errorf("Ecke oben links: Versatz %v, Soll (0, 0)", v)
	}
	// Mitte des Bilds: 450 − 600·4, 300 − 400·4.
	if v := ZentriertAuf(a, document.Pt(600, 400), bild, fenster).Versatz; !gleichf(v.X, -1950) || !gleichf(v.Y, -1300) {
		t.Errorf("Bildmitte: Versatz %v, Soll (−1950, −1300)", v)
	}
	// Ecke unten rechts: Mitte höchstens (1200 − 112,5, 800 − 75).
	if v := ZentriertAuf(a, document.Pt(1200, 800), bild, fenster).Versatz; !gleichf(v.X, 450-1087.5*4) || !gleichf(v.Y, 300-725*4) {
		t.Errorf("Ecke unten rechts: Versatz %v, Soll (%v, %v)", v, 450-1087.5*4, 300-725*4)
	}
	// Das Bild ist in der Achse kleiner als das Fenster: es steht mittig, unabhängig vom Zielpunkt.
	klein := image.Rect(0, 0, 400, 300)
	if v := ZentriertAuf(Abbildung{Zoom: 1}, document.Pt(390, 10), klein, fenster).Versatz; !gleichf(v.X, 250) || !gleichf(v.Y, 150) {
		t.Errorf("kleines Bild: Versatz %v, Soll (250, 150)", v)
	}
	// Der Zoom bleibt unverändert.
	if z := ZentriertAuf(a, document.Pt(10, 10), bild, fenster).Zoom; z != 4 {
		t.Errorf("Zoom %v, Soll 4", z)
	}
}

func navigatorLauf(t *testing.T) *Sitzung {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1200, 800)))
	if err != nil {
		t.Fatal(err)
	}
	s := NeueSitzung(d)
	s.SetzeFenster(image.Pt(900, 600))
	return s
}

func TestSitzungNavigatorErstBeiZoom(t *testing.T) {
	s := navigatorLauf(t)
	if _, ok := s.Navigator(NavigatorSeite, NavigatorRand); ok {
		t.Error("eingepasst: Soll kein Navigator")
	}
	s.SetzeZoom(400)
	n, ok := s.Navigator(NavigatorSeite, NavigatorRand)
	if !ok || n.Flaeche != image.Rect(728, 481, 888, 588) {
		t.Errorf("bei 400 %%: ok %v, Fläche %v", ok, n.Flaeche)
	}
	s.SetzeZoom(50)
	if _, ok := s.Navigator(NavigatorSeite, NavigatorRand); ok {
		t.Error("bei 50 %: Soll kein Navigator")
	}
	s.SetzeZoom(400)
	s.Einpassen()
	if _, ok := s.Navigator(NavigatorSeite, NavigatorRand); ok {
		t.Error("nach Einpassen: Soll kein Navigator")
	}
}

func TestSitzungNavigatorSchalter(t *testing.T) {
	s := navigatorLauf(t)
	s.SetzeZoom(400)
	if !s.NavigatorAn() {
		t.Fatal("Voreinstellung: an")
	}
	s.SchalteNavigator()
	if s.NavigatorAn() {
		t.Error("nach dem Umschalten: aus")
	}
	if _, ok := s.Navigator(NavigatorSeite, NavigatorRand); ok {
		t.Error("ausgeschaltet: Soll kein Navigator, obwohl gezoomt")
	}
	s.SchalteNavigator()
	if _, ok := s.Navigator(NavigatorSeite, NavigatorRand); !ok {
		t.Error("wieder an: Soll Navigator")
	}
}

func TestSitzungNavigatorKlickZentriert(t *testing.T) {
	s := navigatorLauf(t)
	s.SetzeZoom(400)
	n, _ := s.Navigator(NavigatorSeite, NavigatorRand)
	// Kastenmitte = Bildmitte (600, 400): Der Ausschnitt liegt in der Mitte, die Ansicht zeigt (600, 400) in der
	// Fenstermitte.
	s.NavigatorZu(n, document.Pt(808, 481+400*160.0/1200))
	if m := s.Abb.ZuAnsicht(document.Pt(450, 300)); !gleichf(m.X, 600) || !gleichf(m.Y, 400) {
		t.Errorf("Fenstermitte zeigt %v, Soll (600, 400)", m)
	}
	if s.Eingepasst() || s.Abb.Zoom != 4 {
		t.Errorf("eingepasst %v, Zoom %v, Soll nicht eingepasst, Zoom 4", s.Eingepasst(), s.Abb.Zoom)
	}
	// Klick in die obere linke Kastenecke: der Ausschnitt rückt an die Bildecke.
	s.NavigatorZu(n, document.Pt(728, 481))
	if v := s.Abb.Versatz; !gleichf(v.X, 0) || !gleichf(v.Y, 0) {
		t.Errorf("Ecke oben links: Versatz %v, Soll (0, 0)", v)
	}
	// Ziehen aus dem Kasten heraus: unten rechts begrenzt.
	s.NavigatorZu(n, document.Pt(5000, 5000))
	if v := s.Abb.Versatz; !gleichf(v.X, 900-4800) || !gleichf(v.Y, 600-3200) {
		t.Errorf("unten rechts: Versatz %v, Soll (%v, %v)", v, 900-4800, 600-3200)
	}
}

// Eine von Hand gewählte Stelle bleibt, wenn sich das Fenster danach ändert: Die Ansicht folgt dem Fenster nicht mehr.
func TestSitzungNavigatorKlickBeendetEinpassen(t *testing.T) {
	s := navigatorLauf(t)
	s.Einpassen()
	if !s.Eingepasst() {
		t.Fatal("Aufbau: Soll eingepasst sein")
	}
	n, _ := NeuerNavigator(s.Bildflaeche(), s.Fenster(), NavigatorSeite, NavigatorRand)
	s.NavigatorZu(n, document.Pt(780, 520))
	if s.Eingepasst() {
		t.Error("nach dem Klick: Soll nicht mehr eingepasst sein")
	}
	vorher := s.Abb
	s.SetzeFenster(image.Pt(800, 500))
	if s.Abb != vorher {
		t.Errorf("Fenster geändert: Abbildung %v, Soll weiter %v", s.Abb, vorher)
	}
}

func TestSitzungNavigatorRuhtWaehrendZug(t *testing.T) {
	s := navigatorLauf(t)
	s.SetzeZoom(400)
	n, _ := s.Navigator(NavigatorSeite, NavigatorRand)
	s.Druecken(document.Pt(100, 100), TasteMitte) // Verschieben der Ansicht mit der mittleren Taste
	vorher := s.Abb
	s.NavigatorZu(n, document.Pt(728, 481))
	if s.Abb != vorher {
		t.Errorf("während des Zugs: Abbildung %v, Soll unverändert %v", s.Abb, vorher)
	}
}

func TestSitzungNavigatorZoomUmFenstermitte(t *testing.T) {
	s := navigatorLauf(t)
	s.SetzeZoom(400)
	n, _ := s.Navigator(NavigatorSeite, NavigatorRand)
	s.NavigatorZu(n, document.Pt(780, 520))
	vorher := s.Abb.ZuAnsicht(document.Pt(450, 300))
	s.NavigatorZoom(-1) // Rad nach oben: vergrößert
	if s.Abb.Zoom <= 4 {
		t.Errorf("Zoom %v, Soll größer als 4", s.Abb.Zoom)
	}
	if m := s.Abb.ZuAnsicht(document.Pt(450, 300)); !gleichf(m.X, vorher.X) || !gleichf(m.Y, vorher.Y) {
		t.Errorf("Fenstermitte zeigt %v, Soll weiter %v", m, vorher)
	}
	z := s.Abb.Zoom
	s.NavigatorZoom(2)
	if s.Abb.Zoom >= z {
		t.Errorf("Rad nach unten: Zoom %v, Soll kleiner als %v", s.Abb.Zoom, z)
	}
}
