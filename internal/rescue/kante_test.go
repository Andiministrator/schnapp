// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// F-419: Die Form der Stauch-Kante übersteht den Weg Dokument → Rettungskopie → Datei → Anwenden. Das Zieldokument
// startet mit der Voreinstellung (tear); nur wenn die Kopie die Form trägt und Anwenden sie setzt, ist sie danach
// gewechselt. Jede Form wird einmal durchlaufen, auch die, die der Voreinstellung gleicht.
func TestRundlaufTraegtKantenform(t *testing.T) {
	for _, form := range []document.Kante{document.KanteWelle, document.KanteZacken, document.KanteAbriss} {
		t.Run(form.Name(), func(t *testing.T) {
			pfad, k := ausgang(t, "datei")
			ort := filepath.Join(t.TempDir(), "state", "schnapp", "rescue")
			original := bild(64, 64)
			d, err := document.Neu(original)
			if err != nil {
				t.Fatal(err)
			}
			bearbeite(t, d)
			// Gegenform zuerst, damit auch der Wechsel auf die Voreinstellung ein Schritt ist.
			if err := d.SetzeKante(form.Naechste()); err != nil {
				t.Fatal(err)
			}
			if err := d.SetzeKante(form); err != nil {
				t.Fatal(err)
			}
			kp, err := Erzeuge(d, k, time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC))
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
			// Das Zieldokument trägt eine andere Form als die Kopie, damit „Form des Ziels behalten“ auffällt.
			if err := neu.VorgabeKante(form.Naechste()); err != nil {
				t.Fatal(err)
			}
			if err := Anwenden(neu, gelesen, nil); err != nil {
				t.Fatal(err)
			}
			if neu.Kante() != form {
				t.Errorf("Form nach Anwenden %v, erwartet %v", neu.Kante(), form)
			}
		})
	}
}
