// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Regressionstests aus dem Verify der Sitzungs-QA 23.11. abends: zuletzt benutzte Farben bei schwebender Eingabe
// (V-1), Fensterwechsel, Rechtsklick und Feld zu Feld.

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// V-1: schwebende Hex-Eingabe, dann Klick auf eine zuletzt benutzte Farbe: Die Wache übernimmt zuerst, die Liste der
// zuletzt benutzten Farben rückt um eins, der Klick trifft danach eine andere Farbe.
func TestSitzungsVerifyZuletztVerschiebt(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	a := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	b := color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff}
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("112233ff")
	l.taste(key.NameReturn, 0)
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("445566ff")
	l.taste(key.NameReturn, 0)
	z := l.e.s.Zuletzt()
	if len(z) != 2 || z[0] != b || z[1] != a {
		t.Fatalf("Ausgang: zuletzt %v", z)
	}
	n := len(canvas.Beschriftungsfarben)
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("778899ff")                            // schwebt
	ziel := mittenPunkt(l.e.seite.farbRahmen[n+1]) // zeigt a
	l.klick(ziel)
	t.Logf("nach Klick auf das Feld mit %v: aktuelle Farbe %v, zuletzt %v", a, l.e.s.AktuelleFarbe(), l.e.s.Zuletzt())
	if l.e.s.AktuelleFarbe() != a {
		t.Errorf("Klick auf zuletzt benutzte Farbe %v ergab %v", a, l.e.s.AktuelleFarbe())
	}
}

// V-1b: dasselbe mit Drücken und Loslassen in getrennten Bildern, Klick auf das erste zuletzt-Feld.
func TestSitzungsVerifyZuletztErstesFeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	a := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("112233ff")
	l.taste(key.NameReturn, 0)
	l.klick(mittenPunkt(l.e.seite.farbRahmen[0])) // Rot, damit a nicht schon gilt
	n := len(canvas.Beschriftungsfarben)
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("778899ff")                       // schwebt
	p := mittenPunkt(l.e.seite.farbRahmen[n]) // zeigt a
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, p, 0)
	l.setle()
	t.Logf("aktuelle Farbe %v, zuletzt %v", l.e.s.AktuelleFarbe(), l.e.s.Zuletzt())
	if l.e.s.AktuelleFarbe() != a {
		t.Errorf("Klick auf zuletzt benutzte Farbe %v ergab %v", a, l.e.s.AktuelleFarbe())
	}
}

// Fensterwechsel: Eingabe geht nach der Rückkehr weiter.
func TestSitzungsVerifyFensterwechselWeiter(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("336")
	l.r.Queue(pointer.Event{Kind: pointer.Cancel})
	l.r.Queue(key.FocusEvent{Focus: false})
	l.setle()
	t.Logf("ohne Fensterfokus: Feld zeigt %q, Fokus Hex %v", fw.hex.gezeigt, l.fokus(&fw.hex))
	l.r.Queue(key.FocusEvent{Focus: true})
	l.setle()
	l.tippe("699")
	t.Logf("zurück: Feld zeigt %q", fw.hex.gezeigt)
	l.taste(key.NameReturn, 0)
	want := color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: l.e.s.AktuelleFarbe().A}
	if l.e.s.AktuelleFarbe() != want {
		t.Errorf("nach Fensterwechsel: %v, erwartet %v", l.e.s.AktuelleFarbe(), want)
	}
}

// Rechtsklick und mittlere Taste woanders: übernimmt ebenfalls; Mausrad nicht.
func TestSitzungsVerifyRechtsklickWoanders(t *testing.T) {
	for _, taste := range []pointer.Buttons{pointer.ButtonSecondary, pointer.ButtonTertiary} {
		l := neuerMenueLauf(t, image.Pt(1200, 800))
		fw := &l.e.seite.fw
		l.klick(mittenPunkt(fw.hex.rahmen))
		l.tippe("00ff00ff")
		f := l.e.leinwandFlaeche(l.gtx())
		p := f.Min.Add(image.Pt(6, 6))
		l.maus(pointer.Move, p, 0)
		l.setle()
		l.maus(pointer.Press, p, taste)
		l.setle()
		l.maus(pointer.Release, p, 0)
		l.setle()
		t.Logf("Taste %v: Farbe %v, Fokus Hex %v, Fokus Leinwand %v, pan/zieht %v", taste, l.e.s.AktuelleFarbe(), l.fokus(&fw.hex), l.fokus(l.e), l.e.s.Zieht())
	}
}

// Klick vom Deckkraft-Feld ins Hex-Feld und umgekehrt: übernimmt, Fokus im angeklickten Feld.
func TestSitzungsVerifyFeldZuFeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	fw := &l.e.seite.fw
	l.klick(mittenPunkt(fw.hex.rahmen))
	l.tippe("00ff00ff")
	l.klick(mittenPunkt(fw.deckkraft.rahmen))
	t.Logf("Hex→Deckkraft: Farbe %v, Fokus Deckkraft %v, Hex zeigt %q, Deckkraft zeigt %q", l.e.s.AktuelleFarbe(), l.fokus(&fw.deckkraft), fw.hex.gezeigt, fw.deckkraft.gezeigt)
	if !l.fokus(&fw.deckkraft) || l.e.s.AktuelleFarbe() != (color.NRGBA{G: 255, A: 255}) {
		t.Errorf("Hex→Deckkraft: Farbe %v, Fokus %v", l.e.s.AktuelleFarbe(), l.fokus(&fw.deckkraft))
	}
	l.tippe("50")
	l.klick(mittenPunkt(fw.hex.rahmen))
	t.Logf("Deckkraft→Hex: Farbe %v, Fokus Hex %v", l.e.s.AktuelleFarbe(), l.fokus(&fw.hex))
	if !l.fokus(&fw.hex) || canvas.Deckkraft(l.e.s.AktuelleFarbe()) != 50 {
		t.Errorf("Deckkraft→Hex: Farbe %v, Fokus %v", l.e.s.AktuelleFarbe(), l.fokus(&fw.hex))
	}
}
