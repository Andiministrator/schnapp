// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// F-310: Speichern verkleinert das Vorschaubild nebenläufig zum Kodieren; geschrieben wird genau das verkleinerte
// sichtbare Bild (nicht das Original, nicht ein halbfertiges).
func TestSpeichernVorschauIstDasVerkleinerteSichtbare(t *testing.T) {
	b, _ := neueBibliothek(t)
	d := neuesDokument(t, 600, 400)
	e, err := b.Ablegen(d, d.Ansichtsbild(), fileformat.Metadaten{})
	if err != nil {
		t.Fatal(err)
	}
	sichtbar := image.NewNRGBA(image.Rect(0, 0, 600, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 600; x++ {
			sichtbar.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 200, 255})
		}
	}
	if _, _, err := b.Speichern(e.Pfad, standIn(t, b, e.Pfad), d, sichtbar); err != nil {
		t.Fatal(err)
	}
	var soll bytes.Buffer
	if err := png.Encode(&soll, verkleinere(sichtbar, VorschauSeite)); err != nil {
		t.Fatal(err)
	}
	ist, err := os.ReadFile(b.VorschauPfad(e.Pfad))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ist, soll.Bytes()) {
		t.Errorf("Vorschau weicht ab (%d zu %d Bytes)", len(ist), soll.Len())
	}
}

// F-310: Bricht Speichern vor dem Kodieren ab (fremde Datei), wartet es auf die Verkleinerung und schreibt keine
// Vorschau (die Datei bleibt, wie sie war).
func TestSpeichernAbbruchSchreibtKeineVorschau(t *testing.T) {
	b, _ := neueBibliothek(t)
	ordner := filepath.Join(b.Orte.Ablage, "2026-11")
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(ordner, "fremd.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, muster(600, 400)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	d := neuesDokument(t, 600, 400)
	// Fremde Datei: Speichern bricht nach dem Start der Verkleinerung mit ErrFremd ab.
	_, _, err = b.Speichern("2026-11/fremd.png", standIn(t, b, "2026-11/fremd.png"), d, d.Ansichtsbild())
	if !errors.Is(err, ErrFremd) {
		t.Fatalf("Fehler %v, erwartet ErrFremd", err)
	}
	if _, err := os.Stat(b.VorschauPfad("2026-11/fremd.png")); err == nil {
		t.Error("Vorschau trotz Abbruch geschrieben")
	}
}
