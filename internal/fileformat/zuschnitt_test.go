// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0023 Punkt 6, Nachtrag 6: Der SVG-Export zeichnet je Naht alle Schichten der Kante (document.KantenSchichten) mit
// denselben Punkten wie der Renderer, in der Form des Dokuments, halbdurchsichtige mit fill-opacity, ohne Filter, in
// Ansichtskoordinaten nach den Kacheln (also obenauf); ein Streifen am Rand ergibt keine.
func TestExportSVGKante(t *testing.T) {
	for _, form := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		d, err := document.Neu(testbild(80, 60))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.VorgabeKante(form); err != nil {
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
		if strings.Contains(s, "filter") || strings.Contains(s, "s:") {
			t.Errorf("%v: Filter oder s:-Attribut im Export", form)
		}
		// Je Achse ein Band bei Lage 0 in <defs> (F-183), je Naht ein <use>, das es auf die Lage verschiebt.
		gruppen := regexp.MustCompile(`<g id="naht-(zeilen|spalten)">(.*?)</g>`).FindAllStringSubmatch(s, -1)
		verweise := regexp.MustCompile(`<use href="#naht-(zeilen|spalten)" (x|y)="(-?\d+)"/>`).FindAllStringSubmatch(s, -1)
		if len(gruppen) != 2 || len(verweise) != 2 {
			t.Fatalf("%v: %d Bänder, %d Nähte im Export, Soll 2 und 2:\n%.2000s", form, len(gruppen), len(verweise), s)
		}
		pfadRe := regexp.MustCompile(`<path d="([^"]*)" fill="#([0-9a-f]{6})"(?: fill-opacity="([0-9.]+)")?/>`)
		schichten := document.KantenSchichten()
		g := d.Groesse()
		for i, n := range d.Naehte() {
			art, achse, laenge := "spalten", "x", g.Y
			if n.Zeilen {
				art, achse, laenge = "zeilen", "y", g.X
			}
			if v := verweise[i]; v[1] != art || v[2] != achse || v[3] != strconv.Itoa(n.Lage) {
				t.Errorf("%v Naht %d: %v", form, i, v[0])
			}
			grp := gruppen[0]
			if grp[1] != art {
				grp = gruppen[1]
			}
			pfade := pfadRe.FindAllStringSubmatch(grp[2], -1)
			if len(pfade) != len(schichten) {
				t.Fatalf("%v Band %s: %d Pfade, Soll %d Schichten", form, art, len(pfade), len(schichten))
			}
			for j, sch := range schichten {
				p := pfade[j]
				if want := fmt.Sprintf("%02x%02x%02x", sch.Farbe.R, sch.Farbe.G, sch.Farbe.B); p[2] != want {
					t.Errorf("%v Schicht %d: Farbe %s, Soll %s", form, j, p[2], want)
				}
				if want := ""; sch.Farbe.A != 255 {
					want = strconv.FormatFloat(float64(sch.Farbe.A)/255, 'g', -1, 64)
					if p[3] != want {
						t.Errorf("%v Schicht %d: Deckkraft %q, Soll %q", form, j, p[3], want)
					}
				} else if p[3] != want {
					t.Errorf("%v Schicht %d: deckend mit fill-opacity %q", form, j, p[3])
				}
				// Das Band bei Lage 0, um die Lage verschoben, ist das Band des Renderers an der Naht.
				var soll []string
				bei0 := document.Kantenband(form, document.Naht{Zeilen: n.Zeilen}, laenge, d.ZackenFaktor(), sch.Von, sch.Bis)
				for _, q := range bei0 {
					soll = append(soll, punkt(q))
				}
				if p[1] != "M"+strings.Join(soll, " L")+" Z" {
					t.Errorf("%v Band %s Schicht %d: %.200s", form, art, j, p[1])
				}
				for k, q := range document.Kantenband(form, n, laenge, d.ZackenFaktor(), sch.Von, sch.Bis) {
					v := bei0[k]
					if n.Zeilen {
						v.Y += float64(n.Lage)
					} else {
						v.X += float64(n.Lage)
					}
					if math.Abs(q.X-v.X) > 1e-9 || math.Abs(q.Y-v.Y) > 1e-9 {
						t.Fatalf("%v Naht %d Schicht %d Punkt %d: %v, verschoben %v", form, i, j, k, q, v)
					}
				}
			}
		}
		if strings.LastIndex(s, `<use href="#objekte"/>`) > strings.Index(s, `<use href="#naht-`) {
			t.Errorf("%v: Kante liegt unter einer Kachel", form)
		}
	}
}

// F-183: Viele Nähte kosten im SVG je Naht nur eine kurze Zeile (vorher 41 MB für 1023 Zeilen-Nähte in 8192 px
// Breite): ein Band je Achse in <defs>, je Naht ein <use>. Das Band selbst (alle Schichten des Abrisses, je Pixel ein
// Stützpunkt, ADR-0023 Nachtrag 6) steht einmal und hält sich mit Hundertsteln kurz.
func TestExportSVGVieleNaehteKompakt(t *testing.T) {
	export := func(anzahl int) (gesamt, band int) {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8192, 2047)))
		if err != nil {
			t.Fatal(err)
		}
		var st []document.Streifen
		for i := 0; i < anzahl; i++ {
			st = append(st, document.Streifen{Zeilen: true, Von: 1 + 2*i, Bis: 2 + 2*i})
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		if n := len(d.Naehte()); n != anzahl {
			t.Fatalf("%d Nähte, Soll %d", n, anzahl)
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
		ende := strings.Index(s[start:], `</defs>`)
		return len(s) - start, ende
	}
	eins, band1 := export(1)
	viele, band := export(1023)
	if band != band1 {
		t.Errorf("Band in <defs> hängt an der Zahl der Nähte: %d gegen %d Byte", band, band1)
	}
	if je := (viele - eins) / 1022; je > 64 {
		t.Errorf("je weitere Naht %d Byte, Soll höchstens 64", je)
	}
	const soll = 4 << 20
	if band > soll {
		t.Errorf("Band belegt %d Byte, Soll höchstens %d", band, soll)
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

// F-173: Bei 2200×2200 Ansicht gilt Faktor 2; das Band in <defs> hat doppelte Zackenhöhe und Spaltbreite (Form Zacken,
// Naht bei Lage 0: erster Punkt des Spalts bei x = -8 - 10 = -18 statt -4 - 5 = -9, der zweite bei y = 16).
func TestExportSVGKanteFaktor2(t *testing.T) {
	d, err := document.Neu(testbild(2200, 2200))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VorgabeKante(document.KanteZacken); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 110}}); err != nil {
		t.Fatal(err)
	}
	if f := d.ZackenFaktor(); f != 2 {
		t.Fatalf("Faktor %d, Soll 2", f)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `<g id="naht-spalten"><path d="M-18,0 L-2,16 `) {
		t.Errorf("Band bei Faktor 2 nicht gefunden:\n%.600s", buf.String())
	}
}
