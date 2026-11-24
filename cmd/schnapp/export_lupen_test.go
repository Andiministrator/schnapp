// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/render"
)

// lupenUeberGrenze schreibt eine Arbeitsdatei unter dem Rechenbudget, deren Lupen (verschiedene, fast bildgroße
// Quellen) im SVG-Export mehr Pixel einbetteten, als fileformat.PruefeSVGExport zulässt (ADR-0033 Nachtrag 5, QS-2).
func lupenUeberGrenze(t *testing.T, dir string) string {
	t.Helper()
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
	return schreibeDatei(t, dir, "lupen.png", d)
}

// QS-2: --export lehnt ein SVG über der Grenze der Lupenpixel vor dem Kodieren ab (Exit 1, Meldung nennt die Lupen und
// --ignore-limit, kein Ziel); PNG geht, mit --ignore-limit auch das SVG.
func TestExportLupenGrenze(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	pfad := lupenUeberGrenze(t, dir)
	svg := filepath.Join(dir, "aus.svg")
	code, aus, fehler := fuehre(t, keineMonitore, "--export="+svg, pfad)
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "magnifiers") || !strings.Contains(fehler, "--ignore-limit") {
		t.Errorf("SVG über der Grenze: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if _, err := os.Lstat(svg); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz Ablehnung angelegt: %v", err)
	}
	png := filepath.Join(dir, "aus.png")
	if code, aus, fehler := fuehre(t, keineMonitore, "--export="+png, pfad); code != exitOK || aus != png+"\n" {
		t.Errorf("PNG: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	code, aus, fehler = fuehre(t, keineMonitore, "--ignore-limit", "--export="+svg, pfad)
	if code != exitOK || aus != svg+"\n" {
		t.Errorf("SVG mit --ignore-limit: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if b, err := os.ReadFile(svg); err != nil || strings.Count(string(b), "image-rendering:pixelated") != fileformat.LupenExportFaktor+1 {
		t.Errorf("SVG mit --ignore-limit: %v", err)
	}
}

// QS-2: MCP export lehnt dasselbe ab (ein Agent hat kein --ignore-limit); die Antwort nennt den Grund, kein Ziel entsteht.
func TestMcpExportLupenGrenze(t *testing.T) {
	l := neuerMcpLauf(t)
	pfad := lupenUeberGrenze(t, l.heim)
	ziel := filepath.Join(l.heim, "aus.svg")
	e := l.rufe(t, "export", `{"file":"`+pfad+`","target":"`+ziel+`"}`)
	if !e.Fehler || !strings.Contains(text(e), "magnifiers") {
		t.Errorf("MCP export über der Grenze: %+v", e)
	}
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz Ablehnung angelegt: %v", err)
	}
	png := filepath.Join(l.heim, "aus.png")
	if e := l.rufe(t, "export", `{"file":"`+pfad+`","target":"`+png+`"}`); e.Fehler {
		t.Errorf("MCP export als PNG: %s", text(e))
	}
}
