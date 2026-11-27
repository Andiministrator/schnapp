// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"math"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// rad schickt ein Rad-Ereignis an p (Fensterpixel) zur Zeit t mit den Umschaltern m.
func (l *menueLauf) rad(p image.Point, d f32.Point, t time.Duration, m key.Modifiers) {
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y)),
		Scroll: d, Time: t, Modifiers: m})
	l.setle()
}

// TestRadRolltDieAnsicht prüft ADR-0040 Punkt 1: das Rad ohne Strg verschiebt die Ansicht um die gemeldete Strecke,
// senkrecht wie waagerecht, ohne zu zoomen.
func TestRadRolltDieAnsicht(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1000, 700))
	p := l.bildPt(image.Pt(10, 22))
	vorher := l.e.s.Abb
	l.rad(p, f32.Pt(0, 120), time.Second, 0)
	nach := l.e.s.Abb
	if nach.Zoom != vorher.Zoom {
		t.Fatalf("Rad ohne Strg zoomt: %v → %v", vorher.Zoom, nach.Zoom)
	}
	if d := nach.Versatz.Minus(vorher.Versatz); math.Abs(d.X) > 1e-9 || math.Abs(d.Y+120) > 1e-9 {
		t.Fatalf("Rad nach unten verschiebt um %v, Soll (0, -120)", d)
	}
	// Waagerecht (Neigerad, Touchpad seitwärts; Umschalt+Rad meldet Gio als ScrollX).
	vorher = nach
	l.rad(p, f32.Pt(-90, 0), 2*time.Second, key.ModShift)
	if d := l.e.s.Abb.Versatz.Minus(vorher.Versatz); math.Abs(d.X-90) > 1e-9 || math.Abs(d.Y) > 1e-9 {
		t.Fatalf("waagerechtes Rad verschiebt um %v, Soll (90, 0)", d)
	}
	if l.e.s.Abb.Zoom != vorher.Zoom {
		t.Fatal("Umschalt+Rad zoomt")
	}
}

// TestStrgRadZoomtZumZeiger prüft ADR-0040 Punkt 2: Strg+Rad zoomt, der Punkt unter dem Zeiger bleibt stehen.
func TestStrgRadZoomtZumZeiger(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1000, 700))
	p := l.bildPt(image.Pt(10, 22))
	// Zeigerereignisse kommen in Koordinaten der Leinwand an (Ursprung oben links an ihr), nicht des Fensters.
	f := l.e.leinwandFlaeche(l.gtx())
	fp := document.Pt(float64(p.X-f.Min.X), float64(p.Y-f.Min.Y))
	z0 := l.e.s.Abb.Zoom
	unter := l.e.s.Abb.ZuAnsicht(fp)
	// Erstes Rad-Ereignis überhaupt, mit kleiner Zeit (die Uhr des Compositors beginnt irgendwo): Strg zählt.
	l.rad(p, f32.Pt(0, -120), 10*time.Millisecond, key.ModShortcut)
	if l.e.s.Abb.Zoom <= z0 {
		t.Fatalf("Strg+Rad hoch vergrößert nicht: %v → %v", z0, l.e.s.Abb.Zoom)
	}
	if d := l.e.s.Abb.ZuAnsicht(fp).Minus(unter).Laenge(); d > 1e-6 {
		t.Fatalf("Punkt unter dem Zeiger wandert um %v", d)
	}
	// Strg+Umschalt+Rad zoomt ebenso (Gio meldet es senkrecht).
	z1 := l.e.s.Abb.Zoom
	l.rad(p, f32.Pt(0, 120), 2*time.Second, key.ModShortcut|key.ModShift)
	if l.e.s.Abb.Zoom >= z1 {
		t.Fatalf("Strg+Umschalt+Rad runter verkleinert nicht: %v → %v", z1, l.e.s.Abb.Zoom)
	}
}

