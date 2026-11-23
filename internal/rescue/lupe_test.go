// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"image/color"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0033 Punkt 6: Die Rettungskopie trägt die Lupe über das snAP-SVG, ohne Pixel; Dokument → Kopie → Datei →
// Anwenden ergibt dieselbe Lupe.
func TestRundlaufTraegtLupe(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := filepath.Join(t.TempDir(), "state", "schnapp", "rescue")
	original := bild(64, 64)
	d, err := document.Neu(original)
	if err != nil {
		t.Fatal(err)
	}
	l := document.Lupe{Ebene: document.Ebene{Name: "Zoom", Gesperrt: true}, Quelle: document.R(document.Pt(2, 3), document.Pt(10.5, 9)),
		Ziel: document.R(document.Pt(30, 30), document.Pt(47, 42)), Strich: document.Strich{Farbe: color.NRGBA{9, 8, 7, 255}, Breite: 3}}
	if _, err := d.Hinzufuegen(l); err != nil {
		t.Fatal(err)
	}
	kp, err := Erzeuge(d, k, time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(kp.Angaben, `s:typ="lupe"`) || strings.Contains(kp.Angaben, "data:") {
		t.Fatalf("Angaben ohne Lupe oder mit Pixeln: %.400s", kp.Angaben)
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
	l.ID = 1
	if !reflect.DeepEqual(neu.Objekte(), []document.Objekt{l}) {
		t.Errorf("nach Anwenden %#v, erwartet %#v", neu.Objekte(), l)
	}
}
