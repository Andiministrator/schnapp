// SPDX-License-Identifier: GPL-3.0-or-later

package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
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
	if k := eigeneKandidaten(); !slices.Equal(k, []string{p}) {
		t.Errorf("Kandidaten %q: die nicht ausführbare Nachbardatei zählt", k)
	}
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

// setzeFristen verkürzt die Fristen für die Dauer des Tests.
func setzeFristen(t *testing.T, tesseract, ruhe, gesamt, suche time.Duration) {
	t.Helper()
	a, b, c, d := fristTesseract, fristRuhe, fristGesamt, fristSuche
	fristTesseract, fristRuhe, fristGesamt, fristSuche = tesseract, ruhe, gesamt, suche
	t.Cleanup(func() { fristTesseract, fristRuhe, fristGesamt, fristSuche = a, b, c, d })
}

// Liegt schnapp-ocr neben dem Programm und im PATH, gilt das neben dem Programm (Mutant KA3).
func TestSucheNachbarVorPath(t *testing.T) {
	neben, imPath := t.TempDir(), t.TempDir()
	n := eigenesSkript(t, neben, filepath.Join(neben, "protokoll"))
	p := eigenesSkript(t, imPath, filepath.Join(imPath, "protokoll"))
	setzeSuche(t, map[string]string{eigenesName: p}, neben)
	if k := eigeneKandidaten(); !slices.Equal(k, []string{n, p}) {
		t.Errorf("Kandidaten %q, erwartet erst den Nachbarn, dann PATH", k)
	}
	if e, err := Suche(context.Background()); err != nil || e.programm != n {
		t.Errorf("gefunden %v, %v; erwartet %q", e, err, n)
	}
}

// Ein Ordner namens schnapp-ocr neben dem Programm ist kein Kandidat (Mutant KA1).
func TestSucheOrdnerNebenProgramm(t *testing.T) {
	neben := t.TempDir()
	if err := os.Mkdir(filepath.Join(neben, eigenesName), 0o700); err != nil {
		t.Fatal(err)
	}
	setzeSuche(t, nil, neben)
	if k := eigeneKandidaten(); len(k) != 0 {
		t.Errorf("Kandidaten %q", k)
	}
}

// Scheitert schnapp-ocr neben dem Programm, gilt das im PATH (F-647).
func TestSucheNachbarKaputtDannPath(t *testing.T) {
	neben, imPath := t.TempDir(), t.TempDir()
	skript(t, neben, eigenesName, "echo 'schnapp-ocr: kaputt' >&2\nexit 1\n")
	p := eigenesSkript(t, imPath, filepath.Join(imPath, "protokoll"))
	setzeSuche(t, map[string]string{eigenesName: p}, neben)
	if e, err := Suche(context.Background()); err != nil || e.programm != p {
		t.Errorf("gefunden %v, %v; erwartet den Rückfall %q", e, err, p)
	}
	// Ohne PATH-Eintrag kommt der Fehler der Nachbardatei.
	setzeSuche(t, nil, neben)
	if _, err := Suche(context.Background()); err == nil || !strings.Contains(err.Error(), "kaputt") {
		t.Errorf("Fehler %v", err)
	}
}

