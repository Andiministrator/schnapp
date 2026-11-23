// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gomedium"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
)

// exportSVG liefert das exportierte SVG von d.
func exportSVG(t *testing.T, d *document.Dokument) string {
	t.Helper()
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

var tspanX = regexp.MustCompile(`<tspan x="([^"]+)"`)

// F-577: Der SVG-Export verankert Zeilen bei Ausrichtung Mitte und rechts mit text-anchor an Mitte bzw. rechtem Ende
// (x = Mitte/Ende der Zeile im Renderer): Ein Betrachter mit anderer Schrift behält so die Ausrichtung. Links bleibt
// ohne Angabe; die Arbeitsdatei bleibt unverändert (Zeilenanfang, kein text-anchor).
func TestExportTextAnchor(t *testing.T) {
	for _, c := range []struct {
		a      document.Ausrichtung
		anchor string
		anteil float64
	}{{document.AusrichtungLinks, "", 0}, {document.AusrichtungMitte, "middle", 0.5}, {document.AusrichtungRechts, "end", 1}} {
		tx := document.Text{Ort: document.Pt(4, 4), Inhalt: "WWWW\nx", Schriftgrad: 9, Farbe: color.NRGBA{A: 255}, Ausrichtung: c.a}
		d := beispielDok(t, 64, 48)
		fuege(t, d, tx)
		svg := exportSVG(t, d)
		if got := strings.Contains(svg, `text-anchor="`+c.anchor+`"`); c.anchor != "" && !got {
			t.Errorf("Ausrichtung %v: text-anchor=%q fehlt: %s", c.a, c.anchor, svg)
		}
		if c.anchor == "" && strings.Contains(svg, "text-anchor") {
			t.Errorf("links schreibt text-anchor: %s", svg)
		}
		xs := tspanX.FindAllStringSubmatch(svg, -1)
		z := tx.Zeilen()
		if len(xs) != len(z) {
			t.Fatalf("%d tspan, erwartet %d: %s", len(xs), len(z), svg)
		}
		for i, m := range xs {
			x, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				t.Fatal(err)
			}
			if want := z[i].X + c.anteil*z[i].Vorschub; math.Abs(x-want) > 1e-9 {
				t.Errorf("Ausrichtung %v, Zeile %d: x=%v, erwartet %v", c.a, i, x, want)
			}
		}
		// Arbeitsdatei: unverändert.
		var buf bytes.Buffer
		if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
			t.Fatal(err)
		}
		arbeit := string(entpackeSVG(t, buf.Bytes()))
		if strings.Contains(arbeit, "text-anchor") {
			t.Errorf("Arbeitsdatei schreibt text-anchor: %s", arbeit)
		}
		for i, m := range tspanX.FindAllStringSubmatch(arbeit, -1) {
			if x, _ := strconv.ParseFloat(m[1], 64); math.Abs(x-z[i].X) > 1e-9 {
				t.Errorf("Arbeitsdatei, Zeile %d: x=%v, erwartet Zeilenanfang %v", i, x, z[i].X)
			}
		}
	}
}

// mitNurNormal stellt eine Systemfamilie mit nur einem Schnitt (Go Medium, normal) bereit und gibt ihren Namen zurück.
func mitNurNormal(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "GoMedium.ttf"), gomedium.TTF, 0o600); err != nil {
		t.Fatal(err)
	}
	fonts.SetzeOrdner([]string{d})
	fonts.SucheZuruecksetzen()
	fonts.Suche()
	t.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	for _, n := range fonts.Familien() {
		if n != fonts.Go && n != fonts.GoMono {
			return n
		}
	}
	t.Fatal("Go Medium nicht gefunden")
	return ""
}

// F-577: font-weight und font-style folgen dem Schnitt, den Renderer und Editor wirklich setzen. Hat die Familie nur
// den normalen Schnitt, steht trotz „fett“ und „kursiv“ keine Angabe im SVG; Go selbst (alle Schnitte) schreibt sie.
func TestExportGewichtNachGesetztemSchnitt(t *testing.T) {
	fam := mitNurNormal(t)
	tx := document.Text{Ort: document.Pt(4, 4), Inhalt: "ab", Schriftgrad: 9, Farbe: color.NRGBA{A: 255}, Schrift: fam,
		Fett: true, Kursiv: true}
	if s := tx.Schriftdatei(); s.Schnitt != fonts.Normal || s.Ersatz {
		t.Fatalf("Vorbedingung: Schnitt %v, Ersatz %v", s.Schnitt, s.Ersatz)
	}
	d := beispielDok(t, 64, 48)
	fuege(t, d, tx)
	if svg := exportSVG(t, d); strings.Contains(svg, "font-weight") || strings.Contains(svg, "font-style") {
		t.Errorf("Familie nur normal: Gewicht oder Stil im SVG: %s", svg)
	}
	// Die Arbeitsdatei behält die Angabe (s:stil), der Leser bekommt „fett kursiv“ zurück.
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if arbeit := string(entpackeSVG(t, buf.Bytes())); !strings.Contains(arbeit, `s:stil="bold italic"`) {
		t.Errorf("s:stil fehlt: %s", arbeit)
	}
	// Go hat alle Schnitte.
	tx.Schrift = ""
	d2 := beispielDok(t, 64, 48)
	fuege(t, d2, tx)
	if svg := exportSVG(t, d2); !strings.Contains(svg, `font-weight="bold"`) || !strings.Contains(svg, `font-style="italic"`) {
		t.Errorf("Go fett kursiv: Angaben fehlen: %s", svg)
	}
}
