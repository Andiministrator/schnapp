// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
)

// pngBytes liefert ein leeres PNG der Größe w×h (ein fremdes Bild von außerhalb der Bibliothek).
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// blockiereVorschau legt an die Stelle des Vorschaubilds zu rel einen nicht leeren Ordner. Das Neuschreiben der
// Vorschau scheitert dann (Umbenennen einer Datei über einen Ordner), und das alte „Bild“ lässt sich nicht entfernen
// (Ordner nicht leer) — ein Fehlschlag im eigenen Testverzeichnis, ohne Eingriff von außen.
func blockiereVorschau(t *testing.T, b *library.Bibliothek, rel string) {
	t.Helper()
	pfad := b.VorschauPfad(rel)
	if err := os.RemoveAll(pfad); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pfad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pfad, "stein"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// F-127: Bleibt beim Zurückschreiben einer Aufnahme die Vorschau des vorigen Stands liegen (ErrVorschauAlt), warnt der
// Editor mit dem Hinweis auf die alte Vorschau — ohne neue Datei, und das Speichern gilt (kein Fehler). Gegenstück:
// scheitert nur das Schreiben der Vorschau (alte entfernt), ist es ein stilles Speichern (F-115).
func TestSpeichernWarntBeiBleibenderVorschau(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 5, 5)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	g, err := fileformat.Oeffne(abs)
	if err != nil {
		t.Fatal(err)
	}
	blockiereVorschau(t, b, e.Pfad)
	mitKasten(t, g.Dokument)
	erg, err := speichernIn(b, abs, g, standVon(t, abs))(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	if !erg.Warnung || erg.Hinweis != "Saved, but the old preview of this capture could not be removed." || erg.Datei != "" {
		t.Errorf("Ergebnis %+v, Soll Warnung mit dem Hinweis auf die alte Vorschau, ohne neue Datei", erg)
	}
	// Gespeichert ist es trotzdem, an Ort und Stelle.
	if neu, err := fileformat.Oeffne(abs); err != nil || len(neu.Dokument.Objekte()) != 1 {
		t.Errorf("Aufnahme nicht zurückgeschrieben: %v", err)
	}
	// Gegenstück: Lässt sich die alte Vorschau entfernen, bleibt es still (Ordner statt Bild, leer: Remove gelingt).
	if err := os.RemoveAll(b.VorschauPfad(e.Pfad)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(b.VorschauPfad(e.Pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	if erg, err := speichernIn(b, abs, g, standVon(t, abs))(g.Dokument); err != nil || erg.Warnung || erg.Hinweis != "" {
		t.Errorf("Gegenstück: %+v, %v (Soll still)", erg, err)
	}
}

// F-127: Weicht das Speichern auf eine neue Aufnahme aus und bleibt auch deren alte Vorschau liegen, hängt der Hinweis
// die Warnung an den Hinweis auf die neue Aufnahme an.
func TestAblegenWarntBeiBleibenderVorschau(t *testing.T) {
	i18n.Setze(nil)
	b := testBibliothek(t)
	// Der Name der neuen Aufnahme steht mit der festen Uhr fest: erst in einer Wegwerf-Bibliothek ablesen.
	probe := testBibliothek(t)
	pd, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 6, 4)))
	if err != nil {
		t.Fatal(err)
	}
	pe, err := probe.Ablegen(pd, pd.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	blockiereVorschau(t, b, pe.Pfad)

	aussen := filepath.Join(t.TempDir(), "fremd.png")
	if err := os.WriteFile(aussen, pngBytes(t, 6, 4), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	erg, err := speichernIn(b, aussen, g, standVon(t, aussen))(g.Dokument)
	if err != nil {
		t.Fatalf("Speichern: %v", err)
	}
	soll := "Saved as a new capture in the library: schnapp/" + pe.Pfad + " (the original file is unchanged) The old preview could not be removed."
	if !erg.Warnung || erg.Hinweis != soll || erg.Datei != filepath.Base(filepath.FromSlash(pe.Pfad)) {
		t.Errorf("Ergebnis %+v, Soll Warnung mit %q", erg, soll)
	}
}

// F-127: Scheitert das Ablegen beim Speichern (hier: Bibliotheksordner nicht anlegbar), kommt der Fehler übersetzt
// zurück — ohne Pfad und Paketpräfix (F-117) — und bleibt als Ursache erkennbar. Das gilt auch, wenn vorher das
// Zurückschreiben scheiterte.
func TestSpeichernUebersetztAblegenFehler(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("als root wirken Ordnerrechte nicht")
	}
	i18n.Setze(nil)
	b := testBibliothek(t)
	heim := filepath.Dir(filepath.Dir(b.Orte.Ablage)) // Ablage liegt unter <Heim>/<Bilder>/schnapp
	for _, ebene := range []string{b.Orte.Ablage, filepath.Dir(b.Orte.Ablage)} {
		if _, err := os.Stat(ebene); err == nil {
			t.Fatalf("%s existiert schon", ebene)
		}
	}
	aussen := filepath.Join(t.TempDir(), "fremd.png")
	if err := os.WriteFile(aussen, pngBytes(t, 6, 4), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := fileformat.Oeffne(aussen)
	if err != nil {
		t.Fatal(err)
	}
	mitKasten(t, g.Dokument)
	if err := os.Chmod(heim, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(heim, 0o700) })
	_, err = speichernIn(b, aussen, g, standVon(t, aussen))(g.Dokument)
	if err == nil {
		t.Fatal("Fehler erwartet (Heimordner nicht beschreibbar)")
	}
	if err.Error() != "No permission to write the file." || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Fehler %q (%v), Soll übersetzt „No permission to write the file.“ mit fs.ErrPermission als Ursache", err, errors.Unwrap(err))
	}
	if strings.Contains(err.Error(), heim) {
		t.Errorf("Fehlertext nennt den Pfad: %q", err)
	}
	// Als Fehler beim Öffnen gelesen, bleibt der übersetzte Text (nutzerFehler).
	if got := nutzerFehler(err); got != "No permission to write the file." {
		t.Errorf("nutzerFehler: %q", got)
	}
}
