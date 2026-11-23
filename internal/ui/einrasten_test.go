// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 12 (Karte #196): Die Alt-Taste kommt aus dem Zeigerereignis in die Sitzung und schaltet das
// Einrasten während des Ziehens ab; das Menü schaltet es ganz ab.
func TestAltAusDemZeigerereignisSchaltetEinrastenAb(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 1200))
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(4, 4), document.Pt(24, 24)),
		Fuellung: color.NRGBA{R: 200, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	o := l.e.leinwandFlaeche(l.gtx()).Min // die Leinwand hat einen eigenen Ursprung im Fenster
	punkt := func(x, y float64) f32.Point {
		p := l.e.s.Abb.ZuFenster(document.Pt(x, y)).Plus(document.Pt(float64(o.X), float64(o.Y)))
		return f32.Pt(float32(p.X), float32(p.Y))
	}
	zieh := func(mod key.Modifiers) []canvas.Hilfslinie {
		l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: punkt(14, 14)})
		l.setle()
		l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary,
			Position: punkt(14, 14), Modifiers: mod})
		l.setle()
		// Rechte Kante von 24 auf 39,5: 0,5 Pixel vom Bildrand (Breite 40) entfernt.
		l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary,
			Position: punkt(29.5, 14), Modifiers: mod})
		l.setle()
		linien := l.e.s.Hilfslinien()
		l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: punkt(29.5, 14), Modifiers: mod})
		l.setle()
		return linien
	}
	if linien := zieh(0); len(linien) == 0 || l.e.s.AltGedrueckt() {
		t.Errorf("ohne Alt: Linien %v, Alt %v, Soll Linien und kein Alt", linien, l.e.s.AltGedrueckt())
	}
	l.e.s.Rueckgaengig()
	if linien := zieh(key.ModAlt); len(linien) != 0 || !l.e.s.AltGedrueckt() {
		t.Errorf("mit Alt: Linien %v, Alt %v, Soll keine Linien und Alt", linien, l.e.s.AltGedrueckt())
	}
}

// Der Menüeintrag „Snap to objects“ schaltet um und zeigt den Zustand.
func TestMenueSchaltetEinrasten(t *testing.T) {
	e := testEditor(t)
	zustand := func() bool {
		for _, en := range e.menueEintraegeVon(menueAnsicht) {
			if en.akt == aktEinrasten {
				return en.aktiv
			}
		}
		t.Fatal("kein Eintrag Einrasten im Menü Ansicht")
		return false
	}
	if !zustand() || !e.s.EinrastenAn() {
		t.Fatal("Voreinstellung: an")
	}
	e.fuehreAus(layout.Context{}, aktEinrasten, true)
	if zustand() || e.s.EinrastenAn() {
		t.Error("nach dem Menüeintrag: aus")
	}
}
