// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/annotate"
	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/mcp"
	"github.com/Andiministrator/schnapp/internal/render"
	"github.com/Andiministrator/schnapp/internal/selection"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// mcpLauf hält die Attrappen eines Server-Laufs und protokolliert die Schritte in Reihenfolge.
type mcpLauf struct {
	an          mcpAnschluesse
	q           *quelleAttrappe
	agenten     settings.Agenten
	schritte    []string
	antwort     bool   // Antwort des Dialogs
	dialog      error  // Fehler des Dialogs
	hinweis     error  // Fehler der Benachrichtigung
	hinweisNr   uint32 // Kennung des letzten Hinweises der Attrappe
	warteFehler error  // Ergebnis des Wartens nach dem Schließen des vorigen Hinweises (Abbruch)
	orte        library.Orte
	heim        string
	fehler      bytes.Buffer
}

func neuerMcpLauf(t *testing.T) *mcpLauf {
	t.Helper()
	i18n.Setze(nil)
	heim := t.TempDir()
	orte, err := library.OrteAus(func(k string) string { return map[string]string{"HOME": heim}[k] })
	if err != nil {
		t.Fatal(err)
	}
	l := &mcpLauf{
		q: &quelleAttrappe{
			bilder: map[string]*image.NRGBA{"DP-1": testbild(40, 30, 1), "HDMI-A-1": testbild(20, 10, 2)},
			aktiv:  testbild(40, 30, 3), alle: testbild(60, 30, 4),
		},
		agenten: settings.Agenten{Aufnehmen: true, Bibliothek: true, Bestaetigen: true, Bild: true},
		antwort: true, orte: orte, heim: heim,
	}
	l.an = mcpAnschluesse{
		lade: func() (settings.Einstellungen, []string) {
			e := settings.Voreinstellungen()
			e.Agenten = l.agenten
			return e, nil
		},
		quelle: func() (capture.Quelle, error) { l.schritte = append(l.schritte, "aufnahme"); return l.q, nil },
		monitore: func() ([]ui.Monitor, error) {
			return []ui.Monitor{{Name: "DP-1", Breite: 40, Hoehe: 30, Skala: 1}, {Name: "HDMI-A-1", Breite: 20, Hoehe: 10, Skala: 1}}, nil
		},
		bestaetige: func(_ context.Context, ziel, kunde string) (bool, error) {
			l.schritte = append(l.schritte, "dialog:"+ziel+"|"+kunde)
			return l.antwort, l.dialog
		},
		melde: func(_ context.Context, titel, text string) (uint32, error) {
			l.schritte = append(l.schritte, "hinweis:"+text)
			l.hinweisNr++
			return l.hinweisNr, l.hinweis
		},
		schliesse: func(_ context.Context, id uint32) error {
			l.schritte = append(l.schritte, fmt.Sprintf("schliesse:%d", id))
			return nil
		},
		hinweis:     new(letzterHinweis),
		orte:        func() (library.Orte, error) { return orte, nil },
		letzterPfad: filepath.Join(heim, "letzter.json"),
		warte: func(_ context.Context, d time.Duration) error {
			if d == hinweisWeg { // nach dem Schließen des vorigen Hinweises (F-490); sonst nach dem Dialog
				l.schritte = append(l.schritte, "warte-hinweis")
				return l.warteFehler
			}
			l.schritte = append(l.schritte, "warte")
			return nil
		},
		einstellungsOrt: "~/.config/schnapp/settings.ini",
		fehler:          &l.fehler,
	}
	return l
}

func (l *mcpLauf) rufe(t *testing.T, werkzeug, args string) mcp.Ergebnis {
	t.Helper()
	for _, w := range l.an.werkzeuge() {
		if w.Name == werkzeug {
			return w.Aufruf(context.Background(), json.RawMessage(args), mcp.Kunde{Name: "Testagent"})
		}
	}
	t.Fatalf("Werkzeug %s fehlt", werkzeug)
	return mcp.Ergebnis{}
}

func text(e mcp.Ergebnis) string {
	var s []string
	for _, i := range e.Inhalt {
		if i.Typ == "text" {
			s = append(s, i.Text)
		}
	}
	return strings.Join(s, "\n")
}

