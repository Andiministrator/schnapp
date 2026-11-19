// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/einstellungen"
	"github.com/Andiministrator/schnapp/oberflaeche"
	"github.com/Andiministrator/schnapp/po"
	"github.com/Andiministrator/schnapp/sprache"
)

func TestAuswerten(t *testing.T) {
	tests := []struct {
		args []string
		want Aufruf
		fehl string // Teilzeichenkette der Fehlermeldung, leer = kein Fehler
	}{
		{nil, Aufruf{Modus: ModusBereich}, ""},
		{[]string{"--region"}, Aufruf{Modus: ModusBereich}, ""},
		{[]string{"--window"}, Aufruf{Modus: ModusFenster}, ""},
		{[]string{"--repeat"}, Aufruf{Modus: ModusWiederholen}, ""},
		{[]string{"--screen"}, Aufruf{Modus: ModusBildschirm}, ""},
		{[]string{"--list-screens"}, Aufruf{Modus: ModusMonitorliste}, ""},
		{[]string{"--list-screens", "--list-screens"}, Aufruf{Modus: ModusMonitorliste}, ""},
		{[]string{"--screen=Ausgabe-1"}, Aufruf{Modus: ModusBildschirm, Bildschirm: "Ausgabe-1"}, ""},
		{[]string{"--version"}, Aufruf{Modus: ModusVersion}, ""},
		{[]string{"-V"}, Aufruf{Modus: ModusVersion}, ""},
		{[]string{"--help"}, Aufruf{Modus: ModusHilfe}, ""},
		{[]string{"-h"}, Aufruf{Modus: ModusHilfe}, ""},
		{[]string{"--window", "--window"}, Aufruf{Modus: ModusFenster}, ""},
		{[]string{"--unbekannt"}, Aufruf{}, "Unknown option: --unbekannt"},
		{[]string{"--region=5"}, Aufruf{}, "Option --region takes no value"},
		{[]string{"--list-screens=x"}, Aufruf{}, "Option --list-screens takes no value"},
		{[]string{"--list-screens", "--screen"}, Aufruf{}, "Options --list-screens and --screen cannot be combined"},
		{[]string{"--window", "--list-screens"}, Aufruf{}, "Options --window and --list-screens cannot be combined"},
		{[]string{"--screen="}, Aufruf{}, "needs a monitor name"},
		{[]string{"--region", "--window"}, Aufruf{}, "Options --region and --window cannot be combined"},
		{[]string{"--screen", "--screen=x"}, Aufruf{}, "cannot be combined"},
		{[]string{"datei.png"}, Aufruf{}, "Unexpected argument: datei.png"},
		{[]string{"--screen", "x"}, Aufruf{}, "Unexpected argument: x"},
	}
	for _, tc := range tests {
		got, err := Auswerten(tc.args)
		switch {
		case tc.fehl == "" && err != nil:
			t.Errorf("Auswerten(%q): unerwarteter Fehler %v", tc.args, err)
		case tc.fehl != "" && (err == nil || !strings.Contains(err.Error(), tc.fehl)):
			t.Errorf("Auswerten(%q): Fehler %v, erwartet mit %q", tc.args, err, tc.fehl)
		case got != tc.want:
			t.Errorf("Auswerten(%q) = %+v, erwartet %+v", tc.args, got, tc.want)
		}
	}
}

func keineMonitore() ([]oberflaeche.Monitor, error) { return nil, errors.New("keine Verbindung") }

func fuehre(t *testing.T, monitore func() ([]oberflaeche.Monitor, error), args ...string) (int, string, string) {
	t.Helper()
	var aus, fehler bytes.Buffer
	code := laufe(args, &aus, &fehler, monitore, keineEinstellungen(t))
	return code, aus.String(), fehler.String()
}

// keineEinstellungen liefert Voreinstellungen ohne Warnung und zählt die Aufrufe nicht.
func keineEinstellungen(t *testing.T) Ladefunktion {
	return func() (einstellungen.Einstellungen, []string) { return einstellungen.Voreinstellungen(), nil }
}

