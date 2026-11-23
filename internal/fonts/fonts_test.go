// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/sfnt"
)

// mitOrdnern sucht in den Testordnern und stellt danach den leeren Stand her.
func mitOrdnern(t *testing.T, o ...string) {
	t.Helper()
	alt := ordner
	ordner = func() []string { return o }
	Suche()
	t.Cleanup(func() {
		ordner = alt
		mu.Lock()
		system, gesucht = nil, false
		clear(geladen)
		clear(fehler)
		mu.Unlock()
	})
}

func schreibe(t *testing.T, pfad string, roh []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, roh, 0o600); err != nil {
		t.Fatal(err)
	}
}

func familieName(t *testing.T, roh []byte) (string, string) {
	t.Helper()
	f, err := sfnt.Parse(roh)
	if err != nil {
		t.Fatal(err)
	}
	var b sfnt.Buffer
	n, _ := f.Name(&b, sfnt.NameIDFamily)
	u, _ := f.Name(&b, sfnt.NameIDSubfamily)
	return n, u
}

func TestEingebautUndErsatz(t *testing.T) {
	mitOrdnern(t)
	if f := Familien(); len(f) != 2 || f[0] != Go || f[1] != GoMono {
		t.Fatalf("Familien %v", f)
	}
	s := Laden("", FettKursiv)
	if s.Familie != Go || s.Schnitt != FettKursiv || s.Ersatz {
		t.Errorf("Go fett kursiv: %+v", s)
	}
	if s2 := Laden(Go, FettKursiv); s2 != s {
		t.Error("kein Zwischenspeicher")
	}
	m := Laden("Fehlt Ganz", Kursiv)
	if m.Familie != Go || m.Schnitt != Kursiv || !m.Ersatz || Bekannt("Fehlt Ganz") || !Bekannt("") {
		t.Errorf("fehlende Familie: %+v", m)
	}
}

func TestSucheFindetFamilienUndSchnitte(t *testing.T) {
	d := t.TempDir()
	medium, unter := familieName(t, gomedium.TTF)
	_, unterKursiv := familieName(t, gomediumitalic.TTF)
	t.Logf("Go Medium: Familie %q, Schnitt %q / %q", medium, unter, unterKursiv)
	schreibe(t, filepath.Join(d, "a", "GoMedium.TTF"), gomedium.TTF) // Endung in Großbuchstaben
	schreibe(t, filepath.Join(d, "a", "b", "GoMediumItalic.otf"), gomediumitalic.TTF)
	schreibe(t, filepath.Join(d, "GoBold.ttf"), gobold.TTF) // Familie Go: eingebaut geht vor
	schreibe(t, filepath.Join(d, "kaputt.ttf"), []byte("keine Schrift"))
	schreibe(t, filepath.Join(d, "liesmich.txt"), gosmallcaps.TTF) // falsche Endung
	if err := os.Symlink(filepath.Join(d, "a"), filepath.Join(d, "verweis")); err != nil {
		t.Fatal(err)
	}
	mitOrdnern(t, d, filepath.Join(d, "gibt-es-nicht"))
	if !SucheFertig() {
		t.Fatal("Suche nicht fertig")
	}
	f := Familien()
	if len(f) != 3 || f[2] != medium {
		t.Fatalf("Familien %v, erwartet Go, Go Mono, %s", f, medium)
	}
	n := Laden(medium, Normal)
	k := Laden(medium, Kursiv)
	fk := Laden(medium, FettKursiv) // fehlt: kursiv
	if n.Ersatz || n.Familie != medium || k.Schnitt != Kursiv || fk.Schnitt != Kursiv || fk.Ersatz {
		t.Errorf("normal %+v, kursiv %+v, fett kursiv %+v", n, k, fk)
	}
	if fe := Laden(medium, Fett); fe.Schnitt != Normal {
		t.Errorf("fett fehlt: %+v", fe)
	}
}

// TestSucheGrenzeDateien: Nach maxDateien Schriftdateien hört die Suche auf (lexikalische Reihenfolge von WalkDir).
func TestSucheGrenzeDateien(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "a.ttf"), gomedium.TTF)
	schreibe(t, filepath.Join(d, "b.ttf"), gosmallcaps.TTF)
	alt := maxDateien
	maxDateien = 1
	t.Cleanup(func() { maxDateien = alt })
	mitOrdnern(t, d)
	if f := Familien(); len(f) != 3 || f[2] != "Go Medium" {
		t.Errorf("Familien %v", f)
	}
}

func TestSchnittAusName(t *testing.T) {
	for name, want := range map[string]Schnitt{"": Normal, "Regular": Normal, "Book": Normal, "Bold": Fett,
		"Italic": Kursiv, "Oblique": Kursiv, "Bold Italic": FettKursiv, "bold oblique": FettKursiv} {
		if s, ok := SchnittAusName(name); !ok || s != want {
			t.Errorf("%q: %v %v", name, s, ok)
		}
	}
	for _, name := range []string{"Light", "SemiBold", "Medium Italic", "Condensed Bold", "Black"} {
		if _, ok := SchnittAusName(name); ok {
			t.Errorf("%q gilt als Grundschnitt", name)
		}
	}
}

func TestNameGueltig(t *testing.T) {
	lang := string(slices.Repeat([]rune{'x'}, MaxNamenLaenge+1))
	for _, n := range []string{"", "a\nb", "a\x00", lang} {
		if NameGueltig(n) {
			t.Errorf("%q gilt", n)
		}
	}
	if !NameGueltig("Noto Sans") || !NameGueltig("Ünïcødé") {
		t.Error("gültiger Name abgelehnt")
	}
}

func TestSchriftOrdner(t *testing.T) {
	t.Setenv("HOME", "/heim")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "")
	if o := schriftOrdner(); !slices.Equal(o, []string{"/heim/.local/share/fonts", "/heim/.fonts",
		"/usr/local/share/fonts", "/usr/share/fonts"}) {
		t.Errorf("Voreinstellung %v", o)
	}
	t.Setenv("XDG_DATA_HOME", "/daten")
	t.Setenv("XDG_DATA_DIRS", "/a:relativ:/a:/b")
	if o := schriftOrdner(); !slices.Equal(o, []string{"/daten/fonts", "/heim/.fonts", "/a/fonts", "/b/fonts"}) {
		t.Errorf("gesetzt %v", o)
	}
}

// TestLadenSuchtSelbst: Ohne vorherige Suche (Export ohne Editor) sucht Laden einmal selbst und findet die Familie.
func TestLadenSuchtSelbst(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "m.ttf"), gomedium.TTF)
	alt := ordner
	ordner = func() []string { return []string{d} }
	altSuche := suche
	suche = new(sync.Once)
	t.Cleanup(func() {
		ordner, suche = alt, altSuche
		mu.Lock()
		system, gesucht = nil, false
		clear(geladen)
		clear(fehler)
		mu.Unlock()
	})
	mu.Lock()
	system, gesucht = nil, false
	mu.Unlock()
	if s := Laden("Go Medium", Normal); s.Ersatz || s.Familie != "Go Medium" || !SucheFertig() {
		t.Errorf("Laden ohne Suche: %+v, fertig %v", s, SucheFertig())
	}
}