func bild(t *testing.T, e mcp.Ergebnis) image.Image {
	t.Helper()
	for _, i := range e.Inhalt {
		if i.Typ == "image" {
			if i.MIME != "image/png" {
				t.Errorf("MIME %q", i.MIME)
			}
			daten, err := base64.StdEncoding.DecodeString(i.Daten)
			if err != nil {
				t.Fatal(err)
			}
			b, err := png.Decode(bytes.NewReader(daten))
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
	return nil
}

// pngsIn zählt PNG-Dateien unter ordner (fehlt er, 0).
func pngsIn(t *testing.T, ordner string) []string {
	t.Helper()
	var n []string
	_ = filepath.WalkDir(ordner, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".png") {
			n = append(n, p)
		}
		return nil
	})
	return n
}

// TestMcpWerkzeugliste: Namen und Reihenfolge sind Oberfläche (ADR-0030 Nachtrag 2 Punkt 3); jedes Schema ist ein
// Objekt.
func TestMcpWerkzeugliste(t *testing.T) {
	l := neuerMcpLauf(t)
	var namen []string
	for _, w := range l.an.werkzeuge() {
		namen = append(namen, w.Name)
		var s map[string]any
		if err := json.Unmarshal(w.Eingabe, &s); err != nil || s["type"] != "object" {
			t.Errorf("%s: Schema %s", w.Name, w.Eingabe)
		}
	}
	if got := strings.Join(namen, ","); got != "list_screens,capture_screen,capture_repeat,library_search,library_read,export,annotate" {
		t.Errorf("Werkzeuge: %s", got)
	}
}

// TestMcpOhneEinwilligung: Ohne capture = yes kein Dialog, keine Aufnahme, kein Hinweis; der Text nennt Schalter und
// Datei.
func TestMcpOhneEinwilligung(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Aufnehmen = false
	for _, w := range []string{"capture_screen", "capture_repeat"} {
		e := l.rufe(t, w, `{}`)
		if !e.Fehler || !strings.Contains(text(e), "capture = yes") || !strings.Contains(text(e), "settings.ini") {
			t.Errorf("%s: %+v", w, e)
		}
	}
	if len(l.schritte) != 0 {
		t.Errorf("Schritte ohne Einwilligung: %v", l.schritte)
	}
}

