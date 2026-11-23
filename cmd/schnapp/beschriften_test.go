// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

// ADR-0030 Nachtrag 10: --annotate=LISTE DATEI.

// eigenesHeim setzt HOME auf einen frischen Ordner (Bibliothek darunter) und liefert die Bibliothek.
func eigenesHeim(t *testing.T) *library.Bibliothek {
	t.Helper()
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(name, "")
	}
	orte, err := library.OrteAus(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	return library.Neu(orte)
}

func schreibeListe(t *testing.T, dir, inhalt string) string {
	t.Helper()
	p := filepath.Join(dir, "liste.json")
	if err := os.WriteFile(p, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAuswertenAnnotate(t *testing.T) {
	i18n.Setze(nil)
	for _, fall := range []struct {
		args []string
		soll Aufruf
	}{
		{[]string{"--annotate=l.json", "f.png"}, Aufruf{Modus: ModusBeschriften, Datei: "f.png", Spez: "l.json"}},
		{[]string{"f.png", "--annotate=-", "--json", "--ignore-limit"}, Aufruf{Modus: ModusBeschriften, Datei: "f.png", Spez: "-", JSON: true, OhneGrenze: true}},
	} {
		if a, err := Auswerten(fall.args); err != nil || a != fall.soll {
			t.Errorf("%v: %+v, %v", fall.args, a, err)
		}
	}
	for _, fall := range []struct {
		args []string
		text string
	}{
		{[]string{"--annotate", "f.png"}, "--annotate=LIST FILE"},
		{[]string{"--annotate=", "f.png"}, "--annotate=LIST FILE"},
		{[]string{"--annotate=l.json"}, "needs the file"},
		{[]string{"--annotate=l.json", "--region"}, "cannot be combined"},
		{[]string{"--annotate=l.json", "--export=x.png", "f.png"}, "cannot be combined"},
		{[]string{"--annotate=a.json", "--annotate=b.json", "f.png"}, "more than once"},
	} {
		if a, err := Auswerten(fall.args); err == nil || !strings.Contains(err.Error(), fall.text) {
			t.Errorf("%v: %+v, %v; Soll mit %q", fall.args, a, err, fall.text)
		}
	}
}

// TestAnnotateFremdesBild: Ein einfaches PNG bleibt unverändert; das Ergebnis ist eine neue Aufnahme der Bibliothek
// mit den Beschriftungen und der Schwärzung, die JSON-Zeile nennt sie (ADR-0030 Nachtrag 10 Punkte 1 und 4). Die
// Schwärzung lässt die Ausgangsdatei mit dem Klartext stehen: warning true.
func TestAnnotateFremdesBild(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, testbild(40, 30, 7)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	vorher := append([]byte(nil), buf.Bytes()...)
	liste := schreibeListe(t, dir, `{"annotations":[{"type":"arrow","from":[2,2],"to":[30,20]},
		{"type":"redact","from":[0,0],"to":[10,10]}]}`)
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate="+liste, "--json", quelle)
	var e beschriftet
	if code != exitOK || json.Unmarshal([]byte(aus), &e) != nil {
		t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if !strings.HasPrefix(e.Path, b.Orte.Ablage+string(filepath.Separator)) || e.Width != 40 || e.Height != 30 ||
		e.Annotations != 1 || e.Redactions != 1 || !e.Warning || fehler == "" {
		t.Errorf("Ergebnis %+v, fehler %q", e, fehler)
	}
	if b2, _ := os.ReadFile(quelle); !bytes.Equal(b2, vorher) {
		t.Error("Ausgangsdatei verändert")
	}
	g, err := fileformat.Oeffne(e.Path)
	if err != nil {
		t.Fatal(err)
	}
	if o := g.Dokument.Objekte(); len(o) != 1 {
		t.Errorf("Objekte %v", o)
	}
	if c := g.Dokument.OriginalBild().NRGBAAt(5, 5); c != canvas.Schwaerzfarbe {
		t.Errorf("Original nicht geschwärzt: %v", c)
	}
}

// TestAnnotateArbeitsdateiDerBibliothek: Eine Aufnahme der Bibliothek wird fortgeschrieben (Pfad bleibt), die Liste
// kommt von stdin.
func TestAnnotateArbeitsdateiDerBibliothek(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	d, err := document.Neu(testbild(40, 30, 7))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	alt := eingabe
	t.Cleanup(func() { eingabe = alt })
	eingabe = strings.NewReader(`{"annotations":[{"type":"step","at":[20,15]},{"type":"text","at":[1,1],"text":"x"}]}`)
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate=-", pfad)
	if code != exitOK || strings.TrimSpace(aus) != pfad || fehler != "" {
		t.Fatalf("Code %d, aus %q (Soll %q), fehler %q", code, aus, pfad, fehler)
	}
	g, err := fileformat.Oeffne(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if o := g.Dokument.Objekte(); len(o) != 2 {
		t.Errorf("Objekte %v", o)
	}
}

// TestAnnotateAllesOderNichts: Ein falscher Eintrag (hier der zweite) speichert nichts, Exit 2 mit Index; eine
// fehlende Datei Exit 1; eine unlesbare Liste Exit 2.
func TestAnnotateAllesOderNichts(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, testbild(40, 30, 7)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	liste := schreibeListe(t, dir, `{"annotations":[{"type":"redact","from":[0,0],"to":[10,10]},
		{"type":"step","at":[99,1]}]}`)
	code, aus, fehler := fuehre(t, keineMonitore, "--annotate="+liste, quelle)
	if code != exitAufruf || aus != "" || !strings.Contains(fehler, "annotation 1") {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if es, _ := b.Suchen(library.Anfrage{}); len(es) != 0 {
		t.Errorf("trotz Fehler gespeichert: %v", es)
	}
	if code, _, _ := fuehre(t, keineMonitore, "--annotate="+liste, filepath.Join(dir, "fehlt.png")); code != exitFehler {
		t.Errorf("fehlende Datei: Code %d", code)
	}
	kaputt := schreibeListe(t, dir, `{"annotations":[{"type":"step"}]}`)
	if code, _, fehler := fuehre(t, keineMonitore, "--annotate="+kaputt, quelle); code != exitAufruf || !strings.Contains(fehler, `needs key "at"`) {
		t.Errorf("unvollständige Liste: Code %d, %q", code, fehler)
	}
	if code, _, _ := fuehre(t, keineMonitore, "--annotate="+filepath.Join(dir, "keine.json"), quelle); code != exitFehler {
		t.Errorf("fehlende Liste: Code %d", code)
	}
}

// TestAnnotateRechenbudget: Läge das Ergebnis über dem Budget, wird nichts gespeichert (Exit 1); --ignore-limit
// speichert mit Warnung.
func TestAnnotateRechenbudget(t *testing.T) {
	i18n.Setze(nil)
	b := eigenesHeim(t)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "bild.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 4000, 3000))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	var l strings.Builder
	l.WriteString(`{"annotations":[`)
	for i := 0; i < 400; i++ {
		if i > 0 {
			l.WriteString(",")
		}
		l.WriteString(`{"type":"rect","from":[0,0],"to":[4000,3000],"fill":"#11223380","width":500}`)
	}
	l.WriteString("]}")
	liste := schreibeListe(t, dir, l.String())
	code, _, fehler := fuehre(t, keineMonitore, "--annotate="+liste, quelle)
	if code != exitFehler || !strings.Contains(fehler, "too many or too large") {
		t.Fatalf("über dem Budget: Code %d, %q", code, fehler)
	}
	if es, _ := b.Suchen(library.Anfrage{}); len(es) != 0 {
		t.Errorf("über dem Budget gespeichert: %v", es)
	}
}
