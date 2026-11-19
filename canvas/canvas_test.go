// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

func nah(a, b document.Punkt) bool { return a.Minus(b).Laenge() < 1e-9 }

func TestEinpassen(t *testing.T) {
	// Großes Bild: eingepasst mit Rand, mittig.
	a := Einpassen(image.Pt(2000, 1000), image.Pt(1040, 540), 20)
	if a.Zoom != 0.5 || !nah(a.Versatz, document.Pt(20, 20)) {
		t.Errorf("groß: %+v", a)
	}
	// Kleines Bild: nicht vergrößert, mittig.
	a = Einpassen(image.Pt(100, 50), image.Pt(1000, 500), 20)
	if a.Zoom != 1 || !nah(a.Versatz, document.Pt(450, 225)) {
		t.Errorf("klein: %+v", a)
	}
	// Leere Größen.
	if a = Einpassen(image.Point{}, image.Pt(10, 10), 0); a.Zoom != 1 || !nah(a.Versatz, document.Punkt{}) {
		t.Errorf("leer: %+v", a)
	}
	// Winziges Fenster bei riesigem Bild: Zoom nicht unter MinZoom.
	if a = Einpassen(image.Pt(1<<15, 1<<15), image.Pt(10, 10), 0); a.Zoom != MinZoom {
		t.Errorf("Untergrenze: %v", a.Zoom)
	}
}

// Zoomen hält den Punkt unter dem Zeiger fest; Umrechnung hin und zurück ist verlustfrei.
func TestZoomUmHaeltPunktFest(t *testing.T) {
	a := Abbildung{Zoom: 0.5, Versatz: document.Pt(30, 40)}
	um := document.Pt(200, 150)
	vorher := a.ZuAnsicht(um)
	for _, f := range []float64{2, 1.1, 0.25, 1e9, 1e-9} {
		b := a.ZoomUm(um, f)
		if !nah(b.ZuAnsicht(um), vorher) {
			t.Errorf("Faktor %v: %v statt %v", f, b.ZuAnsicht(um), vorher)
		}
		if b.Zoom < MinZoom || b.Zoom > MaxZoom {
			t.Errorf("Faktor %v: Zoom %v außerhalb", f, b.Zoom)
		}
	}
	for _, f := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if b := a.ZoomUm(um, f); b != a {
			t.Errorf("ungültiger Faktor %v änderte die Abbildung: %+v", f, b)
		}
	}
	if e := a.Eins(um); e.Zoom != 1 || !nah(e.ZuAnsicht(um), vorher) {
		t.Errorf("Eins: %+v", e)
	}
	p := document.Pt(12.5, -3)
	if !nah(a.ZuAnsicht(a.ZuFenster(p)), p) {
		t.Error("hin und zurück")
	}
	if v := a.Verschoben(document.Pt(5, -5)); !nah(v.Versatz, document.Pt(35, 35)) || v.Zoom != a.Zoom {
		t.Errorf("Verschoben: %+v", v)
	}
}

var rot = document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 4}

func TestTreffer(t *testing.T) {
	pfeil := document.Pfeil{ID: 1, A: document.Pt(0, 0), B: document.Pt(100, 0), K: document.Pt(50, 100), Strich: rot}
	leer := document.Kasten{ID: 2, Bereich: document.R(document.Pt(200, 200), document.Pt(300, 300)), Strich: rot}
	voll := leer
	voll.ID, voll.Fuellung = 3, color.NRGBA{A: 255}
	voll.Bereich = document.R(document.Pt(400, 400), document.Pt(500, 500))
	nummer := document.Nummer{ID: 4, Mitte: document.Pt(600, 600), Radius: 10, Wert: 1}
	oben := document.Kasten{ID: 5, Bereich: document.R(document.Pt(190, 190), document.Pt(310, 310)), Strich: rot,
		Fuellung: color.NRGBA{A: 255}}
	objekte := []document.Objekt{pfeil, leer, voll, nummer}

	for _, tc := range []struct {
		name string
		p    document.Punkt
		id   int
	}{
		{"Bogen des Pfeils (Scheitel bei y=50)", document.Pt(50, 51), 1},
		{"Sehne des gebogenen Pfeils liegt daneben", document.Pt(50, 0), 0},
		{"Spitze ragt über B hinaus", document.Pt(110, 0), 1},
		{"Rand des leeren Kastens", document.Pt(201, 250), 2},
		{"Inneres des leeren Kastens ist durchlässig", document.Pt(250, 250), 0},
		{"Inneres des gefüllten Kastens", document.Pt(450, 450), 3},
		{"Nummer", document.Pt(608, 600), 4},
		{"daneben", document.Pt(700, 700), 0},
	} {
		id, ok := Treffer(objekte, tc.p, 2)
		if (tc.id == 0) == ok || (ok && id != tc.id) {
			t.Errorf("%s: %d, %v; Soll %d", tc.name, id, ok, tc.id)
		}
	}
	// Das zuletzt eingefügte Objekt liegt oben.
	if id, _ := Treffer(append(objekte, oben), document.Pt(201, 250), 2); id != 5 {
		t.Errorf("oben: %d", id)
	}
}

