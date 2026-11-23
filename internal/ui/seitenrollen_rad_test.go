// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/pointer"
)

// radSchritte dreht das Mausrad wie die Fernbedienung: n Raststufen zu je 15 px nacheinander an derselben Stelle des
// Fensters, nach jeder Stufe ein Bild (der Inhalt der Leiste wandert dabei unter dem Zeiger weg).
func (l *menueLauf) radSchritte(p image.Point, n int, stufe float32) {
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos})
	l.setle()
	for range n {
		l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: pos, Scroll: f32.Pt(0, stufe)})
		l.setle()
	}
}

// Sichtprobe Karte #225: Das Mausrad rollt die Leiste über dem Kopf, über der Farbfläche und über einem Eingabefeld bis ans
// Ende und zurück, auch wenn der Inhalt unter dem Zeiger weiterwandert. Gio deckt ein Rad ab, sobald ein undurchlässiger
// Bereich (Farbfläche, Feld, Knopf) darüber liegt, wenn die Anmeldung der Leiste nicht im selben Ausschnitt-Bereich steht
// wie der Inhalt: früher blieb das Rad nach 75 px stehen.
func TestMausradRolltUeberJedemElementBisZumEnde(t *testing.T) {
	for _, wahl := range []string{"Text", "Kasten"} {
		l, k := ebenenLauf(t)
		if wahl == "Kasten" {
			l.e.s.WaehleEbene(k)
		}
		l.fenst = image.Pt(1000, 330)
		l.setle()
		se, roll := &l.e.seite, &l.e.seite.roll
		if roll.rollMax < 100 {
			t.Fatalf("%s: Kontrolle: rollMax %d zu klein für den Versuch", wahl, roll.rollMax)
		}
		fl := se.fw.flaeche
		stellen := map[string]image.Point{
			"Kopf":         image.Pt(roll.sicht.Min.X+50, roll.sicht.Min.Y+30),
			"Farbfläche":   image.Pt(fl.Min.X+20, roll.sicht.Max.Y-20),
			"Hex-Feld":     image.Pt(se.fw.hex.rahmen.Min.X+20, roll.sicht.Max.Y-20),
			"Mitte rechts": image.Pt(roll.sicht.Max.X-30, (roll.sicht.Min.Y+roll.sicht.Max.Y)/2),
		}
		for name, p := range stellen {
			l.rolle(-1 << 20)
			l.radSchritte(p, 40, 15)
			if roll.rollen != roll.rollMax {
				t.Errorf("%s, Rad über %s %v: gerollt bis %d von %d", wahl, name, p, roll.rollen, roll.rollMax)
			}
			l.radSchritte(p, 40, -15)
			if roll.rollen != 0 {
				t.Errorf("%s, Rad über %s zurück: Versatz %d, Soll 0", wahl, name, roll.rollen)
			}
		}
	}
}
