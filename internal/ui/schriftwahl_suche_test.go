// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// TestSchriftListeNachSuche (F-575): Eine Liste, die geöffnet wird, bevor die Suche nach Systemschriften fertig ist,
// zeigt zuerst nur Go und Go Mono und nach dem Ende der Suche alle Familien; der Cursor bleibt auf demselben Namen.
func TestSchriftListeNachSuche(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	for i := 0; i < 500 && !fonts.SucheFertig(); i++ { // eine etwaige Suche des Aufbaus abwarten
		time.Sleep(10 * time.Millisecond)
	}
	fonts.SetzeOrdner([]string{"../../testdata/fonts"})
	fonts.SucheZuruecksetzen()
	t.Cleanup(func() {
		fonts.SetzeOrdner(nil)
		fonts.SucheZuruecksetzen()
		fonts.Suche()
	})
	sw := &l.e.seite.sw
	l.klick(mittenPunkt(sw.feldRahmen))
	if !sw.offen || len(sw.familien) != 2 {
		t.Fatalf("vor dem Ende der Suche: offen %v, Familien %v", sw.offen, sw.familien)
	}
	sw.cursor = 1 // Go Mono
	fonts.Suche()
	l.setle()
	if !sw.offen || len(sw.familien) != 3 || sw.familien[2] != "Schnapp Testkern" {
		t.Fatalf("nach dem Ende der Suche: offen %v, Familien %v", sw.offen, sw.familien)
	}
	if sw.familien[sw.cursor] != fonts.GoMono {
		t.Errorf("Cursor auf %q, Soll %q", sw.familien[sw.cursor], fonts.GoMono)
	}
	if len(sw.eintraege) < 3 || sw.zeilen[2].Empty() {
		t.Errorf("neue Zeile nicht gezeichnet: Einträge %d, Zeilen %v", len(sw.eintraege), sw.zeilen)
	}
	l.klick(mittenPunkt(sw.zeilen[2]))
	if sw.offen || l.text().Schrift != "Schnapp Testkern" {
		t.Errorf("Klick auf die neue Familie: offen %v, Schrift %q", sw.offen, l.text().Schrift)
	}
}
