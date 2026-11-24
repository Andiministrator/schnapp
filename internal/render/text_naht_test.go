// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// textNahtDokument legt ein weißes Bild mit einem langen Text an, der über x = 100…140 reicht.
func textNahtDokument(t *testing.T, text document.Text) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 360, 90))
	draw.Draw(b, b.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(text); err != nil {
		t.Fatal(err)
	}
	return d
}

// F-739 (ADR-0023 Nachtrag 13): Ein Text über einem Stauch-Streifen ist an der Naht abgeschnitten wie das Bild
// darunter. Links der Naht stehen dieselben Pixel wie ohne Stauchen, rechts davon dieselben wie ohne Stauchen um die
// Streifenbreite verschoben; der Teil im Streifen fehlt. Verglichen wird außerhalb des Kantenbands der Naht.
func TestTextUeberNahtAbgeschnitten(t *testing.T) {
	text := document.Text{Ort: document.Pt(12, 30), Inhalt: "WWW Geheimnis 1234567 WWW", Schriftgrad: 22,
		Farbe: color.NRGBA{A: 255}}
	voll, err := Bild(textNahtDokument(t, text))
	if err != nil {
		t.Fatal(err)
	}
	d := textNahtDokument(t, text)
	if err := d.SetzeStreifen([]document.Streifen{{Von: 100, Bis: 140}}); err != nil {
		t.Fatal(err)
	}
	if n := len(d.TextStuecke(text)); n != 2 {
		t.Fatalf("Testtext liegt nicht über der Naht: %d Stücke", n)
	}
	gestaucht, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	const naht, breite, band = 100, 40, document.KantenBandHalb + 4
	vergleiche := func(x0, x1, versatz int) (abweichend, tinte int) {
		for y := 0; y < 90; y++ {
			for x := x0; x < x1; x++ {
				g, v := gestaucht.NRGBAAt(x, y), voll.NRGBAAt(x+versatz, y)
				if g != v {
					abweichend++
				}
				if v.R < 128 {
					tinte++
				}
			}
		}
		return
	}
	links, tl := vergleiche(0, naht-band, 0)
	rechts, tr := vergleiche(naht+band, 360-breite, breite)
	if tl == 0 || tr == 0 {
		t.Fatalf("Vergleich ohne Tinte: links %d, rechts %d dunkle Pixel", tl, tr)
	}
	if links != 0 || rechts != 0 {
		t.Errorf("Text an der Naht nicht wie das Bild abgeschnitten: links %d, rechts %d Pixel abweichend", links, rechts)
	}
}

// Ein Text, dessen Anfang im Streifen liegt und der nur knapp herausragt, steht nicht mehr ganz hinter der Naht: rechts
// der Naht erscheint nur der herausragende Teil (V3-1).
func TestTextKnappAusStreifen(t *testing.T) {
	text := document.Text{Ort: document.Pt(70, 30), Inhalt: "Geheimnis Ende", Schriftgrad: 22,
		Farbe: color.NRGBA{A: 255}}
	voll, err := Bild(textNahtDokument(t, text))
	if err != nil {
		t.Fatal(err)
	}
	d := textNahtDokument(t, text)
	// Streifen bis knapp vor das Textende: nur das Ende ragt heraus.
	ende := 0
	for x := 359; x >= 0 && ende == 0; x-- {
		for y := 0; y < 90; y++ {
			if voll.NRGBAAt(x, y).R < 128 {
				ende = x
				break
			}
		}
	}
	if ende < 120 {
		t.Fatalf("Textende bei %d", ende)
	}
	bis := ende - 6
	if err := d.SetzeStreifen([]document.Streifen{{Von: 60, Bis: bis}}); err != nil {
		t.Fatal(err)
	}
	gestaucht, err := Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	// Rechts vom Kantenband der Naht (x = 60) darf höchstens die Tinte der letzten 6 Spalten stehen, nicht der Text.
	dunkel := 0
	for y := 0; y < 90; y++ {
		for x := 60 + int(document.KantenBandHalb) + 4; x < gestaucht.Bounds().Dx(); x++ {
			if gestaucht.NRGBAAt(x, y).R < 128 {
				dunkel++
			}
		}
	}
	if dunkel != 0 {
		t.Errorf("hinter der Naht stehen %d dunkle Pixel; der Text gehört in den Streifen", dunkel)
	}
}
