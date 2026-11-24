// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Lupe in der Arbeitsdatei und im Export (ADR-0033 Punkte 5 und 6).

func lupenDok(t *testing.T) (*document.Dokument, document.Lupe) {
	t.Helper()
	d := beispielDok(t, 64, 48)
	l := document.Lupe{Ebene: document.Ebene{Name: "Zoom"}.MitDeckkraft(80), Quelle: document.R(document.Pt(4, 4), document.Pt(12, 10)),
		Ziel: document.R(document.Pt(30, 20), document.Pt(54, 38)), Strich: document.Strich{Farbe: color.NRGBA{1, 2, 3, 255}, Breite: 2}}
	fuege(t, d, l)
	l.ID = 1
	return d, l
}

func schreibeLies(t *testing.T, d *document.Dokument) (*Gelesen, string) {
	t.Helper()
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	g, err := LeseArbeitsdatei(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return g, string(entpackeSVG(t, buf.Bytes()))
}

// TestLupeVersion: Version 2 genau dann, wenn eine Lupe im Dokument steht; der Rundlauf erhält die Lupe samt Ebene.
func TestLupeVersion(t *testing.T) {
	ohne := beispielDok(t, 64, 48)
	fuege(t, ohne, document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(9, 9)), Strich: beispielRot})
	if _, svg := schreibeLies(t, ohne); !strings.Contains(svg, `s:format="1"`) {
		t.Errorf("ohne Lupe nicht Version 1: %.200s", svg)
	}
	d, l := lupenDok(t)
	g, svg := schreibeLies(t, d)
	if !strings.Contains(svg, `s:format="2"`) {
		t.Errorf("mit Lupe nicht Version 2: %.200s", svg)
	}
	if strings.Contains(svg, "data:") || strings.Contains(svg, "<image x=\"0\" y=\"0\" width=\"64\" height=\"48\" href=\"schnapp:original\"/><image") {
		t.Error("snAP trägt Pixel der Lupe")
	}
	if !reflect.DeepEqual(g.Dokument.Objekte(), []document.Objekt{l}) {
		t.Errorf("Rundlauf: %#v, erwartet %#v", g.Dokument.Objekte(), l)
	}
	// Ein Zug zählt nicht (F-409): Die Lupe im laufenden Zug ändert die Version nicht.
	z := beispielDok(t, 64, 48)
	if _, err := z.BeginneNeuenZug(l); err != nil {
		t.Fatal(err)
	}
	if svg := string(svgArbeitsdatei(z, Metadaten{})); !strings.Contains(svg, `s:format="1"`) || strings.Contains(svg, "lupe") {
		t.Errorf("Zug mit Lupe: %.300s", svg)
	}
	// Rettungskopie (ADR-0029): dieselben Angaben, ohne Pixel.
	a, err := LeseAngaben(bytes.NewReader(AngabenSVG(d)))
	if err != nil {
		t.Fatal(err)
	}
	ohneID := l
	ohneID.ID = 0 // die Angaben tragen keine Kennungen
	if !reflect.DeepEqual(a.Objekte, []document.Objekt{ohneID}) {
		t.Errorf("Angaben: %#v", a.Objekte)
	}
}

// TestLupeLeserRegeln: Version 1 kennt keine Lupe (beschädigt), kaputte Lupen sind beschädigt, das Beispiel liest sich.
func TestLupeLeserRegeln(t *testing.T) {
	png, svg := beispielSVG(t, "magnifier")
	lies := func(s string) error {
		_, err := LeseArbeitsdatei(bytes.NewReader(pngMitSVG(t, png, s)))
		if _, kopf := LeseMetadaten(bytes.NewReader(pngMitSVG(t, png, s))); urteil(kopf) != urteil(err) {
			t.Errorf("LeseMetadaten %q, voll %q", urteil(kopf), urteil(err))
		}
		return err
	}
	if err := lies(svg); err != nil {
		t.Fatal(err)
	}
	ersetze := func(alt, neu string) string {
		if !strings.Contains(svg, alt) {
			t.Fatalf("Vorlage ohne %q", alt)
		}
		return strings.Replace(svg, alt, neu, 1)
	}
	for _, f := range []struct{ name, svg, teil string }{
		{"Version 1 mit Lupe", ersetze(`s:format="2"`, `s:format="1"`), "erst ab Formatversion 2"},
		{"Quelle ohne Breite", ersetze(`s:quelle="6,30 16.5,38"`, `s:quelle="6,30 6,38"`), "Lupe"},
		{"Quelle verkehrt", ersetze(`s:quelle="6,30 16.5,38"`, `s:quelle="16.5,38 6,30"`), "Lupe"},
		{"Ziel ohne Höhe", ersetze(`s:ziel="30,6 51,22"`, `s:ziel="30,6 51,6"`), "Lupe"},
		{"Ziel zu groß", ersetze(`s:ziel="30,6 51,22"`, `s:ziel="30,6 100031,22"`), "Lupe"},
		{"Koordinate zu groß", ersetze(`s:ziel="30,6 51,22"`, `s:ziel="2e6,6 2.00001e6,22"`), "Lupe"},
		{"Ziel fehlt", ersetze(` s:ziel="30,6 51,22"`, ``), "ziel fehlt"},
		{"Ziel ein Punkt", ersetze(`s:ziel="30,6 51,22"`, `s:ziel="30,6"`), "Punkt"},
		{"Strich NaN", ersetze(`s:strich="#dc1e1eff 2.5"`, `s:strich="#dc1e1eff NaN"`), "Zahl"},
		{"Strich negativ", ersetze(`s:strich="#dc1e1eff 2.5"`, `s:strich="#dc1e1eff -1"`), "Strichbreite"},
	} {
		if err := lies(f.svg); !errors.Is(err, ErrBeschaedigt) || !strings.Contains(err.Error(), f.teil) {
			t.Errorf("%s: %v (erwartet beschädigt mit %q)", f.name, err, f.teil)
		}
	}
	// Eine Datei der Version 3 bleibt „neuer“, auch mit Lupe.
	if err := lies(ersetze(`s:format="2"`, `s:format="3"`)); !errors.Is(err, ErrFormatNeuer) {
		t.Errorf("Version 3: %v", err)
	}
}

