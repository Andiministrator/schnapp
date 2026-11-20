// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"reflect"
	"regexp"
	"strconv"
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
	// Je Achse ein Band bei Lage 0 in <defs> (F-183), je Naht ein <use>, das es auf die Lage verschiebt.
	baender := regexp.MustCompile(`<g id="naht-(zeilen|spalten)"><path d="([^"]*)" fill="#([0-9a-f]{6})"/><path d="([^"]*)" fill="#([0-9a-f]{6})"/></g>`).
		FindAllStringSubmatch(s, -1)
	verweise := regexp.MustCompile(`<use href="#naht-(zeilen|spalten)" (x|y)="(-?\d+)" s:naht="(zeilen|spalten)"/>`).
		FindAllStringSubmatch(s, -1)
	if len(baender) != 2 || len(verweise) != 2 {
		t.Fatalf("%d Bänder, %d Nähte im Export, Soll 2 und 2:\n%s", len(baender), len(verweise), s)
	}
	g := d.Groesse()
	for i, n := range d.Naehte() {
		art, achse, laenge := "spalten", "x", g.Y
		if n.Zeilen {
			art, achse, laenge = "zeilen", "y", g.X
		}
		if v := verweise[i]; v[1] != art || v[2] != achse || v[3] != strconv.Itoa(n.Lage) || v[4] != art {
			t.Errorf("Naht %d: %v", i, v[0])
		}
		m := baender[0]
		if m[1] != art {
			m = baender[1]
		}
		if m[1] != art || m[3] != "ffffff" || m[5] != "4a5057" {
			t.Errorf("Band %s: %v %v %v", art, m[1], m[3], m[5])
		}
		// Das Band bei Lage 0, um die Lage verschoben, ist das Band des Renderers an der Naht.
		for j, halb := range []float64{document.ZackenRandHalb, document.ZackenLinieHalb} {
			var soll []string
			for _, p := range document.Zackenband(document.Naht{Zeilen: n.Zeilen}, laenge, halb) {
				soll = append(soll, punkt(p))
			}
			if got := m[2+2*j]; got != "M"+strings.Join(soll, " L")+" Z" {
				t.Errorf("Band %s %d: %s", art, j, got)
			}
			verschoben := document.Zackenband(document.Naht{Zeilen: n.Zeilen}, laenge, halb)
			for k, p := range document.Zackenband(n, laenge, halb) {
				q := verschoben[k]
				if n.Zeilen {
					q.Y += float64(n.Lage)
				} else {
					q.X += float64(n.Lage)
				}
				if p != q {
					t.Fatalf("Naht %d Band %d Punkt %d: %v, verschoben %v", i, j, k, p, q)
				}
			}
		}
	}
	if strings.LastIndex(s, `<use href="#objekte"/>`) > strings.Index(s, `s:naht=`) {
		t.Error("Zackenkante liegt unter einer Kachel")
	}
}

// F-183: 1023 Zeilen-Nähte in 8192 px Breite kosten im SVG wenige hundert Kilobyte (vorher 41 MB): ein Band je Achse
// in <defs>, je Naht ein <use>.
func TestExportSVGVieleNaehteKompakt(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8192, 2047)))
	if err != nil {
		t.Fatal(err)
	}
	var st []document.Streifen
	for i := 0; i < 1023; i++ {
		st = append(st, document.Streifen{Zeilen: true, Von: 1 + 2*i, Bis: 2 + 2*i})
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	if n := len(d.Naehte()); n != 1023 {
		t.Fatalf("%d Nähte, Soll 1023", n)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	start := strings.Index(s, `<defs><g id="naht-`)
	if start < 0 {
		t.Fatal("keine Nähte im Export")
	}
	const soll = 256 << 10
	if anteil := len(s) - start; anteil > soll {
		t.Errorf("Nähte belegen %d Byte, Soll höchstens %d", anteil, soll)
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
