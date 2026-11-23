// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/Andiministrator/schnapp/internal/document"
)

// headlessBild lässt zeichne auf eine Fläche ohne Fenster zeichnen (Gio headless, EGL) und liefert die Pixel. Gibt es
// auf dem Rechner kein EGL, wird der Test übersprungen (F-590, F-593: das Zeichnen des Editors war ungetestet).
//
// EGL_PLATFORM=surfaceless (Mesa), wenn nichts anderes verlangt ist: Ohne die Angabe nimmt EGL die Anzeige der Sitzung
// (WAYLAND_DISPLAY) — der Test hinge dann an der laufenden Sitzung und würde ohne sie (CI, Mutantenlauf aus einem
// Dienst) still übersprungen (eglInitialize 0x3001). Mit surfaceless läuft er auch ohne Anzeige und mit
// Software-Rasterung (Kritik zu Karte #224). Wer den Sprung nicht hinnehmen will, setzt SCHNAPP_HEADLESS_PFLICHT=1.
func headlessBild(t *testing.T, groesse image.Point, zeichne func(gtx layout.Context)) *image.RGBA {
	t.Helper()
	if os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	w, err := headless.NewWindow(groesse.X, groesse.Y)
	if err != nil {
		if os.Getenv("SCHNAPP_HEADLESS_PFLICHT") != "" {
			t.Fatalf("keine Zeichenfläche ohne Fenster (SCHNAPP_HEADLESS_PFLICHT gesetzt): %v", err)
		}
		t.Skipf("keine Zeichenfläche ohne Fenster: %v", err)
	}
	t.Cleanup(w.Release)
	ops := new(op.Ops)
	zeichne(layout.Context{Ops: ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(groesse)})
	if err := w.Frame(ops); err != nil {
		t.Fatal(err)
	}
	bild := image.NewRGBA(image.Rectangle{Max: groesse})
	if err := w.Screenshot(bild); err != nil {
		t.Fatal(err)
	}
	return bild
}

// zaehleFarbe zählt die Pixel genau dieser Farbe.
func zaehleFarbe(b *image.RGBA, c color.NRGBA) int {
	n := 0
	for y := b.Rect.Min.Y; y < b.Rect.Max.Y; y++ {
		for x := b.Rect.Min.X; x < b.Rect.Max.X; x++ {
			p := b.RGBAAt(x, y)
			if p.R == c.R && p.G == c.G && p.B == c.B && p.A == c.A {
				n++
			}
		}
	}
	return n
}

// zeichenLauf: Editor mit dem Text „Hallo“ und einem vollrot gefüllten Kasten (4,4)–(24,24) ohne Strich.
func zeichenLauf(t *testing.T) (*menueLauf, int) {
	t.Helper()
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(24, 24)),
		Fuellung: color.NRGBA{R: 255, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	l.setle()
	return l, k
}

// Editor zeichnet Objekte (zeichneObjekte): sichtbar deckend, ausgeblendet gar nicht, mit Ebenen-Deckkraft anteilig
// (ADR-0032 Punkte 4 und 10; Verify V23-5).
func TestEditorZeichnetEbenen(t *testing.T) {
	l, k := zeichenLauf(t)
	mitteKasten := func() color.RGBA {
		b := headlessBild(t, image.Pt(64, 64), func(gtx layout.Context) { l.e.zeichneObjekte(gtx) })
		return b.RGBAAt(14, 14)
	}
	if p := mitteKasten(); p.R != 255 || p.A != 255 {
		t.Fatalf("sichtbarer Kasten: Pixel %v, Soll deckendes Rot", p)
	}
	l.e.s.SetzeDeckkraft(k, 50)
	if p := mitteKasten(); p.A < 110 || p.A > 145 || p.R == 0 {
		t.Errorf("Deckkraft 50 %%: Pixel %v, Soll rund halb durchsichtiges Rot", p)
	}
	l.e.s.SetzeDeckkraft(k, 100)
	l.e.s.SchalteAusgeblendet(k)
	if p := mitteKasten(); p.A != 0 {
		t.Errorf("ausgeblendeter Kasten: Pixel %v, Soll durchsichtig", p)
	}
	l.e.s.SchalteAusgeblendet(k)
	if p := mitteKasten(); p.A != 255 {
		t.Errorf("wieder eingeblendet: Pixel %v, Soll deckend", p)
	}
}

// Auswahlrahmen und Griffe (zeichneAuswahl): sichtbare Wahl bekommt Rahmen und Griffe, eine ausgeblendete nichts, eine
// gesperrte nur den Rahmen (ADR-0032 Punkte 4 und 5).
func TestEditorZeichnetAuswahlRahmenUndGriffe(t *testing.T) {
	l, k := zeichenLauf(t)
	l.e.s.WaehleEbene(k)
	l.setle()
	flaeche := image.Pt(1200, 800)
	zaehle := func() (rahmen, griffe int) {
		b := headlessBild(t, flaeche, func(gtx layout.Context) { l.e.zeichneAuswahl(gtx) })
		return zaehleFarbe(b, farbeAuswahl), zaehleFarbe(b, farbeText)
	}
	r, g := zaehle()
	if r == 0 || g == 0 {
		t.Fatalf("gewählter Kasten: %d Rahmen-, %d Griffpixel, Soll beide über 0", r, g)
	}
	l.e.s.SchalteGesperrt(k)
	r2, g2 := zaehle()
	if r2 == 0 || g2 != 0 {
		t.Errorf("gesperrt: %d Rahmen-, %d Griffpixel, Soll Rahmen ohne Griffe", r2, g2)
	}
	l.e.s.SchalteGesperrt(k)
	l.e.s.SchalteAusgeblendet(k)
	if !l.e.s.KannEbene() || l.e.s.Auswahl != k {
		l.e.s.Auswahl = k // das Ausblenden behält die Wahl; der Test prüft nur das Zeichnen
	}
	r3, g3 := zaehle()
	if r3 != 0 || g3 != 0 {
		t.Errorf("ausgeblendet: %d Rahmen-, %d Griffpixel, Soll keine", r3, g3)
	}
}
