// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

// rolle dreht das Mausrad über dem Kopf der Leiste (dort liegt keine Liste, die das Rad für sich nimmt).
func (l *menueLauf) rolle(dy float32) {
	s := l.e.seite.roll.sicht
	p := f32.Pt(float32(s.Min.X+10), float32(s.Min.Y+4))
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: p, Scroll: f32.Pt(0, dy)})
	l.setle()
}

// imSichtfeld meldet, ob r (Inhaltskoordinaten) beim jetzigen Versatz ganz in der Fläche der Leiste liegt.
func (l *menueLauf) imSichtfeld(r image.Rectangle) bool {
	k := &l.e.seite.roll
	return !r.Empty() && r.Sub(image.Pt(0, k.rollen)).In(k.sicht)
}

func TestRollRechnung(t *testing.T) {
	if got := rollEnde(300, 400); got != 0 {
		t.Errorf("rollEnde passt: %d", got)
	}
	if got := rollEnde(500, 400); got != 100 {
		t.Errorf("rollEnde ragt: %d", got)
	}
	for _, c := range []struct{ rollen, max, want int }{{-5, 100, 0}, {50, 100, 50}, {150, 100, 100}, {10, 0, 0}} {
		if got := rollBegrenze(c.rollen, c.max); got != c.want {
			t.Errorf("rollBegrenze(%d, %d) = %d, Soll %d", c.rollen, c.max, got, c.want)
		}
	}
	// Sichtfläche 100..400, Rand 8, größter Versatz 300.
	r := func(y0, y1 int) image.Rectangle { return image.Rect(0, y0, 10, y1) }
	for _, c := range []struct {
		name   string
		rollen int
		r      image.Rectangle
		want   int
	}{
		{"sichtbar bleibt", 0, r(200, 220), 0},
		{"unten heraus", 0, r(500, 520), 128},            // Unterkante 520+8 auf 400
		{"oben heraus", 200, r(150, 170), 42},            // Oberkante 150-8 auf 100
		{"zu hoch: Oberkante gilt", 0, r(500, 900), 300}, // Oberkante 500-8-100 = 392, gedeckelt
		{"zurück an den Anfang", 80, r(100, 120), 0},
		{"gedeckelt", 0, r(2000, 2020), 300},
	} {
		if got := rollZeige(c.rollen, c.r, 100, 400, 8, 300); got != c.want {
			t.Errorf("%s: rollZeige = %d, Soll %d", c.name, got, c.want)
		}
	}
	if o, n := rollDaumen(200, 200, 200, 0); o != 0 || n != 200 {
		t.Errorf("Daumen ohne Überstand: %d, %d", o, n)
	}
	if o, n := rollDaumen(200, 200, 400, 200); n != 100 || o != 100 {
		t.Errorf("Daumen am Ende: %d, %d", o, n)
	}
}

// F-598 (Sonde aus Kritik #219 B-5): In jedem Fenster ab dem Mindestmaß liegen Hex und Deckkraft des Farbwählers
// entweder ohne Rollen über der Fußzeile oder lassen sich dorthin rollen; das Mausrad erreicht das Ende, und was
// nicht passt, gibt dem Rollen Raum (rollMax > 0).
func TestFarbwaehlerErreichbarInNiedrigenFenstern(t *testing.T) {
	for _, wahl := range []string{"Text", "Kasten"} {
		l, k := ebenenLauf(t)
		if wahl == "Kasten" {
			l.e.s.WaehleEbene(k)
		}
		fw, roll := &l.e.seite.fw, &l.e.seite.roll
		for h := 320; h <= 800; h += 2 {
			l.fenst = image.Pt(1200, h)
			l.rolle(-1 << 20) // zurück nach oben
			if !l.e.seite.farbeGezeigt {
				t.Fatalf("%s, Höhe %d: Farbwähler nicht gezeigt (Auto-Einklappen der Farbe ist verboten, F-592)", wahl, h)
			}
			passt := l.imSichtfeld(fw.hex.rahmen) && l.imSichtfeld(fw.deckkraft.rahmen)
			if !passt && roll.rollMax == 0 {
				t.Fatalf("%s, Höhe %d: Hex %v / Deckkraft %v unter der Fußzeile (%v) und nichts zum Rollen", wahl, h,
					fw.hex.rahmen, fw.deckkraft.rahmen, roll.sicht)
			}
			l.rolle(1 << 20) // ganz nach unten
			if roll.rollen != roll.rollMax {
				t.Fatalf("%s, Höhe %d: Mausrad erreicht das Ende nicht (%d von %d)", wahl, h, roll.rollen, roll.rollMax)
			}
			if !l.imSichtfeld(fw.hex.rahmen) || !l.imSichtfeld(fw.deckkraft.rahmen) {
				t.Fatalf("%s, Höhe %d: nach dem Rollen Hex %v / Deckkraft %v nicht in %v (Versatz %d)", wahl, h,
					fw.hex.rahmen, fw.deckkraft.rahmen, roll.sicht, roll.rollen)
			}
		}
	}
}

