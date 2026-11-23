// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Navigator im Editor (ADR-0031 Nachtrag 9, Karte #197). Bild 1200×800 in blau, links oben ein roter Kasten
// (0,0)–(600,400); das Fenster ist so groß, dass das Bild bei 400 % in beiden Achsen hinausragt.

var (
	navBlau = color.NRGBA{B: 200, A: 255}
	navRot  = color.NRGBA{R: 220, A: 255}
)

func navigatorEditorLauf(t *testing.T) *menueLauf {
	t.Helper()
	i18n.Setze(nil)
	img := image.NewNRGBA(image.Rect(0, 0, 1200, 800))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+2], img.Pix[i+3] = navBlau.B, navBlau.A
	}
	d, err := document.Neu(img)
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	e.thema = material.NewTheme()
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(600, 400)),
		Fuellung: navRot}); err != nil {
		t.Fatal(err)
	}
	e.bildNeu()
	l := &menueLauf{t: t, e: e, r: new(input.Router), ops: new(op.Ops), fenst: image.Pt(1200, 800)}
	l.setle()
	return l
}

// navKasten ist der Kasten in Leinwandpixeln und der Ursprung der Leinwand im Fenster.
func (l *menueLauf) navKasten(t *testing.T) (canvas.Navigator, image.Point) {
	t.Helper()
	n, ok := l.e.s.Navigator(canvas.NavigatorSeite, canvas.NavigatorRand)
	if !ok {
		t.Fatalf("kein Navigator bei Zoom %v im Fenster %v", l.e.s.Abb.Zoom, l.e.s.Fenster())
	}
	return n, l.e.leinwandFlaeche(l.gtx()).Min
}

func fensterPt(o image.Point, p document.Punkt) image.Point {
	return image.Pt(o.X+int(p.X+0.5), o.Y+int(p.Y+0.5))
}

func TestNavigatorErscheintNurBeiZoomUndSchalter(t *testing.T) {
	l := navigatorEditorLauf(t)
	// Eingepasst: das ganze Bild liegt im Fenster, kein Kasten.
	if !l.e.nav.rahmen.Empty() {
		t.Fatalf("eingepasst: Kasten %v, Soll keiner", l.e.nav.rahmen)
	}
	l.e.s.SetzeZoom(400)
	l.setle()
	n, _ := l.navKasten(t)
	if l.e.nav.rahmen != n.Flaeche || n.Flaeche.Empty() {
		t.Errorf("bei 400 %%: gezeichneter Kasten %v, Soll %v", l.e.nav.rahmen, n.Flaeche)
	}
	// Menüschalter Ansicht → Navigator.
	l.e.fuehreAus(layout.Context{}, aktNavigator, true)
	l.setle()
	if !l.e.nav.rahmen.Empty() {
		t.Errorf("Schalter aus: Kasten %v, Soll keiner", l.e.nav.rahmen)
	}
	l.e.fuehreAus(layout.Context{}, aktNavigator, true)
	l.setle()
	if l.e.nav.rahmen.Empty() {
		t.Error("Schalter wieder an: Soll ein Kasten da sein")
	}
	l.e.s.SetzeZoom(50)
	l.setle()
	if !l.e.nav.rahmen.Empty() {
		t.Errorf("bei 50 %%: Kasten %v, Soll keiner", l.e.nav.rahmen)
	}
}

func TestMenueAnsichtHatNavigator(t *testing.T) {
	e := testEditor(t)
	for _, en := range e.menueEintraegeVon(menueAnsicht) {
		if en.akt == aktNavigator {
			if !en.aktiv || !en.an || en.text != "Navigator" {
				t.Errorf("Eintrag %+v, Soll an und angehakt, Text Navigator", en)
			}
			e.fuehreAus(layout.Context{}, aktNavigator, true)
			for _, en := range e.menueEintraegeVon(menueAnsicht) {
				if en.akt == aktNavigator && en.aktiv {
					t.Error("nach dem Schalten: Soll nicht mehr angehakt")
				}
			}
			return
		}
	}
	t.Fatal("kein Eintrag Navigator im Menü Ansicht")
}

func TestHilfeNenntNavigator(t *testing.T) {
	e := testEditor(t)
	for _, z := range e.hilfeZeilen() {
		if z.text == "Move the view in the navigator" && z.taste == "Click, drag" {
			return
		}
	}
	t.Error("Hilfe-Übersicht ohne Zeile zum Navigator")
}

