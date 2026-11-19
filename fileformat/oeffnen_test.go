// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

func schreibeDatei(t *testing.T, pfad string, daten []byte) {
	t.Helper()
	if err := os.WriteFile(pfad, daten, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOeffneArbeitsdatei(t *testing.T) {
	d, err := document.Neu(testbild(8, 6))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(4, 4)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 1}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{Schlagworte: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(t.TempDir(), "a.png")
	schreibeDatei(t, pfad, buf.Bytes())
	g, err := Oeffne(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if g.Fremd || len(g.Dokument.Objekte()) != 1 || len(g.Meta.Schlagworte) != 1 {
		t.Errorf("Fremd %v, Objekte %d, Meta %+v", g.Fremd, len(g.Dokument.Objekte()), g.Meta)
	}
}

func TestOeffneFremdeBilder(t *testing.T) {
	dir := t.TempDir()
	var p, j bytes.Buffer
	if err := png.Encode(&p, testbild(5, 4)); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&j, testbild(7, 3), nil); err != nil {
		t.Fatal(err)
	}
	for name, daten := range map[string][]byte{"p.png": p.Bytes(), "j.jpg": j.Bytes()} {
		pfad := filepath.Join(dir, name)
		schreibeDatei(t, pfad, daten)
		g, err := Oeffne(pfad)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !g.Fremd || len(g.Dokument.Objekte()) != 0 {
			t.Errorf("%s: Fremd %v", name, g.Fremd)
		}
	}
	if g, _ := Oeffne(filepath.Join(dir, "j.jpg")); g.Dokument.Groesse().X != 7 {
		t.Errorf("JPEG-Größe %v", g.Dokument.Groesse())
	}
}

// Eine beschädigte Arbeitsdatei (Block fehlt) öffnet als Bild, Fremd = true: nie an ihren Platz zurückschreiben.
func TestOeffneBeschaedigteArbeitsdateiAlsBild(t *testing.T) {
	d, _ := document.Neu(testbild(4, 4))
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, d.Ansichtsbild(), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	bloecke, err := leseBloecke(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	var kaputt bytes.Buffer
	if err := schreibeBloecke(&kaputt, ohneBloecke(bloecke, blockOriginal)); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(t.TempDir(), "k.png")
	schreibeDatei(t, pfad, kaputt.Bytes())
	g, err := Oeffne(pfad)
	if err != nil || !g.Fremd {
		t.Fatalf("%+v, %v", g, err)
	}
}

func TestOeffneFehler(t *testing.T) {
	dir := t.TempDir()
	if _, err := Oeffne(dir); !errors.Is(err, ErrKeineDatei) {
		t.Errorf("Ordner: %v", err)
	}
	if _, err := Oeffne(filepath.Join(dir, "fehlt.png")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("fehlt: %v", err)
	}
	pfad := filepath.Join(dir, "text.txt")
	schreibeDatei(t, pfad, []byte("kein Bild"))
	if _, err := Oeffne(pfad); !errors.Is(err, ErrKeinPNG) {
		t.Errorf("Text: %v", err)
	}
}

// Eine Datei über MaxDatei wird abgelehnt, ohne sie zu lesen (Kritik A-13, Verify V-2): dünn besetzte Datei.
func TestOeffneZuGrosseDatei(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "riesig.png")
	f, err := os.Create(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxDatei + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := Oeffne(pfad); !errors.Is(err, ErrZuGross) {
		t.Errorf("Fehler %v, Soll ErrZuGross", err)
	}
}
