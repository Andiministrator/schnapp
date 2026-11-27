// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// ADR-0041 Punkt 4: neuerEditor liest tools.ini, das Schließen schreibt sie nur, wenn sich etwas geändert hat, und der
// Wechsel der Aufnahme nimmt die Stile mit.
func TestWerkzeugStileUeberSitzungen(t *testing.T) {
	konfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", konfig)
	pfad := filepath.Join(konfig, "schnapp", "tools.ini")
	if err := settings.SchreibeWerkzeugStile([]settings.WerkzeugStil{{Stil: settings.Stil{Art: settings.ArtKasten,
		Farbe: canvas.Beschriftungsfarben[2], Breite: 9}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pfad); err != nil {
		t.Fatalf("tools.ini liegt nicht am erwarteten Ort: %v", err)
	}
	neu := func() *editor {
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 400, 300)))
		if err != nil {
			t.Fatal(err)
		}
		e := neuerEditor("datei.png", d, nil, nil)
		t.Cleanup(func() { e.rettungEnde() })
		return e
	}
	e := neu()
	e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	if e.s.Stil.Breite != 9 || e.s.Stil.Farbe != canvas.Beschriftungsfarben[2] {
		t.Fatalf("Kasten nach dem Öffnen: %+v", e.s.Stil)
	}

	// Unverändert schließen: die Datei bleibt dieselbe (atomares Schreiben legte eine neue an).
	vorher, err := os.Stat(pfad)
	if err != nil {
		t.Fatal(err)
	}
	e.werkzeugStileSchreiben()
	if nachher, err := os.Stat(pfad); err != nil || !os.SameFile(vorher, nachher) {
		t.Fatalf("ohne Änderung neu geschrieben (%v)", err)
	}

	// Geändert schließen: das nächste Fenster beginnt damit.
	e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	e.s.SetzeBreite(14)
	if err := e.zerstoert(nil); err != nil {
		t.Fatal(err)
	}
	e2 := neu()
	e2.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	if e2.s.Stil.Breite != 14 {
		t.Fatalf("Pfeil im nächsten Fenster: %v", e2.s.Stil.Breite)
	}
	e2.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	if e2.s.Stil.Breite != 9 {
		t.Fatalf("Kasten im nächsten Fenster: %v", e2.s.Stil.Breite)
	}

	// Wechsel der Aufnahme: die neue Sitzung hat die Stile der alten.
	d, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	e2.uebernimm(Geoeffnet{Titel: "neu.png", Dokument: d})
	e2.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	if e2.s.Stil.Breite != 14 {
		t.Fatalf("Pfeil nach dem Wechsel: %v", e2.s.Stil.Breite)
	}
	// Wechsel auf eine 4K-Aufnahme (Faktor 2) und zurück: 28 px dort, wieder 14 hier, die Datei unverändert.
	d4, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 3840, 2160)))
	e2.uebernimm(Geoeffnet{Titel: "gross.png", Dokument: d4})
	e2.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	if e2.s.Stil.Breite != 28 {
		t.Fatalf("Pfeil auf 4K: %v, Soll 28", e2.s.Stil.Breite)
	}
	d1, _ := document.Neu(image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	e2.uebernimm(Geoeffnet{Titel: "klein.png", Dokument: d1})
	e2.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	if e2.s.Stil.Breite != 14 {
		t.Fatalf("Pfeil zurück auf Faktor 1: %v", e2.s.Stil.Breite)
	}
	vorher, _ = os.Stat(pfad)
	e2.werkzeugStileSchreiben()
	if nachher, err := os.Stat(pfad); err != nil || !os.SameFile(vorher, nachher) {
		t.Fatalf("Wechsel über Faktoren schreibt tools.ini neu (%v)", err)
	}
}