// TestMcpReihenfolge: Dialog, Wartezeit, Aufnahme, Hinweis, dann erst die Ablage unter agents/.
func TestMcpReihenfolge(t *testing.T) {
	l := neuerMcpLauf(t)
	e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`)
	if e.Fehler {
		t.Fatalf("Fehler: %s", text(e))
	}
	want := []string{"dialog:monitor DP-1|Testagent", "warte", "aufnahme", "hinweis:monitor DP-1"}
	if strings.Join(l.schritte, ";") != strings.Join(want, ";") {
		t.Errorf("Schritte %v, erwartet %v", l.schritte, want)
	}
	s, ok := e.Strukturiert.(ergebnis)
	if !ok || s.Width != 40 || s.Height != 30 {
		t.Fatalf("structuredContent: %#v", e.Strukturiert)
	}
	if !strings.HasPrefix(s.Path, filepath.Join(l.orte.Ablage, "agents")+string(filepath.Separator)) {
		t.Errorf("Pfad %s nicht unter agents/", s.Path)
	}
	if fi, err := os.Stat(s.Path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Datei: %v, %v", fi, err)
	}
	if b := bild(t, e); b == nil || b.Bounds().Size() != (image.Point{40, 30}) {
		t.Errorf("Bildinhalt fehlt oder falsche Größe")
	}
}

func TestMcpAblehnungUndDialogfehler(t *testing.T) {
	for name, setze := range map[string]func(*mcpLauf){
		"abgelehnt":    func(l *mcpLauf) { l.antwort = false },
		"Dialogfehler": func(l *mcpLauf) { l.dialog = errors.New("kein Wayland") },
	} {
		l := neuerMcpLauf(t)
		setze(l)
		e := l.rufe(t, "capture_screen", `{}`)
		if !e.Fehler {
			t.Errorf("%s: kein Fehler", name)
		}
		if len(l.schritte) != 1 || !strings.HasPrefix(l.schritte[0], "dialog:") {
			t.Errorf("%s: nach dem Dialog ging es weiter: %v", name, l.schritte)
		}
	}
}

// TestMcpHinweisScheitert: Ohne Hinweis kein Bild — weder in der Bibliothek noch am Ziel.
func TestMcpHinweisScheitert(t *testing.T) {
	l := neuerMcpLauf(t)
	l.hinweis = errors.New("kein Dienst")
	ziel := filepath.Join(l.heim, "aus.png")
	for _, args := range []string{`{}`, `{"output":"` + ziel + `"}`} {
		e := l.rufe(t, "capture_screen", args)
		if !e.Fehler || !strings.Contains(text(e), "discarded") {
			t.Errorf("%s: %+v", args, e)
		}
	}
	if n := pngsIn(t, l.orte.Ablage); len(n) != 0 {
		t.Errorf("Bild trotz gescheitertem Hinweis abgelegt: %v", n)
	}
	if _, err := os.Stat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz gescheitertem Hinweis geschrieben: %v", err)
	}
}

// TestMcpOhneDialog: confirm = no überspringt den Dialog, der Hinweis bleibt.
func TestMcpOhneDialog(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	if e := l.rufe(t, "capture_screen", `{"all":true}`); e.Fehler {
		t.Fatal(text(e))
	}
	if strings.Join(l.schritte, ";") != "aufnahme;hinweis:all monitors" {
		t.Errorf("Schritte: %v", l.schritte)
	}
}

// TestMcpSchliesstVorigenHinweis (F-490, Echtlauf 23.11.2026): Der Hinweis der vorigen Aufnahme stand im nächsten Bild.
// Vor jeder weiteren Aufnahme schließt der Server ihn und wartet, mit Dialog erst nach dem Dialog. Scheitert das
// Schließen, wird trotzdem aufgenommen (verborgen wird nichts).
func TestMcpSchliesstVorigenHinweis(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	for i := 0; i < 2; i++ {
		if e := l.rufe(t, "capture_screen", `{"all":true}`); e.Fehler {
			t.Fatal(text(e))
		}
	}
	want := "aufnahme;hinweis:all monitors;schliesse:1;warte-hinweis;aufnahme;hinweis:all monitors"
	if got := strings.Join(l.schritte, ";"); got != want {
		t.Errorf("ohne Dialog: %s, erwartet %s", got, want)
	}
	if *l.an.hinweis != (letzterHinweis{id: 2, text: "all monitors"}) {
		t.Errorf("gemerkter Hinweis %+v", *l.an.hinweis)
	}

	l.schritte = nil
	l.agenten.Bestaetigen = true
	if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); e.Fehler {
		t.Fatal(text(e))
	}
	want = "dialog:monitor DP-1|Testagent;warte;schliesse:2;warte-hinweis;aufnahme;hinweis:monitor DP-1"
	if got := strings.Join(l.schritte, ";"); got != want {
		t.Errorf("mit Dialog: %s, erwartet %s", got, want)
	}

	t.Setenv("SCHNAPP_DEBUG", "1") // diagnose schreibt nur damit
	l.schritte = nil
	l.an.schliesse = func(context.Context, uint32) error {
		l.schritte = append(l.schritte, "schliesse-scheitert")
		return errors.New("kein Dienst")
	}
	if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); e.Fehler || !strings.Contains(strings.Join(l.schritte, ";"), "aufnahme") {
		t.Errorf("gescheitertes Schließen verhindert die Aufnahme: %v %s", l.schritte, text(e))
	}
	if !strings.Contains(l.fehler.String(), "nicht geschlossen") {
		t.Errorf("gescheitertes Schließen ohne Diagnose: %q", l.fehler.String())
	}
}

// TestMcpVorigerHinweisKommtWieder (Sitzungs-QA B-1): Endet ein Aufruf nach dem Schließen ohne eigenen Hinweis, zeigt
// der Server den vorigen wieder und merkt sich dessen neue Kennung; eine Ablehnung im Dialog schließt ihn gar nicht.
// Sonst könnte ein Agent den Hinweis seiner Aufnahme mit einem Fehlversuch verschwinden lassen.
func TestMcpVorigerHinweisKommtWieder(t *testing.T) {
	faelle := map[string]struct {
		vorher func(l *mcpLauf)
		want   string
	}{
		"Abbruch beim Warten": {
			vorher: func(l *mcpLauf) { l.warteFehler = context.Canceled },
			want:   "schliesse:1;warte-hinweis;hinweis:monitor DP-1",
		},
		"Aufnahme scheitert": {
			vorher: func(l *mcpLauf) {
				l.an.quelle = func() (capture.Quelle, error) { return nil, errors.New("kein KWin") }
			},
			want: "schliesse:1;warte-hinweis;hinweis:monitor DP-1",
		},
		"Dialog abgelehnt": {
			vorher: func(l *mcpLauf) { l.agenten.Bestaetigen, l.antwort = true, false },
			want:   "dialog:all monitors|Testagent",
		},
	}
	for name, f := range faelle {
		l := neuerMcpLauf(t)
		l.agenten.Bestaetigen = false
		if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); e.Fehler {
			t.Fatal(text(e))
		}
		l.schritte = nil
		f.vorher(l)
		if e := l.rufe(t, "capture_screen", `{"all":true}`); !e.Fehler {
			t.Errorf("%s: kein Fehler", name)
		}
		if got := strings.Join(l.schritte, ";"); got != f.want {
			t.Errorf("%s: %s, erwartet %s", name, got, f.want)
		}
		if h := *l.an.hinweis; h.id == 0 || h.text != "monitor DP-1" {
			t.Errorf("%s: kein offener Hinweis gemerkt: %+v", name, h)
		}
	}
}

// TestHinweisWegReicht: Die Wartezeit nach dem Schließen muss Plasma das Ausblenden lassen (ADR-0030 Nachtrag 7:
// 600 ms); ein kürzerer Wert brächte den alten Hinweis zurück ins Bild.
func TestHinweisWegReicht(t *testing.T) {
	if hinweisWeg < 500*time.Millisecond {
		t.Errorf("hinweisWeg %v, mindestens 500 ms", hinweisWeg)
	}
}

func TestMcpOhneBild(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bild = false
	e := l.rufe(t, "capture_screen", `{}`)
	if e.Fehler || bild(t, e) != nil {
		t.Errorf("mit image = no kam ein Bild: %+v", e)
	}
	if !strings.Contains(text(e), `"path"`) {
		t.Errorf("Pfad fehlt: %s", text(e))
	}
}

func TestMcpZielwahl(t *testing.T) {
	for args, want := range map[string]string{
		`{}`:                    "Aktiv",
		`{"all":true}`:          "Alle",
		`{"screen":"HDMI-A-1"}`: "Bildschirm HDMI-A-1",
		`{"all":false}`:         "Aktiv",
	} {
		l := neuerMcpLauf(t)
		l.agenten.Bestaetigen = false
		if e := l.rufe(t, "capture_screen", args); e.Fehler {
			t.Errorf("%s: %s", args, text(e))
		}
		if len(l.q.aufrufe) != 1 || l.q.aufrufe[0] != want {
			t.Errorf("%s: Aufrufe %v, erwartet %s", args, l.q.aufrufe, want)
		}
	}
}

// TestMcpOhneZielFolgtEinstellung: Ohne screen und all nimmt [capture] screen = all alle Monitore.
func TestMcpOhneZielFolgtEinstellung(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	lade := l.an.lade
	l.an.lade = func() (settings.Einstellungen, []string) {
		e, w := lade()
		e.Bildschirm = settings.WahlAlle
		return e, w
	}
	if e := l.rufe(t, "capture_screen", `{}`); e.Fehler {
		t.Fatal(text(e))
	}
	if len(l.q.aufrufe) != 1 || l.q.aufrufe[0] != "Alle" {
		t.Errorf("Aufrufe %v, erwartet Alle", l.q.aufrufe)
	}
	if !strings.Contains(strings.Join(l.schritte, ";"), "hinweis:all monitors") {
		t.Errorf("Hinweis nennt nicht alle Monitore: %v", l.schritte)
	}
}

func TestMcpFalscheArgumente(t *testing.T) {
	t.Chdir(t.TempDir())
	l := neuerMcpLauf(t)
	for _, args := range []string{
		`{"screen":"DP-1","all":true}`, `{"output":"rel.png"}`, `{"output":"/x/a.txt"}`, `{"unbekannt":1}`, `{"all":"ja"}`,
	} {
		if e := l.rufe(t, "capture_screen", args); !e.Fehler {
			t.Errorf("%s: kein Fehler", args)
		}
	}
	if len(l.q.aufrufe) != 0 || len(l.schritte) != 0 {
		t.Errorf("trotz falscher Argumente: %v %v", l.q.aufrufe, l.schritte)
	}
}

// TestMcpAusgabe: output schreibt 0600 dorthin, nicht in die Bibliothek.
func TestMcpAusgabe(t *testing.T) {
	l := neuerMcpLauf(t)
	ziel := filepath.Join(l.heim, "aus.png")
	e := l.rufe(t, "capture_screen", `{"output":"`+ziel+`"}`)
	if e.Fehler {
		t.Fatal(text(e))
	}
	if fi, err := os.Stat(ziel); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("Ziel: %v, %v", fi, err)
	}
	if n := pngsIn(t, l.orte.Ablage); len(n) != 0 {
		t.Errorf("mit output trotzdem in der Bibliothek: %v", n)
	}
}

func TestMcpLetzterBereich(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	if e := l.rufe(t, "capture_repeat", `{}`); !e.Fehler || len(l.q.aufrufe) != 0 {
		t.Errorf("ohne letzten Bereich: %+v %v", e, l.q.aufrufe)
	}
	if err := selection.SpeichereLetzten(l.an.letzterPfad, selection.Letzter{Monitor: "DP-1", Groesse: image.Pt(40, 30),
		Skala: 1, Bereich: image.Rect(5, 6, 25, 16)}); err != nil {
		t.Fatal(err)
	}
	e := l.rufe(t, "capture_repeat", `{}`)
	if e.Fehler {
		t.Fatal(text(e))
	}
	if s := e.Strukturiert.(ergebnis); s.Width != 20 || s.Height != 10 {
		t.Errorf("Größe %dx%d, erwartet 20x10", s.Width, s.Height)
	}
	if b := bild(t, e); b == nil {
		t.Error("Bild fehlt")
	} else if r, g, _, _ := b.At(0, 0).RGBA(); r>>8 != 5 || g>>8 != 6 {
		t.Errorf("Ausschnitt beginnt nicht bei 5,6: %d,%d", r>>8, g>>8)
	}
	// Monitor geändert: abgelehnt.
	l.q.bilder["DP-1"] = testbild(50, 30, 1)
	if e := l.rufe(t, "capture_repeat", `{}`); !e.Fehler {
		t.Error("geänderter Monitor nicht erkannt")
	}
}

// legeAb legt eine Aufnahme in die Bibliothek (agent: im Unterordner agents).
func (l *mcpLauf) legeAb(t *testing.T, agent bool, wann string, worte ...string) string {
	t.Helper()
	d, err := document.Neu(testbild(12, 8, 9))
	if err != nil {
		t.Fatal(err)
	}
	sichtbar, _ := render.Bild(d)
	b := library.Neu(l.orte)
	m := fileformat.Metadaten{Erstellt: zeitAus(t, wann), Schlagworte: worte}
	var e library.Eintrag
	if agent {
		e, err = b.AblegenAgent(d, sichtbar, m)
	} else {
		e, err = b.Ablegen(d, sichtbar, m)
	}
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(l.orte.Ablage, filepath.FromSlash(e.Pfad))
}

func zeitAus(t *testing.T, s string) time.Time {
	t.Helper()
	z, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return z
}

func TestMcpBibliothekSuchen(t *testing.T) {
	l := neuerMcpLauf(t)
	l.legeAb(t, false, "2026-11-01T10:00:00Z", "rechnung")
	l.legeAb(t, true, "2026-11-02T10:00:00Z")
	l.legeAb(t, false, "2026-11-03T10:00:00Z")

	l.agenten.Bibliothek = false
	if e := l.rufe(t, "library_search", `{}`); !e.Fehler || !strings.Contains(text(e), "library = yes") {
		t.Errorf("ohne Einwilligung: %+v", e)
	}
	l.agenten.Bibliothek = true
	type antwort struct {
		Entries []struct {
			Path  string   `json:"path"`
			Tags  []string `json:"tags"`
			Agent bool     `json:"agent"`
		} `json:"entries"`
		More bool `json:"more"`
	}
	lies := func(args string) antwort {
		t.Helper()
		e := l.rufe(t, "library_search", args)
		if e.Fehler {
			t.Fatalf("%s: %s", args, text(e))
		}
		var a antwort
		if err := json.Unmarshal([]byte(text(e)), &a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	a := lies(`{}`)
	if len(a.Entries) != 3 || !a.Entries[1].Agent || a.Entries[0].Agent || !filepath.IsAbs(a.Entries[0].Path) {
		t.Errorf("alle: %+v", a)
	}
	if a := lies(`{"agents":false}`); len(a.Entries) != 2 {
		t.Errorf("ohne Agenten: %+v", a)
	}
	if a := lies(`{"tag":"Rechnung"}`); len(a.Entries) != 1 || a.Entries[0].Tags[0] != "rechnung" {
		t.Errorf("Schlagwort: %+v", a)
	}
	// Agenten bekommen das ganze Schlagwort, kein Teilwort wie das Suchfeld der Ansicht (ADR-0036 Zusatz F-671, F-687).
	if a := lies(`{"tag":"rech"}`); len(a.Entries) != 0 {
		t.Errorf("Teilwort trifft über MCP: %+v", a)
	}
	if a := lies(`{"limit":1}`); len(a.Entries) != 1 || !a.More {
		t.Errorf("limit: %+v", a)
	}
	if a := lies(`{"from":"2026-11-02T00:00:00Z","to":"2026-11-03T00:00:00Z"}`); len(a.Entries) != 1 || !a.Entries[0].Agent {
		t.Errorf("Zeitraum: %+v", a)
	}
	for _, args := range []string{`{"limit":0}`, `{"limit":51}`, `{"from":"gestern"}`} {
		if e := l.rufe(t, "library_search", args); !e.Fehler {
			t.Errorf("%s: kein Fehler", args)
		}
	}
}

func TestMcpBibliothekLesen(t *testing.T) {
	l := neuerMcpLauf(t)
	pfad := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	l.agenten.Bibliothek = false
	if e := l.rufe(t, "library_read", `{"path":"`+pfad+`"}`); !e.Fehler {
		t.Error("ohne Einwilligung gelesen")
	}
	l.agenten.Bibliothek = true
	e := l.rufe(t, "library_read", `{"path":"`+pfad+`"}`)
	if e.Fehler || bild(t, e) == nil {
		t.Fatalf("absolut: %+v", e)
	}
	rel, _ := filepath.Rel(l.orte.Ablage, pfad)
	if e := l.rufe(t, "library_read", `{"path":"`+rel+`"}`); e.Fehler {
		t.Errorf("relativ: %s", text(e))
	}
	// Außerhalb der Ablage, auch über einen Link aus der Ablage hinaus oder mit „..“: abgelehnt.
	fremd := filepath.Join(l.heim, "fremd.png")
	if err := os.WriteFile(fremd, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(l.orte.Ablage, "link.png")
	if err := os.Symlink(fremd, link); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fremd, link, "../fremd.png", filepath.Join(l.orte.Ablage, "..", "..", "fremd.png"), l.orte.Ablage} {
		if e := l.rufe(t, "library_read", `{"path":"`+p+`"}`); !e.Fehler {
			t.Errorf("%s: gelesen", p)
		}
	}
}

func mustPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, testbild(4, 4, 1)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestMcpExport(t *testing.T) {
	t.Chdir(t.TempDir()) // bei gebrochener Prüfung entsteht rel.png dort, nicht im Paketordner (F-471)
	l := neuerMcpLauf(t)
	aussen := filepath.Join(l.heim, "aussen.png")
	if err := os.WriteFile(aussen, mustPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	drin := l.legeAb(t, false, "2026-11-01T10:00:00Z")
	l.agenten.Bibliothek = false
	ziel := filepath.Join(l.heim, "ziel.jpg")
	if e := l.rufe(t, "export", `{"file":"`+aussen+`","target":"`+ziel+`"}`); e.Fehler {
		t.Fatalf("Datei außerhalb der Bibliothek: %s", text(e))
	}
	if _, err := os.Stat(ziel); err != nil {
		t.Error(err)
	}
	if e := l.rufe(t, "export", `{"file":"`+drin+`","target":"`+ziel+`"}`); !e.Fehler || !strings.Contains(text(e), "library = yes") {
		t.Errorf("Bibliotheksdatei ohne Einwilligung: %+v", e)
	}
	for _, args := range []string{
		`{"file":"rel.png","target":"` + ziel + `"}`, `{"file":"` + aussen + `","target":"rel.png"}`,
		`{"file":"` + aussen + `","target":"` + aussen + `"}`,
	} {
		if e := l.rufe(t, "export", args); !e.Fehler {
			t.Errorf("%s: kein Fehler", args)
		}
	}
}

// TestMcpWiderrufWirktSofort: Die Einstellungen werden bei jedem Aufruf frisch geladen.
func TestMcpWiderrufWirktSofort(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	if e := l.rufe(t, "capture_screen", `{}`); e.Fehler {
		t.Fatal(text(e))
	}
	l.agenten.Aufnehmen = false
	if e := l.rufe(t, "capture_screen", `{}`); !e.Fehler {
		t.Error("Widerruf wirkte nicht")
	}
}

func TestKundenname(t *testing.T) {
	for ein, want := range map[string]string{
		"Claude Code":           "Claude Code",
		"  a\x1b[31mb\n ":       "a�[31mb",
		strings.Repeat("x", 70): strings.Repeat("x", 60) + "…",
		"\xff":                  "�",
	} {
		if got := kundenname(mcp.Kunde{Name: ein}); got != want {
			t.Errorf("%q: %q, erwartet %q", ein, got, want)
		}
	}
}

// TestLaufeMcpEndeZuEnde: laufeMcp spricht das Protokoll über die echten Ströme (ohne Werkzeugaufruf an die Außenwelt).
func TestLaufeMcpEndeZuEnde(t *testing.T) {
	l := neuerMcpLauf(t)
	ein := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_screens","_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}` + "\n")
	var aus, fehler bytes.Buffer
	if probeGuard() != nil {
		// Probe-Build außerhalb des virtuellen KWin: Der Server darf gar nicht erst antworten (F-229).
		if code := laufeMcp(ein, &aus, &fehler, l.an); code != exitFehler || aus.Len() != 0 {
			t.Errorf("Probe-Build: Exit %d, stdout %q", code, aus.String())
		}
		return
	}
	if code := laufeMcp(ein, &aus, &fehler, l.an); code != exitOK {
		t.Fatalf("Exit %d: %s", code, fehler.String())
	}
	if !strings.Contains(aus.String(), `\"name\":\"DP-1\"`) || strings.Count(aus.String(), "\n") != 1 {
		t.Errorf("stdout: %s", aus.String())
	}
}

