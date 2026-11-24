// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/library"
)

// monatsEintraege liefert je Monat die genannte Zahl Aufnahmen, neueste zuerst (November 2026 abwärts), in UTC.
func monatsEintraege(je ...int) []library.Eintrag {
	var es []library.Eintrag
	for k, n := range je {
		monat := time.Date(2026, time.November-time.Month(k), 20, 12, 0, 0, 0, time.UTC)
		for i := range n {
			es = append(es, library.Eintrag{Pfad: fmt.Sprintf("%s/b%02d.png", monat.Format("2006-01"), i),
				Erstellt: monat.Add(-time.Duration(i) * time.Hour), Breite: 10, Hoehe: 10})
		}
	}
	return es
}

// rm: drei Spalten (Rand 10, Kachel 100, Lücke 10 → 10 + 3·100 + 2·10 + 10 = 340), Zeile 80 + 20, Kopf 30, Oben 5,
// sichtbar 250 hoch.
var rm = RasterMasse{Rand: 10, Oben: 5, KachelB: 100, KachelH: 80, LueckeX: 10, LueckeY: 20, Kopf: 30, Breite: 345, Hoehe: 250}

// ADR-0036 Punkt 3: Monate in Reihenfolge, je mit der Zahl der Aufnahmen; die Zone zählt.
func TestRasterGruppen(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(4, 2))
	g := r.Gruppen()
	if len(g) != 2 || g[0].Monat != time.November || g[0].Anzahl() != 4 || g[1].Monat != time.October ||
		g[1].Von != 4 || g[1].Bis != 6 || g[0].Jahr != 2026 {
		t.Fatalf("Gruppen %+v", g)
	}
	// Kurz nach Mitternacht am 1. in UTC ist es in einer Zone westlich noch der Vormonat.
	es := []library.Eintrag{{Pfad: "2026-11/x.png", Erstellt: time.Date(2026, 11, 1, 0, 30, 0, 0, time.UTC)}}
	r = NeuesRaster(time.FixedZone("W", -3600))
	r.Setze(es)
	if r.Gruppen()[0].Monat != time.October {
		t.Errorf("Zone nicht beachtet: %v", r.Gruppen()[0].Monat)
	}
	if NeuesRaster(nil).zone != time.Local {
		t.Error("ohne Zone: lokale Zeit")
	}
}

func TestRasterSpalten(t *testing.T) {
	if n := Spalten(rm); n != 3 {
		t.Errorf("Spalten %d", n)
	}
	m := rm
	m.Breite = 50
	if n := Spalten(m); n != 1 {
		t.Errorf("schmal: %d", n)
	}
	m.Breite = 10 + 4*100 + 3*10 + 10
	if n := Spalten(m); n != 4 {
		t.Errorf("genau vier: %d", n)
	}
}

// Lage: Überschrift, dann Zeilen zu drei; der neue Monat beginnt mit eigener Überschrift in einer neuen Zeile.
func TestRasterLage(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(4, 2))
	if y := r.KopfLage(0, rm); y != 5 {
		t.Errorf("erste Überschrift bei %d", y)
	}
	for _, f := range []struct {
		i    int
		soll image.Rectangle
	}{
		{0, image.Rect(10, 35, 110, 115)},
		{2, image.Rect(230, 35, 330, 115)},
		{3, image.Rect(10, 135, 110, 215)}, // zweite Zeile
		{4, image.Rect(10, 265, 110, 345)}, // Oktober: Überschrift bei 235, Kacheln bei 265
		{5, image.Rect(120, 265, 220, 345)},
	} {
		if got := r.Lage(f.i, rm); got != f.soll {
			t.Errorf("Kachel %d: %v, Soll %v", f.i, got, f.soll)
		}
	}
	if y := r.KopfLage(1, rm); y != 235 {
		t.Errorf("zweite Überschrift bei %d", y)
	}
	if h := r.Inhalt(rm); h != 365 {
		t.Errorf("Inhalt %d", h)
	}
	if r.Lage(6, rm) != (image.Rectangle{}) || r.Lage(-1, rm) != (image.Rectangle{}) {
		t.Error("Lage außerhalb")
	}
	// Gerollt verschiebt sich alles nach oben.
	r.SetzeRollen(40)
	if got := r.Lage(3, rm); got.Min.Y != 95 || r.KopfLage(0, rm) != -35 {
		t.Errorf("gerollt: %v, Kopf %d", got, r.KopfLage(0, rm))
	}
}