// exportBild holt das eingebettete PNG der Lupe aus einem SVG-Export.
func exportBild(t *testing.T, svg string) (*image.NRGBA, bool) {
	t.Helper()
	m := regexp.MustCompile(`style="image-rendering:pixelated" href="data:image/png;base64,([^"]+)"`).FindStringSubmatch(svg)
	if m == nil {
		return nil, false
	}
	daten, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil {
		t.Fatal(err)
	}
	b, err := png.Decode(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := b.(*image.NRGBA)
	if !ok {
		k := image.NewNRGBA(b.Bounds())
		for y := b.Bounds().Min.Y; y < b.Bounds().Max.Y; y++ {
			for x := b.Bounds().Min.X; x < b.Bounds().Max.X; x++ {
				k.Set(x, y, b.At(x, y))
			}
		}
		n = k
	}
	return n, true
}

// TestLupeExportSVG: Der Export trägt die Ansichtspixel unter der Quelle (mit Schwärzung), auf die Ansicht begrenzt,
// in einem <svg> mit der Quelle in der Ansicht als viewBox und pixelated (ADR-0033 Nachtrag 2 Punkte 1 und 3).
func TestLupeExportSVG(t *testing.T) {
	d, _ := lupenDok(t)
	if err := d.Schwaerzen(image.Rect(4, 4, 6, 6), document.Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}); err != nil {
		t.Fatal(err)
	}
	// Zuschnitt ab x = 8: Die Quelle 4..12 liegt in der Ansicht bei −4..4, eingebettet werden nur die Spalten 0..4 der
	// Ansicht (Original 8..12); was vor dem Zuschnitt liegt, steht nicht im PNG und bleibt im Ziel leer.
	if err := d.SetzeZuschnitt(image.Rect(8, 0, 64, 48)); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	if !strings.Contains(svg, `viewBox="-4 4 8 6" preserveAspectRatio="none"><image x="0" y="4" width="4" height="6" preserveAspectRatio="none"`) {
		t.Errorf("Bild der Lupe fehlt oder liegt falsch: %.2000s", svg)
	}
	px, ok := exportBild(t, svg)
	if !ok {
		t.Fatal("kein eingebettetes PNG der Lupe")
	}
	if px.Bounds().Size() != image.Pt(4, 6) {
		t.Fatalf("PNG %v", px.Bounds())
	}
	orig := beispielBild(64, 48)
	for y := 0; y < 6; y++ {
		for x := 0; x < 4; x++ {
			if got, want := px.NRGBAAt(x, y), orig.NRGBAAt(8+x, 4+y); got != want {
				t.Errorf("PNG (%d,%d): %v, erwartet %v", x, y, got, want)
			}
		}
	}
	// Die Schwärzung liegt im verborgenen Teil; ohne Zuschnitt zeigt das PNG sie schwarz, nicht das Ausgangsbild.
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 64, 48)); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	px, _ = exportBild(t, buf.String())
	if got := px.NRGBAAt(1, 1); got != (color.NRGBA{0, 0, 0, 255}) {
		t.Errorf("geschwärztes Quellpixel im Export: %v", got)
	}
	// Ausgeblendet: keine Lupe und kein Bild im Export.
	o := d.Objekte()[0]
	e := o.Ebenenwerte()
	e.Ausgeblendet = true
	if err := d.Ersetzen(document.MitEbene(o, e)); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if _, ok := exportBild(t, buf.String()); ok {
		t.Error("ausgeblendete Lupe im Export")
	}
}
