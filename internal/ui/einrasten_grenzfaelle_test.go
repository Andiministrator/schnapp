// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// Kritik zu Karte #196 (ADR-0021 Nachtrag 12 Punkt 4): Die Alt-Taste wirkt als Tastenereignis sofort auf den laufenden
// Zug, ohne dass sich der Zeiger bewegt; losgelassen rastet er wieder ein.
func TestAltTasteWaehrendDesZugsOhneZeigerbewegung(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 1200))
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(24, 24)),
		Fuellung: color.NRGBA{R: 200, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	o := l.e.leinwandFlaeche(l.gtx()).Min
	punkt := func(x, y float64) f32.Point {
		p := l.e.s.Abb.ZuFenster(document.Pt(x, y)).Plus(document.Pt(float64(o.X), float64(o.Y)))
		return f32.Pt(float32(p.X), float32(p.Y))
	}
	rechts := func() float64 {
		ob, _ := l.e.s.D.Objekt(id)
		return ob.(document.Kasten).Bereich.Max.X
	}
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: punkt(14, 14)})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: punkt(14, 14)})
	l.setle()
	// Rechte Kante von 24 auf 39,5: Der Kasten rastet ein (am Testtext des Laufs oder am Bildrand), steht also nicht
	// bei 39,5.
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: punkt(29.5, 14)})
	l.setle()
	fest := rechts()
	if len(l.e.s.Hilfslinien()) == 0 || math.Abs(fest-39.5) < 0.2 {
		t.Fatalf("vor Alt: Linien %v, rechte Kante %v, Soll eingerastet (nicht 39,5)", l.e.s.Hilfslinien(), fest)
	}
	l.r.Queue(key.Event{Name: key.NameAlt, State: key.Press}) // der Druck selbst trägt ModAlt noch nicht
	l.setle()
	if len(l.e.s.Hilfslinien()) != 0 || math.Abs(rechts()-39.5) > 0.2 || !l.e.s.AltGedrueckt() {
		t.Errorf("Alt gedrückt: Linien %v, rechte Kante %v, Alt %v, Soll frei bei 39,5", l.e.s.Hilfslinien(), rechts(),
			l.e.s.AltGedrueckt())
	}
	l.r.Queue(key.Event{Name: key.NameAlt, State: key.Release, Modifiers: key.ModAlt})
	l.setle()
	if len(l.e.s.Hilfslinien()) == 0 || math.Abs(rechts()-fest) > 1e-6 || l.e.s.AltGedrueckt() {
		t.Errorf("Alt losgelassen: Linien %v, rechte Kante %v, Alt %v, Soll wieder eingerastet", l.e.s.Hilfslinien(),
			rechts(), l.e.s.AltGedrueckt())
	}
	// Auch mit Umschalt dazu kommt die Taste an (der Filter erlaubt die Umschalter).
	l.r.Queue(key.Event{Name: key.NameAlt, State: key.Press, Modifiers: key.ModShift})
	l.setle()
	if !l.e.s.AltGedrueckt() {
		t.Error("Alt mit Umschalt kommt nicht an")
	}
	l.r.Queue(key.Event{Name: key.NameAlt, State: key.Release, Modifiers: key.ModAlt | key.ModShift})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: punkt(29.5, 14)})
	l.setle()
	if l.e.s.Zieht() || math.Abs(rechts()-fest) > 1e-6 {
		t.Errorf("nach dem Loslassen: zieht %v, rechte Kante %v", l.e.s.Zieht(), rechts())
	}
}

// Die Hilfslinie steht im Fenster dort, wo die Linie des Originals nach Zuschnitt und Stauchen zu sehen ist, und reicht
// über das sichtbare Bild.
func TestHilfslinienFlaechen(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(20, 10, 200, 100)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: false, Von: 60, Bis: 80}}); err != nil {
		t.Fatal(err)
	}
	id, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(30, 20), document.Pt(50, 40)),
		Fuellung: color.NRGBA{R: 200, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	s := canvas.NeueSitzung(d)
	s.SetzeFenster(image.Pt(208, 138)) // Ansicht 160×90
	s.SetzeZoom(100)
	if hilfslinienFlaechen(s, 1) != nil {
		t.Fatal("ohne Zug gibt es keine Flächen")
	}
	fenster := func(x, y float64) document.Punkt { return s.Abb.ZuFenster(d.ZuAnsicht(document.Pt(x, y))) }
	rund := func(v float64) int { return int(math.Round(v)) }
	a, b := s.Abb.ZuFenster(document.Pt(0, 0)), s.Abb.ZuFenster(document.Pt(160, 90))
	// Rechte Kante von 50 auf 199 (über die Naht): rastet am Bildrand 200 ein, der in der Ansicht bei 160 liegt.
	s.Druecken(fenster(40, 30), canvas.TastePrimaer)
	s.Ziehen(fenster(100, 30))
	s.Ziehen(fenster(189, 30))
	if ob, _ := d.Objekt(id); math.Abs(ob.(document.Kasten).Bereich.Max.X-200) > 1e-6 {
		t.Fatalf("Kasten %v, Soll rechte Kante 200", ob)
	}
	soll := image.Rect(rund(b.X), rund(a.Y), rund(b.X)+2, rund(b.Y))
	if fl := hilfslinienFlaechen(s, 2); len(fl) != 1 || fl[0] != soll {
		t.Errorf("senkrechte Linie: %v, Soll %v", fl, soll)
	}
	s.Abbrechen()
	// Obere Kante von 20 auf 12: rastet am oberen Rand des Zuschnitts (10) ein, in der Ansicht bei 0.
	s.Druecken(fenster(40, 30), canvas.TastePrimaer)
	s.Ziehen(fenster(40, 26))
	s.Ziehen(fenster(40, 22))
	soll = image.Rect(rund(a.X), rund(a.Y), rund(b.X), rund(a.Y)+1)
	if fl := hilfslinienFlaechen(s, 1); len(fl) != 1 || fl[0] != soll {
		t.Errorf("waagrechte Linie: %v, Soll %v", fl, soll)
	}
	s.Abbrechen()
}
