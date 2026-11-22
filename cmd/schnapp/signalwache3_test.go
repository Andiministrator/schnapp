// SPDX-License-Identifier: GPL-3.0-or-later

package main

// F-458: zweites Signal im Rückfall und ohne Temp-Datei meldet den wahren Stand.
// F-459: Präfixprüfung in entferneTemp, Ergebniszeile bei --output nach dem Schreiben, Meldung bei gestörter Rettung.

import (
	"bytes"
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/rescue"
)

// ersetzeBeenden fängt beendeProzess ab und liefert den Kanal, der den Exit-Code meldet.
func ersetzeBeenden(t *testing.T) chan int {
	t.Helper()
	gemeldet := make(chan int, 1)
	alt := beendeProzess
	beendeProzess = func(c int) { gemeldet <- c }
	t.Cleanup(func() { beendeProzess = alt })
	return gemeldet
}

// stderrUmleiten leitet os.Stderr in eine Datei um (die Wache schreibt dorthin) und liefert eine Funktion, die den
// bisherigen Inhalt liest.
func stderrUmleiten(t *testing.T) func() string {
	t.Helper()
	datei, err := os.Create(filepath.Join(t.TempDir(), "stderr.txt"))
	if err != nil {
		t.Fatal(err)
	}
	alt := os.Stderr
	os.Stderr = datei
	t.Cleanup(func() { os.Stderr = alt; datei.Close() })
	return func() string {
		b, _ := os.ReadFile(datei.Name())
		return string(b)
	}
}

// zweiSignale schickt zwei SIGTERM an den eigenen Prozess (die Wache fängt sie) und wartet auf das Beenden.
func zweiSignale(t *testing.T, gemeldet chan int) {
	t.Helper()
	for i := 0; i < 2; i++ {
		if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	select {
	case c := <-gemeldet:
		if c != exitFehler {
			t.Errorf("Exit-Code %d, Soll %d", c, exitFehler)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("das zweite Signal beendete den Prozess nicht")
	}
}

// F-458: Im Rückfall (Ordner ohne Schreibrecht, vorhandenes Ziel beschreibbar) gibt es keine temporäre Datei; hängt
// das Schreiben, meldet das zweite Signal, dass das Ziel unvollständig sein kann, und sagt nichts von einer entfernten
// temporären Datei. Das Ziel fasst die Wache nicht an.
func TestSignalwacheZweitesSignalImRueckfall(t *testing.T) {
	fangbar(t, syscall.SIGTERM)
	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if f, err := os.CreateTemp(dir, "probe"); err == nil { // läuft als Administrator: Rechte gelten nicht
		f.Close()
		t.Skip("Ordnerrechte wirken nicht")
	}
	drin := make(chan struct{})
	frei := make(chan struct{})
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		close(drin)
		<-frei
		return errors.New("test: Schreiben hing")
	})
	gemeldet := ersetzeBeenden(t)
	lies := stderrUmleiten(t)
	fertig := make(chan error, 1)
	go func() { fertig <- schreibeInhalt(ziel, []byte("neu"), true, nil) }()
	<-drin
	zweiSignale(t, gemeldet)
	s := lies()
	if !strings.Contains(s, "directly to the target") || !strings.Contains(s, "incomplete") || strings.Contains(s, "temporary file was removed") {
		t.Errorf("Meldung %q", s)
	}
	close(frei)
	<-fertig
	if b, err := os.ReadFile(ziel); err != nil || string(b) != "alt" {
		t.Errorf("Ziel %q, %v", b, err)
	}
}

// F-458: Hängt das Schreiben, bevor es begann (kein Name einer temporären Datei bekannt), sagt das zweite Signal, dass
// das Ziel nicht verändert wurde.
func TestSignalwacheZweitesSignalVorDemSchreiben(t *testing.T) {
	fangbar(t, syscall.SIGTERM)
	gemeldet := ersetzeBeenden(t)
	lies := stderrUmleiten(t)
	drin := make(chan struct{})
	frei := make(chan struct{})
	fertig := make(chan error, 1)
	go func() {
		fertig <- mitSignalwache(func(abgebrochen func() bool, merkeTemp func(string)) error {
			close(drin)
			<-frei
			return errAbgebrochen
		})
	}()
	<-drin
	zweiSignale(t, gemeldet)
	if s := lies(); !strings.Contains(s, "before writing started") || strings.Contains(s, "removed") {
		t.Errorf("Meldung %q", s)
	}
	close(frei)
	<-fertig
}