// Ein hohes Fenster rollt nicht: kein Überstand, das Mausrad bleibt ohne Wirkung, ganz oben.
func TestHohesFensterRolltNicht(t *testing.T) {
	l, _ := ebenenLauf(t)
	roll := &l.e.seite.roll
	for _, h := range []int{800, 1000, 1200} {
		l.fenst = image.Pt(1200, h)
		l.setle()
		l.rolle(500)
		if roll.rollMax != 0 || roll.rollen != 0 {
			t.Errorf("Höhe %d: rollMax %d, rollen %d, Soll beide 0", h, roll.rollMax, roll.rollen)
		}
	}
}

// Tab-Weg: Der Fokus auf Hex und Deckkraft rollt das Feld ins Bild; Umschalt+Tab zurück zum Farbpaar rollt wieder hinauf.
// Die Reihenfolge der Felder bleibt (ADR-0031 Nachtrag 5).
func TestTabRolltFokusInsBild(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 330)
	l.setle()
	se, roll := &l.e.seite, &l.e.seite.roll
	if roll.rollMax == 0 {
		t.Fatal("Kontrolle: Im Fenster von 330 px Höhe muss die Leiste rollen")
	}
	l.fokusiere(&se.farben[0])
	l.fokusiere(&se.fw.pipette)
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&se.fw.hex) {
		t.Fatalf("Tab nach der Pipette: Fokus nicht auf Hex (%s)", menueFokusName(l))
	}
	if !l.imSichtfeld(se.fw.hex.rahmen) {
		t.Errorf("Hex %v nicht im Bild %v (Versatz %d)", se.fw.hex.rahmen, roll.sicht, roll.rollen)
	}
	l.r.MoveFocus(key.FocusForward)
	l.setle()
	if !l.fokus(&se.fw.deckkraft) {
		t.Fatalf("Tab nach Hex: Fokus nicht auf der Deckkraft (%s)", menueFokusName(l))
	}
	if !l.imSichtfeld(se.fw.deckkraft.rahmen) {
		t.Errorf("Deckkraft %v nicht im Bild %v (Versatz %d)", se.fw.deckkraft.rahmen, roll.sicht, roll.rollen)
	}
	if got := l.e.eingaben(); got[0] != &se.fw.hex || got[1] != &se.fw.deckkraft || got[2] != &l.e.ebenen.deck {
		t.Errorf("Reihenfolge der Eingabefelder geändert: %v", got)
	}
	l.fokusiere(&se.paar[paarVorn])
	if !l.imSichtfeld(se.paarRahmen[paarVorn]) {
		t.Errorf("Farbpaar %v nicht im Bild %v (Versatz %d)", se.paarRahmen[paarVorn], roll.sicht, roll.rollen)
	}
}

