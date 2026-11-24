// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/probeguard"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// echteAusfuehrungOhneAufnahme baut die echte Ausführung (echteAnschluesse, samt echter Schutzprüfung), ersetzt aber
// Aufnahmequelle und Editor durch Zähler: Scheitert der Schutz, gäbe es sonst eine echte Aufnahme (F-229).
func echteAusfuehrungOhneAufnahme(t *testing.T) (an anschluesse, quelleGerufen, editorGerufen *int) {
	t.Helper()
	an = echteAnschluesse()
	quelleGerufen, editorGerufen = new(int), new(int)
	an.quelle = func() (capture.Quelle, error) { *quelleGerufen++; return nil, errors.New("Attrappe") }
	an.editor = func(string, bool, document.Kante) error { *editorGerufen++; return nil }
	an.bearbeiten = func(abgelegt) error { *editorGerufen++; return nil }
	return an, quelleGerufen, editorGerufen
}

// TestProbeBuildOhnePrivatenBusEndetVorDerAufnahme (F-229): Der Probe-Build, in der echten Sitzung gestartet (keine
// Probe-Umgebung), endet mit Fehler und Exit 1, ohne die Aufnahmeschicht oder den Editor zu berühren.
func TestProbeBuildOhnePrivatenBusEndetVorDerAufnahme(t *testing.T) {
	for _, modus := range []Modus{ModusBereich, ModusBildschirm, ModusFenster, ModusWiederholen, ModusOeffnen} {
		// echte Sitzung: Marke, Bus und Wayland-Socket der Probe fehlen
		t.Setenv("PROBE_INNEN", "")
		t.Setenv("PROBE_AUSSENBUS", "")
		t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/1000/bus")
		t.Setenv("WAYLAND_DISPLAY", "wayland-0")
		t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
		an, q, e := echteAusfuehrungOhneAufnahme(t)
		var aus, fehler bytes.Buffer
		if code := an.Ausfuehren(Auftrag{Modus: modus, Datei: "x.png"}, &aus, &fehler); code != exitFehler {
			t.Errorf("Modus %v: Exit %d, erwartet %d", modus, code, exitFehler)
		}
		if *q != 0 || *e != 0 {
			t.Errorf("Modus %v: Aufnahme (%d) oder Editor (%d) trotz fehlendem Schutz", modus, *q, *e)
		}
		if fehler.Len() == 0 || aus.Len() != 0 {
			t.Errorf("Modus %v: stderr %q, stdout %q", modus, fehler.String(), aus.String())
		}
	}
}

// TestProbeBuildNurMarkeGenuegtNicht: Eine versehentlich gesetzte PROBE_INNEN=1 allein öffnet den Schutz nicht.
func TestProbeBuildNurMarkeGenuegtNicht(t *testing.T) {
	t.Setenv("PROBE_INNEN", "1")
	t.Setenv("PROBE_AUSSENBUS", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	an, q, _ := echteAusfuehrungOhneAufnahme(t)
	var aus, fehler bytes.Buffer
	if code := an.Ausfuehren(Auftrag{Modus: ModusBereich}, &aus, &fehler); code != exitFehler || *q != 0 {
		t.Errorf("Exit %d, Aufnahmen %d", code, *q)
	}
}

// TestProbeBuildImVirtuellenKWinLaeuft: Mit der vollständigen Probe-Umgebung von scripts/probe.sh geht der Lauf
// bis zur Aufnahmeschicht (die hier die Attrappe ist), der Schutz sperrt ihn nicht.
func TestProbeBuildImVirtuellenKWinLaeuft(t *testing.T) {
	t.Setenv("PROBE_INNEN", "1")
	t.Setenv("PROBE_AUSSENBUS", "unix:path=/run/user/1000/bus")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/a/privat")
	t.Setenv("WAYLAND_DISPLAY", "schnapp-probe")
	t.Setenv("XDG_RUNTIME_DIR", "/a/tmp/probe/run")
	if err := probeGuard(); err != nil {
		t.Fatalf("Schutz sperrt die Probe-Umgebung: %v", err)
	}
	an, q, _ := echteAusfuehrungOhneAufnahme(t)
	var aus, fehler bytes.Buffer
	_ = an.Ausfuehren(Auftrag{Modus: ModusBereich}, &aus, &fehler)
	if *q != 1 {
		t.Errorf("Aufnahmeschicht %d-mal gerufen, erwartet 1", *q)
	}
}

// TestProbeguardIstDerGeteilte: probeGuard prüft dieselben Bedingungen wie die Fernbedienung (internal/probeguard).
func TestProbeguardIstDerGeteilte(t *testing.T) {
	t.Setenv("PROBE_INNEN", "")
	if (probeGuard() == nil) != (probeguard.NurVirtuell(func(k string) string { return "" }) == nil) {
		t.Error("probeGuard weicht von probeguard.NurVirtuell ab")
	}
}

// TestProbeHalterEndetVorDerZwischenablage (F-244): Der Halter der Zwischenablage (--internal-keep-clipboard) im
// Probe-Build endet in der echten Sitzung mit Exit 1, ohne die Zwischenablage zu berühren; im virtuellen KWin läuft er.
func TestProbeHalterEndetVorDerZwischenablage(t *testing.T) {
	t.Setenv("PROBE_INNEN", "")
	t.Setenv("PROBE_AUSSENBUS", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/1000/bus")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	gerufen := 0
	lauf := func(string) int { gerufen++; return exitOK }
	var fehler bytes.Buffer
	code, ok := halter([]string{wlclip.HelferArgument}, &fehler, lauf)
	if !ok || code != exitFehler || gerufen != 0 || fehler.Len() == 0 {
		t.Errorf("echte Sitzung: ok %v, Exit %d, Halter %d-mal gerufen, stderr %q", ok, code, gerufen, fehler.String())
	}
	t.Setenv("PROBE_INNEN", "1")
	t.Setenv("PROBE_AUSSENBUS", "unix:path=/run/user/1000/bus")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/a/privat")
	t.Setenv("WAYLAND_DISPLAY", "schnapp-probe")
	t.Setenv("XDG_RUNTIME_DIR", "/a/tmp/probe/run")
	if code, ok := halter([]string{wlclip.HelferArgument}, &fehler, lauf); !ok || code != exitOK || gerufen != 1 {
		t.Errorf("virtueller KWin: ok %v, Exit %d, Halter %d-mal gerufen", ok, code, gerufen)
	}
}
