// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// ADR-0030 Nachtrag 10 Punkt 5: Werkzeug annotate.

// TestMcpAnnotateEinwilligung: ohne edit abgelehnt (der Text nennt den Schalter); eine Bibliotheksdatei braucht
// zusätzlich library; relative Pfade und fehlende Liste sind Fehler.
func TestMcpAnnotateEinwilligung(t *testing.T) {
	l := neuerMcpLauf(t)
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	args := `{"file":"` + aussen + `","annotations":[{"type":"step","at":[5,5]}]}`
	if e := l.rufe(t, "annotate", args); !e.Fehler || !strings.Contains(text(e), "edit = yes") {
		t.Errorf("ohne edit: %+v", e)
	}
	l.agenten.Bearbeiten = true
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	l.agenten.Bibliothek = false
	if e := l.rufe(t, "annotate", `{"file":"`+drin+`","annotations":[{"type":"step","at":[5,5]}]}`); !e.Fehler ||
		!strings.Contains(text(e), "library = yes") {
		t.Errorf("Bibliotheksdatei ohne library: %+v", e)
	}
	for _, a := range []string{
		`{"file":"rel.png","annotations":[{"type":"step","at":[5,5]}]}`,
		`{"file":"` + aussen + `"}`,
		`{"file":"` + aussen + `","annotations":[],"x":1}`,
	} {
		if e := l.rufe(t, "annotate", a); !e.Fehler {
			t.Errorf("%s: kein Fehler", a)
		}
	}
}

// TestMcpAnnotateBibliothek: Eine Bibliotheksdatei wird fortgeschrieben und geschwärzt; die Antwort trägt Pfad, Größe
// und Zählung. Ein ungültiger Eintrag speichert nichts und nennt den Index.
func TestMcpAnnotateBibliothek(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	vorher, err := os.ReadFile(drin)
	if err != nil {
		t.Fatal(err)
	}
	if e := l.rufe(t, "annotate", `{"file":"`+drin+`","annotations":[{"type":"redact","from":[0,0],"to":[4,4]},`+
		`{"type":"step","at":[50,5]}]}`); !e.Fehler || !strings.Contains(text(e), "annotation 1") ||
		!strings.Contains(text(e), "Nothing was saved") {
		t.Fatalf("ungültiger Eintrag: %+v", e)
	}
	if nachher, _ := os.ReadFile(drin); string(nachher) != string(vorher) {
		t.Fatal("trotz Fehler geschrieben")
	}
	e := l.rufe(t, "annotate", `{"file":"`+drin+`","annotations":[{"type":"redact","from":[0,0],"to":[4,4]},`+
		`{"type":"arrow","from":[1,1],"to":[10,6],"color":"#2F6FDB"}]}`)
	if e.Fehler {
		t.Fatal(text(e))
	}
	var b beschriftet
	if err := json.Unmarshal([]byte(text(e)), &b); err != nil || b.Path != drin || b.Width != 12 || b.Height != 8 ||
		b.Annotations != 1 || b.Redactions != 1 {
		t.Errorf("Antwort %s (%v)", text(e), err)
	}
	g, err := fileformat.Oeffne(drin)
	if err != nil {
		t.Fatal(err)
	}
	if c := g.Dokument.OriginalBild().NRGBAAt(2, 2); c != canvas.Schwaerzfarbe {
		t.Errorf("nicht geschwärzt: %v", c)
	}
	if len(g.Dokument.Objekte()) != 1 {
		t.Errorf("Objekte %v", g.Dokument.Objekte())
	}
}

// TestMcpAnnotateFremdeDateiUnterAgenten (F-699): Eine Datei außerhalb der Bibliothek, die ein Agent beschriftet, wird
// eine Agenten-Aufnahme unter agents/, nicht eine gewöhnliche Aufnahme zwischen denen des Menschen; die Datei selbst
// bleibt unverändert. --annotate auf der Kommandozeile legt weiter gewöhnlich ab.
func TestMcpAnnotateFremdeDateiUnterAgenten(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bearbeiten = true
	aussen := filepath.Join(l.heim, "aussen.png")
	roh := mustPNG(t)
	if err := os.WriteFile(aussen, roh, 0o600); err != nil {
		t.Fatal(err)
	}
	e := l.rufe(t, "annotate", `{"file":"`+aussen+`","annotations":[{"type":"redact","from":[0,0],"to":[1,1]}]}`)
	if e.Fehler {
		t.Fatal(text(e))
	}
	var b beschriftet
	if err := json.Unmarshal([]byte(text(e)), &b); err != nil {
		t.Fatal(err)
	}
	ablage := l.orte.Ablage
	rel, err := filepath.Rel(ablage, b.Path)
	if err != nil || !strings.HasPrefix(filepath.ToSlash(rel), "agents/") {
		t.Errorf("Ergebnis %s liegt nicht unter agents/ der Ablage %s", b.Path, ablage)
	}
	if nachher, _ := os.ReadFile(aussen); string(nachher) != string(roh) {
		t.Error("fremde Datei verändert")
	}
}
