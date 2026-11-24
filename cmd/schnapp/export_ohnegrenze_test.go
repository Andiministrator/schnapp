// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// F-610 (Karte #234, ADR-0033 Nachtrag 4): --ignore-limit gilt auch für den Export aus dem Editor. oeffneImEditor reicht
// den Schalter an die Oberfläche durch, jede andere Öffnung setzt ihn zurück.
func TestOeffnenReichtOhneGrenzeAnExportDurch(t *testing.T) {
	i18n.Setze(nil)
	rettungsLauf(t) // ersetzt das Fenster durch eine Attrappe
	t.Cleanup(func() { ui.SetzeExportOhneGrenze(false) })
	dir := t.TempDir()
	ueber, _ := ueberBudget(t, dir)
	if err := oeffneImEditor(ueber, true, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	if !ui.ExportOhneGrenze() {
		t.Error("mit --ignore-limit geöffnet: der Export des Editors prüft das Budget noch")
	}

	einfach := filepath.Join(dir, "einfach.png")
	f, err := os.Create(einfach)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := oeffneImEditor(einfach, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	if ui.ExportOhneGrenze() {
		t.Error("ohne --ignore-limit geöffnet: der Schalter blieb vom vorigen Editor hängen")
	}
}