// Treffer: Kachel unter dem Zeiger, nichts in Lücken, auf Überschriften und hinter der letzten Kachel einer Zeile.
func TestRasterTreffer(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(4, 2))
	for _, f := range []struct {
		p    image.Point
		soll int
	}{
		{image.Pt(10, 35), 0}, {image.Pt(109, 114), 0}, {image.Pt(110, 50), -1}, {image.Pt(125, 50), 1},
		{image.Pt(50, 20), -1}, {image.Pt(50, 120), -1}, {image.Pt(125, 150), -1}, {image.Pt(15, 150), 3},
		{image.Pt(5, 50), -1}, {image.Pt(125, 240), -1}, {image.Pt(-1, 50), -1}, {image.Pt(50, 250), -1},
		{image.Pt(340, 50), -1},
	} {
		if got := r.Treffer(f.p, rm); got != f.soll {
			t.Errorf("Treffer %v: %d, Soll %d", f.p, got, f.soll)
		}
	}
	r.SetzeRollen(100)
	if got := r.Treffer(image.Pt(15, 170), rm); got != 4 { // 170+100 = 270: Oktober, erste Kachel
		t.Errorf("gerollt: %d", got)
	}
}

// ADR-0036 Punkt 4: Pfeile wandern im Raster, hoch und runter über Monatsgrenzen in derselben Spalte (oder auf die
// letzte Kachel einer kürzeren Zeile), Pos1/Ende springen, Bild hoch/runter blättern eine sichtbare Höhe.
func TestRasterBewege(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(7, 5)) // Nov: 0 1 2 / 3 4 5 / 6;  Okt: 7 8 9 / 10 11
	for _, f := range []struct {
		von  int
		s    Schritt
		soll int
	}{
		{0, SchrittLinks, 0}, {1, SchrittLinks, 0}, {2, SchrittRechts, 3}, {11, SchrittRechts, 11},
		{0, SchrittHoch, 0}, {1, SchrittRunter, 4}, {4, SchrittRunter, 6}, {6, SchrittRunter, 7}, {5, SchrittRunter, 6},
		{8, SchrittHoch, 6}, {7, SchrittHoch, 6}, {9, SchrittRunter, 11}, {11, SchrittRunter, 11},
		{4, SchrittAnfang, 0}, {4, SchrittEnde, 11},
		{1, SchrittSeiteRunter, 6}, {3, SchrittSeiteRunter, 7}, {10, SchrittSeiteHoch, 6}, {11, SchrittSeiteRunter, 11}, {2, SchrittSeiteHoch, 2},
		{-1, SchrittRunter, 0}, {99, SchrittLinks, 0},
	} {
		if got := r.Bewege(f.von, f.s, rm); got != f.soll {
			t.Errorf("von %d mit %d: %d, Soll %d", f.von, f.s, got, f.soll)
		}
	}
	if NeuesRaster(time.UTC).Bewege(0, SchrittRechts, rm) != -1 {
		t.Error("leer: -1")
	}
	// Ganz kleine Fläche: blättern mindestens eine Zeile.
	m := rm
	m.Hoehe = 10
	if got := r.Bewege(1, SchrittSeiteRunter, m); got != 4 {
		t.Errorf("Seite in kleiner Fläche: %d", got)
	}
}

// Sichtbar: nur sichtbare Zeilen und je eine Zeile Vorlauf (ADR-0036 Punkt 6).
func TestRasterSichtbar(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(30)) // zehn Zeilen à 100, ab 35
	von, bis := r.Sichtbar(rm)   // sichtbar 0..250: Zeilen 0, 1, 2 (bis 315 angeschnitten nicht), Vorlauf Zeile 3
	if von != 0 || bis != 12 {
		t.Errorf("oben: %d bis %d", von, bis)
	}
	r.SetzeRollen(500) // sichtbar 500..750: Zeilen 4 (435..515) bis 7 (735), Vorlauf 3 und 8
	if von, bis := r.Sichtbar(rm); von != 9 || bis != 27 {
		t.Errorf("Mitte: %d bis %d", von, bis)
	}
	r.SetzeRollen(1 << 20) // begrenzt auf RollMax = 1035 - 250 = 785
	if r.Rollen(rm) != 785 {
		t.Errorf("RollMax %d", r.Rollen(rm))
	}
	if _, bis := r.Sichtbar(rm); bis != 30 {
		t.Errorf("unten: bis %d", bis)
	}
	if von, bis := NeuesRaster(time.UTC).Sichtbar(rm); von != 0 || bis != 0 {
		t.Error("leer")
	}
	r2 := NeuesRaster(time.UTC)
	r2.Setze(monatsEintraege(3, 3, 3, 3))
	r2.SetzeRollen(200)
	if von, bis := r2.SichtbareGruppen(rm); von != 2 || bis != 4 { // Köpfe bei 5, 135, 265, 395
		t.Errorf("sichtbare Monate %d bis %d", von, bis)
	}
	if von, bis := NeuesRaster(time.UTC).SichtbareGruppen(rm); von != 0 || bis != 0 {
		t.Error("leer: keine Monate")
	}
}

