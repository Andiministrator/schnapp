// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Navigator: flache und schmale Bilder, Sperre beim Speichern, Fokus, Nähte, Zuschneiden (F-613, KS-3, KS-7, KS-8,
// Karte #236). Der Aufbau des Editors steht in navigator_test.go.

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// navigatorBildLauf ist ein Editor über einem einfarbig blauen Bild der Größe g im Fenster 1200×800 bei Zoom 300 %.
func navigatorBildLauf(t *testing.T, g image.Point, streifen []document.Streifen) *menueLauf {
	t.Helper()
	i18n.Setze(nil)
	img := image.NewNRGBA(image.Rectangle{Max: g})
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+2], img.Pix[i+3] = navBlau.B, navBlau.A
	}
	d, err := document.Neu(img)
	if err != nil {
		t.Fatal(err)
	}
	if streifen != nil {
		if err := d.SetzeStreifen(streifen); err != nil {
			t.Fatal(err)
		}
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	e.thema = material.NewTheme()
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	e.bildNeu()
	l := &menueLauf{t: t, e: e, r: new(input.Router), ops: new(op.Ops), fenst: image.Pt(1200, 800)}
	l.setle()
	l.e.s.SetzeZoom(300)
	l.setle()
	return l
}

// Sehr flaches und sehr schmales Bild: Kasten mit Mindestdicke, Ausschnitt füllt den Kasten in der dünnen Achse, und
// ein Klick in den Kasten außerhalb des Bildstreifens führt den Ausschnitt trotzdem.
func TestNavigatorFlachesBildZeigtKastenUndAusschnitt(t *testing.T) {
	for _, c := range []struct {
		name string
		g    image.Point
		dick func(r image.Rectangle) int // Maß in der dünnen Achse
		lang func(r image.Rectangle) int
	}{
		{"flach", image.Pt(3000, 200), func(r image.Rectangle) int { return r.Dy() }, func(r image.Rectangle) int { return r.Dx() }},
		{"schmal", image.Pt(200, 3000), func(r image.Rectangle) int { return r.Dx() }, func(r image.Rectangle) int { return r.Dy() }},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := navigatorBildLauf(t, c.g, nil)
			n, o := l.navKasten(t)
			f := l.e.s.Fenster()
			b := headlessBild(t, f, func(gtx layout.Context) { l.e.zeichneNavigator(gtx) })
			if d := c.dick(n.Flaeche); d != canvas.NavigatorMinSeite {
				t.Errorf("dünne Seite des Kastens %d, Soll %d", d, canvas.NavigatorMinSeite)
			}
			if d := c.lang(n.Flaeche); d != canvas.NavigatorSeite {
				t.Errorf("lange Seite des Kastens %d, Soll %d", d, canvas.NavigatorSeite)
			}
			a := l.e.nav.ausschn
			if c.dick(a) != canvas.NavigatorMinSeite {
				t.Errorf("Ausschnitt %v füllt den Kasten %v in der dünnen Achse nicht", a, n.Flaeche)
			}
			if c.lang(a) >= c.lang(n.Flaeche) || c.lang(a) < 1 {
				t.Errorf("Ausschnitt %v in der langen Achse sollte ein Teil des Kastens %v sein", a, n.Flaeche)
			}
			// Das Bild ist im Kasten zu sehen (blau), auch außerhalb des Ausschnitts, und der Rand des Kastens steht.
			if zaehleFarbe(b, navBlau) < 10 {
				t.Errorf("Bild im flachen Kasten nicht zu sehen: %d blaue Pixel", zaehleFarbe(b, navBlau))
			}
			if zaehleFarbe(b, farbeAkzent) < c.lang(a) {
				t.Errorf("Ausschnitt-Rechteck nicht zu sehen: %d Akzent-Pixel", zaehleFarbe(b, farbeAkzent))
			}
			// Klick in die obere linke Ecke des Kastens (im dünnen Rand, nicht auf dem Bildstreifen): Ansicht rückt
			// an die Bildecke.
			p := fensterPt(o, document.Pt(float64(n.Flaeche.Min.X)+1, float64(n.Flaeche.Min.Y)+1))
			l.maus(pointer.Move, p, 0)
			l.setle()
			l.maus(pointer.Press, p, pointer.ButtonPrimary)
			l.setle()
			l.maus(pointer.Release, p, 0)
			l.setle()
			// Die lange Achse rückt an den Bildrand; die dünne bleibt mittig (das Bild ist dort kleiner als das Fenster).
			if v := l.e.s.Abb.Versatz; (c.name == "flach" && v.X != 0) || (c.name == "schmal" && v.Y != 0) || (v.X == 0 && v.Y == 0) {
				t.Errorf("Klick in die Kastenecke: Versatz %v, Soll an der Bildkante der langen Achse", v)
			}
		})
	}
}

