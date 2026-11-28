// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0045 Punkt 2: Das Bild einer Objekt-Kopie zeigt nur die gewählten Objekte, auf ihre Pixel zugeschnitten, ohne
// das Bild darunter; das Dokument bleibt unverändert.
func TestObjektePNG(t *testing.T) {
	grund := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	for i := range grund.Pix {
		grund.Pix[i] = 200 // undurchsichtiges Grau
	}
	d, err := document.Neu(grund)
	if err != nil {
		t.Fatal(err)
	}
	rot := color.NRGBA{R: 255, A: 255}
	a, _ := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(30, 20)), Fuellung: rot})
	_, _ = d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(50, 50), document.Pt(60, 60)), Fuellung: rot})
	vorher := len(d.Objekte())
	daten, err := ObjektePNG(d, []int{a})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Objekte()) != vorher {
		t.Error("Dokument verändert")
	}
	b, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if g := b.Bounds().Size(); g != image.Pt(20, 10) {
		t.Errorf("Größe %v, erwartet 20×10", g)
	}
	if _, _, _, alpha := b.At(b.Bounds().Min.X+5, b.Bounds().Min.Y+5).RGBA(); alpha == 0 {
		t.Error("Kasten fehlt")
	}

	// Nichts sichtbar (Objekt außerhalb des Zuschnitts): ein durchsichtiges Pixel.
	if err := d.SetzeZuschnitt(image.Rect(40, 40, 100, 80)); err != nil {
		t.Fatal(err)
	}
	daten, err = ObjektePNG(d, []int{a})
	if err != nil {
		t.Fatal(err)
	}
	b, _ = png.Decode(bytes.NewReader(daten))
	if g := b.Bounds().Size(); g != image.Pt(1, 1) {
		t.Errorf("Größe ohne Sichtbares %v", g)
	}
	if _, _, _, alpha := b.At(b.Bounds().Min.X, b.Bounds().Min.Y).RGBA(); alpha != 0 {
		t.Error("Pixel nicht durchsichtig")
	}
}

// ADR-0045 Punkt 3: Die Budgetprüfung des Einfügens läuft auf einem Dokument ohne Pixel (document.OhnePixel); auch
// mit Lupe, Text und Schatten liest sie keine Pixel.
func TestPruefeOhnePixel(t *testing.T) {
	d, err := document.OhnePixel(image.Pt(400, 300), image.Rect(0, 0, 400, 300), nil, document.KanteAbriss)
	if err != nil {
		t.Fatal(err)
	}
	schatten := document.Ebene{Schatten: document.StandardSchatten(1)}
	schatten.Schatten.An = true
	for _, o := range []document.Objekt{
		document.Lupe{Ebene: schatten, Quelle: document.R(document.Pt(10, 10), document.Pt(40, 40)),
			Ziel: document.R(document.Pt(100, 100), document.Pt(160, 160)), Strich: document.Strich{Breite: 2, Farbe: color.NRGBA{A: 255}}},
		document.Text{Ebene: schatten, Ort: document.Pt(50, 200), Inhalt: "Hallo", Schriftgrad: 20, Farbe: color.NRGBA{A: 255}},
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	if err := Pruefe(d); err != nil {
		t.Errorf("Pruefe: %v", err)
	}
}
