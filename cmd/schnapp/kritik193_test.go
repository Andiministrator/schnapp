// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Kritik zu Karte #193, K193-1 (ADR-0033 Nachtrag 2 Punkt 3): Eine Arbeitsdatei mit Lupen, deren Quellen zusammen das
// Rechenbudget überschreiten (je Lupe die ganze Quelle, ein Zielpixel), lehnen --export und das MCP-Werkzeug export
// vor dem Kodieren ab; kein Ziel entsteht.

// lupenUeberBudget legt eine solche Datei in dir an.
func lupenUeberBudget(t *testing.T, dir string) string {
	t.Helper()
	const w, h = 2000, 1000
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, w, h)))
	if err != nil {
		t.Fatal(err)
	}
	n := int(render.Budget/(25*w*h)) + 1 // 25: Gewicht je Pixel einer Füllung (render.fuellGewicht)
	for i := 0; i < n; i++ {
		if _, err := d.Hinzufuegen(document.Lupe{Quelle: document.R(document.Pt(0, 0), document.Pt(w, h)),
			Ziel: document.R(document.Pt(1, 1), document.Pt(2, 2)), Strich: document.Strich{Farbe: color.NRGBA{A: 255}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := render.Pruefe(d); !errors.Is(err, render.ErrZuAufwendig) {
		t.Fatalf("Vorbedingung: Pruefe %v, Soll ErrZuAufwendig", err)
	}
	return schreibeDatei(t, dir, "lupen.png", d)
}

func TestK193ExportLupenBudget(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	pfad := lupenUeberBudget(t, dir)
	ziel := filepath.Join(dir, "aus.svg")
	if code, _, _ := fuehre(t, keineMonitore, "--export="+ziel, pfad); code != exitFehler {
		t.Errorf("--export: Code %d, Soll %d", code, exitFehler)
	}
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz Ablehnung angelegt: %v", err)
	}
}

func TestK193McpExportLupenBudget(t *testing.T) {
	l := neuerMcpLauf(t)
	pfad := lupenUeberBudget(t, l.heim)
	ziel := filepath.Join(l.heim, "aus.svg")
	if e := l.rufe(t, "export", `{"file":"`+pfad+`","target":"`+ziel+`"}`); !e.Fehler {
		t.Errorf("MCP export über dem Budget angenommen: %s", text(e))
	}
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz Ablehnung angelegt: %v", err)
	}
}
