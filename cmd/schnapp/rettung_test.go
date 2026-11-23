// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/rescue"
	"github.com/Andiministrator/schnapp/internal/ui"
)

// Kein Test dieses Pakets berührt den echten Zustandsordner: Die Rettungskopien liegen in einem eigenen Ordner.
func TestMain(m *testing.M) {
	dialogKindAttrappe() // als Dialog-Kind gestartet: endet hier (mcp_kind_test.go, F-471)
	ort, err := os.MkdirTemp("", "schnapp-state-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return ort
		}
		return os.Getenv(k)
	}
	code := m.Run()
	// Eigener Ordner aus MkdirTemp, Name geprüft (Verify 7, V7-5): kein Rest im Temp-Ordner.
	if strings.HasPrefix(filepath.Base(ort), "schnapp-state-") {
		_ = os.RemoveAll(ort)
	}
	os.Exit(code)
}

// F-144, ADR-0029: Bearbeiten, Rettungskopie schreiben, Ende ohne Speichern (wie SIGKILL), dieselbe Datei wieder
// öffnen: Die Bearbeitung ist zurück, das Dokument gilt als geändert. Nach einem Speichern ist die Kopie weg; nach
// einer Änderung der Datei von anderer Seite wird sie nicht angewendet.
func TestRettungskopieNachAbbruch(t *testing.T) {
	i18n.Setze(nil)
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, "")
	}
	staat := t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })

	// Eine Datei von außen.
	d0, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	aussen := filepath.Join(t.TempDir(), "bild.png")
	f, err := os.Create(aussen)
	if err != nil {
		t.Fatal(err)
	}
	if err := fileformat.SchreibeArbeitsdatei(f, d0, d0.Ansichtsbild(), fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var (
		speichern ui.Speichern
		rettung   ui.Rettung
		dok       *document.Dokument
	)
	altStart := editorStarten
	editorStarten = func(_ string, d *document.Dokument, s ui.Speichern, r ui.Rettung) error {
		speichern, rettung, dok = s, r, d
		return nil
	}
	t.Cleanup(func() { editorStarten = altStart })

	// Erster Lauf: bearbeiten, sichern, „abstürzen“.
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	if rettung == nil {
		t.Fatal("keine Rettung")
	}
	if text, _ := rettung.Wiederherstellen(dok); text != "" {
		t.Fatalf("Meldung ohne Kopie: %q", text)
	}
	mitKasten(t, dok)
	rettung.Sichere(dok)
	rettung.Ende()
	kopie := filepath.Join(staat, "schnapp", "rescue", rescue.Name(aussen))
	if _, err := os.Stat(kopie); err != nil {
		t.Fatalf("Rettungskopie fehlt nach dem Sichern: %v", err)
	}

	// Zweiter Lauf: dieselbe Datei — die Bearbeitung ist zurück.
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	text, fehler := rettung.Wiederherstellen(dok)
	if !strings.Contains(text, "Restored unsaved changes") || fehler {
		t.Errorf("Meldung %q (Fehler %v)", text, fehler)
	}
	if len(dok.Objekte()) != 1 || !dok.KannRueckgaengig() {
		t.Fatalf("nicht wiederhergestellt: %d Objekte", len(dok.Objekte()))
	}

	// Speichern legt eine neue Aufnahme an (Datei von außen) und verwirft die Kopie der alten Ausgangsdatei.
	if _, err := speichern(dok); err != nil {
		t.Fatal(err)
	}
	rettung.Verwirf()
	rettung.Ende() // Löschen läuft in der Kette (F-417); Fenster zu gibt die Sperre frei (F-413)
	if _, err := os.Stat(kopie); !os.IsNotExist(err) {
		t.Errorf("Kopie nach dem Speichern noch da: %v", err)
	}
	eintraege, _ := os.ReadDir(filepath.Dir(kopie))
	if len(eintraege) != 0 {
		t.Errorf("%d Kopien liegen noch", len(eintraege))
	}

	// Kopie anlegen, dann die Datei von anderer Seite ändern: nicht angewendet, Kopie bleibt, Meldung als Warnung.
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, dok)
	rettung.Sichere(dok)
	rettung.Ende()
	inhalt, _ := os.ReadFile(aussen)
	if err := os.WriteFile(aussen, append(inhalt, 0), 0o600); err != nil { // anderer Inhalt, PNG liest weiter
		t.Fatal(err)
	}
	if err := oeffneImEditor(aussen, false, document.KanteAbriss); err != nil {
		t.Fatal(err)
	}
	text, fehler = rettung.Wiederherstellen(dok)
	if !strings.Contains(text, "has changed since") || !fehler || len(dok.Objekte()) != 0 {
		t.Errorf("trotz geänderter Datei: %q %v %d", text, fehler, len(dok.Objekte()))
	}
	if !strings.Contains(text, "schnapp/rescue/") || strings.Contains(text, staat) {
		t.Errorf("Meldung nennt den Ort nicht ohne Benutzerpfad: %q", text)
	}
	// F-488: Die Aussage steht vor dem Ort, und der Ort ist gekürzt (8 Hex-Zeichen statt 64 samt Endung).
	if i := strings.Index(text, "has changed since"); i < 0 || i > 100 || i > strings.Index(text, "schnapp/rescue/") {
		t.Errorf("Aussage steht nicht vor dem Ort (Index %d): %q", i, text)
	}
	if strings.Contains(text, ".json") || len(text) > 160 {
		t.Errorf("Ort nicht gekürzt (%d Zeichen): %q", len(text), text)
	}
	if !strings.Contains(text, "schnapp/rescue/"+rescue.Name(aussen)[:8]+"…") {
		t.Errorf("gekürzter Name fehlt: %q", text)
	}
	// Die Kopie liegt beiseite (V7-1) und überlebt das Schließen dieser Sitzung ohne Änderung.
	rettung.Verwirf()
	rettung.Ende()
	eintraege, _ = os.ReadDir(filepath.Dir(kopie))
	if len(eintraege) != 1 || !strings.HasSuffix(eintraege[0].Name(), ".alt.json") {
		t.Errorf("beiseitegelegte Kopie fehlt: %v", eintraege)
	}
}