// TestBestaetigungKindLiestStdin: Ohne gültige Angaben zeigt das Kind keinen Dialog und endet mit einem Aufruffehler.
func TestBestaetigungKindLiestStdin(t *testing.T) {
	var fehler bytes.Buffer
	want := exitAufruf
	if probeGuard() != nil {
		want = exitFehler // Probe-Build außerhalb des virtuellen KWin: kein Dialog, egal was kommt (F-229)
	}
	for _, ein := range []string{"", "kein json", `{"target":""}`} {
		if code := laufeBestaetigung(strings.NewReader(ein), &fehler); code != want {
			t.Errorf("%q: Exit %d", ein, code)
		}
	}
}

// TestMcpVorigerHinweisWeitereWege (Sitzungs-QA N-2, N-3): Scheitert der neue Hinweis oder wird nach der Aufnahme
// abgebrochen, kommt der vorige wieder. Scheitert schon das Schließen, steht der alte noch: Er wird nicht doppelt
// gezeigt und bleibt zum Schließen vorgemerkt.
func TestMcpVorigerHinweisWeitereWege(t *testing.T) {
	vorbereiten := func() *mcpLauf {
		l := neuerMcpLauf(t)
		l.agenten.Bestaetigen = false
		if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); e.Fehler {
			t.Fatal(text(e))
		}
		l.schritte = nil
		return l
	}
	pruefe := func(name string, l *mcpLauf, want string, id uint32) {
		t.Helper()
		if got := strings.Join(l.schritte, ";"); got != want {
			t.Errorf("%s: %s, erwartet %s", name, got, want)
		}
		if h := *l.an.hinweis; h.text != "monitor DP-1" || (id != 0 && h.id != id) || h.id == 0 {
			t.Errorf("%s: gemerkter Hinweis %+v", name, h)
		}
	}

	l := vorbereiten()
	melde := l.an.melde
	l.an.melde = func(c context.Context, ti, te string) (uint32, error) {
		id, _ := melde(c, ti, te)
		if te == "all monitors" {
			return 0, errors.New("Dienst weg")
		}
		return id, nil
	}
	l.rufe(t, "capture_screen", `{"all":true}`)
	pruefe("neuer Hinweis scheitert", l, "schliesse:1;warte-hinweis;aufnahme;hinweis:all monitors;hinweis:monitor DP-1", 0)

	l = vorbereiten()
	ctx, abbrechen := context.WithCancel(context.Background())
	quelle := l.an.quelle
	l.an.quelle = func() (capture.Quelle, error) { abbrechen(); return quelle() }
	if e := l.rufeMit(t, ctx, "capture_screen", `{"all":true}`, mcp.Kunde{}); !e.Fehler {
		t.Error("Abbruch nach der Aufnahme: kein Fehler")
	}
	pruefe("Abbruch nach der Aufnahme", l, "schliesse:1;warte-hinweis;aufnahme;hinweis:monitor DP-1", 0)

	l = vorbereiten()
	l.an.schliesse = func(context.Context, uint32) error {
		l.schritte = append(l.schritte, "schliesse-scheitert")
		return errors.New("kein Dienst")
	}
	l.an.quelle = func() (capture.Quelle, error) { return nil, errors.New("kein KWin") }
	l.rufe(t, "capture_screen", `{"all":true}`)
	pruefe("Schließen und Aufnahme scheitern", l, "schliesse-scheitert", 1)
}

