// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

// Zeit je Editor-Bild über vielen Nähten (F-747, F-750, ADR-0023 Nachtrag 16): Gemessen wird nur der Aufbau der
// Zeichenbefehle ohne GPU. Vorher lief der Editor je Bereich über alle Zeilen (5,3 s je Bild bei 1024 Zeilenstreifen)
// und setzte jede Zeile je Bereich neu (149 ms bei 1023 Spaltenstreifen); jetzt rund 4 ms. Grenze 1 s auch unter -race
// auf einem langsamen Runner.
func TestEditorBildUeberVielenNaehten(t *testing.T) {
	schwarz := color.NRGBA{A: 255}
	for _, fall := range []struct {
		name   string
		w, h   int
		inhalt string
		grad   float64
	}{
		{"5 Texte je 9999 Zeilen, 1024 Zeilenstreifen", 64, 4096, "x" + strings.Repeat("\n", 9998) + "x", 8},
		{"5 Texte 6×60 @ Grad 74, 1023 Spaltenstreifen", 4200, 520,
			strings.TrimSuffix(strings.Repeat(strings.Repeat("@", 60)+"\n", 6), "\n"), 74},
	} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, fall.w, fall.h)))
		if err != nil {
			t.Fatal(err)
		}
		var st []document.Streifen
		if fall.w == 64 {
			for i := 0; i < document.MaxStreifen; i++ {
				st = append(st, document.Streifen{Zeilen: true, Von: 2 * i, Bis: 2*i + 1})
			}
		} else {
			for i := 0; i < 1023; i++ {
				st = append(st, document.Streifen{Von: 4 * i, Bis: 4*i + 1})
			}
		}
		if err := d.SetzeStreifen(st); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(4, 5), Inhalt: fall.inhalt, Schriftgrad: fall.grad,
				Farbe: schwarz}); err != nil {
				t.Fatal(err)
			}
		}
		e := &editor{s: canvas.NeueSitzung(d)}
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Exact(d.Groesse())}
		e.zeichneObjekte(gtx) // Zwischenspeicher (Tinte, Umrisse) füllen
		ops.Reset()
		s := time.Now()
		e.zeichneObjekte(gtx)
		if dt := time.Since(s); dt > time.Second {
			t.Errorf("%s: zeichneObjekte %v, Grenze 1 s", fall.name, dt)
		}
	}
}
