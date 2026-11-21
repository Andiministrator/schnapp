// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ausgaengeMitOriginal liefert alle Wege, auf denen ein Bild Schnapp zum Weitergeben verlässt, plus das sichtbare Bild
// der Arbeitsdatei: gerendertes Bild, Zwischenablage (render.PNG), PNG- und JPG-Export, jede SVG-Kachel. Dazu kommt
// das Original aus dem snAO-Block der Arbeitsdatei.
func ausgaengeMitOriginal(t *testing.T, d *document.Dokument) (ausgaenge map[string]image.Image, original image.Image) {
	t.Helper()
	sichtbar, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]image.Image{"Bild": sichtbar}
	roh, err := render.PNG(d)
	if err != nil {
		t.Fatal(err)
	}
	if m["Zwischenablage"], err = png.Decode(bytes.NewReader(roh)); err != nil {
		t.Fatal(err)
	}
	var p, j, s, a bytes.Buffer
	if err := fileformat.ExportierePNG(&p, sichtbar); err != nil {
		t.Fatal(err)
	}
	if m["PNG"], err = png.Decode(&p); err != nil {
		t.Fatal(err)
	}
	if err := fileformat.ExportiereJPG(&j, sichtbar, 95); err != nil {
		t.Fatal(err)
	}
	if m["JPG"], err = jpeg.Decode(&j); err != nil {
		t.Fatal(err)
	}
	if err := fileformat.ExportiereSVG(&s, d); err != nil {
		t.Fatal(err)
	}
	for i, k := range datenURI.FindAllStringSubmatch(s.String(), -1) {
		daten, err := base64.StdEncoding.DecodeString(k[1])
		if err != nil {
			t.Fatal(err)
		}
		if m["SVG-Kachel "+string(rune('0'+i))], err = png.Decode(bytes.NewReader(daten)); err != nil {
			t.Fatal(err)
		}
	}
	if err := fileformat.SchreibeArbeitsdatei(&a, d, sichtbar, fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if m["Arbeitsdatei sichtbar"], err = fileformat.LeseBild(bytes.NewReader(a.Bytes())); err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.LeseArbeitsdatei(bytes.NewReader(a.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return m, g.Dokument.Original()
}

// ADR-0023 Punkt 8, Kernprinzip 2: Das Muster (x 30 bis 90) wird bei x 60 zugeschnitten, dann wird die ganze Ansicht
// geschwärzt (Rahmen über das Fenster hinaus, auf die Ansicht begrenzt). Kein Ausgang enthält danach etwas vom Muster;
// das Original in der Arbeitsdatei behält genau den Teil jenseits der Kante (30 × 40 Pixel) — Zuschneiden ist kein
// Schwärzen. Gegenprobe: Vor dem Schwärzen zeigt jeder Ausgang den sichtbaren Teil des Musters.
func TestZuschnittSchwaerzenMusterJenseitsDerKante(t *testing.T) {
	d, err := document.Neu(geheimBild())
	if err != nil {
		t.Fatal(err)
	}
	s := canvas.NeueSitzung(d)
	s.SetzeFenster(image.Pt(400, 300))
	s.SetzeWerkzeug(canvas.WerkzeugZuschneiden)
	zieh := func(a, b document.Punkt) {
		s.Druecken(a, canvas.TastePrimaer)
		s.Ziehen(a.Plus(b.Minus(a).Mal(0.5)))
		s.Loslassen(b)
	}
	zieh(s.Abb.ZuFenster(document.Pt(0, 0)), s.Abb.ZuFenster(document.Pt(60, 80)))
	if z := d.Zuschnitt(); z != image.Rect(0, 0, 60, 80) {
		t.Fatalf("Zuschnitt %v", z)
	}
	// Ein Stauch-Streifen außerhalb des Musters: Nähte und Kacheln sind dabei.
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 65, Bis: 70}}); err != nil {
		t.Fatal(err)
	}
	s.Einpassen()
	vorher, _ := ausgaengeMitOriginal(t, d)
	if len(vorher) != 7 {
		t.Fatalf("%d Ausgänge, Soll 7 (zwei SVG-Kacheln)", len(vorher))
	}
	for name, b := range vorher {
		if name == "SVG-Kachel 1" {
			continue // die Kachel unter dem Streifen (Zeilen 70 bis 80) liegt unter dem Muster
		}
		if geheimePixel(b) == 0 {
			t.Errorf("Gegenprobe: %s zeigt das Muster nicht", name)
		}
	}
	s.SetzeWerkzeug(canvas.WerkzeugSchwaerzen)
	zieh(s.Abb.ZuFenster(document.Pt(0, 0)), document.Pt(1000, 1000))
	if !d.Geschwaerzt() {
		t.Fatal("nicht geschwärzt")
	}
	nachher, original := ausgaengeMitOriginal(t, d)
	for name, b := range nachher {
		if n := geheimePixel(b); n != 0 {
			t.Errorf("%s: %d Pixel des Musters", name, n)
		}
	}
	if n := geheimePixel(original); n != 30*40 {
		t.Errorf("Original der Arbeitsdatei: %d Pixel des Musters, Soll %d (jenseits der Kante)", n, 30*40)
	}
}
