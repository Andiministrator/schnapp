// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// F-797 (ADR-0023 Nachtrag 18, F-765): Eine lange Textzeile zeichnet der Editor in Läufen, deren Glyphenursprünge höchstens
// vier seitliche Hüllen auseinanderliegen, damit jede Spalte nur die Läufe abspielt, die sie berührt. Der Zeittest
// TestEditorGanzesBildUeberVielenNaehten merkt eine Zeile in einem Lauf nicht (Mutant B3); dieser Test zählt die Läufe.
func TestZeileInLaeufenZerlegtLangeZeile(t *testing.T) {
	s := fonts.Laden("Go", fonts.Normal)
	const grad = 20.0
	gtx := layout.Context{Ops: new(op.Ops)}
	sicht := image.Rect(0, 0, 5000, 100)
	var ts textsatz
	text := strings.Repeat("x", 60)
	laeufe, n := ts.zeileInLaeufen(gtx, s, text, 0, 50, grad, color.NRGBA{A: 255}, sicht)
	if n != 60 {
		t.Fatalf("gezeichnete Glyphen = %d, soll 60", n)
	}
	if len(laeufe) != 6 {
		t.Fatalf("Läufe = %d, soll 6", len(laeufe))
	}
	_, _, seitlich := s.Huelle()
	for i, l := range laeufe {
		// Ursprünge eines Laufs höchstens 4 Hüllen auseinander: Tinte höchstens 4 + 2 Hüllen plus eine Glyphe breit.
		if b := l.bis - l.von; b > 6*seitlich*grad+2*grad {
			t.Errorf("Lauf %d ist %.1f breit, zu lang", i, b)
		}
		if i > 0 && l.von < laeufe[i-1].von {
			t.Errorf("Lauf %d beginnt vor dem Lauf davor", i)
		}
	}
	if laeufe[0].von >= laeufe[len(laeufe)-1].von {
		t.Errorf("Läufe liegen nicht nacheinander entlang der Zeile")
	}
	// Eine kurze Zeile bleibt ein Lauf.
	kurz, _ := ts.zeileInLaeufen(gtx, s, "xx", 0, 50, grad, color.NRGBA{A: 255}, sicht)
	if len(kurz) != 1 {
		t.Errorf("kurze Zeile: Läufe = %d, soll 1", len(kurz))
	}
}
