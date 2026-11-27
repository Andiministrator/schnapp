// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/Andiministrator/schnapp/internal/canvas"
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

// Editor wie Renderer (F-735): Liegt das Schild einer Sprechblase ganz in einem Stauch-Streifen und ragt nur die Spitze
// heraus, zeichnet zeichneObjekte nur das Dreieck, keine Zeile hinter der Naht.
func TestEditorZeichnetSprechblaseImStreifenNurDreieck(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 300, 200)))
	if err != nil {
		t.Fatal(err)
	}
	schrift := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	fuell := color.NRGBA{R: 30, G: 90, B: 200, A: 255}
	sp := document.Pt(270, 95)
	tx := document.Text{Ort: document.Pt(100.3, 80.6), Inhalt: "GEHEIM-4711\nzwei", Schriftgrad: 14, Farbe: schrift,
		Hintergrund: fuell, Spitze: &sp}
	if _, err := d.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	vorher := headlessBild(t, image.Pt(300, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if zaehleFarbe(vorher, schrift) == 0 {
		t.Fatal("Kontrolle: ohne Streifen keine Schrift gezeichnet")
	}
	g := tx.Grenzen()
	st := document.Streifen{Von: int(g.Min.X) - 6, Bis: int(g.Max.X) + 7}
	if err := d.SetzeStreifen([]document.Streifen{st}); err != nil {
		t.Fatal(err)
	}
	if !d.NurSpitzeSichtbar(tx) {
		t.Fatal("Aufbau: Schrift soll im Streifen liegen")
	}
	b := headlessBild(t, image.Pt(300, 200), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if n := zaehleFarbe(b, schrift); n > 0 {
		t.Errorf("%d Pixel in Schriftfarbe: Text hinter der Naht gezeichnet", n)
	}
	if zaehleFarbe(b, fuell) == 0 {
		t.Error("Dreieck fehlt")
	}
}

// Mehrfachauswahl (ADR-0039 Punkt 5): Jedes weitere gewählte Objekt trägt einen Rahmen ohne Griffe, die Hauptauswahl
// Rahmen und Griffe; ein aufgezogener Auswahlrahmen ist eine Linie in der Auswahlfarbe über einer getönten Fläche.
func TestEditorZeichnetMehrfachauswahlUndRahmen(t *testing.T) {
	l, k := zeichenLauf(t)
	flaeche := image.Pt(1200, 800)
	zaehle := func() (rahmen, griffe, getoent int) {
		b := headlessBild(t, flaeche, func(gtx layout.Context) { l.e.zeichneAuswahl(gtx) })
		for y := 0; y < flaeche.Y; y++ {
			for x := 0; x < flaeche.X; x++ {
				if p := b.RGBAAt(x, y); p.A > 0 && p.A < 0x40 && p.G > p.R {
					getoent++
				}
			}
		}
		return zaehleFarbe(b, farbeAuswahl), zaehleFarbe(b, farbeText), getoent
	}
	l.e.s.WaehleEbene(k)
	r1, g1, _ := zaehle()
	if !l.e.s.WaehleAlle() || l.e.s.AnzahlGewaehlt() != 2 {
		t.Fatalf("Strg+A wählt %d, Soll 2", l.e.s.AnzahlGewaehlt())
	}
	haupt := l.e.s.Auswahl
	l.e.s.WaehleEbene(haupt) // nur die Hauptauswahl, zum Vergleich der Griffe
	_, gHaupt, _ := zaehle()
	l.e.s.WaehleAlle()
	r2, g2, _ := zaehle()
	if r2 <= r1 || g2 != gHaupt {
		t.Errorf("zwei gewählt: %d Rahmenpixel (einer: %d), %d Griffpixel (nur Hauptauswahl: %d); Soll mehr Rahmen, gleiche Griffe",
			r2, r1, g2, gHaupt)
	}
	_ = g1
	// Rahmen aufziehen: im Bild auf freier Fläche (rechts oben, neben Text und Kasten).
	l.e.s.Abwaehlen()
	a := l.e.s.Abb.ZuFenster(document.Pt(31, 1))
	b := l.e.s.Abb.ZuFenster(document.Pt(39.5, 8))
	l.e.s.Druecken(a, canvas.TastePrimaer)
	l.e.s.Ziehen(b)
	if _, ok := l.e.s.AuswahlRahmen(); !ok {
		for _, o := range l.e.s.D.Objekte() {
			t.Logf("Objekt %d Grenzen %v", o.Kennung(), o.Grenzen())
		}
		t.Fatalf("kein Auswahlrahmen beim Ziehen (Werkzeug %v, zieht %v, Auswahl %d, a %v b %v, Zoom %v)", l.e.s.Werkzeug, l.e.s.Zieht(), l.e.s.Auswahl, a, b, l.e.s.Abb.Zoom)
	}
	r3, _, t3 := zaehle()
	if r3 == 0 || t3 == 0 {
		t.Errorf("aufgezogener Rahmen: %d Linien-, %d getönte Pixel, Soll beide über 0", r3, t3)
	}
	l.e.s.Loslassen(b)
}

// Lage und Kontur (Kritik #334, F-909): Der Rahmen eines weiteren gewählten Objekts liegt 2 dp außen um seine Grenzen,
// die dunkle Kontur 1 dp weiter außen; der aufgezogene Rahmen liegt auf den Ecken des Zugs, die Kontur außen daneben.
func TestEditorZeichnetAuswahlrahmenAnDerRichtigenStelle(t *testing.T) {
	l, k := zeichenLauf(t)
	flaeche := image.Pt(1200, 800)
	// Zweiter Kasten abseits als Hauptauswahl (seine Griffe liegen weit weg), der erste ist ein weiteres gewähltes.
	k2, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(30, 2), document.Pt(38, 10)),
		Fuellung: color.NRGBA{B: 255, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.WaehleEbene(k)
	if !l.e.s.SchalteGewaehlt(k2) || l.e.s.Auswahl != k2 || !l.e.s.IstGewaehlt(k) {
		t.Fatalf("Auswahl %d, gewählt %v", l.e.s.Auswahl, l.e.s.Gewaehlt())
	}
	o, _ := l.e.s.D.Objekt(k)
	g := o.Grenzen()
	fe := func(p document.Punkt) image.Point {
		q := l.e.s.Abb.ZuFenster(l.e.s.D.ZuAnsicht(p))
		return image.Pt(int(math.Round(q.X)), int(math.Round(q.Y)))
	}
	r := image.Rectangle{Min: fe(g.Min), Max: fe(g.Max)}.Canon()
	gleich := func(p color.RGBA, c color.NRGBA) bool { return p.R == c.R && p.G == c.G && p.B == c.B && p.A == c.A }
	b := headlessBild(t, flaeche, func(gtx layout.Context) { l.e.zeichneAuswahl(gtx) })
	mitteY := (r.Min.Y + r.Max.Y) / 2
	if p := b.RGBAAt(r.Min.X-2, mitteY); !gleich(p, farbeAuswahl) {
		t.Errorf("weiteres Objekt: Pixel links 2 außen %v, Soll Auswahlfarbe", p)
	}
	if p := b.RGBAAt(r.Min.X-3, mitteY); !gleich(p, farbeAuswahlKontur) {
		t.Errorf("weiteres Objekt: Pixel links 3 außen %v, Soll Kontur", p)
	}
	// Aufgezogener Rahmen (auf freier Fläche: der zweite Kasten weicht vorher).
	l.e.s.Abwaehlen()
	if err := l.e.s.D.Entfernen(k2); err != nil {
		t.Fatal(err)
	}
	a := l.e.s.Abb.ZuFenster(document.Pt(31, 1))
	z := l.e.s.Abb.ZuFenster(document.Pt(39.5, 8))
	l.e.s.Druecken(a, canvas.TastePrimaer)
	l.e.s.Ziehen(z)
	rr, ok := l.e.s.AuswahlRahmen()
	if !ok {
		t.Fatal("kein Auswahlrahmen")
	}
	b = headlessBild(t, flaeche, func(gtx layout.Context) { l.e.zeichneAuswahl(gtx) })
	x0, y := int(math.Round(rr.Min.X)), int(math.Round((rr.Min.Y+rr.Max.Y)/2))
	if p := b.RGBAAt(x0, y); !gleich(p, farbeAuswahl) {
		t.Errorf("aufgezogener Rahmen: Pixel an der linken Kante %v, Soll Auswahlfarbe", p)
	}
	if p := b.RGBAAt(x0-1, y); !gleich(p, farbeAuswahlKontur) {
		t.Errorf("aufgezogener Rahmen: Pixel links daneben %v, Soll Kontur", p)
	}
	l.e.s.Loslassen(z)
}
