// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #359 (ADR-0021 Nachtrag 23), umgestellt mit ADR-0031 Nachtrag 13 (Karte #369): Ohne Füllung gibt es nur den
// Reiter „Farbe“, mit Füllung dazu Tauschen und „Füllung“, bei angeschaltetem Schatten „Schatten“; „keine Füllung“
// ist ein Farbfeld mit Schrägstrich vorn in den Farbfeldern, solange die Füllung das Ziel ist.

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func TestFarbpaarZeileOhneFuellungNurEinFeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	// Vorrichtung: ein Text mit Schild ist gewählt, die Füllung ist das Ziel — Farbe, Tauschen, Füllung und „keine“.
	for _, i := range []int{paarVorn, paarFuellung, paarTausch} {
		if k.paarRahmen[i].Empty() {
			t.Fatalf("mit Füllung: Feld %d fehlt", i)
		}
	}
	if !k.paarRahmen[paarSchatten].Empty() {
		t.Errorf("Schatten %v ohne Schatten gezeigt", k.paarRahmen[paarSchatten])
	}
	// Die gezeigten Reiter füllen die Zeile bis vor die Pipette (8 dp Abstand, Rundung je Reiter höchstens 1 px): Ein
	// unsichtbarer Reiter Schatten nähme sonst Platz weg (Mutant FZ13).
	if rest := k.fw.pipRahmen.Min.X - 8 - k.paarRahmen[paarFuellung].Max.X; rest < 0 || rest > 2 {
		t.Errorf("Reiter enden bei %d, Pipette ab %d: %d px Rest", k.paarRahmen[paarFuellung].Max.X, k.fw.pipRahmen.Min.X, rest)
	}
	l.klick(mittenPunkt(k.paarRahmen[paarVorn]))
	l.setle()
	if !k.paarRahmen[paarKeine].Empty() {
		t.Errorf("„keine“ %v mit der Farbe als Ziel", k.paarRahmen[paarKeine])
	}
	l.klick(mittenPunkt(k.paarRahmen[paarFuellung]))
	l.setle()
	if k.paarRahmen[paarKeine].Empty() || k.paarRahmen[paarKeine].Dx() != k.ff.rahmen[0].Dx() ||
		k.paarRahmen[paarKeine].Max.X >= k.ff.rahmen[0].Min.X {
		t.Errorf("„keine“ %v steht nicht vorn in den Farbfeldern (erstes %v)", k.paarRahmen[paarKeine], k.ff.rahmen[0])
	}
	// Pfeil ohne Auswahl: nur der Reiter Farbe.
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	if k.paarRahmen[paarVorn].Empty() {
		t.Fatal("ohne Füllung: Reiter Farbe fehlt")
	}
	if rest := k.fw.pipRahmen.Min.X - 8 - k.paarRahmen[paarVorn].Max.X; rest < 0 || rest > 1 {
		t.Errorf("ohne Füllung endet der Reiter Farbe bei %d, Pipette ab %d (Mutant F359-1)", k.paarRahmen[paarVorn].Max.X,
			k.fw.pipRahmen.Min.X)
	}
	for _, i := range []int{paarFuellung, paarTausch, paarKeine, paarSchatten} {
		if !k.paarRahmen[i].Empty() {
			t.Errorf("ohne Füllung: Feld %d bei %v", i, k.paarRahmen[i])
		}
	}
}

func TestFarbpaarKeineMachtFuellungDurchsichtig(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	schrift := l.text().Farbe
	l.klick(mittenPunkt(k.paarRahmen[paarFuellung]))
	l.setle()
	l.klick(mittenPunkt(k.paarRahmen[paarKeine]))
	if l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Errorf("nach „keine“: Feld %v, Soll Füllung", l.e.s.AktivesFeld())
	}
	if tx := l.text(); tx.Hintergrund.A != 0 || tx.Farbe != schrift {
		t.Errorf("nach „keine“: Schild %v, Schrift %v (vorher %v)", tx.Hintergrund, tx.Farbe, schrift)
	}
}

