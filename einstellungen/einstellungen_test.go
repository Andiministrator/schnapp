// SPDX-License-Identifier: GPL-3.0-or-later

package einstellungen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/bibliothek"
	"github.com/Andiministrator/schnapp/sprache"
)

// umgebung richtet ein eigenes HOME und XDG_CONFIG_HOME unter t.TempDir() ein und liefert den Ort der Datei.
func umgebung(t *testing.T) string {
	t.Helper()
	sprache.Setze(nil)
	wurzel := t.TempDir()
	t.Setenv("HOME", filepath.Join(wurzel, "heim"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(wurzel, "konfig"))
	return filepath.Join(wurzel, "konfig", "schnapp", "settings.ini")
}

func schreibe(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFehlendeDatei(t *testing.T) {
	umgebung(t)
	e, warn := Lade()
	if e != Voreinstellungen() || len(warn) != 0 {
		t.Errorf("fehlende Datei: %+v, Warnungen %q", e, warn)
	}
	if e.Bildschirm != WahlKlick || e.Bibliothek != bibliothek.Voreinstellung {
		t.Errorf("Voreinstellungen falsch: %+v", e)
	}
}

func TestPfad(t *testing.T) {
	umgebung(t)
	t.Setenv("XDG_CONFIG_HOME", "/xdg/konfig")
	if p, err := Pfad(); err != nil || p != "/xdg/konfig/schnapp/settings.ini" {
		t.Errorf("XDG_CONFIG_HOME absolut: %q, %v", p, err)
	}
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	for _, w := range []string{"", "relativ/pfad"} {
		t.Setenv("XDG_CONFIG_HOME", w)
		if p, err := Pfad(); err != nil || p != filepath.Join(heim, ".config", "schnapp", "settings.ini") {
			t.Errorf("XDG_CONFIG_HOME %q: %q, %v", w, p, err)
		}
	}
}

// Ein relatives XDG_CONFIG_HOME wird ignoriert: die Datei im Heimverzeichnis wird gelesen.
func TestLadeRelativesXDG(t *testing.T) {
	umgebung(t)
	heim := t.TempDir()
	t.Setenv("HOME", heim)
	t.Setenv("XDG_CONFIG_HOME", "relativ")
	schreibe(t, filepath.Join(heim, ".config", "schnapp", "settings.ini"), "[capture]\nscreen = all\n")
	e, warn := Lade()
	if e.Bildschirm != WahlAlle || len(warn) != 0 {
		t.Errorf("%+v, %q", e, warn)
	}
}

func TestGueltigeDateiMitKommentaren(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "\xef\xbb\xbf# Kommentar\r\n; noch einer\n\n  [capture]  \n   # eingerückt\n  screen   =   pointer  \n\n"+
		"[library]\ncleanup=false\n\tcleanup-days =\t30\r\n")
	e, warn := Lade()
	want := Einstellungen{Bildschirm: WahlZeiger, Bibliothek: bibliothek.Einstellung{An: false, Tage: 30}}
	if e != want || len(warn) != 0 {
		t.Errorf("%+v, Warnungen %q; erwartet %+v ohne Warnung", e, warn, want)
	}
}

func TestBildschirmwerte(t *testing.T) {
	for _, w := range []Bildschirmwahl{WahlKlick, WahlZeiger, WahlAlle} {
		pfad := umgebung(t)
		schreibe(t, pfad, "[capture]\nscreen = "+string(w)+"\n")
		if e, warn := Lade(); e.Bildschirm != w || len(warn) != 0 {
			t.Errorf("screen = %s: %+v, %q", w, e, warn)
		}
	}
	pfad := umgebung(t)
	schreibe(t, pfad, "[capture]\nscreen = pointer\nscreen = quatsch\n")
	e, warn := Lade()
	if e.Bildschirm != WahlZeiger || len(warn) != 1 || !strings.Contains(warn[0], ":3: Invalid value \"quatsch\"") {
		t.Errorf("ungültiger Wert: %+v, %q", e, warn)
	}
}

func TestCleanup(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "[library]\ncleanup = false\n")
	if e, warn := Lade(); e.Bibliothek.An || e.Bibliothek.Tage != 90 || len(warn) != 0 {
		t.Errorf("cleanup false: %+v, %q", e, warn)
	}
	schreibe(t, pfad, "[library]\ncleanup = true\n")
	if e, warn := Lade(); !e.Bibliothek.An || len(warn) != 0 {
		t.Errorf("cleanup true: %+v, %q", e, warn)
	}
	schreibe(t, pfad, "[library]\ncleanup = ja\n")
	e, warn := Lade()
	if !e.Bibliothek.An || len(warn) != 1 || !strings.Contains(warn[0], ":2: Invalid value \"ja\" for key \"cleanup\"") {
		t.Errorf("cleanup ja: %+v, %q", e, warn)
	}
}

