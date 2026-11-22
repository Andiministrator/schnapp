// SPDX-License-Identifier: GPL-3.0-or-later

package document_test

import (
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

// F-385 (Verify 7, R11): Hat die Grenze des Verlaufs eine Schwärzung verworfen, schreibt die Rettungskopie nicht; die
// zuletzt geschriebene bleibt stehen. Eine Kopie ohne die verdrängte Schwärzung trüge sonst einen Stand, der beim
// Wiederherstellen eine Schwärzung verlöre. Liegt hier (Paket document_test), weil nur ein Test im Paket document die
// Grenze des Verlaufs setzen kann.
func TestRettungskopieBeiUnvollstaendigemVerlauf(t *testing.T) {
	zurueck := document.SetzeVerlaufGrenze(20) // eine Schwärzung 2 x 2 hält 16 Bytes: die zweite verdrängt die erste
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
	schwarz := document.Schwaerzung{Farbe: color.NRGBA{A: 255}}
	if err := d.Schwaerzen(image.Rect(0, 0, 2, 2), schwarz); err != nil {
		t.Fatal(err)
	}
	s := rescue.NeueSicherung(ort, k)
	var diagnose []error
	s.Diagnose = func(err error) { diagnose = append(diagnose, err) }
	s.Sichere(d)
	s.Ende()
	if err := d.Schwaerzen(image.Rect(4, 4, 6, 6), schwarz); err != nil {
		t.Fatal(err)
	}
	if _, err := rescue.Erzeuge(d, k, time.Now()); !errors.Is(err, rescue.ErrUnfertig) {
		t.Errorf("Erzeuge bei verdrängter Schwärzung: %v, erwartet ErrUnfertig", err)
	}
	s.Sichere(d)
	s.Ende()
	if len(diagnose) != 1 || !errors.Is(diagnose[0], rescue.ErrUnfertig) {
		t.Errorf("Diagnose %v, erwartet einmal ErrUnfertig", diagnose)
	}
	kp, err := rescue.Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	if len(kp.Schwaerzungen) != 1 || kp.Schwaerzungen[0].Bereich != [4]int{0, 0, 2, 2} {
		t.Errorf("Kopie überschrieben: %+v, erwartet die erste mit einer Schwärzung", kp.Schwaerzungen)
	}
}
