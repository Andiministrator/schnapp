// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/po"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// ADR-0030 Punkte 1 bis 4: --json, --output=DATEI und --export=ZIEL DATEI auswerten; was nicht zusammenpasst, ist ein
// Aufruffehler mit einem Text, der die richtige Schreibweise nennt.
func TestAuswertenAgenten(t *testing.T) {
	i18n.Setze(nil)
	for _, fall := range []struct {
		args []string
		soll Aufruf
	}{
		{[]string{"--json"}, Aufruf{Modus: ModusBereich, JSON: true}},
		{[]string{"--screen=A", "--output=x.png", "--json"}, Aufruf{Modus: ModusBildschirm, Bildschirm: "A", Ausgabe: "x.png", JSON: true}},
		{[]string{"--output=a.JPEG", "--output=a.JPEG", "--window"}, Aufruf{Modus: ModusFenster, Ausgabe: "a.JPEG"}},
		{[]string{"--repeat", "--output=b.svg", "--no-editor"}, Aufruf{Modus: ModusWiederholen, Ausgabe: "b.svg", OhneEditor: true}},
		{[]string{"--list-screens", "--json"}, Aufruf{Modus: ModusMonitorliste, JSON: true}},
		{[]string{"--json", "--version"}, Aufruf{Modus: ModusVersion, JSON: true}},
		{[]string{"--export=x.png", "f.png"}, Aufruf{Modus: ModusExport, Datei: "f.png", Ziel: "x.png"}},
		{[]string{"in.png", "--ignore-limit", "--json", "--export=out.svg"}, Aufruf{Modus: ModusExport, Datei: "in.png", Ziel: "out.svg", OhneGrenze: true, JSON: true}},
	} {
		a, err := Auswerten(fall.args)
		if err != nil || a != fall.soll {
			t.Errorf("%v: %+v, %v; Soll %+v", fall.args, a, err, fall.soll)
		}
	}
	for _, fall := range []struct {
		args []string
		text string
	}{
		{[]string{"--output"}, "--output=FILE"},
		{[]string{"--output=", "--window"}, "--output=FILE"},
		{[]string{"--export", "f.png"}, "--export=FILE"},
		{[]string{"--output=x.gif"}, ".png, .jpg or .svg"},
		{[]string{"--output=ohne"}, ".png, .jpg or .svg"},
		{[]string{"--export=x.bmp", "f.png"}, ".png, .jpg or .svg"},
		{[]string{"--output=a.png", "--output=b.png"}, "more than once"},
		{[]string{"--output=a.png", "--list-screens"}, "cannot be combined"},
		{[]string{"--output=a.png", "f.png"}, "cannot be combined"},
		{[]string{"--output=a.png", "--export=b.png", "f.png"}, "cannot be combined"},
		{[]string{"--export=x.png"}, "--export=TARGET FILE"},
		{[]string{"--export=x.png", "--region"}, "cannot be combined"},
		{[]string{"--export=x.png", "--no-editor", "f.png"}, "cannot be combined"},
		{[]string{"--json", "--help"}, "cannot be combined"},
		{[]string{"--json", "f.png"}, "cannot be combined"},
		{[]string{"--json=ja"}, "takes no value"},
	} {
		a, err := Auswerten(fall.args)
		if err == nil || !strings.Contains(err.Error(), fall.text) {
			t.Errorf("%v: %+v, Fehler %v; Soll mit %q", fall.args, a, err, fall.text)
		}
	}
}

// --version und --list-screens mit --json: eine Zeile JSON mit englischen Schlüsseln, auch bei deutscher Sprache
// (F-98: der Name ist genau der Wert für --screen=NAME).
func TestLaufeJSON(t *testing.T) {
	k, err := i18n.Lade(po.Dateien, "de")
	if err != nil {
		t.Fatal(err)
	}
	i18n.Setze(k)
	t.Cleanup(func() { i18n.Setze(nil) })
	code, aus, _ := fuehre(t, keineMonitore, "--version", "--json")
	var v struct{ Version string }
	if code != exitOK || json.Unmarshal([]byte(aus), &v) != nil || v.Version != versionsText() || strings.Count(aus, "\n") != 1 {
		t.Errorf("--version --json: Code %d, %q", code, aus)
	}
	zwei := func() ([]ui.Monitor, error) {
		return []ui.Monitor{{Name: "DP-1", Breite: 2560, Hoehe: 1440, Skala: 2}, {Name: "HDMI-A-1", Breite: 1920, Hoehe: 1080}}, nil
	}
	code, aus, _ = fuehre(t, zwei, "--list-screens", "--json")
	soll := `{"screens":[{"name":"DP-1","width":2560,"height":1440},{"name":"HDMI-A-1","width":1920,"height":1080}]}` + "\n"
	if code != exitOK || aus != soll {
		t.Errorf("--list-screens --json: Code %d, %q; Soll %q", code, aus, soll)
	}
}

// --output schaltet den Editor ab, auch wenn die Einstellung ihn öffnen will (ADR-0030 Punkt 2).
func TestPlaneAusgabeOhneEditor(t *testing.T) {
	e := settings.Voreinstellungen()
	e.Editor = true
	if p := plane(Aufruf{Modus: ModusBereich, Ausgabe: "x.png", JSON: true}, e); p.Editor || p.Ausgabe != "x.png" || !p.JSON {
		t.Errorf("Auftrag %+v", p)
	}
	if p := plane(Aufruf{Modus: ModusBereich}, e); !p.Editor {
		t.Error("ohne --output öffnet der Editor nicht mehr")
	}
}