// Die Nachbardatei zählt nur, wenn sie und ihr Ordner dem eigenen Benutzer oder root gehören und weder Gruppe noch
// andere sie beschreiben dürfen (F-647); ein Verweis zählt mit seinem Ziel (Mutant KA10).
func TestNachbarVertraut(t *testing.T) {
	ordner := t.TempDir()
	p := eigenesSkript(t, ordner, filepath.Join(ordner, "protokoll"))
	alt := besitzer
	t.Cleanup(func() { besitzer = alt })
	fremd := map[string]uint32{} // Pfad → Eigentümer laut Attrappe
	besitzer = func(fi os.FileInfo) (uint32, bool) {
		for pfad, uid := range fremd {
			if filepath.Base(pfad) == fi.Name() {
				return uid, true
			}
		}
		return alt(fi)
	}
	pruefe := func(fall string, soll bool) {
		t.Helper()
		if _, ok := nachbarVertraut(p); ok != soll {
			t.Errorf("%s: vertraut %v, erwartet %v", fall, ok, soll)
		}
	}
	pruefe("eigene Datei, eigener Ordner 0700", true)
	fremd[p] = 4242
	pruefe("Datei eines anderen Benutzers", false)
	fremd[p] = 0
	pruefe("Datei von root", true)
	delete(fremd, p)
	fremd[ordner] = 4242
	pruefe("Ordner eines anderen Benutzers", false)
	delete(fremd, ordner)
	for _, fall := range []struct {
		ziel   string
		rechte os.FileMode
	}{{ordner, 0o777}, {ordner, 0o770}, {ordner, 0o777 | os.ModeSticky}, {p, 0o722}, {p, 0o702}} {
		if err := os.Chmod(fall.ziel, fall.rechte); err != nil {
			t.Fatal(err)
		}
		pruefe(fmt.Sprintf("%s mit %v", filepath.Base(fall.ziel), fall.rechte), false)
		zurueck := os.FileMode(0o700)
		if err := os.Chmod(fall.ziel, zurueck); err != nil {
			t.Fatal(err)
		}
	}
	pruefe("nach dem Zurücksetzen", true)

	// Verweis neben dem Programm auf ein vertrautes Ziel: gilt, geliefert wird das Ziel.
	neben := t.TempDir()
	if err := os.Symlink(p, filepath.Join(neben, eigenesName)); err != nil {
		t.Fatal(err)
	}
	setzeSuche(t, nil, neben)
	if e, err := Suche(context.Background()); err != nil || e.programm != p {
		t.Errorf("Verweis: %v, %v", e, err)
	}
	// Verweis auf eine Datei in einem für alle beschreibbaren Ordner: zählt nicht.
	if err := os.Chmod(ordner, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(ordner, 0o700) })
	if k := eigeneKandidaten(); len(k) != 0 {
		t.Errorf("Verweis in offenen Ordner: Kandidaten %q", k)
	}
}

// Scheitert die Abfrage von tesseract (nicht nur „keine Sprache“), springt schnapp-ocr ein (Mutant KA6).
func TestSucheTesseractFehlerRueckfall(t *testing.T) {
	ordner := t.TempDir()
	tess := skript(t, ordner, programmName, "echo 'kaputt' >&2\nexit 1\n")
	eigenes := eigenesSkript(t, ordner, filepath.Join(ordner, "protokoll"))
	setzeSuche(t, map[string]string{programmName: tess, eigenesName: eigenes}, t.TempDir())
	if e, err := Suche(context.Background()); err != nil || !e.Eigenes() {
		t.Errorf("%v, %v", e, err)
	}
}

