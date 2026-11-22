// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Regressionstests aus der Kritik zu Sprint 11 (ADR-0030; Befunde KA-2, KA-4, KA-6, KA-8, KB-3, KB-11). Der Test zu
// KA-2 geht auf die Gegenprobe von Kritiker A zurück.

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// arbeitsdatei legt eine Arbeitsdatei mit Zuschnitt und Kasten an und liefert Pfad und Inhalt.
func arbeitsdatei(t *testing.T, dir string) (string, []byte) {
	t.Helper()
	d, err := document.Neu(testbild(40, 30, 7))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetzeZuschnitt(image.Rect(10, 5, 30, 25)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(12, 7), document.Pt(20, 15)),
		Strich: document.Strich{Farbe: color.NRGBA{R: 255, A: 255}, Breite: 2}}); err != nil {
		t.Fatal(err)
	}
	var a bytes.Buffer
	if err := fileformat.SchreibeArbeitsdatei(&a, d, image.NewNRGBA(image.Rect(0, 0, 20, 20)), fileformat.Metadaten{}); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(dir, "arbeit.png")
	if err := os.WriteFile(pfad, a.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return pfad, a.Bytes()
}

// KA-2: --export=ZIEL DATEI mit ZIEL = DATEI — über denselben Namen, einen symbolischen oder einen harten Link —
// überschriebe die Arbeitsdatei mit dem flachen Bild (Original und Beschriftungen weg). Es ist ein Fehler (Exit 1) mit
// übersetzter Meldung; die Datei bleibt Byte für Byte, wie sie war.
func TestExportAufSichSelbst(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	quelle, inhalt := arbeitsdatei(t, dir)
	symlink, hardlink := filepath.Join(dir, "link.png"), filepath.Join(dir, "hart.png")
	if err := os.Symlink(quelle, symlink); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(quelle, hardlink); err != nil {
		t.Fatal(err)
	}
	for _, ziel := range []string{quelle, symlink, hardlink} {
		code, aus, fehler := fuehre(t, keineMonitore, "--export="+ziel, quelle)
		if code != exitFehler || aus != "" || !strings.Contains(fehler, "onto itself") {
			t.Errorf("Ziel %s: Code %d, aus %q, fehler %q", filepath.Base(ziel), code, aus, fehler)
		}
		if danach, _ := os.ReadFile(quelle); !bytes.Equal(danach, inhalt) {
			t.Fatalf("Ziel %s: --export hat die Arbeitsdatei verändert", filepath.Base(ziel))
		}
	}
	// Ein anderes vorhandenes Ziel bleibt erlaubt (Gegenprobe).
	anders := filepath.Join(dir, "anders.png")
	if err := os.WriteFile(anders, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, fehler := fuehre(t, keineMonitore, "--export="+anders, quelle); code != exitOK {
		t.Errorf("anderes Ziel: Code %d, %q", code, fehler)
	}
}

// KA-4: Eine neue Datei von --output bekommt die Rechte 0600, auch bei der umask 022 (Bildschirminhalt, wie die
// Bibliothek); eine vorhandene Zieldatei behält ihre Rechte. --export bleibt bei der umask (wie der Dialog-Export).
func TestAusgabeRechte(t *testing.T) {
	i18n.Setze(nil)
	alt := syscall.Umask(0o022)
	defer syscall.Umask(alt)
	dir := t.TempDir()
	neu := filepath.Join(dir, "neu.png")
	if code, _, fehler := neuerLauf(t).fuehre(Auftrag{Modus: ModusFenster, Ausgabe: neu}); code != exitOK {
		t.Fatalf("Code %d, %q", code, fehler)
	}
	if st, err := os.Stat(neu); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("neue Datei von --output: %v, %v; Soll 0600", st.Mode().Perm(), err)
	}
	vorhanden := filepath.Join(dir, "vorhanden.png")
	if err := os.WriteFile(vorhanden, []byte("alt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, fehler := neuerLauf(t).fuehre(Auftrag{Modus: ModusFenster, Ausgabe: vorhanden}); code != exitOK {
		t.Fatalf("Code %d, %q", code, fehler)
	}
	if st, err := os.Stat(vorhanden); err != nil || st.Mode().Perm() != 0o644 {
		t.Errorf("vorhandene Datei: %v, %v; Soll 0644", st.Mode().Perm(), err)
	}
	quelle, _ := arbeitsdatei(t, dir)
	ex := filepath.Join(dir, "export.png")
	if code, _, fehler := fuehre(t, keineMonitore, "--export="+ex, quelle); code != exitOK {
		t.Fatalf("--export: Code %d, %q", code, fehler)
	}
	if st, err := os.Stat(ex); err != nil || st.Mode().Perm() != 0o644 {
		t.Errorf("neue Datei von --export: %v, %v; Soll 0644 (umask)", st.Mode().Perm(), err)
	}
}

// KA-6/KB-4: --export prüft das Rechenbudget wie das Öffnen im Editor: ohne --ignore-limit Exit 1, die Meldung nennt
// den Schalter, kein Ziel entsteht; mit dem Schalter Exit 0 und das Ziel liegt vor.
func TestExportRechenbudget(t *testing.T) {
	i18n.Setze(nil)
	dir := t.TempDir()
	pfad, _ := ueberBudget(t, dir)
	ziel := filepath.Join(dir, "aus.svg")
	code, aus, fehler := fuehre(t, keineMonitore, "--export="+ziel, pfad)
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "--ignore-limit") {
		t.Errorf("ohne --ignore-limit: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if _, err := os.Lstat(ziel); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Ziel trotz Ablehnung angelegt: %v", err)
	}
	code, aus, fehler = fuehre(t, keineMonitore, "--ignore-limit", "--export="+ziel, pfad)
	if code != exitOK || aus != ziel+"\n" {
		t.Errorf("mit --ignore-limit: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if st, err := os.Stat(ziel); err != nil || st.Size() == 0 {
		t.Errorf("Ziel fehlt: %v", err)
	}
}

// KA-8: Mit --json lehnt die Kommandozeile Ziele ab, die kein gültiges UTF-8 sind (Exit 2), statt später still U+FFFD
// zu melden; ohne --json sind sie erlaubt. gibPfadAus schreibt bei einem solchen Pfad keine verfälschte JSON-Zeile.
func TestJSONZielOhneUTF8(t *testing.T) {
	i18n.Setze(nil)
	kaputt := "bild-\xff.png"
	for _, args := range [][]string{{"--output=" + kaputt, "--json"}, {"--json", "--export=" + kaputt, "f.png"}} {
		if _, err := Auswerten(args); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
			t.Errorf("%q: Fehler %v, Soll Hinweis auf UTF-8", args, err)
		}
		code, _, fehler := fuehre(t, keineMonitore, args...)
		if code != exitAufruf || !strings.Contains(fehler, "valid UTF-8") {
			t.Errorf("%q: Code %d, %q", args, code, fehler)
		}
	}
	if a, err := Auswerten([]string{"--output=" + kaputt}); err != nil || a.Ausgabe != kaputt {
		t.Errorf("ohne --json: %+v, %v", a, err)
	}
	var aus, fehler bytes.Buffer
	if gibPfadAus(&aus, &fehler, true, "/x/"+kaputt, image.Pt(1, 1)) || aus.Len() != 0 || !strings.Contains(fehler.String(), "UTF-8") {
		t.Errorf("gibPfadAus: aus %q, fehler %q", aus.String(), fehler.String())
	}
}

