// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Verwaiste Vorschaubilder werden erkannt, nichts wird gelöscht. Eigenes HOME unter t.TempDir() (neueBibliothek).
func TestVerwaisteVorschauen(t *testing.T) {
	b, _ := neueBibliothek(t)
	e1 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z")})
	e2 := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:13Z")})
	if l, err := b.VerwaisteVorschauen(); err != nil || len(l) != 0 {
		t.Fatalf("zu Beginn: %v, %v", l, err)
	}

	// Aufnahme 2 verschwindet aus der Ablage (Dateimanager): ihr Vorschaubild ist verwaist.
	if err := os.Remove(ablageDatei(b, e2.Pfad)); err != nil { // eigene Testdatei im Testverzeichnis
		t.Fatal(err)
	}
	// Dazu ein Vorschaubild ohne jede Aufnahme, eine fremde Datei und ein Unterordner im Cache.
	verwaist1 := filepath.Join(b.Orte.Vorschau, "0123456789abcdef01234567.png")
	schreibe(t, verwaist1, "x")
	fremd := filepath.Join(b.Orte.Vorschau, "notiz.txt")
	schreibe(t, fremd, "bleibt")
	schreibe(t, filepath.Join(b.Orte.Vorschau, "ABCDEF0123456789ABCDEF01.png"), "Name nicht in Vorschauform")
	if err := os.Mkdir(filepath.Join(b.Orte.Vorschau, "aaaaaaaaaaaaaaaaaaaaaaaa.png"), 0o700); err != nil {
		t.Fatal(err)
	}

	vorher := namen(t, b.Orte.Vorschau)
	l, err := b.VerwaisteVorschauen()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{b.VorschauPfad(e2.Pfad), verwaist1}
	if want[0] > want[1] {
		want[0], want[1] = want[1], want[0]
	}
	if !reflect.DeepEqual(l, want) {
		t.Errorf("verwaist %v, soll %v", l, want)
	}
	for _, p := range l {
		if p == b.VorschauPfad(e1.Pfad) {
			t.Error("Vorschau einer vorhandenen Aufnahme gemeldet")
		}
	}
	if n := namen(t, b.Orte.Vorschau); !reflect.DeepEqual(n, vorher) {
		t.Errorf("Cache verändert: %v, vorher %v", n, vorher)
	}
	if _, err := os.Stat(ablageDatei(b, e1.Pfad)); err != nil {
		t.Errorf("Aufnahme berührt: %v", err)
	}
}

// Fehlt die Ablage (etwa nicht eingehängt), kommt ein Fehler statt „alles verwaist“; fehlt der Cache, ist die Liste leer.
func TestVerwaisteVorschauenAblageFehlt(t *testing.T) {
	b, _ := neueBibliothek(t)
	schreibe(t, filepath.Join(b.Orte.Vorschau, "0123456789abcdef01234567.png"), "x")
	if l, err := b.VerwaisteVorschauen(); err == nil {
		t.Errorf("Fehler erwartet, Liste %v", l)
	}
	c, _ := neueBibliothek(t)
	ablegen(t, c, fileformat.Metadaten{})
	if err := os.RemoveAll(c.Orte.Vorschau); err != nil { // eigener Cache im Testverzeichnis
		t.Fatal(err)
	}
	if l, err := c.VerwaisteVorschauen(); err != nil || len(l) != 0 {
		t.Errorf("ohne Cache: %v, %v", l, err)
	}
}
