// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// Karte #352: Ziehen am Regler der Strichstärke ändert den gewählten Pfeil als einen Rückgängig-Schritt, die Anzeige
// folgt; ganz links ist 1 px, ganz rechts 32 px.
func TestStrichReglerZiehen(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(20, 20), document.Pt(120, 80),
		document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 5}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	r := l.e.seite.strRegler
	if r.Empty() {
		t.Fatal("Regler nicht gezeichnet")
	}
	breite := func() float64 {
		o, _ := l.e.s.D.Objekt(id)
		return o.(document.Pfeil).Strich.Breite
	}
	l.ziehe(lageIn(r, 0.4, 0.5), lageIn(r, 1.2, 0.5)) // über das Ende hinaus
	if breite() != canvas.ReglerMax || l.e.seite.strZieht || l.e.s.BreiteZugLaeuft() {
		t.Fatalf("nach dem Ziehen nach rechts: %v px, zieht %v", breite(), l.e.seite.strZieht)
	}
	l.setle()
	if l.e.seite.strichText != "32 px" {
		t.Errorf("Anzeige %q", l.e.seite.strichText)
	}
	if !l.e.s.Rueckgaengig() || breite() != 5 {
		t.Errorf("ein Rückgängig: %v px, Soll 5", breite())
	}
	l.setle()
	l.ziehe(lageIn(l.e.seite.strRegler, 0.6, 0.5), lageIn(l.e.seite.strRegler, -0.5, 0.5))
	if breite() != canvas.ReglerMin {
		t.Errorf("nach dem Ziehen nach links: %v px", breite())
	}
}

// Kritik 26.11. K-7: Klappt die Leiste mitten im Ziehen am Regler ein, endet der Zug trotzdem — sonst hielte Zieht() das
// Speichern bis zu einem Esc auf.
func TestStrichReglerLeisteVerschwindetImZug(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	id, err := l.e.s.D.Hinzufuegen(document.GeraderPfeil(document.Pt(20, 20), document.Pt(120, 80),
		document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 5}))
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	r := l.e.seite.strRegler
	if r.Empty() {
		t.Fatal("Regler nicht gezeichnet")
	}
	a, b := lageIn(r, 0.3, 0.5), lageIn(r, 0.8, 0.5)
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(a.X), float32(a.Y))})
	l.setle()
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(float32(a.X), float32(a.Y))})
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(float32(b.X), float32(b.Y))})
	l.setle()
	if !l.e.s.BreiteZugLaeuft() {
		t.Fatal("kein Zug am Regler")
	}
	l.taste("B", key.ModShortcut|key.ModShift) // Leiste einklappen
	l.setle()
	if !l.e.seite.strRegler.Empty() {
		t.Fatal("Leiste nicht eingeklappt")
	}
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(float32(b.X), float32(b.Y))})
	l.setle()
	l.setle()
	if l.e.s.BreiteZugLaeuft() || l.e.s.Zieht() || l.e.seite.strZieht {
		t.Errorf("Zug hängt: Sitzung %v, Leiste %v", l.e.s.BreiteZugLaeuft(), l.e.seite.strZieht)
	}
}
