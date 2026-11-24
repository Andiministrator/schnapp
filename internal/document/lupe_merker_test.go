// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"testing"
)

// merkerDokument ist ein breites Bild mit je einem Zeilen- und Spaltenstreifen und n kleinen Lupen, deren Quelle an die
// Zeilennaht reicht (Aufbau der Abschluss-Kritik Sprint 12, A-2, kleiner).
func merkerDokument(t *testing.T, n int) (*Dokument, []Lupe) {
	t.Helper()
	const w, h = 4096, 300
	bild := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bild.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	d, err := NeuAus(bild)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen([]Streifen{{Zeilen: true, Von: 100, Bis: 110}, {Zeilen: false, Von: 1000, Bis: 1010}}); err != nil {
		t.Fatal(err)
	}
	var lupen []Lupe
	for i := 0; i < n; i++ {
		x := float64(900 + i%50)
		if i%2 == 1 {
			x = float64(990 + i%7) // reicht auch an die Spaltennaht bei 1000: beide Achsen im Spiel
		}
		l := Lupe{Quelle: R(Pt(x, 88), Pt(x+3, 99)), Ziel: R(Pt(x, 20), Pt(x+6, 42)),
			Strich: Strich{Farbe: color.NRGBA{0, 200, 0, 255}, Breite: 0}}
		l.Ziel.Max.X += float64(i) * 1e-6 // jede Lupe etwas anders
		id, err := d.Hinzufuegen(l)
		if err != nil {
			t.Fatal(err)
		}
		l.ID = id
		lupen = append(lupen, l)
	}
	d.VergissVerlauf()
	return d, lupen
}

// zielRaster sind die ganzen Pixel des Ziels von l.
func zielRaster(l Lupe) image.Rectangle {
	return image.Rect(int(math.Floor(l.Ziel.Min.X)), int(math.Floor(l.Ziel.Min.Y)), int(math.Ceil(l.Ziel.Max.X)),
		int(math.Ceil(l.Ziel.Max.Y)))
}

// TestLupenQuelleKantenlinieEinmal (F-693): 400 Lupen neben einer Naht rechnen die Mittellinie der Kante einmal je
// Achse, nicht einmal je Lupe — auch über mehrere Durchgänge (Bilder) hinweg. Dabei zeigt jede Lupe genau die Pixel, die
// ein frisches Dokument ohne gemerkte Linie zeigt (bitgleich), und der Spalt der Naht ist im Ziel wirklich zu sehen.
func TestLupenQuelleKantenlinieEinmal(t *testing.T) {
	d, lupen := merkerDokument(t, 400)
	vor := kantenlinienGerechnet.Load() // Tests dieses Pakets laufen nicht parallel (kein t.Parallel)
	bilder := make([]*image.NRGBA, len(lupen))
	for durchgang := 0; durchgang < 3; durchgang++ {
		for i, l := range lupen {
			bilder[i] = d.LupenQuelle(l).Zielbild(zielRaster(l))
		}
	}
	if n := kantenlinienGerechnet.Load() - vor; n > 2 {
		t.Errorf("%d Berechnungen der Kantenlinie für %d Lupen in 3 Durchgängen, höchstens 2 (eine je Achse) erlaubt", n, len(lupen))
	}
	// Bitgleich zu einem frischen Dokument (leerer Merker) für eine Auswahl der Lupen.
	spalt := false
	for i := 0; i < len(lupen); i += 37 {
		frisch, err := d.Abzug()
		if err != nil {
			t.Fatal(err)
		}
		soll := frisch.LupenQuelle(lupen[i]).Zielbild(zielRaster(lupen[i]))
		if !bytes.Equal(bilder[i].Pix, soll.Pix) {
			t.Fatalf("Lupe %d: Bild mit gemerkter Kantenlinie weicht vom frisch gerechneten ab", i)
		}
		for j := 3; j < len(soll.Pix); j += 4 {
			if soll.Pix[j] == 0 {
				spalt = true
			}
		}
	}
	if !spalt {
		t.Error("Aufbau: kein Zielpixel im Spalt der Naht, die Kantenlinie wirkt nicht")
	}
}

// TestLupenQuelleMerkerFolgtKante (F-693): Ändert sich die Form der Kante oder die Ansicht, rechnet der Merker neu; die
// Lupe zeigt dann dasselbe wie ein frisches Dokument.
func TestLupenQuelleMerkerFolgtKante(t *testing.T) {
	d, lupen := merkerDokument(t, 1)
	l := lupen[0]
	// Eine zweite Lupe weit rechts an derselben Naht, nur im ganzen Bild sichtbar.
	rechts := Lupe{Quelle: R(Pt(3900, 88), Pt(3903, 99)), Ziel: R(Pt(3900, 20), Pt(3906, 42))}
	vorher := d.LupenQuelle(l).Zielbild(zielRaster(l))
	pruefe := func(was string, l Lupe) {
		t.Helper()
		frisch, err := d.Abzug()
		if err != nil {
			t.Fatal(err)
		}
		ist := d.LupenQuelle(l).Zielbild(zielRaster(l))
		soll := frisch.LupenQuelle(l).Zielbild(zielRaster(l))
		if !bytes.Equal(ist.Pix, soll.Pix) {
			t.Errorf("%s: Lupe zeigt eine alte Kantenlinie", was)
		}
	}
	if err := d.SetzeKante(KanteZacken); err != nil {
		t.Fatal(err)
	}
	pruefe("andere Form", l)
	if bytes.Equal(vorher.Pix, d.LupenQuelle(l).Zielbild(zielRaster(l)).Pix) {
		t.Error("Aufbau: die Form der Kante ändert das Lupenbild nicht")
	}
	// Erst eine kurze Ansicht mit neuer Form (der Merker rechnet dort eine kurze Linie), dann wieder die ganze: Die Lupen
	// rechts brauchen die lange Linie (eine kurze bliebe an ihrem Ende stehen).
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 2000, 300)); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeKante(KanteWelle); err != nil {
		t.Fatal(err)
	}
	pruefe("kurze Ansicht", l)
	if err := d.SetzeZuschnitt(image.Rect(0, 0, 4096, 300)); err != nil {
		t.Fatal(err)
	}
	for x := 2100.0; x < 4080; x += 23 { // viele Stellen: Die Kante schwankt
		rechts.Quelle = R(Pt(x, 84), Pt(x+3, 99))
		rechts.Ziel = R(Pt(x, 20), Pt(x+6, 50))
		pruefe("wieder lange Ansicht", rechts)
	}
}
