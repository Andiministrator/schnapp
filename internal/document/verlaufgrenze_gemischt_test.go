// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// momentaufnahme hält fest, was ein Verlaufsstand ausmacht: Objekte, Zuschnitt und die Pixel des Originals.
type momentaufnahme struct {
	objekte   int
	zuschnitt image.Rectangle
	pixel     []byte
}

func nimmAuf(d *Dokument) momentaufnahme {
	return momentaufnahme{
		objekte:   len(d.Objekte()),
		zuschnitt: d.Zuschnitt(),
		pixel:     append([]byte(nil), d.Original().Pix...),
	}
}

func (a momentaufnahme) gleich(b momentaufnahme) bool {
	return a.objekte == b.objekte && a.zuschnitt == b.zuschnitt && bytes.Equal(a.pixel, b.pixel)
}

// F-323 c: Die Verlaufsgrenze wirft auch Objekt- und Zuschnittschritte weg (nur die ältesten, und nur so, dass der
// Stand darunter erhalten bleibt). Hier wird das Verhalten für eine gemischte Folge festgenagelt: Rückgängig führt nur
// auf Stände, die es wirklich gab und deren Inhalt stimmt; Stand() ist nach dem Verwerfen und nach jedem Rückgängig
// und Wiederholen stimmig; Wiederholen kehrt zum Endstand zurück. Ob das Verwerfen gemeldet werden soll, ist F-327.
func TestVerlaufGrenzeGemischteFolge(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 40 // eine Schwärzung 2 x 2 hält 16 Bytes: zwei passen, drei nicht

	weiss := image.NewNRGBA(image.Rect(0, 0, 12, 4))
	for i := range weiss.Pix {
		weiss.Pix[i] = 255
	}
	d, err := Neu(weiss)
	if err != nil {
		t.Fatal(err)
	}
	schwarz := Schwaerzung{Farbe: color.NRGBA{0, 0, 0, 255}}
	kasten := func(n int) error {
		_, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(float64(n), 1), Pt(float64(n)+2, 3)), Fuellung: color.NRGBA{A: 255}})
		return err
	}
	schritte := []func() error{
		func() error { return kasten(1) },                                     // Objekt
		func() error { return d.SetzeZuschnitt(image.Rect(0, 0, 12, 3)) },     // Zuschnitt
		func() error { return d.Schwaerzen(image.Rect(0, 0, 2, 2), schwarz) }, // Pixel
		func() error { return kasten(4) },                                     // Objekt
		func() error { return d.Schwaerzen(image.Rect(2, 0, 4, 2), schwarz) }, // Pixel
		func() error { return d.SetzeZuschnitt(image.Rect(0, 0, 10, 3)) },     // Zuschnitt
		func() error { return d.Schwaerzen(image.Rect(4, 0, 6, 2), schwarz) }, // Pixel
		func() error { return kasten(7) },                                     // Objekt
		func() error { return d.Schwaerzen(image.Rect(6, 0, 8, 2), schwarz) }, // Pixel
	}
	aufnahmen := map[uint64]momentaufnahme{d.Stand(): nimmAuf(d)}
	for i, s := range schritte {
		if err := s(); err != nil {
			t.Fatalf("Schritt %d: %v", i, err)
		}
		if _, doppelt := aufnahmen[d.Stand()]; doppelt {
			t.Fatalf("Schritt %d: Stand %d kehrt wieder", i, d.Stand())
		}
		aufnahmen[d.Stand()] = nimmAuf(d)
	}
	endStand := d.Stand()
	endAufnahme := aufnahmen[endStand]

	// Die Grenze hat Schritte verworfen, der jüngste Schritt bleibt, die Pixelkopien liegen innerhalb der Grenze.
	n := len(d.verlauf.erledigt)
	if n < 1 || n >= len(schritte) {
		t.Fatalf("Verlauf hält %d von %d Schritten: erwartet teilweise verworfen, mindestens einer bleibt", n, len(schritte))
	}
	var summe int64
	for _, s := range d.verlauf.erledigt {
		if g, ok := s.b.(groessenBefehl); ok {
			summe += g.pixelBytes()
		}
	}
	if summe > verlaufGrenze {
		t.Errorf("Pixelkopien im Verlauf %d Bytes, Grenze %d", summe, verlaufGrenze)
	}
	if d.verlauf.basis == 0 {
		t.Error("basis blieb 0, obwohl Schritte verworfen wurden")
	}
	if !nimmAuf(d).gleich(endAufnahme) {
		t.Error("das Verwerfen hat den Inhalt verändert")
	}

	// Rückgängig, soweit es geht: jeder erreichte Stand ist ein echter früherer Stand mit genau dem damaligen Inhalt.
	zurueck := 0
	for d.KannRueckgaengig() {
		if err := d.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
		zurueck++
		a, ok := aufnahmen[d.Stand()]
		if !ok {
			t.Fatalf("nach %d x Rückgängig: Stand %d hat es nie gegeben", zurueck, d.Stand())
		}
		if !nimmAuf(d).gleich(a) {
			t.Errorf("nach %d x Rückgängig: Inhalt passt nicht zum Stand %d", zurueck, d.Stand())
		}
	}
	if zurueck != n {
		t.Errorf("%d x Rückgängig möglich, der Verlauf hielt %d Schritte", zurueck, n)
	}
	if d.Stand() != d.verlauf.basis {
		t.Errorf("Stand %d am Anfang des Verlaufs, erwartet basis %d", d.Stand(), d.verlauf.basis)
	}
	if err := d.Rueckgaengig(); err != ErrNichtsDa {
		t.Errorf("Rückgängig am Anfang: %v, erwartet ErrNichtsDa", err)
	}

	// Wiederholen bis zum Ende: Stand und Inhalt kehren zum Endstand zurück.
	for i := 0; i < zurueck; i++ {
		if err := d.Wiederholen(); err != nil {
			t.Fatal(err)
		}
		if a, ok := aufnahmen[d.Stand()]; !ok || !nimmAuf(d).gleich(a) {
			t.Fatalf("nach %d x Wiederholen: Stand %d oder Inhalt stimmt nicht", i+1, d.Stand())
		}
	}
	if d.Stand() != endStand || !nimmAuf(d).gleich(endAufnahme) {
		t.Errorf("Endstand nach Wiederholen: %d, erwartet %d", d.Stand(), endStand)
	}
	if err := d.Wiederholen(); err != ErrNichtsDa {
		t.Errorf("Wiederholen am Ende: %v, erwartet ErrNichtsDa", err)
	}

	// Eine neue Änderung nach teilweisem Rückgängig verwirft das Wiederholen und bekommt eine neue Kennung.
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if err := kasten(9); err != nil {
		t.Fatal(err)
	}
	if _, doppelt := aufnahmen[d.Stand()]; doppelt {
		t.Errorf("neue Änderung bekam eine alte Kennung %d", d.Stand())
	}
	if d.KannWiederholen() {
		t.Error("Wiederholen nach neuer Änderung noch möglich")
	}
}
