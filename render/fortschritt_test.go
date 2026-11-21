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

// Verify F-298 B-1: Füllungen unter 64 Zeilen Höhe melden je Füllung Fortschritt (der Zeilenzähler erreicht sie nie).
func TestFlacheFuellungenMeldenFortschritt(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 2000, 400)))
	if err != nil {
		t.Fatal(err)
	}
	const n = 20
	for i := 0; i < n; i++ {
		y := float64(i * 10)
		if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(0, y), document.Pt(2000, y+60)),
			Fuellung: color.NRGBA{R: 200, A: 128}}); err != nil {
			t.Fatal(err)
		}
	}
	vorher := speicherschritt.Pulse()
	if _, err := Bild(d); err != nil {
		t.Fatal(err)
	}
	if got := speicherschritt.Pulse() - vorher; got < n {
		t.Errorf("%d Pulse für %d flache Füllungen", got, n)
	}
}

// Verify F-298 (Mutation R1): Eine einzelne hohe Füllung meldet unterwegs Fortschritt (alle 64 Zeilen), nicht nur
// zu Beginn.
func TestHoheFuellungMeldetUnterwegsFortschritt(t *testing.T) {
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 2000)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(0, 0), document.Pt(200, 1920)),
		Fuellung: color.NRGBA{G: 200, A: 255}}); err != nil {
		t.Fatal(err)
	}
	vorher := speicherschritt.Pulse()
	if _, err := Bild(d); err != nil {
		t.Fatal(err)
	}
	if got := speicherschritt.Pulse() - vorher; got < 20 { // 1920 Zeilen / 64 = 30, dazu der Puls je Füllung
		t.Errorf("nur %d Pulse für eine Füllung über 1920 Zeilen", got)
	}
}