func TestVerschiebe(t *testing.T) {
	d := document.Pt(10, -5)
	pf := Verschiebe(document.Pfeil{ID: 1, A: document.Pt(0, 0), B: document.Pt(10, 0), K: document.Pt(5, 5)}, d).(document.Pfeil)
	if pf.ID != 1 || !nah(pf.A, document.Pt(10, -5)) || !nah(pf.K, document.Pt(15, 0)) || !nah(pf.B, document.Pt(20, -5)) {
		t.Errorf("Pfeil: %+v", pf)
	}
	k := Verschiebe(document.Kasten{ID: 2, Bereich: document.R(document.Pt(0, 0), document.Pt(4, 4))}, d).(document.Kasten)
	if k.ID != 2 || !nah(k.Bereich.Min, d) || !nah(k.Bereich.Max, document.Pt(14, -1)) {
		t.Errorf("Kasten: %+v", k)
	}
	tx := Verschiebe(document.Text{ID: 3, Ort: document.Pt(1, 1)}, d).(document.Text)
	n := Verschiebe(document.Nummer{ID: 4, Mitte: document.Pt(1, 1)}, d).(document.Nummer)
	if !nah(tx.Ort, document.Pt(11, -4)) || !nah(n.Mitte, document.Pt(11, -4)) {
		t.Errorf("Text %+v, Nummer %+v", tx, n)
	}
}

func TestGriffePfeil(t *testing.T) {
	pf := document.GeraderPfeil(document.Pt(0, 0), document.Pt(100, 0), rot)
	pf.ID = 7
	gebogen := ZieheGriff(pf, GriffBogen, document.Pt(50, 40)).(document.Pfeil)
	if !nah(gebogen.Griff(), document.Pt(50, 40)) || gebogen.ID != 7 {
		t.Fatalf("Bogen: %+v", gebogen)
	}
	// Ende verschieben behält die Biegung relativ zur Sehnenmitte.
	verlegt := ZieheGriff(gebogen, GriffEnde, document.Pt(200, 0)).(document.Pfeil)
	if !nah(verlegt.B, document.Pt(200, 0)) || !nah(verlegt.Griff(), document.Pt(100, 40)) {
		t.Errorf("Ende: B %v, Griff %v", verlegt.B, verlegt.Griff())
	}
	anfang := ZieheGriff(gebogen, GriffAnfang, document.Pt(-100, 0)).(document.Pfeil)
	if !nah(anfang.A, document.Pt(-100, 0)) || !nah(anfang.Griff(), document.Pt(0, 40)) {
		t.Errorf("Anfang: A %v, Griff %v", anfang.A, anfang.Griff())
	}
	if art, ok := GriffTreffer(gebogen, document.Pt(51, 41), 3); !ok || art != GriffBogen {
		t.Errorf("GriffTreffer Bogen: %v %v", art, ok)
	}
	if _, ok := GriffTreffer(gebogen, document.Pt(30, 30), 3); ok {
		t.Error("GriffTreffer daneben")
	}
	// Ein Kastengriff ändert einen Pfeil nicht.
	if o := ZieheGriff(pf, GriffNW, document.Pt(1, 1)); o != document.Objekt(pf) {
		t.Errorf("fremder Griff: %+v", o)
	}
}

func TestGriffeKasten(t *testing.T) {
	k := document.Kasten{ID: 3, Bereich: document.R(document.Pt(10, 10), document.Pt(110, 60)), Strich: rot}
	if g := Griffe(k); len(g) != 8 || g[0].Art != GriffNW || !nah(g[1].Ort, document.Pt(60, 10)) || !nah(g[3].Ort, document.Pt(110, 35)) {
		t.Fatalf("Griffe: %+v", g)
	}
	for _, tc := range []struct {
		art      GriffArt
		p        document.Punkt
		min, max document.Punkt
	}{
		{GriffNW, document.Pt(0, 0), document.Pt(0, 0), document.Pt(110, 60)},
		{GriffN, document.Pt(999, 5), document.Pt(10, 5), document.Pt(110, 60)},
		{GriffNO, document.Pt(120, 0), document.Pt(10, 0), document.Pt(120, 60)},
		{GriffO, document.Pt(200, 999), document.Pt(10, 10), document.Pt(200, 60)},
		{GriffSO, document.Pt(150, 70), document.Pt(10, 10), document.Pt(150, 70)},
		{GriffS, document.Pt(0, 80), document.Pt(10, 10), document.Pt(110, 80)},
		{GriffSW, document.Pt(0, 80), document.Pt(0, 10), document.Pt(110, 80)},
		{GriffW, document.Pt(5, 0), document.Pt(5, 10), document.Pt(110, 60)},
		// Über die Gegenseite gezogen: geordnet statt negativ.
		{GriffW, document.Pt(150, 0), document.Pt(110, 10), document.Pt(150, 60)},
	} {
		got := ZieheGriff(k, tc.art, tc.p).(document.Kasten)
		if got.ID != 3 || !nah(got.Bereich.Min, tc.min) || !nah(got.Bereich.Max, tc.max) {
			t.Errorf("Griff %d nach %v: %+v, Soll %v–%v", tc.art, tc.p, got.Bereich, tc.min, tc.max)
		}
	}
	if g := Griffe(document.Text{}); g != nil {
		t.Errorf("Text hat Griffe: %v", g)
	}
}
