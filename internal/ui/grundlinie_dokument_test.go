// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-342: Die Grundlinien, die zeichneObjekte benutzt, stimmen mit denen des Dokuments (und damit des Renderers) überein.
func TestEditorGrundlinieWieDokument(t *testing.T) {
	tx := document.Text{Ort: document.Pt(7, 11), Inhalt: "a\nb\nc", Schriftgrad: 40}
	for i := 0; i < 3; i++ {
		if got, soll := float64(textGrundlinieAnsicht(tx.Schriftgrad, i)), tx.Grundlinie(i)-tx.Ort.Y; math.Abs(got-soll) > 1e-3 {
			t.Errorf("Zeile %d: %v, Soll %v", i, got, soll)
		}
	}
	n := document.Nummer{Mitte: document.Pt(50, 50), Radius: 20, Wert: 3}
	if v := nummerGrundlinieAnsicht(n); v <= 0 || math.Abs(float64(v)-(n.Grundlinie()-50)) > 1e-3 {
		t.Errorf("Nummer: %v", v)
	}
}
