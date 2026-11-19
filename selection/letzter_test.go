// SPDX-License-Identifier: GPL-3.0-or-later

package selection

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
)

// F-85: Die Skala wird mitgespeichert und geladen.
func TestLetzterSkalaRundlauf(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "schnapp", "last-area.json")
	l := Letzter{Monitor: "M", Groesse: image.Pt(100, 50), Skala: 1.5, Bereich: image.Rect(1, 2, 30, 40)}
	if err := SpeichereLetzten(pfad, l); err != nil {
		t.Fatal(err)
	}
	g, err := LadeLetzten(pfad)
	if err != nil || g != l || g.Skala != 1.5 {
		t.Fatalf("geladen %+v, %v", g, err)
	}
	if err := SpeichereLetzten(pfad, Letzter{Monitor: "M", Groesse: image.Pt(100, 50), Bereich: image.Rect(1, 2, 30, 40)}); err == nil {
		t.Error("ohne Skala wurde gespeichert")
	}
}

// Altes Format (vor F-85) ohne Skala ist unbrauchbar: neu wählen.
func TestLetzterAltesFormatOhneSkala(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "last-area.json")
	alt := `{"monitor":"M","size":{"X":100,"Y":50},"area":{"Min":{"X":1,"Y":2},"Max":{"X":30,"Y":40}}}`
	if err := os.WriteFile(pfad, []byte(alt), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LadeLetzten(pfad); !errors.Is(err, ErrKeinLetzter) {
		t.Errorf("altes Format: %v", err)
	}
}

func TestLetzterSkalaPasst(t *testing.T) {
	l := Letzter{Skala: 1.25}
	for skala, soll := range map[float64]bool{1.25: true, 1.2500001: true, 1.0: false, 2: false, 1.26: false} {
		if l.SkalaPasst(skala) != soll {
			t.Errorf("SkalaPasst(%v) = %v, Soll %v", skala, !soll, soll)
		}
	}
}
