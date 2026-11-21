// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// F-225 (ADR-0025 Punkt 4, Kernprinzip 3): Der SVG-Export trägt weder den Programm-Namensraum noch Bearbeitungsattribute
// mit Präfix `s:`; die Arbeitsdatei behält sie für den Rundlauf. Die viewBox der Kacheln beginnt bei 0,0 (die Lage des
// Zuschnitts steht weiter im transform und in den Objekt-Koordinaten, Verify V-5).
func TestExportSVGOhneProgrammspuren(t *testing.T) {
	d := volles(t) // alle Objektarten, Zuschnitt und Streifen
	var ex bytes.Buffer
	if err := ExportiereSVG(&ex, d); err != nil {
		t.Fatal(err)
	}
	s := ex.String()
	for _, spur := range []string{"x-schnapp", "xmlns:s", " s:", "<s:", "s:typ", "s:inhalt", "s:naht", "s:ebene"} {
		if strings.Contains(s, spur) {
			t.Errorf("Export enthält %q", spur)
		}
	}
	// Die Nähte und Objekte sind dennoch da (Sollwert, kein Rauchtest).
	for _, soll := range []string{`<use href="#naht-`, `<tspan`, `<circle`, `<rect`, `id="objekte"`} {
		if !strings.Contains(s, soll) {
			t.Errorf("Export ohne %q", soll)
		}
	}
	for _, vb := range regexp.MustCompile(`viewBox="([^"]*)"`).FindAllStringSubmatch(s, -1) {
		if !strings.HasPrefix(vb[1], "0 0 ") {
			t.Errorf("viewBox %q beginnt nicht bei 0,0", vb[1])
		}
	}
	// Gegenprobe: Die Arbeitsdatei trägt die Attribute weiter.
	arb := string(svgArbeitsdatei(d, Metadaten{}))
	for _, soll := range []string{`xmlns:s="urn:x-schnapp:1"`, `s:typ="pfeil"`, `s:typ="text"`, `s:ebene="objekte"`, `s:format=`} {
		if !strings.Contains(arb, soll) {
			t.Errorf("Arbeitsdatei ohne %q", soll)
		}
	}
}