// F-458: Die Meldung zum zweiten Signal richtet sich nach dem Stand der temporären Datei.
func TestZweitesSignalMeldungNachStand(t *testing.T) {
	dir := t.TempDir()
	vorhanden := filepath.Join(dir, tempVorsatz+"a")
	if err := os.WriteFile(vorhanden, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fehlt := filepath.Join(dir, tempVorsatz+"b") // schon umbenannt
	haertefall := filepath.Join(dir, tempVorsatz+"c")
	if err := os.Mkdir(haertefall, 0o700); err != nil { // Remove scheitert: Verzeichnis nicht leer
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(haertefall, "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fremd := filepath.Join(dir, "fremd.png")
	if err := os.WriteFile(fremd, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		direkt bool
		temp   *string
		soll   string
	}{
		{true, nil, "directly to the target"},
		{false, nil, "before writing started"},
		{false, &vorhanden, "temporary file was removed"},
		{false, &fehlt, "already complete"},
		{false, &haertefall, "could not be removed"},
		{false, &fremd, "could not be removed"},
	} {
		if s := zweitesSignalMeldung(tc.direkt, tc.temp); !strings.Contains(s, tc.soll) {
			t.Errorf("direkt %v, temp %v: Meldung %q, Soll %q", tc.direkt, tc.temp, s, tc.soll)
		}
	}
	if _, err := os.Stat(vorhanden); !os.IsNotExist(err) {
		t.Errorf("temporäre Datei blieb: %v", err)
	}
	if _, err := os.Stat(fremd); err != nil {
		t.Errorf("fremde Datei angefasst: %v", err)
	}
}

// F-459: entferneTemp löscht nur Dateien mit dem Vorsatz der Exportdatei, nie eine andere (im Rückfall heißt der
// gemerkte Name das Ziel selbst).
func TestEntferneTempNurMitVorsatz(t *testing.T) {
	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	if err := os.WriteFile(ziel, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := entferneTemp(ziel); !errors.Is(err, errKeinTemp) {
		t.Errorf("Fehler %v, Soll errKeinTemp", err)
	}
	if _, err := os.Stat(ziel); err != nil {
		t.Errorf("Datei ohne Vorsatz entfernt: %v", err)
	}
	temp := filepath.Join(dir, tempVorsatz+"123")
	if err := os.WriteFile(temp, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := entferneTemp(temp); err != nil {
		t.Errorf("Fehler %v", err)
	}
	if _, err := os.Stat(temp); !os.IsNotExist(err) {
		t.Errorf("Datei mit Vorsatz blieb: %v", err)
	}
}

// F-441 und F-459: Bei --output mit Signal nach dem Schreiben meldet Schnapp „vollständig geschrieben“, gibt die
// Ergebniszeile aus und endet mit Exit 1; die Datei liegt vollständig vor.
func TestOutputSignalNachSchreibenGibtErgebniszeile(t *testing.T) {
	fangbar(t, syscall.SIGHUP)
	dir := t.TempDir()
	ziel := filepath.Join(dir, "aus.png")
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		if _, err := w.Write(inhalt); err != nil {
			return err
		}
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Error(err)
		}
		wartePunkt(abgebrochen)
		return nil
	})
	var aus, fehler bytes.Buffer
	b := testbild(20, 10, 3)
	code, _ := anschluesse{}.speichere(b, image.Rect(0, 0, 20, 10), Auftrag{Ausgabe: ziel, JSON: true}, &aus, &fehler)
	if code != exitFehler || !strings.Contains(fehler.String(), "written completely") || !strings.Contains(aus.String(), `"path"`) {
		t.Fatalf("Code %d, aus %q, fehler %q", code, aus.String(), fehler.String())
	}
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Fatalf("Ziel %v, %v", st, err)
	}
	// Ein gewöhnlicher Fehler gibt keine Ergebniszeile aus.
	aus.Reset()
	fehler.Reset()
	setzeNaht(t, func(w io.Writer, inhalt []byte, abgebrochen func() bool) error { return errors.New("test: kaputt") })
	code, _ = anschluesse{}.speichere(b, image.Rect(0, 0, 20, 10), Auftrag{Ausgabe: filepath.Join(dir, "zwei.png"), JSON: true}, &aus, &fehler)
	if code != exitFehler || aus.Len() != 0 {
		t.Errorf("Fehlschlag: Code %d, aus %q", code, aus.String())
	}
}

// F-459: Lässt sich die Sperre der Ausgangsdatei nicht prüfen (hier: ein Verzeichnis am Sperrnamen), meldet die
// Statusleiste, dass die Rettungskopie nicht zur Verfügung steht, als Warnung.
func TestRettungGestoertMeldet(t *testing.T) {
	staat := t.TempDir()
	alt := rettungUmgebung
	rettungUmgebung = func(k string) string {
		if k == "XDG_STATE_HOME" {
			return staat
		}
		return os.Getenv(k)
	}
	t.Cleanup(func() { rettungUmgebung = alt })
	abs := filepath.Join(t.TempDir(), "datei.png")
	if err := os.WriteFile(abs, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ort := filepath.Join(staat, "schnapp", "rescue")
	if err := os.MkdirAll(ort, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(ort, strings.TrimSuffix(rescue.Name(abs), ".json")+".lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	r := rettungFuer(abs, false)
	if r == nil {
		t.Fatal("keine Rettung")
	}
	defer r.Ende()
	d, err := document.Neu(testbild(20, 10, 3))
	if err != nil {
		t.Fatal(err)
	}
	text, fehler := r.Wiederherstellen(d)
	if !strings.Contains(text, "rescue copy for this file is unavailable") || !fehler {
		t.Errorf("Meldung %q, Warnung %v", text, fehler)
	}
}
