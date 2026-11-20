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
	"syscall"
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
		f, p, ang := Waehle(c.pfad, c.filter)
		if f != c.f || p != c.soll {
			t.Errorf("Waehle(%q, %v) = %v, %q; soll %v, %q", c.pfad, c.filter, f, p, c.f, c.soll)
		}
		if ang != (p != c.pfad) {
			t.Errorf("Waehle(%q): angehängt %v, Pfad %q", c.pfad, ang, p)
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
	for n, marke := range map[string]string{"APP1/EXIF": "\xff\xe1", "COM": "\xff\xfe"} {
		if bytes.Contains(j.Bytes(), []byte(marke)) {
			t.Errorf("JPG enthält %s", n)
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
	alt := syscall.Umask(0o027)
	defer syscall.Umask(alt)
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "neu.png")
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { _, err := w.Write([]byte("inhalt")); return err }); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ziel)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o640 {
		t.Errorf("Rechte %v, soll 0640 (0666 ohne die umask 027)", st.Mode().Perm())
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

// Eine ersetzte Datei behält ihre Rechte (0600 wird nicht auf 0644 geöffnet), unabhängig von der umask (A-3).
func TestSchreibeAtomarBehaeltRechteDerErsetzten(t *testing.T) {
	alt := syscall.Umask(0o022)
	defer syscall.Umask(alt)
	ziel := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ziel, 0o600); err != nil { // WriteFile unterliegt der umask nur nach unten; sicher ist sicher
		t.Fatal(err)
	}
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { _, err := w.Write([]byte("neu")); return err }); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ziel)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("Rechte %v, soll 0600", st.Mode().Perm())
	}
}

// Eine neue Datei folgt der umask, nicht einem festen Wert (A-3).
func TestSchreibeAtomarNeueDateiFolgtUmask(t *testing.T) {
	alt := syscall.Umask(0o077)
	defer syscall.Umask(alt)
	ziel := filepath.Join(t.TempDir(), "neu.png")
	if err := SchreibeAtomar(ziel, func(w io.Writer) error { return nil }); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ziel)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("Rechte %v, soll 0600 (umask 077)", st.Mode().Perm())
	}
}

// Hängt Waehle die Endung an und das Ziel gibt es schon, wird nichts geschrieben (A-1).
func TestPruefeAngehaengt(t *testing.T) {
	ordner := t.TempDir()
	vorhanden := filepath.Join(ordner, "bericht.png")
	if err := os.WriteFile(vorhanden, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, ziel, ang := Waehle(filepath.Join(ordner, "bericht"), PNG)
	if !ang || ziel != vorhanden {
		t.Fatalf("Waehle: %q %v", ziel, ang)
	}
	err := PruefeAngehaengt(ziel, ang)
	if !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("Fehler %v, soll ErrZielVorhanden", err)
	}
	if b, _ := os.ReadFile(vorhanden); string(b) != "alt" {
		t.Errorf("Datei verändert: %q", b)
	}
	// Der Dialog hat den vollen Namen bestätigt: Ersetzen ist gewollt.
	if err := PruefeAngehaengt(vorhanden, false); err != nil {
		t.Errorf("nicht angehängt: %v", err)
	}
	// Ziel gibt es nicht: frei.
	if err := PruefeAngehaengt(filepath.Join(ordner, "neu.png"), true); err != nil {
		t.Errorf("neues Ziel: %v", err)
	}
}

// TestRechteVorDemSchreiben: Ersetzt der Export eine 0600-Datei, ist schon die temporäre Datei während des Schreibens
// nicht offener als das Ziel (Verify N-2).
func TestRechteVorDemSchreiben(t *testing.T) {
	alt := syscall.Umask(0o022)
	defer syscall.Umask(alt)
	ziel := filepath.Join(t.TempDir(), "x.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	var waehrend os.FileMode
	err := SchreibeAtomar(ziel, func(w io.Writer) error {
		st, err := w.(*os.File).Stat()
		if err != nil {
			return err
		}
		waehrend = st.Mode().Perm()
		_, err = io.WriteString(w, "neu")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if waehrend != 0o600 {
		t.Errorf("temporäre Datei während des Schreibens %o, soll 600", waehrend)
	}
}

// TestSymlinkZiel: Ein symbolischer Link als Ziel bleibt ein Link; geschrieben wird in die Datei, auf die er zeigt,
// mit deren Rechten (Verify N-3). Ein Link ins Leere wird abgelehnt.
func TestSymlinkZiel(t *testing.T) {
	dir := t.TempDir()
	echt := filepath.Join(dir, "echt.png")
	link := filepath.Join(dir, "link.png")
	if err := os.WriteFile(echt, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("echt.png", link); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeAtomar(link, func(w io.Writer) error { _, err := io.WriteString(w, "neu"); return err }); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Lstat(link); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Link ersetzt: %v %v", st, err)
	}
	if b, _ := os.ReadFile(echt); string(b) != "neu" {
		t.Errorf("Ziel des Links hat %q", b)
	}
	if st, _ := os.Stat(echt); st.Mode().Perm() != 0o600 {
		t.Errorf("Rechte %o, soll 600", st.Mode().Perm())
	}
	leer := filepath.Join(dir, "leer.png")
	if err := os.Symlink("gibtsnicht/x.png", leer); err != nil {
		t.Fatal(err)
	}
	if err := SchreibeAtomar(leer, func(w io.Writer) error { return nil }); err == nil {
		t.Error("Link ins Leere angenommen")
	}
}

// TestOhneErsetzen: Entsteht das Ziel während des Schreibens, legt SchreibeOhneErsetzen nichts an und meldet
// ErrZielVorhanden; die fremde Datei bleibt, keine temporäre Datei bleibt liegen (Verify N-4). Ohne Ziel legt es an.
func TestOhneErsetzen(t *testing.T) {
	dir := t.TempDir()
	ziel := filepath.Join(dir, "a.png")
	err := SchreibeOhneErsetzen(ziel, func(w io.Writer) error {
		if err := os.WriteFile(ziel, []byte("fremd"), 0o644); err != nil {
			return err
		}
		_, err := io.WriteString(w, "meins")
		return err
	})
	if !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("Fehler %v, soll ErrZielVorhanden", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "fremd" {
		t.Errorf("fremde Datei überschrieben: %q", b)
	}
	if e, _ := os.ReadDir(dir); len(e) != 1 {
		t.Errorf("%d Einträge im Ordner, soll 1 (temporäre Datei liegen geblieben?)", len(e))
	}
	neu := filepath.Join(dir, "b.png")
	if err := SchreibeOhneErsetzen(neu, func(w io.Writer) error { _, err := io.WriteString(w, "x"); return err }); err != nil {
		t.Fatal(err)
	}
	if e, _ := os.ReadDir(dir); len(e) != 2 {
		t.Errorf("%d Einträge, soll 2", len(e))
	}
}