func TestLaufeEnglisch(t *testing.T) {
	sprache.Setze(nil)
	code, aus, _ := fuehre(t, keineMonitore, "--version")
	if code != exitOK || aus != "Schnapp "+version+"\n" {
		t.Errorf("--version: Code %d, Ausgabe %q", code, aus)
	}
	code, aus, _ = fuehre(t, keineMonitore, "--help")
	if code != exitOK || !strings.HasPrefix(aus, "Usage: schnapp [option]\n") || !strings.Contains(aus, "--repeat") {
		t.Errorf("--help: Code %d, Ausgabe %q", code, aus)
	}
	code, _, fehler := fuehre(t, keineMonitore, "--quatsch")
	if code != exitAufruf || !strings.Contains(fehler, "Unknown option: --quatsch") || !strings.Contains(fehler, "--help") {
		t.Errorf("--quatsch: Code %d, Fehler %q", code, fehler)
	}
}

// Die Aufnahme-Modi gibt es noch nicht: sie melden das und enden mit Code ungleich 0, ohne Monitore abzufragen.
func TestLaufeModiNochNichtUmgesetzt(t *testing.T) {
	sprache.Setze(nil)
	abfrage := func() ([]oberflaeche.Monitor, error) {
		t.Error("Monitorabfrage darf hier nicht laufen")
		return nil, nil
	}
	for _, args := range [][]string{nil, {"--region"}, {"--window"}, {"--repeat"}, {"--screen"}, {"--screen=x"}} {
		code, aus, fehler := fuehre(t, abfrage, args...)
		if code != exitFehler || aus != "" || !strings.Contains(fehler, "not implemented yet") {
			t.Errorf("%q: Code %d, Ausgabe %q, Fehler %q", args, code, aus, fehler)
		}
	}
	_, _, fehler := fuehre(t, abfrage, "--window")
	if !strings.Contains(fehler, "--window") {
		t.Errorf("Meldung nennt den Modus nicht: %q", fehler)
	}
	// --screen ohne Wert listet nicht mehr (F-58), sondern meldet sich als Bildschirm-Modus.
	if _, _, fehler = fuehre(t, abfrage, "--screen"); !strings.Contains(fehler, "mode --screen is not") {
		t.Errorf("--screen ohne Wert: Meldung %q", fehler)
	}
}

func TestMonitorliste(t *testing.T) {
	sprache.Setze(nil)
	zwei := func() ([]oberflaeche.Monitor, error) {
		return []oberflaeche.Monitor{{Name: "A", Breite: 1920, Hoehe: 1080, Skala: 1}, {Name: "B", Breite: 3840, Hoehe: 2160, Skala: 2}}, nil
	}
	code, aus, _ := fuehre(t, zwei, "--list-screens")
	want := "2 monitors found:\n  A: 1920 × 1080 pixels\n  B: 3840 × 2160 pixels\n"
	if code != exitOK || aus != want {
		t.Errorf("Code %d, Ausgabe %q, erwartet %q", code, aus, want)
	}
	eins := func() ([]oberflaeche.Monitor, error) { return []oberflaeche.Monitor{{Name: "A"}}, nil }
	if _, aus, _ = fuehre(t, eins, "--list-screens"); !strings.HasPrefix(aus, "1 monitor found:") {
		t.Errorf("Einzahl: %q", aus)
	}
	code, _, fehler := fuehre(t, keineMonitore, "--list-screens")
	if code != exitFehler || !strings.Contains(fehler, "Could not query the monitors: keine Verbindung") {
		t.Errorf("Fehlerfall: Code %d, Fehler %q", code, fehler)
	}
	leer := func() ([]oberflaeche.Monitor, error) { return nil, nil }
	if code, _, fehler = fuehre(t, leer, "--list-screens"); code != exitFehler || !strings.Contains(fehler, "No monitors found.") {
		t.Errorf("leer: Code %d, Ausgabe %q", code, aus)
	}
}

// Mit deutschem Katalog erscheinen die Texte auf Deutsch (Platzhalter und Mehrzahl eingeschlossen).
func TestLaufeDeutsch(t *testing.T) {
	k, err := sprache.Lade(po.Dateien, "de")
	if err != nil {
		t.Fatal(err)
	}
	sprache.Setze(k)
	t.Cleanup(func() { sprache.Setze(nil) })
	_, aus, _ := fuehre(t, keineMonitore, "--help")
	if !strings.HasPrefix(aus, "Aufruf: schnapp [Option]\n") {
		t.Errorf("--help deutsch: %q", aus)
	}
	zwei := func() ([]oberflaeche.Monitor, error) {
		return []oberflaeche.Monitor{{Name: "A", Breite: 1, Hoehe: 2, Skala: 1}, {Name: "B"}}, nil
	}
	_, aus, _ = fuehre(t, zwei, "--list-screens")
	if !strings.HasPrefix(aus, "2 Monitore gefunden:\n  A: 1 × 2 Pixel\n") {
		t.Errorf("Monitorliste deutsch: %q", aus)
	}
	_, _, fehler := fuehre(t, keineMonitore, "--window")
	if !strings.Contains(fehler, "noch nicht umgesetzt") {
		t.Errorf("Modus deutsch: %q", fehler)
	}
}

