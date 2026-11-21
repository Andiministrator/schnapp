// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"encoding/xml"
	"image/color"
	"io"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-343: Leerraum im SVG wie im Renderer: mehrere und führende Leerzeichen bleiben, der Tabulator wird zu vier
// Leerzeichen. Das SVG bleibt gültiges XML, die Arbeitsdatei behält den Originaltext (s:inhalt) für den Rundlauf.
func TestSVGLeerraumWieRenderer(t *testing.T) {
	d, err := document.Neu(testbild(300, 200))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(20, 20), Inhalt: "a     b\n   c\n\td", Schriftgrad: 20, Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `xml:space="preserve"`) {
		t.Error("SVG-Text ohne erhaltenen Leerraum: Betrachter falten Leerzeichen")
	}
	if strings.Contains(s, "\t") || strings.Contains(s, "&#x9;") {
		t.Error("Tabulator steht roh im SVG, der Renderer zeichnet vier Leerzeichen")
	}
	for _, soll := range []string{">a     b<", ">   c<", ">    d<"} {
		if !strings.Contains(s, soll) {
			t.Errorf("SVG ohne %q", soll)
		}
	}
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("SVG kein gültiges XML: %v", err)
		}
	}
	if a := svgArbeitsdatei(d, Metadaten{}); !bytes.Contains(a, []byte("&#x9;d")) {
		t.Error("Arbeitsdatei verliert den Tabulator im Originaltext (s:inhalt)")
	}
}