// KS-3/KS-8 (NU12): Während des Speicherns ruht der Navigator (Druck und Ziehen ändern die Ansicht nicht), das Loslassen
// gilt trotzdem.
func TestNavigatorRuhtWaehrendSpeichernLoslassenGilt(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	if _, err := l.e.s.SpeichernJetzt(); err != nil || !l.e.s.SpeichernLaeuft() {
		t.Fatalf("Kontrolle: Speichern beginnt nicht (%v)", err)
	}
	vor := l.e.s.Abb
	p := fensterPt(o, n.ZuKasten(document.Pt(900, 400)))
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Move, fensterPt(o, n.ZuKasten(document.Pt(300, 200))), pointer.ButtonPrimary)
	l.setle()
	if l.e.s.Abb != vor || l.e.nav.zieht {
		t.Errorf("Navigator arbeitet während des Speicherns: Ansicht %v (vorher %v), zieht %v", l.e.s.Abb, vor, l.e.nav.zieht)
	}
	l.maus(pointer.Release, p, 0)
	l.setle()
	l.e.s.SpeichernEnde(false, nil)
}

// KS-8 (KM8): Ein Abbruch des Zeigers (Cancel) beendet das Ziehen auch während des Speicherns.
func TestNavigatorAbbruchBeendetDasZiehen(t *testing.T) {
	for _, speichert := range []bool{false, true} {
		l := navigatorEditorLauf(t)
		l.e.s.SetzeZoom(400)
		l.setle()
		n, o := l.navKasten(t)
		p := fensterPt(o, n.ZuKasten(document.Pt(900, 400)))
		l.maus(pointer.Move, p, 0)
		l.setle()
		l.maus(pointer.Press, p, pointer.ButtonPrimary)
		l.setle()
		if !l.e.nav.zieht {
			t.Fatal("Kontrolle: Druck im Navigator zieht nicht")
		}
		if speichert {
			if _, err := l.e.s.SpeichernJetzt(); err != nil {
				t.Fatal(err)
			}
		}
		l.maus(pointer.Cancel, p, 0)
		l.setle()
		if l.e.nav.zieht {
			t.Errorf("nach Cancel (Speichern %v): nav.zieht steht noch", speichert)
		}
		if speichert {
			l.e.s.SpeichernEnde(false, nil)
		}
	}
}

// KS-8 (KM2): Ein Klick im Navigator gibt der Leinwand den Tastaturfokus, wie ein Klick auf die Leinwand.
func TestNavigatorKlickGibtDerLeinwandDenFokus(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	var anderer int
	l.fokusiere(&anderer)
	if l.fokus(l.e) {
		t.Fatal("Kontrolle: die Leinwand hat den Fokus schon")
	}
	l.klick(fensterPt(o, n.ZuKasten(document.Pt(300, 200))))
	if !l.fokus(l.e) {
		t.Error("nach dem Klick im Navigator hat die Leinwand den Fokus nicht")
	}
}

