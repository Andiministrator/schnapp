// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Sitzungs-QA vom 24.11.2026 (Kritiker Fable), QS-6: Tests für zwei Zusagen, die ein Handmutant ohne Folge brechen konnte.

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// haltenBeimSchreiben merkt sich bei jedem Schreiben auf stderr, ob gerade ein Beschriften ein Signal zurückhält.
type haltenBeimSchreiben struct {
	mu     sync.Mutex
	text   strings.Builder
	halten []bool
}

func (h *haltenBeimSchreiben) Write(b []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.halten = append(h.halten, signalHalten())
	return h.text.Write(b)
}

// QS-M1 (F-603, ADR-0030 Nachtrag 12): Im MCP-Server endet die Lage eines annotate gleich nach beschrifteDatei, vor dem
// Hinweis auf stderr — sonst hielte ein blockierter Hinweis ein zweites Signal bis zur Frist zurück.
func TestMcpAnnotateLageEndetVorDemHinweis(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	h := new(haltenBeimSchreiben)
	l.an.fehler = h
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	// Eine Datei außerhalb der Bibliothek wird als neue Aufnahme gespeichert: Das Speichern meldet einen Hinweis.
	if e := l.rufe(t, "annotate", `{"file":"`+aussen+`","annotations":[{"type":"redact","from":[0,0],"to":[2,2]}]}`); e.Fehler {
		t.Fatal(text(e))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.halten) == 0 {
		t.Fatalf("kein Hinweis auf stderr (Aufbau); stderr %q", h.text.String())
	}
	for i, halt := range h.halten {
		if halt {
			t.Errorf("Schreiben %d auf stderr (%q): die Lage hält ein Signal noch zurück", i, h.text.String())
		}
	}
}

// QS-M2 (F-610, ADR-0033 Nachtrag 4): Jede andere Öffnung als oeffneImEditor setzt den Export-Schalter zurück, auch
// bearbeiteNeu nach einer Aufnahme.
func TestBearbeiteNeuSetztExportOhneGrenzeZurueck(t *testing.T) {
	rettungsLauf(t)
	t.Cleanup(func() { ui.SetzeExportOhneGrenze(false) })
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(orte)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	ui.SetzeExportOhneGrenze(true) // wie nach einem Editor mit --ignore-limit
	if err := bearbeiteNeu(abgelegt{pfad: abs, d: d, stand: standVon(t, abs)}); err != nil {
		t.Fatal(err)
	}
	if ui.ExportOhneGrenze() {
		t.Error("bearbeiteNeu ließ den Export-Schalter des vorigen Editors stehen")
	}
}