// TestMcpAbbruchBeimSchliessen (Verify 2, Fund 2): Wird mitten im Schließen abgebrochen, ist unbekannt, ob der Dienst
// den Hinweis schon geschlossen hat; er gilt als geschlossen und kommt wieder, statt ersatzlos zu verschwinden.
func TestMcpAbbruchBeimSchliessen(t *testing.T) {
	l := neuerMcpLauf(t)
	l.agenten.Bestaetigen = false
	if e := l.rufe(t, "capture_screen", `{"screen":"DP-1"}`); e.Fehler {
		t.Fatal(text(e))
	}
	l.schritte = nil
	ctx, abbrechen := context.WithCancel(context.Background())
	l.an.schliesse = func(context.Context, uint32) error {
		l.schritte = append(l.schritte, "schliesse-abgebrochen")
		abbrechen()
		return context.Canceled
	}
	if e := l.rufeMit(t, ctx, "capture_screen", `{"all":true}`, mcp.Kunde{}); !e.Fehler {
		t.Error("kein Fehler nach dem Abbruch")
	}
	if got, want := strings.Join(l.schritte, ";"), "schliesse-abgebrochen;warte-hinweis;aufnahme;hinweis:monitor DP-1"; got != want {
		// Die Warte-Attrappe beachtet den Abbruch nicht; die Aufnahme läuft an und wird verworfen.
		t.Errorf("%s, erwartet %s", got, want)
	}
	if h := *l.an.hinweis; h.id == 0 || h.text != "monitor DP-1" {
		t.Errorf("kein Hinweis gemerkt: %+v", h)
	}
}

