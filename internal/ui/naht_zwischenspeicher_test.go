// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/document"
)

// editorMitNaht ist testEditor mit einer Naht (Bild 40×30, Streifen von Zeile 10 bis 15).
func editorMitNaht(t *testing.T, w, h int) *editor {
	t.Helper()
	e := testEditor(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, w, h)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 10, Bis: 15}}); err != nil {
		t.Fatal(err)
	}
	e.s.D = d
	return e
}

// F-420: Der Zwischenspeicher der Nähte (e.naht) wird neu gebaut, wenn sich Kantenform oder Zackenfaktor ändern. Ohne
// Form und Faktor im Schlüssel zeigte die Leinwand nach „Seam style“ weiter die alte Kante. Ohne Fenster prüfbar, weil
// zeichneNaehte nur Ops aufzeichnet: der Schlüssel im Zwischenspeicher gibt Auskunft.
func TestZwischenspeicherNaehteFolgtFormUndFaktor(t *testing.T) {
	e := editorMitNaht(t, 40, 30)
	gtx := layout.Context{Ops: new(op.Ops)}
	e.zeichneNaehte(gtx)
	if !e.naht.gueltig || e.naht.form != document.KanteAbriss || len(e.naht.naehte) != 1 {
		t.Fatalf("erster Aufbau: gültig %v, Form %v, %d Nähte", e.naht.gueltig, e.naht.form, len(e.naht.naehte))
	}
	for _, k := range []document.Kante{document.KanteWelle, document.KanteZacken, document.KanteAbriss} {
		if err := e.s.D.SetzeKante(k); err != nil {
			t.Fatal(err)
		}
		e.zeichneNaehte(gtx)
		if e.naht.form != k {
			t.Errorf("Zwischenspeicher hält Form %v nach Wechsel auf %v", e.naht.form, k)
		}
	}
	// Faktor: Der Schlüssel ist der Zackenfaktor des Dokuments; ein Wert, der nicht passt, löst den Neuaufbau aus.
	e.naht.faktor = e.s.D.KantenFaktor() + 1
	e.zeichneNaehte(gtx)
	if e.naht.faktor != e.s.D.KantenFaktor() {
		t.Errorf("Zwischenspeicher hält Faktor %d, Soll %d", e.naht.faktor, e.s.D.KantenFaktor())
	}
}

// F-433, Editor-Seite: Die Linie je Achse hat die Länge der Naht — Breite der Ansicht für eine Zeilen-Naht, Höhe für
// eine Spalten-Naht — und die beiden Achsen teilen sich keine Linie. Das Soll kommt von document.Kantenband.
func TestNahtLinienJeAchse(t *testing.T) {
	for _, gr := range []image.Point{{300, 120}, {120, 300}} {
		for _, form := range []document.Kante{document.KanteWelle, document.KanteAbriss} {
			g := image.Pt(gr.X-5, gr.Y-5)
			l := &nahtLinien{g: g, form: form, faktor: 1}
			// Spalten-Naht zuerst: Mit gemeinsamer Linie bestimmte sie die Länge für beide Achsen.
			for _, n := range []document.Naht{{Zeilen: false, Lage: 45}, {Zeilen: true, Lage: 45}, {Zeilen: false, Lage: 60}} {
				linie, laenge := l.fuer(n)
				soll := g.Y
				if n.Zeilen {
					soll = g.X
				}
				if laenge != soll {
					t.Errorf("%v %v Naht %+v: Länge %d, Soll %d", gr, form, n, laenge, soll)
				}
				von, bis := document.Spaltband()
				got := linie.Band(nil, n, von, bis)
				want := document.Kantenband(form, n, soll, 1, von, bis)
				if len(got) != len(want) {
					t.Fatalf("%v %v Naht %+v: %d Punkte, Soll %d", gr, form, n, len(got), len(want))
				}
				for i := range got {
					if got[i] != want[i] {
						t.Fatalf("%v %v Naht %+v: Punkt %d %v, Soll %v", gr, form, n, i, got[i], want[i])
					}
				}
			}
		}
	}
}