// Eine Aufnahme mit --output landet in der Datei, nicht in der Bibliothek; kein Editor, kein Aufräumen. Mit --json
// steht der absolute Pfad samt Größe als JSON da; jedes Format nach seiner Endung.
func TestAufnahmeMitOutput(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	for _, fall := range []struct {
		name  string
		magie string
	}{{"bild.png", "\x89PNG"}, {"bild.JPG", "\xff\xd8\xff"}, {"bild.svg", "<?xml"}} {
		l := neuerLauf(t)
		bearbeitet := 0
		l.an.bearbeiten = func(abgelegt) error { bearbeitet++; return nil }
		l.an.aufraeumen = nil
		ziel := filepath.Join(dir, fall.name)
		code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster, Ausgabe: ziel, JSON: true, Editor: true,
			Aufraeumen: settings.Voreinstellungen().Bibliothek})
		var e ergebnis
		if code != exitOK || fehler != "" || json.Unmarshal([]byte(aus), &e) != nil || e != (ergebnis{ziel, 2, 2}) {
			t.Fatalf("%s: Code %d, aus %q, fehler %q", fall.name, code, aus, fehler)
		}
		if len(l.abgelegt) != 0 || bearbeitet != 0 {
			t.Errorf("%s: abgelegt %d, Editor %d", fall.name, len(l.abgelegt), bearbeitet)
		}
		b, err := os.ReadFile(ziel)
		if err != nil || !strings.HasPrefix(string(b), fall.magie) {
			t.Fatalf("%s: %v, Anfang %q", fall.name, err, b[:min(8, len(b))])
		}
		if fall.magie == "\x89PNG" {
			p, err := png.Decode(bytes.NewReader(b))
			if err != nil || p.Bounds().Size() != image.Pt(2, 2) {
				t.Fatalf("PNG: %v", err)
			}
			if r, g, bl, _ := p.At(1, 0).RGBA(); r>>8 != 1 || g>>8 != 0 || bl>>8 != 4 {
				t.Errorf("Pixel (1,0) = %d %d %d, Soll das Fensterbild", r>>8, g>>8, bl>>8)
			}
			if bytes.Contains(b, []byte("snAP")) || bytes.Contains(b, []byte("snAO")) {
				t.Error("--output trägt Arbeitsdatei-Blöcke")
			}
		}
	}
	// Ohne --json nur der absolute Pfad; ein relativer Pfad wird absolut gemeldet.
	l := neuerLauf(t)
	t.Chdir(dir)
	code, aus, _ := l.fuehre(Auftrag{Modus: ModusFenster, Ausgabe: "rel.png"})
	if code != exitOK || aus != filepath.Join(dir, "rel.png")+"\n" {
		t.Errorf("relativ: Code %d, aus %q", code, aus)
	}
	// Ordner fehlt: Fehler, nichts abgelegt, keine Pfadzeile.
	l = neuerLauf(t)
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster, Ausgabe: filepath.Join(dir, "fehlt", "x.png")})
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "The folder does not exist.") || len(l.abgelegt) != 0 {
		t.Errorf("fehlender Ordner: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
}

// In die Bibliothek mit --json: dieselbe Angabe als JSON-Zeile.
func TestAufnahmeInBibliothekMitJSON(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.overlayErg = ui.OverlayErgebnis{Monitor: 1, Bereich: image.Rect(1, 1, 4, 3)}
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich, JSON: true})
	soll := `{"path":"/ablage/2026-11/datei.png","width":3,"height":2}` + "\n"
	if code != exitOK || aus != soll || fehler != "" || len(l.abgelegt) != 1 {
		t.Errorf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
}

// --export=ZIEL DATEI: das sichtbare Bild (Zuschnitt, Beschriftungen) ohne Fenster; die Datei bleibt unverändert.
func TestExportOhneFenster(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	quelle := filepath.Join(dir, "arbeit.png")
	d, err := document.Neu(testbild(40, 30, 7))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 5, 30, 25)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(12, 7), document.Pt(20, 15)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	var a bytes.Buffer
	sichtbar := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	if err := fileformat.SchreibeArbeitsdatei(&a, d, sichtbar, fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quelle, a.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	ziel := filepath.Join(dir, "aus.png")
	code, aus, fehler := fuehre(t, keineMonitore, "--export="+ziel, "--json", quelle)
	var e ergebnis
	if code != exitOK || fehler != "" || json.Unmarshal([]byte(aus), &e) != nil || e != (ergebnis{ziel, 20, 20}) {
		t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	b, err := os.ReadFile(ziel)
	if err != nil {
		t.Fatal(err)
	}
	p, err := png.Decode(bytes.NewReader(b))
	if err != nil || p.Bounds().Size() != image.Pt(20, 20) {
		t.Fatalf("Export %v", err)
	}
	if r, _, _, _ := p.At(2, 6).RGBA(); r>>8 < 200 { // Kastenrand bei Original (12,11) = Ansicht (2,6)
		t.Errorf("Kasten fehlt im Export: R %d", r>>8)
	}
	if r, g, bl, _ := p.At(15, 15).RGBA(); r>>8 != 25 || g>>8 != 20 || bl>>8 != 7 {
		t.Errorf("Pixel (15,15) = %d %d %d, Soll Original (25,20)", r>>8, g>>8, bl>>8)
	}
	if danach, _ := os.ReadFile(quelle); !bytes.Equal(danach, a.Bytes()) {
		t.Error("--export hat die Datei verändert")
	}
	// Fehlende Datei: Fehler, kein Ziel.
	code, aus, fehler = fuehre(t, keineMonitore, "--export="+filepath.Join(dir, "nie.png"), filepath.Join(dir, "fehlt.png"))
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "does not exist") {
		t.Errorf("fehlende Datei: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if _, err := os.Stat(filepath.Join(dir, "nie.png")); err == nil {
		t.Error("Ziel trotz Fehler angelegt")
	}
}