// Klick zentriert, Ziehen führt den Ausschnitt, und nichts davon erreicht die Werkzeuge der Leinwand.
func TestNavigatorKlickUndZiehenErreichenDieWerkzeugeNicht(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.Werkzeug = canvas.WerkzeugKasten
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	vorher := len(l.e.s.D.Objekte())

	// Gegenprobe: Ziehen auf der Leinwand zeichnet einen Kasten (der Aufbau erreicht die Werkzeuge).
	a := image.Pt(o.X+100, o.Y+100)
	l.maus(pointer.Move, a, 0)
	l.setle()
	l.maus(pointer.Press, a, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Move, a.Add(image.Pt(80, 60)), pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, a.Add(image.Pt(80, 60)), 0)
	l.setle()
	if got := len(l.e.s.D.Objekte()); got != vorher+1 {
		t.Fatalf("Gegenprobe Leinwand: %d Objekte, Soll %d", got, vorher+1)
	}
	l.e.hatDruck = false
	vorher++

	// Klick neben die Bildmitte (Ansicht etwa (900, 400)): Diese Stelle liegt danach in der Fenstermitte der Leinwand.
	mitte := fensterPt(o, n.ZuKasten(document.Pt(900, 400)))
	soll := n.ZuAnsicht(document.Pt(float64(mitte.X-o.X), float64(mitte.Y-o.Y)))
	l.maus(pointer.Move, mitte, 0)
	l.setle()
	l.maus(pointer.Press, mitte, pointer.ButtonPrimary)
	l.setle()
	f := l.e.s.Fenster()
	m := l.e.s.Abb.ZuAnsicht(document.Pt(float64(f.X)/2, float64(f.Y)/2))
	if d := m.Minus(soll).Laenge(); d > 1e-6 || soll.X < 850 || soll.X > 950 {
		t.Errorf("nach dem Klick zeigt die Fenstermitte %v, Soll %v (um 900, 400)", m, soll)
	}
	// Ziehen in die linke obere Kastenecke: Der Ausschnitt rückt an die Bildecke.
	ecke := fensterPt(o, document.Pt(float64(n.Flaeche.Min.X)+1, float64(n.Flaeche.Min.Y)+1))
	l.maus(pointer.Move, ecke, pointer.ButtonPrimary)
	l.setle()
	if v := l.e.s.Abb.Versatz; v.X != 0 || v.Y != 0 {
		t.Errorf("Ziehen in die Ecke: Versatz %v, Soll (0, 0)", v)
	}
	// Ziehen weit aus dem Kasten heraus: Der Ausschnitt bleibt im Bild, unten rechts.
	weit := image.Pt(o.X+f.X-1, o.Y+f.Y-1).Sub(image.Pt(0, 0))
	l.maus(pointer.Move, weit, pointer.ButtonPrimary)
	l.setle()
	if v := l.e.s.Abb.Versatz; v.X != float64(f.X)-4800 || v.Y != float64(f.Y)-3200 {
		t.Errorf("Ziehen aus dem Kasten: Versatz %v, Soll (%v, %v)", v, float64(f.X)-4800, float64(f.Y)-3200)
	}
	l.maus(pointer.Release, weit, 0)
	l.setle()
	if got := len(l.e.s.D.Objekte()); got != vorher {
		t.Errorf("Klick und Ziehen im Navigator: %d Objekte, Soll %d (kein Zeichnen)", got, vorher)
	}
	if l.e.hatDruck || l.e.s.Zieht() {
		t.Errorf("Druck erreichte die Leinwand: hatDruck %v, Zug %v", l.e.hatDruck, l.e.s.Zieht())
	}
	if l.e.nav.zieht {
		t.Error("nach dem Loslassen: Soll nicht mehr ziehen")
	}
	// Nach dem Loslassen führt eine Bewegung ohne Taste den Ausschnitt nicht weiter.
	fest := l.e.s.Abb
	l.maus(pointer.Move, mitte, 0)
	l.setle()
	if l.e.s.Abb != fest {
		t.Errorf("Bewegung ohne Taste verschob die Ansicht: %v, Soll %v", l.e.s.Abb, fest)
	}
}

