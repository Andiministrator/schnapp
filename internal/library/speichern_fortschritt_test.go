// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"image"
	"math/rand"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// rauschBild liefert ein schlecht komprimierbares Bild (die PNG-Daten sind deutlich größer als ein Stück des Kodierers).
func rauschBild(w, h int) *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	rand.New(rand.NewSource(298)).Read(b.Pix)
	for i := 3; i < len(b.Pix); i += 4 {
		b.Pix[i] = 255
	}
	return b
}

// F-298 (Verify B-2, Mutation L1): Speichern meldet schon beim Lesen der alten Datei Fortschritt, also vor der Marke
// „alte-datei-gelesen“.
func TestSpeichernMeldetFortschrittBeimLesen(t *testing.T) {
	heim := t.TempDir()
	o, err := OrteAus(func(k string) string {
		if k == "HOME" {
			return heim
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	bib := Neu(o)
	bib.Jetzt = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }
	d, err := document.Neu(rauschBild(400, 300))
	if err != nil {
		t.Fatal(err)
	}
	sicht, err := render.Bild(d)
	if err != nil {
		t.Fatal(err)
	}
	e, err := bib.Ablegen(d, sicht, fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	stand, err := StandVon(bib.Orte.Ablage + "/" + e.Pfad)
	if err != nil {
		t.Fatal(err)
	}
	var beimLesen uint64
	saveprogress.Haken = func(s string) {
		if s == "alte-datei-gelesen" {
			beimLesen = saveprogress.Pulse()
		}
	}
	t.Cleanup(func() { saveprogress.Haken = nil })
	vorher := saveprogress.Pulse()
	if _, _, err := bib.Speichern(e.Pfad, stand, d, sicht); err != nil {
		t.Fatal(err)
	}
	if beimLesen <= vorher {
		t.Errorf("kein Fortschritt beim Lesen der alten Datei (%d → %d)", vorher, beimLesen)
	}
}

// F-298 (Verify B-2, Mutation L2): Das Verkleinern für die Vorschau meldet Fortschritt.
func TestVerkleinereMeldetFortschritt(t *testing.T) {
	vorher := saveprogress.Pulse()
	verkleinere(rauschBild(800, 600), 100)
	if saveprogress.Pulse() == vorher {
		t.Error("kein Fortschritt beim Verkleinern")
	}
}
