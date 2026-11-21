// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-70: Der Filter des SVG-Exports (Dokument.ObjektSichtbar, Reichweite) ist ein Obermaß. Eine Pfeilspitze, die aus
// der Hülle von A, B, K herausragt, und ein Text, den BreiteGrenze in Grenzen() kappt, der Export aber nicht
// umbricht, bleiben im Export, sobald ein Stück in die Ansicht reicht; weit außerhalb Liegendes bleibt draußen.
func TestExportSVGFilterObermass(t *testing.T) {
	d, err := document.Neu(testbild(80, 60))
	if err != nil {
		t.Fatal(err)
	}
	schwarz := color.NRGBA{0, 0, 0, 255}
	st := document.Strich{Farbe: schwarz, Breite: 1}
	// Hülle aus A, B, K endet bei x = 34 (links vom Zuschnitt ab 40); die Spitze (Länge 8) zeigt nach links, ihre
	// Basis sitzt bei x = 41,5, also 1,5 px in der Ansicht.
	knapp := document.Pfeil{A: document.Pt(33.6, 30), B: document.Pt(33.5, 30), K: document.Pt(33.55, 30), Strich: st}
	weit := document.Pfeil{A: document.Pt(2, 30), B: document.Pt(1, 30), K: document.Pt(1.5, 30), Strich: st}
	for _, o := range []document.Objekt{
		knapp, weit,
		// BreiteGrenze 10: der Export umbricht nicht, die Zeile reicht weit in die Ansicht (Grenzen() kappt seit F-172
		// nicht mehr, die Reichweite ist unabhängig davon ein Obermaß).
		document.Text{Ort: document.Pt(0, 12), Inhalt: "LANGERTEXTBLEIBT", Schriftgrad: 5, Farbe: schwarz, BreiteGrenze: 10},
		document.Text{Ort: document.Pt(0, 40), Inhalt: "WEIT", Schriftgrad: 5, Farbe: schwarz},
	} {
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.SetzeZuschnitt(image.Rect(40, 10, 80, 50)); err != nil {
		t.Fatal(err)
	}
	if g := knapp.Grenzen(); d.Sichtbar(g) {
		t.Fatalf("Voraussetzung: Grenzen %v des knappen Pfeils sollen außerhalb liegen", g)
	}
	var buf bytes.Buffer
	if err := ExportiereSVG(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "LANGERTEXTBLEIBT") {
		t.Error("Text mit BreiteGrenze, der in die Ansicht reicht, fehlt im Export")
	}
	if strings.Contains(s, "WEIT") {
		t.Error("Text weit außerhalb steht im Export")
	}
	if n := strings.Count(s, ` Z"`); n != 1 {
		t.Errorf("Pfeilspitzen im Export: %d, Soll 1 (nur der knappe Pfeil)", n)
	}
}