func TestNavigatorRadZoomtUmFenstermitte(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	f := l.e.s.Fenster()
	vorher := l.e.s.Abb.ZuAnsicht(document.Pt(float64(f.X)/2, float64(f.Y)/2))
	p := fensterPt(o, n.ZuKasten(document.Pt(1100, 700)))
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y)),
		Scroll: f32.Pt(0, -40)})
	l.setle()
	if z := l.e.s.Abb.Zoom; z < 4.39 || z > 4.41 {
		t.Errorf("Zoom nach einem Rastschritt %v, Soll 4,4", z)
	}
	if m := l.e.s.Abb.ZuAnsicht(document.Pt(float64(f.X)/2, float64(f.Y)/2)); m.X < vorher.X-0.01 || m.X > vorher.X+0.01 ||
		m.Y < vorher.Y-0.01 || m.Y > vorher.Y+0.01 {
		t.Errorf("Fenstermitte zeigt nach dem Rad %v, Soll weiter %v", m, vorher)
	}
	if l.e.hatDruck {
		t.Error("das Rad erreichte die Leinwand")
	}
}

// Zeichnen ohne Fenster: Rand, Bild, Objekt und Ausschnitt-Rechteck im Kasten (Gio headless).
func TestNavigatorZeichnet(t *testing.T) {
	l := navigatorEditorLauf(t)
	f := l.e.s.Fenster()
	zeichne := func() *image.RGBA {
		return headlessBild(t, f, func(gtx layout.Context) { l.e.zeichneNavigator(gtx) })
	}
	// Eingepasst: nichts gezeichnet.
	if b := zeichne(); zaehleFarbe(b, farbeAkzent) != 0 || zaehleFarbe(b, farbeLinie) != 0 || zaehleFarbe(b, navBlau) != 0 {
		t.Error("eingepasst: Soll nichts gezeichnet sein")
	}
	l.e.s.SetzeZoom(400)
	n, _ := l.navKasten(t)
	l.e.s.NavigatorZu(n, document.Pt(float64(n.Flaeche.Min.X+n.Flaeche.Max.X)/2, float64(n.Flaeche.Min.Y+n.Flaeche.Max.Y)/2))
	b := zeichne()
	r := n.Flaeche
	px := func(x, y int) color.RGBA { return b.RGBAAt(x, y) }
	if p := px(r.Min.X, r.Min.Y); p != (color.RGBA{R: farbeLinie.R, G: farbeLinie.G, B: farbeLinie.B, A: 255}) {
		t.Errorf("Ecke des Kastens %v, Soll der Rand in farbeLinie", p)
	}
	if p := px(r.Min.X-1, r.Min.Y+5); p.A != 0 {
		t.Errorf("ein Pixel links vom Kasten %v, Soll durchsichtig", p)
	}
	if p := px(r.Min.X+3, r.Min.Y+3); p.R != navRot.R || p.B != 0 {
		t.Errorf("Pixel im roten Kasten %v, Soll Rot", p)
	}
	if p := px(r.Max.X-4, r.Max.Y-4); p.B != navBlau.B || p.R != 0 {
		t.Errorf("Pixel im Bild unten rechts %v, Soll Blau", p)
	}
	// Ausschnitt-Rechteck: 2 px breiter Rahmen in farbeAkzent, ganz im Kasten.
	a := l.e.nav.ausschn
	if a.Dx() < 5 || a.Dy() < 5 || !a.In(r) {
		t.Fatalf("Ausschnitt %v im Kasten %v", a, r)
	}
	if want := a.Dx()*a.Dy() - (a.Dx()-4)*(a.Dy()-4); zaehleFarbe(b, farbeAkzent) != want {
		t.Errorf("Akzentpixel %d, Soll %d (Rahmen 2 px)", zaehleFarbe(b, farbeAkzent), want)
	}
	if p := px(a.Min.X, (a.Min.Y+a.Max.Y)/2); p.R != farbeAkzent.R || p.G != farbeAkzent.G {
		t.Errorf("linke Kante des Ausschnitts %v, Soll orange", p)
	}
	if p := px((a.Min.X+a.Max.X)/2, (a.Min.Y+a.Max.Y)/2); p.R == farbeAkzent.R && p.G == farbeAkzent.G {
		t.Errorf("Mitte des Ausschnitts %v, Soll nicht orange (nur der Rahmen)", p)
	}
	// Der Ausschnitt folgt der Ansicht: in die Ecke geschoben, steht er oben links im Kasten.
	l.e.s.NavigatorZu(n, document.Pt(float64(r.Min.X), float64(r.Min.Y)))
	zeichne()
	if a := l.e.nav.ausschn; a.Min != r.Min {
		t.Errorf("Ausschnitt in der Ecke %v, Soll an %v beginnen", a, r.Min)
	}
}