// Jede Abfrage der Suche hat ihre eigene Frist: Ein hängendes tesseract verhindert schnapp-ocr nicht (F-650, Mutant KA9);
// hängt es allein, endet die Suche mit ErrSucheFrist.
func TestSucheFristJeProgramm(t *testing.T) {
	setzeFristen(t, fristTesseract, fristRuhe, fristGesamt, 300*time.Millisecond)
	ordner := t.TempDir()
	tess := skript(t, ordner, programmName, "exec sleep 30\n")
	eigenes := eigenesSkript(t, ordner, filepath.Join(ordner, "protokoll"))
	setzeSuche(t, map[string]string{programmName: tess, eigenesName: eigenes}, t.TempDir())
	start := time.Now()
	if e, err := Suche(context.Background()); err != nil || !e.Eigenes() {
		t.Errorf("hängendes tesseract: %v, %v", e, err)
	}
	setzeSuche(t, map[string]string{programmName: tess}, t.TempDir())
	if _, err := Suche(context.Background()); !errors.Is(err, ErrSucheFrist) {
		t.Errorf("nur hängendes tesseract: %v", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("Suche dauerte %v", d)
	}
}

// schnapp-ocr hat eine Frist, die nur ohne Fortschritt abläuft (F-641), nicht die kurze von tesseract (Mutant KA2), und
// eine Obergrenze auch mit Fortschritt.
func TestEigenesRuhefrist(t *testing.T) {
	setzeFristen(t, 100*time.Millisecond, 500*time.Millisecond, 5*time.Second, fristSuche)
	ordner := t.TempDir()
	// 1,2 s Arbeit mit Fortschritt alle 0,1 s: länger als die Frist von tesseract und als die Ruhefrist, aber nie ruhig.
	fleissig := skript(t, ordner, "fleissig", `cat > /dev/null
i=0
while [ $i -lt 12 ]; do echo "progress $((i * 8))" >&2; sleep 0.1; i=$((i + 1)); done
printf 'Fertig\n'
`)
	e := &Erkenner{programm: fleissig, sprachen: []string{"eng"}, eigenes: true}
	var mu sync.Mutex
	var n int
	text, err := e.Erkenne(context.Background(), testbild(10, 10), "eng", func(int) { mu.Lock(); n++; mu.Unlock() })
	if err != nil || text != "Fertig" {
		t.Errorf("mit Fortschritt: %q, %v", text, err)
	}
	if mu.Lock(); n != 12 {
		t.Errorf("%d Meldungen weitergegeben, erwartet 12", n)
	}
	mu.Unlock()
	// Ohne Fortschritt (und ohne Meldefunktion des Aufrufers) endet die Ruhefrist.
	still := skript(t, ordner, "still", "cat > /dev/null\necho 'progress 1' >&2\nexec sleep 30\n")
	e.programm = still
	start := time.Now()
	if _, err := e.Erkenne(context.Background(), testbild(10, 10), "eng", nil); !errors.Is(err, ErrFrist) {
		t.Errorf("ohne Fortschritt: %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Ruhefrist griff erst nach %v", d)
	}
	// Die Obergrenze gilt auch mit Fortschritt.
	setzeFristen(t, 100*time.Millisecond, 500*time.Millisecond, 600*time.Millisecond, fristSuche)
	e.programm = fleissig
	if _, err := e.Erkenne(context.Background(), testbild(10, 10), "eng", nil); !errors.Is(err, ErrFrist) {
		t.Errorf("über der Obergrenze: %v", err)
	}
}

// tesseract bekommt keinen Fortschrittsleser: Seine Zeilen „progress N“ sind gewöhnliche Fehlerausgabe (Mutant KA4).
func TestTesseractOhneFortschritt(t *testing.T) {
	ordner := t.TempDir()
	p := skript(t, ordner, programmName, "cat > /dev/null\necho 'progress 50' >&2\nexit 1\n")
	e := &Erkenner{programm: p, sprachen: []string{"eng"}}
	gemeldet := false
	_, err := e.Erkenne(context.Background(), testbild(10, 10), "eng", func(int) { gemeldet = true })
	if gemeldet {
		t.Error("Fortschritt von tesseract weitergegeben")
	}
	if err == nil || !strings.Contains(err.Error(), "progress 50") {
		t.Errorf("Fehler %v, erwartet die Zeile in der Meldung", err)
	}
}

// Eine überlange Zeile der Fehlerausgabe wird nur bis maxFehlerausgabe gesammelt (Mutant KA5).
func TestFortschrittLeserZeilengrenze(t *testing.T) {
	var rest bytes.Buffer
	f := &fortschrittLeser{ziel: &rest, melde: func(int) {}}
	lang := bytes.Repeat([]byte("x"), 3*maxFehlerausgabe)
	if _, err := f.Write(lang); err != nil {
		t.Fatal(err)
	}
	if len(f.zeile) > maxFehlerausgabe || cap(f.zeile) > 2*maxFehlerausgabe {
		t.Errorf("Zeile %d Byte gesammelt (Kapazität %d), Grenze %d", len(f.zeile), cap(f.zeile), maxFehlerausgabe)
	}
	f.Write([]byte("\n"))
	if rest.Len() != maxFehlerausgabe+1 {
		t.Errorf("Rest %d Byte", rest.Len())
	}
}
