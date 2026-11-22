// SPDX-License-Identifier: GPL-3.0-or-later

package document_test

// Flächenbudget der Rettungskopie beim Schreiben und beim Anwenden (F-431, F-434 b). Liegt im Paket document_test,
// weil nur hier ein Test die Grenze des Verlaufs klein setzen kann (document.SetzeVerlaufGrenze).

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/rescue"
)

var schwarzBudget = document.Schwaerzung{Farbe: color.NRGBA{A: 255}}

// F-431: Hält der Verlauf eine einzelne Schwärzung über dem Budget (der jüngste Schritt bleibt immer, der Merker für
// verworfene Schwärzungen steht nicht), schreibt die Sicherung keine Kopie, die beim Öffnen abgelehnt würde: Die
// vorige, anwendbare bleibt stehen, die Diagnose nennt ErrZuGross. An der Grenze selbst wird geschrieben und angewendet.
func TestRettungskopieUeberDemBudget(t *testing.T) {
	zurueck := document.SetzeVerlaufGrenze(20) // 3 x 3 Pixel halten 36 Bytes: über der Grenze
	defer zurueck()

	pfad := filepath.Join(t.TempDir(), "aufnahme.png")
	if err := os.WriteFile(pfad, []byte("datei"), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := rescue.KennungVon(pfad)
	if err != nil {
		t.Fatal(err)
	}
	ort := t.TempDir()
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if _, err := d.Hinzufuegen(document.GeraderPfeil(document.Pt(1, 1), document.Pt(8, 8),
		document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2})); err != nil {
		t.Fatal(err)
	}
	s := rescue.NeueSicherung(ort, k)
	var diagnose []error
	s.Diagnose = func(err error) { diagnose = append(diagnose, err) }
	s.Sichere(d)
	s.Ende()
	kopie := filepath.Join(ort, rescue.Name(pfad))
	vorher, err := os.ReadFile(kopie)
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Schwaerzen(image.Rect(0, 0, 3, 3), schwarzBudget); err != nil {
		t.Fatal(err)
	}
	if _, voll := d.SchwaerzungenImVerlauf(); !voll {
		t.Fatal("Vorbedingung: Liste vollständig (sonst greift ErrUnfertig, nicht F-431)")
	}
	if _, err := rescue.Erzeuge(d, k, time.Now()); !errors.Is(err, rescue.ErrZuGross) {
		t.Errorf("Erzeuge über dem Budget: %v, erwartet ErrZuGross", err)
	}
	s.Sichere(d)
	s.Ende()
	if nachher, err := os.ReadFile(kopie); err != nil || !bytes.Equal(nachher, vorher) {
		t.Errorf("vorige Kopie überschrieben (%v)", err)
	}
	if len(diagnose) != 1 || !errors.Is(diagnose[0], rescue.ErrZuGross) {
		t.Errorf("Diagnose %v, erwartet einmal ErrZuGross", diagnose)
	}
	kp, err := rescue.Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	frisch, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if err := rescue.Anwenden(frisch, kp, nil); err != nil || len(frisch.Objekte()) != 1 {
		t.Errorf("vorige Kopie nicht anwendbar: %v", err)
	}

	// Genau an der Grenze: Erzeuge schreibt, Anwenden auf ein frisch geöffnetes Dokument nimmt an.
	zurueck36 := document.SetzeVerlaufGrenze(36)
	defer zurueck36()
	kp2, err := rescue.Erzeuge(d, k, time.Now())
	if err != nil {
		t.Fatalf("Erzeuge an der Grenze: %v", err)
	}
	frisch2, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if err := rescue.Anwenden(frisch2, &kp2, nil); err != nil {
		t.Errorf("Anwenden an der Grenze: %v", err)
	}
}

// F-434 (b): Anwenden rechnet mit dem Restplatz des Verlaufs (VerlaufPlatz), nicht mit der festen Grenze: Hält das
// Dokument schon eine Schwärzung, passt eine Kopie, die den Rest genau füllt, eine mit vier Bytes mehr nicht.
func TestAnwendenRechnetMitBelegtemVerlauf(t *testing.T) {
	zurueck := document.SetzeVerlaufGrenze(64)
	defer zurueck()
	quelle, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	kp, err := rescue.Erzeuge(quelle, rescue.Kennung{Pfad: "/x"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ { // 3 x 16 Bytes = 48 = Restplatz
		kp.Schwaerzungen = append(kp.Schwaerzungen, rescue.Schwaerzung{Bereich: [4]int{0, 0, 2, 2}, Farbe: "#000000ff"})
	}
	belegt := func() *document.Dokument {
		d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
		if err := d.Schwaerzen(image.Rect(4, 4, 6, 6), schwarzBudget); err != nil { // 16 Bytes im Verlauf
			t.Fatal(err)
		}
		return d
	}
	if err := rescue.Anwenden(belegt(), &kp, nil); err != nil {
		t.Errorf("Kopie füllt den Restplatz genau: %v", err)
	}
	kp.Schwaerzungen = append(kp.Schwaerzungen, rescue.Schwaerzung{Bereich: [4]int{0, 0, 1, 1}, Farbe: "#000000ff"})
	d := belegt()
	stand := d.Stand()
	if err := rescue.Anwenden(d, &kp, nil); !errors.Is(err, rescue.ErrZuGross) {
		t.Errorf("Kopie knapp über dem Restplatz: %v, erwartet ErrZuGross", err)
	}
	if d.Stand() != stand {
		t.Error("Dokument trotz Ablehnung geändert")
	}
}