// Zeige rollt minimal; die erste Zeile eines Monats bringt dessen Überschrift mit.
func TestRasterZeige(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(30))
	r.Zeige(29, rm) // letzte Zeile 935..1015
	if ro := r.Rollen(rm); ro != 1015+20-250 || ro != r.RollMax(rm) {
		t.Errorf("nach unten: %d", ro)
	}
	r.Zeige(15, rm) // Zeile 5 bei 535: oben bündig
	if ro := r.Rollen(rm); ro != 535 {
		t.Errorf("nach oben: %d", ro)
	}
	r.Zeige(16, rm) // schon sichtbar: bleibt
	if ro := r.Rollen(rm); ro != 535 {
		t.Errorf("sichtbar: %d", ro)
	}
	r.Zeige(1, rm) // erste Zeile: samt Überschrift bei 5 und dem Rand darüber
	if ro := r.Rollen(rm); ro != 0 {
		t.Errorf("mit Überschrift: %d", ro)
	}
	r.Zeige(-1, rm)
	r.Zeige(30, rm)
	if ro := r.Rollen(rm); ro != 0 {
		t.Errorf("außerhalb: %d", ro)
	}
	m := rm
	m.Hoehe = 50 // Kachel höher als die Fläche: ihre Oberkante gilt
	r.Zeige(29, m)
	if ro := r.Rollen(m); ro != 935 {
		t.Errorf("hohe Kachel: %d", ro)
	}
}

// Mausrad mit Rest unter einem Pixel; am Anschlag verfällt er.
func TestRasterRad(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(30))
	r.Rad(0.6, rm)
	r.Rad(0.6, rm)
	if r.Rollen(rm) != 1 {
		t.Errorf("Rest: %d", r.Rollen(rm))
	}
	r.Rad(-10, rm)
	r.Rad(0.5, rm)
	if r.Rollen(rm) != 0 {
		t.Errorf("Anschlag oben: %d", r.Rollen(rm))
	}
	r.Rad(0.6, rm)
	if r.Rollen(rm) != 1 {
		t.Errorf("Rest nach dem Anschlag: %d", r.Rollen(rm))
	}
	r.Rad(1e6, rm)
	if r.Rollen(rm) != r.RollMax(rm) {
		t.Errorf("Anschlag unten: %d", r.Rollen(rm))
	}
}

// Der Fokus bleibt beim Neulesen bei derselben Aufnahme; fällt sie weg (Papierkorb), rückt die nächste an ihre Stelle.
func TestRasterFokusBleibt(t *testing.T) {
	r := NeuesRaster(time.UTC)
	es := monatsEintraege(6)
	r.Setze(es[1:])
	r.SetzeFokus(2) // b03
	r.Setze(es)
	if r.Fokus() != 3 {
		t.Errorf("Fokus nach neuer Aufnahme: %d", r.Fokus())
	}
	r.Setze(append(append([]library.Eintrag(nil), es[:3]...), es[4:]...))
	if r.Fokus() != 3 || r.Eintraege()[3].Pfad != es[4].Pfad {
		t.Errorf("Fokus nach Wegfall: %d", r.Fokus())
	}
	r.SetzeFokus(4)
	r.Setze(es[:2])
	if r.Fokus() != 1 {
		t.Errorf("Fokus nach Kürzen: %d", r.Fokus())
	}
	r.Setze(nil)
	if r.Fokus() != -1 {
		t.Errorf("leer: %d", r.Fokus())
	}
	r.Setze(es)
	if r.Fokus() != -1 {
		t.Error("ohne Fokus bleibt ohne")
	}
	r.SetzeFokus(9)
	if r.Fokus() != -1 {
		t.Error("Fokus außerhalb")
	}
	r.SetzeAktuell(es[2].Pfad)
	if !r.IstAktuell(2) || r.IstAktuell(1) || r.IstAktuell(-1) || r.Stelle(es[2].Pfad) != 2 || r.Stelle("") != -1 {
		t.Error("offene Aufnahme")
	}
	r.SetzeAktuell("")
	if r.IstAktuell(2) {
		t.Error("ohne offene")
	}
}

// Nach Setze stimmt die Anordnung zu den neuen Einträgen (kein alter Zwischenstand).
func TestRasterNeuAngeordnet(t *testing.T) {
	r := NeuesRaster(time.UTC)
	r.Setze(monatsEintraege(30))
	_ = r.Inhalt(rm)
	r.Setze(monatsEintraege(2))
	if h := r.Inhalt(rm); h != 135 {
		t.Errorf("Inhalt nach Setze: %d", h)
	}
	r.Setze(nil)
	if h := r.Inhalt(rm); h != 0 {
		t.Errorf("leer: %d", h)
	}
}