// KS-8 (NU14): Der Kasten ist 160 dp lang, bei Skala 2 also 320 px, und die Mindestdicke wächst mit (32 px).
func TestNavigatorFlachBeiSkala2(t *testing.T) {
	l := navigatorBildLauf(t, image.Pt(3000, 200), nil)
	f := l.e.s.Fenster()
	g := l.gtx()
	g.Metric.PxPerDp, g.Metric.PxPerSp = 2, 2
	g.Constraints = layout.Exact(f)
	l.e.zeichneNavigator(g)
	if r := l.e.nav.rahmen; r.Dx() != 320 || r.Dy() != 32 {
		t.Errorf("Skala 2: Kasten %v, Soll 320 × 32", r)
	}
}

// KS-8 (NU15): Die Nähte stehen in der Übersicht: Der durchsichtige Spalt zeigt das Schachbrett des Editors (graue Felder
// schachHell/schachDunkel). Ohne zeichneNaehte bliebe er der Grund des Kastens.
func TestNavigatorZeigtNaehte(t *testing.T) {
	schach := func(streifen []document.Streifen) int {
		l := navigatorBildLauf(t, image.Pt(400, 300), streifen)
		l.e.s.SetzeZoom(800)
		l.setle()
		n, _ := l.navKasten(t)
		b := headlessBild(t, l.e.s.Fenster(), func(gtx layout.Context) { l.e.zeichneNavigator(gtx) })
		nah := func(c color.RGBA, k color.NRGBA) bool {
			d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
			return d(c.R, k.R) <= 6 && d(c.G, k.G) <= 6 && d(c.B, k.B) <= 6
		}
		z := 0
		for y := n.Flaeche.Min.Y; y < n.Flaeche.Max.Y; y++ {
			for x := n.Flaeche.Min.X; x < n.Flaeche.Max.X; x++ {
				if c := b.RGBAAt(x, y); nah(c, schachHell) || nah(c, schachDunkel) {
					z++
				}
			}
		}
		return z
	}
	ohne := schach(nil)
	mit := schach([]document.Streifen{{Zeilen: true, Von: 100, Bis: 130}, {Zeilen: false, Von: 200, Bis: 230}})
	t.Logf("Schachbrett-Pixel mit Nähten %d, ohne %d", mit, ohne)
	if ohne != 0 || mit < 20 {
		t.Errorf("Schachbrett-Pixel in der Übersicht: mit Nähten %d (Soll mindestens 20), ohne %d (Soll 0): die Nähte fehlen", mit, ohne)
	}
}

// KS-8 (NU16): Beim Zuschneiden liegt das Bild in der Vollansicht bei bildOrt; die Übersicht zeichnet es dort, nicht am
// Ursprung: Die obere linke Ecke des Kastens zeigt Bild, nicht den Grund.
func TestNavigatorZeichnetDieVollansichtAnIhrerStelle(t *testing.T) {
	l := navigatorEditorLauf(t)
	if err := l.e.s.D.SetzeZuschnitt(image.Rect(300, 200, 900, 600)); err != nil {
		t.Fatal(err)
	}
	l.e.bildNeu()
	l.e.s.SetzeWerkzeug(canvas.WerkzeugZuschneiden)
	l.setle()
	l.e.s.SetzeZoom(400)
	l.setle()
	n, _ := l.navKasten(t)
	b := headlessBild(t, l.e.s.Fenster(), func(gtx layout.Context) { l.e.zeichneNavigator(gtx) })
	for _, p := range []image.Point{n.Flaeche.Min.Add(image.Pt(3, 3)), n.Flaeche.Max.Sub(image.Pt(4, 4))} {
		if c := b.RGBAAt(p.X, p.Y); c.A == 0 || (c.R == farbeLeisteVoll.R && c.G == farbeLeisteVoll.G && c.B == farbeLeisteVoll.B) {
			t.Errorf("Pixel %v des Kastens %v zeigt den Grund %v statt des Bilds", p, n.Flaeche, c)
		}
	}
}
