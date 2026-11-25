// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// frei-Tests (F-805): Die Objektebene hält bis rund 262 MiB Bild; jede der drei Stellen, die sie freigibt, hat ihren
// eigenen Test (Mutanten FR1 bis FR3, scripts/mutanten/objektebene-frei.json).

func objEbeneLeer(o *objektEbene) bool {
	return !o.an && o.dok == nil && o.objekte == nil && o.bild == (paint.ImageOp{})
}

func freiTestDokument(t *testing.T) *document.Dokument {
	t.Helper()
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 150)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(80, 60)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 3}}); err != nil {
		t.Fatal(err)
	}
	return d
}

// FR1: Zeichnet der Editor unter der Bereichsgrenze, gibt er eine zuvor gebaute Ebene frei.
func TestZeichnenUnterGrenzeGibtObjektebeneFrei(t *testing.T) {
	d := freiTestDokument(t)
	e := &editor{s: canvas.NeueSitzung(d)}
	if !e.objEbene.baue(e) || objEbeneLeer(&e.objEbene) {
		t.Fatal("Ausgangslage: Objektebene nicht gebaut")
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(90, 70), document.Pt(150, 120)),
		Strich: document.Strich{Farbe: color.NRGBA{B: 255, A: 255}, Breite: 3}}); err != nil {
		t.Fatal(err)
	}
	headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
	if e.bereiche > bereichGrenze {
		t.Fatalf("Ausgangslage: %d Bereiche über der Grenze", e.bereiche)
	}
	if !objEbeneLeer(&e.objEbene) {
		t.Errorf("Ebene bleibt nach dem Zeichnen unter der Grenze liegen: %+v", e.objEbene)
	}
}

// FR2: Der Wechsel der Aufnahme (uebernimm) gibt die Ebene der alten Aufnahme frei.
func TestUebernimmGibtObjektebeneFrei(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	if !l.e.objEbene.baue(l.e) || objEbeneLeer(&l.e.objEbene) {
		t.Fatal("Ausgangslage: Objektebene nicht gebaut")
	}
	l.e.uebernimm(Geoeffnet{Titel: "neu.png", Dokument: freiTestDokument(t), Rettung: &leistenRettung{}})
	if !objEbeneLeer(&l.e.objEbene) {
		t.Errorf("Ebene der alten Aufnahme bleibt liegen: %+v", l.e.objEbene)
	}
}

// FR3: Scheitert der Renderer beim Aufbau, bleibt keine alte Ebene liegen und der Editor zeichnet selbst.
func TestBaueFehlerGibtObjektebeneFrei(t *testing.T) {
	d := freiTestDokument(t)
	e := &editor{s: canvas.NeueSitzung(d)}
	if !e.objEbene.baue(e) || objEbeneLeer(&e.objEbene) {
		t.Fatal("Ausgangslage: Objektebene nicht gebaut")
	}
	alt := objektebeneRendern
	t.Cleanup(func() { objektebeneRendern = alt })
	objektebeneRendern = func(*document.Dokument) (*image.NRGBA, error) { return nil, errors.New("Attrappe") }
	if e.objEbene.baue(e) {
		t.Fatal("baue meldet Erfolg trotz Renderfehler")
	}
	if !objEbeneLeer(&e.objEbene) {
		t.Errorf("Ebene bleibt nach dem Fehler liegen: %+v", e.objEbene)
	}
}