// Klicks und Eingaben treffen im gerollten Bild das Feld, das man sieht (Zeigerlage und Zeichnung stimmen überein).
func TestKlickImGerolltenBild(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 330)
	l.setle()
	se, roll := &l.e.seite, &l.e.seite.roll
	l.rolle(1 << 20)
	if roll.rollen == 0 {
		t.Fatal("Kontrolle: nicht gerollt")
	}
	l.klick(mitte(se.fw.hex.rahmen).Sub(image.Pt(0, roll.rollen)))
	if !l.fokus(&se.fw.hex) {
		t.Fatalf("Klick auf das sichtbare Hex-Feld: Fokus woanders (%s)", menueFokusName(l))
	}
	l.klick(mitte(se.fw.deckkraft.rahmen).Sub(image.Pt(0, roll.rollen)))
	if !l.fokus(&se.fw.deckkraft) {
		t.Fatalf("Klick auf das sichtbare Deckkraft-Feld: Fokus woanders (%s)", menueFokusName(l))
	}
	// Ein Klick auf die Stelle, an der das Feld ohne Rollen läge, trifft es nicht mehr.
	l.klick(image.Pt(se.fw.hex.rahmen.Min.X+4, se.fw.hex.rahmen.Min.Y+4))
	if l.fokus(&se.fw.hex) {
		t.Error("Klick an der ungerollten Lage trifft das Hex-Feld")
	}
}

// Ebenen-Vorrang (ADR-0032 Punkt 6) bleibt im niedrigen Fenster: Klick auf „Ebenen“ öffnet sie und klappt die Farbe ein,
// die Knöpfe der Ebenen sind erreichbar; Klick auf „Farbe“ gibt den Vorrang zurück. Die Farbe klappt nie von selbst ein.
func TestEbenenVorrangImNiedrigenFenster(t *testing.T) {
	l, _ := ebenenLauf(t)
	l.fenst = image.Pt(1200, 330)
	l.setle()
	eb, se, roll := &l.e.ebenen, &l.e.seite, &l.e.seite.roll
	if !se.farbeGezeigt || eb.gezeigt {
		t.Fatalf("Ausgang: Farbe %v, Ebenen %v, Soll Farbe offen und Ebenen zu", se.farbeGezeigt, eb.gezeigt)
	}
	l.fokusiere(&eb.kopf)
	l.taste(key.NameReturn, 0)
	if !eb.gezeigt || se.farbeGezeigt {
		t.Fatalf("Vorrang Ebenen: Ebenen %v, Farbe %v", eb.gezeigt, se.farbeGezeigt)
	}
	l.rolle(1 << 20)
	if !l.imSichtfeld(eb.knopfRahmen[ebKnopfLoeschen]) {
		t.Errorf("Löschen-Knopf %v nicht im Bild %v (Versatz %d, größter %d)", eb.knopfRahmen[ebKnopfLoeschen], roll.sicht,
			roll.rollen, roll.rollMax)
	}
	l.fokusiere(&se.farbeKopf)
	l.taste(key.NameReturn, 0)
	if !se.farbeGezeigt || eb.gezeigt {
		t.Errorf("Vorrang Farbe: Farbe %v, Ebenen %v", se.farbeGezeigt, eb.gezeigt)
	}
}

// Zeichnet das niedrige, gerollte Fenster ohne Fenster: Das Hex-Feld steht an der gerollten Stelle (Grund des Felds),
// unter der Fußzeile bleibt der Raum der Leiste leer.
func TestRollenZeichnetImNiedrigenFenster(t *testing.T) {
	l, _ := ebenenLauf(t)
	groesse := image.Pt(1200, 330)
	l.fenst = groesse
	l.setle()
	l.rolle(1 << 20)
	se, roll := &l.e.seite, &l.e.seite.roll
	if roll.rollen == 0 {
		t.Fatal("Kontrolle: nicht gerollt")
	}
	b := headlessBild(t, groesse, func(gtx layout.Context) { l.e.zeichneSeitenleiste(gtx) })
	r := se.fw.hex.rahmen.Sub(image.Pt(0, roll.rollen))
	p := b.RGBAAt(r.Min.X+3, r.Max.Y-3)
	if want := farbeGrund; p.R != want.R || p.G != want.G || p.B != want.B || p.A != 255 {
		t.Errorf("Pixel im Hex-Feld bei %v: %v, Soll Feldgrund %v", r, p, want)
	}
	if u := b.RGBAAt(r.Min.X+3, groesse.Y-fusszeileDp+3); u.A != 0 {
		t.Errorf("Pixel unter der Leiste (Fußzeile): %v, Soll leer", u)
	}
	if n := zaehleFarbe(b, farbeLinie); n == 0 {
		t.Error("kein Rollbalken gezeichnet")
	}
}
