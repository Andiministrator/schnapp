// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
	"github.com/Andiministrator/schnapp/internal/render"
)

// ueberZeichenBudgetEditor liefert einen Editor, dessen Dokument über dem Rechenbudget liegt (mehr Zeichen als
// render.ZeichenBudget), aber schnell zu zeichnen ist: kleine Schrift in einem Bild von 40 × 30.
func ueberZeichenBudgetEditor(t *testing.T, ohneGrenze bool) *editor {
	t.Helper()
	i18n.Setze(nil)
	fehlerAbfangen(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	zeile := strings.Repeat("x", 9000)
	for i := 0; i < render.ZeichenBudget/9000+1; i++ {
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(1, 1), Inhalt: zeile, Schriftgrad: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if render.Pruefe(d) == nil {
		t.Fatal("Aufbau: das Dokument liegt nicht über dem Budget")
	}
	return &editor{s: canvas.NeueSitzung(d), ohneGrenze: ohneGrenze}
}

// F-610 (Karte #234, ADR-0033 Nachtrag 4): Wurde die Datei mit --ignore-limit geöffnet, exportiert der Editor auch über
// dem Rechenbudget (wie --export --ignore-limit und wie Speichern) und sagt in der Leiste, dass es lange dauern kann;
// ohne den Schalter bleibt die Ablehnung vor dem Kodieren.
func TestEditorExportUeberBudgetNurMitOhneGrenze(t *testing.T) {
	e := ueberZeichenBudgetEditor(t, true)
	ziel := filepath.Join(t.TempDir(), "mit.png")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Errorf("mit --ignore-limit nicht geschrieben (%v); Leiste %q", err, e.meldung.Text)
	}
	if !strings.Contains(e.meldung.Text, "can take long") {
		t.Errorf("Leiste ohne Hinweis auf die Dauer: %q", e.meldung.Text)
	}

	e = ueberZeichenBudgetEditor(t, false)
	ziel = filepath.Join(t.TempDir(), "ohne.png")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ohne --ignore-limit trotz Budget geschrieben (%v)", err)
	}
	if !strings.Contains(e.meldung.Text, "too many or too large") {
		t.Errorf("ohne --ignore-limit: Leiste %q", e.meldung.Text)
	}
}

// SetzeExportOhneGrenze wirkt auf den nächsten Editor (neuerEditor liest sie).
func TestNeuerEditorUebernimmtExportOhneGrenze(t *testing.T) {
	i18n.Setze(nil)
	t.Cleanup(func() { SetzeExportOhneGrenze(false) })
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	for _, an := range []bool{true, false} {
		SetzeExportOhneGrenze(an)
		if e := neuerEditor("datei.png", d, nil, nil); e.ohneGrenze != an {
			t.Errorf("SetzeExportOhneGrenze(%v): Editor %v", an, e.ohneGrenze)
		}
	}
}

// QS-2 (ADR-0033 Nachtrag 5): Betteten die Lupen im SVG mehr Pixel ein, als fileformat.PruefeSVGExport zulässt, lehnt der
// Export aus dem Editor das SVG vor dem Kodieren mit eigener Meldung ab; PNG geht, mit --ignore-limit auch das SVG (mit
// dem Hinweis auf die Dauer).
func TestEditorSVGExportLupenGrenze(t *testing.T) {
	neu := func(ohneGrenze bool) *editor {
		i18n.Setze(nil)
		fehlerAbfangen(t)
		d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 60, 40)))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i <= fileformat.LupenExportFaktor; i++ {
			q := document.R(document.Pt(float64(i), 0), document.Pt(60, 40))
			if _, err := d.Hinzufuegen(document.Lupe{Quelle: q, Ziel: q}); err != nil {
				t.Fatal(err)
			}
		}
		if render.Pruefe(d) != nil || fileformat.PruefeSVGExport(d) == nil {
			t.Fatal("Aufbau: unter dem Budget und über der Grenze der Lupenpixel verlangt")
		}
		return &editor{s: canvas.NeueSitzung(d), ohneGrenze: ohneGrenze}
	}
	e := neu(false)
	ziel := filepath.Join(t.TempDir(), "ohne.svg")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("SVG über der Grenze geschrieben (%v)", err)
	}
	if !strings.Contains(e.meldung.Text, "magnifiers") {
		t.Errorf("Leiste ohne Grund: %q", e.meldung.Text)
	}
	ziel = filepath.Join(t.TempDir(), "ohne.png")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Errorf("PNG nicht geschrieben (%v); Leiste %q", err, e.meldung.Text)
	}
	e = neu(true)
	ziel = filepath.Join(t.TempDir(), "mit.svg")
	e.exportFertig(exportErgebnis{ergebnis: portal.Ergebnis{URI: "file://" + ziel}})
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Errorf("mit --ignore-limit kein SVG (%v); Leiste %q", err, e.meldung.Text)
	}
	if !strings.Contains(e.meldung.Text, "can take long") {
		t.Errorf("mit --ignore-limit ohne Hinweis auf die Dauer: %q", e.meldung.Text)
	}
}
