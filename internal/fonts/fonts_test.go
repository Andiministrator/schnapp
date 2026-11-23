// SPDX-License-Identifier: GPL-3.0-or-later

package fonts

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

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
		clear(dateien)
		dateiBytes = 0
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
	for _, n := range []string{"", "a\nb", "a\x00", lang, "   ", "a\u0085b", "a\u009fb", "a\x7fb", "a\ufffdb"} {
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
	altSuche, altKanal := suche, fertigKanal
	suche, fertigKanal = new(sync.Once), make(chan struct{})
	t.Cleanup(func() {
		ordner, suche, fertigKanal = alt, altSuche, altKanal
		clear(dateien)
		dateiBytes = 0
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

// TestHintergrundsucheUndWarten (Sitzungs-QA B-5): Läuft die Suche im Hintergrund, wartet Laden auf ihr Ende statt
// sofort Go zu liefern; der Rückruf kommt danach.
func TestHintergrundsucheUndWarten(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "m.ttf"), gomedium.TTF)
	frei := make(chan struct{})
	alt, altSuche, altKanal := ordner, suche, fertigKanal
	ordner = func() []string { <-frei; return []string{d} }
	suche, fertigKanal = new(sync.Once), make(chan struct{})
	t.Cleanup(func() {
		ordner, suche, fertigKanal = alt, altSuche, altKanal
		mu.Lock()
		system, gesucht = nil, false
		clear(geladen)
		clear(fehler)
		clear(dateien)
		dateiBytes = 0
		mu.Unlock()
	})
	mu.Lock()
	system, gesucht = nil, false
	mu.Unlock()
	gerufen := make(chan struct{})
	StarteSuche(func() { close(gerufen) })
	ergebnis := make(chan *Schrift)
	go func() { ergebnis <- Laden("Go Medium", Normal) }()
	select {
	case s := <-ergebnis:
		t.Fatalf("Laden kam vor dem Ende der Suche zurück: %+v", s)
	case <-time.After(50 * time.Millisecond):
	}
	close(frei)
	if s := <-ergebnis; s.Ersatz || s.Familie != "Go Medium" {
		t.Errorf("nach der Suche: %+v", s)
	}
	<-gerufen
}

// TestGrenzeGeladen (Sitzungs-QA B-2): Über der Summe geladener Bytes kommt Go als Ersatz; eingebaute Schriften
// zählen nicht mit.
func TestGrenzeGeladen(t *testing.T) {
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "m.ttf"), gomedium.TTF)
	mitOrdnern(t, d)
	mu.Lock()
	dateiBytes = MaxGeladenBytes
	mu.Unlock()
	if s := Laden("Go Medium", Normal); !s.Ersatz {
		t.Errorf("über der Grenze: %+v", s)
	}
	if s := Laden(GoMono, Fett); s.Ersatz {
		t.Errorf("eingebaute Schrift: %+v", s)
	}
	// Ebenso über der Zahl der Dateien (eigene Prüfung vor dem Lesen).
	mu.Lock()
	dateiBytes = 0
	clear(geladen)
	clear(fehler)
	for i := range MaxGeladen {
		dateien[filepath.Join(d, "andere", string(rune('a'+i%26))+string(rune('0'+i/26)))] = nil
	}
	mu.Unlock()
	if s := Laden("Go Medium", Kursiv); !s.Ersatz {
		t.Errorf("über der Dateizahl: %+v", s)
	}
	// Die Grenze wird nicht als Fehler der Datei gemerkt: Ist wieder Platz, lädt dieselbe Familie (Verify V-2).
	mu.Lock()
	clear(dateien)
	mu.Unlock()
	if s := Laden("Go Medium", Kursiv); s.Ersatz {
		t.Errorf("nach Freigabe: %+v", s)
	}
}

// TestLadenFrist (Verify V-4): Hängt die Suche, kehrt Laden nach der Frist mit Go zurück, statt das Fenster
// einzufrieren.
func TestLadenFrist(t *testing.T) {
	frei := make(chan struct{})
	alt, altSuche, altKanal, altFrist := ordner, suche, fertigKanal, sucheFrist
	k := make(chan struct{})
	mu.Lock()
	ordner = func() []string { <-frei; return nil }
	suche, fertigKanal, sucheFrist = new(sync.Once), k, 50*time.Millisecond
	mu.Unlock()
	t.Cleanup(func() {
		close(frei)
		<-k // die Suche ist zu Ende, erst dann zurückstellen
		mu.Lock()
		ordner, suche, fertigKanal, sucheFrist = alt, altSuche, altKanal, altFrist
		mu.Unlock()
		mu.Lock()
		system, gesucht = nil, false
		mu.Unlock()
	})
	mu.Lock()
	system, gesucht = nil, false
	mu.Unlock()
	start := time.Now()
	if s := Laden("Irgendeine Familie", Normal); !s.Ersatz {
		t.Errorf("hängende Suche: %+v", s)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("Laden wartete %v", d)
	}
}

// TestSetzeOrdner (Sitzungs-QA B-3): Die festgelegten Ordner gelten statt HOME und XDG_*.
func TestSetzeOrdner(t *testing.T) {
	alt := ordner
	t.Cleanup(func() { ordner = alt })
	d := t.TempDir()
	schreibe(t, filepath.Join(d, "m.ttf"), gomedium.TTF)
	t.Setenv("XDG_DATA_DIRS", "/gibt/es/nicht")
	SetzeOrdner([]string{d})
	Suche()
	t.Cleanup(func() {
		mu.Lock()
		system, gesucht = nil, false
		mu.Unlock()
	})
	if f := Familien(); len(f) != 3 || f[2] != "Go Medium" {
		t.Errorf("Familien %v", f)
	}
}
