// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"errors"
	"image"
	"image/color"
	"math"
	"testing"
)

// ADR-0029: SetzeAngaben ersetzt Objekte, Zuschnitt und Streifen in einem Rückgängig-Schritt, mit neuen Kennungen.
func TestSetzeAngabenEinSchritt(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 50, 40)))
	if err != nil {
		t.Fatal(err)
	}
	altID, err := d.Hinzufuegen(Text{Ort: Pt(2, 20), Inhalt: "alt", Schriftgrad: 10})
	if err != nil {
		t.Fatal(err)
	}
	vorher := d.Stand()
	neu := []Objekt{
		Text{Ort: Pt(5, 25), Inhalt: "neu", Schriftgrad: 12},
		Nummer{Mitte: Pt(20, 20), Radius: 6, Wert: 1, Farbe: color.NRGBA{R: 200, A: 255}},
	}
	z := image.Rect(2, 3, 45, 38)
	st := []Streifen{{Von: 10, Bis: 15}}
	if err := d.SetzeAngaben(neu, z, st); err != nil {
		t.Fatal(err)
	}
	o := d.Objekte()
	if len(o) != 2 || d.Zuschnitt() != z || len(d.Streifen()) != 1 {
		t.Fatalf("Angaben nicht gesetzt: %d %v %v", len(o), d.Zuschnitt(), d.Streifen())
	}
	if o[0].Kennung() == 0 || o[0].Kennung() == altID || o[0].Kennung() == o[1].Kennung() {
		t.Errorf("Kennungen %d %d (alt %d)", o[0].Kennung(), o[1].Kennung(), altID)
	}
	if err := d.Rueckgaengig(); err != nil {
		t.Fatal(err)
	}
	if d.Stand() != vorher || len(d.Objekte()) != 1 || d.Objekte()[0].Kennung() != altID || d.Zuschnitt() != image.Rect(0, 0, 50, 40) || len(d.Streifen()) != 0 {
		t.Error("ein Rückgängig führt nicht auf den Stand davor")
	}
	if err := d.Wiederholen(); err != nil || len(d.Objekte()) != 2 {
		t.Errorf("Wiederholen: %v", err)
	}
}

// Dieselben Grenzen wie die einzelnen Wege; bei einem Fehler bleibt das Dokument unverändert.
func TestSetzeAngabenGrenzen(t *testing.T) {
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 50, 40)))
	ganz := image.Rect(0, 0, 50, 40)
	faelle := map[string]func() error{
		"Zuschnitt außerhalb": func() error { return d.SetzeAngaben(nil, image.Rect(0, 0, 60, 40), nil) },
		"Streifen ungültig":   func() error { return d.SetzeAngaben(nil, ganz, []Streifen{{Von: 30, Bis: 10}}) },
		"Objekt ungültig": func() error {
			return d.SetzeAngaben([]Objekt{Text{Ort: Pt(math.NaN(), 1), Inhalt: "a", Schriftgrad: 10}}, ganz, nil)
		},
		"nil": func() error { return d.SetzeAngaben([]Objekt{nil}, ganz, nil) },
		"zu viele": func() error {
			return d.SetzeAngaben(make([]Objekt, MaxObjekte+1), ganz, nil)
		},
	}
	for name, f := range faelle {
		if err := f(); err == nil {
			t.Errorf("%s: angenommen", name)
		}
		if d.KannRueckgaengig() {
			t.Fatalf("%s: Dokument geändert", name)
		}
	}
	if _, err := d.BeginneNeuenZug(Text{Ort: Pt(2, 20), Inhalt: "z", Schriftgrad: 10}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAngaben(nil, ganz, nil); !errors.Is(err, ErrZugLaeuft) {
		t.Errorf("während eines Zugs: %v", err)
	}
}

// SchwaerzungenImVerlauf nennt die wirksamen Schwärzungen in Reihenfolge; Rückgängig nimmt sie heraus, VergissVerlauf
// leert die Liste; verwirft die Grenze des Verlaufs eine Schwärzung, ist die Liste unvollständig.
func TestSchwaerzungenImVerlauf(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	schwarz := Schwaerzung{Farbe: color.NRGBA{A: 255}}
	for _, r := range []image.Rectangle{image.Rect(0, 0, 2, 2), image.Rect(4, 4, 6, 6)} {
		if err := d.Schwaerzen(r, schwarz); err != nil {
			t.Fatal(err)
		}
	}
	l, ok := d.SchwaerzungenImVerlauf()
	if !ok || len(l) != 2 || l[0].Bereich != image.Rect(0, 0, 2, 2) || l[1].Bereich != image.Rect(4, 4, 6, 6) {
		t.Fatalf("Liste %v, vollständig %v", l, ok)
	}
	_ = d.Rueckgaengig()
	if l, _ := d.SchwaerzungenImVerlauf(); len(l) != 1 {
		t.Errorf("nach Rückgängig %d", len(l))
	}
	_ = d.VergissVerlauf()
	if l, ok := d.SchwaerzungenImVerlauf(); len(l) != 0 || !ok {
		t.Errorf("nach VergissVerlauf %v %v", l, ok)
	}
	verlaufGrenze = 20 // eine Schwärzung 2 x 2 hält 16 Bytes: die zweite verdrängt die erste
	for _, r := range []image.Rectangle{image.Rect(0, 0, 2, 2), image.Rect(4, 4, 6, 6)} {
		if err := d.Schwaerzen(r, schwarz); err != nil {
			t.Fatal(err)
		}
	}
	if l, ok := d.SchwaerzungenImVerlauf(); ok || len(l) != 1 {
		t.Errorf("verdrängte Schwärzung: Liste %d, vollständig %v", len(l), ok)
	}
	_ = d.VergissVerlauf()
	if _, ok := d.SchwaerzungenImVerlauf(); !ok {
		t.Error("nach VergissVerlauf weiter unvollständig")
	}
}

// V7-3: Nach einem gelungenen Speichern ist die Liste wieder vollständig, auch wenn der Verlauf dabei nicht geleert wurde.
func TestSchwaerzungenGesichertSetztZurueck(t *testing.T) {
	alt := verlaufGrenze
	defer func() { verlaufGrenze = alt }()
	verlaufGrenze = 20
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	schwarz := Schwaerzung{Farbe: color.NRGBA{A: 255}}
	for _, r := range []image.Rectangle{image.Rect(0, 0, 2, 2), image.Rect(4, 4, 6, 6)} {
		if err := d.Schwaerzen(r, schwarz); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := d.SchwaerzungenImVerlauf(); ok {
		t.Fatal("Vorbedingung: unvollständig")
	}
	d.SchwaerzungenGesichert()
	if _, ok := d.SchwaerzungenImVerlauf(); !ok {
		t.Error("nach dem Speichern weiter unvollständig")
	}
}

// V7-4: Eine riesige Blockgröße (etwa aus einer präparierten Rettungskopie) wird auf den Bereich geklemmt, statt
// bloecke überlaufen zu lassen.
func TestSchwaerzenRiesigerBlock(t *testing.T) {
	d, _ := Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err := d.Schwaerzen(image.Rect(0, 0, 20, 16), Schwaerzung{Block: math.MaxInt - 3}); err != nil {
		t.Fatal(err)
	}
	if l, _ := d.SchwaerzungenImVerlauf(); len(l) != 1 || l[0].Art.Block != 20 {
		t.Errorf("Block nicht geklemmt: %+v", l)
	}
}