// KA-8: schreibFehler gibt keine rohen (deutschen) Paketfehler an den Nutzer, sondern eine übersetzte Meldung; der
// Rückfall-Schaden (ADR-0025 Nachtrag 3/4) bekommt eine eigene.
func TestSchreibFehlerUebersetzt(t *testing.T) {
	i18n.Setze(nil)
	roh := errors.New("export: schließen: irgendwas Deutsches")
	if got := schreibFehler(roh); got != "The file could not be written." {
		t.Errorf("unbekannter Fehler: %q", got)
	}
	beschaedigt := fmt.Errorf("%w: a; Wiederherstellung gescheitert: b", export.ErrZielBeschaedigt)
	if got := schreibFehler(beschaedigt); !strings.Contains(got, "incomplete") || strings.Contains(got, "Wiederherstellung") {
		t.Errorf("ErrZielBeschaedigt: %q", got)
	}
}

// KB-3: Positivkontrolle zum Zähler in TestAufnahmeMitOutput — derselbe Aufbau ohne --output räumt einmal auf, mit
// --output nicht.
func TestAusgabeOhneAufraeumen(t *testing.T) {
	i18n.Setze(nil)
	for _, fall := range []struct {
		ausgabe string
		soll    int
	}{{filepath.Join(t.TempDir(), "x.png"), 0}, {"", 1}} {
		l := neuerLauf(t)
		n := 0
		l.an.aufraeumen = func(library.Einstellung, time.Time) (library.AufraeumErgebnis, error) {
			n++
			return library.AufraeumErgebnis{}, nil
		}
		l.an.tagesmarke = filepath.Join(t.TempDir(), "cleanup-stamp")
		code, _, fehler := l.fuehre(Auftrag{Modus: ModusFenster, Ausgabe: fall.ausgabe,
			Aufraeumen: library.Einstellung{An: true, Tage: 7}})
		if code != exitOK || n != fall.soll {
			t.Errorf("--output %q: Code %d, Aufräumen %d, Soll %d (%q)", fall.ausgabe, code, n, fall.soll, fehler)
		}
	}
}

// KB-11: starteSprache nimmt die Sprache aus der Einstellung ([general] language), vor der Umgebung.
func TestStarteSprache(t *testing.T) {
	t.Cleanup(func() { i18n.Setze(nil) })
	t.Setenv("LANGUAGE", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "en_US.UTF-8")
	mit := func(sprache string) Ladefunktion {
		return func() (settings.Einstellungen, []string) {
			e := settings.Voreinstellungen()
			e.Sprache = sprache
			return e, []string{"Warnung, die hier nicht zählt"}
		}
	}
	starteSprache(mit("de"))
	if got := i18n.T("Show the version"); got != "Version anzeigen" {
		t.Errorf("Einstellung de: %q", got)
	}
	starteSprache(mit(""))
	if got := i18n.T("Show the version"); got != "Show the version" {
		t.Errorf("auto mit LANG=en_US: %q", got)
	}
}
