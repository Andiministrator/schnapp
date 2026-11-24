// SPDX-License-Identifier: GPL-3.0-or-later

package ocr

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// skript legt ein ausführbares Shell-Skript name in ordner an.
func skript(t *testing.T, ordner, name, inhalt string) string {
	t.Helper()
	p := filepath.Join(ordner, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+inhalt), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

// eigenesSkript spielt schnapp-ocr: zwei Sprachen, Fortschritt auf stderr, Text auf stdout; Argumente ins Protokoll.
func eigenesSkript(t *testing.T, ordner, protokoll string) string {
	return skript(t, ordner, eigenesName, `if [ "$1" = "--list-langs" ]; then
  echo 'List of available languages in "schnapp-ocr" (2):'
  printf 'deu\neng\n'
  exit 0
fi
echo "$@" > "`+protokoll+`"
cat > /dev/null
printf 'progress 0\nprogress 4' >&2
printf '2\nprogress 100\nfremde Zeile\n' >&2
printf 'Hallo\n'
`)
}

// setzeSuche ersetzt PATH-Suche und eigenen Ordner für die Dauer des Tests: pfad ordnet Programmnamen einen Pfad zu.
func setzeSuche(t *testing.T, pfad map[string]string, ordner string) {
	t.Helper()
	altP, altO := suchePfad, eigenerOrdner
	suchePfad = func(n string) (string, error) {
		if p, ok := pfad[n]; ok {
			return p, nil
		}
		return "", exec.ErrNotFound
	}
	eigenerOrdner = func() (string, error) { return ordner, nil }
	t.Cleanup(func() { suchePfad, eigenerOrdner = altP, altO })
}

// Ohne tesseract findet Suche schnapp-ocr neben dem eigenen Programm; eine Sprache je Aufruf, Fortschritt kommt an.
func TestSucheEigenesNebenProgramm(t *testing.T) {
	ordner := t.TempDir()
	protokoll := filepath.Join(ordner, "protokoll")
	p := eigenesSkript(t, ordner, protokoll)
	setzeSuche(t, nil, ordner)
	e, err := Suche(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eigenes() || e.programm != p {
		t.Fatalf("gefunden %q, eigenes %v", e.programm, e.Eigenes())
	}
	for ober, soll := range map[string]string{"de": "deu", "en": "eng", "fr": "eng", "": "eng"} {
		if got := e.Sprachen(ober); got != soll {
			t.Errorf("Sprachen(%q) = %q, erwartet %q", ober, got, soll)
		}
	}
	var mu sync.Mutex
	var stufen []int
	text, err := e.Erkenne(context.Background(), testbild(20, 20), "deu", func(n int) {
		mu.Lock()
		stufen = append(stufen, n)
		mu.Unlock()
	})
	if err != nil || text != "Hallo" {
		t.Fatalf("Text %q, %v", text, err)
	}
	if !slices.Equal(stufen, []int{0, 42, 100}) {
		t.Errorf("Fortschritt %v", stufen)
	}
	a, _ := os.ReadFile(protokoll)
	if got := strings.TrimSpace(string(a)); got != "- - -l deu --psm 3 --progress" {
		t.Errorf("Argumente %q", got)
	}
	if _, err := e.Erkenne(context.Background(), testbild(20, 20), "deu+eng", nil); err == nil {
		t.Error("zwei Sprachen an schnapp-ocr angenommen")
	}
}

// schnapp-ocr im PATH gilt, wenn es neben dem Programm fehlt; ein nicht ausführbares neben dem Programm zählt nicht.
func TestSucheEigenesImPath(t *testing.T) {
	ordner, neben := t.TempDir(), t.TempDir()
	p := eigenesSkript(t, ordner, filepath.Join(ordner, "protokoll"))
	if err := os.WriteFile(filepath.Join(neben, eigenesName), []byte("kein Programm"), 0o600); err != nil {
		t.Fatal(err)
	}
	setzeSuche(t, map[string]string{eigenesName: p}, neben)
	e, err := Suche(context.Background())
	if err != nil || e.programm != p || !e.Eigenes() {
		t.Fatalf("%v, %v", e, err)
	}
}

// tesseract hat Vorrang; hat es keine Sprache, übernimmt schnapp-ocr; fehlt das, bleibt der Grund von tesseract.
func TestSucheVorrangUndRueckfall(t *testing.T) {
	ordner := t.TempDir()
	tess := skript(t, ordner, programmName, `echo 'List of available languages in "/x/" (2):'
printf 'osd\n'
`)
	eigenes := eigenesSkript(t, ordner, filepath.Join(ordner, "protokoll"))
	setzeSuche(t, map[string]string{programmName: tess, eigenesName: eigenes}, t.TempDir())
	e, err := Suche(context.Background())
	if err != nil || !e.Eigenes() {
		t.Fatalf("tesseract ohne Sprache, schnapp-ocr da: %v, %v", e, err)
	}
	setzeSuche(t, map[string]string{programmName: tess}, t.TempDir())
	if _, err := Suche(context.Background()); !errors.Is(err, ErrKeineSprache) || !strings.HasPrefix(err.Error(), "ocr: ") {
		t.Errorf("nur tesseract ohne Sprache: %v", err)
	}
	gut := skript(t, t.TempDir(), programmName, `printf 'List (1):\ndeu\n'
`)
	setzeSuche(t, map[string]string{programmName: gut, eigenesName: eigenes}, t.TempDir())
	if e, err := Suche(context.Background()); err != nil || e.Eigenes() {
		t.Errorf("tesseract mit Sprache muss Vorrang haben: %v, %v", e, err)
	}
	setzeSuche(t, nil, t.TempDir())
	if _, err := Suche(context.Background()); !errors.Is(err, ErrKeinProgramm) {
		t.Errorf("keines da: %v", err)
	}
}

// Ein scheiterndes schnapp-ocr meldet die letzte fremde Zeile (ohne Zeilenende), nicht die Fortschrittszeilen.
func TestEigenesFehlerMeldung(t *testing.T) {
	ordner := t.TempDir()
	p := skript(t, ordner, eigenesName, `cat > /dev/null
printf 'progress 10\nschnapp-ocr: load image: kaputt' >&2
exit 1
`)
	e := &Erkenner{programm: p, sprachen: []string{"eng"}, eigenes: true}
	_, err := e.Erkenne(context.Background(), testbild(20, 20), "eng", func(int) {})
	if err == nil || !strings.Contains(err.Error(), "schnapp-ocr failed") || !strings.Contains(err.Error(), "kaputt") ||
		strings.Contains(err.Error(), "progress") {
		t.Errorf("Fehler %v", err)
	}
}

func TestFortschrittLeser(t *testing.T) {
	var rest bytes.Buffer
	var stufen []int
	f := &fortschrittLeser{ziel: &rest, melde: func(n int) { stufen = append(stufen, n) }}
	for _, teil := range []string{"progr", "ess 7\nprogress 101\nprogress -1\nprogress x\nEnde\npro", "gress 100\n"} {
		if n, err := f.Write([]byte(teil)); err != nil || n != len(teil) {
			t.Fatal(n, err)
		}
	}
	if !slices.Equal(stufen, []int{7, 100}) {
		t.Errorf("Stufen %v", stufen)
	}
	if rest.String() != "progress 101\nprogress -1\nprogress x\nEnde\n" {
		t.Errorf("Rest %q", rest.String())
	}
}

// TestErkenneEigenesEcht fährt das gebaute schnapp-ocr (Pfad in SCHNAPP_OCR_PROGRAMM, sonst übersprungen; die CI baut
// es im Job ocr): Suche ohne tesseract, eine Sprache, Fortschritt bis 100, Text erkannt.
func TestErkenneEigenesEcht(t *testing.T) {
	p := os.Getenv("SCHNAPP_OCR_PROGRAMM")
	if p == "" {
		t.Skip("SCHNAPP_OCR_PROGRAMM nicht gesetzt")
	}
	setzeSuche(t, map[string]string{eigenesName: p}, t.TempDir())
	e, err := Suche(context.Background())
	if err != nil || !e.Eigenes() {
		t.Fatalf("%v, %v", e, err)
	}
	var mu sync.Mutex
	letzter := -1
	text, err := e.Erkenne(context.Background(), schriftBild(t, "Schnapp 2026 Text"), e.Sprachen("de"), func(n int) {
		mu.Lock()
		letzter = n
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Schnapp") || !strings.Contains(text, "2026") {
		t.Errorf("erkannt %q", text)
	}
	if letzter != 100 {
		t.Errorf("letzter Fortschritt %d", letzter)
	}
}