func TestCleanupTage(t *testing.T) {
	tests := []struct {
		wert string
		tage int
		warn bool
	}{
		{"1", 1, false},
		{"36500", 36500, false},
		{"365", 365, false},
		{"0", 90, true},
		{"36501", 90, true},
		{"-5", 90, true},
		{"abc", 90, true},
		{"", 90, true},
		{"1.5", 90, true},
		{"99999999999999999999", 90, true},
	}
	for _, tc := range tests {
		pfad := umgebung(t)
		schreibe(t, pfad, "[library]\ncleanup-days = "+tc.wert+"\n")
		e, warn := Lade()
		if e.Bibliothek.Tage != tc.tage || (len(warn) == 1) != tc.warn || len(warn) > 1 {
			t.Errorf("cleanup-days = %q: Tage %d, Warnungen %q; erwartet %d, Warnung %v", tc.wert, e.Bibliothek.Tage, warn, tc.tage, tc.warn)
		}
		if tc.warn && !strings.Contains(warn[0], ":2: ") {
			t.Errorf("cleanup-days = %q: Warnung ohne Zeilennummer 2: %q", tc.wert, warn)
		}
		if !e.Bibliothek.An {
			t.Errorf("cleanup-days = %q: cleanup darf nicht kippen", tc.wert)
		}
	}
}

func TestUnbekanntes(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "[capture]\nscreen = all\nfarbe = rot\n[quatsch]\na = 1\nb = 2\n[library]\ncleanup-days = 7\nfrist = 3\n")
	e, warn := Lade()
	want := Einstellungen{Bildschirm: WahlAlle, Bibliothek: bibliothek.Einstellung{An: true, Tage: 7}}
	if e != want {
		t.Errorf("Werte %+v, erwartet %+v", e, want)
	}
	erwartet := []string{
		":3: Unknown key \"farbe\" in section [capture]",
		":4: Unknown section [quatsch]",
		":9: Unknown key \"frist\" in section [library]",
	}
	if len(warn) != len(erwartet) {
		t.Fatalf("Warnungen %q, erwartet %d", warn, len(erwartet))
	}
	for i, teil := range erwartet {
		if !strings.Contains(warn[i], teil) || !strings.HasPrefix(warn[i], pfad+":") {
			t.Errorf("Warnung %d = %q, erwartet mit %q", i, warn[i], teil)
		}
	}
}

func TestKaputteZeilen(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "vorn = 1\nkeinGleich\n[capture\n[capture]\n= leer\nscreen = pointer\nnochkaputt\n")
	e, warn := Lade()
	if e.Bildschirm != WahlZeiger {
		t.Errorf("Bildschirm %q, erwartet pointer: Rest muss weitergelesen werden", e.Bildschirm)
	}
	erwartet := []string{
		":1: Setting \"vorn\" is outside of any section",
		":2: Line is neither a section",
		":3: Broken section header",
		":5: Line is neither a section",
		":7: Line is neither a section",
	}
	if len(warn) != len(erwartet) {
		t.Fatalf("Warnungen %q, erwartet %d", warn, len(erwartet))
	}
	for i, teil := range erwartet {
		if !strings.Contains(warn[i], teil) {
			t.Errorf("Warnung %d = %q, erwartet mit %q", i, warn[i], teil)
		}
	}
}

func TestDoppelterSchluessel(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "[library]\ncleanup-days = 10\ncleanup-days = 20\n")
	e, warn := Lade()
	if e.Bibliothek.Tage != 20 || len(warn) != 1 || !strings.Contains(warn[0], ":3: Key \"cleanup-days\" in section [library] is set again") {
		t.Errorf("doppelt: Tage %d, Warnungen %q", e.Bibliothek.Tage, warn)
	}
	// Später ungültig: der frühere gültige Wert bleibt, eine Warnung (ungültig).
	schreibe(t, pfad, "[library]\ncleanup-days = 10\ncleanup-days = 0\n")
	e, warn = Lade()
	if e.Bibliothek.Tage != 10 || len(warn) != 1 || !strings.Contains(warn[0], ":3: Invalid value") {
		t.Errorf("doppelt ungültig: Tage %d, Warnungen %q", e.Bibliothek.Tage, warn)
	}
}

func TestNichtLesbar(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "[capture]\nscreen = all\n")
	if err := os.Chmod(pfad, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(pfad, 0o600) //nolint:errcheck // TempDir räumt ohnehin ab
	if _, err := os.ReadFile(pfad); err == nil {
		t.Skip("Datei trotz Rechte 000 lesbar (root?)")
	}
	e, warn := Lade()
	if e != Voreinstellungen() || len(warn) != 1 || !strings.Contains(warn[0], "cannot read the settings file") || !strings.HasPrefix(warn[0], pfad+": ") {
		t.Errorf("nicht lesbar: %+v, Warnungen %q", e, warn)
	}
}

// Ein Verzeichnis statt der Datei ist ein Lesefehler, kein Absturz.
func TestVerzeichnisStattDatei(t *testing.T) {
	pfad := umgebung(t)
	if err := os.MkdirAll(pfad, 0o700); err != nil {
		t.Fatal(err)
	}
	e, warn := Lade()
	if e != Voreinstellungen() || len(warn) != 1 {
		t.Errorf("%+v, %q", e, warn)
	}
}

func TestWarnungenUebersetzt(t *testing.T) {
	pfad := umgebung(t)
	schreibe(t, pfad, "[library]\ncleanup-days = 0\n")
	sprache.Setze(nil)
	_, warn := Lade()
	if len(warn) != 1 || !strings.Contains(warn[0], "Invalid value") {
		t.Errorf("englisch: %q", warn)
	}
}
