// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/internal/speicherschritt"
)

// F-298: Rendern meldet Fortschritt (internal/speicherschritt) — beim Füllen großer Flächen und je Textzeichen —,
// damit ein Speichern am Budget nicht als hängend gilt.
func TestRendernMeldetFortschritt(t *testing.T) {
	for name, o := range map[string]document.Objekt{
		"Fläche": document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(1500, 1500)),
			Fuellung: color.NRGBA{R: 200, A: 255}},
		"Text": document.Text{Ort: document.Pt(10, 40), Inhalt: "Hallo", Schriftgrad: 20,
			Farbe: color.NRGBA{A: 255}},
	} {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 1600, 1600)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
		vorher := time.Now()
		time.Sleep(time.Millisecond)
		if _, err := Bild(d); err != nil {
			t.Fatal(err)
		}
		if !speicherschritt.Zuletzt().After(vorher) {
			t.Errorf("%s: kein Fortschritt beim Rendern", name)
		}
	}
}