// TestRadGesteBehaeltIhreArt prüft ADR-0040 Risiko 1: Nachlauf-Ereignisse (gleiche Zeit wie das letzte echte) und
// dicht folgende Ereignisse behalten die Art der Geste, auch wenn Strg mittendrin wechselt; erst eine Pause beginnt
// eine neue Geste.
func TestRadGesteBehaeltIhreArt(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1000, 700))
	p := l.bildPt(image.Pt(10, 22))
	l.rad(p, f32.Pt(0, 30), time.Second, 0)
	z := l.e.s.Abb.Zoom
	l.rad(p, f32.Pt(0, 20), time.Second, key.ModShortcut) // Nachlauf, Strg inzwischen gedrückt
	l.rad(p, f32.Pt(0, 10), time.Second+50*time.Millisecond, key.ModShortcut)
	if l.e.s.Abb.Zoom != z {
		t.Fatalf("Strg im Nachlauf macht aus Rollen Zoomen: %v → %v", z, l.e.s.Abb.Zoom)
	}
	// Neue Geste nach einer Pause: Strg zählt.
	l.rad(p, f32.Pt(0, -120), 2*time.Second, key.ModShortcut)
	if l.e.s.Abb.Zoom <= z {
		t.Fatalf("neue Geste mit Strg zoomt nicht: %v → %v", z, l.e.s.Abb.Zoom)
	}
	// Umgekehrt: Strg im Nachlauf losgelassen, die Geste zoomt weiter und rollt nicht.
	z = l.e.s.Abb.Zoom
	l.rad(p, f32.Pt(0, -60), 2*time.Second, 0)
	if l.e.s.Abb.Zoom <= z {
		t.Fatal("Nachlauf nach losgelassenem Strg zoomt nicht weiter")
	}
	// Ein Ereignis mit kleinerer Zeit (Zeitquelle springt) beginnt eine neue Geste.
	z = l.e.s.Abb.Zoom
	l.rad(p, f32.Pt(0, 40), time.Second, 0)
	if l.e.s.Abb.Zoom != z {
		t.Fatal("Ereignis mit früherer Zeit gehört noch zur zoomenden Geste")
	}
}

// TestRadRolltMitDerSkala prüft ADR-0040 Risiko 2: Gio meldet Scroll in logischen Pixeln, die Leinwand rechnet in
// Fensterpixeln — bei Skala 2 rollt die Ansicht doppelt so weit.
func TestRadRolltMitDerSkala(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1000, 700))
	l.skala = 2
	l.setle()
	p := l.bildPt(image.Pt(10, 22))
	vorher := l.e.s.Abb.Versatz
	l.rad(p, f32.Pt(15, 40), time.Second, 0)
	if d := l.e.s.Abb.Versatz.Minus(vorher); math.Abs(d.X+30) > 1e-9 || math.Abs(d.Y+80) > 1e-9 {
		t.Fatalf("bei Skala 2 verschiebt das Rad um %v, Soll (-30, -80)", d)
	}
}

// TestRadImZugTutNichts prüft ADR-0040 Punkt 1: während eines Zugs rollt und zoomt das Rad nicht.
func TestRadImZugTutNichts(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1000, 700))
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	a := l.bildPt(image.Pt(22, 18))
	l.maus(pointer.Move, a, 0)
	l.setle()
	l.maus(pointer.Press, a, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Move, a.Add(image.Pt(5, 5)), pointer.ButtonPrimary)
	l.setle()
	if !l.e.s.Zieht() {
		t.Fatal("Kontrolle: kein Zug")
	}
	abb := l.e.s.Abb
	l.rad(a, f32.Pt(0, 120), time.Second, 0)
	l.rad(a, f32.Pt(0, -120), 2*time.Second, key.ModShortcut)
	if l.e.s.Abb != abb {
		t.Fatalf("Rad im Zug ändert die Ansicht: %+v → %+v", abb, l.e.s.Abb)
	}
}
