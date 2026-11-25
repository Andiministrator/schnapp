// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #323: Die Zeigerform der Sitzung wird auf den passenden Zeiger des Systems abgebildet, und ein Wechsel der Form
// beim Bewegen fordert ein Folgebild an, auch wenn das Pixel unter dem Zeiger gleich bleibt.

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func TestSystemZeiger(t *testing.T) {
	for z, want := range map[canvas.ZeigerArt]pointer.Cursor{
		canvas.ZeigerStandard: pointer.CursorDefault, canvas.ZeigerFadenkreuz: pointer.CursorCrosshair,
		canvas.ZeigerText: pointer.CursorText, canvas.ZeigerVerschieben: pointer.CursorAllScroll,
		canvas.ZeigerGreifen: pointer.CursorGrab, canvas.ZeigerGreifend: pointer.CursorGrabbing,
		canvas.ZeigerNWSO: pointer.CursorNorthWestSouthEastResize, canvas.ZeigerNOSW: pointer.CursorNorthEastSouthWestResize,
		canvas.ZeigerNS: pointer.CursorNorthSouthResize, canvas.ZeigerOW: pointer.CursorEastWestResize,
	} {
		if got := systemZeiger(z); got != want {
			t.Errorf("%v: %v, Soll %v", z, got, want)
		}
	}
}

func TestZeigerFormWechselFordertBild(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	e := l.e
	// Der Testtext ist gewählt; über ihm steht der Verschieben-Zeiger, daneben der Standard.
	o, _ := e.s.D.Objekt(l.id)
	g := o.Grenzen()
	mitteFenster := e.s.Abb.ZuFenster(g.Min.Plus(g.Max).Mal(0.5))
	e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	weit := mitteFenster.Plus(document.Pt(300, 200))
	e.merkeZeiger(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(weit.X), float32(weit.Y))})
	if z := e.s.Zeiger(e.zeigerFenster); z != canvas.ZeigerStandard {
		t.Fatalf("Vorbedingung: neben dem Text %v", z)
	}
	if !e.merkeZeiger(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(mitteFenster.X), float32(mitteFenster.Y))}) {
		t.Error("Wechsel auf den Verschieben-Zeiger fordert kein Folgebild")
	}
	if z := e.s.Zeiger(e.zeigerFenster); z != canvas.ZeigerVerschieben {
		t.Errorf("über dem Text: %v", z)
	}
}

// Bei starkem Zoom bleibt das Pixel unter dem Zeiger gleich, während er den Griff erreicht: Auch dann braucht es ein
// Folgebild, sonst stünde der alte Zeiger da (Mutant ZF9).
func TestZeigerFormWechselOhnePixelwechsel(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	e := l.e
	k, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(20, 20), document.Pt(40, 40)),
		Strich: document.Strich{Farbe: canvas.Beschriftungsfarben[0], Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	e.s.Auswahl = k
	e.s.SetzeZoom(1600)
	ecke := e.s.Abb.ZuFenster(document.Pt(20, 20))
	// Innen, im selben Bildpixel wie die Ecke, aber außerhalb der Griff-Toleranz von 6 Fensterpixeln.
	innen := ecke.Plus(document.Pt(canvas.Toleranz+3, canvas.Toleranz+3))
	bewege := func(p document.Punkt) bool {
		return e.merkeZeiger(pointer.Event{Kind: pointer.Move, Position: f32.Pt(float32(p.X), float32(p.Y))})
	}
	bewege(innen)
	if z := e.s.Zeiger(e.zeigerFenster); z == canvas.ZeigerNWSO {
		t.Fatalf("Vorbedingung: innen schon der Eckzeiger")
	}
	ortInnen, _ := e.s.ZeigerOrt(*e.zeigerAnsicht())
	if !bewege(ecke.Plus(document.Pt(1, 1))) {
		t.Error("Wechsel auf den Eckzeiger im selben Pixel fordert kein Folgebild")
	}
	if ort, _ := e.s.ZeigerOrt(*e.zeigerAnsicht()); ort != ortInnen {
		t.Fatalf("Vorbedingung: Pixel gewechselt (%v → %v)", ortInnen, ort)
	}
	if z := e.s.Zeiger(e.zeigerFenster); z != canvas.ZeigerNWSO {
		t.Errorf("an der Ecke: %v", z)
	}
}
