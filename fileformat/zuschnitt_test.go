// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

// ADR-0023 Punkt 6: Der SVG-Export zeichnet je Naht ein Zackenband (heller Rand, dunkle Linie) mit denselben Punkten
// wie der Renderer, in Ansichtskoordinaten nach den Kacheln (also obenauf); ein Streifen am Rand ergibt keins.
func TestExportSVGZackenkante(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 25}, {Von: 30, Bis: 33},
		{Von: 77, Bis: 80}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	naehte := regexp.MustCompile(`<g s:naht="(zeilen|spalten)"><path d="([^"]*)" fill="#([0-9a-f]{6})"/><path d="([^"]*)" fill="#([0-9a-f]{6})"/></g>`).
		FindAllStringSubmatch(s, -1)
	if len(naehte) != 2 {
		t.Fatalf("%d Nähte im Export, Soll 2:\n%s", len(naehte), s)
	}
	g := d.Groesse()
	for i, n := range d.Naehte() {
		m := naehte[i]
		art, laenge := "spalten", g.Y
		if n.Zeilen {
			art, laenge = "zeilen", g.X
		}
		if m[1] != art || m[3] != "ffffff" || m[5] != "4a5057" {
			t.Errorf("Naht %d: %v %v %v", i, m[1], m[3], m[5])
		}
		for j, halb := range []float64{document.ZackenRandHalb, document.ZackenLinieHalb} {
			var soll []string
			for _, p := range document.Zackenband(n, laenge, halb) {
				soll = append(soll, punkt(p))
			}
			if got := m[2+2*j]; got != "M"+strings.Join(soll, " L")+" Z" {
				t.Errorf("Naht %d Band %d: %s", i, j, got)
			}
		}
	}
	if strings.LastIndex(s, `<use href="#objekte"/>`) > strings.Index(s, `s:naht=`) {
		t.Error("Zackenkante liegt unter einer Kachel")
	}
}

// Rundlauf (ADR-0023): Zuschnitt und Streifen — auch verborgene außerhalb des Zuschnitts und anstoßende — kommen
// unverändert zurück, die Ansicht samt Nähten ist dieselbe, das sichtbare Bild der Datei ist das übergebene.
func TestRundlaufZuschnittUndStreifen(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 2, Bis: 4}, {Von: 30, Bis: 33}, {Von: 33, Bis: 35},
		{Zeilen: true, Von: 40, Bis: 45}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 5, 70, 55)); err != nil { // der Streifen 2..4 liegt außerhalb
		t.Fatal(err)
	}
	daten := schreibe(t, d, meta)
	g, err := LeseArbeitsdatei(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	e := g.Dokument
	if e.Zuschnitt() != d.Zuschnitt() || !reflect.DeepEqual(e.Streifen(), d.Streifen()) {
		t.Errorf("Ansicht: %v %v, Soll %v %v", e.Zuschnitt(), e.Streifen(), d.Zuschnitt(), d.Streifen())
	}
	if e.Groesse() != d.Groesse() || !reflect.DeepEqual(e.Naehte(), d.Naehte()) {
		t.Errorf("Größe %v Nähte %v, Soll %v %v", e.Groesse(), e.Naehte(), d.Groesse(), d.Naehte())
	}
	if !bytes.Equal(e.Ansichtsbild().Pix, d.Ansichtsbild().Pix) {
		t.Error("Ansichtsbild weicht ab")
	}
	sichtbar, err := LeseBild(bytes.NewReader(daten))
	if err != nil {
		t.Fatal(err)
	}
	if !gleich(sichtbar, d.Ansichtsbild()) {
		t.Error("sichtbares Bild der Datei weicht ab")
	}
}
