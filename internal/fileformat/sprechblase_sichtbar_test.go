// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// sichtbarBild ist ein buntes Testbild (keine Fläche einer Farbe, damit ein gezeichnetes Pixel auffällt).
func sichtbarBild(b, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, b, h))
	for y := 0; y < h; y++ {
		for x := 0; x < b; x++ {
			m.SetNRGBA(x, y, color.NRGBA{uint8(40 + x%100), uint8(60 + y%100), uint8(200 - (x+y)%100), 255})
		}
	}
	return m
}

func sichtbarSVG(t *testing.T, d *document.Dokument) string {
	t.Helper()
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestSprechblaseNurSpitzeSichtbarOhneText (F-692, ADR-0021 Nachtrag 14): Ist von einer Sprechblase nur das Dreieck
// sichtbar (Schild und Glyphen weggeschnitten), steht der Text nicht im exportierten SVG; das Dreieck bleibt als Pfad.
// Kontrolle: Derselbe Text ohne Zuschnitt steht im SVG.
func TestSprechblaseNurSpitzeSichtbarOhneText(t *testing.T) {
	neu := func() (*document.Dokument, document.Text) {
		d, err := document.Neu(sichtbarBild(200, 100))
		if err != nil {
			t.Fatal(err)
		}
		sp := document.Pt(150, 80)
		tx := document.Text{Ort: document.Pt(10, 10), Inhalt: "GEHEIM-4711", Schriftgrad: 12,
			Farbe: color.NRGBA{A: 255}, Hintergrund: color.NRGBA{30, 90, 200, 255}, Spitze: &sp}
		if _, err := d.Hinzufuegen(tx); err != nil {
			t.Fatal(err)
		}
		return d, tx
	}
	d, tx := neu()
	if !strings.Contains(sichtbarSVG(t, d), "GEHEIM-4711") {
		t.Fatal("Kontrolle: sichtbarer Text fehlt im SVG")
	}
	g := tx.Grenzen()
	if err := d.SetzeZuschnitt(image.Rect(int(math.Ceil(g.Max.X))+3, 0, 200, 100)); err != nil {
		t.Fatal(err)
	}
	if schrift, dreieck := d.TextSichtbar(tx); schrift || !dreieck {
		t.Fatalf("Aufbau: TextSichtbar = (%v, %v), soll (false, true)", schrift, dreieck)
	}
	svg := sichtbarSVG(t, d)
	if strings.Contains(svg, "GEHEIM") || strings.Contains(svg, "<text") || strings.Contains(svg, "<tspan") {
		t.Errorf("Text steht im SVG, obwohl nur das Dreieck sichtbar ist:\n%s", svg)
	}
	if !strings.Contains(svg, `<path d="M`) || !strings.Contains(svg, `fill="#1e5ac8"`) {
		t.Errorf("sichtbares Dreieck fehlt im SVG:\n%s", svg)
	}
	// Nur das Dreieck, nicht der Umriss mit dem Schild: sonst stünden Breite und Höhe des unsichtbaren Schilds in der
	// Datei (F-728). Das Dreieck hat drei Ecken, also genau zwei „L“.
	if i := strings.Index(svg, `<path d="M`); i >= 0 {
		d := svg[i+len(`<path d="`):]
		d = d[:strings.Index(d, `"`)]
		if n := strings.Count(d, " L"); n != 2 {
			t.Errorf("Pfad mit %d Strecken statt des Dreiecks (Umriss des Schilds im SVG): %s", n, d)
		}
	}
	// Treffer und Auswahl rechnen weiter mit der Hülle samt Spitze (Reichweite unverändert).
	if !d.ObjektSichtbar(d.Objekte()[0]) {
		t.Error("ObjektSichtbar ist false, obwohl das Dreieck sichtbar ist")
	}
}

// TestSprechblaseNichtsSichtbarNichtImSVG (F-692): Text ohne Füllung, Spitze so nah am Schild, dass das Dreieck
// entfällt, Zuschnitt rechts vom Schild: Der Renderer zeichnet nichts vom Objekt, und das SVG enthält es nicht.
func TestSprechblaseNichtsSichtbarNichtImSVG(t *testing.T) {
	d, err := document.Neu(sichtbarBild(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	tx := document.Text{Ort: document.Pt(10, 10), Inhalt: "GEHEIM-0815", Schriftgrad: 12, Farbe: color.NRGBA{A: 255},
		Unterstrichen: true}
	g := tx.Grenzen()
	sp := document.Pt(g.Max.X+2, (g.Min.Y+g.Max.Y)/2)
	tx.Spitze = &sp
	if _, _, _, ok := tx.Spitzendreieck(); ok {
		t.Fatal("Aufbau: Dreieck sollte entfallen")
	}
	if _, err := d.Hinzufuegen(tx); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(int(math.Floor(g.Max.X+3.4)), 0, 200, 100)); err != nil {
		t.Fatal(err)
	}
	b, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Pix, d.Ansichtsbild().Pix) {
		t.Fatal("Aufbau: Der Renderer zeichnet etwas vom Objekt")
	}
	svg := sichtbarSVG(t, d)
	if strings.Contains(svg, "GEHEIM") || strings.Contains(svg, "<text") ||
		!strings.Contains(svg, `<g id="objekte"><g></g></g>`) { // Objektgruppe leer
		t.Errorf("unsichtbare Sprechblase steht im SVG:\n%s", svg)
	}
}
