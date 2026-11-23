// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// lupenEditor: Bild 40×30 mit eigener Farbe je Pixel und eine Lupe Quelle (2,3)–(6,6), Ziel (20,10)–(28,16) (2×).
func lupenEditor(t *testing.T, strich document.Strich) (*editor, document.Lupe) {
	t.Helper()
	i18n.Setze(nil)
	b := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			b.SetNRGBA(x, y, color.NRGBA{uint8(x * 6), uint8(y * 8), uint8(200 - x - y), 255})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	l := document.Lupe{Quelle: document.R(document.Pt(2, 3), document.Pt(6, 6)), Ziel: document.R(document.Pt(20, 10), document.Pt(28, 16)),
		Strich: strich}
	id, err := d.Hinzufuegen(l)
	if err != nil {
		t.Fatal(err)
	}
	l.ID = id
	return &editor{s: canvas.NeueSitzung(d)}, l
}

func nahFarbe(a color.RGBA, b color.NRGBA) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= 1 && int(y)-int(x) <= 1 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

// Der Editor zeichnet das Bild im Ziel mit denselben Pixeln wie der Renderer (nächster Nachbar), außerhalb nichts;
// eine Schwärzung in der Quelle zeigt er im nächsten Bild (ADR-0033 Punkte 2 und 3).
func TestEditorZeichnetLupe(t *testing.T) {
	e, l := lupenEditor(t, document.Strich{Farbe: color.NRGBA{G: 255, A: 255}})
	zeichne := func() *image.RGBA {
		return headlessBild(t, image.Pt(64, 48), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	}
	b := zeichne()
	soll := render.LupenInhalt(e.s.D, l)
	for y := 10; y < 16; y++ {
		for x := 20; x < 28; x++ {
			if p := b.RGBAAt(x, y); !nahFarbe(p, soll.NRGBAAt(x, y)) {
				t.Errorf("Ziel (%d,%d): Editor %v, Renderer %v", x, y, p, soll.NRGBAAt(x, y))
			}
		}
	}
	if p := b.RGBAAt(34, 24); p.A != 0 {
		t.Errorf("neben der Lupe gezeichnet: %v", p)
	}
	if err := e.s.D.Schwaerzen(image.Rect(2, 3, 4, 5), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	b = zeichne()
	if p := b.RGBAAt(21, 11); !nahFarbe(p, color.NRGBA{A: 255}) {
		t.Errorf("geschwärzte Quelle im Ziel: %v, Soll schwarz", p)
	}
	if p := b.RGBAAt(25, 11); nahFarbe(p, color.NRGBA{A: 255}) {
		t.Errorf("ungeschwärzter Teil schwarz: %v", p)
	}
}

// Rahmen und Linie in der Strichfarbe; mit Strichbreite 0 nur das Bild.
func TestEditorZeichnetLupenRahmen(t *testing.T) {
	gruen := color.NRGBA{G: 255, A: 255}
	e, _ := lupenEditor(t, document.Strich{Farbe: gruen, Breite: 2})
	b := headlessBild(t, image.Pt(64, 48), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	// Rahmen mittig auf der Kante: x 19..21 links am Ziel, x 1..3 links an der Quelle; Linie zwischen den Rechtecken.
	for _, p := range []image.Point{{19, 13}, {27, 9}, {1, 4}, {5, 6}, {13, 8}} {
		if c := b.RGBAAt(p.X, p.Y); !nahFarbe(c, gruen) {
			t.Errorf("%v: %v, Soll Strichfarbe", p, c)
		}
	}
	if n := zaehleFarbe(b, gruen); n < 40 {
		t.Errorf("nur %d Pixel in der Strichfarbe", n)
	}
	e2, _ := lupenEditor(t, document.Strich{Farbe: gruen})
	b2 := headlessBild(t, image.Pt(64, 48), func(gtx layout.Context) { e2.zeichneObjekte(gtx) })
	if n := zaehleFarbe(b2, gruen); n != 0 {
		t.Errorf("ohne Strich %d Pixel in der Strichfarbe", n)
	}
}

// Die Vorschau der Ebenenliste zeigt Rahmen und Bild der Lupe (über zeichneObjekt mit vorschau).
func TestEbenenVorschauLupe(t *testing.T) {
	gruen := color.NRGBA{G: 255, A: 255}
	e, l := lupenEditor(t, document.Strich{Farbe: gruen, Breite: 1})
	b := headlessBild(t, image.Pt(80, 60), func(gtx layout.Context) {
		e.zeichneVorschau(gtx, l, image.Rect(0, 0, 80, 60))
	})
	if n := zaehleFarbe(b, gruen); n == 0 {
		t.Error("Vorschau ohne Rahmen")
	}
	// Bild im Ziel: Grenzen 1,5..28,5 × 2,5..16,5 (Strich 1,5 in der Vorschau), Faktor min(74/27, 54/14, 4) ≈ 2,74.
	// Die Mitte des Ziels (24,13) liegt bei etwa (3 + (24 − 1,5)·2,74, 3 + (54 − 14·2,74)/2 + (13 − 2,5)·2,74).
	f := 74 / 27.0
	x := int(3 + (24-1.5)*f)
	y := int(3 + (54-14*f)/2 + (13-2.5)*f)
	want := render.LupenVorschau(e.s.D, l, lupenVorschauSeite)
	q := want.NRGBAAt(want.Rect.Dx()/2, want.Rect.Dy()/2)
	if p := b.RGBAAt(x, y); p.A != 255 || (p.R == 255 && p.G == 255 && p.B == 255) {
		t.Errorf("Vorschau bei (%d,%d): %v, Soll ein Pixel der Quelle wie %v", x, y, p, q)
	}
}

// Vergrößerung als Text der Leiste.
func TestVergroesserungText(t *testing.T) {
	i18n.Setze(nil)
	for k, want := range map[float64]string{2: "2×", 3: "3×", 2.5: "2.5×", 2.04: "2×", 3.96: "4×"} {
		if got := vergroesserungText(k); got != want {
			t.Errorf("%v: %q, erwartet %q", k, got, want)
		}
	}
}

// Taste M wählt das Werkzeug Lupe (Menü Werkzeuge, Knopf, Hilfe); bei gewählter Lupe zeigt die Leiste „Vergrößerung“,
// + geht auf 3× (ein Rückgängig-Schritt), ohne Lupe fehlt die Zeile (ADR-0033 Punkt 8).
func TestLupeTasteUndLeiste(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	if werkzeugKuerzel(canvas.WerkzeugLupe) != "M" || l.e.werkzeugName(canvas.WerkzeugLupe) != "Magnifier" {
		t.Fatalf("Kürzel %q, Name %q", werkzeugKuerzel(canvas.WerkzeugLupe), l.e.werkzeugName(canvas.WerkzeugLupe))
	}
	gefunden := false
	for _, z := range l.e.hilfeZeilen() {
		gefunden = gefunden || (z.text == "Magnifier" && z.taste == "M")
	}
	if !gefunden {
		t.Error("Lupe fehlt in der Übersicht der Tastenkürzel")
	}
	if !l.e.lupe.rahmen[1].Empty() {
		t.Fatal("Vergrößerung beim gewählten Text sichtbar")
	}
	l.e.s.Auswahl = 0
	l.taste("M", 0)
	if l.e.s.Werkzeug != canvas.WerkzeugLupe {
		t.Fatalf("Werkzeug nach M: %v", l.e.s.Werkzeug)
	}
	if !l.e.lupe.rahmen[1].Empty() || l.e.seite.strRahmen[1].Empty() {
		t.Error("Werkzeug Lupe ohne Auswahl: Strichstärke ja, Vergrößerung nein")
	}
	id, err := l.e.s.D.Hinzufuegen(document.Lupe{Quelle: document.R(document.Pt(1, 1), document.Pt(5, 4)),
		Ziel: document.R(document.Pt(20, 10), document.Pt(28, 16)), Strich: document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	plus := l.e.lupe.rahmen[1]
	if plus.Empty() || l.e.lupe.text != "2×" {
		t.Fatalf("Vergrößerung %v %q", plus, l.e.lupe.text)
	}
	vor := l.e.s.D.Stand()
	l.klick(plus.Min.Add(plus.Max).Div(2))
	o, _ := l.e.s.D.Objekt(id)
	if k, _ := o.(document.Lupe).Vergroesserung(); k != 3 || l.e.s.D.Stand() == vor {
		t.Errorf("+ an der Lupe: %v", k)
	}
	l.setle()
	if l.e.lupe.text != "3×" {
		t.Errorf("angezeigt %q", l.e.lupe.text)
	}
}