// Nur die Aufnahme-Modi lesen die Einstellungen; --version, --help, --list-screens und Aufruffehler nicht.
func TestLaufeLadetEinstellungenNurFuerAufnahme(t *testing.T) {
	sprache.Setze(nil)
	for _, tc := range []struct {
		args  []string
		laedt bool
	}{
		{[]string{"--version"}, false},
		{[]string{"--help"}, false},
		{[]string{"--list-screens"}, false},
		{[]string{"--quatsch"}, false},
		{nil, true},
		{[]string{"--region"}, true},
		{[]string{"--screen"}, true},
		{[]string{"--screen=x"}, true},
		{[]string{"--window"}, true},
		{[]string{"--repeat"}, true},
	} {
		aufrufe := 0
		lade := func() (einstellungen.Einstellungen, []string) {
			aufrufe++
			return einstellungen.Voreinstellungen(), []string{"settings.ini:3: Warnung"}
		}
		var aus, fehler bytes.Buffer
		laufe(tc.args, &aus, &fehler, func() ([]oberflaeche.Monitor, error) { return nil, nil }, lade)
		if (aufrufe == 1) != tc.laedt || aufrufe > 1 {
			t.Errorf("%q: Einstellungen %d-mal geladen, erwartet geladen=%v", tc.args, aufrufe, tc.laedt)
		}
		if got := strings.Contains(fehler.String(), "settings.ini:3: Warnung\n"); got != tc.laedt {
			t.Errorf("%q: Warnung auf stderr = %v, erwartet %v (stderr %q)", tc.args, got, tc.laedt, fehler.String())
		}
	}
}

// Jede Warnung steht in einer eigenen Zeile auf stderr, die Aufnahme läuft weiter (hier: dieselbe Meldung wie sonst).
func TestLaufeWarnungenWeiter(t *testing.T) {
	sprache.Setze(nil)
	lade := func() (einstellungen.Einstellungen, []string) {
		return einstellungen.Voreinstellungen(), []string{"w1", "w2"}
	}
	var aus, fehler bytes.Buffer
	code := laufe([]string{"--window"}, &aus, &fehler, keineMonitore, lade)
	if code != exitFehler || !strings.HasPrefix(fehler.String(), "w1\nw2\n") || !strings.Contains(fehler.String(), "not implemented yet") {
		t.Errorf("Code %d, stderr %q", code, fehler.String())
	}
}

// Vorrang: --screen=NAME vor Datei, --screen ohne Wert folgt der Datei, sonst Voreinstellung.
func TestPlaneVorrang(t *testing.T) {
	alle := einstellungen.Voreinstellungen()
	alle.Bildschirm = einstellungen.WahlAlle
	tests := []struct {
		name string
		a    Aufruf
		e    einstellungen.Einstellungen
		want Bildschirmziel
	}{
		{"Name schlägt Datei", Aufruf{Modus: ModusBildschirm, Bildschirm: "HDMI-1"}, alle, Bildschirmziel{Name: "HDMI-1"}},
		{"ohne Wert folgt Datei", Aufruf{Modus: ModusBildschirm}, alle, Bildschirmziel{Wahl: einstellungen.WahlAlle}},
		{"ohne Datei Voreinstellung", Aufruf{Modus: ModusBildschirm}, einstellungen.Voreinstellungen(), Bildschirmziel{Wahl: einstellungen.WahlKlick}},
		{"anderer Modus ohne Ziel", Aufruf{Modus: ModusBereich}, alle, Bildschirmziel{}},
	}
	for _, tc := range tests {
		if got := plane(tc.a, tc.e); got.Ziel != tc.want || got.Modus != tc.a.Modus {
			t.Errorf("%s: %+v, erwartet Ziel %+v", tc.name, got, tc.want)
		}
	}
}
