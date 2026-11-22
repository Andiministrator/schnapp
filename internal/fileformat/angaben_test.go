// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"image"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-385 (Verify 7, F1): LeseAngaben prüft mit denselben Grenzen wie das Laden (document.PruefeInhalt): Ein Zuschnitt
// außerhalb des Bildes wird abgelehnt, nicht erst beim Anwenden.
func TestLeseAngabenGrenzen(t *testing.T) {
	d, _ := document.Neu(testbild(80, 60))
	if err := d.SetzeZuschnitt(image.Rect(2, 3, 70, 50)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(10, 20), Inhalt: "a", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	svg := string(AngabenSVG(d))
	a, err := LeseAngaben(strings.NewReader(svg))
	if err != nil || a.Groesse != image.Pt(80, 60) || a.Zuschnitt != image.Rect(2, 3, 70, 50) || len(a.Objekte) != 1 {
		t.Fatalf("Rundlauf: %v %+v", err, a)
	}
	for name, ersatz := range map[string][2]string{
		"Zuschnitt außerhalb": {`s:zuschnitt="2 3 70 50"`, `s:zuschnitt="2 3 90 50"`},
		"Bild zu groß":        {`width="80" height="60" viewBox="0 0 80 60"`, `width="80000" height="60000" viewBox="0 0 80000 60000"`},
	} {
		if !strings.Contains(svg, ersatz[0]) {
			t.Fatalf("%s: Muster %q fehlt im SVG", name, ersatz[0])
		}
		kaputt := strings.Replace(svg, ersatz[0], ersatz[1], 1)
		if a, err := LeseAngaben(strings.NewReader(kaputt)); err == nil || !strings.HasPrefix(err.Error(), "fileformat: ") {
			t.Errorf("%s: %v %+v, erwartet Fehler mit Präfix", name, err, a)
		}
	}
}
