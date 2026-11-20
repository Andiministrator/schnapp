// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/document"
)

func TestWaehle(t *testing.T) {
	for _, c := range []struct {
		pfad   string
		filter Format
		f      Format
		soll   string
	}{
		{"/x/a.png", JPG, PNG, "/x/a.png"},   // Endung gewinnt gegen den Filter
		{"/x/a.JPG", PNG, JPG, "/x/a.JPG"},   // Groß-/Kleinschreibung egal
		{"/x/a.jpeg", PNG, JPG, "/x/a.jpeg"}, // jpeg zählt als JPG
		{"/x/a.SvG", PNG, SVG, "/x/a.SvG"},
		{"/x/a", JPG, JPG, "/x/a.jpg"}, // keine Endung: Filter, Endung wird angehängt
		{"/x/a", SVG, SVG, "/x/a.svg"},
		{"/x/a", PNG, PNG, "/x/a.png"},
		{"/x/a.txt", SVG, SVG, "/x/a.txt.svg"}, // unbekannte Endung: Filter, Endung angehängt
		{"/x/v1.2", JPG, JPG, "/x/v1.2.jpg"},
	} {
		f, p := Waehle(c.pfad, c.filter)
		if f != c.f || p != c.soll {
			t.Errorf("Waehle(%q, %v) = %v, %q; soll %v, %q", c.pfad, c.filter, f, p, c.f, c.soll)
		}
	}
}

func TestAusIndex(t *testing.T) {
	for i, f := range []Format{PNG, JPG, SVG} {
		if AusIndex(i) != f {
			t.Errorf("Index %d", i)
		}
	}
	if AusIndex(-1) != PNG || AusIndex(9) != PNG {
		t.Error("unbekannter Index soll PNG sein")
	}
}

func testDokument(t *testing.T) *document.Dokument {
	t.Helper()
	b := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			b.SetNRGBA(x, y, color.NRGBA{0x80, 0x90, 0xA0, 0xFF})
		}
	}
	d, err := document.Neu(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Die drei Formate lassen sich wieder lesen, tragen keine Bearbeitungsdaten und keine Pfade oder Namen.
func TestSchreibeFormateOhneMetadaten(t *testing.T) {
	d := testDokument(t)
	var p, j, s bytes.Buffer
	for f, w := range map[Format]*bytes.Buffer{PNG: &p, JPG: &j, SVG: &s} {
		if err := Schreibe(d, f, w); err != nil {
			t.Fatalf("Format %v: %v", f, err)
		}
	}
	if _, err := png.Decode(bytes.NewReader(p.Bytes())); err != nil {
		t.Errorf("PNG: %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(j.Bytes())); err != nil {
		t.Errorf("JPG: %v", err)
	}
	for _, kennung := range []string{"snAP", "snAO", "tEXt", "iTXt", "zTXt", "tIME", "eXIf"} {
		if bytes.Contains(p.Bytes(), []byte(kennung)) {
			t.Errorf("PNG enthält Block %s", kennung)
		}
	}
	for _, spur := range []string{"/home", "/tmp", "/Users", "C:\\"} {
		for n, b := range map[string][]byte{"PNG": p.Bytes(), "JPG": j.Bytes(), "SVG": s.Bytes()} {
			if bytes.Contains(b, []byte(spur)) {
				t.Errorf("%s enthält %q", n, spur)
			}
		}
	}
	if hn, err := os.Hostname(); err == nil && len(hn) > 3 {
		for n, b := range map[string][]byte{"PNG": p.Bytes(), "JPG": j.Bytes(), "SVG": s.Bytes()} {
			if bytes.Contains(b, []byte(hn)) {
				t.Errorf("%s enthält den Rechnernamen", n)
			}
		}
	}
	if !strings.HasPrefix(s.String(), "<?xml") || !strings.Contains(s.String(), "<svg") {
		t.Error("SVG ohne Kopf")
	}
}

func TestSchreibeAtomarNeueDatei(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "neu.png")
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { _, err := w.Write([]byte("inhalt")); return err }); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ziel)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o644 {
		t.Errorf("Rechte %v, soll 0644", st.Mode().Perm())
	}
	if b, _ := os.ReadFile(ziel); string(b) != "inhalt" {
		t.Errorf("Inhalt %q", b)
	}
	if n := eintraege(t, ordner); n != 1 {
		t.Errorf("%d Einträge im Ordner, soll 1 (keine Reste)", n)
	}
}

// Scheitert das Schreiben mittendrin, bleibt eine vorhandene Datei unverändert und es bleibt kein Rest liegen
// (nie eine halbe Datei am Ziel).
func TestSchreibeAtomarFehlerLaesstZielUnberuehrt(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "alt.png")
	if err := os.WriteFile(ziel, []byte("alter Inhalt"), 0o600); err != nil {
		t.Fatal(err)
	}
	kaputt := errors.New("Platte voll")
	err := SchreibeAtomar(ziel, func(w io.Writer) error {
		w.Write([]byte("halb"))
		return kaputt
	})
	if !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "alter Inhalt" {
		t.Errorf("Ziel verändert: %q", b)
	}
	if n := eintraege(t, ordner); n != 1 {
		t.Errorf("%d Einträge im Ordner, soll 1 (temporäre Datei aufgeräumt)", n)
	}
}

func TestSchreibeAtomarUeberschreibt(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "a.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { _, err := w.Write([]byte("neu")); return err }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "neu" {
		t.Errorf("Inhalt %q", b)
	}
}

func TestSchreibeAtomarOrdnerFehlt(t *testing.T) {
	ziel := filepath.Join(t.TempDir(), "gibtsnicht", "a.png")
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { return nil }); err == nil {
		t.Error("fehlender Ordner soll scheitern")
	}
}

func eintraege(t *testing.T, ordner string) int {
	t.Helper()
	l, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatal(err)
	}
	return len(l)
}
