// SPDX-License-Identifier: GPL-3.0-or-later

package render

import (
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-136: Viele Streifen (an der Grenze) und viele Pfeile dürfen das Prüfen beim Öffnen nicht aufhalten.
func TestPruefeVieleStreifenUndPfeile(t *testing.T) {
	if testing.Short() {
		t.Skip("kurzer Lauf")
	}
	d := weissesDokument(t, 32768, 16)
	var st []document.Streifen
	for i := 0; i < document.MaxStreifen; i++ {
		st = append(st, document.Streifen{Zeilen: false, Von: 2 * i, Bis: 2*i + 1})
	}
	if err := d.SetzeStreifen(st); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		hinzu(t, d, document.Pfeil{A: document.Pt(1, 1), B: document.Pt(30000, 8), K: document.Pt(3, 6),
			Strich: document.Strich{Farbe: rot, Breite: 0.5}})
	}
	s := time.Now()
	_ = Pruefe(d) // abgelehnt oder geprüft, beides gilt; es kommt auf die Dauer an
	dt := time.Since(s)
	t.Logf("Pruefe: %v", dt)
	if pruefeStreifenZeit > 0 && dt > pruefeStreifenZeit {
		t.Errorf("Pruefe dauert %v, soll unter %v bleiben", dt, pruefeStreifenZeit)
	}
}
