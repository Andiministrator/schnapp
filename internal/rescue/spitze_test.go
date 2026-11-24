// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 13: Die Rettungskopie trägt die Spitze einer Sprechblase mit (die Angaben sind ein snAP-SVG).
func TestRundlaufSpitze(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := filepath.Join(t.TempDir(), "state", "schnapp", "rescue")
	original := bild(64, 64)
	d, err := document.Neu(original)
	if err != nil {
		t.Fatal(err)
	}
	sp := document.Pt(50, 55.5)
	if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(4, 4), Inhalt: "Hi", Schriftgrad: 10, Spitze: &sp}); err != nil {
		t.Fatal(err)
	}
	kp, err := Erzeuge(d, k, time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := Schreibe(ort, kp); err != nil {
		t.Fatal(err)
	}
	gelesen, err := Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	neu, err := document.Neu(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := Anwenden(neu, gelesen, nil); err != nil {
		t.Fatal(err)
	}
	o := neu.Objekte()
	if len(o) != 1 {
		t.Fatalf("%d Objekte", len(o))
	}
	if tx := o[0].(document.Text); tx.Spitze == nil || *tx.Spitze != sp {
		t.Errorf("Spitze nach der Rettung: %v", tx.Spitze)
	}
}