// ADR-0031 Nachtrag 13 Punkt 1: Mit Schatten gibt es den Reiter „Schatten“; ein Klick darauf macht den Schatten zum
// Ziel des Farbwählers, und die Farbwahl färbt ihn, nicht den Strich.
func TestFarbReiterSchatten(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 1000))
	id, err := l.e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(30, 20)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 200, A: 255}, Breite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	k := &l.e.seite
	if !k.paarRahmen[paarSchatten].Empty() {
		t.Fatal("Reiter Schatten ohne Schatten")
	}
	l.e.s.SchalteSchatten()
	l.setle()
	if k.paarRahmen[paarSchatten].Empty() {
		t.Fatal("Reiter Schatten fehlt")
	}
	l.klick(mittenPunkt(k.paarRahmen[paarSchatten]))
	if l.e.s.AktivesFeld() != canvas.FeldSchatten {
		t.Fatalf("Feld %v, Soll Schatten", l.e.s.AktivesFeld())
	}
	gruen := color.NRGBA{G: 180, A: 90}
	l.e.s.SetzeFarbe(gruen)
	o, _ := l.e.s.D.Objekt(id)
	if o.Ebenenwerte().Schatten.Farbe != gruen || o.(document.Kasten).Strich.Farbe == gruen {
		t.Errorf("Schatten %v, Strich %v", o.Ebenenwerte().Schatten.Farbe, o.(document.Kasten).Strich.Farbe)
	}
	l.klick(mittenPunkt(k.paarRahmen[paarVorn]))
	if l.e.s.AktivesFeld() != canvas.FeldVordergrund {
		t.Errorf("nach Reiter Farbe: Feld %v", l.e.s.AktivesFeld())
	}
}

func TestFarbReiterName(t *testing.T) {
	for f, soll := range map[canvas.Farbfeld]string{canvas.FeldVordergrund: "Color", canvas.FeldFuellung: "Fill",
		canvas.FeldSchatten: "Shadow"} {
		if got := farbReiterName(f); got != soll {
			t.Errorf("%v: %q, Soll %q", f, got, soll)
		}
	}
}

// Unter der Werkzeugleiste: Ohne Füllung sind Füllfeld und ⇄ nicht angemeldet — der Zeiger darüber zeigt keinen
// Hinweis und trifft keinen Knopf.
func TestFarbpaarLinksOhneFuellungNichtAngemeldet(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	_, farbe := werkzeugRahmen(l.gtx(), l.e.werkzeugKnoepfe())
	_, fuel, tausch := farbpaarRahmen(farbe)
	l.maus(pointer.Move, mittenPunkt(fuel), 0)
	l.setle()
	if !l.e.knoepfe.fuellung.Hovered() {
		t.Fatal("Vorbedingung: mit Füllung ist das Füllfeld unter dem Zeiger")
	}
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	for _, p := range []image.Point{mittenPunkt(fuel), mittenPunkt(tausch)} {
		l.maus(pointer.Move, p, 0)
		l.setle()
		if l.e.knoepfe.fuellung.Hovered() || l.e.knoepfe.tausch.Hovered() {
			t.Errorf("Pfeil: Zeiger bei %v trifft Füllung %v / Tauschen %v", p, l.e.knoepfe.fuellung.Hovered(),
				l.e.knoepfe.tausch.Hovered())
		}
	}
}

// Die Reiter bekommen, was ihr Name braucht, und teilen den Rest gleich; reicht der Platz nicht, sind sie gleich breit.
func TestFarbReiterBreiten(t *testing.T) {
	felder := []canvas.Farbfeld{canvas.FeldVordergrund, canvas.FeldFuellung, canvas.FeldSchatten}
	bedarf := func(f canvas.Farbfeld) int { return []int{50, 70, 80}[f] }
	if b := farbReiterBreiten(230, bedarf, felder); b[0] != 60 || b[1] != 80 || b[2] != 90 {
		t.Errorf("mit Platz: %v, Soll [60 80 90]", b)
	}
	if b := farbReiterBreiten(150, bedarf, felder); b[0] != 50 || b[1] != 50 || b[2] != 50 {
		t.Errorf("zu eng: %v, Soll gleich breit 50", b)
	}
}

// F-879 (V-2): Hervorgehoben ist genau der Reiter des Felds, das der Farbwähler bearbeitet.
func TestFarbReiterHervorgehoben(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	for _, f := range []struct {
		platz int
		feld  canvas.Farbfeld
	}{{paarVorn, canvas.FeldVordergrund}, {paarFuellung, canvas.FeldFuellung}} {
		l.klick(mittenPunkt(k.paarRahmen[f.platz]))
		l.setle()
		if l.e.s.AktivesFeld() != f.feld || k.reiterAktiv != k.paarRahmen[f.platz] {
			t.Errorf("Feld %v: hervorgehoben %v, Soll %v", f.feld, k.reiterAktiv, k.paarRahmen[f.platz])
		}
	}
}
