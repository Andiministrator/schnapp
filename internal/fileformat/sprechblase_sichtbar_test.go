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

// TestSprechblaseImStreifenNurDreieck (F-735): Liegt das Schild einer Sprechblase ganz in einem Stauch-Streifen und
// ragt nur die Spitze heraus, zeichnet der Renderer nur das Dreieck bzw. die Schenkel. Vorher setzte er die Zeilen starr
// ab dem Anker auf der Naht, der weggestauchte Text stand lesbar dahinter im Bild (PNG, JPG, Zwischenablage, OCR).
func TestSprechblaseImStreifenNurDreieck(t *testing.T) {
	rot := color.NRGBA{255, 0, 0, 255}
	for _, f := range []struct {
		name   string
		zeilen bool
		sp     document.Punkt
		fuell  color.NRGBA
	}{
		{"Spaltenstreifen gefüllt", false, document.Pt(270, 95), color.NRGBA{30, 90, 200, 255}},
		{"Zeilenstreifen gefüllt", true, document.Pt(140, 190), color.NRGBA{30, 90, 200, 255}},
		{"Spaltenstreifen ohne Füllung", false, document.Pt(270, 95), color.NRGBA{}},
		{"Zeilenstreifen ohne Füllung", true, document.Pt(140, 190), color.NRGBA{}},
	} {
		d, err := document.Neu(sichtbarBild(300, 200))
		if err != nil {
			t.Fatal(err)
		}
		sp := f.sp
		tx := document.Text{Ort: document.Pt(100.3, 80.6), Inhalt: "GEHEIM-4711\nzwei", Schriftgrad: 14, Farbe: rot,
			Hintergrund: f.fuell, Spitze: &sp}
		if _, err := d.Hinzufuegen(tx); err != nil {
			t.Fatal(err)
		}
		g := tx.Grenzen()
		st := document.Streifen{Von: int(math.Floor(g.Min.X)) - 6, Bis: int(math.Ceil(g.Max.X)) + 6}
		if f.zeilen {
			st = document.Streifen{Zeilen: true, Von: int(math.Floor(g.Min.Y)) - 6, Bis: int(math.Ceil(g.Max.Y)) + 6}
		}
		if err := d.SetzeStreifen([]document.Streifen{st}); err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		if !d.NurSpitzeSichtbar(tx) {
			t.Fatalf("%s: Aufbau: Schrift soll unsichtbar sein", f.name)
		}
		b, err := render.Bild(d)
		if err != nil {
			t.Fatal(err)
		}
		// Ohne Objekt: Vergleichsbild. Jedes geänderte Pixel muss beim abgebildeten Dreieck liegen.
		leer, err := document.Neu(sichtbarBild(300, 200))
		if err != nil {
			t.Fatal(err)
		}
		if err := leer.SetzeStreifen([]document.Streifen{st}); err != nil {
			t.Fatal(err)
		}
		ohne, err := render.Bild(leer)
		if err != nil {
			t.Fatal(err)
		}
		a, s, c, _ := tx.Spitzendreieck()
		va, vs, vc := d.ZuAnsicht(a), d.ZuAnsicht(s), d.ZuAnsicht(c)
		rand := 2 + tx.SchenkelBreite()
		box := image.Rect(int(math.Floor(min(va.X, vs.X, vc.X)-rand)), int(math.Floor(min(va.Y, vs.Y, vc.Y)-rand)),
			int(math.Ceil(max(va.X, vs.X, vc.X)+rand)), int(math.Ceil(max(va.Y, vs.Y, vc.Y)+rand)))
		aussen, schrift := 0, 0
		for y := b.Bounds().Min.Y; y < b.Bounds().Max.Y; y++ {
			for x := b.Bounds().Min.X; x < b.Bounds().Max.X; x++ {
				p, q := b.NRGBAAt(x, y), ohne.NRGBAAt(x, y)
				if p == q {
					continue
				}
				if !image.Pt(x, y).In(box) {
					aussen++
				}
				if f.fuell.A > 0 && p.R > 200 && p.G < 80 && p.B < 80 {
					schrift++
				}
			}
		}
		if aussen > 0 || schrift > 0 {
			t.Errorf("%s: %d geänderte Pixel außerhalb des Dreiecks, %d in Schriftfarbe (Text hinter der Naht)", f.name, aussen, schrift)
		}
	}
}
