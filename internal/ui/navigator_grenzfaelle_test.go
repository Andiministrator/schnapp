// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Regressionstests zum Navigator (Sammelkritik Sprint 12 KS-3, KS-8, F-613, Karte #236).

import (
	"image"
	"testing"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// KS-4 (#197): Beginnt das Speichern, während die Taste im Navigator gedrückt ist, verwirft navigatorEreignisse das
// Loslassen (continue vor dem switch); nav.zieht bleibt stehen, und danach führt auch die rechte Taste den Ausschnitt.
func TestNavigatorLoslassenWaehrendSpeichern(t *testing.T) {
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
	if _, err := l.e.s.SpeichernJetzt(); err != nil || !l.e.s.SpeichernLaeuft() { // wie Strg+S bei gedrückter Taste
		t.Fatalf("Kontrolle: Speichern beginnt nicht (%v, läuft %v)", err, l.e.s.SpeichernLaeuft())
	}
	l.maus(pointer.Release, p, 0)
	l.setle()
	l.e.s.SpeichernEnde(false, nil)
	l.setle()
	if l.e.nav.zieht {
		t.Error("nach Loslassen während des Speicherns: nav.zieht steht noch")
	}
	vor := l.e.s.Abb
	q := fensterPt(o, n.ZuKasten(document.Pt(300, 200)))
	l.maus(pointer.Press, p, pointer.ButtonSecondary)
	l.setle()
	l.maus(pointer.Move, q, pointer.ButtonSecondary)
	l.setle()
	l.maus(pointer.Release, q, 0)
	l.setle()
	if l.e.s.Abb != vor {
		t.Errorf("Ziehen mit der rechten Taste verschob die Ansicht: %v, vorher %v", l.e.s.Abb, vor)
	}
}

// Gegenprobe zu KS-4 ohne Speichern: Die rechte Taste führt den Ausschnitt nicht.
func TestNavigatorRechteTasteFuehrtNicht(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	p := fensterPt(o, n.ZuKasten(document.Pt(900, 400)))
	q := fensterPt(o, n.ZuKasten(document.Pt(300, 200)))
	vor := l.e.s.Abb
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonSecondary)
	l.setle()
	l.maus(pointer.Move, q, pointer.ButtonSecondary)
	l.setle()
	l.maus(pointer.Release, q, 0)
	l.setle()
	if l.e.s.Abb != vor {
		t.Errorf("rechte Taste verschob die Ansicht: %v, vorher %v", l.e.s.Abb, vor)
	}
}

// Prüfung (#197): Bei Skala 2 ist der Kasten 320 px lang und 24 px von den Kanten entfernt.
func TestNavigatorHiDPI(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	f := l.e.s.Fenster()
	g := layout.Context{Ops: new(op.Ops), Source: l.r.Source(), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
		Constraints: layout.Exact(f)}
	l.e.zeichneNavigator(g)
	r := l.e.nav.rahmen
	if r.Dx() != 320 || r.Max != image.Pt(f.X-24, f.Y-24) {
		t.Errorf("Skala 2: Kasten %v im Fenster %v, Soll 320 breit, Ende bei %v", r, f, image.Pt(f.X-24, f.Y-24))
	}
}

// Prüfung (#197): Unter der Hilfetafel wirkt ein Klick an der Stelle des Navigators nicht auf die Ansicht.
func TestNavigatorUnterHilfetafel(t *testing.T) {
	l := navigatorEditorLauf(t)
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	l.e.hilfeOffen = true
	l.setle()
	vor := l.e.s.Abb
	l.klick(fensterPt(o, n.ZuKasten(document.Pt(100, 100))))
	if l.e.s.Abb != vor {
		t.Errorf("Klick unter der Hilfetafel verschob die Ansicht: %v, vorher %v", l.e.s.Abb, vor)
	}
}

// Prüfung (#197): Vollansicht beim Zuschneiden mit vorhandenem Zuschnitt. Ein Klick auf die Kastenmitte legt die Mitte
// der Vollansicht in die Fenstermitte.
func TestNavigatorBeimZuschneiden(t *testing.T) {
	l := navigatorEditorLauf(t)
	if err := l.e.s.D.SetzeZuschnitt(image.Rect(300, 200, 900, 600)); err != nil {
		t.Fatalf("Kontrolle: Zuschnitt: %v", err)
	}
	l.e.bildNeu()
	l.e.s.SetzeWerkzeug(canvas.WerkzeugZuschneiden)
	l.setle()
	l.e.s.SetzeZoom(400)
	l.setle()
	n, o := l.navKasten(t)
	bf := l.e.s.Bildflaeche()
	t.Logf("Bildfläche %v, Kasten %v", bf, n.Flaeche)
	if bf.Dx() != 1200 || bf.Dy() != 800 {
		t.Fatalf("Kontrolle: Bildfläche %v ist nicht die Vollansicht", bf)
	}
	m := mitte(n.Flaeche).Add(o)
	l.maus(pointer.Move, m, 0)
	l.setle()
	l.maus(pointer.Press, m, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, m, 0)
	l.setle()
	f := l.e.s.Fenster()
	got := l.e.s.Abb.ZuAnsicht(document.Pt(float64(f.X)/2, float64(f.Y)/2))
	soll := document.Pt(float64(bf.Min.X+bf.Max.X)/2, float64(bf.Min.Y+bf.Max.Y)/2)
	if d := got.Minus(soll).Laenge(); d > 5 {
		t.Errorf("Fenstermitte zeigt %v, Soll rund %v (Mitte der Vollansicht)", got, soll)
	}
	if l.e.s.Zieht() {
		t.Error("Klick im Navigator begann einen Zug des Zuschnitts")
	}
}
