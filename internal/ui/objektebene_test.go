// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// F-787 (ADR-0023 Nachtrag 20): Die Objektebene gilt nur für den Stand, für den sie gebaut wurde. Ein neuer Schritt,
// Rückgängig, ein Zug ohne Schritt (Objekt ändert sich, Stand nicht), eine neue Naht und geschwärzte Pixel machen sie
// ungültig; derselbe Stand nicht.
func TestObjektebeneGiltNurFuerIhrenStand(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 150)))
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(80, 60)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 3}})
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	neu := func(name string) {
		t.Helper()
		if e.objEbene.passt(e) {
			t.Errorf("%s: Objektebene gilt noch", name)
		}
		if !e.objEbene.baue(e) || !e.objEbene.passt(e) {
			t.Fatalf("%s: neu gebaute Objektebene gilt nicht", name)
		}
	}
	neu("Anfang")
	if !e.objEbene.passt(e) {
		t.Error("ohne Änderung gilt die Objektebene nicht mehr")
	}
	if _, err := d.Hinzufuegen(document.Nummer{Mitte: document.Pt(120, 90), Radius: 12, Wert: 1,
		Farbe: color.NRGBA{B: 255, A: 255}}); err != nil {
		t.Fatal(err)
	}
	neu("neues Objekt")
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	neu("Rückgängig")
	if err := d.BeginneZug(id); err != nil {
		t.Fatal(err)
	}
	if err := d.ZugSetze(document.Kasten{ID: id, Bereich: document.R(document.Pt(20, 20), document.Pt(90, 70)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 3}}); err != nil {
		t.Fatal(err)
	}
	neu("Zug ohne Schritt")
	if err := d.BeendeZug(); err != nil {
		t.Fatal(err)
	}
	neu("Zug beendet")
	if err := d.SetzeStreifen([]document.Streifen{{Von: 40, Bis: 50}}); err != nil {
		t.Fatal(err)
	}
	neu("neue Naht")
	if e.objEbene.gebaut != 6 {
		t.Errorf("%d Aufbauten, erwartet 6", e.objEbene.gebaut)
	}
}

// F-787: Über bereichGrenze zeigt der Editor die Objektebene des Renderers: dasselbe Bild wie render.Objektebene, auf
// die Zeichenfläche gelegt (bis auf Rundung beim Hochladen der Textur), und ein Objekt, das sich ändert, erscheint im
// nächsten Bild.
func TestEditorZeigtObjektebeneWieRenderer(t *testing.T) {
	var kacheln []document.Streifen
	for i := 0; i < 63; i++ {
		kacheln = append(kacheln, document.Streifen{Von: 4 * i, Bis: 4*i + 1},
			document.Streifen{Zeilen: true, Von: 4 * i, Bis: 4*i + 1})
	}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 256, 256)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen(kacheln); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(2, 2), document.Pt(250, 250)),
			Eckradius: 20, Fuellung: color.NRGBA{R: 200, G: 40, A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	gruen := color.NRGBA{G: 200, A: 255}
	strich := document.Strich{Farbe: gruen, Breite: 6}
	id, err := d.Hinzufuegen(document.Pfeil{A: document.Pt(20, 20), B: document.Pt(90, 60), Strich: strich})
	if err != nil {
		t.Fatal(err)
	}
	e := &editor{s: canvas.NeueSitzung(d)}
	vergleiche := func(name string) {
		t.Helper()
		bild := headlessBild(t, d.Groesse(), func(gtx layout.Context) { e.zeichneObjekte(gtx) })
		if !e.objEbene.an {
			t.Fatalf("%s: keine Objektebene bei %d Bereichen", name, e.objEbene.bereiche)
		}
		soll, err := render.Objektebene(d)
		if err != nil {
			t.Fatal(err)
		}
		abw := 0
		for y := 0; y < 256; y++ {
			for x := 0; x < 256; x++ {
				s := color.RGBAModel.Convert(soll.At(x, y)).(color.RGBA)
				if !rgbaNah(bild.RGBAAt(x, y), s) {
					abw++
				}
			}
		}
		if abw != 0 {
			t.Errorf("%s: %d Pixel weichen von der Objektebene des Renderers ab", name, abw)
		}
	}
	vergleiche("Anfang")
	if err := d.Ersetzen(document.Pfeil{ID: id, A: document.Pt(120, 200), B: document.Pt(230, 140), Strich: strich}); err != nil {
		t.Fatal(err)
	}
	vergleiche("Pfeil verschoben")
	if e.objEbene.gebaut != 2 {
		t.Errorf("%d Aufbauten, erwartet 2", e.objEbene.gebaut)
	}
}