// F-681: list_screens nimmt die Monitore mit Skala, die Namensprüfung von capture_screen die ohne (keine zweite
// Wayland-Verbindung vor dem Dialog).
func TestMcpMonitoreMitSkalaNurFuerListe(t *testing.T) {
	l := neuerMcpLauf(t)
	var liste, pruefung int
	alt := l.an.monitore
	l.an.monitore = func() ([]ui.Monitor, error) { pruefung++; return alt() }
	l.an.monitoreListe = func() ([]ui.Monitor, error) {
		liste++
		return []ui.Monitor{{Name: "DP-1", Breite: 40, Hoehe: 30, Skala: 2, Bruch: 1.5}}, nil
	}
	e := l.rufe(t, "list_screens", `{}`)
	if e.Fehler || !strings.Contains(text(e), `"scale":1.5`) || liste != 1 || pruefung != 0 {
		t.Errorf("list_screens: %s, Liste %d, Prüfung %d", text(e), liste, pruefung)
	}
	l.rufe(t, "capture_screen", `{"screen":"DP-1"}`)
	if liste != 1 || pruefung != 1 {
		t.Errorf("capture_screen: Liste %d, Prüfung %d", liste, pruefung)
	}
}

// Kritik #329 K-2 (F-1030): Das Schema von annotate nennt genau die Typen, die internal/annotate kennt.
func TestMcpAnnotateTypenWieAnnotate(t *testing.T) {
	l := neuerMcpLauf(t)
	for _, w := range l.an.werkzeuge() {
		if w.Name != "annotate" {
			continue
		}
		var s struct {
			Properties struct {
				Annotations struct {
					Items struct {
						Properties struct {
							Type struct {
								Enum []string `json:"enum"`
							} `json:"type"`
						} `json:"properties"`
					} `json:"items"`
				} `json:"annotations"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(w.Eingabe, &s); err != nil {
			t.Fatal(err)
		}
		enum := slices.Clone(s.Properties.Annotations.Items.Properties.Type.Enum)
		slices.Sort(enum)
		if got, soll := strings.Join(enum, ","), strings.Join(annotate.Typen(), ","); got != soll {
			t.Errorf("Schema: %s, annotate: %s", got, soll)
		}
		return
	}
	t.Fatal("kein Werkzeug annotate")
}
