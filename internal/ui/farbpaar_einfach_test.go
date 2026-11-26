// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #359 (ADR-0021 Nachtrag 23): Ohne Füllung zeigt das Farbpaar nur das eine Farbfeld, „keine Füllung“ ist ein
// Farbfeld mit Schrägstrich, das die Füllung aktiv und durchsichtig macht, und neben dem Paar steht der Name des
// aktiven Felds.

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

func TestFarbpaarZeileOhneFuellungNurEinFeld(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	// Vorrichtung: ein Text mit Schild ist gewählt — alle vier Felder.
	for _, i := range []int{paarVorn, paarFuellung, paarTausch, paarKeine} {
		if k.paarRahmen[i].Empty() {
			t.Fatalf("mit Füllung: Feld %d fehlt", i)
		}
	}
	if k.paarRahmen[paarKeine].Dx() != k.paarRahmen[paarVorn].Dx() {
		t.Errorf("„keine“ %v ist kein Farbfeld wie %v", k.paarRahmen[paarKeine], k.paarRahmen[paarVorn])
	}
	// Pfeil ohne Auswahl: nur das Farbfeld.
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.setle()
	if k.paarRahmen[paarVorn].Empty() {
		t.Fatal("ohne Füllung: Farbfeld fehlt")
	}
	for _, i := range []int{paarFuellung, paarTausch, paarKeine} {
		if !k.paarRahmen[i].Empty() {
			t.Errorf("ohne Füllung: Feld %d bei %v", i, k.paarRahmen[i])
		}
	}
}

func TestFarbpaarKeineMachtFuellungAktivUndDurchsichtig(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	k := &l.e.seite
	l.klick(mittenPunkt(k.paarRahmen[paarVorn]))
	if l.e.s.AktivesFeld() != canvas.FeldVordergrund {
		t.Fatalf("Vorbedingung: Feld %v", l.e.s.AktivesFeld())
	}
	schrift := l.text().Farbe
	l.klick(mittenPunkt(k.paarRahmen[paarKeine]))
	if l.e.s.AktivesFeld() != canvas.FeldFuellung {
		t.Errorf("nach „keine“: Feld %v, Soll Füllung", l.e.s.AktivesFeld())
	}
	if tx := l.text(); tx.Hintergrund.A != 0 || tx.Farbe != schrift {
		t.Errorf("nach „keine“: Schild %v, Schrift %v (vorher %v)", tx.Hintergrund, tx.Farbe, schrift)
	}
}

func TestFarbfeldName(t *testing.T) {
	rot := color.NRGBA{R: 200, A: 255}
	for _, f := range []struct {
		fuellbar bool
		aktiv    canvas.Farbfeld
		fuellung color.NRGBA
		soll     string
	}{
		{false, canvas.FeldVordergrund, rot, "Color"},
		{false, canvas.FeldFuellung, rot, "Color"},
		{true, canvas.FeldVordergrund, rot, "Color"},
		{true, canvas.FeldFuellung, rot, "Fill"},
		{true, canvas.FeldFuellung, color.NRGBA{R: 200}, "Fill: none"},
	} {
		if got := farbfeldName(f.fuellbar, f.aktiv, f.fuellung); got != f.soll {
			t.Errorf("fuellbar %v, aktiv %v, A %d: %q, Soll %q", f.fuellbar, f.aktiv, f.fuellung.A, got, f.soll)
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
