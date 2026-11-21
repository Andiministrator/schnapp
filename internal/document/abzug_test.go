// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"reflect"
	"testing"
)

// F-143: Der Abzug zeigt denselben Inhalt wie das Dokument (Ansichtsbild, Objekte, Ansicht, Stände) und ist danach
// von ihm unabhängig, in beide Richtungen.
func TestAbzugGleichUndUnabhaengig(t *testing.T) {
	bild := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for i := range bild.Pix {
		bild.Pix[i] = uint8(i * 7)
	}
	d, err := Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	rot := Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}
	if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(2, 2), Pt(12, 12)), Strich: rot}); err != nil {
		t.Fatal(err)
	}
	if err := d.Schwaerzen(image.Rect(0, 0, 10, 10), Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeAnsicht(image.Rect(1, 1, 39, 29), []Streifen{{Zeilen: true, Von: 10, Bis: 14}}); err != nil {
		t.Fatal(err)
	}

	a, err := d.Abzug()
	if err != nil {
		t.Fatal(err)
	}
	if a.Stand() != d.Stand() || a.SchwaerzStand() != d.SchwaerzStand() || a.Geschwaerzt() != d.Geschwaerzt() ||
		a.Zuschnitt() != d.Zuschnitt() || !reflect.DeepEqual(a.Streifen(), d.Streifen()) ||
		!reflect.DeepEqual(a.Objekte(), d.Objekte()) || a.NaechsteNummer() != d.NaechsteNummer() ||
		a.Groesse() != d.Groesse() {
		t.Fatal("Abzug weicht vom Dokument ab")
	}
	if !bytes.Equal(a.Ansichtsbild().Pix, d.Ansichtsbild().Pix) || !bytes.Equal(a.OriginalBild().Pix, d.OriginalBild().Pix) {
		t.Fatal("Pixel des Abzugs weichen ab")
	}
	if a.KannRueckgaengig() {
		t.Error("Abzug trägt einen Verlauf")
	}
	vorher := append([]byte(nil), a.OriginalBild().Pix...)
	standVorher := a.Stand()

	// Das Dokument ändert sich weiter: Der Abzug bleibt, wie er war.
	if err := d.Schwaerzen(image.Rect(20, 20, 30, 30), Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(20, 2), Pt(30, 12)), Strich: rot}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeStreifen(nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.OriginalBild().Pix, vorher) || len(a.Objekte()) != 1 || len(a.Streifen()) != 1 || a.Stand() != standVorher {
		t.Error("Änderung am Dokument erreicht den Abzug")
	}
	// Und umgekehrt.
	n := len(d.Objekte())
	if _, err := a.Hinzufuegen(Kasten{Bereich: R(Pt(1, 1), Pt(3, 3)), Strich: rot}); err != nil {
		t.Fatal(err)
	}
	if len(d.Objekte()) != n {
		t.Error("Änderung am Abzug erreicht das Dokument")
	}
}

// Während eines Zugs gibt es keinen Abzug: Der Inhalt stünde zwischen zwei Ständen.
func TestAbzugNichtWaehrendZug(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.BeginneNeuenZug(Kasten{Bereich: R(Pt(1, 1), Pt(4, 4)), Strich: Strich{Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Abzug(); !errors.Is(err, ErrZugLaeuft) {
		t.Errorf("Abzug im Zug: %v", err)
	}
}

// F-282: Der Abzug teilt kein Feld des Objekte-Slices mit dem Dokument. Entfernen und Rückgängig schieben Elemente im
// Slice des Dokuments; teilte der Abzug dessen Feld, sähe die Speicher-Goroutine einen verschobenen Inhalt (und
// umgekehrt).
func TestAbzugTeiltKeinenObjekteSlice(t *testing.T) {
	d, err := Neu(image.NewNRGBA(image.Rect(0, 0, 40, 40)))
	if err != nil {
		t.Fatal(err)
	}
	rot := Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}
	var ids []int
	for i := 0; i < 4; i++ {
		id, err := d.Hinzufuegen(Kasten{Bereich: R(Pt(float64(i*8), 1), Pt(float64(i*8+6), 7)), Strich: rot})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	a, err := d.Abzug()
	if err != nil {
		t.Fatal(err)
	}
	vorher := a.Objekte()
	if err := d.Entfernen(ids[0]); err != nil { // schiebt die übrigen im Slice des Dokuments nach vorn
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Objekte(), vorher) {
		t.Error("Entfernen im Dokument verändert die Objekte des Abzugs")
	}
	if err := d.Rueckgaengig(); err != nil { // fügt es an der alten Stelle wieder ein
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Objekte(), vorher) {
		t.Error("Rückgängig im Dokument verändert die Objekte des Abzugs")
	}
	// Umgekehrt: Der Abzug schiebt Elemente, das Dokument bleibt.
	soll := d.Objekte()
	if err := a.Entfernen(ids[1]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Objekte(), soll) {
		t.Error("Entfernen im Abzug verändert die Objekte des Dokuments")
	}
}
